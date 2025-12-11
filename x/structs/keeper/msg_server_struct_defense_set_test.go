package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgStructDefenseSet(t *testing.T) {
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
		Id:                 1,
		Type:               types.CommandStruct,
		Category:           types.ObjectType_player,
		DefendChangeCharge: 10,
	}
	k.SetStructType(ctx, structType)

	// Create defender struct
	defenderStruct := types.Struct{
		Creator: player.Creator,
		Owner:   player.Id,
		Type:    structType.Id,
	}
	defenderStruct = k.AppendStruct(ctx, defenderStruct)

	// Create protected struct
	protectedStruct := types.Struct{
		Creator: player.Creator,
		Owner:   player.Id,
		Type:    structType.Id,
	}
	protectedStruct = k.AppendStruct(ctx, protectedStruct)

	// Mark structs as built and online
	statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, defenderStruct.Id)
	builtFlag := uint64(types.StructStateBuilt)
	k.SetStructAttributeFlagAdd(ctx, statusAttrId, builtFlag)

	protectedStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, protectedStruct.Id)
	k.SetStructAttributeFlagAdd(ctx, protectedStatusAttrId, builtFlag)

	testCases := []struct {
		name      string
		input     *types.MsgStructDefenseSet
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid defense set",
			input: &types.MsgStructDefenseSet{
				Creator:           player.Creator,
				DefenderStructId:  defenderStruct.Id,
				ProtectedStructId: protectedStruct.Id,
			},
			expErr: false,
		},
		{
			name: "defender struct not found",
			input: &types.MsgStructDefenseSet{
				Creator:           player.Creator,
				DefenderStructId:  "invalid-struct",
				ProtectedStructId: protectedStruct.Id,
			},
			expErr:    true,
			expErrMsg: "does not exist",
		},
		{
			name: "protected struct not found",
			input: &types.MsgStructDefenseSet{
				Creator:           player.Creator,
				DefenderStructId:  defenderStruct.Id,
				ProtectedStructId: "invalid-struct",
			},
			expErr:    true,
			expErrMsg: "not found",
		},
		{
			name: "no play permissions",
			input: &types.MsgStructDefenseSet{
				Creator:           "cosmos1noperms",
				DefenderStructId:  defenderStruct.Id,
				ProtectedStructId: protectedStruct.Id,
			},
			expErr:    true,
			expErrMsg: "has no",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.StructDefenseSet(wctx, tc.input)

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
