package app

import (
	"encoding/binary"
	"fmt"
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// The SDK's staking genesis restores rows and time queues, but v0.53.5 omits
// operation-ID indices and their sequence. Rebuild indices through the public
// keeper API. Only the sequence lacks an SDK setter; its pinned eight-byte store
// encoding is read/written here at app import/export boundaries, never per block.
func (app *App) stakingUnbondingID(ctx sdk.Context) (uint64, error) {
	value := ctx.KVStore(app.GetKey(stakingtypes.StoreKey)).Get(stakingtypes.UnbondingIDKey)
	if value == nil {
		return 0, nil
	}
	if len(value) != 8 {
		return 0, fmt.Errorf("invalid staking unbonding sequence length %d", len(value))
	}
	return binary.BigEndian.Uint64(value), nil
}

func (app *App) setStakingUnbondingID(ctx sdk.Context, id uint64) {
	if id == 0 {
		return
	}
	ctx.KVStore(app.GetKey(stakingtypes.StoreKey)).Set(stakingtypes.UnbondingIDKey, sdk.Uint64ToBigEndian(id))
}

func stakingGenesisLastID(genesis stakingtypes.GenesisState, exported uint64) (uint64, error) {
	seen := make(map[uint64]bool)
	last := uint64(0)
	add := func(id uint64) error {
		if id == 0 || seen[id] {
			return fmt.Errorf("invalid or duplicate staking operation ID %d", id)
		}
		seen[id] = true
		if id > last {
			last = id
		}
		return nil
	}
	for _, record := range genesis.Validators {
		for _, id := range record.UnbondingIds {
			if err := add(id); err != nil {
				return 0, err
			}
		}
	}
	for _, record := range genesis.UnbondingDelegations {
		for _, entry := range record.Entries {
			if err := add(entry.UnbondingId); err != nil {
				return 0, err
			}
		}
	}
	for _, record := range genesis.Redelegations {
		for _, entry := range record.Entries {
			if err := add(entry.UnbondingId); err != nil {
				return 0, err
			}
		}
	}
	// Old SDK exports have no counter. Recover at least the live maximum. New
	// exports preserve IDs of completed operations too, even with no live rows.
	if exported != 0 {
		if exported < last {
			return 0, fmt.Errorf("staking operation sequence %d is below live ID %d", exported, last)
		}
		last = exported
	}
	if last == math.MaxUint64 {
		return 0, fmt.Errorf("staking operation sequence exhausted")
	}
	return last, nil
}

func (app *App) restoreStakingOperationIndices(ctx sdk.Context, genesis stakingtypes.GenesisState) error {
	for _, record := range genesis.Validators {
		for _, id := range record.UnbondingIds {
			if err := app.StakingKeeper.SetValidatorByUnbondingID(ctx, record, id); err != nil {
				return err
			}
		}
	}
	for _, record := range genesis.UnbondingDelegations {
		for _, entry := range record.Entries {
			if err := app.StakingKeeper.SetUnbondingDelegationByUnbondingID(ctx, record, entry.UnbondingId); err != nil {
				return err
			}
		}
	}
	for _, record := range genesis.Redelegations {
		for _, entry := range record.Entries {
			if err := app.StakingKeeper.SetRedelegationByUnbondingID(ctx, record, entry.UnbondingId); err != nil {
				return err
			}
		}
	}
	return nil
}
