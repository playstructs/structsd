package ante

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// StakingThrottleDecorator enforces a per-address-per-block rate limit on free
// staking transactions via the transient store. Each address may submit at most
// one free staking tx per block. Only active during DeliverTx.
type StakingThrottleDecorator struct {
	keeper StructsAnteKeeper
}

func NewStakingThrottleDecorator(keeper StructsAnteKeeper) StakingThrottleDecorator {
	return StakingThrottleDecorator{keeper: keeper}
}

func (d StakingThrottleDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if !IsFreeStakingTx(ctx) {
		return next(ctx, tx, simulate)
	}

	if ctx.IsCheckTx() || ctx.IsReCheckTx() || simulate {
		return next(ctx, tx, simulate)
	}

	if !d.keeper.HasTransientStore() {
		return next(ctx, tx, simulate)
	}

	seen := make(map[string]bool)
	for _, msg := range tx.GetMsgs() {
		typeURL := sdk.MsgTypeURL(msg)
		extractor, ok := StakingSignerExtractors[typeURL]
		if !ok {
			return ctx, fmt.Errorf("structs ante: unknown free staking message %s", typeURL)
		}
		addr := extractor(msg)
		if addr == "" {
			return ctx, fmt.Errorf("structs ante: could not extract signer from %s", typeURL)
		}
		seen[addr] = true
	}

	for addr := range seen {
		throttleKey := "staking/" + addr
		if d.keeper.HasThrottleKey(ctx, throttleKey) {
			return ctx, fmt.Errorf("structs ante: address %s already submitted a free staking tx this block", addr)
		}
		d.keeper.SetThrottleKey(ctx, throttleKey)
	}

	return next(ctx, tx, simulate)
}
