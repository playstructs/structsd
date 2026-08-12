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

// AfterValidatorBonded restores reactor energy once a validator is back in the
// active set, which is the path a routine unjail takes: MsgUnjail clears the
// flag and staking rebonds the validator at the end of that block.
//
// An operator who unjails but stays below the active-set cutoff never reaches
// here, and recovers through the permissionless MsgReactorRestart instead.
//
// Restoring is a reconciliation against live staking state, so calling it for a
// validator that was never gated is harmless.
func (h Hooks) AfterValidatorBonded(ctx context.Context, _ sdk.ConsAddress, valAddr sdk.ValAddress) error {
	h.k.ReactorRestoreEnergy(ctx, valAddr)

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

// AfterValidatorBeginUnbonding is the primary jail gate for reactor energy.
//
// A jail sets the Jailed flag and drops the validator from the power index, and
// staking's EndBlock in that same block moves it out of the bonded set through
// bondedToUnbonding, which is what calls us here. Because the structs module is
// last in the EndBlocker order, the gate lands before our own GridCascade, so
// the capacity drop and its fallout resolve in the block the jail happened.
//
// Gating is conditional on the jail flag: leaving the active set voluntarily is
// not a reason to stop producing energy, so an unjailed validator that is merely
// unbonding keeps its reactor running.
func (h Hooks) AfterValidatorBeginUnbonding(ctx context.Context, _ sdk.ConsAddress, valAddr sdk.ValAddress) error {
	validator, err := h.k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil || validator.IsJailed() {
		// A validator that cannot be read is treated as unhealthy, matching the
		// fail-closed behaviour of reactorEnergyRatio.
		h.k.ReactorGateEnergy(ctx, valAddr)
	}

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

// BeforeDelegationRemoved clears the infusion behind a delegation that is being
// removed outright, which is the only signal staking gives for that case:
// Unbond takes the RemoveDelegation branch when shares reach zero and skips
// AfterDelegationModified entirely. A full redelegation is the path that made
// this matter, since it removes the source delegation and immediately grants
// the destination the same stake.
func (h Hooks) BeforeDelegationRemoved(ctx context.Context, playerAddress sdk.AccAddress, valAddr sdk.ValAddress) error {
	h.k.ReactorInfusionDelegationRemoved(ctx, playerAddress, valAddr)

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
