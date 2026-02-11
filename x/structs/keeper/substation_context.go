package keeper

// GetSubstation returns a SubstationCache by ID, loading from store if not already cached.
// Returns nil if the substation has been deleted in this context.
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
                  ConnectionCountAttributeId:   GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCount, substationId),
                  ConnectionCapacityAttributeId:GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, substationId),
            }

	return cc.substations[substationId]
}


// Build this initial Substation Cache object
// This does no validation on the provided substationId
func (cc *CurrentContext) NewSubstation(creatorAddress string, owner *PlayerCache, allocation *AllocationCache) (*SubstationCache, error) {
    var substation types.Substation
    substationId := GetObjectID(types.ObjectType_substation, k.GetNextSubstationId(ctx))

    substation.Id       = substationId
    substation.Owner    = owner.ID()
    substation.Creator  = creatorAddress

    // Start to put the pieces together
    cc.substations[substationId] := &SubstationCache{
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
    cc.PermissionAdd(permissionId, types.PermissionAll)

    return cc.substations[substationId], nil
}
