package keeper

import (
	"errors"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"structs/x/structs/types"
)

// Genesis associations have been validated but their indexes may not yet be
// committed. Resolve the supplied asset owner directly, never through the
// signing-address index: protocol vaults intentionally have no signing identity.
func (cc *CurrentContext) loadGenesisReactorBacking(record types.Infusion, ownerID string) (validator stakingtypes.Validator, fuel uint64, defusing uint64, ubd stakingtypes.UnbondingDelegation, err error) {
	if record.AssetOwnerID() != ownerID {
		return validator, 0, 0, ubd, fmt.Errorf("reactor infusion owner does not match address association")
	}
	if _, err := cc.GetExistingAssetOwner(ownerID); err != nil {
		return validator, 0, 0, ubd, err
	}
	reactor := cc.GetReactor(record.DestinationId)
	if err := reactor.CheckReactor(); err != nil {
		return validator, 0, 0, ubd, err
	}
	validatorAddress, err := sdk.ValAddressFromBech32(reactor.GetReactor().Validator)
	if err != nil {
		return validator, 0, 0, ubd, err
	}
	address, err := sdk.AccAddressFromBech32(record.Address)
	if err != nil {
		return validator, 0, 0, ubd, err
	}
	validator, err = cc.k.stakingKeeper.GetValidator(cc.ctx, validatorAddress)
	if err != nil {
		return validator, 0, 0, ubd, err
	}
	delegation, err := cc.k.stakingKeeper.GetDelegation(cc.ctx, address, validatorAddress)
	if err == nil {
		fuel = delegationShareValue(delegation.Shares, validator).Uint64()
	} else if !errors.Is(err, stakingtypes.ErrNoDelegation) {
		return validator, 0, 0, ubd, err
	}
	unbondingBalance := math.ZeroInt()
	ubd, err = cc.k.stakingKeeper.GetUnbondingDelegation(cc.ctx, address, validatorAddress)
	if err == nil {
		for _, entry := range ubd.Entries {
			unbondingBalance = unbondingBalance.Add(entry.Balance)
		}
	} else if !errors.Is(err, stakingtypes.ErrNoUnbondingDelegation) {
		return validator, 0, 0, ubd, err
	}
	if unbondingBalance.IsNegative() || !unbondingBalance.IsUint64() {
		return validator, 0, 0, ubd, fmt.Errorf("invalid reactor infusion unbonding balance")
	}
	return validator, fuel, unbondingBalance.Uint64(), ubd, nil
}

// stageGenesisReactorInfusion installs reconstructed accounting once and
// schedules every native UBD entry, including positions with no delegation left.
func (cc *CurrentContext) stageGenesisReactorInfusion(record types.Infusion, ubd stakingtypes.UnbondingDelegation) {
	cache := cc.GetInfusion(record.DestinationId, record.Address)
	oldFuel, oldDestination, oldOwner := uint64(0), uint64(0), uint64(0)
	if cache.CheckInfusion() == nil {
		oldFuel = cache.Infusion.Fuel
		_, oldDestination, oldOwner = cache.Infusion.GetPowerDistribution()
	}
	cache.Infusion = record
	cache.InfusionLoaded = true
	cache.indexCachedOwner()
	cache.Changed = true
	cache.applyGridDeltas(oldFuel, oldDestination, oldOwner)
	for _, entry := range ubd.Entries {
		cc.k.EnqueueInfusionMaturitySweep(cc.ctx, entry.CompletionTime, cache.ID())
	}
}

// GenesisImportReactorInfusion rebuilds flat power from native staking, including
// vault-owned and unbonding-only positions. Exported power is never trusted.
func (cc *CurrentContext) GenesisImportReactorInfusion(record types.Infusion, ownerID string) error {
	if err := record.ValidateAssetOwner(); err != nil { return err }
	validator, fuel, defusing, ubd, err := cc.loadGenesisReactorBacking(record, ownerID)
	if err != nil { return err }
	record.Fuel, record.Defusing = fuel, defusing
	record.Ratio = reactorEnergyRatio(validator, nil)
	record.Commission = cc.GetReactor(record.DestinationId).GetReactor().DefaultCommission
	record.Recalculate()
	cc.stageGenesisReactorInfusion(record, ubd)
	return nil
}
