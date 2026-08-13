package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"structs/x/structs/types"
)

// TestMsgGuildCreate is the handler's basic shape. The charter mechanics — the
// proof, the anchor, the founder consent, the reactor entitlement and the
// membership guard — live in guild_charter_test.go.
func TestMsgGuildCreate(t *testing.T) {
	testCases := []struct {
		name      string
		endpoint  string
		reactorId string
		expErrMsg string
	}{
		{
			name:     "valid guild creation",
			endpoint: "test-endpoint",
		},
		{
			// The reactor is still required on both paths, and not merely as a
			// permission subject: GuildMembershipJoin hard-fails on a guild whose
			// primary reactor does not resolve, so a guild without one could
			// never be joined.
			name:      "missing reactor id",
			endpoint:  "test-endpoint",
			reactorId: "nonexistent",
			expErrMsg: "reactor",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newCharterFixture(t)
			f.advanceTo(int64(charterTestReactorAge) + 2)

			reactorId := f.reactor.Id
			if tc.reactorId != "" {
				reactorId = tc.reactorId
			}

			resp, err := f.ms.GuildCreate(f.ctx, &types.MsgGuildCreate{
				Creator:   f.player.Creator,
				ReactorId: reactorId,
				Endpoint:  tc.endpoint,
			})

			if tc.expErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.NotEmpty(t, resp.GuildId)

			guild, found := f.k.GetGuild(f.ctx, resp.GuildId)
			require.True(t, found)
			require.Equal(t, tc.endpoint, guild.Endpoint)
			require.Equal(t, f.reactor.Id, guild.PrimaryReactorId)
		})
	}
}
