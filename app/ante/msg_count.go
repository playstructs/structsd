package ante

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const DefaultMaxMsgCount = 40

type MsgCountDecorator struct {
	maxCount int
}

func NewMsgCountDecorator(maxCount int) MsgCountDecorator {
	if maxCount <= 0 {
		maxCount = DefaultMaxMsgCount
	}
	return MsgCountDecorator{maxCount: maxCount}
}

func (d MsgCountDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	msgs := tx.GetMsgs()
	if len(msgs) > d.maxCount {
		return ctx, fmt.Errorf("structs ante: tx contains %d messages, cap is %d", len(msgs), d.maxCount)
	}

	return next(ctx, tx, simulate)
}
