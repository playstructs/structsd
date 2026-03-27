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
		Id:            1,
		Type:          "Test Struct",
		Category:      types.ObjectType_planet,
		MoveCharge:    10,
		PossibleAmbit: 1<<uint64(types.Ambit_land) | 1<<uint64(types.Ambit_space),
	}
	k.SetStructType(sdkCtx, structType)

	planet := testAppendPlanet(k, sdkCtx, types.Planet{
		Creator:   player.Creator,
		Owner:     player.Id,
		LandSlots: 2,
		Land:      []string{"", ""},
	})
	player.PlanetId = planet.Id
	k.SetPlayer(sdkCtx, player)

	structObj := types.Struct{
		Creator:      player.Creator,
		Owner:        player.Id,
		Type:         structType.Id,
		LocationId:   planet.Id,
		LocationType: types.ObjectType_planet,
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
			Ambit:        types.Ambit_land,
			Slot:         0,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
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
