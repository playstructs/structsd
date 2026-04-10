package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
)

// ConditionalFeeDecorator skips fee deduction for free Structs transactions
// and delegates to the SDK's DeductFeeDecorator for everything else.
type ConditionalFeeDecorator struct {
	inner sdk.AnteDecorator
}

func NewConditionalFeeDecorator(
	ak authkeeper.AccountKeeper,
	bk bankkeeper.Keeper,
	fk feegrantkeeper.Keeper,
) ConditionalFeeDecorator {
	return ConditionalFeeDecorator{
		inner: ante.NewDeductFeeDecorator(ak, bk, fk, nil),
	}
}

func (d ConditionalFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if IsFreeTx(ctx) {
		return next(ctx, tx, simulate)
	}

	return d.inner.AnteHandle(ctx, tx, simulate, next)
}
