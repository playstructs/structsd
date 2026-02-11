package keeper

import (
	"context"

	//sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)


type SubstationCache struct {
    SubstationId string
    CC  *CurrentContext

    Ready bool
    Changed bool

    SubstationLoaded  bool
    Substation  types.Substation

    LoadAttributeId string
    CapacityAttributeId string
    ConnectionCountAttributeId string
    ConnectionCapacityAttributeId string

}



func (cache *SubstationCache) Commit() () {
    if cache.Changed {
        cache.CC.k.logger.Info("Updating Substation From Cache","substationId", cache.SubstationId)

        if cache.Deleted {
            cache.K.ClearSubstation(cache.CC.ctx, cache.SubstationId)
        } else {
            cache.CC.k.SetSubstation(cache.CC.ctx, cache.Substation)
        }
        cache.Changed = false
    }

}

func (cache *SubstationCache) IsChanged() bool {
    return cache.Changed
}

func (cache *SubstationCache) ID() string {
    return cache.SubstationId
}


/* Separate Loading functions for each of the underlying containers */

// Load the core Substation data
func (cache *SubstationCache) LoadSubstation() (bool) {
    cache.Substation, cache.SubstationLoaded = cache.K.GetSubstation(cache.Ctx, cache.SubstationId)
    return cache.SubstationLoaded
}

func (cache *SubstationCache) CheckSubstation() (error) {
    if (!cache.SubstationLoaded) {
        if !cache.LoadSubstation() {
           return types.NewObjectNotFoundError("substation", cache.SubstationId)
        }
    }
    return nil
}


/* Getters
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */

func (cache *SubstationCache) GetSubstation()   (types.Substation)  { if (!cache.SubstationLoaded) { cache.LoadSubstation() }; return cache.Substation }
func (cache *SubstationCache) GetSubstationId() (string)            { return cache.SubstationId }

func (cache *SubstationCache) GetOwnerId()      (string)            { if (!cache.SubstationLoaded) { cache.LoadSubstation() }; return cache.Substation.Owner }
func (cache *SubstationCache) GetAvailableCapacity() (uint64)       { return cache.GetGrid().GetCapacity() - cache.GetGrid().GetLoad() }

func (cache *SubstationCache) GetOwner()        (*PlayerCache)      { return cache.CC.GetPlayer( cache.GetOwnerId() ) }

func (cache *SubstationCache) Delete(migrationSubstation *SubstationCache) {

    // Migrate everyone away
    playerConnections := cache.CC.GetGridAttribute(cache.ConnectionCountAttributeId)
    if (playerConnections > 0) {
        connectedPlayers := cc.GetAllPlayerBySubstation(substationId)
        for _, disconnectPlayer := range connectedPlayers {
            if migrationSubstation.GetSubstationId() == cache.GetSubstationId() {
               disconnectPlayer.DisconnectSubstation()
            } else {
               disconnectPlayer.MigrateSubstation(migrationSubstation.GetSubstationId())
            }
        }
	}

	// Destroy allocations out
    allocationsOut := cc.k.GetAllAllocationIdBySourceIndex(cache.CC.ctx, substationId)
    cache.CC.DestroyMultipleAllocations(ctx, allocationsOut)

    allocationsIn := cc.k.GetAllAllocationIdByDestinationIndex(cache.CC.ctx, substationId)
    cache.CC.DestroyMultipleAllocations(ctx, allocationsIn)

	// Clear out Grid attributes
	cache.CC.ClearGridAttribute(cache.LoadAttributeId)
	cache.CC.ClearGridAttribute(cache.CapacityAttributeId)
	cache.CC.ClearGridAttribute(cache.ConnectionCountAttributeId)
	cache.CC.ClearGridAttribute(cache.ConnectionCapacityAttributeId)

    cache.Deleted = true
    cache.Changed = true
}

// Set the Owner Id data
func (cache *SubstationCache) SetOwnerId(owner string) {
    if (!cache.SubstationLoaded) { cache.LoadSubstation() }
    cache.Substation.Owner = owner
    cache.Changed = true
}


// Delete Permission
func (cache *SubstationCache) CanBeDeleteDBy(activePlayer *PlayerCache) (error) {
    return cache.PermissionCheck(types.PermissionDelete, activePlayer);
}

// Association Permission
func (cache *SubstationCache) CanManagePlayerConnections(activePlayer *PlayerCache) (error) {
    return cache.PermissionCheck(types.PermissionAssociations, activePlayer)
}

// Grid Permission
func (cache *SubstationCache) CanManageAllocationConnections(activePlayer *PlayerCache) (error) {
    return cache.PermissionCheck(types.PermissionGrid, activePlayer)
}

// Asset Permission
func (cache *SubstationCache) CanCreateAllocations(activePlayer *PlayerCache) (error) {
    return cache.PermissionCheck(types.PermissionAssets, activePlayer)
}

func (cache *SubstationCache) PermissionCheck(permission types.Permission, activePlayer *PlayerCache) (error) {
    // Make sure the address calling this has Play permissions
    if (!cache.CC.PermissionHasOneOf(GetAddressPermissionIDBytes(activePlayer.GetActiveAddress()), permission)) {
        return types.NewPermissionError("address", activePlayer.GetActiveAddress(), "", "", uint64(permission), "substation_action")
    }

    if !activePlayer.HasPlayerAccount() {
        return types.NewPlayerRequiredError(activePlayer.GetActiveAddress(), "substation_action")
    } else {
        if (activePlayer.GetPlayerId() != cache.GetOwnerId()) {
            if (!cache.CC.PermissionHasOneOf(GetObjectPermissionIDBytes(cache.GetSubstationId(), activePlayer.GetPlayerId()), permission)) {
               return types.NewPermissionError("player", activePlayer.GetPlayerId(), "substation", cache.GetSubstationId(), uint64(permission), "substation_action")
            }
        }
    }
    return nil
}

func (cache *SubstationCache) ConnectionCountDecrement(amount uint64) {
    cache.CC.SetGridAttributeDecrement(cache.ConnectionCountAttributeId, amount)
    cache.CC.UpdateSubstationConnectionCapacity(cache.SubstationId)
}

func (cache *SubstationCache) ConnectionCountIncrement(amount uint64) {
    cache.CC.SetGridAttributeIncrement(cache.ConnectionCountAttributeId, amount)
    cache.CC.UpdateSubstationConnectionCapacity(cache.SubstationId)
}

func (cache *SubstationCache) PlayerIncrease() {
    cache.CC.SetGridAttributeIncrement(cache.ConnectionCountAttributeId, 1)
    cc.UpdateSubstationConnectionCapacity(cache.ID())
}

func (cache *SubstationCache) PlayerDecrease(player *PlayerCache){
    cache.SetGridAttributeDecrement(cache.ConnectionCountAttributeId, 1)
    cc.UpdateSubstationConnectionCapacity(cache.ID())
}