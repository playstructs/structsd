package app_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
	"github.com/stretchr/testify/require"
	v22 "structs/app/upgrades/v0_22_0"
	"structs/x/structs/types"
)

func TestDiversityCompleteUpgradeRollbackAndBlockPublication(t *testing.T) {
	bApp, ctx := setupJailGateAppWithValidators(t, 4)
	legacyICA := icahosttypes.NewParams(true, []string{"*"})
	bApp.ICAHostKeeper.SetParams(ctx, legacyICA)
	k := bApp.StructsKeeper
	// Model legacy flat accounting before the unshipped bonus was introduced.
	k.SuspendReactorDiversity(ctx)
	primary, secondary, owner := newTransferPlayer(t, bApp, ctx)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	staking := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	for _, validator := range validators[:2] {
		address, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
		require.NoError(t, err)
		delegate(t, bApp, ctx, staking, denom, primary, address, math.NewInt(1000000))
	}
	first, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	delegate(t, bApp, ctx, staking, denom, secondary, first, math.NewInt(500000))
	require.Equal(t, uint64(2400000), redelegationCapacity(t, bApp, ctx, owner))
	_, found := k.GetOwnerReactorDiversity(ctx, owner)
	require.False(t, found)
	vault, err := types.VaultAddress(owner)
	require.NoError(t, err)
	restoreLegacyCascadeQueueFixture(t, bApp, ctx)
	versions := bApp.ModuleManager.GetVersionMap()
	versions[banktypes.ModuleName]--
	config := module.NewConfigurator(bApp.AppCodec(), nil, nil)
	called := false
	require.NoError(t, config.RegisterMigration(banktypes.ModuleName, versions[banktypes.ModuleName], func(work sdk.Context) error {
		called = true
		_, provisioned := k.GetVaultOwner(work, vault)
		require.True(t, provisioned, "earlier vault provisioning must have run before injected failure")
		_, err := bApp.StakingKeeper.GetDelegation(work, primary, first)
		require.NoError(t, err, "legacy stake remains at its original address")
		// Fail in diversity reconstruction, after all ordinary module migrations.
		work.KVStore(bApp.GetKey(types.StoreKey)).Set([]byte("ReactorDiversity/owner/1-999999"), []byte{0xff})
		return nil
	}))
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	before := applicationAssetStateRows(bApp, ctx)
	beforeStaking := bApp.StakingKeeper.ExportGenesis(ctx)
	_, err = v22.CreateUpgradeHandler(bApp.ModuleManager, config, delegationUpgradeKeepers(bApp))(ctx, upgradetypes.Plan{Name: v22.UpgradeName}, versions)
	require.Error(t, err)
	require.True(t, called)
	require.Equal(t, before, applicationAssetStateRows(bApp, ctx))
	require.Equal(t, beforeStaking, bApp.StakingKeeper.ExportGenesis(ctx))
	require.Empty(t, ctx.EventManager().Events())
	require.False(t, k.ReactorDiversityReady(ctx))
	require.Equal(t, legacyICA, bApp.ICAHostKeeper.GetParams(ctx), "failed upgrade rolls back ICA policy too")
	const name = "test-diversity-complete-upgrade"
	require.NoError(t, bApp.UpgradeKeeper.ScheduleUpgrade(ctx, upgradetypes.Plan{Name: name, Height: 2}))
	_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: ctx.BlockTime()})
	require.NoError(t, err)
	_, err = bApp.Commit()
	require.NoError(t, err)
	bApp.UpgradeKeeper.SetUpgradeHandler(name, func(goCtx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		restoreLegacyCascadeQueueFixture(t, bApp, sdk.UnwrapSDKContext(goCtx))
		return v22.CreateUpgradeHandler(bApp.ModuleManager, bApp.Configurator(), delegationUpgradeKeepers(bApp))(goCtx, upgradetypes.Plan{Name: v22.UpgradeName}, vm)
	})
	now := ctx.BlockTime().Add(time.Second)
	block, err := bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 2, Time: now})
	require.NoError(t, err)
	_, err = bApp.Commit()
	require.NoError(t, err)
	final := bApp.NewContextLegacy(true, cmtproto.Header{Height: 2, ChainID: jailGateChainID, Time: now})
	require.False(t, bApp.ICAHostKeeper.GetParams(final).HostEnabled)
	require.Empty(t, bApp.ICAHostKeeper.GetParams(final).AllowMessages)
	policyEvents := 0
	for _, event := range block.Events {
		if event.Type == "ica_host_policy" {
			policyEvents++
		}
	}
	require.Equal(t, 1, policyEvents)
	query, err := k.OwnerReactorDiversity(final, &types.QueryOwnerReactorDiversityRequest{OwnerId: owner})
	require.NoError(t, err)
	require.Equal(t, "2400000", query.Stored.Power)
	require.Equal(t, "2995200000000", query.Stored.SquaredPower)
	require.Equal(t, uint64(123076), query.Stored.AppliedBonus)
	require.Equal(t, uint64(2523076), query.StoredCapacity)
	require.Equal(t, query.StoredCapacity, query.UsablePrimaryCapacity)
	require.True(t, query.Initialized)
	require.NoError(t, k.CheckReactorDiversityAccounting(final))
	requireStakingAndDistributionHealthy(t, bApp, final)
	rows := 0
	for _, event := range block.Events {
		if event.Type != "structs.structs.EventOwnerReactorDiversity" {
			continue
		}
		payload := abci.Event{Type: event.Type}
		for _, attribute := range event.Attributes {
			if attribute.Key != "mode" {
				payload.Attributes = append(payload.Attributes, attribute)
			}
		}
		decoded, err := sdk.ParseTypedEvent(payload)
		require.NoError(t, err)
		row := decoded.(*types.EventOwnerReactorDiversity).Diversity
		if row.OwnerId == owner {
			rows++
			require.Equal(t, query.Stored, row)
		}
	}
	require.Equal(t, 1, rows, "publish the final merged portfolio once in the upgrade block")
}
