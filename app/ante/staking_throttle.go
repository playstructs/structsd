package ante

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// StakingThrottleDecorator enforces a per-address-per-block rate limit on free
// staking transactions. Each address may submit at most one free staking tx
// per block.
//
// It runs in every transaction phase (CheckTx, ReCheckTx, DeliverTx, and
// simulation):
//
//  1. Per-tx in-memory deduplication: a free staking tx may not bundle multiple
//     staking messages from the same address (this would let one tx burn
//     several "free" quotas in one shot).
//  2. Cross-tx transient-store check: at most one free staking tx per address
//     per block, enforced across all transactions admitted in the block.
//
// Both layers run in CheckTx for the same reason as ThrottleDecorator: txs
// that will fail at DeliverTx must also fail at CheckTx so the mempool can
// evict them via ReCheckTx instead of holding them indefinitely.
type StakingThrottleDecorator struct {
	keeper StructsAnteKeeper
}

func NewStakingThrottleDecorator(keeper StructsAnteKeeper) StakingThrottleDecorator {
	return StakingThrottleDecorator{keeper: keeper}
}

func (d StakingThrottleDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// Throttling follows the message, not the fee: see ContainsStakingMessage.
	// One staking operation per address per block is the rule on the free path,
	// and keying this off free-ness meant a tx pairing a staking message with
	// anything else reserved nothing and could be repeated without limit.
	if !ContainsStakingMessage(tx.GetMsgs()) {
		return next(ctx, tx, simulate)
	}

	hasStore := d.keeper.HasTransientStore()

	seen := make(map[string]struct{}, len(tx.GetMsgs()))
	for _, msg := range tx.GetMsgs() {
		typeURL := sdk.MsgTypeURL(msg)

		// A tx may pair staking with messages from other modules. Those are
		// gated by their own modules; skip them rather than rejecting the tx.
		if !IsStakingMessage(typeURL) {
			continue
		}

		// Reaching here means FreeStakingMessages lists a type URL that
		// StakingSignerExtractors does not, which is a maps.go inconsistency
		// rather than anything a transaction can cause.
		// TestStakingSignerExtractorsCoverFreeStakingMessages holds the pairing.
		extractor, ok := StakingSignerExtractors[typeURL]
		if !ok {
			return ctx, observeReject(ctx, "StakingThrottleDecorator",
				errorsmod.Wrapf(ErrUnknownStructsMessage, "unknown free staking message %s", typeURL))
		}
		addr := extractor(msg)
		if addr == "" {
			return ctx, observeReject(ctx, "StakingThrottleDecorator",
				errorsmod.Wrapf(ErrMissingCreator, "could not extract signer from %s", typeURL))
		}
		throttleKey := "staking/" + addr
		if _, dup := seen[throttleKey]; dup {
			return ctx, observeReject(ctx, "StakingThrottleDecorator",
				errorsmod.Wrapf(ErrDuplicateStakingInTx, "address %s in %s", addr, typeURL))
		}
		seen[throttleKey] = struct{}{}
		if hasStore && d.keeper.HasThrottleKey(ctx, throttleKey) {
			return ctx, observeReject(ctx, "StakingThrottleDecorator",
				errorsmod.Wrapf(ErrStakingAlreadySubmittedThisBlock, "address %s", addr))
		}
		if hasStore {
			d.keeper.SetThrottleKey(ctx, throttleKey)
		}
	}

	return next(ctx, tx, simulate)
}
