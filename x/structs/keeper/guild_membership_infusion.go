package keeper

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"structs/x/structs/types"
)

// Join minima count owned, still-delegated fuel, not unreconciled pre-slash
// records. This read cannot credit a deposit or change grid accounting.
// Validator health affects power, not the existing
// membership policy's stake minimum. Unbonding balances do not count as fuel.
func (cc *CurrentContext) guildJoinInfusionFuel(infusion types.Infusion, reactor types.Reactor, address sdk.AccAddress) (uint64, error) {
	if infusion.Fuel == 0 {
		return 0, nil
	}
	validatorAddress, err := sdk.ValAddressFromBech32(reactor.Validator)
	if err != nil {
		return 0, types.NewAddressValidationError(reactor.Validator, "invalid_validator")
	}
	validator, err := cc.k.stakingKeeper.GetValidator(cc.ctx, validatorAddress)
	if errors.Is(err, stakingtypes.ErrNoValidatorFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	delegation, err := cc.k.stakingKeeper.GetDelegation(cc.ctx, address, validatorAddress)
	if errors.Is(err, stakingtypes.ErrNoDelegation) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return min(infusion.Fuel, delegationShareValue(delegation.Shares, validator).Uint64()), nil
}
