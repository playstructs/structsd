package ante

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

// ThrottleDecorator enforces per-object-per-block and per-player-per-block rate
// limits on free Structs transactions.
//
// It runs in every transaction phase (CheckTx, ReCheckTx, DeliverTx, and
// simulation) and combines two layers of protection:
//
//  1. Per-tx in-memory deduplication. A local `seen` map is built for every
//     tx, indexed by throttle key (e.g. "charge/<playerId>", "proof/<structId>",
//     "fleet/<fleetId>"). Two messages in the same tx that map to the same key
//     are always rejected, in every phase, with no dependency on consensus or
//     mempool state. This is the layer that catches the duplicate-charge-in-
//     same-tx admission bug (incident 2026-05).
//
//  2. Cross-tx transient-store check. After per-tx dedup passes, every key is
//     looked up in the keeper's transient store (writes are scoped per-phase
//     by the SDK: CheckTx state writes are isolated from DeliverTx state). A
//     key already set this block (because an earlier tx from the same block
//     consumed it) causes a reject. This is the layer that prevents two
//     distinct charge txs from the same player landing in one block.
//
// Both layers run in CheckTx. Letting the throttle skip CheckTx is what caused
// invalid txs to sit in the mempool indefinitely on structstestnet-111, since
// CometBFT v0.38.21 only evicts txs that fail ReCheckTx or get included in a
// block. See docs/incident-2026-05-ante.md.
type ThrottleDecorator struct {
	keeper StructsAnteKeeper
}

func NewThrottleDecorator(keeper StructsAnteKeeper) ThrottleDecorator {
	return ThrottleDecorator{keeper: keeper}
}

func (d ThrottleDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if !IsFreeTx(ctx) || IsFreeStakingTx(ctx) {
		return next(ctx, tx, simulate)
	}

	hasStore := d.keeper.HasTransientStore()

	seen := make(map[string]struct{}, len(tx.GetMsgs())*2)

	for _, msg := range tx.GetMsgs() {
		typeURL := sdk.MsgTypeURL(msg)

		// Per-object proof throttle: one attempt per struct/fleet per block.
		if extractor, isProof := ProofMessages[typeURL]; isProof {
			objectId := extractor(msg)
			if objectId == "" {
				return ctx, observeReject(ctx, "ThrottleDecorator",
					errorsmod.Wrapf(ErrMissingCreator, "type assertion failed for %s (proof extractor returned empty)", typeURL))
			}
			throttleKey := "proof/" + objectId
			if _, dup := seen[throttleKey]; dup {
				return ctx, observeReject(ctx, "ThrottleDecorator",
					errorsmod.Wrapf(ErrDuplicateProofInTx, "%s: %s", typeURL, objectId))
			}
			seen[throttleKey] = struct{}{}
			if hasStore && d.keeper.HasThrottleKey(ctx, throttleKey) {
				return ctx, observeReject(ctx, "ThrottleDecorator",
					errorsmod.Wrapf(ErrProofAlreadyAttemptedThisBlock, "%s: %s", typeURL, objectId))
			}
			if hasStore {
				d.keeper.SetThrottleKey(ctx, throttleKey)
			}
		}

		// Per-object operational throttle (fleet move, planet explore, address register).
		if extractor, hasThrottle := ThrottleKeyExtractors[typeURL]; hasThrottle {
			throttleKey := extractor(msg)
			if throttleKey == "" {
				return ctx, observeReject(ctx, "ThrottleDecorator",
					errorsmod.Wrapf(ErrMissingCreator, "type assertion failed for %s (throttle extractor returned empty)", typeURL))
			}
			if _, dup := seen[throttleKey]; dup {
				return ctx, observeReject(ctx, "ThrottleDecorator",
					errorsmod.Wrapf(ErrDuplicateThrottleKeyInTx, "%s: %s", typeURL, throttleKey))
			}
			seen[throttleKey] = struct{}{}
			if hasStore && d.keeper.HasThrottleKey(ctx, throttleKey) {
				return ctx, observeReject(ctx, "ThrottleDecorator",
					errorsmod.Wrapf(ErrObjectThrottledThisBlock, "%s: key %s", typeURL, throttleKey))
			}
			if hasStore {
				d.keeper.SetThrottleKey(ctx, throttleKey)
			}
		}

		// Per-player charge throttle: one charge-consuming action per player per block.
		if ChargeMessages[typeURL] {
			var creator string
			if cg, ok := msg.(creatorGetter); ok {
				creator = cg.GetCreator()
			} else if extractor, hasExtractor := CreatorExtractors[typeURL]; hasExtractor {
				creator = extractor(msg)
			}
			if creator == "" {
				return ctx, observeReject(ctx, "ThrottleDecorator",
					errorsmod.Wrapf(ErrMissingCreator, "%s declared in ChargeMessages but creator could not be resolved", typeURL))
			}
			playerIndex := d.keeper.GetPlayerIndexFromAddress(ctx, creator)
			if playerIndex == 0 {
				// An unregistered address attempting a charge action will be
				// rejected by StructsDecorator earlier in the chain; we treat
				// this as a no-op for the throttle. The chain-of-decorators
				// ordering (StructsDecorator before ThrottleDecorator) makes
				// this path effectively unreachable, but we keep the branch so
				// the throttle is robust to chain reordering during testing.
				continue
			}
			playerId := fmt.Sprintf("%d-%d", types.ObjectType_player, playerIndex)
			throttleKey := "charge/" + playerId
			if _, dup := seen[throttleKey]; dup {
				return ctx, observeReject(ctx, "ThrottleDecorator",
					errorsmod.Wrapf(ErrDuplicateChargeInTx, "player %s, msg %s", playerId, typeURL))
			}
			seen[throttleKey] = struct{}{}
			if hasStore && d.keeper.HasThrottleKey(ctx, throttleKey) {
				return ctx, observeReject(ctx, "ThrottleDecorator",
					errorsmod.Wrapf(ErrChargeAlreadyUsedThisBlock, "player %s, msg %s", playerId, typeURL))
			}
			if hasStore {
				d.keeper.SetThrottleKey(ctx, throttleKey)
			}
		}
	}

	return next(ctx, tx, simulate)
}
