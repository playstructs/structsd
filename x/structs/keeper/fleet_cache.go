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
    Planet *PlanetCache

    ForwardFleetLoaded  bool
    ForwardFleetChanged bool
    ForwardFleet        *FleetCache


    BackwardFleetLoaded  bool
    BackwardFleetChanged bool
    BackwardFleet        *FleetCache

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
    }
    return cache.PlanetLoaded
}

// Load the Forward Fleet data
func (cache *FleetCache) LoadForwardFleet() (bool) {
    if (cache.GetLocationListForward() != "") {
        forwardFleet := cache.K.GetFleetCacheFromId(cache.Ctx, cache.GetLocationListForward())
        cache.ForwardFleet = &forwardFleet
        cache.ForwardFleetLoaded = true
    }
    return cache.ForwardLoaded
}

// Load the Forward Fleet data
func (cache *FleetCache) LoadBackwardFleet() (bool) {
    if (cache.GetLocationListBackward() != "") {
        backwardFleet := cache.K.GetFleetCacheFromId(cache.Ctx, cache.GetLocationListBackward())
        cache.BackwardFleet = &backwardFleet
        cache.BackwardFleetLoaded = true
    }
    return cache.BackwardFleetLoaded
}

func (cache *FleetCache) ManualLoadPlanet(planet *PlanetCache) {
    cache.Planet = planet
    cache.PlanetLoaded = true
}

func (cache *FleetCache) ManualLoadOwner(owner *PlayerCache) {
    cache.Owner = owner
    cache.OwnerLoaded = true
}



func (cache *FleetCache) GetFleet() (types.Fleet) {
    if (!cache.FleetLoaded) { cache.LoadFleet() }

    return cache.Fleet
}


func (cache *FleetCache) GetFleetId() (string) {
    return cache.FleetId
}

func (cache *FleetCache) GetPlanet() (*PlanetCache) {
    if (!cache.PlanetLoaded) { cache.LoadPlanet() }
    return cache.Planet
}


func (cache *FleetCache) GetLocationId() (string) {
    return cache.GetFleet().LocationId
}

func (cache *FleetCache) GetLocationType() (types.ObjectType) {
    return cache.GetFleet().LocationType
}


func (cache *FleetCache) GetLocationListForward() (string) {
    return cache.GetFleet().LocationListForward
}

func (cache *FleetCache) SetLocationListForward(fleetId string) () {
    cache.GetFleet().SetLocationListForward(fleetId)
    FleetChanged = true
}


func (cache *FleetCache) SetLocationListBackward(fleetId string) () {
    cache.GetFleet().LocationListBackward = fleetId
    FleetChanged = true
}

func (cache *FleetCache) SetForwardLocationListBackward(fleetId string) () {
    cache.GetForwardFleet().SetLocationListBackward(fleetId)
    ForwardFleetChanged = true
}

func (cache *FleetCache) SetForwardLocationListForward(fleetId string) () {
    cache.GetForwardFleet().SetLocationListForward(fleetId)
    ForwardFleetChanged = true
}

func (cache *FleetCache) GetForwardFleet() (*FleetCache) {
    if (!cache.ForwardFleetLoaded) { cache.LoadForwardFleet() }

    return cache.ForwardFleet
}


func (cache *FleetCache) GetLocationListBackward() (string) {
    return cache.GetFleet().LocationListBackward
}


func (cache *FleetCache) SetBackwardsLocationListBackward(fleetId string) () {
    cache.GetBackwardFleet().SetLocationListBackward(fleetId)
    BackwardFleetChanged = true
}

func (cache *FleetCache) SetBackwardsLocationListForward(fleetId string) () {
    cache.GetBackwardFleet().SetLocationListForward(fleetId)
    BackwardFleetChanged = true
}

func (cache *FleetCache) GetBackwardFleet() (*FleetCache) {
    if (!cache.BackwardFleetLoaded) { cache.LoadBackwardFleet() }

    return cache.BackwardFleet
}



// Get the Owner data
func (cache *FleetCache) GetOwner() (*PlayerCache) {
    if (!cache.OwnerLoaded) { cache.LoadOwner() }
    return cache.Owner
}

func (cache *FleetCache) GetOwnerId() (string) {
    return cache.GetFleet().Owner
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

func (cache *FleetCache) SetLocation(locationId string, locationType types.ObjectType) {
    if (!cache.FleetLoaded) { cache.LoadFleet() }

    // Position the Fleet in the queue
    if (cache.GetOwner().GetPlanetId() == locationId) {
        // The Fleet has returned home. Clear out the other markets
        // This needs to update the related Fleets as well!
        cache.ClearFleetQueue()

        if (cache.GetLocationListForward() == "") {
            if (cache.GetLocationListBackward() != "") {
                cache.GetPlanet().SetLocationListStart(cache.GetLocationListBackward())
                cache.SetBackwardsLocationListForward("")
            } else {
                // might need to make this a local function and track updating planet for cascade of commit
                cache.GetPlanet().SetLocationListStart("")
            }
        }  else {
            cache.
        }


    } else {
        if (destination.GetLocationListLast() != "") {
            previousLastFleet, _ := k.GetFleetCacheFromId(ctx, destination.GetLocationListLast())
            previousLastFleet.SetLocationListBackward(fleet.GetFleetId())
            previousLastFleet.Commit()

            fleet.SetLocationListForward(destination.GetLocationListLast())

        }

        destination.SetLocationListLast(fleet.GetFleetId())

    }


    cache.Fleet.LocationId = locationId
    cache.Fleet.LocationType = locationType
    cache.FleetChanged = true



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
    cache.SetLocation(cache.GetOwner().GetPlanetId(), types.ObjectType_planet)
}

