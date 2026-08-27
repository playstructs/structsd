package keeper

import (
	"structs/x/structs/types"
)

func (cc *CurrentContext) GetSubstation(substationId string) *SubstationCache {
	if cache, exists := cc.substations[substationId]; exists {
		return cache
	}

	cc.substations[substationId] = &SubstationCache{
                SubstationId: substationId,
                CC: cc,
                Changed: false,

                LoadAttributeId:    	        GetGridAttributeIDByObjectId(types.GridAttributeType_load, substationId),
                CapacityAttributeId:    	    GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, substationId),
                ConnectionCountAttributeId:     GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCount, substationId),
                ConnectionCapacityAttributeId:  GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, substationId),
            }

	return cc.substations[substationId]
}

/* GetExistingSubstation resolves a substation id a message chose.
 *
 * GetSubstation is a cache allocator: it builds a SubstationCache around any
 * string and says nothing about whether a substation exists at it, which is fine
 * for an id read off something already loaded from state and wrong for one a
 * transaction supplied. ProviderCreate took the second and treated it as the
 * first, and the permission check did not save it - object permissions are keyed
 * by the raw id string, and every player holds PermAll on their own player id,
 * so submitting that id passed CanAllocateAsSourceBy against a substation that
 * was never there.
 *
 * The type check is not redundant with the existence check. Grid attribute ids
 * are derived from the object id alone, so a provider whose "substation" is a
 * player id would read and write that player's own capacity and load counters -
 * and a substation stored under a wrongly-typed id, which only a hand-written
 * genesis could produce, would collide the same way while existing perfectly
 * well.
 */
func (cc *CurrentContext) GetExistingSubstation(substationId string) (*SubstationCache, error) {
	if !ObjectIdHasType(substationId, types.ObjectType_substation) {
		return nil, types.NewObjectNotFoundError("substation", substationId)
	}

	substation := cc.GetSubstation(substationId)
	if err := substation.CheckSubstation(); err != nil {
		return nil, err
	}

	return substation, nil
}

func (cc *CurrentContext) GenesisImportSubstation(substation types.Substation) {
	cache := cc.GetSubstation(substation.Id)
	cache.Substation = substation
	cache.SubstationLoaded = true
	cache.Changed = true
}

// Build this initial Substation Cache object
// This does no validation on the provided substationId
func (cc *CurrentContext) NewSubstation(creatorAddress string, owner *PlayerCache, allocation *AllocationCache) (*SubstationCache, error) {
    var substation types.Substation
    substationId := GetObjectID(types.ObjectType_substation, cc.k.GetNextSubstationId(cc.ctx))

    substation.Id       = substationId
    substation.Owner    = owner.ID()
    substation.Creator  = creatorAddress

    // Start to put the pieces together
    cc.substations[substationId] = &SubstationCache{
                  SubstationId: substationId,
                  CC: cc,

                  Changed: true,
                  Substation: substation,
                  SubstationLoaded: true,

                  LoadAttributeId:    	        GetGridAttributeIDByObjectId(types.GridAttributeType_load, substationId),
                  CapacityAttributeId:    	    GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, substationId),
                  ConnectionCountAttributeId:   GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCount, substationId),
                  ConnectionCapacityAttributeId:GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, substationId),

    }

    // Update the allocations new destination
    allocationDestinationError := allocation.SetDestination(substation.Id)
    if allocationDestinationError != nil {
        cc.substations[substationId].Changed = false
        return cc.substations[substationId], allocationDestinationError
    }

    permissionId := GetObjectPermissionIDBytes(substationId, owner.ID())
    cc.PermissionAdd(permissionId, types.PermSubstationAll)

    return cc.substations[substationId], nil
}
