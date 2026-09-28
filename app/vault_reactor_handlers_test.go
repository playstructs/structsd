package app_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestGuildVaultReactorHandlersUseRealStaking(t *testing.T) {
	testVaultReactorHandlers(t, true, false)
}

func TestPlayerDefaultVaultReactorHandlersUseRealStaking(t *testing.T) {
	testVaultReactorHandlers(t, false, false)
}

func TestGuildOwnerIDVaultReactorHandlersUseRealStaking(t *testing.T) {
	testVaultReactorHandlers(t, true, true)
}

func testVaultReactorHandlers(t *testing.T, guildOwned, ownerSelection bool) {
	t.Helper()
	bApp, ctx := setupJailGateAppWithValidators(t, 2)
	primary, secondary, playerID := newTransferPlayer(t, bApp, ctx)
	stranger, _, _ := newTransferPlayer(t, bApp, ctx)
	guild := types.CreateEmptyGuild()
	guild.Id, guild.Index, guild.Owner = "0-997", 997, playerID
	bApp.StructsKeeper.SetGuild(ctx, guild)
	ownerID := playerID
	if guildOwned {
		ownerID = guild.Id
	}
	vault, err := types.VaultAddress(ownerID)
	require.NoError(t, err)
	delegatorArgument, ownerArgument := "", ""
	if ownerSelection {
		ownerArgument = ownerID
	}
	if guildOwned && !ownerSelection {
		_, err = bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(ownerID)
		require.NoError(t, err)
		delegatorArgument = vault.String()
	} else {
		_, exists := bApp.StructsKeeper.GetVaultOwner(ctx, vault)
		require.False(t, exists, "owner selection must provision the vault")
	}
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	require.Len(t, validators, 2)
	source, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	destination, err := sdk.ValAddressFromBech32(validators[1].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	fundRedelegationAccount(t, bApp, ctx, vault, sdk.NewCoins(sdk.NewInt64Coin(denom, 1000000)))
	ms := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
	infuse := &types.MsgReactorInfuse{Creator: stranger.String(), DelegatorAddress: vault.String(), OwnerId: ownerArgument, ValidatorAddress: source.String(), Amount: sdk.NewInt64Coin(denom, 1000000)}
	if ownerSelection {
		conflict := *infuse
		conflict.Creator = primary.String()
		conflict.DelegatorAddress = primary.String()
		_, err = ms.ReactorInfuse(ctx, &conflict)
		require.Error(t, err, "owner selection must not redirect to a signing address")
		_, exists := bApp.StructsKeeper.GetVaultOwner(ctx, vault)
		require.False(t, exists, "conflicting selector must not provision an association")
	}
	_, err = ms.ReactorInfuse(ctx, infuse)
	require.Error(t, err, "a vault association is not public spending authority")
	infuse.Creator = secondary.String()
	infuse.DelegatorAddress = delegatorArgument
	bApp.StructsKeeper.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(secondary.String()), types.PermAll &^ types.PermTokenInfuse)
	_, err = ms.ReactorInfuse(ctx, infuse)
	require.Error(t, err, "guild ownership cannot exceed the signing key's bits")
	infuse.Creator = primary.String()
	_, err = ms.ReactorInfuse(ctx, infuse)
	require.NoError(t, err)
	sourceID := jailGateReactorId(t, bApp, ctx, source)
	readSource := func() types.Infusion {
		row, found := bApp.StructsKeeper.GetInfusion(ctx, sourceID, vault.String())
		require.True(t, found)
		require.Equal(t, ownerID, row.AssetOwnerID())
		return row
	}
	require.Equal(t, uint64(1000000), readSource().Fuel)
	_, err = ms.ReactorDefuse(ctx, &types.MsgReactorDefuse{Creator: primary.String(), DelegatorAddress: delegatorArgument, OwnerId: ownerArgument, ValidatorAddress: source.String(), Amount: sdk.NewInt64Coin(denom, 250000)})
	require.NoError(t, err)
	require.Equal(t, uint64(250000), readSource().Defusing)
	_, err = ms.ReactorCancelDefusion(ctx, &types.MsgReactorCancelDefusion{Creator: primary.String(), DelegatorAddress: delegatorArgument, OwnerId: ownerArgument, ValidatorAddress: source.String(), Amount: sdk.NewInt64Coin(denom, 100000), CreationHeight: ctx.BlockHeight()})
	require.NoError(t, err)
	require.Equal(t, uint64(850000), readSource().Fuel)
	require.Equal(t, uint64(150000), readSource().Defusing)
	_, err = ms.ReactorBeginMigration(ctx, &types.MsgReactorBeginMigration{Creator: primary.String(), DelegatorAddress: delegatorArgument, OwnerId: ownerArgument, ValidatorSrcAddress: source.String(), ValidatorDstAddress: destination.String(), Amount: sdk.NewInt64Coin(denom, 200000)})
	require.NoError(t, err)
	require.Equal(t, uint64(650000), readSource().Fuel)
	dest, found := bApp.StructsKeeper.GetInfusion(ctx, jailGateReactorId(t, bApp, ctx, destination), vault.String())
	require.True(t, found)
	require.Equal(t, ownerID, dest.AssetOwnerID())
	require.Equal(t, uint64(200000), dest.Fuel)
	associatedOwner, exists := bApp.StructsKeeper.GetVaultOwner(ctx, vault)
	require.True(t, exists)
	require.Equal(t, ownerID, associatedOwner)
	delegatorEvents := 0
	for _, event := range ctx.EventManager().Events() {
		for _, attribute := range event.Attributes {
			if attribute.Key == "delegator" {
				require.Equal(t, vault.String(), attribute.Value, "staking events must name the resolved vault")
				delegatorEvents++
			}
		}
	}
	require.GreaterOrEqual(t, delegatorEvents, 4)
	require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, vault.String()))
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}
