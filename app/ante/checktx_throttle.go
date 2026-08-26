package ante

import (
	"sync"

	errorsmod "cosmossdk.io/errors"
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
	// Admission follows the message, not the fee: see ContainsGatedStructsMessage.
	// Keying this off free-ness let a tx pairing gameplay with any non-Structs
	// message escape the per-address cap, and with a zero min gas price that
	// costs nothing. A tx holding neither kind of message is somebody else's
	// business and falls through untouched.
	msgs := tx.GetMsgs()
	if !ContainsGatedStructsMessage(msgs) && !ContainsStakingMessage(msgs) {
		return next(ctx, tx, simulate)
	}

	// SKIP_RATIONALE: CheckTxThrottleDecorator is a node-local mempool-flood
	// defense. It only runs in fresh CheckTx (not ReCheckTx, DeliverTx, or
	// simulate) because:
	//   1. The counter is in-memory and not part of consensus state.
	//   2. ReCheckTx and DeliverTx run on txs that already passed CheckTx, so
	//      re-counting them would double-charge the address quota.
	//   3. Simulate is a wallet-side estimate, not real admission.
	if !ctx.IsCheckTx() || ctx.IsReCheckTx() || simulate {
		return next(ctx, tx, simulate)
	}

	addresses := make(map[string]bool)
	for _, msg := range msgs {
		typeURL := sdk.MsgTypeURL(msg)
		var addr string
		if extractor, ok := StakingSignerExtractors[typeURL]; ok {
			addr = extractor(msg)
		} else if IsStructsMessage(typeURL) && typeURL != MsgUpdateParamsTypeURL {
			// creatorGetter is a bare interface assertion, so it must be reached
			// only for Structs messages. It was safe unguarded while this ran on
			// pure-Structs txs; now that a mixed tx gets here, another module's
			// message exposing GetCreator() would otherwise spend this address's
			// quota. CreatorExtractors is keyed by Structs type URL already.
			if cg, ok := msg.(creatorGetter); ok {
				addr = cg.GetCreator()
			} else if extractor, hasExtractor := CreatorExtractors[typeURL]; hasExtractor {
				addr = extractor(msg)
			}
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
			return ctx, observeReject(ctx, "CheckTxThrottleDecorator",
				errorsmod.Wrapf(ErrCheckTxAddrCapExceeded, "address %s: %d/%d at block %d", addr, newCount, d.addrCap, height))
		}
		d.counter.counts[addr] = newCount
	}

	return next(ctx, tx, simulate)
}
