package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Failed positions use a deduplicated FIFO, independent of the finite main
// repair interval. A failed retry goes to the tail, so neither a bad first row
// nor new requests can indefinitely hold up rows already waiting behind it.
const reactorRepairRetryPrefix = "Reactor/repairRetry/"

type reactorRepairRetry struct {
	address string
	// Hook retries can discover a delegation whose infusion was never written.
	// Mode 2 also permits the original native-delegation auto-registration.
	mode byte
}

func (k Keeper) reactorRepairRetryStore(ctx context.Context, suffix string) prefix.Store {
	return prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx)), []byte(reactorRepairRetryPrefix+suffix))
}

func (k Keeper) enqueueReactorRepairRetry(ctx context.Context, reactorID, address string, mode byte) {
	reverse := k.reactorRepairRetryStore(ctx, "address/"+reactorID+"/")
	entries := k.reactorRepairRetryStore(ctx, "entry/"+reactorID+"/")
	if key := reverse.Get([]byte(address)); key != nil {
		value := entries.Get(key)
		if len(value) == 0 {
			panic("missing reactor repair retry")
		}
		if mode > value[0] {
			value[0] = mode
			entries.Set(key, value)
		}
		return
	}
	ends := k.reactorRepairRetryStore(ctx, "end/")
	sequence := uint64(0)
	if value := ends.Get([]byte(reactorID)); value != nil {
		if len(value) != 8 {
			panic("invalid reactor repair retry sequence")
		}
		sequence = binary.BigEndian.Uint64(value)
	}
	if sequence == ^uint64(0) {
		panic("reactor repair retry sequence exhausted")
	}
	key := infusionOrdinalBytes(sequence + 1)
	entries.Set(key, append([]byte{mode}, []byte(address)...))
	reverse.Set([]byte(address), key)
	ends.Set([]byte(reactorID), key)
}

// EnqueueReactorInfusionRetry preserves an in-flight repair cursor. Repeated
// failed hooks update one obligation instead of restarting every delegator.
func (k Keeper) EnqueueReactorInfusionRetry(ctx context.Context, reactorID string, address sdk.AccAddress) {
	mode := byte(1)
	if _, exists := k.GetInfusion(ctx, reactorID, address.String()); exists {
		mode = 0
	} else if _, vault := k.GetVaultOwner(ctx, address); !vault && k.GetPlayerIndexFromAddress(ctx, address.String()) == 0 {
		mode = 2
	}
	k.enqueueReactorRepairRetry(ctx, reactorID, address.String(), mode)
	k.EnqueueReactorRestart(ctx, reactorID)
}

func (k Keeper) readReactorRepairRetries(ctx context.Context, reactorID string, limit int) ([]reactorRepairRetry, error) {
	iterator := k.reactorRepairRetryStore(ctx, "entry/"+reactorID+"/").Iterator(nil, nil)
	defer iterator.Close()
	var batch []reactorRepairRetry
	for ; iterator.Valid() && len(batch) < limit; iterator.Next() {
		value := iterator.Value()
		if len(iterator.Key()) != 8 || len(value) < 2 || value[0] > 2 {
			return nil, fmt.Errorf("invalid reactor repair retry for %s", reactorID)
		}
		batch = append(batch, reactorRepairRetry{address: string(value[1:]), mode: value[0]})
	}
	return batch, nil
}

func (k Keeper) removeReactorRepairRetry(ctx context.Context, reactorID, address string) {
	reverse := k.reactorRepairRetryStore(ctx, "address/"+reactorID+"/")
	if key := reverse.Get([]byte(address)); key != nil {
		k.reactorRepairRetryStore(ctx, "entry/"+reactorID+"/").Delete(key)
		reverse.Delete([]byte(address))
	}
}

func (k Keeper) hasReactorRepairRetries(ctx context.Context, reactorID string) bool {
	iterator := k.reactorRepairRetryStore(ctx, "entry/"+reactorID+"/").Iterator(nil, nil)
	defer iterator.Close()
	return iterator.Valid()
}
