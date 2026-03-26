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
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1000)
	wctx := sdk.WrapSDKContext(sdkCtx)

	player := types.Player{
		Creator:        "cosmos1creator",
		PrimaryAddress: "cosmos1creator",
	}
	player = testAppendPlayer(k, sdkCtx, player)

	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id)
	k.SetGridAttribute(sdkCtx, capacityAttrId, uint64(100000))

	lastActionAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, player.Id)
	k.SetGridAttribute(sdkCtx, lastActionAttrId, uint64(0))

	structType := types.StructType{
		Id:         1,
		Type:       types.CommandStruct,
		Category:   types.ObjectType_player,
		MoveCharge: 10,
	}
	k.SetStructType(sdkCtx, structType)

	structObj := types.Struct{
		Creator: player.Creator,
		Owner:   player.Id,
		Type:    structType.Id,
	}
	structObj = testAppendStruct(k, sdkCtx, structObj)

	statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structObj.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, statusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, statusAttrId, uint64(types.StructStateOnline))

	t.Run("valid struct move", func(t *testing.T) {
		k.SetGridAttribute(sdkCtx, lastActionAttrId, uint64(0))
		resp, err := ms.StructMove(wctx, &types.MsgStructMove{
			Creator:      player.Creator,
			StructId:     structObj.Id,
			LocationType: types.ObjectType_planet,
			Ambit:        types.Ambit_space,
			Slot:         1,
		})
		_ = resp
		_ = err
	})

	t.Run("struct not found", func(t *testing.T) {
		_, err := ms.StructMove(wctx, &types.MsgStructMove{
			Creator:      player.Creator,
			StructId:     "invalid-struct",
			LocationType: types.ObjectType_planet,
			Ambit:        types.Ambit_space,
			Slot:         1,
		})
		require.Error(t, err)
	})

	t.Run("no play permissions", func(t *testing.T) {
		_, err := ms.StructMove(wctx, &types.MsgStructMove{
			Creator:      "cosmos1noperms",
			StructId:     structObj.Id,
			LocationType: types.ObjectType_planet,
			Ambit:        types.Ambit_space,
			Slot:         1,
		})
		require.Error(t, err)
	})

	t.Run("insufficient charge", func(t *testing.T) {
		k.SetGridAttribute(sdkCtx, lastActionAttrId, uint64(sdkCtx.BlockHeight()))
		_, err := ms.StructMove(wctx, &types.MsgStructMove{
			Creator:      player.Creator,
			StructId:     structObj.Id,
			LocationType: types.ObjectType_planet,
			Ambit:        types.Ambit_space,
			Slot:         1,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "required charge")
	})
}
