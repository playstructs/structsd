package ante

import (
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const DefaultFreeGasCap uint64 = 20_000_000

// freeGasContextKey is a context value key indicating the tx is on the free path.
type freeGasContextKey struct{}

// FreeGasCtxKey returns the context key used to tag free transactions.
// Exported for use in tests.
func FreeGasCtxKey() freeGasContextKey { return freeGasContextKey{} }

type GasRouterDecorator struct {
	freeGasCap uint64
}

func NewGasRouterDecorator(freeGasCap uint64) GasRouterDecorator {
	if freeGasCap == 0 {
		freeGasCap = DefaultFreeGasCap
	}
	return GasRouterDecorator{freeGasCap: freeGasCap}
}

func (d GasRouterDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	msgs := tx.GetMsgs()
	if IsFreeTransaction(msgs) {
		freeMeter := storetypes.NewGasMeter(d.freeGasCap)
		ctx = ctx.WithGasMeter(freeMeter)
		ctx = ctx.WithValue(freeGasContextKey{}, true)
	}

	return next(ctx, tx, simulate)
}

// IsFreeTx checks the context for the free-gas flag set by GasRouterDecorator.
func IsFreeTx(ctx sdk.Context) bool {
	v := ctx.Value(freeGasContextKey{})
	if v == nil {
		return false
	}
	free, ok := v.(bool)
	return ok && free
}
