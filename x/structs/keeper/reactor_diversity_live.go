package keeper

import (
	"errors"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"structs/x/structs/types"
)

type liveOwnerReactorPortfolio struct {
	shares   map[string]uint64
	bonus    *big.Int
	complete bool
}

// Read current native delegations, not just known infusion rows: a failed hook
// may leave a higher balance, a zero row, or no row at all. Per-owner address
// and delegation inventories are separately bounded. A partial inventory cannot
// establish a nonlinear bonus and grants none. It does not remove base credit.
func (cc *CurrentContext) liveOwnerReactorPortfolio(owner string) liveOwnerReactorPortfolio {
	unknown := liveOwnerReactorPortfolio{}
	addresses, complete := cc.getOwnerStakeAddressesUpTo(owner, OwnerSupplyReadBudget)
	if !complete {
		return unknown
	}
	result := liveOwnerReactorPortfolio{shares: make(map[string]uint64), complete: true}
	byReactor := make(map[string]*big.Int)
	validators := make(map[string]stakingtypes.Validator)
	reactors := make(map[string]types.Reactor)
	remaining := OwnerSupplyReadBudget
	for _, address := range addresses {
		delegations, err := cc.k.stakingKeeper.GetDelegatorDelegations(cc.ctx, address, uint16(remaining+1))
		if err != nil || len(delegations) > remaining {
			return unknown
		}
		remaining -= len(delegations)
		for _, delegation := range delegations {
			if delegation.DelegatorAddress != address.String() {
				return unknown
			}
			validator, cached := validators[delegation.ValidatorAddress]
			reactor := reactors[delegation.ValidatorAddress]
			if !cached {
				valAddress, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
				if err != nil {
					return unknown
				}
				validator, err = cc.k.stakingKeeper.GetValidator(cc.ctx, valAddress)
				if err != nil && !errors.Is(err, stakingtypes.ErrNoValidatorFound) {
					return unknown
				}
				if err == nil {
					if key, found := cc.k.GetReactorBytesFromValidator(cc.ctx, valAddress); found {
						if row, found := cc.k.GetReactorByBytes(cc.ctx, key); found {
							c := cc.GetReactor(row.Id)
							if !c.Deleted && c.CheckReactor() == nil {
								reactor = c.GetReactor()
							}
						}
					}
				}
				validators[delegation.ValidatorAddress] = validator
				reactors[delegation.ValidatorAddress] = reactor
			}
			if reactor.Id == "" || !validator.IsBonded() || validator.IsJailed() {
				continue
			}
			if reactor.DefaultCommission.IsNil() || reactor.DefaultCommission.IsNegative() || reactor.DefaultCommission.GT(sdkmath.LegacyOneDec()) {
				return unknown
			}
			fuel := delegationShareValue(delegation.Shares, validator)
			if fuel.IsNegative() || !fuel.IsUint64() {
				return unknown
			}
			row := types.Infusion{Power: fuel.Uint64(), Commission: reactor.DefaultCommission}
			_, _, share := row.GetPowerDistribution()
			result.shares[types.BuildInfusionKey(reactor.Id, address.String())] = share
			if byReactor[reactor.Id] == nil {
				byReactor[reactor.Id] = new(big.Int)
			}
			byReactor[reactor.Id].Add(byReactor[reactor.Id], new(big.Int).SetUint64(share))
		}
	}
	sum, squares := new(big.Int), new(big.Int)
	for _, power := range byReactor {
		sum.Add(sum, power)
		squares.Add(squares, new(big.Int).Mul(power, power))
	}
	p := cc.k.GetParams(cc.ctx).WithReactorDefaults()
	bonus, err := types.CalculateReactorDiversityBonus(sum, squares, p.DiversityMaxBonusBps, p.DiversityTargetExtraReactors)
	if err != nil {
		return unknown
	}
	result.bonus = bonus
	return result
}
