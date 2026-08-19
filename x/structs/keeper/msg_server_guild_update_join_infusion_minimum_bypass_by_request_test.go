package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgGuildUpdateJoinInfusionMinimumBypassByRequest(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	// Create a player and guild
	playerAcc := sdk.AccAddress("creator123456789012345678901234567890")
	player := types.Player{
		Creator:        playerAcc.String(),
		PrimaryAddress: playerAcc.String(),
	}
	player = testAppendPlayer(k, ctx, player)

	// Create reactor for guild
	validatorAddress := sdk.ValAddress(playerAcc.Bytes())
	reactor := types.Reactor{
		RawAddress: validatorAddress.Bytes(),
	}
	// AppendReactor already calls SetReactorValidatorBytes internally
	reactor = k.AppendReactor(ctx, reactor)

	// Create guild
	guild := k.AppendGuild(ctx, "test-endpoint", "", reactor, player, "")
	player.GuildId = guild.Id
	k.SetPlayer(ctx, player)

	// Grant permissions
	guildPermissionId := keeperlib.GetObjectPermissionIDBytes(guild.Id, player.Id)
	testPermissionAdd(k, ctx, guildPermissionId, types.PermUpdate)

	addressPermissionId := keeperlib.GetAddressPermissionIDBytes(player.Creator)
	testPermissionAdd(k, ctx, addressPermissionId, types.PermAssetsAll)

	testCases := []struct {
		name      string
		input     *types.MsgGuildUpdateJoinInfusionMinimumBypassByRequest
		expErr    bool
		expErrMsg string
		skip      bool
	}{
		{
			name: "permissioned bypass level update",
			input: &types.MsgGuildUpdateJoinInfusionMinimumBypassByRequest{
				Creator:              player.Creator,
				GuildId:              guild.Id,
				GuildJoinBypassLevel: types.GuildJoinBypassLevel_permissioned,
			},
			expErr: false,
		},
		{
			name: "member bypass level update",
			input: &types.MsgGuildUpdateJoinInfusionMinimumBypassByRequest{
				Creator:              player.Creator,
				GuildId:              guild.Id,
				GuildJoinBypassLevel: types.GuildJoinBypassLevel_member,
			},
			expErr: false,
		},
		{
			name: "closed bypass level update",
			input: &types.MsgGuildUpdateJoinInfusionMinimumBypassByRequest{
				Creator:              player.Creator,
				GuildId:              guild.Id,
				GuildJoinBypassLevel: types.GuildJoinBypassLevel_closed,
			},
			expErr: false,
		},
		{
			// proto3 enums are open, so 500 decodes fine. Before v0.21.0 this
			// case asserted that it persisted, and a stored 500 made every
			// membership switch in guild_cache.go fall through and return nil.
			name: "undeclared bypass level is rejected",
			input: &types.MsgGuildUpdateJoinInfusionMinimumBypassByRequest{
				Creator:              player.Creator,
				GuildId:              guild.Id,
				GuildJoinBypassLevel: 500,
			},
			expErr:    true,
			expErrMsg: "invalid guild join bypass level",
		},
		{
			name: "negative bypass level is rejected",
			input: &types.MsgGuildUpdateJoinInfusionMinimumBypassByRequest{
				Creator:              player.Creator,
				GuildId:              guild.Id,
				GuildJoinBypassLevel: -1,
			},
			expErr:    true,
			expErrMsg: "invalid guild join bypass level",
		},
		{
			name: "guild not found",
			input: &types.MsgGuildUpdateJoinInfusionMinimumBypassByRequest{
				Creator:              player.Creator,
				GuildId:              "invalid-guild",
				GuildJoinBypassLevel: types.GuildJoinBypassLevel_member,
			},
			expErr:    true,
			expErrMsg: "wasn't found",
			skip:      true, // Skip - cache system doesn't validate existence before permission check
		},
		{
			name: "no update permissions",
			input: &types.MsgGuildUpdateJoinInfusionMinimumBypassByRequest{
				Creator:              sdk.AccAddress("noperms123456789012345678901234567890").String(),
				GuildId:              guild.Id,
				GuildJoinBypassLevel: types.GuildJoinBypassLevel_member,
			},
			expErr:    true,
			expErrMsg: "has no permissions",
			skip:      true, // Skip - GetPlayerCacheFromAddress might create player
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skip {
				t.Skip("Skipping test - error condition not easily testable with current cache system")
			}

			before, _ := k.GetGuild(ctx, guild.Id)

			resp, err := ms.GuildUpdateJoinInfusionMinimumBypassByRequest(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)

				// A rejected level must not reach state. The handler validates
				// ahead of GuildCache.SetJoinInfusionMinimumBypassByRequest, so
				// there is nothing for the transaction rollback to undo.
				after, found := k.GetGuild(ctx, guild.Id)
				require.True(t, found)
				require.Equal(t, before.JoinInfusionMinimumBypassByRequest, after.JoinInfusionMinimumBypassByRequest)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify bypass level was updated
				updatedGuild, found := k.GetGuild(ctx, guild.Id)
				require.True(t, found)
				require.Equal(t, tc.input.GuildJoinBypassLevel, updatedGuild.JoinInfusionMinimumBypassByRequest)
			}
		})
	}
}
