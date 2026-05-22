package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
	"structs/x/structs/types"
)

// TestMsgGuildUpdatePrimaryReactor exercises the recovery handler that lets
// guild admins rotate to a new primary reactor when their original validator
// has been retired or jailed. See msg_server_guild_update_primary_reactor.go
// for authorization and validation rationale.
func TestMsgGuildUpdatePrimaryReactor(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	setup := testCreateGuild(k, ctx)
	owner := setup.GuildOwner
	guild := setup.Guild

	// Build a second, healthy reactor + validator that the guild can rotate to.
	rescueAcc := sdk.AccAddress("rescue_padding_____________________01")
	rescueValAddr := sdk.ValAddress(rescueAcc.Bytes())
	testAddValidator(k, rescueValAddr, math.NewInt(2_000_000))
	rescueReactor := testAppendReactor(k, ctx, types.Reactor{
		Validator:  rescueValAddr.String(),
		RawAddress: rescueValAddr.Bytes(),
	})

	// Build a jailed validator + reactor pair so we can assert the jailed-
	// validator rejection path.
	jailedAcc := sdk.AccAddress("jailed_padding_____________________02")
	jailedValAddr := sdk.ValAddress(jailedAcc.Bytes())
	testAddValidator(k, jailedValAddr, math.NewInt(2_000_000))
	mock := k.StakingKeeper().(*keepertest.MockStakingKeeper)
	mock.JailValidator(jailedValAddr)
	jailedReactor := testAppendReactor(k, ctx, types.Reactor{
		Validator:  jailedValAddr.String(),
		RawAddress: jailedValAddr.Bytes(),
	})

	t.Run("happy path: owner rotates to healthy reactor", func(t *testing.T) {
		resp, err := ms.GuildUpdatePrimaryReactor(wctx, &types.MsgGuildUpdatePrimaryReactor{
			Creator:   owner.Creator,
			GuildId:   guild.Id,
			ReactorId: rescueReactor.Id,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		updated, found := k.GetGuild(ctx, guild.Id)
		require.True(t, found)
		require.Equal(t, rescueReactor.Id, updated.PrimaryReactorId)
	})

	t.Run("rejects unknown reactor", func(t *testing.T) {
		_, err := ms.GuildUpdatePrimaryReactor(wctx, &types.MsgGuildUpdatePrimaryReactor{
			Creator:   owner.Creator,
			GuildId:   guild.Id,
			ReactorId: "reactor-does-not-exist",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("rejects jailed validator", func(t *testing.T) {
		_, err := ms.GuildUpdatePrimaryReactor(wctx, &types.MsgGuildUpdatePrimaryReactor{
			Creator:   owner.Creator,
			GuildId:   guild.Id,
			ReactorId: jailedReactor.Id,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "jailed")
	})

	t.Run("rejects validator missing from staking", func(t *testing.T) {
		// Build a reactor whose validator is not registered with x/staking.
		ghostAcc := sdk.AccAddress("ghost_padding______________________03")
		ghostValAddr := sdk.ValAddress(ghostAcc.Bytes())
		ghostReactor := testAppendReactor(k, ctx, types.Reactor{
			Validator:  ghostValAddr.String(),
			RawAddress: ghostValAddr.Bytes(),
		})

		_, err := ms.GuildUpdatePrimaryReactor(wctx, &types.MsgGuildUpdatePrimaryReactor{
			Creator:   owner.Creator,
			GuildId:   guild.Id,
			ReactorId: ghostReactor.Id,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("rejects caller without PermAdmin", func(t *testing.T) {
		strangerAcc := sdk.AccAddress("strangr_padding____________________04")
		stranger := types.Player{Creator: strangerAcc.String(), PrimaryAddress: strangerAcc.String()}
		stranger = testAppendPlayer(k, ctx, stranger)

		_, err := ms.GuildUpdatePrimaryReactor(wctx, &types.MsgGuildUpdatePrimaryReactor{
			Creator:   stranger.Creator,
			GuildId:   guild.Id,
			ReactorId: rescueReactor.Id,
		})
		require.Error(t, err)
	})
}
