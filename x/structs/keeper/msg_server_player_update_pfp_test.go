package keeper_test

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"structs/x/structs/types"
)

func TestMsgPlayerUpdatePfp(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	playerAcc := sdk.AccAddress("playerupdatepfp12345678901234567890")
	player := types.Player{
		Creator:        playerAcc.String(),
		PrimaryAddress: playerAcc.String(),
	}
	player = testAppendPlayer(k, ctx, player)

	testCases := []struct {
		name      string
		input     *types.MsgPlayerUpdatePfp
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid pfp",
			input: &types.MsgPlayerUpdatePfp{
				Creator:  player.Creator,
				PlayerId: player.Id,
				Pfp:      "https://example.com/avatar.png",
			},
			expErr: false,
		},
		{
			name: "clear pfp",
			input: &types.MsgPlayerUpdatePfp{
				Creator:  player.Creator,
				PlayerId: player.Id,
				Pfp:      "",
			},
			expErr: false,
		},
		{
			name: "pfp too long",
			input: &types.MsgPlayerUpdatePfp{
				Creator:  player.Creator,
				PlayerId: player.Id,
				Pfp:      strings.Repeat("x", types.MaxPfpLength+1),
			},
			expErr:    true,
			expErrMsg: "at most",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.PlayerUpdatePfp(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				updatedPlayer, found := k.GetPlayer(ctx, player.Id)
				require.True(t, found)
				require.Equal(t, tc.input.Pfp, updatedPlayer.Pfp)
			}
		})
	}
}
