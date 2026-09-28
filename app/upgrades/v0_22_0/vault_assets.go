package v0_22_0

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/app/upgrades"
)

// MigrateVaultAssets provisions vaults and moves ordinary liquid balances only.
// Existing delegations, pending staking entries, rewards and native vesting
// accounts stay at their original addresses. Players may move funds after normal
// unbonding; the upgrade never merges staking records or changes their indices.
func MigrateVaultAssets(ctx context.Context, keepers *upgrades.Keepers) error {
	cache, commit := sdk.UnwrapSDKContext(ctx).CacheContext()
	for _, phase := range []func(context.Context, *upgrades.Keepers) error{
		MigrateProvisionVaults,
		MigrateVaultBankBalances,
		MigrateLegacyReactorInfusions,
	} {
		if err := phase(cache, keepers); err != nil {
			return err
		}
	}
	commit()
	return nil
}
