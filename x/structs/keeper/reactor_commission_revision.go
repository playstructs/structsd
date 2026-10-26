package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
)

const reactorCommissionRevisionPrefix = "Reactor/commissionRevision/"

// ReactorCommissionRevision tracks actual slash transitions independently of
// permissionless repair requests. A missing record is the initial clean state.
// Genesis rebuilds commission from staking before this derived state is usable.
type ReactorCommissionRevision struct {
	Issued    uint64
	Certified uint64
}

func (k Keeper) reactorCommissionRevisionStore(ctx context.Context) prefix.Store {
	return prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte(reactorCommissionRevisionPrefix))
}

func (k Keeper) GetReactorCommissionRevision(ctx context.Context, id string) (ReactorCommissionRevision, error) {
	return decodeReactorCommissionRevision(id, k.reactorCommissionRevisionStore(ctx).Get([]byte(id)))
}

func decodeReactorCommissionRevision(id string, value []byte) (ReactorCommissionRevision, error) {
	if value == nil {
		return ReactorCommissionRevision{}, nil
	}
	if len(value) != 16 {
		return ReactorCommissionRevision{}, fmt.Errorf("invalid reactor commission revision for %s", id)
	}
	revision := ReactorCommissionRevision{Issued: binary.BigEndian.Uint64(value[:8]), Certified: binary.BigEndian.Uint64(value[8:])}
	if revision.Certified > revision.Issued {
		return revision, fmt.Errorf("reactor commission certification exceeds issued revision for %s", id)
	}
	return revision, nil
}

func (k Keeper) setReactorCommissionRevision(ctx context.Context, id string, revision ReactorCommissionRevision) {
	value := make([]byte, 16)
	binary.BigEndian.PutUint64(value[:8], revision.Issued)
	binary.BigEndian.PutUint64(value[8:], revision.Certified)
	k.reactorCommissionRevisionStore(ctx).Set([]byte(id), value)
}

func (k Keeper) advanceReactorCommissionRevision(ctx context.Context, id string) error {
	revision, err := k.GetReactorCommissionRevision(ctx, id)
	if err != nil {
		return err
	}
	if revision.Issued == ^uint64(0) {
		// SDK Slash logs hook errors and continues. Exhaustion cannot be an
		// ordinary error while the last revision still looks certified.
		panic("reactor commission revision exhausted")
	}
	revision.Issued++
	k.setReactorCommissionRevision(ctx, id, revision)
	if err := k.emitReactorCommissionRevision(ctx, id, revision); err != nil {
		// SDK Slash ignores ordinary hook errors. Neither a missing gate nor
		// an unreported transition may be accepted as a successful slash.
		panic(err)
	}
	return nil
}

// certifyReactorCommission must run after the repair CurrentContext commits.
// An older pass can never certify or clear a newer transition's queue.
func (k Keeper) certifyReactorCommission(ctx context.Context, id string, repairedRevision uint64) bool {
	revision, err := k.GetReactorCommissionRevision(ctx, id)
	if err != nil || revision.Issued != repairedRevision {
		return false
	}
	if revision.Certified != revision.Issued {
		revision.Certified = revision.Issued
		if err := k.emitReactorCommissionRevision(ctx, id, revision); err != nil {
			k.logger.Error("Cannot publish reactor commission certification", "reactorId", id, "error", err)
			return false
		}
		k.setReactorCommissionRevision(ctx, id, revision)
	}
	return true
}
