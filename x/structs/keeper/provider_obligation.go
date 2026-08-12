package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

// ProviderCollateralObligation returns the amount that must remain in a
// provider's collateral pool to settle its open agreements at the current
// height: unearned consumer collateral plus the provider-cancellation amount
// accrued over served blocks.
//
// Confiscation and the solvency invariant share this calculation. Duplicating it
// would let a guild burn value the invariant still considers owed.
func (k Keeper) ProviderCollateralObligation(ctx context.Context, provider types.Provider) math.Int {
	if provider.Rate.Denom == "" || provider.Rate.Amount.IsNil() || provider.ProviderCancellationPenalty.IsNil() {
		return math.ZeroInt()
	}

	currentBlock := uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())
	owed := math.ZeroInt()

	for _, agreement := range k.GetAllAgreementByProviderIndex(ctx, provider.Id) {
		capacity := math.NewIntFromUint64(agreement.Capacity)
		serviceBlock := clampToWindow(currentBlock, agreement.StartBlock, agreement.EndBlock)

		unearnedBlocks := math.NewIntFromUint64(blocksBetween(serviceBlock, agreement.EndBlock))
		owed = owed.Add(unearnedBlocks.Mul(provider.Rate.Amount).Mul(capacity))

		servedBlocks := math.NewIntFromUint64(blocksBetween(agreement.StartBlock, serviceBlock))
		accrued := math.LegacyNewDecFromInt(servedBlocks.Mul(provider.Rate.Amount).Mul(capacity))
		owed = owed.Add(accrued.Mul(provider.ProviderCancellationPenalty).TruncateInt())
	}

	return owed
}
