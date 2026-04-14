package ante

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

// ThrottleDecorator enforces per-object-per-block rate limits via the transient
// store. Covers proof-of-work messages, fleet movement, planet exploration, and
// address registration. Only active during DeliverTx (skipped in CheckTx and
// simulation since the transient store resets each block).
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

	if ctx.IsCheckTx() || ctx.IsReCheckTx() || simulate {
		return next(ctx, tx, simulate)
	}

	for _, msg := range tx.GetMsgs() {
		typeURL := sdk.MsgTypeURL(msg)

		// Per-object proof throttle: one attempt per struct/fleet per block
		if extractor, isProof := ProofMessages[typeURL]; isProof {
			objectId := extractor(msg)
			if objectId == "" {
				return ctx, fmt.Errorf("structs ante: type assertion failed for %s -- message type mismatch", typeURL)
			}
			throttleKey := "proof/" + objectId
			if d.keeper.HasThrottleKey(ctx, throttleKey) {
				return ctx, fmt.Errorf("structs ante: proof already attempted for %s this block", objectId)
			}
			d.keeper.SetThrottleKey(ctx, throttleKey)
		}

		// Per-object operational throttle (fleet move, planet explore, address register)
		if extractor, hasThrottle := ThrottleKeyExtractors[typeURL]; hasThrottle {
			throttleKey := extractor(msg)
			if throttleKey == "" {
				return ctx, fmt.Errorf("structs ante: type assertion failed for %s -- message type mismatch", typeURL)
			}
			if d.keeper.HasThrottleKey(ctx, throttleKey) {
				return ctx, fmt.Errorf("structs ante: %s throttled this block (key: %s)", typeURL, throttleKey)
			}
			d.keeper.SetThrottleKey(ctx, throttleKey)
		}

		// Per-player charge throttle: one charge-consuming action per player per block
		if ChargeMessages[typeURL] {
			var creator string
			if cg, ok := msg.(creatorGetter); ok {
				creator = cg.GetCreator()
			} else if extractor, hasExtractor := CreatorExtractors[typeURL]; hasExtractor {
				creator = extractor(msg)
			}
			if creator != "" {
				playerIndex := d.keeper.GetPlayerIndexFromAddress(ctx, creator)
				if playerIndex != 0 {
					playerId := fmt.Sprintf("%d-%d", types.ObjectType_player, playerIndex)
					throttleKey := "charge/" + playerId
					if d.keeper.HasThrottleKey(ctx, throttleKey) {
						return ctx, fmt.Errorf("structs ante: player %s already used charge action this block", playerId)
					}
					d.keeper.SetThrottleKey(ctx, throttleKey)
				}
			}
		}
	}

	return next(ctx, tx, simulate)
}
