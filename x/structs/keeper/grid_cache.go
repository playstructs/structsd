package keeper

import (
	"context"

	//sdk "github.com/cosmos/cosmos-sdk/types"
    //sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"

    "fmt"

)


type GridCache struct {
    ObjectId string
    K *Keeper
    Ctx context.Context

    AnyChange bool

    OreAttributeId string
    OreLoaded bool
    OreChanged bool
    Ore uint64

    FuelAttributeId string
    FuelLoaded bool
    FuelChanged bool
    Fuel uint64

    CapacityAttributeId string
    CapacityLoaded bool
    CapacityChanged bool
    Capacity   uint64

    LoadAttributeId string
    LoadLoaded bool
    LoadChanged bool
    Load uint64

    StructsLoadAttributeId string
    StructsLoadLoaded bool
    StructsLoadChanged bool
    StructsLoad uint64

    PowerAttributeId string
    PowerLoaded bool
    PowerChanged bool
    Power uint64

    ConnectionCapacityAttributeId string
    ConnectionCapacityLoaded bool
    ConnectionCapacityChanged bool
    ConnectionCapacity uint64

    ConnectionCountAttributeId string
    ConnectionCountLoaded bool
    ConnectionCountChanged bool
    ConnectionCount uint64

    ReadyAttributeId string
    ReadyLoaded bool
    ReadyChanged bool
    Ready uint64

}

// Build this initial Grid Cache object
// This does no validation on the provided gridId
func (k *Keeper) GetGridCacheFromId(ctx context.Context, objectId string) (GridCache) {
    return GridCache{
        ObjectId: objectId,
        K: k,
        Ctx: ctx,

        AnyChange: false,

        OreAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_ore, objectId),
        OreLoaded: false,
        OreChanged: false,

        FuelAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_fuel, objectId),
        FuelLoaded: false,
        FuelChanged: false,

        CapacityAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, objectId),
        CapacityLoaded: false,
        CapacityChanged: false,

        LoadAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_load, objectId),
        LoadLoaded: false,
        LoadChanged: false,

        StructsLoadAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, objectId),
        StructsLoadLoaded: false,
        StructsLoadChanged: false,

        PowerAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_power, objectId),
        PowerLoaded: false,
        PowerChanged: false,

        ConnectionCapacityAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, objectId),
        ConnectionCapacityLoaded: false,
        ConnectionCapacityChanged: false,

        ConnectionCountAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCount, objectId),
        ConnectionCountLoaded: false,
        ConnectionCountChanged: false,

        ReadyAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_ready, objectId),
        ReadyLoaded: false,
        ReadyChanged: false,

    }
}



func (cache *GridCache) Commit() () {
    cache.AnyChange = false

    fmt.Printf("\n Updating Grid From Cache (%s) \n", cache.ObjectId)

    if (cache.OreChanged) {
        cache.K.SetGridAttribute(cache.Ctx, cache.OreAttributeId, cache.Ore)
        cache.OreChanged = false
    }

    if (cache.FuelChanged) {
        cache.K.SetGridAttribute(cache.Ctx, cache.FuelAttributeId, cache.Fuel)
        cache.FuelChanged = false
    }

    if (cache.CapacityChanged) {
        cache.K.SetGridAttribute(cache.Ctx, cache.CapacityAttributeId, cache.Capacity)
        cache.CapacityChanged = false
    }

    if (cache.LoadChanged) {
        cache.K.SetGridAttribute(cache.Ctx, cache.LoadAttributeId, cache.Load)
        cache.LoadChanged = false
    }

    if (cache.StructsLoadChanged) {
        cache.K.SetGridAttribute(cache.Ctx, cache.StructsLoadAttributeId, cache.StructsLoad)
        cache.StructsLoadChanged = false
    }

    if (cache.PowerChanged) {
        cache.K.SetGridAttribute(cache.Ctx, cache.PowerAttributeId, cache.Power)
        cache.PowerChanged = false
    }

    if (cache.ConnectionCapacityChanged) {
        cache.K.SetGridAttribute(cache.Ctx, cache.ConnectionCapacityAttributeId, cache.ConnectionCapacity)
        cache.ConnectionCapacityChanged = false
    }

    if (cache.ConnectionCountChanged) {
        cache.K.SetGridAttribute(cache.Ctx, cache.ConnectionCountAttributeId, cache.ConnectionCount)
        cache.ConnectionCountChanged = false
    }

    if (cache.ReadyChanged) {
        cache.K.SetGridAttribute(cache.Ctx, cache.ReadyAttributeId, cache.Ready)
        cache.ReadyChanged = false
    }

}

func (cache *GridCache) IsChanged() bool {
    return cache.AnyChange
}

func (cache *GridCache) Changed() {
    cache.AnyChange = true
}

/* Separate Loading functions for each of the underlying containers */

func (cache *GridCache) LoadOre() {
    cache.Ore = cache.K.GetGridAttribute(cache.Ctx, cache.OreAttributeId)
    cache.OreLoaded = true
}

func (cache *GridCache) LoadFuel() {
    cache.Fuel = cache.K.GetGridAttribute(cache.Ctx, cache.FuelAttributeId)
    cache.FuelLoaded = true
}

func (cache *GridCache) LoadCapacity() {
    cache.Capacity = cache.K.GetGridAttribute(cache.Ctx, cache.CapacityAttributeId)
    cache.CapacityLoaded = true
}

func (cache *GridCache) LoadLoad() {
    cache.Load = cache.K.GetGridAttribute(cache.Ctx, cache.LoadAttributeId)
    cache.LoadLoaded = true
}

func (cache *GridCache) LoadStructsLoad() {
    cache.StructsLoad = cache.K.GetGridAttribute(cache.Ctx, cache.StructsLoadAttributeId)
    cache.StructsLoadLoaded = true
}

func (cache *GridCache) LoadPower() {
    cache.Power = cache.K.GetGridAttribute(cache.Ctx, cache.PowerAttributeId)
    cache.PowerLoaded = true
}

func (cache *GridCache) LoadConnectionCapacity() {
    cache.ConnectionCapacity = cache.K.GetGridAttribute(cache.Ctx, cache.ConnectionCapacityAttributeId)
    cache.ConnectionCapacityLoaded = true
}

func (cache *GridCache) LoadConnectionCount() {
    cache.ConnectionCount = cache.K.GetGridAttribute(cache.Ctx, cache.ConnectionCountAttributeId)
    cache.ConnectionCountLoaded = true
}

func (cache *GridCache) LoadReady() {
    cache.Ready = cache.K.GetGridAttribute(cache.Ctx, cache.ReadyAttributeId)
    cache.ReadyLoaded = true
}


/* Getters
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */

func (cache *GridCache) GetObjectId()               (string) {  return cache.ObjectId }

func (cache *GridCache) GetOre()                    (uint64) { if (!cache.OreLoaded) { cache.LoadOre() }; return cache.Ore }
func (cache *GridCache) GetFuel()                   (uint64) { if (!cache.FuelLoaded) { cache.LoadFuel() }; return cache.Fuel }
func (cache *GridCache) GetCapacity()               (uint64) { if (!cache.CapacityLoaded) { cache.LoadCapacity() }; return cache.Capacity }
func (cache *GridCache) GetLoad()                   (uint64) { if (!cache.LoadLoaded) { cache.LoadLoad() }; return cache.Load }
func (cache *GridCache) GetStructsLoad()            (uint64) { if (!cache.StructsLoadLoaded) { cache.LoadStructsLoad() }; return cache.StructsLoad }
func (cache *GridCache) GetPower()                  (uint64) { if (!cache.PowerLoaded) { cache.LoadPower() }; return cache.Power }

func (cache *GridCache) GetConnectionCapacity()     (uint64) { if (!cache.ConnectionCapacityLoaded) { cache.LoadConnectionCapacity() }; return cache.ConnectionCapacity }
func (cache *GridCache) GetConnectionCount()        (uint64) { if (!cache.ConnectionCountLoaded) { cache.LoadConnectionCount() }; return cache.ConnectionCount }

func (cache *GridCache) GetReady()                  (uint64) { if (!cache.ReadyLoaded) { cache.LoadReady() }; return cache.Ready }

/* Setters - SET DOES NOT COMMIT()
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */



