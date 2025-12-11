package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgStructBuildCancel(t *testing.T) {
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

	// Create a struct type
	structType := types.StructType{
		Id:          1,
		Type:        types.CommandStruct,
		Category:    types.ObjectType_player,
		BuildCharge: 10,
		BuildDraw:   100,
	}
	k.SetStructType(ctx, structType)

	// Create a struct
	structObj := types.Struct{
		Creator: player.Creator,
		Owner:   player.Id,
		Type:    structType.Id,
	}
	structObj = k.AppendStruct(ctx, structObj)

	// Set block start build
	blockStartAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartBuild, structObj.Id)
	k.SetStructAttribute(ctx, blockStartAttrId, uint64(1))

	// Set last action to ensure player has charge
	lastActionAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, player.Id)
	k.SetGridAttribute(ctx, lastActionAttrId, uint64(0))

	testCases := []struct {
		name      string
		input     *types.MsgStructBuildCancel
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid build cancel",
			input: &types.MsgStructBuildCancel{
				Creator:  player.Creator,
				StructId: structObj.Id,
			},
			expErr: false,
		},
		{
			name: "struct not found",
			input: &types.MsgStructBuildCancel{
				Creator:  player.Creator,
				StructId: "invalid-struct",
			},
			expErr:    true,
			expErrMsg: "does not exist",
		},
		{
			name: "struct already built",
			input: &types.MsgStructBuildCancel{
				Creator:  player.Creator,
				StructId: structObj.Id,
			},
			expErr:    true,
			expErrMsg: "already built",
		},
		{
			name: "no play permissions",
			input: &types.MsgStructBuildCancel{
				Creator:  "cosmos1noperms",
				StructId: structObj.Id,
			},
			expErr:    true,
			expErrMsg: "has no",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up struct state for each test
			if tc.name == "valid build cancel" {
				// Ensure struct is not built
				statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structObj.Id)
				builtFlag := uint64(types.StructStateBuilt)
				k.SetStructAttributeFlagRemove(ctx, statusAttrId, builtFlag)
			} else if tc.name == "struct already built" {
				statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structObj.Id)
				builtFlag := uint64(types.StructStateBuilt)
				k.SetStructAttributeFlagAdd(ctx, statusAttrId, builtFlag)
			}

			resp, err := ms.StructBuildCancel(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify struct was destroyed
				_, found := k.GetStruct(ctx, tc.input.StructId)
				require.False(t, found)
			}
		})
	}
}
