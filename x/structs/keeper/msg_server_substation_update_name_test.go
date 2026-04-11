package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgSubstationUpdateName(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	playerAcc := sdk.AccAddress("substationname12345678901234567890")
	player := types.Player{
		Creator:        playerAcc.String(),
		PrimaryAddress: playerAcc.String(),
	}
	player = testAppendPlayer(k, ctx, player)

	alloc := types.Allocation{
		SourceObjectId: player.Id,
		Controller:     player.Id,
		Type:           types.AllocationType_static,
	}
	alloc, _ = testAppendAllocation(k, ctx, alloc, 100)

	substation, _, _ := testAppendSubstation(k, ctx, alloc, player)
	substationPermId := keeperlib.GetObjectPermissionIDBytes(substation.Id, player.Id)
	testPermissionAdd(k, ctx, substationPermId, types.PermUpdate)

	testCases := []struct {
		name      string
		input     *types.MsgSubstationUpdateName
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid name",
			input: &types.MsgSubstationUpdateName{
				Creator:      player.Creator,
				SubstationId: substation.Id,
				Name:         "Power Station",
			},
			expErr: false,
		},
		{
			name: "name too short",
			input: &types.MsgSubstationUpdateName{
				Creator:      player.Creator,
				SubstationId: substation.Id,
				Name:         "ab",
			},
			expErr:    true,
			expErrMsg: "must be 3-20 characters",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.SubstationUpdateName(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				updatedSubstation, found := k.GetSubstation(ctx, substation.Id)
				require.True(t, found)
				require.Equal(t, tc.input.Name, updatedSubstation.Name)
			}
		})
	}
}
