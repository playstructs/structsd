package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgStructMove(t *testing.T) {
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

	// Set last action to ensure player has charge
	lastActionAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, player.Id)
	k.SetGridAttribute(ctx, lastActionAttrId, uint64(0))

	// Create a struct type
	structType := types.StructType{
		Id:         1,
		Type:       types.CommandStruct,
		Category:   types.ObjectType_player,
		MoveCharge: 10,
	}
	k.SetStructType(ctx, structType)

	// Create a struct
	structObj := types.Struct{
		Creator: player.Creator,
		Owner:   player.Id,
		Type:    structType.Id,
	}
	structObj = k.AppendStruct(ctx, structObj)

	// Note: LocationId is determined from the struct's current location

	testCases := []struct {
		name      string
		input     *types.MsgStructMove
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid struct move",
			input: &types.MsgStructMove{
				Creator:      player.Creator,
				StructId:     structObj.Id,
				LocationType: types.ObjectType_planet,
				Ambit:        types.Ambit_space,
				Slot:         1,
			},
			expErr: false,
		},
		{
			name: "struct not found",
			input: &types.MsgStructMove{
				Creator:      player.Creator,
				StructId:     "invalid-struct",
				LocationType: types.ObjectType_planet,
				Ambit:        types.Ambit_space,
				Slot:         1,
			},
			expErr:    true,
			expErrMsg: "does not exist",
		},
		{
			name: "no play permissions",
			input: &types.MsgStructMove{
				Creator:      "cosmos1noperms",
				StructId:     structObj.Id,
				LocationType: types.ObjectType_planet,
				Ambit:        types.Ambit_space,
				Slot:         1,
			},
			expErr:    true,
			expErrMsg: "has no",
		},
		{
			name: "insufficient charge",
			input: &types.MsgStructMove{
				Creator:      player.Creator,
				StructId:     structObj.Id,
				LocationType: types.ObjectType_planet,
				Ambit:        types.Ambit_space,
				Slot:         1,
			},
			expErr:    true,
			expErrMsg: "required a charge",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up charge if needed
			if tc.name == "insufficient charge" {
				ctxSDK := sdk.UnwrapSDKContext(ctx)
				k.SetGridAttribute(ctx, lastActionAttrId, uint64(ctxSDK.BlockHeight()))
			} else {
				k.SetGridAttribute(ctx, lastActionAttrId, uint64(0))
			}

			resp, err := ms.StructMove(wctx, tc.input)

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
