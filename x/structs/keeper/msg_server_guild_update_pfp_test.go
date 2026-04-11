package keeper_test

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"structs/x/structs/types"
)

func TestMsgGuildUpdatePfp(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	gs := testCreateGuild(k, ctx)

	testCases := []struct {
		name      string
		input     *types.MsgGuildUpdatePfp
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid pfp update",
			input: &types.MsgGuildUpdatePfp{
				Creator: gs.GuildOwner.Creator,
				GuildId: gs.Guild.Id,
				Pfp:     "https://example.com/image.png",
			},
			expErr: false,
		},
		{
			name: "clear pfp",
			input: &types.MsgGuildUpdatePfp{
				Creator: gs.GuildOwner.Creator,
				GuildId: gs.Guild.Id,
				Pfp:     "",
			},
			expErr: false,
		},
		{
			name: "pfp too long",
			input: &types.MsgGuildUpdatePfp{
				Creator: gs.GuildOwner.Creator,
				GuildId: gs.Guild.Id,
				Pfp:     strings.Repeat("a", types.MaxPfpLength+1),
			},
			expErr:    true,
			expErrMsg: "at most",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.GuildUpdatePfp(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				updatedGuild, found := k.GetGuild(ctx, gs.Guild.Id)
				require.True(t, found)
				require.Equal(t, tc.input.Pfp, updatedGuild.Pfp)
			}
		})
	}
}
