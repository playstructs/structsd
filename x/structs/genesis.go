package structs

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/keeper"
	"structs/x/structs/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {

    /*

	// Set all the reactor
	for _, elem := range genState.ReactorList {
		k.SetReactor(ctx, elem)
		k.SetReactorValidatorBytes(ctx, elem.Id, elem.Validator)
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
	//k.SetAllocationCount(ctx, genState.AllocationCount)

	// Set all the guild
	for _, elem := range genState.GuildList {
		k.SetGuild(ctx, elem)
	}

	// Set guild count
	k.SetGuildCount(ctx, genState.GuildCount)
	// Set all the player
	for _, elem := range genState.PlayerList {
		k.SetPlayer(ctx, elem)
	}

	// Set player count
	//k.SetPlayerCount(ctx, genState.PlayerCount)
	// Set all the address
	for _, elem := range genState.AddressList {
		k.SetAddress(ctx, elem)
	}

	// Set address count
	k.SetAddressCount(ctx, genState.AddressCount)

	*/
	//custom genesis code will be needed due to some race conditions taking place

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
	genesis.ReactorList = k.GetAllReactor(ctx, false)
	genesis.ReactorCount = k.GetReactorCount(ctx)
	genesis.SubstationList = k.GetAllSubstation(ctx, false)
	genesis.SubstationCount = k.GetSubstationCount(ctx)
	genesis.AllocationList = k.GetAllAllocation(ctx, true)
	genesis.GuildList = k.GetAllGuild(ctx)
	genesis.GuildCount = k.GetGuildCount(ctx)
	genesis.PlayerList = k.GetAllPlayer(ctx, true)
	genesis.PlayerCount = k.GetPlayerCount(ctx)

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
