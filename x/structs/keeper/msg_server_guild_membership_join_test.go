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
