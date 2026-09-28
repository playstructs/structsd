package v0_22_0

import (
	"context"
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	vestingexported "github.com/cosmos/cosmos-sdk/x/auth/vesting/exported"
	"structs/app/upgrades"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// VaultAddressMovement maps ordinary liquid balances to their owner vault.
// It never changes signer identity, staking records or reward destinations.
type VaultAddressMovement struct {
	OwnerID     string
	Source      string
	Destination string
}

// BuildVaultAddressMigrationPlan is read-only. Only the live player-address
// index establishes ownership; unregistered bank accounts are outside the plan.
// Run it before any asset mutations, and use this same mapping in every phase.
func BuildVaultAddressMigrationPlan(ctx context.Context, keepers *upgrades.Keepers) ([]VaultAddressMovement, error) {
	k := keepers.StructsKeeper
	cc := k.NewCurrentContext(ctx)
	movements := make([]VaultAddressMovement, 0)
	for _, record := range k.GetAllAddressExport(ctx) {
		if record.PlayerIndex == 0 {
			continue
		}
		source, err := sdk.AccAddressFromBech32(record.Address)
		if err != nil {
			return nil, err
		}
		ownerID := keeperlib.GetObjectID(types.ObjectType_player, record.PlayerIndex)
		if _, err := cc.GetExistingAssetOwner(ownerID); err != nil {
			return nil, err
		}
		vault, err := cc.GetVault(ownerID)
		if err != nil {
			return nil, err
		}
		if k.BankKeeper().BlockedAddr(source) {
			return nil, fmt.Errorf("player address index names blocked account %s", record.Address)
		}
		if _, indexed := k.GetVaultOwner(ctx, source); indexed || source.Equals(vault.Address()) {
			return nil, fmt.Errorf("protocol vault %s is indexed as a player signing address", record.Address)
		}
		// Keep native vesting accounts whole: their bank balances, schedules
		// and delegated counters still belong to stake at this address.
		if _, vesting := k.AccountKeeper().GetAccount(ctx, source).(vestingexported.VestingAccount); vesting {
			continue
		}
		movements = append(movements, VaultAddressMovement{OwnerID: ownerID, Source: source.String(), Destination: vault.Address().String()})
	}
	sort.Slice(movements, func(i, j int) bool { return movements[i].Source < movements[j].Source })
	return movements, nil
}
