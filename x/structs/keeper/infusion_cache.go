package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
    sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
	"github.com/nethruster/go-fraction"

    // Used in Randomness Orb
	"math/rand"
    "bytes"
    "encoding/binary"


)


type InfusionCache struct {
    DestinationType types.ObjectType
    DestinationId string
    Address string

    K *Keeper
    Ctx context.Context

    AnyChange bool
    Ready bool

    InfusionLoaded  bool
    InfusionChanged bool
    Infusion  types.Infusion

    OwnerLoaded bool
    Owner *PlayerCache

    DestinationFuelAttributeId string
    DestinationFuelLoaded bool
    DestinationFuelChanged bool
    DestinationFuel uint64

    DestinationCapacityAttributeId string
    DestinationCapacityLoaded bool
    DestinationCapacityChanged bool
    DestinationCapacity uint64

}

func (k *Keeper) GetInfusionCache(ctx context.Context, destinationType types.ObjectType, destinationId string, address string) (InfusionCache) {
    return PlanetCache{
        DestinationType: destinationType,
        DestinationId: destinationId,
        Address: address,

        K: k,
        Ctx: ctx,

        AnyChange: false,

        InfusionLoaded: false,
        InfusionChanged: false,
        OwnerLoaded: false,

        DestinationFuelAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_fuel, destinationId),
        DestinationFuelLoaded: false,
        DestinationFuelChanged: false,

        DestinationCapacityAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, destinationId),
        DestinationCapacityLoaded: false,
        DestinationCapacityChanged: false,

        //PlayerCapacityAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, destinationId),
        PlayerCapacityLoaded: false,
        PlayerCapacityChanged: false,

    }
}

func (cache *InfusionCache) Commit() () {
    cache.AnyChange = false

    cache.K.logger.Info("Updating Infusion From Cache","destinationId",cache.DestinationId,"address",cache.Address)

    if (cache.InfusionChanged) {
        cache.K.SetInfusion(cache.Ctx, cache.Infusion)
        cache.InfusionChanged = false
    }

    if (cache.Owner != nil && cache.GetOwner().IsChanged()) {
        cache.GetOwner().Commit()
    }

    if (cache.DestinationFuelChanged) {
        cache.K.SetPlanetAttribute(cache.Ctx, cache.DestinationFuelAttributeId, cache.DestinationFuel)
        cache.DestinationFuelChanged = false
    }

    if (cache.DestinationCapacityChanged) {
        cache.K.SetGridAttribute(cache.Ctx, cache.DestinationCapacityAttributeId, cache.DestinationCapacity)
        cache.DestinationCapacityChanged = false
    }

    if (cache.PLayerCapacityChanged) {
        cache.K.SetGridAttribute(cache.Ctx, cache.PlayerCapacityAttributeId, cache.PlayerCapacity)
        cache.PlayerCapacityChanged = false
    }

}

func (cache *InfusionCache) IsChanged() bool {
    return cache.AnyChange
}

func (cache *InfusionCache) Changed() {
    cache.AnyChange = true
}

// Load the core Infusion data
func (cache *InfusionCache) LoadInfusion() (bool) {
    cache.Infusion, cache.InfusionLoaded = cache.K.GetInfusion(cache.Ctx, cache.DestinationId, cache.Address)

    if !cache.InfusionLoaded {
        // TODO create it instead

        // TODO set ratio based on DestinationType DestinationId

        // TODO set commission based on DestinationType Destination


    }

    return cache.InfusionLoaded
}


// Load the Player data
func (cache *InfusionCache) LoadOwner() (bool) {
    newOwner, _ := cache.K.GetPlayerCacheFromId(cache.Ctx, cache.GetOwnerId())
    cache.Owner = &newOwner
    cache.OwnerLoaded = true
    return cache.OwnerLoaded
}


func (cache *InfusionCache) LoadDestinationFuel() {
    cache.DestinationFuel = cache.K.GetGridAttribute(cache.Ctx, cache.DestinationFuelAttributeId)
    cache.DestinationFuelLoaded = true
}

func (cache *InfusionCache) LoadDestinationCapacity() {
    cache.DestinationCapacity = cache.K.GetGridAttribute(cache.Ctx, cache.DestinationCapacityAttributeId)
    cache.DestinationCapacityLoaded = true
}

func (cache *InfusionCache) LoadPlayerCapacity() {
    // TODO get playerId from address

    cache.PlayerCapacity = cache.K.GetGridAttribute(cache.Ctx, cache.PlayerCapacityAttributeId)
    cache.PlayerCapacityLoaded = true
}

/* Getters
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */

// Get the Owner ID data
func (cache *InfusionCache) GetOwnerId() (string) {
    if (!cache.InfusionLoaded) { cache.LoadInfusion() }
    return cache.Infusion.PlayerId
}

// Get the Owner data
func (cache *InfusionCache) GetOwner() (*PlayerCache) {
    if (!cache.OwnerLoaded) { cache.LoadOwner() }
    return cache.Owner
}

func (cache *InfusionCache) GetInfusion() (types.Infusion) {
    if (!cache.InfusionLoaded) { cache.LoadInfusion() }
    return cache.Infusion
}

func (cache *InfusionCache) GetDestinationFuel() (uint64) {
    if (!cache.DestinationFuelLoaded) { cache.LoadDestinationFuel() }
    return cache.DestinationFuel
}

func (cache *InfusionCache) GetDestinationCapacity() (uint64) {
    if (!cache.DestinationCapacityLoaded) { cache.LoadDestinationCapacity() }
    return cache.DestinationCapacity
}

func (cache *InfusionCache) GetPlayerCapacity() (uint64) {
    if (!cache.PlayerCapacityLoaded) { cache.LoadPlayerCapacity() }
    return cache.PlayerCapacity
}


func (cache *InfusionCache) GetInfusionIdentifiers() (destinationId string, address string) {
    return cache.DestinationId, cache.Address
}


/* Setters - SET DOES NOT COMMIT()
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */

// TODO this function is stupid, how can we get this information in.

func (cache *InfusionCache) SetCommission(commission math.LegacyDec) {
    if (!cache.InfusionLoaded) { cache.LoadInfusion() }

    oldInfusionPower       = cache.Infusion.Power

    oldCommissionPower     = cache.Infusion.Commission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(oldInfusionPower))).RoundInt().Uint64()
    oldPlayerPower         = cache.Infusion.Power - oldCommissionPower

    newInfusionPower            = CalculateInfusionPower(cache.Infusion.Ratio, cache.Infusion.Fuel)
    newCommissionPower          = newCommission.Mul(math.LegacyNewDecFromInt(math.NewIntFromUint64(newInfusionPower))).RoundInt().Uint64()
    newPlayerPower              = newInfusionPower - newCommissionPower


    cache.Infusion.Commission  = newCommission
    cache.Infusion.Power       = newInfusionPower


    // TODO... the rest of things that could change


}

func (cache *InfusionCache) SetFuel(fuel uint64) () {
    if (!cache.InfusionLoaded) { cache.LoadInfusion() }

    // TODO make suck less
    cache.Infusion.Fuel = fuel
    cache.InfusionChanged = true
    cache.Changed()
}



// Set the Owner data manually
// Useful for loading multiple defenders
func (cache *InfusionCache) ManualLoadOwner(owner *PlayerCache) {
    cache.Owner = owner
    cache.OwnerLoaded = true
}


