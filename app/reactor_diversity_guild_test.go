package app_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
)

func TestDiversityGuildVaultRemainsSeparateThroughOwnershipAndOutage(t *testing.T) {
	bApp, ctx := setupJailGateAppWithValidators(t, 4)
	k := bApp.StructsKeeper
	primary, _, playerID := newTransferPlayer(t, bApp, ctx)
	_, _, buyerID := newTransferPlayer(t, bApp, ctx)
	guild := types.CreateEmptyGuild()
	guild.Id, guild.Index, guild.Owner = "0-997", 997, playerID
	k.SetGuild(ctx, guild)
	player, found := k.GetPlayer(ctx, playerID)
	require.True(t, found)
	player.GuildId = guild.Id
	k.SetPlayer(ctx, player)
	vaults, err := k.NewCurrentContext(ctx).EnsureVaults(guild.Id)
	require.NoError(t, err)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	for _, validator := range validators[:2] {
		address, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
		require.NoError(t, err)
		delegate(t, bApp, ctx, server, denom, vaults[0], address, math.NewInt(1000000))
	}
	query := func(owner string) *types.QueryOwnerReactorDiversityResponse {
		t.Helper()
		row, err := k.OwnerReactorDiversity(ctx, &types.QueryOwnerReactorDiversityRequest{OwnerId: owner})
		require.NoError(t, err)
		require.True(t, row.LivePortfolioComplete)
		require.True(t, row.InfusionInventoryComplete)
		return row
	}
	before := query(guild.Id)
	require.Positive(t, before.Stored.AppliedBonus)
	require.Equal(t, before.Stored.AppliedBonus, before.UsableBonus)
	// A member's personal stake never dilutes or increases guild diversity.
	third, err := sdk.ValAddressFromBech32(validators[2].OperatorAddress)
	require.NoError(t, err)
	delegate(t, bApp, ctx, server, denom, primary, third, math.NewInt(3000000))
	require.Equal(t, before, query(guild.Id))
	personal := query(playerID)
	require.Equal(t, "0", personal.LiveBonus, "a single personal reactor has no diversity")
	cc := k.NewCurrentContext(ctx)
	cc.GetGuild(guild.Id).SetOwner(buyerID)
	cc.CommitAll()
	require.Equal(t, before, query(guild.Id), "selling the guild preserves its portfolio and vault")
	require.Equal(t, personal, query(playerID), "selling a guild cannot move the seller's personal stake")
	consensus, err := validators[0].GetConsAddr()
	require.NoError(t, err)
	require.NoError(t, bApp.StakingKeeper.Jail(ctx, consensus))
	after := query(guild.Id)
	require.Equal(t, before.Stored, after.Stored, "stored state awaits bounded repair")
	require.Zero(t, after.UsableBonus)
	require.Equal(t, "0", after.LiveBonus)
	require.Equal(t, (before.StoredCapacity-before.Stored.AppliedBonus)/2, after.UsablePrimaryCapacity)
	require.Equal(t, personal, query(playerID))
	// The validator-set transition schedules the ordinary bounded repair.
	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)
	k.ProcessReactorSlashReconcileQueue(ctx)
	require.NoError(t, k.CheckReactorDiversityAccounting(ctx))
	repaired := query(guild.Id)
	require.Zero(t, repaired.Stored.AppliedBonus)
	require.Equal(t, after.UsablePrimaryCapacity, repaired.StoredCapacity)
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}
