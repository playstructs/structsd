package app_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
)

// Exercise ownership/accounting through real SDK hooks. This does not claim a
// guild vault signs SDK messages; authorized Structs handlers are exercised
// separately in vault_reactor_handlers_test.go.
func TestGuildVaultInfusionCreditsGuildWithoutCreatingPlayer(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	_, _, playerID := newTransferPlayer(t, bApp, ctx)
	guild := types.CreateEmptyGuild()
	guild.Id, guild.Index, guild.Owner = "0-995", 995, playerID
	bApp.StructsKeeper.SetGuild(ctx, guild)
	vaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(guild.Id)
	require.NoError(t, err)
	vault := vaults[0]
	playersBefore := bApp.StructsKeeper.GetPlayerCount(ctx)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	delegate(t, bApp, ctx, server, denom, vault, validator, math.NewInt(1000000))
	reactorID := jailGateReactorId(t, bApp, ctx, validator)
	infusion, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, vault.String())
	require.True(t, found)
	require.Equal(t, guild.Id, infusion.OwnerId)
	require.Empty(t, infusion.PlayerId)
	require.Equal(t, uint64(960000), redelegationCapacity(t, bApp, ctx, guild.Id))
	require.Zero(t, redelegationCapacity(t, bApp, ctx, playerID), "guild owner does not personally own guild stake")
	cc := bApp.StructsKeeper.NewCurrentContext(ctx)
	cache := cc.GetInfusion(reactorID, vault.String())
	owner, err := cache.GetOwner()
	require.NoError(t, err)
	require.Same(t, cc.GetGuild(guild.Id), owner)

	cc.CommitAll()
	_, err = server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(vault.String(), validator.String(), sdk.NewInt64Coin(denom, 250000)))
	require.NoError(t, err)
	after, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, vault.String())
	require.True(t, found)
	require.Equal(t, guild.Id, after.AssetOwnerID())
	require.Equal(t, uint64(750000), after.Fuel)
	require.Equal(t, uint64(250000), after.Defusing)
	require.Equal(t, uint64(720000), redelegationCapacity(t, bApp, ctx, guild.Id))
	require.NoError(t, after.ValidateAssetOwner())
	require.Equal(t, playersBefore, bApp.StructsKeeper.GetPlayerCount(ctx))
	require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, vault.String()))
	var last *types.Infusion
	for _, event := range migrationEvents[*types.EventInfusion](t, ctx, "EventInfusion") {
		if event.Infusion.Address == vault.String() {
			last = event.Infusion
		}
	}
	require.NotNil(t, last)
	requireInfusionEventEqual(t, after, *last, "indexers receive explicit guild ownership and the updated weight")
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}
