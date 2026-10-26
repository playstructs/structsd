package keeper

import (
	"context"
	"math/big"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

var reactorDiversityPositionPrefix = []byte("ReactorDiversity/position/")
var ownerReactorDiversityPrefix = []byte("ReactorDiversity/owner/")
var reactorDiversityReadyKey = []byte("ReactorDiversity/ready")

// The marker is reconstruction state, not an economic switch. Legacy upgrade
// phases must finish their canonical base repair before delta tracking starts.
// Genesis rebuilds it; it does not need an exported configuration field.
func (k Keeper) ReactorDiversityReady(ctx context.Context) bool {
	return runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)).Has(reactorDiversityReadyKey)
}

func (k Keeper) SuspendReactorDiversity(ctx context.Context) {
	runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)).Delete(reactorDiversityReadyKey)
}

func reactorDiversityPositionKey(owner, reactor string) []byte {
	return append(append([]byte{}, reactorDiversityPositionPrefix...), []byte(owner+"/"+reactor)...)
}

func ownerReactorDiversityKey(owner string) []byte {
	return append(append([]byte{}, ownerReactorDiversityPrefix...), []byte(owner)...)
}

func (k Keeper) GetReactorDiversityPosition(ctx context.Context, owner, reactor string) (row types.ReactorDiversityPosition, found bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	data := store.Get(reactorDiversityPositionKey(owner, reactor))
	if data == nil {
		return row, false
	}
	k.cdc.MustUnmarshal(data, &row)
	return row, true
}

func (k Keeper) GetOwnerReactorDiversity(ctx context.Context, owner string) (row types.OwnerReactorDiversity, found bool) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	data := store.Get(ownerReactorDiversityKey(owner))
	if data == nil {
		return row, false
	}
	k.cdc.MustUnmarshal(data, &row)
	return row, true
}

// Only the canonical CurrentContext caches write these derived records. Normal
// execution touches one owner/position key; reconstruction scans belong to
// genesis, upgrades, and the explicit invariant.
func (k Keeper) setReactorDiversityPosition(ctx context.Context, row types.ReactorDiversityPosition) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	key := reactorDiversityPositionKey(row.OwnerId, row.ReactorId)
	if row.Power == "0" {
		store.Delete(key)
	} else {
		store.Set(key, k.cdc.MustMarshal(&row))
	}
}

func (k Keeper) setOwnerReactorDiversity(ctx context.Context, row types.OwnerReactorDiversity) {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	key := ownerReactorDiversityKey(row.OwnerId)
	if row.Power == "0" && row.SquaredPower == "0" && row.AppliedBonus == 0 {
		store.Delete(key)
	} else {
		store.Set(key, k.cdc.MustMarshal(&row))
	}
	_ = sdk.UnwrapSDKContext(ctx).EventManager().EmitTypedEvent(&types.EventOwnerReactorDiversity{Diversity: &row})
}

// Store values are written only by the derived caches using big.Int.String.
// Treat malformed persisted state like other malformed module protobuf rows;
// never silently replace it with spendable capacity.
func mustDiversityInteger(value string) *big.Int {
	n, ok := new(big.Int).SetString(value, 10)
	if !ok || n.Sign() < 0 || n.String() != value {
		panic("invalid stored reactor diversity integer")
	}
	return n
}
