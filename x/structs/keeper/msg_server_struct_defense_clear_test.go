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

// TestMsgStructDefenseClearCommandShipNotRequired verifies the v0.19.0 change
// that clearing a defensive assignment no longer requires the fleet's Command
// Ship to be present or online. The defender (a fleet-category struct) sits on
// a fleet whose Command Ship is offline; the clear must still succeed because
// the defender itself is online.
func TestMsgStructDefenseClearCommandShipNotRequired(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1000)
	wctx := sdk.WrapSDKContext(sdkCtx)

	player := types.Player{Creator: "cosmos1dcnc", PrimaryAddress: "cosmos1dcnc"}
	player = testAppendPlayer(k, sdkCtx, player)
	k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id), uint64(100000))
	k.SetGridAttribute(sdkCtx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, player.Id), uint64(0))

	planet := testAppendPlanet(k, sdkCtx, types.Planet{
		Creator:   player.Creator,
		Owner:     player.Id,
		LandSlots: 4,
		Land:      []string{"", "", "", ""},
	})
	player.PlanetId = planet.Id
	k.SetPlayer(sdkCtx, player)

	cmdStructType := types.StructType{Id: 710, Type: types.CommandStruct, Category: types.ObjectType_fleet}
	k.SetStructType(sdkCtx, cmdStructType)

	fleetStructType := types.StructType{
		Id:                 711,
		Type:               "FleetDefender",
		Category:           types.ObjectType_fleet,
		DefendChangeCharge: 10,
		PossibleAmbit:      1 << uint64(types.Ambit_space),
	}
	k.SetStructType(sdkCtx, fleetStructType)

	fleet := testAppendFleet(k, sdkCtx, types.Fleet{
		Owner:      player.Id,
		LocationId: planet.Id,
		Status:     types.FleetStatus_away,
	})

	cmdStruct := testAppendStruct(k, sdkCtx, types.Struct{
		Creator:        player.Creator,
		Owner:          player.Id,
		Type:           cmdStructType.Id,
		LocationId:     fleet.Id,
		LocationType:   types.ObjectType_fleet,
		OperatingAmbit: types.Ambit_space,
	})
	cmdStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, cmdStruct.Id)
	// Command ship is built but deliberately NOT online.
	testSetStructAttributeFlagAdd(k, sdkCtx, cmdStatusAttrId, uint64(types.StructStateBuilt))

	fleet.CommandStruct = cmdStruct.Id
	k.SetFleet(sdkCtx, fleet)
	player.FleetId = fleet.Id
	k.SetPlayer(sdkCtx, player)

	defenderStruct := testAppendStruct(k, sdkCtx, types.Struct{
		Creator:        player.Creator,
		Owner:          player.Id,
		Type:           fleetStructType.Id,
		LocationId:     fleet.Id,
		LocationType:   types.ObjectType_fleet,
		OperatingAmbit: types.Ambit_space,
	})
	defStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, defenderStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, defStatusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, defStatusAttrId, uint64(types.StructStateOnline))

	protectedStruct := testAppendStruct(k, sdkCtx, types.Struct{
		Creator:        player.Creator,
		Owner:          player.Id,
		Type:           fleetStructType.Id,
		LocationId:     fleet.Id,
		LocationType:   types.ObjectType_fleet,
		OperatingAmbit: types.Ambit_space,
	})
	protStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, protectedStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, protStatusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, protStatusAttrId, uint64(types.StructStateOnline))

	k.SetStructDefender(sdkCtx, protectedStruct.Id, protectedStruct.Index, defenderStruct.Id)

	resp, err := ms.StructDefenseClear(wctx, &types.MsgStructDefenseClear{
		Creator:          player.Creator,
		DefenderStructId: defenderStruct.Id,
	})
	require.NoError(t, err, "defense clear should succeed with the command ship offline")
	require.NotNil(t, resp)
}
