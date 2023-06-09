package structs

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/keeper"
	"structs/x/structs/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set all the reactor
	for _, elem := range genState.ReactorList {
		k.SetReactor(ctx, elem)
	}

	// Set reactor count
	k.SetReactorCount(ctx, genState.ReactorCount)
	// Set all the substation
	for _, elem := range genState.SubstationList {
		k.SetSubstation(ctx, elem)
	}

	// Set substation count
	k.SetSubstationCount(ctx, genState.SubstationCount)
	// Set all the allocation
	for _, elem := range genState.AllocationList {
		k.SetAllocation(ctx, elem)
	}

	// Set allocation count
	k.SetAllocationCount(ctx, genState.AllocationCount)
	// Set all the allocationProposal
	for _, elem := range genState.AllocationProposalList {
		k.SetAllocationProposal(ctx, elem)
	}

	// Set allocationProposal count
	k.SetAllocationProposalCount(ctx, genState.AllocationProposalCount)

	// Set Allocation Status to Online where appropriate
    for _, elem := range genState.AllocationStatus {
        k.SetAllocationStatus(ctx, elem, types.AllocationStatus_Online)
    }

	// this line is used by starport scaffolding # genesis/module/init
	k.SetPort(ctx, genState.PortId)
	// Only try to bind to port if it is not already bound, since we may already own
	// port capability from capability InitGenesis
	if !k.IsBound(ctx, genState.PortId) {
		// module binds to the port on InitChain
		// and claims the returned capability
		err := k.BindPort(ctx, genState.PortId)
		if err != nil {
			panic("could not claim port capability: " + err.Error())
		}
	}
	k.SetParams(ctx, genState.Params)
}

// ExportGenesis returns the module's exported genesis
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.PortId = k.GetPort(ctx)
	genesis.ReactorList = k.GetAllReactor(ctx)
	genesis.ReactorCount = k.GetReactorCount(ctx)
	genesis.SubstationList = k.GetAllSubstation(ctx)
	genesis.SubstationCount = k.GetSubstationCount(ctx)
	genesis.AllocationList = k.GetAllAllocation(ctx)
	genesis.AllocationCount = k.GetAllocationCount(ctx)
	genesis.AllocationProposalList = k.GetAllAllocationProposal(ctx)
	genesis.AllocationProposalCount = k.GetAllocationProposalCount(ctx)
	genesis.AllocationStatus = k.GetAllOnlineAllocation(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
