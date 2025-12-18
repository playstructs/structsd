package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgStructDeactivate(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	// Create a player first
	player := types.Player{
		Creator:        "cosmos1creator",
		PrimaryAddress: "cosmos1creator",
	}
	player = k.AppendPlayer(ctx, player)

	// Set up player capacity to be online
	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id)
	k.SetGridAttribute(ctx, capacityAttrId, uint64(100000))

	// Create a struct
	structObj := types.Struct{
		Creator: player.Creator,
		Owner:   player.Id,
		Type:    1,
	}
	structObj = k.AppendStruct(ctx, structObj)

	// Mark struct as built and online
	statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structObj.Id)
	builtFlag := uint64(types.StructStateBuilt)
	k.SetStructAttributeFlagAdd(ctx, statusAttrId, builtFlag)

	testCases := []struct {
		name      string
		input     *types.MsgStructDeactivate
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid struct deactivation",
			input: &types.MsgStructDeactivate{
				Creator:  player.Creator,
				StructId: structObj.Id,
			},
			expErr: false,
		},
		{
			name: "struct not found",
			input: &types.MsgStructDeactivate{
				Creator:  player.Creator,
				StructId: "invalid-struct",
			},
			expErr:    true,
			expErrMsg: "does not exist",
		},
		{
			name: "struct not built",
			input: &types.MsgStructDeactivate{
				Creator:  player.Creator,
				StructId: structObj.Id,
			},
			expErr:    true,
			expErrMsg: "isn't built yet",
		},
		{
			name: "struct already offline",
			input: &types.MsgStructDeactivate{
				Creator:  player.Creator,
				StructId: structObj.Id,
			},
			expErr:    true,
			expErrMsg: "already offline",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up struct state for each test
			if tc.name == "valid struct deactivation" {
				// Ensure struct is built and online
				k.SetStructAttributeFlagAdd(ctx, statusAttrId, builtFlag)
			} else if tc.name == "struct not built" {
				// Clear built flag
				k.SetStructAttributeFlagRemove(ctx, statusAttrId, builtFlag)
			} else if tc.name == "struct already offline" {
				// Struct is offline by default, just ensure it's built
				// The deactivate will check if it's already offline
			}

			resp, err := ms.StructDeactivate(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
			}
		})
	}
}
