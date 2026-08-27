package ante

import (
	"fmt"
	"sort"
	"sync"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

const DefaultPlayerMsgCap uint64 = 40

// creatorGetter matches all Structs messages which expose GetCreator().
type creatorGetter interface {
	GetCreator() string
}

/* checkTxPlayerCounter is the admission-side mirror of the per-player message
 * cap, and is deliberately not the same counter.
 *
 * The authoritative cap lives in the transient store and is only touched in
 * DeliverTx, for the reasons in the SKIP_RATIONALE below. That left admission
 * bounded per *address* by CheckTxThrottleDecorator while the cap it is
 * predicting is per *player*, and address associations only converge on a player
 * inside this decorator - so a player with several registered addresses could
 * fill the mempool with transactions that the delivery cap was always going to
 * refuse. Structs transactions are free, so those transactions cost their sender
 * nothing and cost the block its bytes.
 *
 * This map is node-local, in memory, and reset when the height moves: it never
 * touches consensus state, so it cannot double-count the authoritative counter
 * or make one node's block differ from another's. Nodes may admit slightly
 * differently, which is already true of the address throttle beside it.
 */
type checkTxPlayerCounter struct {
	mu         sync.Mutex
	lastHeight int64
	counts     map[string]uint64
}

type StructsDecorator struct {
	keeper       StructsAnteKeeper
	playerMsgCap uint64
	checkTx      *checkTxPlayerCounter
}

func NewStructsDecorator(keeper StructsAnteKeeper, playerMsgCap uint64) StructsDecorator {
	if playerMsgCap == 0 {
		playerMsgCap = DefaultPlayerMsgCap
	}
	return StructsDecorator{
		keeper:       keeper,
		playerMsgCap: playerMsgCap,
		checkTx:      &checkTxPlayerCounter{counts: make(map[string]uint64)},
	}
}

/* admitCheckTx applies the same cap at admission, against the node-local count.
 *
 * Fresh CheckTx only. ReCheckTx and DeliverTx run on transactions that already
 * passed here, so counting them again would charge the quota twice, and
 * simulate is a wallet-side estimate rather than admission - the same three
 * reasons CheckTxThrottleDecorator gives for its own counter.
 */
func (d StructsDecorator) admitCheckTx(ctx sdk.Context, playerMsgCounts map[string]uint64) error {
	d.checkTx.mu.Lock()
	defer d.checkTx.mu.Unlock()

	height := ctx.BlockHeight()
	if height != d.checkTx.lastHeight {
		d.checkTx.counts = make(map[string]uint64)
		d.checkTx.lastHeight = height
	}

	playerIds := make([]string, 0, len(playerMsgCounts))
	for playerId := range playerMsgCounts {
		playerIds = append(playerIds, playerId)
	}
	sort.Strings(playerIds)

	for _, playerId := range playerIds {
		newTotal := d.checkTx.counts[playerId] + playerMsgCounts[playerId]
		if newTotal > d.playerMsgCap {
			return observeReject(ctx, "StructsDecorator",
				errorsmod.Wrapf(ErrPlayerMsgCapExceeded, "player %s: %d/%d at admission", playerId, newTotal, d.playerMsgCap))
		}
		d.checkTx.counts[playerId] = newTotal
	}

	return nil
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

		/* A recovery message from the player's own primary address does not spend
		 * the shared quota. See RecoveryMessages: the quota is per player and
		 * charged here in the ante, so a delegated key can exhaust it on messages
		 * that fail, and without this the message it blocks is the one that would
		 * evict it.
		 *
		 * Not counting is the whole exemption. A player whose only messages are
		 * recovery ones never enters playerMsgCounts, so the loop below neither
		 * increments nor compares for them - which is what lets a revoke through
		 * a quota that is already full.
		 */
		if RecoveryMessages[typeURL] && d.keeper.IsPlayerPrimaryAddress(ctx, playerId, creator) {
			continue
		}

		playerMsgCounts[playerId]++
	}

	// Fresh CheckTx gets the same cap applied against a node-local count. See
	// checkTxPlayerCounter: this is admission, not the authoritative counter,
	// and keeping them separate is what avoids the double-count the rationale
	// below describes.
	if ctx.IsCheckTx() && !ctx.IsReCheckTx() && !simulate {
		if err := d.admitCheckTx(ctx, playerMsgCounts); err != nil {
			return ctx, err
		}
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
		// Iterate in sorted player order. IncrementPlayerMsgCount writes to the
		// transient store (gas-metered), and this loop can reject mid-way, so
		// Go's randomized map order would otherwise make the rejected tx's
		// GasUsed node-dependent, which forks the LastResultsHash.
		playerIds := make([]string, 0, len(playerMsgCounts))
		for playerId := range playerMsgCounts {
			playerIds = append(playerIds, playerId)
		}
		sort.Strings(playerIds)

		for _, playerId := range playerIds {
			newTotal := d.keeper.IncrementPlayerMsgCount(ctx, playerId, playerMsgCounts[playerId])
			if newTotal > d.playerMsgCap {
				return ctx, observeReject(ctx, "StructsDecorator",
					errorsmod.Wrapf(ErrPlayerMsgCapExceeded, "player %s: %d/%d", playerId, newTotal, d.playerMsgCap))
			}
		}
	}

	return next(ctx, tx, simulate)
}
