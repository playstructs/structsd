package ante

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const DefaultMaxFreeTxSize = 32768 // 32KB

type TxSizeDecorator struct {
	maxSize int
}

func NewTxSizeDecorator(maxSize int) TxSizeDecorator {
	if maxSize <= 0 {
		maxSize = DefaultMaxFreeTxSize
	}
	return TxSizeDecorator{maxSize: maxSize}
}

func (d TxSizeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	msgs := tx.GetMsgs()
	if !IsAnyFreeTransaction(msgs) {
		return next(ctx, tx, simulate)
	}

	txBytes := ctx.TxBytes()
	if len(txBytes) > d.maxSize {
		return ctx, fmt.Errorf("structs ante: free tx size %d exceeds cap %d bytes", len(txBytes), d.maxSize)
	}

	return next(ctx, tx, simulate)
}
