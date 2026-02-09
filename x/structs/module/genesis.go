package structs

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

    "strconv"
    "strings"

	"structs/x/structs/keeper"
	"structs/x/structs/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	k.SetPort(ctx, genState.PortId)
	// Only try to bind to port if it is not already bound, since we may already own
	// port capability from capability InitGenesis
	if k.ShouldBound(ctx, genState.PortId) {
		// module binds to the port on InitChain
		// and claims the returned capability
		err := k.BindPort(ctx, genState.PortId)
		if err != nil {
			panic("could not claim port capability: " + err.Error())
		}
	}
	k.SetParams(ctx, genState.Params)

    var structTypeTop uint64
    for _, elem := range types.CreateStructTypeGenesis() {
        if (elem.Id > structTypeTop) { structTypeTop = elem.Id }
        k.SetStructType(ctx, elem)
    }
    k.SetStructTypeCount(ctx, structTypeTop + 1)

    for _, elem := range genState.AddressList {
        k.SetPlayerIndexForAddress(ctx, elem.Address, elem.PlayerIndex)
    }

    k.SetAllocationCount(ctx, genState.AllocationCount + k.GetAllocationCount(ctx))
    for _, allocation := range genState.AllocationList {
        k.ImportAllocation(ctx, allocation)

        if allocation.Type == types.AllocationType_automated {
        	k.SetAutoResizeAllocationSource(ctx, allocation.Id, allocation.SourceObjectId)
        }
    }

    for _, elem := range genState.InfusionList {
        k.SetInfusion(ctx, elem)
    }

    for _, elem := range genState.InfusionDestructionQueue {
        _ = k.AppendInfusionDestructionQueue(ctx, elem)
    }

    k.SetGuildCount(ctx, genState.GuildCount + k.GetGuildCount(ctx))
    for _, elem := range genState.GuildList {
        k.SetGuild(ctx, elem)
    }
    for _, elem := range genState.GuildMembershipApplicationList {
    	k.SetGuildMembershipApplication(ctx, elem)
    }

    k.SetPlanetCount(ctx, genState.PlanetCount + k.GetPlanetCount(ctx))
    for _, elem := range genState.PlanetList {
        k.SetPlanet(ctx, elem)
    }

    // Planet attributes
    for _, elem := range genState.PlanetAttributeList {
        value := elem.Value
        if isPlanetBlockHeightAttribute(elem.AttributeId) {
            value = 0
        }
        k.SetPlanetAttribute(ctx, elem.AttributeId, value)
    }

    k.SetPlayerCount(ctx, genState.PlayerCount + k.GetPlayerCount(ctx))
    for _, elem := range genState.PlayerList {
        k.SetPlayer(ctx, elem)
    }


    k.SetReactorCount(ctx, genState.ReactorCount + k.GetReactorCount(ctx))
    for _, elem := range genState.ReactorList {
        k.SetReactor(ctx, elem)
        k.SetReactorValidatorBytes(ctx, elem.Id, elem.RawAddress)
    }

    k.SetStructCount(ctx, genState.StructCount + k.GetStructCount(ctx))
    for _, elem := range genState.StructList {
        k.SetStruct(ctx, elem)
    }
    // Struct attributes
    for _, elem := range genState.StructAttributeList {
        value := elem.Value
        if isStructBlockHeightAttribute(elem.AttributeId) {
            value = 0
        }
        k.SetStructAttribute(ctx, elem.AttributeId, value)
    }

    // Struct defenders (after structs exist)
    for _, elem := range genState.StructDefenderList {
        protected, found := k.GetStruct(ctx, elem.ProtectedStructId)
        if !found {
            continue
        }
        k.SetStructDefender(ctx, elem.ProtectedStructId, protected.Index, elem.DefendingStructId)
    }

    for _, elem := range genState.StructDestructionQueue {
    	k.SetStructDestructionQueueAtHeight(ctx, elem.SweepHeight, elem.StructId)
    }

    k.SetSubstationCount(ctx, genState.SubstationCount + k.GetSubstationCount(ctx))
    for _, elem := range genState.SubstationList {
        k.SetSubstation(ctx, elem)
    }

    for _, elem := range genState.GridList {
        value := elem.Value
        if isGridBlockHeightAttribute(elem.AttributeId) {
            value = 0
        }
        k.SetGridAttribute(ctx, elem.AttributeId, value)
    }

    for _, elem := range genState.GridCascadeQueue {
        _ = k.AppendGridCascadeQueue(ctx, elem)
    }

    for _, elem := range genState.PermissionList {
        k.SetPermissionsByBytes(ctx, []byte(elem.PermissionId), types.Permission(elem.Value))
    }

    k.SetProviderCount(ctx, genState.ProviderCount + k.GetProviderCount(ctx))
    for _, elem := range genState.ProviderList {
        k.ImportProvider(ctx, elem)
    }
    // Provider guild access
    for _, elem := range genState.ProviderGuildAccessList {
        k.ProviderGrantGuild(ctx, elem.ProviderId, elem.GuildId)
    }

    for _, agreement := range genState.AgreementList {
        k.ImportAgreement(ctx, agreement)
        k.SetAgreementExpirationIndex(ctx, agreement.EndBlock, agreement.Id)
    }

    // Fleet
    for _, elem := range genState.FleetList {
        k.SetFleet(ctx, elem)
    }

    // Struct destruction queue (restore exact sweep height)


}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.PortId = k.GetPort(ctx)

	genesis.AddressList    = k.GetAllAddressExport(ctx)

	genesis.AgreementList = k.GetAllAgreement(ctx)

	genesis.AllocationList = k.GetAllAllocation(ctx)
	genesis.AllocationCount = k.GetAllocationCount(ctx)

	genesis.InfusionList = k.GetAllInfusion(ctx)
	genesis.InfusionDestructionQueue = k.GetInfusionDestructionQueueExport(ctx)

    genesis.FleetList = k.GetAllFleet(ctx)

	genesis.GuildList = k.GetAllGuild(ctx)
	genesis.GuildCount = k.GetGuildCount(ctx)
    genesis.GuildMembershipApplicationList = k.GetAllGuildMembershipApplicationExport(ctx)

	genesis.PlanetList = k.GetAllPlanet(ctx)
	genesis.PlanetCount = k.GetPlanetCount(ctx)
    genesis.PlanetAttributeList = k.GetAllPlanetAttributeExport(ctx)

	genesis.PlayerList = k.GetAllPlayer(ctx)
	genesis.PlayerCount = k.GetPlayerCount(ctx)

	genesis.ProviderList = k.GetAllProvider(ctx)
	genesis.ProviderCount = k.GetProviderCount(ctx)
    genesis.ProviderGuildAccessList = k.GetAllProviderGuildAccessExport(ctx)

	genesis.ReactorList = k.GetAllReactor(ctx)
	genesis.ReactorCount = k.GetReactorCount(ctx)

	genesis.StructList = k.GetAllStruct(ctx)
    genesis.StructCount = k.GetStructCount(ctx)
    genesis.StructAttributeList = k.GetAllStructAttributeExport(ctx)
    genesis.StructDefenderList = k.GetAllStructDefenderExport(ctx)
    genesis.StructDestructionQueue = k.GetStructDestructionQueueExport(ctx)

	genesis.SubstationList = k.GetAllSubstation(ctx)
	genesis.SubstationCount = k.GetSubstationCount(ctx)

	genesis.GridList        = k.GetAllGridExport(ctx)
	genesis.GridCascadeQueue = k.GetGridCascadeQueueExport(ctx)

	genesis.PermissionList  = k.GetAllPermissionExport(ctx)

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}


func parseAttributeTypeId(attributeId string) (uint64, bool) {
    parts := strings.SplitN(attributeId, "-", 2)
    if len(parts) == 0 {
        return 0, false
    }
    id, err := strconv.ParseUint(parts[0], 10, 64)
    if err != nil {
        return 0, false
    }
    return id, true
}

func isGridBlockHeightAttribute(attributeId string) bool {
    attrId, ok := parseAttributeTypeId(attributeId)
    if !ok {
        return false
    }
    switch types.GridAttributeType(attrId) {
    case types.GridAttributeType_lastAction,
        types.GridAttributeType_checkpointBlock:
        return true
    default:
        return false
    }
}

func isStructBlockHeightAttribute(attributeId string) bool {
    attrId, ok := parseAttributeTypeId(attributeId)
    if !ok {
        return false
    }
    switch types.StructAttributeType(attrId) {
    case types.StructAttributeType_blockStartBuild,
        types.StructAttributeType_blockStartOreMine,
        types.StructAttributeType_blockStartOreRefine:
        return true
    default:
        return false
    }
}

func isPlanetBlockHeightAttribute(attributeId string) bool {
    attrId, ok := parseAttributeTypeId(attributeId)
    if !ok {
        return false
    }
    switch types.PlanetAttributeType(attrId) {
    case types.PlanetAttributeType_blockStartRaid:
        return true
    default:
        return false
    }
}