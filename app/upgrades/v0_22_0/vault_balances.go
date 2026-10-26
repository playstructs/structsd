package v0_22_0

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	vestingexported "github.com/cosmos/cosmos-sdk/x/auth/vesting/exported"
	"structs/app/upgrades"
)

// MigrateVaultBankBalances moves all ordinary liquid denominations without
// withdrawing staking rewards. Native vesting accounts are excluded by the plan;
// their balances and counters remain together. The enclosing upgrade is atomic.
func MigrateVaultBankBalances(ctx context.Context, keepers *upgrades.Keepers) error {
	plan, err := BuildVaultAddressMigrationPlan(ctx, keepers)
	if err != nil {
		return err
	}
	type transfer struct {
		source, destination sdk.AccAddress
		coins               sdk.Coins
	}
	pending := make([]transfer, 0, len(plan))
	bank := keepers.StructsKeeper.BankKeeper()
	accounts := keepers.StructsKeeper.AccountKeeper()
	for _, move := range plan {
		source, err := sdk.AccAddressFromBech32(move.Source)
		if err != nil {
			return err
		}
		destination, err := sdk.AccAddressFromBech32(move.Destination)
		if err != nil {
			return err
		}
		account := accounts.GetAccount(ctx, source)
		if account == nil {
			return fmt.Errorf("migration source %s has no auth account", move.Source)
		}
		if _, vesting := account.(vestingexported.VestingAccount); vesting {
			return fmt.Errorf("vesting balances at %s require vesting-account migration", move.Source)
		}
		coins := bank.GetAllBalances(ctx, source)
		if coins.IsZero() {
			continue
		}
		if _, indexed := keepers.StructsKeeper.GetVaultOwner(ctx, destination); !indexed || accounts.GetAccount(ctx, destination) == nil {
			return fmt.Errorf("destination vault %s is not provisioned", move.Destination)
		}
		if !coins.IsValid() {
			return fmt.Errorf("migration source %s has invalid balances", move.Source)
		}
		pending = append(pending, transfer{source, destination, coins})
	}
	// Unlike a user transfer, relocation on upgrade must include denominations
	// with user sends disabled. SendCoins retains the registered holder restriction
	// but does not apply MsgSend's send-enabled gate.
	for _, move := range pending {
		if err := bank.SendCoins(ctx, move.source, move.destination, move.coins); err != nil {
			return err
		}
	}
	return nil
}
