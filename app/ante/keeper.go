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
