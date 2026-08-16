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
// simulation). It rejects duplicate keys within one transaction, checks the
// per-block transient store across transactions, and authorizes caller-supplied
// targets before reserving them. All three checks run in CheckTx so invalid
// transactions are not admitted to the mempool.
type ThrottleDecorator struct {
	keeper StructsAnteKeeper
}

func NewThrottleDecorator(keeper StructsAnteKeeper) ThrottleDecorator {
	return ThrottleDecorator{keeper: keeper}
}

func (d ThrottleDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// Throttling follows the message, not the fee: see
	// ContainsGatedStructsMessage. Every branch below keys off a map holding only
	// Structs type URLs, so messages from other modules fall through untouched.
	if !ContainsGatedStructsMessage(tx.GetMsgs()) {
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
			if hasStore && d.mayReserve(ctx, msg, typeURL) {
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
			if hasStore && d.mayReserve(ctx, msg, typeURL) {
				d.keeper.SetThrottleKey(ctx, throttleKey)
			}
		}

		// Per-player charge throttle: one charge-consuming action per player per block.
		if ChargeMessages[typeURL] {
			creator := resolveCreator(msg, typeURL)
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

// mayReserve reports whether the signer of msg may claim the object-global
// throttle key it names, by asking the keeper for the same target-object
// authorization the handler will perform.
//
// A refusal skips the reservation and lets the message continue to its handler,
// which produces the real error. It is not an ante reject, on purpose: the ante
// sees pre-transaction state while a handler sees the state left by earlier
// messages in the same transaction, so a transaction that grants a permission
// and then uses it is authorized at the handler and not here. Rejecting on this
// answer would break that transaction; skipping the reservation only leaves the
// object unthrottled for the block, which is the same position it would be in
// had the message not been sent.
//
// A message with no ThrottleTargetAuth entry reserves nothing. That fails
// closed against the poisoning this guards, and TestThrottleTargetAuthCompleteness
// catches the omission before it silently disables a throttle.
func (d ThrottleDecorator) mayReserve(ctx sdk.Context, msg sdk.Msg, typeURL string) bool {
	extractor, hasAuth := ThrottleTargetAuth[typeURL]
	if !hasAuth {
		return false
	}

	target, ok := extractor(msg)
	if !ok || target.TargetId == "" {
		return false
	}

	creator := resolveCreator(msg, typeURL)
	if creator == "" {
		return false
	}

	return d.keeper.ThrottleTargetAuthorized(ctx, creator, target.Kind, target.TargetId, target.Permission)
}

// resolveCreator returns the signing address a message declares, preferring the
// generated GetCreator accessor and falling back to the explicit extractor for
// the messages that name the field something else.
func resolveCreator(msg sdk.Msg, typeURL string) string {
	if cg, ok := msg.(creatorGetter); ok {
		return cg.GetCreator()
	}
	if extractor, hasExtractor := CreatorExtractors[typeURL]; hasExtractor {
		return extractor(msg)
	}
	return ""
}
