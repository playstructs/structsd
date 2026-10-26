package v0_22_0

import (
	"context"
	"fmt"
	"sort"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"structs/app/upgrades"
	"structs/x/structs/types"
)

// MigrateLegacyReactorInfusions rebuilds game backing and maturity work at the
// original delegator addresses. Native staking and distribution records are read
// only. A fresh CurrentContext per reconcile avoids nested stale-cache commits.
func MigrateLegacyReactorInfusions(ctx context.Context, keepers *upgrades.Keepers) error {
	if keepers.StakingKeeper == nil {
		return fmt.Errorf("infusion migration requires staking keeper")
	}
	k := keepers.StructsKeeper
	associated := func(address string) bool {
		acc, err := sdk.AccAddressFromBech32(address)
		if err != nil {
			return false
		}
		_, vault := k.GetVaultOwner(ctx, acc)
		return vault || k.GetPlayerIndexFromAddress(ctx, address) != 0
	}
	type pair struct{ address, validator string }
	old := map[pair]bool{}
	current := map[pair]bool{}
	maturities := map[pair][]time.Time{}
	for _, infusion := range k.GetAllInfusion(ctx) {
		if infusion.DestinationType != types.ObjectType_reactor || !associated(infusion.Address) {
			continue
		}
		reactor, found := k.GetReactor(ctx, infusion.DestinationId)
		if !found {
			return fmt.Errorf("infusion %s references missing reactor %s", infusion.Address, infusion.DestinationId)
		}
		old[pair{infusion.Address, reactor.Validator}] = true
	}
	if err := keepers.StakingKeeper.IterateAllDelegations(ctx, func(record stakingtypes.Delegation) bool {
		if associated(record.DelegatorAddress) {
			current[pair{record.DelegatorAddress, record.ValidatorAddress}] = true
		}
		return false
	}); err != nil {
		return err
	}
	if err := keepers.StakingKeeper.IterateUnbondingDelegations(ctx, func(_ int64, record stakingtypes.UnbondingDelegation) bool {
		if associated(record.DelegatorAddress) {
			key := pair{record.DelegatorAddress, record.ValidatorAddress}
			current[key] = true
			for _, entry := range record.Entries {
				maturities[key] = append(maturities[key], entry.CompletionTime)
			}
		}
		return false
	}); err != nil {
		return err
	}
	sorted := func(records map[pair]bool) []pair {
		keys := make([]pair, 0, len(records))
		for key := range records {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool {
			if keys[i].address == keys[j].address {
				return keys[i].validator < keys[j].validator
			}
			return keys[i].address < keys[j].address
		})
		return keys
	}
	oldKeys, currentKeys := sorted(old), sorted(current)
	// Validate every address before any reconcile commits.
	for _, key := range append(append([]pair{}, oldKeys...), currentKeys...) {
		_, err := sdk.AccAddressFromBech32(key.address)
		if err != nil {
			return err
		}
		if _, err := sdk.ValAddressFromBech32(key.validator); err != nil {
			return err
		}
	}
	for _, key := range oldKeys {
		if current[key] {
			continue
		}
		address, _ := sdk.AccAddressFromBech32(key.address)
		validator, _ := sdk.ValAddressFromBech32(key.validator)
		if err := k.ReconcileExistingInfusionForDelegation(ctx, address, validator); err != nil {
			return err
		}
	}
	for _, key := range currentKeys {
		address, _ := sdk.AccAddressFromBech32(key.address)
		validator, _ := sdk.ValAddressFromBech32(key.validator)
		if err := k.ReconcileInfusionForDelegation(ctx, address, validator); err != nil {
			return err
		}
		reactorBytes, found := k.GetReactorBytesFromValidator(ctx, validator)
		if !found {
			continue
		}
		reactor, found := k.GetReactorByBytes(ctx, reactorBytes)
		if !found {
			continue
		}
		for _, completion := range maturities[key] {
			k.EnqueueInfusionMaturitySweep(ctx, completion, reactor.Id+"-"+key.address)
		}
	}
	return nil
}
