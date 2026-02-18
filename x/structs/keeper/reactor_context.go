package keeper

import (
	"structs/x/structs/types"

	"cosmossdk.io/math"
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

func (cc *CurrentContext) GenesisImportReactorInfusions(reactor types.Reactor) {
	valAddr, err := sdk.ValAddressFromBech32(reactor.Validator)
	if err != nil {
		return
	}

	validator, err := cc.k.stakingKeeper.GetValidator(cc.ctx, valAddr)
	if err != nil {
		return
	}

	delegations, err := cc.k.stakingKeeper.GetValidatorDelegations(cc.ctx, valAddr)
	if err != nil {
		return
	}

	for _, delegation := range delegations {
		playerIndex := cc.GetPlayerIndexFromAddress(delegation.DelegatorAddress)
		if playerIndex == 0 {
			continue
		}

		player, err := cc.GetPlayerByIndex(playerIndex)
		if err != nil {
			continue
		}

		infusion := cc.UpsertInfusion(
			types.ObjectType_reactor, reactor.Id,
			delegation.DelegatorAddress, player.GetPlayerId())

		delegationShare := delegation.Shares.Quo(validator.DelegatorShares).Mul(
			math.LegacyNewDecFromInt(validator.Tokens)).RoundInt()

		infusion.SetRatio(types.ReactorFuelToEnergyConversion)
		infusion.SetFuelAndCommission(delegationShare.Uint64(), reactor.DefaultCommission)

		delegatorAddr, err := sdk.AccAddressFromBech32(delegation.DelegatorAddress)
		if err != nil {
			continue
		}
		unbondingDelegation, err := cc.k.stakingKeeper.GetUnbondingDelegation(cc.ctx, delegatorAddr, valAddr)
		if err == nil {
			defusingAmount := math.ZeroInt()
			for _, entry := range unbondingDelegation.Entries {
				defusingAmount = defusingAmount.Add(entry.Balance)
			}
			if defusingAmount.IsPositive() {
				infusion.SetDefusing(defusingAmount.Uint64())
			}
		}
	}
}

