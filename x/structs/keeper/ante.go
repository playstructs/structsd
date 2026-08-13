package keeper

import (
	"context"
	"encoding/binary"

	"structs/x/structs/types"
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
//
// The caller must have established that the signer is authorized to act on the
// object the key names — see ThrottleTargetAuthorized. An ante write survives a
// failed message, so an unauthorized reservation is not undone by the handler
// rejecting it.
func (k Keeper) SetThrottleKey(ctx context.Context, throttleKey string) {
	// Defensive: unreachable since NewKeeper panics if transientStoreService is nil
	if k.transientStoreService == nil {
		return
	}
	tStore := k.transientStoreService.OpenTransientStore(ctx)
	tStore.Set([]byte(anteThrottlePrefix+throttleKey), []byte{0x01})
}

// ThrottleTargetAuthorized reports whether creator may act on targetId under
// perm. It mirrors the target check the handler performs, so an unauthorized
// signer cannot reserve an object-global throttle key it could never
// legitimately consume.
//
// Every throttled handler authorizes against a *player* — the owner of the
// struct or fleet, or the named player itself — so resolving the target to an
// owner PlayerCache and calling the handlers' own PermissionCheck reproduces
// all seven checks without restating any policy. Owner equality alone would not
// do: a player can delegate a hash bit to a second account, and that account's
// transactions would then never reserve.
//
// Read-only by construction. Caches only reach state through CommitAll, which
// this never calls — which matters more here than in a handler, because ante
// writes are committed even when the message goes on to fail.
//
// A false answer only skips the reservation; the handler still runs and still
// produces the real error. That is deliberate. The ante sees pre-transaction
// state while the handler sees state left by earlier messages in the same
// transaction, so a transaction that grants a permission and then uses it is
// authorized there and not here. Rejecting on this answer would break it.
func (k Keeper) ThrottleTargetAuthorized(ctx context.Context, creator string, kind types.ObjectType, targetId string, perm types.Permission) bool {
	if creator == "" || targetId == "" {
		return false
	}

	cc := k.NewCurrentContext(ctx)

	signer, err := cc.GetSigningPlayer(creator)
	if err != nil {
		return false
	}

	var owner *PlayerCache

	switch kind {
	case types.ObjectType_player:
		// A PlayerCache is its own owner, so a self-targeted explore takes the
		// owner shortcut inside PermissionCheck. A target that does not exist
		// needs no existence check of its own: it owns nothing and holds no
		// permission row, so the check below refuses it, which declines the
		// reservation without rejecting the transaction.
		owner = cc.GetPlayer(targetId)

	case types.ObjectType_struct:
		structure := cc.GetStruct(targetId)
		if !structure.LoadStruct() {
			return false
		}
		owner = structure.GetOwner()

	case types.ObjectType_fleet:
		// The malformed-id path returns a zero FleetCache with a nil CC, so
		// GetOwner would panic on it. The id here is caller-supplied.
		fleet, fleetErr := cc.GetFleetById(targetId)
		if fleetErr != nil {
			return false
		}
		owner = fleet.GetOwner()

	case types.ObjectType_address:
		// A signer-scoped throttle key (see SignerScopedThrottleMessages). The
		// target is the signing address itself, so there is no third party to
		// resolve and the check collapses to "is this a registered player" —
		// which the shared PermissionCheck below still performs rather than
		// being assumed, so an unregistered signer reserves nothing.
		if targetId != creator {
			return false
		}
		owner = signer

	default:
		return false
	}

	if owner == nil {
		return false
	}

	return cc.PermissionCheck(owner, signer, perm) == nil
}
