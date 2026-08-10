package keeper

import (
	"fmt"
	"strings"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

// RegisterInvariants registers the module's invariants with the crisis module.
func RegisterInvariants(ir sdk.InvariantRegistry, k Keeper) {
	ir.RegisterRoute(types.ModuleName, "provider-collateral-solvency", ProviderCollateralSolvencyInvariant(k))
}

// ProviderCollateralSolvencyInvariant asserts that every provider's collateral
// pool still holds at least what its open agreements are owed.
//
// The pool is keyed only by provider, so all of a provider's agreements share
// one account and nothing about the account itself prevents one agreement's
// teardown from being funded by another's collateral. This invariant is what
// makes that detectable: it is the check that would have caught the reciprocal
// teardown paying the same agreement twice.
//
// Per open agreement the pool should hold:
//
//   - the unearned collateral, which the consumer gets back if the agreement
//     ends now, and
//   - the provider cancellation penalty accrued so far, which Checkpoint
//     deliberately leaves behind and which settles to one side or the other
//     only when the agreement ends.
//
// The comparison is greater-or-equal rather than exact because truncation on
// every payout leaves dust behind, and because revenue accrued since the last
// checkpoint is still sitting in the pool.
func ProviderCollateralSolvencyInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var broken bool
		var msg strings.Builder

		currentBlock := uint64(ctx.BlockHeight())

		for _, provider := range k.GetAllProvider(ctx) {
			// Nothing can be owed against a provider with no usable published
			// rate, and the arithmetic below would be meaningless.
			if provider.Rate.Denom == "" || provider.Rate.Amount.IsNil() || provider.ProviderCancellationPenalty.IsNil() {
				continue
			}

			owed := math.ZeroInt()

			for _, agreement := range k.GetAllAgreementByProviderIndex(ctx, provider.Id) {
				capacity := math.NewIntFromUint64(agreement.Capacity)

				// Both spans are clamped to the agreement's own window, matching
				// AgreementCache, so served + unearned is always the agreement's
				// full duration and never more than was deposited against it.
				unearnedBlocks := math.NewIntFromUint64(blocksBetween(clampToWindow(currentBlock, agreement.StartBlock, agreement.EndBlock), agreement.EndBlock))
				owed = owed.Add(unearnedBlocks.Mul(provider.Rate.Amount).Mul(capacity))

				servedBlocks := math.NewIntFromUint64(blocksBetween(agreement.StartBlock, clampToWindow(currentBlock, agreement.StartBlock, agreement.EndBlock)))
				accrued := math.LegacyNewDecFromInt(servedBlocks.Mul(provider.Rate.Amount).Mul(capacity))
				owed = owed.Add(accrued.Mul(provider.ProviderCancellationPenalty).TruncateInt())
			}

			if !owed.IsPositive() {
				continue
			}

			collateralPool := GetProviderCollateralPoolLocation(provider.Id)
			held := k.bankKeeper.SpendableCoin(ctx, collateralPool, provider.Rate.Denom).Amount

			if held.LT(owed) {
				broken = true
				msg.WriteString(fmt.Sprintf(
					"\tprovider %s collateral pool holds %s%s but owes %s%s (deficit %s)\n",
					provider.Id, held.String(), provider.Rate.Denom,
					owed.String(), provider.Rate.Denom,
					owed.Sub(held).String(),
				))
			}
		}

		return sdk.FormatInvariant(types.ModuleName, "provider-collateral-solvency",
			fmt.Sprintf("insolvent provider collateral pools found\n%s", msg.String())), broken
	}
}

// blocksBetween is the span from one block to another, floored at zero.
func blocksBetween(from uint64, to uint64) uint64 {
	if to <= from {
		return 0
	}
	return to - from
}

// clampToWindow pins a height inside an agreement's start and end blocks.
func clampToWindow(block uint64, start uint64, end uint64) uint64 {
	if block < start {
		return start
	}
	if block > end {
		return end
	}
	return block
}
