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

// AfterValidatorBonded queues power reconstruction after a validator enters the
// bonded set. Positions already zeroed regain power through the bounded worker;
// unvisited positions remain subject to live backing and eligibility checks.
func (h Hooks) AfterValidatorBonded(ctx context.Context, _ sdk.ConsAddress, valAddr sdk.ValAddress) error {
	h.k.QueueReactorRestoreEnergy(ctx, valAddr)
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
// Leaving the bonded set immediately makes live admission checks disregard the
// reactor. Stored power is cleaned in bounded batches that enqueue GridCascade; a
// large validator cannot force an unbounded scan in its staking transition hook.
func (h Hooks) AfterValidatorBeginUnbonding(ctx context.Context, _ sdk.ConsAddress, valAddr sdk.ValAddress) error {
	h.k.queueReactorEnergyReconcile(ctx, valAddr)
	return nil
}

func (h Hooks) BeforeValidatorModified(ctx context.Context, valAddr sdk.ValAddress) error {
	h.k.ReactorUpdateFromValidator(ctx, valAddr)
	return nil
}

func (h Hooks) BeforeDelegationCreated(ctx context.Context, playerAddress sdk.AccAddress, valAddr sdk.ValAddress) error {
	return h.k.snapshotReactorStake(ctx, playerAddress, valAddr)
}

func (h Hooks) BeforeDelegationSharesModified(ctx context.Context, playerAddress sdk.AccAddress, valAddr sdk.ValAddress) error {
	return h.k.snapshotReactorStake(ctx, playerAddress, valAddr)
}

// BeforeDelegationRemoved clears the infusion behind a delegation that is being
// removed outright, which is the only signal staking gives for that case:
// Unbond takes the RemoveDelegation branch when shares reach zero and skips
// AfterDelegationModified entirely. A full redelegation is the path that made
// this matter, since it removes the source delegation and immediately grants
// the destination the same stake.
func (h Hooks) BeforeDelegationRemoved(ctx context.Context, playerAddress sdk.AccAddress, valAddr sdk.ValAddress) error {
	if err := h.k.clearReactorStakeSnapshot(ctx, playerAddress, valAddr); err != nil { return err }
	h.k.ReactorInfusionDelegationRemoved(ctx, playerAddress, valAddr)

	return nil
}

func (h Hooks) AfterDelegationModified(ctx context.Context, playerAddress sdk.AccAddress, valAddr sdk.ValAddress) error {
	if err := h.k.enforceReactorStakeCap(ctx, playerAddress, valAddr); err != nil { return err }
	err := h.k.ReactorUpdatePlayerInfusion(ctx, playerAddress, valAddr)

    if err != nil {
        // The consensus hook cannot veto a slash. Persist the retry through the
        // existing bounded reactor repair queue, including a position already
        // passed by an in-flight repair snapshot.
        if reactorBytes, found := h.k.GetReactorBytesFromValidator(ctx, valAddr.Bytes()); found {
            if reactor, found := h.k.GetReactorByBytes(ctx, reactorBytes); found {
                h.k.EnqueueReactorInfusionRetry(ctx, reactor.Id, playerAddress)
            }
        }
        h.k.logger.Error("reactor infusion reconciliation deferred", "error", err)
    }
	return nil
}

func (h Hooks) BeforeValidatorSlashed(ctx context.Context, valAddr sdk.ValAddress, fraction math.LegacyDec) error {
	return h.k.ReactorUpdateInfusionsFromSlashing(ctx, valAddr, fraction)
}

func (h Hooks) AfterUnbondingInitiated(ctx context.Context, unbondingId uint64) error {
	h.k.ReactorInfusionUnbonding(ctx, unbondingId)
	return nil
}
