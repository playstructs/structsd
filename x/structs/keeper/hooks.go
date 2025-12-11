package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"context"

	"cosmossdk.io/math"

	"structs/x/structs/types"
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
func (h Hooks) AfterValidatorBonded(ctx context.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {

	return nil
}

// AfterValidatorRemoved deletes the address-pubkey relation when a validator is removed,
func (h Hooks) AfterValidatorRemoved(ctx context.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {

	return nil
}

// AfterValidatorCreated adds the address-pubkey relation when a validator is created.
func (h Hooks) AfterValidatorCreated(ctx context.Context, valAddr sdk.ValAddress) error {

	// Setup the Reactor object once a validator comes online
	h.k.ReactorInitialize(ctx, valAddr)

	return nil
}

func (h Hooks) AfterValidatorBeginUnbonding(_ context.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {

	return nil
}

func (h Hooks) BeforeValidatorModified(ctx context.Context, valAddr sdk.ValAddress) error {
	h.k.ReactorUpdateFromValidator(ctx, valAddr)
	return nil
}

func (h Hooks) BeforeDelegationCreated(_ context.Context, _ sdk.AccAddress, _ sdk.ValAddress) error {
	//_ = h.k.ReactorUpdatePlayerAllocation(ctx, playerAddress, valAddr)
	return nil
}

func (h Hooks) BeforeDelegationSharesModified(_ context.Context, _ sdk.AccAddress, _ sdk.ValAddress) error {
	//_ = h.k.ReactorUpdatePlayerAllocation(ctx, playerAddress, valAddr)
	return nil
}

func (h Hooks) BeforeDelegationRemoved(_ context.Context, _ sdk.AccAddress, _ sdk.ValAddress) error {

	return nil
}

func (h Hooks) AfterDelegationModified(ctx context.Context, playerAddress sdk.AccAddress, valAddr sdk.ValAddress) error {
	h.k.ReactorUpdatePlayerInfusion(ctx, playerAddress, valAddr)

	return nil
}

func (h Hooks) BeforeValidatorSlashed(ctx context.Context, valAddr sdk.ValAddress, fraction math.LegacyDec) error {
	h.k.ReactorUpdateInfusionsFromSlashing(ctx, valAddr, fraction)
	return nil
}

func (h Hooks) AfterUnbondingInitiated(ctx context.Context, unbondingId uint64) error {
	h.k.ReactorInfusionUnbonding(ctx, unbondingId)
	return nil
}
