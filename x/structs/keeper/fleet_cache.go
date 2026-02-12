package keeper

import (

	"context"
    //"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"

)

type FleetCache struct {
    FleetId string
    Index uint64
    PlayerId string

    CC  *CurrentContext

    Changed bool

    FleetLoaded  bool
    Fleet        types.Fleet

}


func (cache *FleetCache) Commit() () {
    if (cache.Changed) {
        cache.CC.k.logger.Info("Updating Fleet From Cache", "fleetId", cache.FleetId)
        cache.CC.k.SetFleet(cache.CC.ctx, cache.Fleet)
    }
    cache.Changed = false
}

func (cache *FleetCache) IsChanged() bool {
    return cache.Changed
}

func (cache *FleetCache) ID() string {
    return cache.FleetId
}

func (cache *FleetCache) LoadFleet() (found bool) {
    cache.Fleet, cache.FleetLoaded = cache.CC.k.GetFleet(cache.CC.ctx, cache.FleetId)

    if (!cache.FleetLoaded) {
        fleet := types.CreateEmptyFleet()
        // Set the ID of the appended value
        fleet.Id = cache.FleetId
        fleet.Owner = cache.PlayerId

        player, _ := cache.CC.GetPlayer(cache.PlayerId)
        player.SetFleetId(cache.FleetId)

        cache.Fleet = fleet
        cache.Changed = true
        cache.FleetLoaded = true
        structure := cache.CC.InitialCommandShipStruct(cache)
        cache.SetCommandStruct(structure.GetStructId())
    }

    return cache.FleetLoaded
}


// Fleet Details
func (cache *FleetCache) GetFleet()     (types.Fleet)   { if (!cache.FleetLoaded) { cache.LoadFleet() }; return cache.Fleet }
func (cache *FleetCache) GetFleetId()   (string)        { return cache.FleetId }

// Ownership Details
func (cache *FleetCache) GetOwnerId()   (string)        { return cache.PlayerId }

// Command Struct Details
func (cache *FleetCache) GetCommandStructId()   (string)        { return cache.GetFleet().CommandStruct }

// Location Details
func (cache *FleetCache) GetLocationId()        (string)            { return cache.GetFleet().LocationId }
func (cache *FleetCache) GetLocationType()      (types.ObjectType)  { return cache.GetFleet().LocationType }

// Planet Battle Queue Position
func (cache *FleetCache) GetLocationListForward()   (string)        { return cache.GetFleet().LocationListForward }
func (cache *FleetCache) GetLocationListBackward()  (string)        { return cache.GetFleet().LocationListBackward }

func (cache *FleetCache) GetOwner()  (*PlayerCache)  {
    player, _ := cache.CC.GetPlayer( cache.PlayerId )
    return player
}

func (cache *FleetCache) GetCommandStruct() (*StructCache)  {
    structure := cache.CC.GetStruct( cache.GetCommandStructId() )
    return structure
}

func (cache *FleetCache) GetPlanet() (planet *PlanetCache) {
    if (cache.GetLocationType() == types.ObjectType_planet) {
        planet = cache.CC.GetPlanet( cache.GetLocationId() )
    }
    return planet
}


func (cache *FleetCache) GetForwardFleet() (forwardFleet *FleetCache) {
    if (cache.GetLocationListForward() != "") {
        forwardFleet, _ = cache.CC.GetFleetById( cache.GetLocationListForward() )
    }
    return forwardFleet
}

func (cache *FleetCache) GetBackwardFleet() (backwardFleet *FleetCache) {
    if (cache.GetLocationListBackward() != "") {
        backwardFleet, _ = cache.CC.GetFleetById( cache.GetLocationListBackward() )
    }
    return backwardFleet
}



//func (cache *FleetCache) GetPreviousPlanet()    (*PlanetCache)      { return cache.PreviousPlanet }
//func (cache *FleetCache) GetPreviousForwardFleet()  (*FleetCache)   { return cache.PreviousForwardFleet }
//func (cache *FleetCache) GetPreviousBackwardFleet() (*FleetCache)   { return cache.PreviousBackwardFleet }




func (cache *FleetCache) SetLocationListForward(fleetId string) () {
    if (!cache.FleetLoaded) { cache.LoadFleet() }

    cache.Fleet.LocationListForward = fleetId
    cache.Changed = true
}

func (cache *FleetCache) SetLocationListBackward(fleetId string) () {
    if (!cache.FleetLoaded) { cache.LoadFleet() }

    cache.Fleet.LocationListBackward = fleetId
    cache.Changed = true
}

func (cache *FleetCache) IsOnStation() (bool) {
    return (cache.GetFleet().Status == types.FleetStatus_onStation)
}

func (cache *FleetCache) IsAway() (bool) {
    return (cache.GetFleet().Status == types.FleetStatus_away)
}

func (cache *FleetCache) HasCommandStruct() (bool) {
    return (cache.GetFleet().CommandStruct != "")
}


func (cache *FleetCache) SetLocationToPlanet(destination *PlanetCache) {
    if (!cache.FleetLoaded) { cache.LoadFleet() }

    // TODO/MVP
    // One day it'll matter that the previous destination might not be a planet
    // Until that day, let's not complicate this further.

    // If we're already there, let's not waste cycles and writes.
    if (cache.GetLocationId() == destination.GetPlanetId()) { return }

    // Let's do some initial copies
    previousPlanetId := cache.GetLocationId()
    previousPlanet := cache.GetPlanet()

    previousForwardFleetId := cache.GetLocationListForward()
    previousForwardFleet := cache.GetForwardFleet()

    previousBackwardFleetId := cache.GetLocationListBackward()
    previousBackwardFleet := cache.GetBackwardFleet()

    // Location updated and next call to GetPlanet() will pull the new location
    cache.Fleet.LocationId = destination.GetPlanetId()
    cache.Fleet.LocationType = types.ObjectType_planet


    // Old destination wasn't home - update all the previous stuff
    if (cache.GetOwner().GetPlanetId() != previousPlanetId) {

        // Are we at the start of the list?
        if (previousForwardFleetId == "") {
            previousPlanet.SetLocationListStart(previousBackwardFleetId)
            if (previousBackwardFleetId != "") {
                previousBackwardFleet.SetLocationListForward("")
            }
        // The back of the list
        } else if (previousBackwardFleetId == "") {
            previousPlanet.SetLocationListLast(previousForwardFleetId)
            previousForwardFleet.SetLocationListBackward("")

        // Or Somewhere In The Between
        } else {
            previousForwardFleet.SetLocationListBackward(previousBackwardFleetId)
            previousBackwardFleet.SetLocationListForward(previousForwardFleetId)
        }

        cache.SetLocationListForward("")
        cache.SetLocationListBackward("")
    }

    // New destination isn't home - add it to the end of the list
    if (cache.GetOwner().GetPlanetId() != destination.GetPlanetId()) {

        // Is it the first fleet to arrive?
        if (cache.GetPlanet().GetLocationListStart() == "") {
            cache.GetPlanet().SetLocationListStart(cache.GetFleetId())
        } else {
            cache.SetLocationListForward(cache.GetPlanet().GetLocationListLast())
            cache.GetForwardFleet().SetLocationListBackward(cache.GetFleetId())
        }

        cache.GetPlanet().SetLocationListLast(cache.GetFleetId())

        cache.Fleet.Status = types.FleetStatus_away
    } else {
        cache.Fleet.Status = types.FleetStatus_onStation
    }

    cache.Changed = true
}

func (cache *FleetCache) PlanetMoveReadinessCheck() (error) {
    if cache.GetOwner().IsOffline() {
        return types.NewPlayerPowerError(cache.GetOwnerId(), "offline")
    }

    if !cache.HasCommandStruct() {
        return types.NewFleetCommandError(cache.GetFleetId(), "no_command_struct")
    }

    if cache.GetCommandStruct().IsOffline() {
        return types.NewFleetCommandError(cache.GetFleetId(), "command_offline")
    }

    return nil
}

func (cache *FleetCache) Defeat() (){
    if (!cache.FleetLoaded) { cache.LoadFleet() }

    uctx := sdk.UnwrapSDKContext(cache.CC.ctx)
    _ = uctx.EventManager().EmitTypedEvent(&types.EventRaid{&types.EventRaidDetail{FleetId: cache.GetFleetId(), PlanetId: cache.GetPlanet().GetPlanetId(), Status: types.RaidStatus_attackerDefeated}})

    // Send Fleet home
    cache.SetLocationToPlanet(cache.GetOwner().GetPlanet())
}


func (cache *FleetCache) PeaceDeal() (){
    if (!cache.FleetLoaded) { cache.LoadFleet() }

    uctx := sdk.UnwrapSDKContext(cache.CC.ctx)
    _ = uctx.EventManager().EmitTypedEvent(&types.EventRaid{&types.EventRaidDetail{FleetId: cache.GetFleetId(), PlanetId: cache.GetPlanet().GetPlanetId(), Status: types.RaidStatus_demilitarized}})

    // Send Fleet home
    cache.SetLocationToPlanet(cache.GetOwner().GetPlanet())
}

func (cache *FleetCache) BuildInitiateReadiness(structure *types.Struct, structType *StructTypeCache, ambit types.Ambit, ambitSlot uint64) (error) {
    if structure.GetOwner() != cache.GetOwnerId() {
         return types.NewStructOwnershipError(structure.Id, cache.GetOwnerId(), structure.GetOwner()).WithLocation("fleet", cache.GetFleetId())
    }

    if cache.IsAway() {
        return types.NewFleetStateError(cache.GetFleetId(), "away", "build")
    }


    if structType.GetStructType().Type != types.CommandStruct {
        if !cache.HasCommandStruct() {
            return types.NewFleetCommandError(cache.GetFleetId(), "no_command_struct")
        }

        if cache.GetCommandStruct().IsOffline() {
            return types.NewFleetCommandError(cache.GetFleetId(), "command_offline")
        }
    }

    if (structType.GetStructType().Category != types.ObjectType_fleet) {
        return types.NewStructLocationError(structType.ID(), ambit.String(), "outside_planet")
    }

    // Check that the Struct can exist in the specified ambit
    if types.Ambit_flag[ambit]&structType.GetStructType().PossibleAmbit == 0 {
        return types.NewStructLocationError(structType.ID(), ambit.String(), "invalid_ambit")
    }

    if structType.GetStructType().Type == types.CommandStruct {
        if cache.HasCommandStruct() {
            return types.NewStructBuildError(structType.ID(), "fleet", cache.GetFleetId(), "command_exists")
        }

    } else {

        var slots uint64
        var slot string
        // Check Ambit / Slot
        switch ambit {
            case types.Ambit_land:
                slots = cache.GetFleet().LandSlots
                slot  = cache.GetFleet().Land[ambitSlot]
            case types.Ambit_water:
                slots = cache.GetFleet().WaterSlots
                slot  = cache.GetFleet().Water[ambitSlot]
            case types.Ambit_air:
                slots = cache.GetFleet().AirSlots
                slot  = cache.GetFleet().Air[ambitSlot]
            case types.Ambit_space:
                slots = cache.GetFleet().SpaceSlots
                slot  = cache.GetFleet().Space[ambitSlot]
            default:
                return types.NewStructBuildError(structType.ID(), "fleet", cache.GetFleetId(), "invalid_ambit").WithAmbit(ambit.String())
        }

        if (ambitSlot >= slots) {
            return types.NewStructBuildError(structType.ID(), "fleet", cache.GetFleetId(), "slot_unavailable").WithSlot(ambitSlot)
        }
        if (slot != "") {
            return types.NewStructBuildError(structType.ID(), "fleet", cache.GetFleetId(), "slot_occupied").WithSlot(ambitSlot).WithExistingStruct(slot)
        }
    }
    return nil
}



func (cache *FleetCache) MoveReadiness(structure *StructCache, ambit types.Ambit, ambitSlot uint64) (error) {
    if structure.GetOwnerId() != cache.GetOwnerId() {
         return types.NewStructOwnershipError(structure.GetStructId(), cache.GetOwnerId(), structure.GetOwnerId()).WithLocation("fleet", cache.GetFleetId())
    }

    if (structure.GetStructType().Category != types.ObjectType_fleet) {
        return types.NewStructLocationError(structure.GetTypeId(), ambit.String(), "outside_planet")
    }

    // Check that the Struct can exist in the specified ambit
    if types.Ambit_flag[ambit]&structure.GetStructType().PossibleAmbit == 0 {
        return types.NewStructLocationError(structure.GetTypeId(), ambit.String(), "invalid_ambit")
    }

    if structure.GetStructType().Type != types.CommandStruct {

        var slots uint64
        var slot string
        // Check Ambit / Slot
        switch ambit {
            case types.Ambit_land:
                slots = cache.GetFleet().LandSlots
                slot  = cache.GetFleet().Land[ambitSlot]
            case types.Ambit_water:
                slots = cache.GetFleet().WaterSlots
                slot  = cache.GetFleet().Water[ambitSlot]
            case types.Ambit_air:
                slots = cache.GetFleet().AirSlots
                slot  = cache.GetFleet().Air[ambitSlot]
            case types.Ambit_space:
                slots = cache.GetFleet().SpaceSlots
                slot  = cache.GetFleet().Space[ambitSlot]
            default:
                return types.NewStructBuildError(structure.GetTypeId(), "fleet", cache.GetFleetId(), "invalid_ambit").WithAmbit(ambit.String())
        }

        if (ambitSlot >= slots) {
            return types.NewStructBuildError(structure.GetTypeId(), "fleet", cache.GetFleetId(), "slot_unavailable").WithSlot(ambitSlot)
        }
        if (slot != "") {
            return types.NewStructBuildError(structure.GetTypeId(), "fleet", cache.GetFleetId(), "slot_occupied").WithSlot(ambitSlot).WithExistingStruct(slot)
        }
    }
    return nil
}


func (cache *FleetCache) SetSlot(structure types.Struct) (err error) {
    if (!cache.FleetLoaded) { cache.LoadFleet() }

    switch structure.OperatingAmbit {
        case types.Ambit_water:
            cache.Fleet.Water[structure.Slot] = structure.Id
        case types.Ambit_land:
            cache.Fleet.Land[structure.Slot]  = structure.Id
        case types.Ambit_air:
            cache.Fleet.Air[structure.Slot]   = structure.Id
        case types.Ambit_space:
            cache.Fleet.Space[structure.Slot] = structure.Id
        default:
            err = types.NewStructLocationError(0, structure.OperatingAmbit.String(), "invalid_ambit").WithStruct(structure.Id)
    }
	cache.Changed = true
	return
}



func (cache *FleetCache) ClearSlot(ambit types.Ambit, slot uint64)  {
    if (!cache.FleetLoaded) { cache.LoadFleet() }

    switch ambit {
        case types.Ambit_water:
            cache.Fleet.Water[slot] = ""
        case types.Ambit_land:
            cache.Fleet.Land[slot]  = ""
        case types.Ambit_air:
            cache.Fleet.Air[slot]   = ""
        case types.Ambit_space:
            cache.Fleet.Space[slot] = ""
    }
    cache.Changed = true
}


func (cache *FleetCache) SetCommandStruct(structId string) {
    if (!cache.FleetLoaded) { cache.LoadFleet() }

    cache.Fleet.CommandStruct = structId
    cache.Changed = true
}

func (cache *FleetCache) ClearCommandStruct() {
    if (!cache.FleetLoaded) { cache.LoadFleet() }

    cache.Fleet.CommandStruct = ""
    cache.Changed = true
}


func (cache *FleetCache) MigrateToNewPlanet(destination *PlanetCache) {
    if (!cache.FleetLoaded) {
        cache.LoadFleet()
    }

    // Online Migrate if it's at home
    if cache.IsAway() { return }

    // Location updated and next call to GetPlanet() will pull the new location
    cache.Fleet.LocationId = destination.GetPlanetId()
    cache.Fleet.LocationType = types.ObjectType_planet

    cache.Changed = true
}