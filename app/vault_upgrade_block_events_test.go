package app_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	v22 "structs/app/upgrades/v0_22_0"
	"structs/x/structs/types"
)

// Exercises real x/upgrade PreBlock delivery using a test-only handler composed
// from the production asset phase, alongside the assembled upgrade handler.
// Neither variant establishes downstream indexer compatibility.
func TestVaultAssetMigrationEventsReachUpgradeFinalizeBlock(t *testing.T) {
	testVaultAssetMigrationBlockEvents(t, false)
}

func TestV22HandlerBankAndVestingEventsReachUpgradeBlock(t *testing.T) {
	testVaultAssetMigrationBlockEvents(t, true)
}

func testVaultAssetMigrationBlockEvents(t *testing.T, completeHandler bool) {
	t.Helper()
	bApp, ctx := setupJailGateApp(t)
	const orphanPower = "5-6-999999"
	if completeHandler {
		bApp.StructsKeeper.SetGridAttribute(ctx, orphanPower, 123)
	}
	primary, secondary, playerID := newTransferPlayer(t, bApp, ctx)
	guild := types.CreateEmptyGuild()
	guild.Id, guild.Index, guild.Owner = "0-991", 991, playerID
	bApp.StructsKeeper.SetGuild(ctx, guild)
	first := sdk.NewCoins(sdk.NewInt64Coin("ualpha", 20), sdk.NewInt64Coin("oretest", 7), sdk.NewInt64Coin("uguild.0-991", 3))
	second := sdk.NewCoins(sdk.NewInt64Coin("ualpha", 8))
	fundRedelegationAccount(t, bApp, ctx, primary, first)
	fundRedelegationAccount(t, bApp, ctx, secondary, second)
	base := bApp.AccountKeeper.GetAccount(ctx, primary).(*authtypes.BaseAccount)
	vesting, err := vestingtypes.NewDelayedVestingAccount(base, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 10)), ctx.BlockTime().Add(365*24*time.Hour).Unix())
	require.NoError(t, err)
	bApp.AccountKeeper.SetAccount(ctx, vesting)
	vault, err := types.VaultAddress(playerID)
	require.NoError(t, err)
	const upgradeName = "test-vault-event-delivery"
	require.NoError(t, bApp.UpgradeKeeper.ScheduleUpgrade(ctx, upgradetypes.Plan{Name: upgradeName, Height: 2}))
	_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime()})
	require.NoError(t, err)
	_, err = bApp.Commit()
	require.NoError(t, err)
	// Simulate installing the new binary at the upgrade boundary. Registering
	// before block one correctly triggers x/upgrade's premature-binary guard.
	fixtureRoot := os.Getenv("STRUCTS_UPGRADE_EVENT_FIXTURE_ROOT")
	var beforeExport map[string]json.RawMessage
	if fixtureRoot != "" {
		beforeCtx := bApp.NewContextLegacy(true, cmtproto.Header{Height: 1, ChainID: jailGateChainID, Time: ctx.BlockTime()})
		beforeExport, err = bApp.ModuleManager.ExportGenesisForModules(beforeCtx, bApp.AppCodec(), nil)
		require.NoError(t, err)
	}
	bApp.UpgradeKeeper.SetUpgradeHandler(upgradeName, func(goCtx context.Context, _ upgradetypes.Plan, versions module.VersionMap) (module.VersionMap, error) {
		if completeHandler {
			// No stake operations in this fixture, so the pre-upgrade queue is
			// empty and does not need conversion from the new binary's format.
			require.Empty(t, bApp.StructsKeeper.GetGridCascadeQueue(goCtx, false))
			return v22.CreateUpgradeHandler(bApp.ModuleManager, bApp.Configurator(), delegationUpgradeKeepers(bApp))(goCtx, upgradetypes.Plan{Name: v22.UpgradeName}, versions)
		}
		return versions, v22.MigrateVaultAssets(goCtx, delegationUpgradeKeepers(bApp))
	})
	blockTime := ctx.BlockTime().Add(time.Second)
	result, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 2, Time: blockTime})
	require.NoError(t, err)
	require.Empty(t, result.TxResults, "migration events have no user transaction")
    revisionResets, revisions := commissionEvents(t, result.Events)
    if completeHandler {
        require.Equal(t, 1, revisionResets, "the actual upgrade block initializes indexed commission revision state")
        require.Empty(t, revisions, "the old chain has no revision records")
    } else {
        require.Zero(t, revisionResets, "asset migration alone cannot reset reactor state")
    }

	orphanClears := 0
	transfers := map[string]sdk.Coins{}
	associationIndex, firstTransferIndex, vestingEvents := -1, -1, 0
	for index, event := range result.Events {
		switch event.Type {
		case "structs.structs.EventGrid":
			decoded, err := sdk.ParseTypedEvent(event)
			require.NoError(t, err)
			row := decoded.(*types.EventGrid).GridRecord
			if row.AttributeId == orphanPower {
				require.Zero(t, row.Value)
				orphanClears++
			}
		case "structs.structs.EventVaultAddress":
			decoded, err := sdk.ParseTypedEvent(event)
			require.NoError(t, err)
			detail := decoded.(*types.EventVaultAddress).EventVaultAddressDetail
			if detail.OwnerId == playerID {
				require.Equal(t, -1, associationIndex, "one association per owner")
				associationIndex = index
				require.Equal(t, vault.String(), detail.VaultAddress)
				require.True(t, detail.AccountCreated)
			}
		case banktypes.EventTypeTransfer:
			attributes := map[string]string{}
			for _, attribute := range event.Attributes {
				attributes[attribute.Key] = attribute.Value
			}
			if attributes[banktypes.AttributeKeyRecipient] != vault.String() {
				continue
			}
			amount, err := sdk.ParseCoinsNormalized(attributes[sdk.AttributeKeyAmount])
			require.NoError(t, err)
			sender := attributes[sdk.AttributeKeySender]
			transfers[sender] = transfers[sender].Add(amount...)
			if firstTransferIndex == -1 {
				firstTransferIndex = index
			}
		case "structs.structs.EventVaultVestingMigration":
			decoded, err := sdk.ParseTypedEvent(event)
			require.NoError(t, err)
			migration := decoded.(*types.EventVaultVestingMigration)
			require.Equal(t, playerID, migration.Vault.OwnerId)
			require.Equal(t, []string{primary.String()}, migration.SourceAddresses)
			require.Len(t, migration.Vault.Schedules, 1)
			vestingEvents++
		}
	}
	require.NotEqual(t, -1, associationIndex)
	require.Greater(t, firstTransferIndex, associationIndex, "association precedes its asset transfers")
	require.Zero(t, vestingEvents, "native vesting accounts are retained")
	require.Equal(t, map[string]sdk.Coins{secondary.String(): second}, transfers)
	_, err = bApp.Commit()
	require.NoError(t, err)
	finalCtx := bApp.NewContextLegacy(true, cmtproto.Header{Height: 2, ChainID: jailGateChainID, Time: blockTime})
	require.Equal(t, second, bApp.BankKeeper.GetAllBalances(finalCtx, vault))
	require.Equal(t, first, bApp.BankKeeper.GetAllBalances(finalCtx, primary))
	require.True(t, bApp.BankKeeper.GetAllBalances(finalCtx, secondary).IsZero())
	require.Equal(t, vesting, bApp.AccountKeeper.GetAccount(finalCtx, primary))
	if completeHandler {

		require.Equal(t, 1, orphanClears, "the upgrade block must deliver the exact historical power clear")
		require.Nil(t, finalCtx.KVStore(bApp.GetKey(types.StoreKey)).Get([]byte(types.GridAttributeKey+orphanPower)))
	} else {
		require.Zero(t, orphanClears)
	}
	installed := bApp.AccountKeeper.GetAccount(finalCtx, primary).(*vestingtypes.DelayedVestingAccount)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("ualpha", 10)), installed.LockedCoins(blockTime))
	require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(finalCtx, vault.String()))
	if fixtureRoot != "" {
		directory, err := os.MkdirTemp(fixtureRoot, "vault-block-events-")
		require.NoError(t, err)
		afterExport, err := bApp.ModuleManager.ExportGenesisForModules(finalCtx, bApp.AppCodec(), nil)
		require.NoError(t, err)
		owners, err := bApp.StructsKeeper.GetVaultOwnersForExport(finalCtx)
		require.NoError(t, err)
		vaults := map[string]string{}
		for _, owner := range owners {
			address, err := types.VaultAddress(owner)
			require.NoError(t, err)
			vaults[owner] = address.String()
		}
		files := map[string]interface{}{
			"before.json": map[string]interface{}{"initial_height": "2", "app_state": beforeExport},
			"after.json": map[string]interface{}{"initial_height": "3", "app_state": afterExport},
			"blocks.json": []interface{}{map[string]interface{}{"height": "2", "finalize_block_events": result.Events, "txs_results": result.TxResults}},
			"vault-map.json": vaults,
		}
		for name, value := range files {
			data, err := json.MarshalIndent(value, "", "  ")
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(directory, name), data, 0600))
		}
		t.Logf("SDK upgrade-block event fixture: %s", directory)
	}
}
