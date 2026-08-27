package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

/* TestPlayerUpdateGuildRankKeepsTheSigningKeyCeiling is the regression on rank
 * standing in for a permission it was only ever meant to stand beside.
 *
 * PermissionCheck fails for two unrelated reasons: the signing key does not
 * carry the bit, or it does but the player has no standing on the object. This
 * handler treated both alike and fell through to a rank comparison that reads
 * only player-level attributes - so a deliberately restricted secondary key
 * inherited its player's whole rank authority, and could re-rank every member
 * below them.
 *
 * It is the only rank check in the module that substitutes for a permission.
 * GuildUpdateEntryRank and the kick path both apply their rank bound *after* a
 * PermissionCheck that has already enforced the ceiling.
 */
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

	// The control: the officer's own primary address holds PermAll, so the rank
	// fallback still does exactly what it is for.
	t.Run("the officer's own key still outranks the junior", func(t *testing.T) {
		_, err := ms.PlayerUpdateGuildRank(wctx, &types.MsgPlayerUpdateGuildRank{
			Creator:   officer.Creator,
			GuildId:   gs.Guild.Id,
			PlayerId:  junior.Id,
			GuildRank: 2,
		})
		require.NoError(t, err)

		after, found := k.GetPlayer(ctx, junior.Id)
		require.True(t, found)
		require.Equal(t, uint64(2), after.GuildRank)
	})

	// And the rank bounds themselves still hold: nobody promotes past their own.
	t.Run("the rank bounds are unchanged", func(t *testing.T) {
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
		require.Error(t, err, "an actor may not hand out a rank better than their own")
	})
}
