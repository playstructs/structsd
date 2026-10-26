package app_test

import (
	"testing"

	"cosmossdk.io/math"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Exercise the actual staking hooks: keeper mocks cannot detect source capacity
// surviving a full redelegation or an outer membership cache clobbering a hook.
func TestGuildMembershipJoinRedelegatesPlayerVault(t *testing.T) {
	t.Run("healthy", func(t *testing.T) { testGuildMembershipJoinRedelegatesPlayerVault(t, false) })
	t.Run("pending-slash", func(t *testing.T) { testGuildMembershipJoinRedelegatesPlayerVault(t, true) })
}

func testGuildMembershipJoinRedelegatesPlayerVault(t *testing.T, slash bool) {
	t.Helper()
	bApp, ctx := setupJailGateApp(t)
	primary, _, playerID := newTransferPlayer(t, bApp, ctx)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	staking := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	source := createRedelegationValidator(t, bApp, ctx, staking, denom, "join-source")
	destination := createRedelegationValidator(t, bApp, ctx, staking, denom, "join-destination")
	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)


	vaults, err := bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(playerID)
	require.NoError(t, err)
	vault := vaults[0]
	stake := math.NewInt(1000000)
	delegate(t, bApp, ctx, staking, denom, vault, source, stake)
	sourceID := jailGateReactorId(t, bApp, ctx, source)
	destinationID := jailGateReactorId(t, bApp, ctx, destination)
	if slash {
		validator, err := bApp.StakingKeeper.GetValidator(ctx, source)
		require.NoError(t, err)
		consensus, err := validator.GetConsAddr()
		require.NoError(t, err)
		_, err = bApp.StakingKeeper.Slash(ctx, consensus, ctx.BlockHeight(), validator.ConsensusPower(bApp.StakingKeeper.PowerReduction(ctx)), math.LegacyNewDecWithPrec(5, 2))
		require.NoError(t, err)
		stale, found := bApp.StructsKeeper.GetInfusion(ctx, sourceID, vault.String())
		require.True(t, found)
		require.Equal(t, uint64(1000000), stale.Fuel)
		validator, err = bApp.StakingKeeper.GetValidator(ctx, source)
		require.NoError(t, err)
		delegation, err := bApp.StakingKeeper.GetDelegation(ctx, vault, source)
		require.NoError(t, err)
		stake = validator.TokensFromShares(delegation.Shares).TruncateInt()
		require.Equal(t, int64(950000), stake.Int64())
	}
	guild := types.CreateEmptyGuild()
	guild.Id, guild.Index, guild.Owner, guild.Creator = "0-913", 913, playerID, primary.String()
	guild.PrimaryReactorId = destinationID
	guild.JoinInfusionMinimum = stake.Uint64() + 1
	bApp.StructsKeeper.SetGuild(ctx, guild)
	ms := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
	msg := &types.MsgGuildMembershipJoin{
		Creator: primary.String(), PlayerId: playerID, GuildId: guild.Id,
		InfusionId: []string{sourceID + "-" + vault.String()},
	}
	_, err = ms.GuildMembershipJoin(ctx, msg)
	require.ErrorContains(t, err, "join infusion minimum not met")
	_, err = bApp.StakingKeeper.GetDelegation(ctx, vault, source)
	require.NoError(t, err, "the minimum must be checked before moving stake")
	guild.JoinInfusionMinimum = stake.Uint64()
	bApp.StructsKeeper.SetGuild(ctx, guild)
	_, err = ms.GuildMembershipJoin(ctx, msg)
	require.NoError(t, err)
	player, found := bApp.StructsKeeper.GetPlayer(ctx, playerID)
	require.True(t, found)
	require.Equal(t, guild.Id, player.GuildId)
	require.Equal(t, guild.EntryRank, player.GuildRank)
	require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, vault.String()))
	_, err = bApp.StakingKeeper.GetDelegation(ctx, vault, source)
	require.Error(t, err)
	_, err = bApp.StakingKeeper.GetDelegation(ctx, vault, destination)
	require.NoError(t, err)
	if old, found := bApp.StructsKeeper.GetInfusion(ctx, sourceID, vault.String()); found {
		require.Zero(t, old.Fuel)
		require.Zero(t, old.Power)
	}
	infusion, found := bApp.StructsKeeper.GetInfusion(ctx, destinationID, vault.String())
	require.True(t, found)
	require.Equal(t, playerID, infusion.PlayerId)
	require.Equal(t, stake.Uint64(), infusion.Fuel)
	require.Equal(t, stake.Uint64(), infusion.Power)
	_, _, playerPower := infusion.GetPowerDistribution()
	require.Equal(t, playerPower, redelegationCapacity(t, bApp, ctx, playerID))
	_, err = bApp.StakingKeeper.GetRedelegation(ctx, vault, source, destination)
	require.NoError(t, err, "the pending slashable record must retain the vault address")
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}
