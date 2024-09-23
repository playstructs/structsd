package keeper

import (

	"context"
    //"math"
    "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"

)

type FleetCache struct {
    FleetId string
    K *Keeper
    Ctx context.Context

    AnyChange bool

    FleetLoaded  bool
    FleetChanged bool
    Fleet        types.Fleet

    OwnerLoaded bool
    Owner *PlayerCache

    PlanetLoaded bool
    PlanetChanged bool
    Planet *PlanetCache

    PreviousPlanetLoaded bool
    PreviousPlanetChanged bool
    PreviousPlanet *PlanetCache

    ForwardFleetLoaded  bool
    ForwardFleetChanged bool
    ForwardFleet        *FleetCache

    PreviousForwardFleetLoaded  bool
    PreviousForwardFleetChanged bool
    PreviousForwardFleet        *FleetCache

    BackwardFleetLoaded  bool
    BackwardFleetChanged bool
    BackwardFleet        *FleetCache

    PreviousBackwardFleetLoaded  bool
    PreviousBackwardFleetChanged bool
    PreviousBackwardFleet        *FleetCache


}


func (k *Keeper) GetFleetCacheFromId(ctx context.Context, fleetId string) (FleetCache, error) {
    return FleetCache{
        FleetId: fleetId,
        K: k,
        Ctx: ctx,

        AnyChange: false,

    }, nil
}


func (cache *FleetCache) Commit() () {
    cache.AnyChange = false

    fmt.Printf("\n Updating Fleet From Cache (%s) \n", cache.FleetId)

    if (cache.FleetChanged) { cache.K.SetFleet(cache.Ctx, cache.Fleet) }

    if (cache.Owner != nil && cache.GetOwner().IsChanged()) { cache.GetOwner().Commit() }

    if (cache.Planet != nil && cache.GetPlanet().IsChanged()) { cache.GetPlanet().Commit() }
    if (cache.PreviousPlanet != nil && cache.GetPreviousPlanet().IsChanged()) { cache.GetPreviousPlanet().Commit() }

    if (cache.ForwardFleet != nil && cache.GetForwardFleet().IsChanged()) { cache.GetForwardFleet().Commit() }
    if (cache.BackwardFleet != nil && cache.GetBackwardFleet().IsChanged()) { cache.GetBackwardFleet().Commit() }

    if (cache.PreviousForwardFleet != nil && cache.GetPreviousForwardFleet().IsChanged()) { cache.GetPreviousForwardFleet().Commit() }
    if (cache.PreviousBackwardFleet != nil && cache.GetPreviousBackwardFleet().IsChanged()) { cache.GetPreviousBackwardFleet().Commit() }

}

func (cache *FleetCache) IsChanged() bool {
    return cache.AnyChange
}

func (cache *FleetCache) Changed() {
    cache.AnyChange = true
}


func (cache *FleetCache) LoadFleet() (found bool) {
    cache.Fleet, found = cache.K.GetFleet(cache.Ctx, cache.FleetId)

    if (found) {
        cache.FleetLoaded = true
    }

    return found
}

// Load the Player data
func (cache *FleetCache) LoadOwner() (bool) {
    newOwner, _ := cache.K.GetPlayerCacheFromId(cache.Ctx, cache.GetOwnerId())
    cache.Owner = &newOwner
    cache.OwnerLoaded = true
    return cache.OwnerLoaded
}

// Load the Planet data
func (cache *FleetCache) LoadPlanet() (bool) {
    if (cache.GetLocationType() == types.ObjectType_planet) {
        newPlanet := cache.K.GetPlanetCacheFromId(cache.Ctx, cache.GetLocationId())
        cache.Planet = &newPlanet
        cache.PlanetLoaded = true
        cache.PlanetChanged = false
    }
    return cache.PlanetLoaded
}

// Load the Forward Fleet data
func (cache *FleetCache) LoadForwardFleet() (bool) {
    if (cache.GetLocationListForward() != "") {
        forwardFleet, err := cache.K.GetFleetCacheFromId(cache.Ctx, cache.GetLocationListForward())
        cache.ForwardFleet = &forwardFleet
        if (err == nil) {
            cache.ForwardFleetLoaded = true
        }

    }
    return cache.ForwardFleetLoaded
}

// Load the Forward Fleet data
func (cache *FleetCache) LoadBackwardFleet() (bool) {
    if (cache.GetLocationListBackward() != "") {
        backwardFleet, err := cache.K.GetFleetCacheFromId(cache.Ctx, cache.GetLocationListBackward())
        cache.BackwardFleet = &backwardFleet
        if (err == nil) {
            cache.BackwardFleetLoaded = true
        }

    }
    return cache.BackwardFleetLoaded
}

func (cache *FleetCache) ManualLoadPlanet(planet *PlanetCache) {
    cache.Planet = planet
    cache.PlanetLoaded = true
    cache.PlanetChanged = false
}

func (cache *FleetCache) ManualLoadOwner(owner *PlayerCache) {
    cache.Owner = owner
    cache.OwnerLoaded = true
}

// Fleet Details
func (cache *FleetCache) GetFleet()     (types.Fleet)   { if (!cache.FleetLoaded) { cache.LoadFleet() }; return cache.Fleet }
func (cache *FleetCache) GetFleetId()   (string)        { return cache.FleetId }

// Ownership Details
func (cache *FleetCache) GetOwner()     (*PlayerCache)  { if (!cache.OwnerLoaded) { cache.LoadOwner() }; return cache.Owner }
func (cache *FleetCache) GetOwnerId()   (string)        { return cache.GetFleet().Owner }

// Location Details
func (cache *FleetCache) GetLocationId()        (string)            { return cache.GetFleet().LocationId }
func (cache *FleetCache) GetLocationType()      (types.ObjectType)  { return cache.GetFleet().LocationType }
func (cache *FleetCache) GetPlanet()            (*PlanetCache)      { if (!cache.PlanetLoaded) { cache.LoadPlanet() }; return cache.Planet }
func (cache *FleetCache) GetPreviousPlanet()    (*PlanetCache)      { return cache.PreviousPlanet }

// Planet Battle Queue Position
func (cache *FleetCache) GetLocationListForward()   (string)        { return cache.GetFleet().LocationListForward }
func (cache *FleetCache) GetForwardFleet()          (*FleetCache)   { if (!cache.ForwardFleetLoaded) { cache.LoadForwardFleet() }; return cache.ForwardFleet }
func (cache *FleetCache) GetPreviousForwardFleet()  (*FleetCache)   { return cache.PreviousForwardFleet }

func (cache *FleetCache) GetLocationListBackward()  (string)        { return cache.GetFleet().LocationListBackward }
func (cache *FleetCache) GetBackwardFleet()         (*FleetCache)   { if (!cache.BackwardFleetLoaded) { cache.LoadBackwardFleet() }; return cache.BackwardFleet }
func (cache *FleetCache) GetPreviousBackwardFleet() (*FleetCache)   { return cache.PreviousBackwardFleet }


func (cache *FleetCache) SetLocationListForward(fleetId string) () {
    if (!cache.FleetLoaded) { cache.LoadFleet() }

    cache.Fleet.LocationListForward = fleetId
    cache.FleetChanged = true
}

func (cache *FleetCache) SetLocationListBackward(fleetId string) () {
    if (!cache.FleetLoaded) { cache.LoadFleet() }

    cache.Fleet.LocationListBackward = fleetId
    cache.FleetChanged = true
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

    // TODO/MVP
    // One day it'll matter that the previous destination might not be a planet
    // Until that day, let's not complicate this further.

    // If we're already there, let's not waste cycles and writes.
    if (cache.GetLocationId() == destination.GetPlanetId()) { return }

    // Let's do some initial copies
    cache.PreviousPlanet = cache.GetPlanet()
    cache.PreviousPlanetChanged = cache.PlanetChanged
    cache.PreviousPlanetLoaded = cache.PlanetLoaded

    previousForwardFleetId := cache.GetLocationListForward()
    cache.PreviousForwardFleet = cache.GetForwardFleet()
    cache.PreviousForwardFleetLoaded = cache.ForwardFleetLoaded
    cache.PreviousForwardFleetChanged = cache.ForwardFleetChanged

    previousBackwardFleetId := cache.GetLocationListBackward()
    cache.PreviousBackwardFleet = cache.GetBackwardFleet()
    cache.PreviousBackwardFleetLoaded = cache.BackwardFleetLoaded
    cache.PreviousBackwardFleetChanged = cache.BackwardFleetChanged

    // Location updated and next call to GetPlanet() will pull the new location
    cache.Fleet.LocationId = destination.GetPlanetId()
    cache.Fleet.LocationType = types.ObjectType_planet
    cache.FleetChanged = true

    cache.Planet = destination


    // Old destination wasn't home - update all the previous stuff
    if (cache.GetOwner().GetPlanetId() != cache.GetPreviousPlanet().GetPlanetId()) {

        // Are we at the start of the list?
        if (previousForwardFleetId == "") {
            cache.GetPreviousPlanet().SetLocationListStart(previousBackwardFleetId)
            cache.PreviousPlanetChanged = true

            if (cache.GetPreviousBackwardFleet() != nil) {
                cache.GetPreviousBackwardFleet().SetLocationListForward("")
                cache.PreviousBackwardFleetChanged = true
            }


        // The back of the list
        } else if (previousBackwardFleetId == "") {
            cache.GetPreviousPlanet().SetLocationListLast(previousForwardFleetId)
            cache.PreviousPlanetChanged = true

            cache.GetPreviousForwardFleet().SetLocationListBackward("")
            cache.PreviousForwardFleetChanged = true

        // Or Somewhere In The Between
        } else {
            cache.GetPreviousForwardFleet().SetLocationListBackward(previousBackwardFleetId)
            cache.PreviousForwardFleetChanged = true

            cache.GetPreviousBackwardFleet().SetLocationListForward(previousForwardFleetId)
            cache.PreviousBackwardFleetChanged = true

        }

        cache.SetLocationListForward("")
        cache.SetLocationListBackward("")
    }

    // New destination isn't home - add it to the end of the list
    if (cache.GetOwner().GetPlanetId() != destination.GetPlanetId()) {

        // Is it the first fleet to arrive?
        if (cache.GetPlanet().GetLocationListStart() == "") {
            cache.GetPlanet().SetLocationListStart(cache.GetFleetId())
            cache.PlanetChanged = true

        } else {
            cache.SetLocationListForward(cache.GetPlanet().GetLocationListLast())

            cache.GetForwardFleet().SetLocationListBackward(cache.GetFleetId())
            cache.ForwardFleetChanged = true
        }

        cache.GetPlanet().SetLocationListLast(cache.GetFleetId())
        cache.PlanetChanged = true

        cache.Fleet.Status = types.FleetStatus_away
    } else {
        cache.Fleet.Status = types.FleetStatus_onStation
    }


    cache.Changed()

}

func (cache *FleetCache) PlanetMoveReadinessCheck() (error) {
    if cache.GetOwner().IsOffline() {
        return sdkerrors.Wrapf(types.ErrGridMalfunction, "Player (%s) is offline due to power", cache.GetOwnerId())
    }

    if !cache.HasCommandStruct() {
        return sdkerrors.Wrapf(types.ErrGridMalfunction, "Fleet (%s) needs a Command Struct before deploy", cache.GetFleetId())
    }

    return nil
}

func (cache *FleetCache) Defeat() (){
    if (!cache.FleetLoaded) { cache.LoadFleet() }

    uctx := sdk.UnwrapSDKContext(cache.Ctx)
    _ = uctx.EventManager().EmitTypedEvent(&types.EventRaid{&types.EventRaidDetail{FleetId: cache.GetFleetId(), PlanetId: cache.GetPlanet().GetPlanetId(), Status: types.RaidStatus_attackerDefeated}})

    // Send Fleet home
    cache.SetLocationToPlanet(cache.GetOwner().GetPlanet())
}


func (cache *FleetCache) BuildInitiateReadiness(structure *types.Struct, structType *types.StructType, ambit types.Ambit, ambitSlot uint64) (error) {
    if structure.GetOwner() != cache.GetOwnerId() {
         sdkerrors.Wrapf(types.ErrStructAction, "Struct owner must match fleet ")
    }

    if cache.IsAway() {
        sdkerrors.Wrapf(types.ErrStructAction, "Structs cannot be built unless Fleet is On Station")
    }

    if (structType.Category != types.ObjectType_fleet) {
        sdkerrors.Wrapf(types.ErrStructAction, "Struct Type cannot exist in this location ")
    }

    // Check that the Struct can exist in the specified ambit
    if types.Ambit_flag[ambit]&structType.PossibleAmbit == 0 {
        return sdkerrors.Wrapf(types.ErrStructAction, "Struct cannot be exist in the defined ambit (%s) based on structType (%d) ", ambit, structType.Id)
    }

    if structType.Type == types.CommandStruct {
        if cache.HasCommandStruct() {
            return sdkerrors.Wrapf(types.ErrStructBuildInitiate, "The fleet (%s) already has a Command Struct", cache.GetFleetId())
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
                return sdkerrors.Wrapf(types.ErrStructBuildInitiate, "The Struct Build was initiated on a non-existent ambit")
        }

        if (ambitSlot >= slots) {
            return sdkerrors.Wrapf(types.ErrStructBuildInitiate, "The Fleet (%s) specified doesn't have that slot available to build on", cache.GetFleetId())
        }
        if (slot != "") {
            return sdkerrors.Wrapf(types.ErrStructBuildInitiate, "The Fleet (%s) specified already has a struct on that slot", cache.GetFleetId())
        }
    }
    return nil
}



func (cache *FleetCache) MoveReadiness(structure *StructCache, ambit types.Ambit, ambitSlot uint64) (error) {
    if structure.GetOwnerId() != cache.GetOwnerId() {
         sdkerrors.Wrapf(types.ErrStructAction, "Struct owner must match fleet ")
    }

    if cache.IsAway() {
        sdkerrors.Wrapf(types.ErrStructAction, "Structs cannot be built unless Fleet is On Station")
    }

    if (structure.GetStructType().Category != types.ObjectType_fleet) {
        sdkerrors.Wrapf(types.ErrStructAction, "Struct Type cannot exist in this location ")
    }

    // Check that the Struct can exist in the specified ambit
    if types.Ambit_flag[ambit]&structure.GetStructType().PossibleAmbit == 0 {
        return sdkerrors.Wrapf(types.ErrStructAction, "Struct cannot be exist in the defined ambit (%s) based on structType (%d) ", ambit, structure.GetStructType().GetId())
    }

    if structure.GetStructType().Type == types.CommandStruct {
        if cache.HasCommandStruct() {
            return sdkerrors.Wrapf(types.ErrStructBuildInitiate, "The fleet (%s) already has a Command Struct", cache.GetFleetId())
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
                return sdkerrors.Wrapf(types.ErrStructBuildInitiate, "The Struct Build was initiated on a non-existent ambit")
        }

        if (ambitSlot >= slots) {
            return sdkerrors.Wrapf(types.ErrStructBuildInitiate, "The Fleet (%s) specified doesn't have that slot available to build on", cache.GetFleetId())
        }
        if (slot != "") {
            return sdkerrors.Wrapf(types.ErrStructBuildInitiate, "The Fleet (%s) specified already has a struct on that slot", cache.GetFleetId())
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
            err = sdkerrors.Wrapf(types.ErrStructAction, "Struct cannot exist in the defined ambit (%s) ", structure.OperatingAmbit)
    }
	cache.FleetChanged = true
	cache.Changed()
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
    cache.FleetChanged = true
    cache.Changed()
}


func (cache *FleetCache) SetCommandStruct(structure types.Struct) {
    if (!cache.FleetLoaded) { cache.LoadFleet() }

    cache.Fleet.CommandStruct = structure.Id
    cache.FleetChanged = true
    cache.Changed()
}

func (cache *FleetCache) ClearCommandStruct() {
    if (!cache.FleetLoaded) { cache.LoadFleet() }

    cache.Fleet.CommandStruct = ""
    cache.FleetChanged = true
    cache.Changed()
}


func (cache *FleetCache) MigrateToNewPlanet(destination *PlanetCache) {

    if (!cache.FleetLoaded) {
        if !cache.LoadFleet() {
            newFleet := cache.K.AppendFleet(cache.Ctx, cache.GetOwner())
            cache.Fleet = newFleet
            cache.FleetLoaded = true
        }
    }

    // Online Migrate if it's at home
    if cache.IsAway() { return }


    // Location updated and next call to GetPlanet() will pull the new location
    cache.Fleet.LocationId = destination.GetPlanetId()
    cache.Fleet.LocationType = types.ObjectType_planet
    cache.FleetChanged = true

    cache.Planet = destination
    cache.PlanetChanged = true
    cache.Changed()

}