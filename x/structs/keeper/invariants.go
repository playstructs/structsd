package keeper

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

// RegisterInvariants registers the module's invariants with the crisis module.
func RegisterInvariants(ir sdk.InvariantRegistry, k Keeper) {
	ir.RegisterRoute(types.ModuleName, "provider-collateral-solvency", ProviderCollateralSolvencyInvariant(k))
	ir.RegisterRoute(types.ModuleName, "agreement-expiry-liveness", AgreementExpiryLivenessInvariant(k))
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

		for _, provider := range k.GetAllProvider(ctx) {
			owed := k.ProviderCollateralObligation(ctx, provider)

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

// AgreementExpiryLivenessInvariant asserts that no agreement outlives its own
// end block.
//
// An agreement is expired by AgreementExpirations from the EndBlocker, which
// reads the expiration index at exactly the current height. There is no range
// scan and no retry, so an agreement that is not torn down on the one block it
// comes up is never revisited: it keeps its capacity in the provider's load, and
// Checkpoint goes on billing that capacity against the shared collateral pool
// every block, funding the overcharge out of other consumers' collateral.
//
// The solvency invariant cannot stand in for this. It clamps every span to the
// agreement's own window, so an overdue agreement reads as one that has simply
// been fully served, and it only breaks on a shortfall — by the time the pool is
// visibly short, the draining has already happened.
//
// The comparison is strictly less-than so that an agreement ending on the
// current height has the whole block to be expired in, whatever order the crisis
// module and this module's EndBlocker run in.
func AgreementExpiryLivenessInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var broken bool
		var msg strings.Builder

		currentBlock := uint64(ctx.BlockHeight())

		for _, agreement := range k.GetAllAgreement(ctx) {
			if agreement.EndBlock >= currentBlock {
				continue
			}

			broken = true
			msg.WriteString(fmt.Sprintf(
				"\tagreement %s (provider %s) ended at block %d but still exists at block %d, holding %d capacity in the provider's load\n",
				agreement.Id, agreement.ProviderId, agreement.EndBlock, currentBlock, agreement.Capacity,
			))
		}

		return sdk.FormatInvariant(types.ModuleName, "agreement-expiry-liveness",
			fmt.Sprintf("agreements found past their end block\n%s", msg.String())), broken
	}
}
