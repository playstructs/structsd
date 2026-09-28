package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// Rank never substitutes for administrative authority or signing-key bits.
func TestPlayerUpdateGuildRankKeepsTheSigningKeyCeiling(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	gs := testCreateGuild(k, ctx)

	// A well-ranked member, and a lower-ranked one for them to act on.
	officerAcc := sdk.AccAddress("rankceiling_officer_pad1")
	officer := testAppendPlayer(k, ctx, types.Player{
		Creator:        officerAcc.String(),
		PrimaryAddress: officerAcc.String(),
		GuildId:        gs.Guild.Id,
		GuildRank:      2,
	})

	juniorAcc := sdk.AccAddress("rankceiling_junior_pad01")
	junior := testAppendPlayer(k, ctx, types.Player{
		Creator:        juniorAcc.String(),
		PrimaryAddress: juniorAcc.String(),
		GuildId:        gs.Guild.Id,
		GuildRank:      50,
	})

	// A delegated key of the officer, scoped to play and nothing else.
	limitedAcc := sdk.AccAddress("rankceiling_limited_pad1")
	_ = k.SetPlayerIndexForAddress(ctx, limitedAcc.String(), officer.Index)
	k.SetPermissionsByBytes(ctx, keeperlib.GetAddressPermissionIDBytes(limitedAcc.String()), types.PermPlay)

	t.Run("a restricted key cannot use its player's rank", func(t *testing.T) {
		_, err := ms.PlayerUpdateGuildRank(wctx, &types.MsgPlayerUpdateGuildRank{
			Creator:   limitedAcc.String(),
			GuildId:   gs.Guild.Id,
			PlayerId:  junior.Id,
			GuildRank: 2,
		})
		require.Error(t, err, "rank substitutes for standing on the guild, not for the key's own bits")

		after, found := k.GetPlayer(ctx, junior.Id)
		require.True(t, found)
		require.Equal(t, uint64(50), after.GuildRank, "the promotion must not have landed")
	})

	// The primary key also fails without an explicit administrative grant.
	t.Run("rank alone cannot promote", func(t *testing.T) {
		_, err := ms.PlayerUpdateGuildRank(wctx, &types.MsgPlayerUpdateGuildRank{
			Creator:   officer.Creator,
			GuildId:   gs.Guild.Id,
			PlayerId:  junior.Id,
			GuildRank: 2,
		})
		require.Error(t, err)

		after, found := k.GetPlayer(ctx, junior.Id)
		require.True(t, found)
		require.Equal(t, uint64(50), after.GuildRank)
	})

	// An explicit administrator can manage ranks regardless of their own rank.
	t.Run("explicit admin can promote", func(t *testing.T) {
        k.SetPermissionsByBytes(ctx, keeperlib.GetObjectPermissionIDBytes(gs.Guild.Id, officer.Id), types.PermAdmin)
		peerAcc := sdk.AccAddress("rankceiling_peer_pad0001")
		peer := testAppendPlayer(k, ctx, types.Player{
			Creator:        peerAcc.String(),
			PrimaryAddress: peerAcc.String(),
			GuildId:        gs.Guild.Id,
			GuildRank:      40,
		})

		_, err := ms.PlayerUpdateGuildRank(wctx, &types.MsgPlayerUpdateGuildRank{
			Creator:   officer.Creator,
			GuildId:   gs.Guild.Id,
			PlayerId:  peer.Id,
			GuildRank: 1,
		})
		require.NoError(t, err)
        after, found := k.GetPlayer(ctx, peer.Id)
        require.True(t, found)
        require.Equal(t, uint64(1), after.GuildRank)
	})
}
