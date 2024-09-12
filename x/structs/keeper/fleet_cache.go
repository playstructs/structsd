package keeper

import (

	"context"
    //"math"
    //"fmt"

	//sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"

)

type FleetCache struct {
    FleetId string
    K *Keeper
    Ctx context.Context


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

    }, nil
}


func (cache *FleetCache) Commit() () {
    if (cache.FleetChanged) { cache.K.SetFleet(cache.Ctx, cache.Fleet) }

    if (cache.PlanetChanged) { cache.GetPlanet().Commit() }
    if (cache.PreviousPlanetChanged) { cache.GetPreviousPlanet().Commit() }

    if (cache.ForwardFleetChanged) { cache.GetForwardFleet().Commit() }

    if (cache.BackwardFleetChanged) { cache.GetBackwardFleet().Commit() }

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
    cache.PreviousPlanet = cache.Planet
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

            cache.GetPreviousBackwardFleet().SetLocationListForward("")
            cache.PreviousBackwardFleetChanged = true

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
    }

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

    // Send Fleet home
    cache.SetLocationToPlanet(cache.GetOwner().GetPlanet())
}

