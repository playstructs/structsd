package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"structs/x/structs/types"
)

// OwnerSupplyReadBudget bounds the indexed infusion inventory for one owner.
const OwnerSupplyReadBudget = 2048

type readinessValidator struct {
	value stakingtypes.Validator
	err   error
}

// getUsablePrimaryCapacity discounts stale infusion contributions for readiness.
// It never writes capacity: deferred reconciliation still owes the full stored
// delta. Generator fuel remains stored while the generator is offline. This is
// not a proof that every unit of primary capacity has a canonical backing row.
func (cache *PlayerCache) getUsablePrimaryCapacity() uint64 {
	capacity, _ := cache.getUsablePrimaryCapacityWithin(OwnerSupplyReadBudget)
	return capacity
}

// Bound a single owner inventory; secondary power uses local grid accounting.
func (cache *PlayerCache) getUsablePrimaryCapacityWithin(limit int) (uint64, int) {
	return cache.CC.getUsableOwnerCapacityWithin(cache.ID(), limit)
}

// Guild and player allocations spend the same kind of infusion-backed capacity.
// Keep their projections together so guild vault power cannot bypass a gate
// merely because its owner id has a different prefix.
func (cc *CurrentContext) getUsableOwnerCapacity(ownerID string) uint64 {
	capacity, _ := cc.getUsableOwnerCapacityWithin(ownerID, OwnerSupplyReadBudget)
	return capacity
}

func (cc *CurrentContext) getUsableOwnerCapacityWithin(ownerID string, limit int) (uint64, int) {
	capacity, work, _ := cc.getUsableOwnerCapacityProofWithin(ownerID, limit)
	return capacity, work
}

// A zero admission allowance is not always a proven absence of power. Preserve
// incomplete/invalid inventory separately for callers that need that distinction.
func (cc *CurrentContext) getUsableOwnerCapacityProofWithin(ownerID string, limit int) (uint64, int, bool) {
	return cc.getUsableOwnerCapacityProofWithPortfolio(ownerID, limit, nil)
}

// A query can supply the portfolio it just read, avoiding a second native scan.
// Callers must not retain it across staking or ownership mutations.
func (cc *CurrentContext) getUsableOwnerCapacityProofWithPortfolio(ownerID string, limit int, supplied *liveOwnerReactorPortfolio) (uint64, int, bool) {
	capacity := cc.GetGridAttribute(GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, ownerID))
	if capacity == 0 {
		return capacity, 0, true
	}
	infusions, complete, work := cc.GetInfusionsByOwnerUpTo(ownerID, limit)
	if !complete {
		return 0, work, false
	}
	bonus := uint64(0)
	portfolio := liveOwnerReactorPortfolio{}
	if supplied != nil {
		portfolio = *supplied
	}
	if cc.k.ReactorDiversityReady(cc.ctx) {
		applied := cc.GetOwnerReactorDiversity(ownerID).AppliedBonus
		if applied > capacity {
			return 0, work, false
		}
		capacity -= applied
		if applied != 0 {
			if supplied == nil {
				portfolio = cc.liveOwnerReactorPortfolio(ownerID)
			}
			if portfolio.complete {
				bonus = applied
				if portfolio.bonus.IsUint64() {
					bonus = min(bonus, portfolio.bonus.Uint64())
				}
			}
		}
	}
	validators := make(map[string]readinessValidator)
	for _, infusion := range infusions {
		row := infusion.GetInfusion()
		// Validate before commission arithmetic: invalid splits can panic or
		// wrap the subtraction even if the source will subsequently be denied.
		if row.ValidateAssetOwner() != nil || row.Commission.IsNil() || row.Commission.IsNegative() || row.Commission.GT(math.LegacyOneDec()) {
			return 0, work, false
		}
		_, _, storedShare := row.GetPowerDistribution()
		usableShare := uint64(0)
		switch row.DestinationType {
		case types.ObjectType_reactor:
			if portfolio.complete {
				usableShare = min(storedShare, portfolio.shares[types.BuildInfusionKey(row.DestinationId, row.Address)])
			} else {
				row.Power = cc.usableReactorInfusionPower(row, validators)
				_, _, usableShare = row.GetPowerDistribution()
			}
		case types.ObjectType_struct:
			if row.ValidateGeneratorPower() != nil {
				return 0, work, false
			}
			// A zero owner contribution cannot provide readiness. Its row
			// still consumes inventory budget, but needs no lifecycle reads.
			if storedShare != 0 && !cc.generatorInfusionEligible(row) {
				row.Power = 0
			}
			_, _, usableShare = row.GetPowerDistribution()
		default:
			return 0, work, false
		}
		loss := storedShare - usableShare
		if loss >= capacity {
			capacity = 0
		} else {
			capacity -= loss
		}
	}
	return capacity + bonus, work, true
}

// usableReactorInfusionPower projects only reductions against live staking.
// The caller validates the infusion first. Deposits and restoration never
// gain power here; their ordinary hooks must credit and emit the stored change.
// Validators are shared only within this read. Never retain eligibility across
// calls: native staking can change while a CurrentContext remains in use.
func (cc *CurrentContext) usableReactorInfusionPower(row types.Infusion, validators map[string]readinessValidator) uint64 {
	if row.Power == 0 {
		return 0
	}
	address, err := sdk.AccAddressFromBech32(row.Address)
	if err != nil {
		return 0
	}
	owner, err := cc.GetAssetOwnerByAddress(address)
	if err != nil || owner.ID() != row.AssetOwnerID() {
		return 0
	}
	reactor := cc.GetReactor(row.DestinationId)
	if reactor.Deleted || reactor.CheckReactor() != nil {
		return 0
	}
	validatorAddress, err := sdk.ValAddressFromBech32(reactor.GetReactor().Validator)
	if err != nil {
		return 0
	}
	state, loaded := validators[reactor.GetReactor().Validator]
	if !loaded {
		state.value, state.err = cc.k.stakingKeeper.GetValidator(cc.ctx, validatorAddress)
		validators[reactor.GetReactor().Validator] = state
	}
	validator := state.value
	if state.err != nil || !validator.IsBonded() || validator.IsJailed() {
		return 0
	}
	delegation, err := cc.k.stakingKeeper.GetDelegation(cc.ctx, address, validatorAddress)
	if err != nil {
		return 0
	}
	power, err := projectReactorInfusionPower(row, validator, delegation)
	if err != nil {
		return 0
	}
	return power
}

// Project only power backed by both the stored record and live delegation.
// Restoration and deposits are credited through reconciliation, never a read.
func projectReactorInfusionPower(row types.Infusion, validator stakingtypes.Validator, delegation stakingtypes.Delegation) (uint64, error) {
	fuel := min(row.Fuel, delegationShareValue(delegation.Shares, validator).Uint64())
	return min(row.Power, types.CalculateInfusionPower(row.Ratio, fuel)), nil
}
