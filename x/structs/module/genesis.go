package structs

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

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

    for _, allocation := range genState.AllocationList {
        k.ImportAllocation(ctx, allocation)
        k.SetAllocationSourceIndex(ctx, allocation.SourceObjectId, allocation.Id)
        k.SetAllocationDestinationIndex(ctx, allocation.DestinationId, allocation.Id)

        if allocation.Type == types.AllocationType_automated {
        	k.SetAutoResizeAllocationSource(ctx, allocation.Id, allocation.SourceObjectId)
        }
    }

    for _, elem := range genState.InfusionList {
        k.SetInfusion(ctx, elem)
    }

    k.SetGuildCount(ctx, genState.GuildCount + k.GetGuildCount(ctx))
    for _, elem := range genState.GuildList {
        k.SetGuild(ctx, elem)
    }

    k.SetPlanetCount(ctx, genState.PlanetCount + k.GetPlanetCount(ctx))
    for _, elem := range genState.PlanetList {
        k.SetPlanet(ctx, elem)
    }

    // Planet attributes
    for _, elem := range genState.PlanetAttributeList {
        k.SetPlanetAttribute(ctx, elem.AttributeId, elem.Value)
    }



    k.SetPlayerCount(ctx, genState.PlayerCount + k.GetPlayerCount(ctx))
    for _, elem := range genState.PlayerList {
        k.SetPlayer(ctx, elem)
    }
    k.SetAllHaltedPlayerId(ctx, genState.PlayerHalted)

    k.SetReactorCount(ctx, genState.ReactorCount + k.GetReactorCount(ctx))
    for _, elem := range genState.ReactorList {
        k.SetReactor(ctx, elem)
    }

    k.SetStructCount(ctx, genState.StructCount + k.GetStructCount(ctx))
    for _, elem := range genState.StructList {
        k.SetStruct(ctx, elem)
    }
    // Struct attributes
    // TODO Update block based values to 0
    for _, elem := range genState.StructAttributeList {
    	k.SetStructAttribute(ctx, elem.AttributeId, elem.Value)
    }

    // Struct defenders (after structs exist)
    for _, elem := range genState.StructDefenderList {
        protected, found := k.GetStruct(ctx, elem.ProtectedStructId)
        if !found {
            continue
        }
        k.SetStructDefender(ctx, elem.ProtectedStructId, protected.Index, elem.DefendingStructId)
    }

    k.SetSubstationCount(ctx, genState.SubstationCount + k.GetSubstationCount(ctx))
    for _, elem := range genState.SubstationList {
        k.SetSubstation(ctx, elem)
    }

    for _, elem := range genState.GridList {
        k.SetGridAttribute(ctx, elem.AttributeId, elem.Value)
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
        k.SetAgreementProviderIndex(ctx, agreement.ProviderId, agreement.Id)
        k.SetAgreementExpirationIndex(ctx, agreement.EndBlock, agreement.Id)
    }

    // Fleet
    for _, elem := range genState.FleetList {
        k.SetFleet(ctx, elem)
    }

}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.PortId = k.GetPort(ctx)

	genesis.AddressList    = k.GetAllAddressExport(ctx)

	genesis.AgreementList = k.GetAllAgreement(ctx)

	genesis.AllocationList = k.GetAllAllocation(ctx)

	genesis.InfusionList = k.GetAllInfusion(ctx)

    genesis.FleetList = k.GetAllFleet(ctx)

	genesis.GuildList = k.GetAllGuild(ctx)
	genesis.GuildCount = k.GetGuildCount(ctx)

	genesis.PlanetList = k.GetAllPlanet(ctx)
	genesis.PlanetCount = k.GetPlanetCount(ctx)
    genesis.PlanetAttributeList = k.GetAllPlanetAttributeExport(ctx)

	genesis.PlayerList = k.GetAllPlayer(ctx)
	genesis.PlayerCount = k.GetPlayerCount(ctx)
	genesis.PlayerHalted = k.GetAllHaltedPlayerId(ctx)

	genesis.ProviderList = k.GetAllProvider(ctx)
	genesis.ProviderCount = k.GetProviderCount(ctx)
    genesis.ProviderGuildAccessList = k.GetAllProviderGuildAccessExport(ctx)

	genesis.ReactorList = k.GetAllReactor(ctx)
	genesis.ReactorCount = k.GetReactorCount(ctx)

	genesis.StructList = k.GetAllStruct(ctx)
    genesis.StructCount = k.GetStructCount(ctx)
    genesis.StructAttributeList = k.GetAllStructAttributeExport(ctx)
    genesis.StructDefenderList = k.GetAllStructDefenderExport(ctx)

	genesis.SubstationList = k.GetAllSubstation(ctx)
	genesis.SubstationCount = k.GetSubstationCount(ctx)

	genesis.GridList        = k.GetAllGridExport(ctx)
	genesis.PermissionList  = k.GetAllPermissionExport(ctx)

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}


