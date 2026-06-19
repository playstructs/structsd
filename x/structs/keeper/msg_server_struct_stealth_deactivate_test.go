package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgStructStealthDeactivate(t *testing.T) {
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
		Id:                    1,
		Type:                  types.CommandStruct,
		Category:              types.ObjectType_player,
		UnitDefenses:          types.TechUnitDefenses_stealthMode,
		StealthActivateCharge: 10,
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

	t.Run("valid stealth deactivation", func(t *testing.T) {
		testSetStructAttributeFlagAdd(k, sdkCtx, statusAttrId, uint64(types.StructStateHidden))
		k.SetGridAttribute(sdkCtx, lastActionAttrId, uint64(0))
		resp, err := ms.StructStealthDeactivate(wctx, &types.MsgStructStealthDeactivate{
			Creator:  player.Creator,
			StructId: structObj.Id,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("struct not found", func(t *testing.T) {
		_, err := ms.StructStealthDeactivate(wctx, &types.MsgStructStealthDeactivate{
			Creator:  player.Creator,
			StructId: "invalid-struct",
		})
		require.Error(t, err)
	})

	t.Run("not in stealth", func(t *testing.T) {
		testSetStructAttributeFlagRemove(k, sdkCtx, statusAttrId, uint64(types.StructStateHidden))
		_, err := ms.StructStealthDeactivate(wctx, &types.MsgStructStealthDeactivate{
			Creator:  player.Creator,
			StructId: structObj.Id,
		})
		require.Error(t, err)
	})

	t.Run("no stealth system", func(t *testing.T) {
		testSetStructAttributeFlagAdd(k, sdkCtx, statusAttrId, uint64(types.StructStateHidden))
		noStealthType := types.StructType{
			Id:       2,
			Type:     types.CommandStruct,
			Category: types.ObjectType_player,
		}
		k.SetStructType(sdkCtx, noStealthType)
		structObj.Type = noStealthType.Id
		k.SetStruct(sdkCtx, structObj)

		_, err := ms.StructStealthDeactivate(wctx, &types.MsgStructStealthDeactivate{
			Creator:  player.Creator,
			StructId: structObj.Id,
		})
		require.Error(t, err)

		structObj.Type = structType.Id
		k.SetStruct(sdkCtx, structObj)
	})

	t.Run("no play permissions", func(t *testing.T) {
		_, err := ms.StructStealthDeactivate(wctx, &types.MsgStructStealthDeactivate{
			Creator:  "cosmos1noperms",
			StructId: structObj.Id,
		})
		require.Error(t, err)
	})
}

// TestMsgStructStealthDeactivateCommandShipNotRequired verifies the v0.19.0
// change that deactivating stealth no longer requires the fleet's Command Ship
// to be present or online. The hidden stealth struct (fleet-category) sits on
// a fleet whose Command Ship is offline; deactivation must still succeed
// because the struct itself is online.
func TestMsgStructStealthDeactivateCommandShipNotRequired(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1000)
	wctx := sdk.WrapSDKContext(sdkCtx)

	player := types.Player{Creator: "cosmos1stdnc", PrimaryAddress: "cosmos1stdnc"}
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

	cmdStructType := types.StructType{Id: 730, Type: types.CommandStruct, Category: types.ObjectType_fleet}
	k.SetStructType(sdkCtx, cmdStructType)

	stealthStructType := types.StructType{
		Id:                    731,
		Type:                  "StealthFighter",
		Category:              types.ObjectType_fleet,
		UnitDefenses:          types.TechUnitDefenses_stealthMode,
		StealthActivateCharge: 10,
		PossibleAmbit:         1 << uint64(types.Ambit_space),
	}
	k.SetStructType(sdkCtx, stealthStructType)

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

	stealthStruct := testAppendStruct(k, sdkCtx, types.Struct{
		Creator:        player.Creator,
		Owner:          player.Id,
		Type:           stealthStructType.Id,
		LocationId:     fleet.Id,
		LocationType:   types.ObjectType_fleet,
		OperatingAmbit: types.Ambit_space,
	})
	stealthStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, stealthStruct.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, stealthStatusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, stealthStatusAttrId, uint64(types.StructStateOnline))
	testSetStructAttributeFlagAdd(k, sdkCtx, stealthStatusAttrId, uint64(types.StructStateHidden))

	resp, err := ms.StructStealthDeactivate(wctx, &types.MsgStructStealthDeactivate{
		Creator:  player.Creator,
		StructId: stealthStruct.Id,
	})
	require.NoError(t, err, "stealth deactivate should succeed with the command ship offline")
	require.NotNil(t, resp)
}
