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

func (cache *FleetCache) GetPreviousPlanet() (*PlanetCache) {
    return cache.PreviousPlanet
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
    if (!cache.FleetLoaded) { cache.LoadFleet() }

    cache.Fleet.LocationListForward = fleetId
    cache.FleetChanged = true
}


func (cache *FleetCache) SetLocationListBackward(fleetId string) () {
    if (!cache.FleetLoaded) { cache.LoadFleet() }

    cache.Fleet.LocationListBackward = fleetId
    cache.FleetChanged = true
}

func (cache *FleetCache) SetForwardFleetLocationListBackward(fleetId string) () {
    cache.GetForwardFleet().SetLocationListBackward(fleetId)
    cache.ForwardFleetChanged = true
}

func (cache *FleetCache) SetForwardFleetLocationListForward(fleetId string) () {
    cache.GetForwardFleet().SetLocationListForward(fleetId)
    cache.ForwardFleetChanged = true
}

func (cache *FleetCache) GetForwardFleet() (*FleetCache) {
    if (!cache.ForwardFleetLoaded) { cache.LoadForwardFleet() }

    return cache.ForwardFleet
}


func (cache *FleetCache) GetLocationListBackward() (string) {
    return cache.GetFleet().LocationListBackward
}


func (cache *FleetCache) SetBackwardFleetLocationListBackward(fleetId string) () {
    cache.GetBackwardFleet().SetLocationListBackward(fleetId)
    cache.BackwardFleetChanged = true
}

func (cache *FleetCache) SetBackwardFleetLocationListForward(fleetId string) () {
    cache.GetBackwardFleet().SetLocationListForward(fleetId)
    cache.BackwardFleetChanged = true
}

func (cache *FleetCache) GetBackwardFleet() (*FleetCache) {
    if (!cache.BackwardFleetLoaded) { cache.LoadBackwardFleet() }

    return cache.BackwardFleet
}

func (cache *FleetCache) SetPlanetLocationListStart(fleetId string) ()  {
    cache.GetPlanet().SetLocationListStart(fleetId)
    cache.PlanetChanged = true
}

func (cache *FleetCache) SetPlanetLocationListLast(fleetId string) ()  {
    cache.GetPlanet().SetLocationListLast(fleetId)
    cache.PlanetChanged = true
}


func (cache *FleetCache) ClearFleetListPointers() () {
    if (!cache.FleetLoaded) { cache.LoadFleet() }

    cache.Fleet.LocationListForward = ""
    cache.Fleet.LocationListBackward = ""
    cache.FleetChanged = true
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

func (cache *FleetCache) SetLocationToPlanet(destination *PlanetCache) {

    // TODO/MVP
    // One day it'll matter that the previous destination might not be a planet
    // Until that day, let's not complicate this further.

    if (cache.GetLocationId() == destination.GetPlanetId()) { return }

    // Let's do some initial copies
    cache.PreviousPlanet = cache.Planet
    cache.PreviousPlanetChanged = cache.PlanetChanged
    cache.PreviousPlanetLoaded = cache.PlanetLoaded

    cache.PreviousForwardFleet = cache.GetBackwardFleet()
    cache.PreviousForwardFleetLoaded = cache.BackwardFleetLoaded
    cache.PreviousForwardFleetChanged = cache.BackwardFleetChanged

    cache.PreviousBackwardFleet = cache.GetBackwardFleet()
    cache.PreviousBackwardFleetLoaded = cache.BackwardFleetLoaded
    cache.PreviousBackwardFleetChanged = cache.BackwardFleetChanged

    // Location updated and next call to GetPlanet() will pull the new location
    cache.Fleet.LocationId = destination.GetPlanetId()
    cache.Fleet.LocationType = types.ObjectType_planet
    cache.FleetChanged = true

    cache.Planet = destination.GetPlanet()



    // Old destination wasn't home - update all the previous stuff
    if (cache.GetOwner().GetPlanetId() != destination.GetPlanetId()) {
        update the new shit

    }


    // New destination isn't home - add it to the end of the list
    if (cache.GetOwner().GetPlanetId() != destination.GetPlanetId()) {
        update the new shit

    }



    // always update the  old shit

    // Position the Fleet in the queue
    if (cache.GetOwner().GetPlanetId() == destination.GetPlanetId()) {
        // The Fleet has returned home. Clear out the other markets
        // This needs to update the related Fleets as well!

        // Is the Fleet currently the first in line?
        if (cache.GetLocationListForward() == "") {
            // Is there another Fleet in the list?
            if (cache.GetLocationListBackward() != "") {
                // Promote the next ship in the list to be at the planet

                // Update the planet pointer so it knows which
                // Fleet is now first
                cache.SetPlanetLocationListStart(cache.GetLocationListBackward())

                // Update the next Fleet in line to know it's at the Planet
                cache.Set - previous? - BackwardFleetLocationListForward("")

            // There were no other fleets in the list, so empty the Planets pointers
            } else {
                cache.SetPlanetLocationListStart("")
                cache.SetPlanetLocationListLast("")
            }
        // The Fleet was not the first in line and is somewhere else in the fleet list
        }  else {

            // If the Fleet is at the end of the line
            if (cache.GetLocationListBackward() == "") {

                // Update the Planet to point to the new end, the Fleet in front
                cache.SetPlanetLocationListLast(cache.GetLocationListForward())
                // The fleet in front should not point to nothing instead of the current fleet
                cache.SetForwardFleetLocationListBackward("")

            // The fleet is somewhere in the middle and needs updates to the other two fleets flanking it
            } else {
                // Nothing on the planet needs to be updated

                // Update the forward fleet's backwards pointer to point to the backward fleet
                cache.SetForwardFleetLocationListBackward(cache.GetLocationListBackward())
                // Update the backward fleet's forward pointer to point to the forward fleet
                cache.SetBackwardFleetLocationListForward(cache.GetLocationListForward())
            }


        }

        // Now that the other fleets and the planet have all been updated, we can clear the old pointer values
        cache.ClearFleetListPointers()

        // Now let's push the old planet into the previous pointer
        // This leaves it for the Commit() to cascade to it
        cache.PreviousPlanet = cache.Planet
        cache.PreviousPlanetChanged = cache.PlanetChanged
        cache.PreviousPlanetLoaded = true

        // Location updated and next call to GetPlanet() will pull the new location
        cache.Fleet.LocationId = destination.GetPlanetId()
        cache.Fleet.LocationType = types.ObjectType_planet
        cache.FleetChanged = true

        cache.Planet = destination
        cache.PlanetLoaded = true


    // This fleet is moving to another planet other than it's home world
    } else {

        // Is it leaving its home?
        if (cache.GetLocationId() == cache.GetOwner().GetPlanetId()) {
            // No updates should be made to the pointers of fleets and planets since
            // fleets at home are treated differently than fleets abroad.
            // They're considered to be on the planet



            // Update the current end to point to this fleet


            // Update the end of the Planet Pointer
            destination.SetLocationListLast(cache.GetFleetId())




            // update this fleet to point to that fleet

        // We're leaving one planet to go to another.
        // Probably the more complicated movement
        } else {

        }




        // Is the Fleet currently the first in line?
        if (cache.GetLocationListForward() == "") {
            // Is there another Fleet in the list?
            if (cache.GetLocationListBackward() != "") {
                // Promote the next ship in the list to be at the planet

                // Update the planet pointer so it knows which
                // Fleet is now first
                cache.SetPlanetLocationListStart(cache.GetLocationListBackward())

                // Update the next Fleet in line to know it's at the Planet
                cache.SetBackwardFleetLocationListForward("")

            // There were no other fleets in the list, so empty the Planets pointers
            } else {
                cache.SetPlanetLocationListStart("")
                cache.SetPlanetLocationListLast("")
            }
        // The Fleet was not the first in line and is somewhere else in the fleet list
        }  else {

            // If the Fleet is at the end of the line
            if (cache.GetLocationListBackward() == "") {

                // Update the Planet to point to the new end, the Fleet in front
                cache.SetPlanetLocationListLast(cache.GetLocationListForward())
                // The fleet in front should not point to nothing instead of the current fleet
                cache.SetForwardFleetLocationListBackward("")

            // The fleet is somewhere in the middle and needs updates to the other two fleets flanking it
            } else {
                // Nothing on the planet needs to be updated

                // Update the forward fleet's backwards pointer to point to the backward fleet
                cache.SetForwardFleetLocationListBackward(cache.GetLocationListBackward())
                // Update the backward fleet's forward pointer to point to the forward fleet
                cache.SetBackwardFleetLocationListForward(cache.GetLocationListForward())
            }


        }


    }


    cache.Fleet.LocationId = destination.GetPlanetId()
    cache.Fleet.LocationType = types.ObjectType_planet
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
    cache.SetLocationToPlanet(cache.GetOwner().GetPlanet())
}

