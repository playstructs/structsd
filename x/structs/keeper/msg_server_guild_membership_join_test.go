package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"structs/x/structs/types"
)

func TestMsgGuildMembershipJoin(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	gs := testCreateGuild(k, ctx)

	joinerAcc := sdk.AccAddress("join_test_addr_pad01")
	joiner := types.Player{
		Creator:        joinerAcc.String(),
		PrimaryAddress: joinerAcc.String(),
	}
	joiner = testAppendPlayer(k, ctx, joiner)

	t.Run("valid direct join", func(t *testing.T) {
		resp, err := ms.GuildMembershipJoin(wctx, &types.MsgGuildMembershipJoin{
			Creator:    joiner.Creator,
			GuildId:    gs.Guild.Id,
			PlayerId:   joiner.Id,
			InfusionId: []string{},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		p, found := k.GetPlayer(ctx, joiner.Id)
		require.True(t, found)
		require.Equal(t, gs.Guild.Id, p.GuildId)
	})

	t.Run("guild not found", func(t *testing.T) {
		_, err := ms.GuildMembershipJoin(wctx, &types.MsgGuildMembershipJoin{
			Creator:    joiner.Creator,
			GuildId:    "0-999",
			PlayerId:   joiner.Id,
			InfusionId: []string{},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("already a member", func(t *testing.T) {
		memberAcc := sdk.AccAddress("join_alrmemb_pad0001")
		member := types.Player{
			Creator:        memberAcc.String(),
			PrimaryAddress: memberAcc.String(),
			GuildId:        gs.Guild.Id,
		}
		member = testAppendPlayer(k, ctx, member)
		member.GuildId = gs.Guild.Id
		k.SetPlayer(ctx, member)

		_, err := ms.GuildMembershipJoin(wctx, &types.MsgGuildMembershipJoin{
			Creator:    member.Creator,
			GuildId:    gs.Guild.Id,
			PlayerId:   member.Id,
			InfusionId: []string{},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "already a member")
	})

	t.Run("unregistered creator", func(t *testing.T) {
		unregAcc := sdk.AccAddress("unreg_creator_pad000")
		_, err := ms.GuildMembershipJoin(wctx, &types.MsgGuildMembershipJoin{
			Creator:    unregAcc.String(),
			GuildId:    gs.Guild.Id,
			PlayerId:   joiner.Id,
			InfusionId: []string{},
		})
		require.Error(t, err)
	})
}

/* TestGuildMembershipJoinRejectsDuplicateInfusions is the regression on the
 * join minimum being cleared with fuel the player does not hold.
 *
 * msg.InfusionId is an unconstrained repeated field, and the branch taken when
 * the infusion's reactor is already inside the destination guild is pure
 * accumulation - it reads the record and adds its fuel, mutating nothing. So
 * naming one owned infusion N times counted its fuel N times, and
 * JoinInfusionMinimum is the only economic gate on a direct join.
 *
 * The two halves are asserted together deliberately: that one copy is genuinely
 * short is what makes the duplicate rejection meaningful rather than incidental.
 */
func TestGuildMembershipJoinRejectsDuplicateInfusions(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	gs := testCreateGuild(k, ctx)

	// The reactor has to be inside the guild, or the handler takes the migration
	// branch instead of the accumulate-only one this covers.
	reactor := gs.Reactor
	reactor.GuildId = gs.Guild.Id
	k.SetReactor(ctx, reactor)

	const infusionFuel = 600
	guild, found := k.GetGuild(ctx, gs.Guild.Id)
	require.True(t, found)
	// Deliberately more than one infusion's worth and less than two.
	guild.JoinInfusionMinimum = 1000
	k.SetGuild(ctx, guild)

	joinerAcc := sdk.AccAddress("dupe_join_addr_pad01")
	joiner := types.Player{
		Creator:        joinerAcc.String(),
		PrimaryAddress: joinerAcc.String(),
	}
	joiner = testAppendPlayer(k, ctx, joiner)
	require.NoError(t, k.SetPlayerIndexForAddress(ctx, joinerAcc.String(), joiner.Index))

	testAppendInfusion(k, ctx, types.Infusion{
		DestinationId:   reactor.Id,
		DestinationType: types.ObjectType_reactor,
		Address:         joinerAcc.String(),
		PlayerId:        joiner.Id,
		Fuel:            infusionFuel,
	})
	infusionId := reactor.Id + "-" + joinerAcc.String()

	t.Run("one infusion is genuinely short of the minimum", func(t *testing.T) {
		_, err := ms.GuildMembershipJoin(wctx, &types.MsgGuildMembershipJoin{
			Creator:    joiner.Creator,
			GuildId:    gs.Guild.Id,
			PlayerId:   joiner.Id,
			InfusionId: []string{infusionId},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "join infusion minimum not met")
	})

	t.Run("naming it twice must not clear the minimum", func(t *testing.T) {
		_, err := ms.GuildMembershipJoin(wctx, &types.MsgGuildMembershipJoin{
			Creator:    joiner.Creator,
			GuildId:    gs.Guild.Id,
			PlayerId:   joiner.Id,
			InfusionId: []string{infusionId, infusionId},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "listed more than once")

		p, playerFound := k.GetPlayer(ctx, joiner.Id)
		require.True(t, playerFound)
		require.Empty(t, p.GuildId, "the join must not have gone through")
	})

	t.Run("a duplicate anywhere in a longer list is caught", func(t *testing.T) {
		_, err := ms.GuildMembershipJoin(wctx, &types.MsgGuildMembershipJoin{
			Creator:    joiner.Creator,
			GuildId:    gs.Guild.Id,
			PlayerId:   joiner.Id,
			InfusionId: []string{infusionId, infusionId, infusionId},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "listed more than once")
	})

	// Enough genuine fuel still joins, so the fix rejects repetition rather than
	// the infusion list itself.
	t.Run("a single sufficient infusion still joins", func(t *testing.T) {
		guild, guildFound := k.GetGuild(ctx, gs.Guild.Id)
		require.True(t, guildFound)
		guild.JoinInfusionMinimum = infusionFuel
		k.SetGuild(ctx, guild)

		_, err := ms.GuildMembershipJoin(wctx, &types.MsgGuildMembershipJoin{
			Creator:    joiner.Creator,
			GuildId:    gs.Guild.Id,
			PlayerId:   joiner.Id,
			InfusionId: []string{infusionId},
		})
		require.NoError(t, err)

		p, playerFound := k.GetPlayer(ctx, joiner.Id)
		require.True(t, playerFound)
		require.Equal(t, gs.Guild.Id, p.GuildId)
	})
}
