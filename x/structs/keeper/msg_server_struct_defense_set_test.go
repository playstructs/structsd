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
		Category:           types.ObjectType_fleet,
		DefendChangeCharge: 10,
		PossibleAmbit:      1 << uint64(types.Ambit_land),
		CanDefend:          true,
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

	t.Run("struct type cannot defend", func(t *testing.T) {
		nonDefenderType := types.StructType{
			Id:                 2,
			Type:               "NonDefender",
			Category:           types.ObjectType_planet,
			DefendChangeCharge: 10,
			PossibleAmbit:      1 << uint64(types.Ambit_land),
			CanDefend:          false,
		}
		k.SetStructType(sdkCtx, nonDefenderType)

		nonDefenderStruct := testAppendStruct(k, sdkCtx, types.Struct{
			Creator:      player.Creator,
			Owner:        player.Id,
			Type:         nonDefenderType.Id,
			LocationId:   planet.Id,
			LocationType: types.ObjectType_planet,
		})
		nonDefStatusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, nonDefenderStruct.Id)
		testSetStructAttributeFlagAdd(k, sdkCtx, nonDefStatusAttrId, uint64(types.StructStateBuilt))
		testSetStructAttributeFlagAdd(k, sdkCtx, nonDefStatusAttrId, uint64(types.StructStateOnline))

		k.SetGridAttribute(sdkCtx, lastActionAttrId, uint64(0))
		_, err := ms.StructDefenseSet(wctx, &types.MsgStructDefenseSet{
			Creator:           player.Creator,
			DefenderStructId:  nonDefenderStruct.Id,
			ProtectedStructId: protectedStruct.Id,
		})
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrStructCannotDefend)
	})
}

// TestMsgStructDefenseSetCommandShipNotRequired verifies the v0.19.0 change
// that setting a defensive assignment no longer requires the fleet's Command
// Ship to be present or online. The defender (a fleet-category struct) and the
// protected struct share a fleet whose Command Ship is offline; the defense
// set must still succeed because the defender itself is online.
func TestMsgStructDefenseSetCommandShipNotRequired(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1000)
	wctx := sdk.WrapSDKContext(sdkCtx)

	player := types.Player{Creator: "cosmos1dsnc", PrimaryAddress: "cosmos1dsnc"}
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

	cmdStructType := types.StructType{Id: 700, Type: types.CommandStruct, Category: types.ObjectType_fleet}
	k.SetStructType(sdkCtx, cmdStructType)

	fleetStructType := types.StructType{
		Id:                 701,
		Type:               "FleetDefender",
		Category:           types.ObjectType_fleet,
		DefendChangeCharge: 10,
		PossibleAmbit:      1 << uint64(types.Ambit_space),
		CanDefend:          true,
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

	resp, err := ms.StructDefenseSet(wctx, &types.MsgStructDefenseSet{
		Creator:           player.Creator,
		DefenderStructId:  defenderStruct.Id,
		ProtectedStructId: protectedStruct.Id,
	})
	require.NoError(t, err, "defense set should succeed with the command ship offline")
	require.NotNil(t, resp)
}

/* TestMsgStructDefenseSetRejectsSelfDefense is the registration half of the
 * preemptive-counter regression.
 *
 * Nothing refused a struct naming itself: IsProtecting compares locations and a
 * struct is trivially co-located with itself. The damage is in the resolution
 * order. StructAttack runs defender counters, then the volley, then the target's
 * own counter - a target counters only after surviving the shots. A
 * self-registered target is picked up in the first pass instead, so its counter
 * lands *before* the volley, and a counter that destroys the attacker voids the
 * volley entirely, leaving the target untouched. CounterSpent is per-transaction,
 * so it worked on every attack.
 */
func TestMsgStructDefenseSetRejectsSelfDefense(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(1000)
	wctx := sdk.WrapSDKContext(sdkCtx)

	player := testAppendPlayer(k, sdkCtx, types.Player{
		Creator:        "cosmos1creator",
		PrimaryAddress: "cosmos1creator",
	})
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

	k.SetStructType(sdkCtx, types.StructType{
		Id:                 1,
		Type:               "Defender",
		Category:           types.ObjectType_fleet,
		DefendChangeCharge: 10,
		PossibleAmbit:      1 << uint64(types.Ambit_land),
		CanDefend:          true,
	})

	structure := testAppendStruct(k, sdkCtx, types.Struct{
		Creator:      player.Creator,
		Owner:        player.Id,
		Type:         1,
		LocationId:   planet.Id,
		LocationType: types.ObjectType_planet,
	})
	statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, structure.Id)
	testSetStructAttributeFlagAdd(k, sdkCtx, statusAttrId, uint64(types.StructStateBuilt))
	testSetStructAttributeFlagAdd(k, sdkCtx, statusAttrId, uint64(types.StructStateOnline))

	_, err := ms.StructDefenseSet(wctx, &types.MsgStructDefenseSet{
		Creator:           player.Creator,
		DefenderStructId:  structure.Id,
		ProtectedStructId: structure.Id,
	})
	require.Error(t, err, "a struct must not be assignable as its own defender")
	require.ErrorContains(t, err, "its own defender")

	require.Empty(t, k.GetAllStructDefender(sdkCtx, structure.Id),
		"a refused registration must not have been written")
}
