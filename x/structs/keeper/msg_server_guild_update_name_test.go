package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgGuildUpdateName(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	gs := testCreateGuild(k, ctx)

	testCases := []struct {
		name      string
		input     *types.MsgGuildUpdateName
		expErr    bool
		expErrMsg string
		skip      bool
	}{
		{
			name: "valid name update",
			input: &types.MsgGuildUpdateName{
				Creator: gs.GuildOwner.Creator,
				GuildId: gs.Guild.Id,
				Name:    "test guild",
			},
			expErr: false,
		},
		{
			name: "name too short",
			input: &types.MsgGuildUpdateName{
				Creator: gs.GuildOwner.Creator,
				GuildId: gs.Guild.Id,
				Name:    "ab",
			},
			expErr:    true,
			expErrMsg: "must be 3-20 characters",
		},
		{
			name: "name looks like object id",
			input: &types.MsgGuildUpdateName{
				Creator: gs.GuildOwner.Creator,
				GuildId: gs.Guild.Id,
				Name:    "1-23",
			},
			expErr:    true,
			expErrMsg: "cannot resemble an object ID",
		},
		{
			name: "no permission",
			input: &types.MsgGuildUpdateName{
				Creator: sdk.AccAddress("noperms12345678901234567890123456789").String(),
				GuildId: gs.Guild.Id,
				Name:    "cool guild",
			},
			expErr:    true,
			expErrMsg: "has no permissions",
			skip:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skip {
				t.Skip("Skipping test - error condition not easily testable with current cache system")
			}

			resp, err := ms.GuildUpdateName(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				updatedGuild, found := k.GetGuild(ctx, gs.Guild.Id)
				require.True(t, found)
				require.Equal(t, tc.input.Name, updatedGuild.Name)
			}
		})
	}

	// Test guild name uniqueness
	t.Run("name uniqueness", func(t *testing.T) {
		// Set up a second guild
		gs2 := testCreateGuild(k, ctx)

		// First guild already has the name "test guild" from the valid test above
		// Try to set the second guild to the same name (case-insensitive)
		_, err := ms.GuildUpdateName(wctx, &types.MsgGuildUpdateName{
			Creator: gs2.GuildOwner.Creator,
			GuildId: gs2.Guild.Id,
			Name:    "Test Guild",
		})
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrGuildNameTaken)
	})

	// Test name release on rename
	t.Run("name released on rename", func(t *testing.T) {
		// Rename first guild
		resp, err := ms.GuildUpdateName(wctx, &types.MsgGuildUpdateName{
			Creator: gs.GuildOwner.Creator,
			GuildId: gs.Guild.Id,
			Name:    "renamed guild",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Now the old name should be available for the second guild
		gs2 := testCreateGuild(k, ctx)
		resp2, err := ms.GuildUpdateName(wctx, &types.MsgGuildUpdateName{
			Creator: gs2.GuildOwner.Creator,
			GuildId: gs2.Guild.Id,
			Name:    "test guild",
		})
		require.NoError(t, err)
		require.NotNil(t, resp2)
	})
}

func TestMsgGuildUpdateNameIndex(t *testing.T) {
	k, _, ctx := setupMsgServer(t)

	k.SetGuildNameIndex(ctx, "Alpha Guild", "1-1")

	guildId, found := k.GetGuildIdByName(ctx, "alpha guild")
	require.True(t, found)
	require.Equal(t, "1-1", guildId)

	_, found = k.GetGuildIdByName(ctx, "nonexistent")
	require.False(t, found)

	k.RemoveGuildNameIndex(ctx, "Alpha Guild")
	_, found = k.GetGuildIdByName(ctx, "alpha guild")
	require.False(t, found)
}

func TestMsgGuildUpdateNameRequiresPermUpdate(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	gs := testCreateGuild(k, ctx)

	// Create a member with PermGuildUGCUpdate but NOT PermUpdate
	memberAcc := sdk.AccAddress("guildmember1234567890123456789012345")
	member := types.Player{
		Creator:        memberAcc.String(),
		PrimaryAddress: memberAcc.String(),
		GuildId:        gs.Guild.Id,
		GuildRank:      1,
	}
	member = testAppendPlayer(k, ctx, member)

	guildPermId := keeperlib.GetObjectPermissionIDBytes(gs.Guild.Id, member.Id)
	testPermissionAdd(k, ctx, guildPermId, types.PermGuildUGCUpdate)

	// PermGuildUGCUpdate alone should NOT be sufficient for guild rename
	_, err := ms.GuildUpdateName(wctx, &types.MsgGuildUpdateName{
		Creator: member.Creator,
		GuildId: gs.Guild.Id,
		Name:    "ugc guild",
	})
	require.Error(t, err)

	// Grant PermUpdate and it should succeed
	testPermissionAdd(k, ctx, guildPermId, types.PermUpdate)

	resp, err := ms.GuildUpdateName(wctx, &types.MsgGuildUpdateName{
		Creator: member.Creator,
		GuildId: gs.Guild.Id,
		Name:    "ugc guild",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	updatedGuild, found := k.GetGuild(ctx, gs.Guild.Id)
	require.True(t, found)
	require.Equal(t, "ugc guild", updatedGuild.Name)
}
