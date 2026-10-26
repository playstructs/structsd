package keeper

import (
	"structs/x/structs/types"
    sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetStruct returns a StructCache, loading from store if not already cached.
// Returns the same instance if called multiple times with the same ID.
func (cc *CurrentContext) GetStruct(structId string) *StructCache {
	if cache, exists := cc.structs[structId]; exists {
		return cache
	}

	cc.structs[structId] = &StructCache{
            StructId: structId,
            CC: cc,

            Changed: false,

            HealthAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_health, structId),
            StatusAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_status, structId),

            BlockStartBuildAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartBuild, structId),

            ProtectedStructIndexAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_protectedStructIndex, structId),

            ReadyAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_ready, structId),
        }

	return cc.structs[structId]
}

func (cc *CurrentContext) GenesisImportStruct(
	s types.Struct,
	importedStatus uint64,
) {
	if importedStatus&uint64(types.StructStateMaterialized) == 0 {
		return
	}

	cache := cc.GetStruct(s.Id)
	cache.Structure = s
	cache.StructureLoaded = true
	cache.Changed = true

	cc.SetStructAttribute(cache.StatusAttributeId, importedStatus)

	if importedStatus&uint64(types.StructStateDestroyed) != 0 {
		return
	}

	structTypeCache, _ := cc.GetStructType(s.Type)
	structType := structTypeCache.GetStructType()

	cc.SetStructAttribute(cache.HealthAttributeId, structType.MaxHealth)

	typeCountAttrId := GetStructAttributeIDByObjectIdAndSubIndex(
		types.StructAttributeType_typeCount, s.Owner, s.Type)
	cc.SetStructAttributeIncrement(typeCountAttrId, 1)

	isOnline := importedStatus&uint64(types.StructStateOnline) != 0
	isBuilt := importedStatus&uint64(types.StructStateBuilt) != 0

	if !isBuilt {
		/* An unfinished build restarts its clock at the genesis height.
		 *
		 * BlockStartBuild is the age the build proof is priced from:
		 * StructBuildComplete computes currentHeight - BlockStartBuild and
		 * CalculateDifficulty falls with that age, clamping at 1. Storing zero
		 * therefore did not mean "no progress", it meant "as old as the chain" -
		 * so on a height-preserving restart of a mature chain, every build in
		 * flight arrived with its puzzle already decayed to the minimum. Unlike
		 * the raid clock, nothing downstream refuses a zero start.
		 *
		 * The genesis height is the only value that is safe in both directions.
		 * Carrying the exported start would be more faithful on a
		 * height-preserving restart and unusable after a zero-height one, where
		 * it sits ahead of the new height and StructBuildComplete refuses the
		 * proof outright - permanently, since nothing moves it. Resetting costs
		 * the builder their accumulated age, which is the direction that asks for
		 * more work rather than less.
		 *
		 * This is what the online case already did; only the offline one wrote
		 * zero, which is why the two arms have collapsed into one.
		 */
		ctxSDK := sdk.UnwrapSDKContext(cc.ctx)
		cc.SetStructAttribute(cache.BlockStartBuildAttributeId, uint64(ctxSDK.BlockHeight()))

		/* Restore the build reservation.
		 *
		 * InitiateStruct charges BuildDraw against the owner's load the moment a
		 * build starts, and only StructBuildComplete or DestroyAndCommit gives it
		 * back. The import rebuilds a player's load from scratch -
		 * GenesisImportPlayer resets it to PlayerPassiveDraw and structsLoad is
		 * deliberately not importable, because it is derived - so an unbuilt
		 * struct has to put its own reservation back or the load simply is not
		 * there.
		 *
		 * Missing it is not only under-counting. StructBuildComplete decrements
		 * BuildDraw unconditionally when the build finishes, and the decrement
		 * clamps at zero, so completing an imported build would eat load
		 * belonging to the player's passive draw and their other structs.
		 *
		 * A destroyed struct never reaches here: the branch above returns first,
		 * which is right, because destruction already released this.
		 */
		cache.GetOwner().StructsLoadIncrement(structType.BuildDraw)
	}

	if isOnline {
		cache.GoOnline()
	}
}

func (cc *CurrentContext) GetAllStructDefender(protectedStructId string) (defenders []*StructCache) {
    defenderList := cc.k.GetAllStructDefender(cc.ctx, protectedStructId)
    for _, defenderId := range defenderList {
        structure := cc.GetStruct(defenderId)
        if structure.CheckStruct() == nil {
            defenders = append(defenders, structure)
        }
    }
    return
}

func (cc *CurrentContext) InitialCommandShipStruct(fleet *FleetCache) *StructCache {

	structType, _ := cc.GetStructType(types.CommandStructTypeId)
	structure := types.CreateBaseStruct(types.CommandStructTypeId, fleet.GetOwner().GetPrimaryAddress(), fleet.GetOwner().GetPlayerId(), structType.GetStructType().Category, types.Ambit_land)
	structure.LocationId = fleet.GetFleetId()

    // Create the struct
    structure.Index = cc.k.GetStructCount(cc.ctx)
    cc.k.SetStructCount(cc.ctx, structure.Index+1)

    structId := GetObjectID(types.ObjectType_struct, structure.Index)
    structure.Id = structId

	fleet.GetOwner().BuildQuantityIncrement(types.CommandStructTypeId)

	var structStatus types.StructState
	if fleet.GetOwner().CanSupportLoadAddition(structType.GetStructType().PassiveDraw) {
		fleet.GetOwner().StructsLoadIncrement(structType.GetStructType().PassiveDraw)
		structStatus = types.StructState(types.StructStateMaterialized | types.StructStateBuilt | types.StructStateOnline)
	} else {
		structStatus = types.StructState(types.StructStateMaterialized | types.StructStateBuilt)
	}

	// Start to put the pieces together
	cc.structs[structId] = &StructCache{
		StructId: structId,
		CC: cc,

		Changed: true,

		Structure:        structure,
		StructureLoaded:  true,

        HealthAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_health, structId),
        StatusAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_status, structId),

        BlockStartBuildAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartBuild, structId),

        ProtectedStructIndexAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_protectedStructIndex, structId),

	    ReadyAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_ready, structId),
	}

	cc.SetStructAttribute(cc.structs[structId].HealthAttributeId, structType.GetStructType().MaxHealth)
	cc.SetStructAttribute(cc.structs[structId].StatusAttributeId, uint64(structStatus))

	return cc.structs[structId]
}


// Build this initial Struct Cache object
// This does no validation on the provided structId
func (cc *CurrentContext) InitiateStruct(creatorAddress string, owner *PlayerCache, structType *StructTypeCache, ambit types.Ambit, slot uint64) (*StructCache, error) {

	structure := types.CreateBaseStruct(structType.ID(), creatorAddress, owner.GetPlayerId(), structType.GetStructType().Category, ambit)

    structure.Index = cc.k.GetStructCount(cc.ctx)
    cc.k.SetStructCount(cc.ctx, structure.Index+1)

    structId := GetObjectID(types.ObjectType_struct, structure.Index)
    structure.Id = structId

	switch structType.GetStructType().Category {
	case types.ObjectType_planet:
		err := owner.GetPlanet().BuildInitiateReadiness(&structure, structType, ambit, slot)
		if err != nil {
			return &StructCache{}, err
		}

		structure.LocationId = owner.GetPlanetId()
		structure.Slot = slot
	case types.ObjectType_fleet:
		err := owner.GetFleet().BuildInitiateReadiness(&structure, structType, ambit, slot)
		if err != nil {
			return &StructCache{}, err
		}

		if structType.GetStructType().Type != types.CommandStruct {
			structure.Slot = slot
		}

		structure.LocationId = owner.GetFleetId()
	default:
		return &StructCache{}, types.NewStructBuildError(structType.ID(), "", "", "type_unsupported")
	}

	owner.StructsLoadIncrement(structType.GetStructType().BuildDraw)
	owner.BuildQuantityIncrement(structType.ID())

	switch structType.GetStructType().Category {
	case types.ObjectType_planet:
		// Update the cross reference on the planet
		cc.k.logger.Info("Struct Set Slot", "slot", structure.Slot, "planetId", owner.GetPlanet().GetPlanetId())
		err := owner.GetPlanet().SetSlot(structure)
		if err != nil {
			return &StructCache{}, err
		}

	case types.ObjectType_fleet:
		// Update the cross reference on the planet
		if structType.GetStructType().Type == types.CommandStruct {
			owner.GetFleet().SetCommandStruct(structId)
		} else {
			err := owner.GetFleet().SetSlot(structure)
			if err != nil {
				return &StructCache{}, err
			}
		}
	}

	// Start to put the pieces together
	cc.structs[structId] = &StructCache{
		StructId: structId,
		CC: cc,

		Changed: true,

		Structure:        structure,
		StructureLoaded:  true,

        HealthAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_health, structId),
        StatusAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_status, structId),

        BlockStartBuildAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartBuild, structId),

        ProtectedStructIndexAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_protectedStructIndex, structId),

	    ReadyAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_ready, structId),
	}

	cc.SetStructAttribute(cc.structs[structId].HealthAttributeId, structType.GetStructType().MaxHealth)
	cc.SetStructAttribute(cc.structs[structId].StatusAttributeId, uint64(types.StructState(types.StructStateMaterialized)))

    ctxSDK := sdk.UnwrapSDKContext(cc.ctx)
    cc.SetStructAttribute(cc.structs[structId].BlockStartBuildAttributeId, uint64(ctxSDK.BlockHeight()))

	return cc.structs[structId], nil
}