package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"structs/x/structs/types"
)

func TestMsgPlayerUpdateName(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	playerAcc := sdk.AccAddress("playerupdatename1234567890123456789")
	player := types.Player{
		Creator:        playerAcc.String(),
		PrimaryAddress: playerAcc.String(),
	}
	player = testAppendPlayer(k, ctx, player)

	testCases := []struct {
		name      string
		input     *types.MsgPlayerUpdateName
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid name",
			input: &types.MsgPlayerUpdateName{
				Creator:  player.Creator,
				PlayerId: player.Id,
				Name:     "coolplayer",
			},
			expErr: false,
		},
		{
			name: "name too short",
			input: &types.MsgPlayerUpdateName{
				Creator:  player.Creator,
				PlayerId: player.Id,
				Name:     "ab",
			},
			expErr:    true,
			expErrMsg: "must be 3-20 characters",
		},
		{
			name: "name with spaces rejected for player",
			input: &types.MsgPlayerUpdateName{
				Creator:  player.Creator,
				PlayerId: player.Id,
				Name:     "my player",
			},
			expErr:    true,
			expErrMsg: "must be 3-20 characters",
		},
		{
			name: "object id pattern rejected",
			input: &types.MsgPlayerUpdateName{
				Creator:  player.Creator,
				PlayerId: player.Id,
				Name:     "2-15",
			},
			expErr:    true,
			expErrMsg: "cannot resemble an object ID",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.PlayerUpdateName(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				updatedPlayer, found := k.GetPlayer(ctx, player.Id)
				require.True(t, found)
				require.Equal(t, tc.input.Name, updatedPlayer.Name)
			}
		})
	}
}
