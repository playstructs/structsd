package keeper

import (
	"context"
	"fmt"
	"math/big"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"structs/x/structs/types"
)

var reactorStakeCapActiveKey = []byte("Reactor/stakeCapActive")

func (k Keeper) ReactorStakeCapActive(ctx context.Context) bool {
	return runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)).Has(reactorStakeCapActiveKey)
}

// SetReactorStakeCapActive imports the genesis flag. Runtime activation only
// calls this with true: falling validator counts cannot reopen concentration.
func (k Keeper) SetReactorStakeCapActive(ctx context.Context, active bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	if active {
		store.Set(reactorStakeCapActiveKey, []byte{1})
	} else {
		store.Delete(reactorStakeCapActiveKey)
	}
}

func (k Keeper) EmitReactorStakeCap(ctx context.Context) error {
	p := k.GetParams(ctx).WithReactorDefaults()
	return sdk.UnwrapSDKContext(ctx).EventManager().EmitTypedEvent(&types.EventReactorStakeCap{
		Active: k.ReactorStakeCapActive(ctx), CapBps: p.ReactorStakeCapBps,
		MinValidators: p.ReactorStakeCapMinValidators,
	})
}

// EvaluateReactorStakeCap runs at genesis/upgrade and after staking's EndBlock,
// never on a delegation path. GetLastValidators reads the installed bonded set,
// bounded by staking's own MaxValidators, rather than all registered validators.
func (k Keeper) EvaluateReactorStakeCap(ctx context.Context) error {
	if k.ReactorStakeCapActive(ctx) {
		return nil
	}
	p := k.GetParams(ctx).WithReactorDefaults()
	if err := p.ValidateReactorParams(); err != nil {
		return err
	}
	validators, err := k.stakingKeeper.GetLastValidators(ctx)
	if err != nil {
		return err
	}
	if uint64(len(validators)) < p.ReactorStakeCapMinValidators {
		return nil
	}
	k.SetReactorStakeCapActive(ctx, true)
	return k.EmitReactorStakeCap(ctx)
}

func reactorStakeSnapshotKey(delegator sdk.AccAddress, validator sdk.ValAddress) []byte {
	return []byte("reactor/stake-before/" + delegator.String() + "/" + validator.String())
}

// A before/after pair shares the SDK transaction's transient store, including
// its rollback boundary. Each before overwrites a previous attempt; each after
// consumes its snapshot, and full delegation removal clears it without an after.
// No process-local map or unbounded historical record participates in consensus.
func (k Keeper) snapshotReactorStake(ctx context.Context, delegator sdk.AccAddress, validator sdk.ValAddress) error {
	if !k.ReactorStakeCapActive(ctx) {
		return nil
	}
	value, err := k.stakingKeeper.GetValidator(ctx, validator)
	if err != nil {
		return err
	}
	if value.Tokens.IsNil() || value.Tokens.IsNegative() {
		return fmt.Errorf("invalid validator token balance")
	}
	return k.transientStoreService.OpenTransientStore(ctx).Set(reactorStakeSnapshotKey(delegator, validator), []byte(value.Tokens.String()))
}

func (k Keeper) clearReactorStakeSnapshot(ctx context.Context, delegator sdk.AccAddress, validator sdk.ValAddress) error {
	return k.transientStoreService.OpenTransientStore(ctx).Delete(reactorStakeSnapshotKey(delegator, validator))
}

// enforceReactorStakeCap precedes any deferrable infusion error. SDK Delegate
// reaches this after validator tokens and pool transfers, including cancellation
// and redelegation. Partial Unbond reaches it before validator token reduction,
// so unchanged tokens must be allowed even above the cap. Slash is not an inflow.
func (k Keeper) enforceReactorStakeCap(ctx context.Context, delegator sdk.AccAddress, validator sdk.ValAddress) error {
	if !k.ReactorStakeCapActive(ctx) {
		return nil
	}
	store := k.transientStoreService.OpenTransientStore(ctx)
	key := reactorStakeSnapshotKey(delegator, validator)
	data, err := store.Get(key)
	if err != nil {
		return err
	}
	if err := store.Delete(key); err != nil {
		return err
	}
	before, ok := new(big.Int).SetString(string(data), 10)
	if !ok || before.Sign() < 0 {
		return fmt.Errorf("missing or invalid reactor stake snapshot")
	}
	value, err := k.stakingKeeper.GetValidator(ctx, validator)
	if err != nil {
		return err
	}
	if value.Tokens.IsNil() || value.Tokens.IsNegative() {
		return fmt.Errorf("invalid validator token balance")
	}
	tokens := value.Tokens.BigInt()
	if tokens.Cmp(before) <= 0 {
		return nil
	}
	p := k.GetParams(ctx).WithReactorDefaults()
	if err := p.ValidateReactorParams(); err != nil {
		return err
	}
	denominator, err := k.reactorStakeDenominator(ctx, value)
	if err != nil {
		return err
	}
	if !types.ReactorStakeWithinCap(tokens, denominator, p.ReactorStakeCapBps) {
		return &types.ReactorStakeCapError{ValidatorAddress: validator.String(),
			Tokens: tokens.String(), Denominator: denominator.String(), CapBps: p.ReactorStakeCapBps}
	}
	return nil
}

func (k Keeper) reactorStakeDenominator(ctx context.Context, validator stakingtypes.Validator) (*big.Int, error) {
	denom, err := k.stakingKeeper.BondDenom(ctx)
	if err != nil {
		return nil, err
	}
	balance := k.bankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(stakingtypes.BondedPoolName), denom).Amount
	if balance.IsNil() || balance.IsNegative() || validator.Tokens.IsNil() || validator.Tokens.IsNegative() {
		return nil, fmt.Errorf("invalid reactor stake denominator")
	}
	denominator := new(big.Int).Set(balance.BigInt())
	if !validator.IsBonded() {
		denominator.Add(denominator, validator.Tokens.BigInt())
	}
	return denominator, nil
}
