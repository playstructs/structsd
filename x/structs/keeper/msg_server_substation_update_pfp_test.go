package keeper_test

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgSubstationUpdatePfp(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	playerAcc := sdk.AccAddress("substationpfp123456789012345678901")
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
		input     *types.MsgSubstationUpdatePfp
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid pfp",
			input: &types.MsgSubstationUpdatePfp{
				Creator:      player.Creator,
				SubstationId: substation.Id,
				Pfp:          "https://example.com/sub.png",
			},
			expErr: false,
		},
		{
			name: "pfp too long",
			input: &types.MsgSubstationUpdatePfp{
				Creator:      player.Creator,
				SubstationId: substation.Id,
				Pfp:          strings.Repeat("z", types.MaxPfpLength+1),
			},
			expErr:    true,
			expErrMsg: "at most",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.SubstationUpdatePfp(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				updatedSubstation, found := k.GetSubstation(ctx, substation.Id)
				require.True(t, found)
				require.Equal(t, tc.input.Pfp, updatedSubstation.Pfp)
			}
		})
	}
}
