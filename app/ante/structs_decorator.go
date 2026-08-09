package ante

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

const DefaultPlayerMsgCap uint64 = 40

// creatorGetter matches all Structs messages which expose GetCreator().
type creatorGetter interface {
	GetCreator() string
}

type StructsDecorator struct {
	keeper       StructsAnteKeeper
	playerMsgCap uint64
}

func NewStructsDecorator(keeper StructsAnteKeeper, playerMsgCap uint64) StructsDecorator {
	if playerMsgCap == 0 {
		playerMsgCap = DefaultPlayerMsgCap
	}
	return StructsDecorator{keeper: keeper, playerMsgCap: playerMsgCap}
}

func (d StructsDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	msgs := tx.GetMsgs()

	if !ContainsGatedStructsMessage(msgs) {
		return next(ctx, tx, simulate)
	}

	// Cache address -> playerIndex to avoid repeat lookups in multi-msg txs
	addressCache := make(map[string]uint64)

	// Count Structs messages per player in this tx for the message cap check
	playerMsgCounts := make(map[string]uint64)

	for _, msg := range msgs {
		typeURL := sdk.MsgTypeURL(msg)

		// A tx may legitimately mix Structs gameplay with messages from other
		// modules. Those are gated by their own modules; skip them rather than
		// demanding a player-owned creator from them.
		if !IsStructsMessage(typeURL) || typeURL == MsgUpdateParamsTypeURL {
			continue
		}

		if !KnownStructsMessages[typeURL] {
			return ctx, observeReject(ctx, "StructsDecorator",
				errorsmod.Wrapf(ErrUnknownStructsMessage, "%s (update ante maps)", typeURL))
		}

		var creator string
		if cg, ok := msg.(creatorGetter); ok {
			creator = cg.GetCreator()
		} else if extractor, hasExtractor := CreatorExtractors[typeURL]; hasExtractor {
			creator = extractor(msg)
			if creator == "" {
				return ctx, observeReject(ctx, "StructsDecorator",
					errorsmod.Wrapf(ErrMissingCreator, "type assertion failed for %s", typeURL))
			}
		} else {
			return ctx, observeReject(ctx, "StructsDecorator",
				errorsmod.Wrapf(ErrMissingCreator, "%s has no creator accessor", typeURL))
		}

		playerIndex, cached := addressCache[creator]
		if !cached {
			playerIndex = d.keeper.GetPlayerIndexFromAddress(ctx, creator)
			if playerIndex == 0 {
				return ctx, observeReject(ctx, "StructsDecorator",
					errorsmod.Wrapf(ErrUnregisteredAddress, "%s for %s", creator, typeURL))
			}
			addressCache[creator] = playerIndex
		}

		playerId := fmt.Sprintf("%d-%d", types.ObjectType_player, playerIndex)

		// Address-level permission check (Layer 1 only)
		if !DynamicPermissionMessages[typeURL] {
			requiredPerm, hasPerm := PermissionMap[typeURL]
			if hasPerm && requiredPerm != 0 {
				addrPermId := []byte(fmt.Sprintf("%d-%s@0", types.ObjectType_address, creator))
				currentPerm := d.keeper.GetPermissionsByBytes(ctx, addrPermId)
				if currentPerm&requiredPerm != requiredPerm {
					return ctx, observeReject(ctx, "StructsDecorator",
						errorsmod.Wrapf(ErrMissingPermission, "address %s wants %d for %s (has %d)", creator, requiredPerm, typeURL, currentPerm))
				}
			}
		}

		// Charge floor check: verify player has not already discharged this block
		if ChargeMessages[typeURL] {
			lastActionAttrId := fmt.Sprintf("%d-%s", types.GridAttributeType_lastAction, playerId)
			lastAction := d.keeper.GetGridAttribute(ctx, lastActionAttrId)
			currentBlock := uint64(ctx.BlockHeight())
			if currentBlock > 0 && lastAction >= currentBlock {
				return ctx, observeReject(ctx, "StructsDecorator",
					errorsmod.Wrapf(ErrPlayerDischargedThisBlock, "player %s for %s", playerId, typeURL))
			}
		}

		playerMsgCounts[playerId]++
	}

	// SKIP_RATIONALE: the per-player-per-block message cap aggregates across
	// every tx in the block, so it can only be evaluated authoritatively in
	// DeliverTx (the SDK's per-block player msg counter only increments
	// during DeliverTx). Running it during CheckTx would double-count
	// because the same tx is re-evaluated when it's later DeliverTx'd, and
	// running it during ReCheckTx would evict txs admitted under a stale
	// counter value. Same-tx duplicate-charge rejection is handled by
	// ThrottleDecorator's per-tx in-memory dedup, which DOES run in every
	// phase. See docs/incident-2026-05-ante.md.
	if !ctx.IsCheckTx() && !ctx.IsReCheckTx() && !simulate {
		for playerId, count := range playerMsgCounts {
			newTotal := d.keeper.IncrementPlayerMsgCount(ctx, playerId, count)
			if newTotal > d.playerMsgCap {
				return ctx, observeReject(ctx, "StructsDecorator",
					errorsmod.Wrapf(ErrPlayerMsgCapExceeded, "player %s: %d/%d", playerId, newTotal, d.playerMsgCap))
			}
		}
	}

	return next(ctx, tx, simulate)
}
