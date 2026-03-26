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

	player := types.Player{
		Creator:        "cosmos1creator",
		PrimaryAddress: "cosmos1creator",
	}
	player = testAppendPlayer(k, ctx, player)

	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id)
	k.SetGridAttribute(ctx, capacityAttrId, uint64(100000))

	structType := types.StructType{
		Id:          1,
		Type:        types.CommandStruct,
		Category:    types.ObjectType_player,
		BuildCharge: 10,
		BuildDraw:   100,
	}
	k.SetStructType(ctx, structType)

	lastActionAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, player.Id)
	k.SetGridAttribute(ctx, lastActionAttrId, uint64(0))

	t.Run("valid build cancel", func(t *testing.T) {
		structObj := types.Struct{
			Creator: player.Creator,
			Owner:   player.Id,
			Type:    structType.Id,
		}
		structObj = testAppendStruct(k, ctx, structObj)
		blockStartAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartBuild, structObj.Id)
		k.SetStructAttribute(ctx, blockStartAttrId, uint64(1))

		resp, err := ms.StructBuildCancel(wctx, &types.MsgStructBuildCancel{
			Creator:  player.Creator,
			StructId: structObj.Id,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("struct not found", func(t *testing.T) {
		_, err := ms.StructBuildCancel(wctx, &types.MsgStructBuildCancel{
			Creator:  player.Creator,
			StructId: "invalid-struct",
		})
		require.Error(t, err)
	})

	t.Run("struct already built", func(t *testing.T) {
		builtStruct := types.Struct{
			Creator: player.Creator,
			Owner:   player.Id,
			Type:    structType.Id,
		}
		builtStruct = testAppendStruct(k, ctx, builtStruct)
		statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, builtStruct.Id)
		testSetStructAttributeFlagAdd(k, ctx, statusAttrId, uint64(types.StructStateBuilt))

		_, err := ms.StructBuildCancel(wctx, &types.MsgStructBuildCancel{
			Creator:  player.Creator,
			StructId: builtStruct.Id,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "built")
	})

	t.Run("no play permissions", func(t *testing.T) {
		_, err := ms.StructBuildCancel(wctx, &types.MsgStructBuildCancel{
			Creator:  "cosmos1noperms",
			StructId: "5-1",
		})
		require.Error(t, err)
	})
}
