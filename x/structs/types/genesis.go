package types

import (
	"fmt"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
)

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		PortId:         PortID,
		ReactorList:    []Reactor{},
		SubstationList: []Substation{},
		AllocationList: []Allocation{},
		GuildList:      []Guild{},
		PlayerList:     []Player{},
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := host.PortIdentifierValidator(gs.PortId); err != nil {
		return err
	}
	// Check for duplicated ID in reactor
	reactorIdMap := make(map[string]bool)
	//reactorCount := gs.GetReactorCount()
	for _, elem := range gs.ReactorList {
		if _, ok := reactorIdMap[elem.Id]; ok {
			return fmt.Errorf("duplicated id for reactor")
		}
		//if elem.Id >= reactorCount {
		//	return fmt.Errorf("reactor id should be lower or equal than the last id")
		//}
		reactorIdMap[elem.Id] = true
	}
	// Check for duplicated ID in substation
	substationIdMap := make(map[string]bool)
	//substationCount := gs.GetSubstationCount()
	for _, elem := range gs.SubstationList {
		if _, ok := substationIdMap[elem.Id]; ok {
			return fmt.Errorf("duplicated id for substation")
		}
		//if elem.Id >= substationCount {
		//	return fmt.Errorf("substation id should be lower or equal than the last id")
		//}
		substationIdMap[elem.Id] = true
	}
	// Check for duplicated ID in allocation
	allocationIdMap := make(map[string]bool)
	//allocationCount := gs.GetAllocationCount()
	for _, elem := range gs.AllocationList {
		if _, ok := allocationIdMap[elem.Id]; ok {
			return fmt.Errorf("duplicated id for allocation")
		}
		//if elem.Id >= allocationCount {
		//	return fmt.Errorf("allocation id should be lower or equal than the last id")
		//}
		allocationIdMap[elem.Id] = true
	}

	// Check for duplicated ID in guild
	guildIdMap := make(map[string]bool)
	//guildCount := gs.GetGuildCount()
	for _, elem := range gs.GuildList {
		if _, ok := guildIdMap[elem.Id]; ok {
			return fmt.Errorf("duplicated id for guild")
		}
		//if elem.Id >= guildCount {
		//	return fmt.Errorf("guild id should be lower or equal than the last id")
		//}
		guildIdMap[elem.Id] = true
	}
	// Check for duplicated ID in player
	playerIdMap := make(map[string]bool)
	//playerCount := gs.GetPlayerCount()
	for _, elem := range gs.PlayerList {
		if _, ok := playerIdMap[elem.Id]; ok {
			return fmt.Errorf("duplicated id for player")
		}
		//if elem.Index >= playerCount {
		//	return fmt.Errorf("player id should be lower or equal than the last id")
		//}
		playerIdMap[elem.Id] = true
	}

	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
