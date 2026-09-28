package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestStructMoveValidatesSlotsBeforeIndexingAndMovesExactly(t *testing.T) {
	for _, placement := range []types.ObjectType{types.ObjectType_planet, types.ObjectType_fleet} {
		for _, ambit := range []types.Ambit{types.Ambit_land, types.Ambit_water, types.Ambit_air, types.Ambit_space} {
			for _, mode := range []string{"move", "occupied", "limit", "max_uint", "short_inventory", "closed_slot"} {
				t.Run(placement.String()+"/"+ambit.String()+"/"+mode, func(t *testing.T) {
					k, server, wrapped := setupMsgServer(t)
					ctx := sdk.UnwrapSDKContext(wrapped).WithBlockHeight(1000)
					player := testAppendPlayer(k, ctx, types.Player{Creator: "cosmos1creator", PrimaryAddress: "cosmos1creator"})
					kind := types.StructType{Id: 99001, Type: "Movable fixture", Category: placement, Movable: true,
						PossibleAmbit: types.Ambit_flag[types.Ambit_land] | types.Ambit_flag[types.Ambit_water] | types.Ambit_flag[types.Ambit_air] | types.Ambit_flag[types.Ambit_space]}
					k.SetStructType(ctx, kind)
					count, length := uint64(2), 2
					if mode == "short_inventory" {
						length = 1
					}
					if mode == "closed_slot" {
						count = 1
					}
					land, water, air, space := make([]string, length), make([]string, length), make([]string, length), make([]string, length)
					planet := testAppendPlanet(k, ctx, types.Planet{Owner: player.Id, Creator: player.Creator,
						LandSlots: count, WaterSlots: count, AirSlots: count, SpaceSlots: count, Land: land, Water: water, Air: air, Space: space})
					fleet := types.Fleet{Id: keeperlib.GetObjectID(types.ObjectType_fleet, player.Index), Owner: player.Id,
						LandSlots: count, WaterSlots: count, AirSlots: count, SpaceSlots: count, Land: land, Water: water, Air: air, Space: space}
					command := testAppendStruct(k, ctx, types.Struct{Owner: player.Id, Creator: player.Creator, Type: 1,
						LocationId: fleet.Id, LocationType: types.ObjectType_fleet, OperatingAmbit: types.Ambit_space})
					fleet.CommandStruct = command.Id
					k.SetStructAttribute(ctx, keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, command.Id), uint64(types.StructStateBuilt|types.StructStateOnline))
					k.SetFleet(ctx, fleet)
					player.PlanetId, player.FleetId = planet.Id, fleet.Id
					k.SetPlayer(ctx, player)
					k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id), 100000)
					location := planet.Id
					if placement == types.ObjectType_fleet {
						location = fleet.Id
					}
					unit := testAppendStruct(k, ctx, types.Struct{Owner: player.Id, Creator: player.Creator, Type: kind.Id,
						LocationId: location, LocationType: placement, OperatingAmbit: types.Ambit_land, Slot: 0})
					land[0] = unit.Id
					if mode == "move" && ambit != types.Ambit_land {
						switch ambit {
						case types.Ambit_water:
							water[0] = "5-99998"
						case types.Ambit_air:
							air[0] = "5-99998"
						case types.Ambit_space:
							space[0] = "5-99998"
						}
					}
					if mode == "occupied" {
						switch ambit {
						case types.Ambit_land:
							land[1] = "5-99999"
						case types.Ambit_water:
							water[1] = "5-99999"
						case types.Ambit_air:
							air[1] = "5-99999"
						case types.Ambit_space:
							space[1] = "5-99999"
						}
					}
					k.SetPlanet(ctx, planet)
					k.SetFleet(ctx, fleet)
					k.SetStructAttribute(ctx, keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, unit.Id), uint64(types.StructStateBuilt|types.StructStateOnline|types.StructStateMaterialized))
					slot := uint64(1)
					if mode == "limit" {
						slot = 2
					}
					if mode == "max_uint" {
						slot = ^uint64(0)
					}
					t.Run("build", func(t *testing.T) {
						buildCtx, _ := ctx.CacheContext()
						beforeGrid := k.GetAllGridExport(buildCtx)
						beforeUnits := k.GetAllStruct(buildCtx)
						var response *types.MsgStructStatusResponse
						var buildErr error
						require.NotPanics(t, func() {
							response, buildErr = server.StructBuildInitiate(buildCtx, &types.MsgStructBuildInitiate{
								Creator: player.Creator, PlayerId: player.Id, StructTypeId: kind.Id, OperatingAmbit: ambit, Slot: slot,
							})
						})
						if mode != "move" {
							var typed *types.StructBuildError
							require.ErrorAs(t, buildErr, &typed)
							expected := "slot_unavailable"
							if mode == "occupied" {
								expected = "slot_occupied"
							}
							require.Equal(t, expected, typed.Reason)
							require.Equal(t, beforeGrid, k.GetAllGridExport(buildCtx))
							require.Equal(t, beforeUnits, k.GetAllStruct(buildCtx))
							return
						}
						require.NoError(t, buildErr)
						require.Equal(t, slot, response.Struct.Slot)
						require.Equal(t, ambit, response.Struct.OperatingAmbit)
						require.Equal(t, location, response.Struct.LocationId)
					})
					beforeGrid := k.GetAllGridExport(ctx)
					var err error
					require.NotPanics(t, func() {
						_, err = server.StructMove(ctx, &types.MsgStructMove{Creator: player.Creator, StructId: unit.Id, LocationType: placement, Ambit: ambit, Slot: slot})
					})
					current, found := k.GetStruct(ctx, unit.Id)
					require.True(t, found)
					currentPlanet, found := k.GetPlanet(ctx, planet.Id)
					require.True(t, found)
					currentFleet, found := k.GetFleet(ctx, fleet.Id)
					require.True(t, found)
					if mode != "move" {
						require.Error(t, err)
						reason := "slot_unavailable"
						if mode == "occupied" {
							reason = "slot_occupied"
						}
						var buildError *types.StructBuildError
						require.ErrorAs(t, err, &buildError)
						require.Equal(t, reason, buildError.Reason)
						require.Equal(t, unit, current)
						require.Equal(t, planet, currentPlanet)
						require.Equal(t, fleet, currentFleet)
						require.Equal(t, beforeGrid, k.GetAllGridExport(ctx))
						return
					}
					require.NoError(t, err)
					require.Equal(t, slot, current.Slot)
					require.Equal(t, ambit, current.OperatingAmbit)
					require.Equal(t, location, current.LocationId)
					rows := map[types.Ambit][]string{types.Ambit_land: currentPlanet.Land, types.Ambit_water: currentPlanet.Water, types.Ambit_air: currentPlanet.Air, types.Ambit_space: currentPlanet.Space}
					if placement == types.ObjectType_fleet {
						rows = map[types.Ambit][]string{types.Ambit_land: currentFleet.Land, types.Ambit_water: currentFleet.Water, types.Ambit_air: currentFleet.Air, types.Ambit_space: currentFleet.Space}
					}
					require.Empty(t, rows[types.Ambit_land][0])
					require.Equal(t, unit.Id, rows[ambit][slot])
					if ambit != types.Ambit_land {
						require.Equal(t, "5-99998", rows[ambit][0], "moving must not overwrite the destination ambit's old-numbered slot")
					}
				})
			}
		}
	}
}
