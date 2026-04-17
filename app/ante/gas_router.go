package ante

import (
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DefaultFreeGasCap        uint64 = 20_000_000
	DefaultFreeStakingGasCap uint64 = 40_000_000
)

type freeGasContextKey struct{}
type freeStakingContextKey struct{}

func FreeGasCtxKey() freeGasContextKey       { return freeGasContextKey{} }
func FreeStakingCtxKey() freeStakingContextKey { return freeStakingContextKey{} }

type GasRouterDecorator struct {
	freeGasCap        uint64
	freeStakingGasCap uint64
}

func NewGasRouterDecorator(freeGasCap, freeStakingGasCap uint64) GasRouterDecorator {
	if freeGasCap == 0 {
		freeGasCap = DefaultFreeGasCap
	}
	if freeStakingGasCap == 0 {
		freeStakingGasCap = DefaultFreeStakingGasCap
	}
	return GasRouterDecorator{freeGasCap: freeGasCap, freeStakingGasCap: freeStakingGasCap}
}

func (d GasRouterDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	msgs := tx.GetMsgs()

	if IsFreeTransaction(msgs) {
		freeMeter := storetypes.NewGasMeter(d.freeGasCap)
		ctx = ctx.WithGasMeter(freeMeter)
		ctx = ctx.WithValue(freeGasContextKey{}, true)
	} else if IsFreeStakingTransaction(msgs) {
		freeMeter := storetypes.NewGasMeter(d.freeStakingGasCap)
		ctx = ctx.WithGasMeter(freeMeter)
		ctx = ctx.WithValue(freeGasContextKey{}, true)
		ctx = ctx.WithValue(freeStakingContextKey{}, true)
	}

	return next(ctx, tx, simulate)
}

// IsFreeTx checks the context for the free-gas flag (covers both Structs and staking).
func IsFreeTx(ctx sdk.Context) bool {
	v := ctx.Value(freeGasContextKey{})
	if v == nil {
		return false
	}
	free, ok := v.(bool)
	return ok && free
}

// IsFreeStakingTx checks the context for the staking-specific free flag.
func IsFreeStakingTx(ctx sdk.Context) bool {
	v := ctx.Value(freeStakingContextKey{})
	if v == nil {
		return false
	}
	free, ok := v.(bool)
	return ok && free
}
