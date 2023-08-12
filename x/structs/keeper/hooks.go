package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/slashing/types"
)

var _ types.StakingHooks = Hooks{}

// Hooks wrapper struct for slashing keeper
type Hooks struct {
	k Keeper
}

// Return the slashing hooks
func (k Keeper) Hooks() Hooks {
	return Hooks{k}
}

// AfterValidatorBonded updates the signing info start height or create a new signing info
func (h Hooks) AfterValidatorBonded(ctx sdk.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {

	return nil
}

// AfterValidatorRemoved deletes the address-pubkey relation when a validator is removed,
func (h Hooks) AfterValidatorRemoved(ctx sdk.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {

	return nil
}

// AfterValidatorCreated adds the address-pubkey relation when a validator is created.
func (h Hooks) AfterValidatorCreated(ctx sdk.Context, valAddr sdk.ValAddress) error {

    // Setup the Reactor object once a validator comes online
	h.k.ReactorInitialize(ctx, valAddr)

	return nil
}

func (h Hooks) AfterValidatorBeginUnbonding(_ sdk.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {

	return nil
}

func (h Hooks) BeforeValidatorModified(ctx sdk.Context, valAddr sdk.ValAddress) error {
	h.k.ReactorUpdateFromValidator(ctx, valAddr)
	return nil
}

func (h Hooks) BeforeDelegationCreated(_ sdk.Context, _ sdk.AccAddress, _ sdk.ValAddress) error {
    //_ = h.k.ReactorUpdatePlayerAllocation(ctx, playerAddress, valAddr)
	return nil
}

func (h Hooks) BeforeDelegationSharesModified(_ sdk.Context, _ sdk.AccAddress, _ sdk.ValAddress) error {
	//_ = h.k.ReactorUpdatePlayerAllocation(ctx, playerAddress, valAddr)
	return nil
}

func (h Hooks) BeforeDelegationRemoved(_ sdk.Context, _ sdk.AccAddress, _ sdk.ValAddress) error {

	return nil
}

/* This doesn't actually exist yet, but I'd like it to */
func (h Hooks) AfterDelegationRemoved(ctx sdk.Context, playerAddress sdk.AccAddress, valAddr sdk.ValAddress) error {
	_ = h.k.ReactorUpdatePlayerAllocation(ctx, playerAddress, valAddr)
	return nil
}

func (h Hooks) AfterDelegationModified(ctx sdk.Context, playerAddress sdk.AccAddress, valAddr sdk.ValAddress) error {
	_ = h.k.ReactorUpdatePlayerAllocation(ctx, playerAddress, valAddr)

	return nil
}

func (h Hooks) BeforeValidatorSlashed(_ sdk.Context, _ sdk.ValAddress, _ sdk.Dec) error {
	return nil
}

func (h Hooks) AfterUnbondingInitiated(ctx sdk.Context, unbondingId uint64) error {
    h.k.ReactorRemoveInfusion(ctx, unbondingId)
	return nil
}
