package keeper

import (
	"context"
	"encoding/binary"
)

const (
	anteThrottlePrefix = "ante/throttle/"
	anteMsgCountPrefix = "ante/msgcount/"
)

// HasTransientStore reports whether the keeper was initialized with a
// transient store. NewKeeper panics if the transient store is nil, so this
// should always return true in production.
func (k Keeper) HasTransientStore() bool {
	return k.transientStoreService != nil
}

// IncrementPlayerMsgCount atomically reads, adds delta, and writes the
// per-player message count in the transient store. Returns the new total.
func (k Keeper) IncrementPlayerMsgCount(ctx context.Context, playerId string, delta uint64) uint64 {
	// Defensive: unreachable since NewKeeper panics if transientStoreService is nil
	if k.transientStoreService == nil {
		return 0
	}
	tStore := k.transientStoreService.OpenTransientStore(ctx)
	key := []byte(anteMsgCountPrefix + playerId)

	var current uint64
	bz, err := tStore.Get(key)
	if err == nil && bz != nil {
		current = binary.BigEndian.Uint64(bz)
	}

	newCount := current + delta
	out := make([]byte, 8)
	binary.BigEndian.PutUint64(out, newCount)
	tStore.Set(key, out)

	return newCount
}

// GetPlayerMsgCount reads the current per-player message count from the
// transient store without modifying it.
func (k Keeper) GetPlayerMsgCount(ctx context.Context, playerId string) uint64 {
	// Defensive: unreachable since NewKeeper panics if transientStoreService is nil
	if k.transientStoreService == nil {
		return 0
	}
	tStore := k.transientStoreService.OpenTransientStore(ctx)
	key := []byte(anteMsgCountPrefix + playerId)

	bz, err := tStore.Get(key)
	if err != nil || bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// HasThrottleKey checks whether a throttle key exists in the transient store.
func (k Keeper) HasThrottleKey(ctx context.Context, throttleKey string) bool {
	// Defensive: unreachable since NewKeeper panics if transientStoreService is nil
	if k.transientStoreService == nil {
		return false
	}
	tStore := k.transientStoreService.OpenTransientStore(ctx)
	has, err := tStore.Has([]byte(anteThrottlePrefix + throttleKey))
	return err == nil && has
}

// SetThrottleKey writes a throttle key to the transient store. The value is
// a single byte (presence marker). Auto-clears at block boundary.
func (k Keeper) SetThrottleKey(ctx context.Context, throttleKey string) {
	// Defensive: unreachable since NewKeeper panics if transientStoreService is nil
	if k.transientStoreService == nil {
		return
	}
	tStore := k.transientStoreService.OpenTransientStore(ctx)
	tStore.Set([]byte(anteThrottlePrefix+throttleKey), []byte{0x01})
}
