package ante

import (
	"fmt"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const DefaultCheckTxAddrCap uint64 = 5

// CheckTxThrottleDecorator limits the number of free Structs transactions a
// single address can submit per block during CheckTx. This is a node-local
// defense against mempool flooding (not consensus state).
//
// Placement: MUST be after SigVerificationDecorator so the signer identity is
// authenticated. Runs only during CheckTx (not ReCheckTx, DeliverTx, or simulation).
type CheckTxThrottleDecorator struct {
	addrCap uint64
	counter *addressCounter
}

type addressCounter struct {
	mu         sync.Mutex
	lastHeight int64
	counts     map[string]uint64
}

func NewCheckTxThrottleDecorator(addrCap uint64) CheckTxThrottleDecorator {
	if addrCap == 0 {
		addrCap = DefaultCheckTxAddrCap
	}
	return CheckTxThrottleDecorator{
		addrCap: addrCap,
		counter: &addressCounter{counts: make(map[string]uint64)},
	}
}

func (d CheckTxThrottleDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if !IsFreeTx(ctx) {
		return next(ctx, tx, simulate)
	}

	if !ctx.IsCheckTx() || ctx.IsReCheckTx() || simulate {
		return next(ctx, tx, simulate)
	}

	msgs := tx.GetMsgs()

	addresses := make(map[string]bool)
	for _, msg := range msgs {
		typeURL := sdk.MsgTypeURL(msg)
		var addr string
		if extractor, ok := StakingSignerExtractors[typeURL]; ok {
			addr = extractor(msg)
		} else if cg, ok := msg.(creatorGetter); ok {
			addr = cg.GetCreator()
		} else if extractor, hasExtractor := CreatorExtractors[typeURL]; hasExtractor {
			addr = extractor(msg)
		}
		if addr != "" {
			addresses[addr] = true
		}
	}

	if len(addresses) == 0 {
		return next(ctx, tx, simulate)
	}

	d.counter.mu.Lock()
	defer d.counter.mu.Unlock()

	height := ctx.BlockHeight()
	if height != d.counter.lastHeight {
		d.counter.counts = make(map[string]uint64)
		d.counter.lastHeight = height
	}

	for addr := range addresses {
		newCount := d.counter.counts[addr] + 1
		if newCount > d.addrCap {
			return ctx, fmt.Errorf("structs ante: address %s exceeded CheckTx free-tx cap (%d/%d) for block %d", addr, newCount, d.addrCap, height)
		}
		d.counter.counts[addr] = newCount
	}

	return next(ctx, tx, simulate)
}
