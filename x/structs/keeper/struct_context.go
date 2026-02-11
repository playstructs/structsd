package keeper

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

            BlockStartBuildAttributeId:     GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartBuild, structId),
            BlockStartOreMineAttributeId:   GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreMine, structId),
            BlockStartOreRefineAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreRefine, structId),

            ProtectedStructIndexAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_protectedStructIndex, structId),
        }

	return cc.structs[structId]
}

func (cc *CurrentContext) InitialCommandShipStruct(fleet *FleetCache) *StructCache {

	structType, _ := cc.GetStructType()
	structure := types.CreateBaseStruct(types.CommandStructTypeId, fleet.GetOwner().GetPrimaryAddress(), fleet.GetOwner().GetPlayerId(), structType.GetStructType().Category, types.Ambit_land)
	structure.LocationId = fleet.GetFleetId()

    // Create the struct
    structure.Index := cc.k.GetStructCount(cc.ctx)
    cc.k.SetStructCount(cc.ctx, structure.Index+1)

    structId := GetObjectID(types.ObjectType_struct, structure.Index)
    structure.Id = structId

	fleet.GetOwner().BuildQuantityIncrement(types.CommandStructTypeId)

	var structStatus types.StructState
	if fleet.GetOwner().CanSupportLoadAddition(structType.GetPassiveDraw()) {
		fleet.GetOwner().StructsLoadIncrement(structType.GetPassiveDraw())
		structStatus = types.StructState(types.StructStateMaterialized | types.StructStateBuilt | types.StructStateOnline)
	} else {
		structStatus = types.StructState(types.StructStateMaterialized | types.StructStateBuilt)
	}

	// Start to put the pieces together
	cc.structs[structId] := &StructCache{
		StructId: structId,
		CC: cc,

		Changed: true,

		Structure:        structure,
		StructureLoaded:  true,

        HealthAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_health, structId),
        StatusAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_status, structId),

        BlockStartBuildAttributeId:     GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartBuild, structId),
        BlockStartOreMineAttributeId:   GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreMine, structId),
        BlockStartOreRefineAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreRefine, structId),

        ProtectedStructIndexAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_protectedStructIndex, structId),
	}

	cc.SetStructAttribute(cc.structs[structId].HealthAttributeId, structType.GetMaxHealth())
	cc.SetStructAttribute(cc.structs[structId].StatusAttributeId, uint64(structStatus))

    // Set the Permissions
    permissionId := GetObjectPermissionIDBytes(structure.Id, structure.Owner)
    cc.PermissionAdd(permissionId, types.PermissionAll)


	return cc.structs[structId]
}


// Build this initial Struct Cache object
// This does no validation on the provided structId
func (cc *CurrentContext) InitiateStruct(creatorAddress string, owner *PlayerCache, structType *StructTypeCache, ambit types.Ambit, slot uint64) (*StructCache, error) {

	structure := types.CreateBaseStruct(structType.ID(), creatorAddress, owner.GetPlayerId(), structType.GetStructType().Category, ambit)

    structure.Index := cc.k.GetStructCount(cc.ctx)
    cc.k.SetStructCount(cc.ctx, structure.Index+1)

    structId := GetObjectID(types.ObjectType_struct, structure.Index)
    structure.Id = structId

	switch structType.GetStructType().Category {
	case types.ObjectType_planet:
		err := owner.GetPlanet().BuildInitiateReadiness(&structure, structType, ambit, slot)
		if err != nil {
			return StructCache{}, err
		}

		structure.LocationId = owner.GetPlanetId()
		structure.Slot = slot
	case types.ObjectType_fleet:
		err := owner.GetFleet().BuildInitiateReadiness(&structure, structType, ambit, slot)
		if err != nil {
			return StructCache{}, err
		}

		if structType.GetStructType().Type != types.CommandStruct {
			structure.Slot = slot
		}

		structure.LocationId = owner.GetFleetId()
	default:
		return &StructCache{}, types.NewStructBuildError(structType.GetId(), "", "", "type_unsupported")
	}

	owner.StructsLoadIncrement(structType.GetStructType().GetBuildDraw())
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
			owner.GetFleet().SetCommandStruct(structure)
		} else {
			err := owner.GetFleet().SetSlot(structure)
			if err != nil {
				return &StructCache{}, err
			}
		}
	}

	// Start to put the pieces together
	cc.structs[structId] := &StructCache{
		StructId: structId,
		CC: cc,

		Changed: true,

		Structure:        structure,
		StructureLoaded:  true,

        HealthAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_health, structId),
        StatusAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_status, structId),

        BlockStartBuildAttributeId:     GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartBuild, structId),
        BlockStartOreMineAttributeId:   GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreMine, structId),
        BlockStartOreRefineAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_blockStartOreRefine, structId),

        ProtectedStructIndexAttributeId: GetStructAttributeIDByObjectId(types.StructAttributeType_protectedStructIndex, structId),
	}

	cc.SetStructAttribute(cc.structs[structId].HealthAttributeId, structType.GetMaxHealth())
	cc.SetStructAttribute(cc.structs[structId].StatusAttributeId, uint64(types.StructState(types.StructStateMaterialized),))

    ctxSDK := sdk.UnwrapSDKContext(cc.ctx)
    cc.SetStructAttribute(cc.structs[structId].BlockStartBuildAttributeId, uint64(ctxSDK.BlockHeight()))

    // Set the Permissions
    permissionId := GetObjectPermissionIDBytes(structure.Id, structure.Owner)
    cc.PermissionAdd(permissionId, types.PermissionAll)

	return cc.structs[structId], nil
}