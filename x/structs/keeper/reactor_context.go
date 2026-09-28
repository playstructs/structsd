package keeper

import (
	"structs/x/structs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetReactor returns a Reactor by ID, caching the result.
func (cc *CurrentContext) GetReactor(reactorId string) (*ReactorCache) {
	if cache, exists := cc.reactors[reactorId]; exists {
		return cache
	}

	cc.reactors[reactorId] = &ReactorCache{
	    ReactorId: reactorId,
	    CC: cc,
	    ReactorLoaded: false,
	    Changed: false,
	}

	return cc.reactors[reactorId]
}

func (cc *CurrentContext) GenesisImportReactor(reactor types.Reactor) {
	cache := cc.GetReactor(reactor.Id)
	cache.Reactor = reactor
	cache.ReactorLoaded = true
	cache.Changed = true

	cc.k.SetReactorValidatorBytes(cc.ctx, reactor.Id, reactor.RawAddress)
}

// Discover live positions absent from the exported infusion list. Imported
// associations are passed explicitly because vault indexes commit afterwards.
func (cc *CurrentContext) GenesisImportReactorInfusions(reactor types.Reactor, owners map[string]string) error {
	validator, err := sdk.ValAddressFromBech32(reactor.Validator)
	if err != nil { return err }
	delegations, err := cc.k.stakingKeeper.GetValidatorDelegations(cc.ctx, validator)
	if err != nil { return err }
	for _, delegation := range delegations {
		ownerID, found := owners[delegation.DelegatorAddress]
		if !found {
			// Genutil hooks can register a new operator before module genesis.
			index := cc.GetPlayerIndexFromAddress(delegation.DelegatorAddress)
			if index == 0 { continue }
			ownerID = GetObjectID(types.ObjectType_player, index)
		}
		key := types.BuildInfusionKey(reactor.Id, delegation.DelegatorAddress)
		if cached, imported := cc.infusions[key]; imported && cached.InfusionLoaded { continue }
		row := types.Infusion{DestinationId: reactor.Id, DestinationType: types.ObjectType_reactor,
			Address: delegation.DelegatorAddress, OwnerId: ownerID, Commission: reactor.DefaultCommission}
		if ObjectIdHasType(ownerID, types.ObjectType_player) { row.PlayerId = ownerID }
		if err := cc.GenesisImportReactorInfusion(row, ownerID); err != nil { return err }
	}
	return nil
}
