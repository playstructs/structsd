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

	planet := testAppendPlanet(k, sdkCtx, types.Planet{
		Creator:   player.Creator,
		Owner:     player.Id,
		LandSlots: 4,
		Land:      []string{"", "", "", ""},
	})
	player.PlanetId = planet.Id
	k.SetPlayer(sdkCtx, player)

	structType := types.StructType{
		Id:                 1,
		Type:               "Defender",
		Category:           types.ObjectType_planet,
		DefendChangeCharge: 10,
		PossibleAmbit:      1 << uint64(types.Ambit_land),
	}
	k.SetStructType(sdkCtx, structType)

	defenderStruct := types.Struct{
		Creator:      player.Creator,
		Owner:        player.Id,
		Type:         structType.Id,
		LocationId:   planet.Id,
		LocationType: types.ObjectType_planet,
	}
	defenderStruct = testAppendStruct(k, sdkCtx, defenderStruct)
	defStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, defenderStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, defStatusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, defStatusAttrId, uint64(types.StructStateOnline))

	protectedStruct := types.Struct{
		Creator:      player.Creator,
		Owner:        player.Id,
		Type:         structType.Id,
		LocationId:   planet.Id,
		LocationType: types.ObjectType_planet,
	}
	protectedStruct = testAppendStruct(k, sdkCtx, protectedStruct)
	protStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, protectedStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, protStatusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, protStatusAttrId, uint64(types.StructStateOnline))

	t.Run("valid defense set", func(t *testing.T) {
		k.SetGridAttribute(sdkCtx, lastActionAttrId, uint64(0))
		resp, err := ms.StructDefenseSet(wctx, &types.MsgStructDefenseSet{
			Creator:           player.Creator,
			DefenderStructId:  defenderStruct.Id,
			ProtectedStructId: protectedStruct.Id,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("defender struct not found", func(t *testing.T) {
		_, err := ms.StructDefenseSet(wctx, &types.MsgStructDefenseSet{
			Creator:           player.Creator,
			DefenderStructId:  "invalid-struct",
			ProtectedStructId: protectedStruct.Id,
		})
		require.Error(t, err)
	})

	t.Run("no play permissions", func(t *testing.T) {
		_, err := ms.StructDefenseSet(wctx, &types.MsgStructDefenseSet{
			Creator:           "cosmos1noperms",
			DefenderStructId:  defenderStruct.Id,
			ProtectedStructId: protectedStruct.Id,
		})
		require.Error(t, err)
	})
}
