package keeper_test

import (
	"testing"

	keepertest "structs/testutil/keeper"

	"github.com/stretchr/testify/require"
)

func TestIncrementPlayerMsgCount(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	count := k.IncrementPlayerMsgCount(ctx, "1-1", 5)
	require.Equal(t, uint64(5), count)

	count = k.IncrementPlayerMsgCount(ctx, "1-1", 3)
	require.Equal(t, uint64(8), count)

	// Different player is independent
	count = k.IncrementPlayerMsgCount(ctx, "1-2", 10)
	require.Equal(t, uint64(10), count)

	// Original player unchanged
	count = k.GetPlayerMsgCount(ctx, "1-1")
	require.Equal(t, uint64(8), count)
}

func TestGetPlayerMsgCountDefault(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	count := k.GetPlayerMsgCount(ctx, "1-99")
	require.Equal(t, uint64(0), count)
}

func TestThrottleKey(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	require.False(t, k.HasThrottleKey(ctx, "fleet/2-1"))

	k.SetThrottleKey(ctx, "fleet/2-1")
	require.True(t, k.HasThrottleKey(ctx, "fleet/2-1"))

	// Different key is independent
	require.False(t, k.HasThrottleKey(ctx, "fleet/2-2"))
}

func TestThrottleKeyMultiple(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	k.SetThrottleKey(ctx, "proof/5-1")
	k.SetThrottleKey(ctx, "proof/5-2")
	k.SetThrottleKey(ctx, "explore/1-3")

	require.True(t, k.HasThrottleKey(ctx, "proof/5-1"))
	require.True(t, k.HasThrottleKey(ctx, "proof/5-2"))
	require.True(t, k.HasThrottleKey(ctx, "explore/1-3"))
	require.False(t, k.HasThrottleKey(ctx, "proof/5-3"))
	require.False(t, k.HasThrottleKey(ctx, "register/1-1"))
}

func TestSetThrottleKeyIdempotent(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	k.SetThrottleKey(ctx, "fleet/2-5")
	k.SetThrottleKey(ctx, "fleet/2-5")
	require.True(t, k.HasThrottleKey(ctx, "fleet/2-5"))
}
