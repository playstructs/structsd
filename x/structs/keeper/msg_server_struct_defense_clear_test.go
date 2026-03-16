package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgStructDefenseClear(t *testing.T) {
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
		Id:                 1,
		Type:               types.CommandStruct,
		Category:           types.ObjectType_player,
		DefendChangeCharge: 10,
	}
	k.SetStructType(sdkCtx, structType)

	defenderStruct := types.Struct{
		Creator:      player.Creator,
		Owner:        player.Id,
		Type:         structType.Id,
		LocationId:   "planet1",
		LocationType: types.ObjectType_planet,
	}
	defenderStruct = testAppendStruct(k, sdkCtx, defenderStruct)

	protectedStruct := types.Struct{
		Creator:      player.Creator,
		Owner:        player.Id,
		Type:         structType.Id,
		LocationId:   "planet1",
		LocationType: types.ObjectType_planet,
	}
	protectedStruct = testAppendStruct(k, sdkCtx, protectedStruct)

	defenderStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, defenderStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, defenderStatusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, defenderStatusAttrId, uint64(types.StructStateOnline))

	t.Run("valid defense clear", func(t *testing.T) {
		k.SetStructDefender(sdkCtx, protectedStruct.Id, protectedStruct.Index, defenderStruct.Id)
		resp, err := ms.StructDefenseClear(wctx, &types.MsgStructDefenseClear{
			Creator:          player.Creator,
			DefenderStructId: defenderStruct.Id,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("defender struct not found", func(t *testing.T) {
		_, err := ms.StructDefenseClear(wctx, &types.MsgStructDefenseClear{
			Creator:          player.Creator,
			DefenderStructId: "invalid-struct",
		})
		require.Error(t, err)
	})

	t.Run("no play permissions", func(t *testing.T) {
		_, err := ms.StructDefenseClear(wctx, &types.MsgStructDefenseClear{
			Creator:          "cosmos1noperms",
			DefenderStructId: defenderStruct.Id,
		})
		require.Error(t, err)
	})
}
