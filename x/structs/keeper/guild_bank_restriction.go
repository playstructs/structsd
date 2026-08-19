package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

// GuildBankDenomSendRestriction confines clawback-enabled guild tokens to
// registered player addresses and the protocol accounts needed by the guild bank
// and provider agreements. The restriction is destination-only: balances written
// before this rule can always move out to an eligible holder.
func (k Keeper) GuildBankDenomSendRestriction(
	ctx context.Context,
	_ sdk.AccAddress,
	toAddr sdk.AccAddress,
	amount sdk.Coins,
) (sdk.AccAddress, error) {
	guildDenom := ""
	for _, coin := range amount {
		if types.IsGuildBankDenom(coin.Denom) {
			guildDenom = coin.Denom
			break
		}
	}
	if guildDenom == "" {
		return toAddr, nil
	}

	if k.GetPlayerIndexFromAddress(ctx, toAddr.String()) > 0 {
		return toAddr, nil
	}

	moduleAddress := k.accountKeeper.GetModuleAddress(types.ModuleName)
	if moduleAddress != nil && moduleAddress.Equals(toAddr) {
		return toAddr, nil
	}

	if _, _, found := k.GetProviderPoolAddress(ctx, toAddr.String()); found {
		return toAddr, nil
	}

	return nil, types.NewGuildBankDestinationError(guildDenom, toAddr.String(), "recipient_not_eligible")
}
