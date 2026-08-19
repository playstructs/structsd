package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"structs/x/structs/types"
)

// TestGuildJoinBypassLevelIsValid covers the classifier the write paths gate on.
// proto3 enums are open — the generated decoder shifts bytes into an int32 and
// never consults the enum — so this is what separates a level the chain has
// policy for from an arbitrary number a transaction supplied.
func TestGuildJoinBypassLevelIsValid(t *testing.T) {
	for name, value := range types.GuildJoinBypassLevel_value {
		require.True(t, types.GuildJoinBypassLevel(value).IsValid(),
			"%s (%d) is declared in the proto and must be accepted", name, value)
	}

	for _, level := range []types.GuildJoinBypassLevel{-1, 3, 500, 1 << 30} {
		require.False(t, level.IsValid(), "level %d is not declared and must be refused", int32(level))
	}
}

// TestNormalizeJoinBypassLevels covers the migration's repair step. Clamping to
// closed is the recoverable direction: CanUpdateJoinConstraintsBy reads only the
// permission bit, so the owner reopens the guild in one transaction.
func TestNormalizeJoinBypassLevels(t *testing.T) {
	testCases := []struct {
		name            string
		byRequest       types.GuildJoinBypassLevel
		byInvite        types.GuildJoinBypassLevel
		expectChanged   bool
		expectByRequest types.GuildJoinBypassLevel
		expectByInvite  types.GuildJoinBypassLevel
	}{
		{
			name:            "declared levels are left alone",
			byRequest:       types.GuildJoinBypassLevel_member,
			byInvite:        types.GuildJoinBypassLevel_permissioned,
			expectChanged:   false,
			expectByRequest: types.GuildJoinBypassLevel_member,
			expectByInvite:  types.GuildJoinBypassLevel_permissioned,
		},
		{
			name:            "undeclared byRequest is clamped",
			byRequest:       500,
			byInvite:        types.GuildJoinBypassLevel_member,
			expectChanged:   true,
			expectByRequest: types.GuildJoinBypassLevel_closed,
			expectByInvite:  types.GuildJoinBypassLevel_member,
		},
		{
			name:            "both undeclared are clamped",
			byRequest:       500,
			byInvite:        -1,
			expectChanged:   true,
			expectByRequest: types.GuildJoinBypassLevel_closed,
			expectByInvite:  types.GuildJoinBypassLevel_closed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			guild := types.Guild{
				JoinInfusionMinimumBypassByRequest: tc.byRequest,
				JoinInfusionMinimumBypassByInvite:  tc.byInvite,
			}

			require.Equal(t, tc.expectChanged, guild.NormalizeJoinBypassLevels())
			require.Equal(t, tc.expectByRequest, guild.JoinInfusionMinimumBypassByRequest)
			require.Equal(t, tc.expectByInvite, guild.JoinInfusionMinimumBypassByInvite)

			// Idempotent: closed is itself a declared value, so replaying the
			// upgrade block produces identical rows.
			require.False(t, guild.NormalizeJoinBypassLevels(), "a second pass must be a no-op")
		})
	}
}
