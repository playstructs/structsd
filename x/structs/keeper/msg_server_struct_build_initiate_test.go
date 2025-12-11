package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgStructBuildInitiate(t *testing.T) {
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
		Id:          1,
		Type:        types.CommandStruct,
		Category:    types.ObjectType_player,
		BuildCharge: 10,
		BuildDraw:   100,
	}
	k.SetStructType(ctx, structType)

	// Note: PlanetId is not in the message, it's determined from the player's current planet

	testCases := []struct {
		name      string
		input     *types.MsgStructBuildInitiate
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid struct build initiation",
			input: &types.MsgStructBuildInitiate{
				Creator:        player.Creator,
				PlayerId:       player.Id,
				StructTypeId:   structType.Id,
				OperatingAmbit: types.Ambit_space,
				Slot:           1,
			},
			expErr: false,
		},
		{
			name: "invalid player id",
			input: &types.MsgStructBuildInitiate{
				Creator:        player.Creator,
				PlayerId:       "invalid-player",
				StructTypeId:   structType.Id,
				OperatingAmbit: types.Ambit_space,
				Slot:           1,
			},
			expErr:    true,
			expErrMsg: "requires Player account",
		},
		{
			name: "struct type not found",
			input: &types.MsgStructBuildInitiate{
				Creator:        player.Creator,
				PlayerId:       player.Id,
				StructTypeId:   999,
				OperatingAmbit: types.Ambit_space,
				Slot:           1,
			},
			expErr:    true,
			expErrMsg: "was not found",
		},
		{
			name: "no play permissions",
			input: &types.MsgStructBuildInitiate{
				Creator:        "cosmos1noperms",
				PlayerId:       player.Id,
				StructTypeId:   structType.Id,
				OperatingAmbit: types.Ambit_space,
				Slot:           1,
			},
			expErr:    true,
			expErrMsg: "has no",
		},
		{
			name: "player is halted",
			input: &types.MsgStructBuildInitiate{
				Creator:        player.Creator,
				PlayerId:       player.Id,
				StructTypeId:   structType.Id,
				OperatingAmbit: types.Ambit_space,
				Slot:           1,
			},
			expErr:    true,
			expErrMsg: "is Halted",
		},
		{
			name: "insufficient charge",
			input: &types.MsgStructBuildInitiate{
				Creator:        player.Creator,
				PlayerId:       player.Id,
				StructTypeId:   structType.Id,
				OperatingAmbit: types.Ambit_space,
				Slot:           1,
			},
			expErr:    true,
			expErrMsg: "required a charge",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up player state for each test
			if tc.name == "player is halted" {
				k.PlayerHalt(ctx, player.Id)
			} else {
				k.PlayerResume(ctx, player.Id)
			}

			if tc.name == "insufficient charge" {
				// Set last action to current block to have no charge
				ctxSDK := sdk.UnwrapSDKContext(ctx)
				k.SetGridAttribute(ctx, lastActionAttrId, uint64(ctxSDK.BlockHeight()))
			} else {
				k.SetGridAttribute(ctx, lastActionAttrId, uint64(0))
			}

			resp, err := ms.StructBuildInitiate(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.NotNil(t, resp.Struct)
			}
		})
	}
}
