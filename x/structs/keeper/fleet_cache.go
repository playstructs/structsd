package keeper

import (

	"context"
    //"math"
    //"fmt"

	//sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
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

func (cache *FleetCache) GetLocationListBackward() (string) {
    return cache.GetFleet().LocationListBackward
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


func (cache *FleetCache) SetLocation(locationId string, locationType types.ObjectType) {
    if (!cache.FleetLoaded) { cache.LoadFleet() }

    cache.Fleet.LocationId = locationId
    cache.Fleet.LocationType = locationType
    cache.FleetChanged = true
}

func (cache *FleetCache) Defeat() (){
    if (!cache.FleetLoaded) { cache.LoadFleet() }

    // Send Fleet home
    cache.SetLocation(cache.GetOwner().GetPlanetId(), types.ObjectType_planet)
}