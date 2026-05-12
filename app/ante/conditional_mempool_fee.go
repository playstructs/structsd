package ante

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// ConditionalMempoolFeeDecorator enforces mempool min-fee checks for paid
// transactions while skipping the check for free Structs transactions.
type ConditionalMempoolFeeDecorator struct {
}

func NewConditionalMempoolFeeDecorator() ConditionalMempoolFeeDecorator {
	return ConditionalMempoolFeeDecorator{}
}

func (d ConditionalMempoolFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if IsFreeTx(ctx) {
		return next(ctx, tx, simulate)
	}

	// SKIP_RATIONALE: min-gas-price checks are node-local mempool policy
	// (each node operator can set their own min-gas-prices) and are not
	// part of consensus state. They should only run during initial CheckTx
	// admission; running them in ReCheckTx/DeliverTx would risk evicting
	// txs that were admitted under a different operator's policy. Simulate
	// is a wallet-side estimate, not real admission.
	if !ctx.IsCheckTx() || ctx.IsReCheckTx() || simulate {
		return next(ctx, tx, simulate)
	}

	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, errorsmod.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	minGasPrices := ctx.MinGasPrices()
	if minGasPrices.IsZero() {
		return next(ctx, tx, simulate)
	}

	feeCoins := feeTx.GetFee()
	gas := feeTx.GetGas()
	requiredFees := make(sdk.Coins, len(minGasPrices))
	glDec := sdkmath.LegacyNewDec(int64(gas))

	for i, gp := range minGasPrices {
		fee := gp.Amount.Mul(glDec)
		requiredFees[i] = sdk.NewCoin(gp.Denom, fee.Ceil().RoundInt())
	}

	if !feeCoins.IsAnyGTE(requiredFees) {
		return ctx, errorsmod.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient fees; got: %s required: %s", feeCoins, requiredFees)
	}

	return next(ctx, tx, simulate)
}
