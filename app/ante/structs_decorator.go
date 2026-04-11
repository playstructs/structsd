package ante

import (
	"fmt"

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
	if !IsFreeTx(ctx) {
		return next(ctx, tx, simulate)
	}

	msgs := tx.GetMsgs()

	// Cache address -> playerIndex to avoid repeat lookups in multi-msg txs
	addressCache := make(map[string]uint64)

	// Count Structs messages per player in this tx for the message cap check
	playerMsgCounts := make(map[string]uint64)

	for _, msg := range msgs {
		typeURL := sdk.MsgTypeURL(msg)

		if !KnownStructsMessages[typeURL] {
			return ctx, fmt.Errorf("structs ante: unknown structs message type %s, update ante maps", typeURL)
		}

		var creator string
		if cg, ok := msg.(creatorGetter); ok {
			creator = cg.GetCreator()
		} else if extractor, hasExtractor := CreatorExtractors[typeURL]; hasExtractor {
			creator = extractor(msg)
		} else {
			return ctx, fmt.Errorf("structs ante: message %s has no creator accessor", typeURL)
		}

		playerIndex, cached := addressCache[creator]
		if !cached {
			playerIndex = d.keeper.GetPlayerIndexFromAddress(ctx, creator)
			if playerIndex == 0 {
				return ctx, fmt.Errorf("structs ante: address %s not registered as player", creator)
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
					return ctx, fmt.Errorf("structs ante: address %s lacks permission %d for %s (has %d)", creator, requiredPerm, typeURL, currentPerm)
				}
			}
		}

		// Charge floor check: verify player has not already discharged this block
		if ChargeMessages[typeURL] {
			lastActionAttrId := fmt.Sprintf("%d-%s", types.GridAttributeType_lastAction, playerId)
			lastAction := d.keeper.GetGridAttribute(ctx, lastActionAttrId)
			currentBlock := uint64(ctx.BlockHeight())
			if currentBlock > 0 && lastAction >= currentBlock {
				return ctx, fmt.Errorf("structs ante: player %s has zero charge (discharged this block) for %s", playerId, typeURL)
			}
		}

		playerMsgCounts[playerId]++
	}

	// Per-player-per-block message cap (only during DeliverTx, not CheckTx/simulate)
	if !ctx.IsCheckTx() && !ctx.IsReCheckTx() && !simulate {
		for playerId, count := range playerMsgCounts {
			newTotal := d.keeper.IncrementPlayerMsgCount(ctx, playerId, count)
			if newTotal > d.playerMsgCap {
				return ctx, fmt.Errorf("structs ante: player %s exceeded per-block message cap (%d/%d)", playerId, newTotal, d.playerMsgCap)
			}
		}
	}

	return next(ctx, tx, simulate)
}
