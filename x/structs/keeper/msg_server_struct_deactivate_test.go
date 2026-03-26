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
		PassiveDraw: 50,
	}
	k.SetStructType(ctx, structType)

	t.Run("valid struct deactivation", func(t *testing.T) {
		structObj := types.Struct{
			Creator: player.Creator,
			Owner:   player.Id,
			Type:    structType.Id,
		}
		structObj = testAppendStruct(k, ctx, structObj)

		statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structObj.Id)
		testSetStructAttributeFlagAdd(k, ctx, statusAttrId, uint64(types.StructStateBuilt))
		testSetStructAttributeFlagAdd(k, ctx, statusAttrId, uint64(types.StructStateOnline))

		resp, err := ms.StructDeactivate(wctx, &types.MsgStructDeactivate{
			Creator:  player.Creator,
			StructId: structObj.Id,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("struct not found", func(t *testing.T) {
		_, err := ms.StructDeactivate(wctx, &types.MsgStructDeactivate{
			Creator:  player.Creator,
			StructId: "invalid-struct",
		})
		require.Error(t, err)
	})

	t.Run("struct not built", func(t *testing.T) {
		newStruct := types.Struct{
			Creator: player.Creator,
			Owner:   player.Id,
			Type:    structType.Id,
		}
		newStruct = testAppendStruct(k, ctx, newStruct)

		_, err := ms.StructDeactivate(wctx, &types.MsgStructDeactivate{
			Creator:  player.Creator,
			StructId: newStruct.Id,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "building")
	})

	t.Run("struct already offline", func(t *testing.T) {
		offlineStruct := types.Struct{
			Creator: player.Creator,
			Owner:   player.Id,
			Type:    structType.Id,
		}
		offlineStruct = testAppendStruct(k, ctx, offlineStruct)
		offlineStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, offlineStruct.Id)
		testSetStructAttributeFlagAdd(k, ctx, offlineStatusAttrId, uint64(types.StructStateBuilt))

		_, err := ms.StructDeactivate(wctx, &types.MsgStructDeactivate{
			Creator:  player.Creator,
			StructId: offlineStruct.Id,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "offline")
	})
}
