package ante

import (
	"context"

	"structs/x/structs/types"
)

// StructsAnteKeeper defines the narrow interface the ante decorators need
// from the Structs module keeper. Using an interface avoids importing the
// keeper package directly and keeps the ante package testable.
type StructsAnteKeeper interface {
	// Main KV store reads (used for permission and charge checks)
	GetPlayerIndexFromAddress(ctx context.Context, address string) uint64
	GetPermissionsByBytes(ctx context.Context, permissionId []byte) types.Permission
	GetGridAttribute(ctx context.Context, gridAttributeId string) uint64

	// Target-object authorization, mirroring the check the handler makes. Read
	// only: it loads objects but never commits. The throttle uses it to decide
	// whether a signer may reserve an object-global key, which the address-level
	// PermissionMap check cannot answer.
	ThrottleTargetAuthorized(ctx context.Context, creator string, kind types.ObjectType, targetId string, perm types.Permission) bool

	// IsPlayerPrimaryAddress reports whether an address is the player's primary
	// one. Only the recovery-message exemption asks, so this costs a read on
	// that path alone.
	IsPlayerPrimaryAddress(ctx context.Context, playerId string, address string) bool

	// Transient store availability check
	HasTransientStore() bool

	// Transient store operations (per-block throttling).
	// All are safe to call even when HasTransientStore() is false — they
	// degrade gracefully (counts return 0, throttle keys return false).
	IncrementPlayerMsgCount(ctx context.Context, playerId string, delta uint64) uint64
	GetPlayerMsgCount(ctx context.Context, playerId string) uint64
	HasThrottleKey(ctx context.Context, throttleKey string) bool
	SetThrottleKey(ctx context.Context, throttleKey string)
}
