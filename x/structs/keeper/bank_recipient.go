package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

/* resolveExternalRecipient turns an address a message chose into one coins may
 * actually be sent to.
 *
 * BaseSendKeeper.SendCoins is not the bank's policy layer. It validates the coin
 * structure, applies any registered send restriction, and moves balances - it
 * does not consult the blocked-address set or the per-denom send-enabled flags.
 * Those live in the bank's own MsgServer, which a module calling SendCoins
 * directly never goes through. So the app can declare the fee collector, the
 * distribution account and both staking pools unreachable, and a module handler
 * will still reach them; governance can freeze a denom, and this path will still
 * move it.
 *
 * The staking pools are the sharp end. Coins sent into the bonded or not-bonded
 * pool with no matching delegation leave the pool balance disagreeing with the
 * staking module's recorded tokens, which is an invariant break that surfaces as
 * a panic on the next export-and-restart rather than as a failed transaction.
 *
 * Only destinations a message names go through here. Every other SendCoins in
 * this module sends to an address it derived itself - a provider pool, a guild
 * bank, a player's own primary address - and those are neither blocked nor
 * attacker-chosen.
 */
func (k Keeper) resolveExternalRecipient(address string) (sdk.AccAddress, error) {
	recipient, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return nil, types.NewAddressValidationError(address, "invalid_format")
	}

	if k.bankKeeper.BlockedAddr(recipient) {
		return nil, types.NewAddressValidationError(address, "blocked_recipient")
	}

	return recipient, nil
}

// requireSendableAmount rejects an amount the bank would not accept from its own
// MsgServer: malformed or non-positive, or carrying a denom governance has
// frozen.
func (k Keeper) requireSendableAmount(ctx context.Context, amount sdk.Coins) error {
	if !amount.IsValid() || !amount.IsAllPositive() {
		return types.NewParameterValidationError("amount", 0, "not_positive")
	}

	if err := k.bankKeeper.IsSendEnabledCoins(ctx, amount...); err != nil {
		return err
	}

	return nil
}
