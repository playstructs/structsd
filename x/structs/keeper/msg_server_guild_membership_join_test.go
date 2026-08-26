package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestMsgGuildMembershipJoin(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	gs := testCreateGuild(k, ctx)

	joinerAcc := sdk.AccAddress("join_test_addr_pad01")
	joiner := types.Player{
		Creator:        joinerAcc.String(),
		PrimaryAddress: joinerAcc.String(),
	}
	joiner = testAppendPlayer(k, ctx, joiner)

	t.Run("valid direct join", func(t *testing.T) {
		resp, err := ms.GuildMembershipJoin(wctx, &types.MsgGuildMembershipJoin{
			Creator:    joiner.Creator,
			GuildId:    gs.Guild.Id,
			PlayerId:   joiner.Id,
			InfusionId: []string{},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		p, found := k.GetPlayer(ctx, joiner.Id)
		require.True(t, found)
		require.Equal(t, gs.Guild.Id, p.GuildId)
	})

	t.Run("guild not found", func(t *testing.T) {
		_, err := ms.GuildMembershipJoin(wctx, &types.MsgGuildMembershipJoin{
			Creator:    joiner.Creator,
			GuildId:    "0-999",
			PlayerId:   joiner.Id,
			InfusionId: []string{},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("already a member", func(t *testing.T) {
		memberAcc := sdk.AccAddress("join_alrmemb_pad0001")
		member := types.Player{
			Creator:        memberAcc.String(),
			PrimaryAddress: memberAcc.String(),
			GuildId:        gs.Guild.Id,
		}
		member = testAppendPlayer(k, ctx, member)
		member.GuildId = gs.Guild.Id
		k.SetPlayer(ctx, member)

		_, err := ms.GuildMembershipJoin(wctx, &types.MsgGuildMembershipJoin{
			Creator:    member.Creator,
			GuildId:    gs.Guild.Id,
			PlayerId:   member.Id,
			InfusionId: []string{},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "already a member")
	})

	t.Run("unregistered creator", func(t *testing.T) {
		unregAcc := sdk.AccAddress("unreg_creator_pad000")
		_, err := ms.GuildMembershipJoin(wctx, &types.MsgGuildMembershipJoin{
			Creator:    unregAcc.String(),
			GuildId:    gs.Guild.Id,
			PlayerId:   joiner.Id,
			InfusionId: []string{},
		})
		require.Error(t, err)
	})
}

/* TestGuildMembershipJoinRejectsDuplicateInfusions is the regression on the
 * join minimum being cleared with fuel the player does not hold.
 *
 * msg.InfusionId is an unconstrained repeated field, and the branch taken when
 * the infusion's reactor is already inside the destination guild is pure
 * accumulation - it reads the record and adds its fuel, mutating nothing. So
 * naming one owned infusion N times counted its fuel N times, and
 * JoinInfusionMinimum is the only economic gate on a direct join.
 *
 * The two halves are asserted together deliberately: that one copy is genuinely
 * short is what makes the duplicate rejection meaningful rather than incidental.
 */
func TestGuildMembershipJoinRejectsDuplicateInfusions(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	gs := testCreateGuild(k, ctx)

	// The reactor has to be inside the guild, or the handler takes the migration
	// branch instead of the accumulate-only one this covers.
	reactor := gs.Reactor
	reactor.GuildId = gs.Guild.Id
	k.SetReactor(ctx, reactor)

	const infusionFuel = 600
	guild, found := k.GetGuild(ctx, gs.Guild.Id)
	require.True(t, found)
	// Deliberately more than one infusion's worth and less than two.
	guild.JoinInfusionMinimum = 1000
	k.SetGuild(ctx, guild)

	joinerAcc := sdk.AccAddress("dupe_join_addr_pad01")
	joiner := types.Player{
		Creator:        joinerAcc.String(),
		PrimaryAddress: joinerAcc.String(),
	}
	joiner = testAppendPlayer(k, ctx, joiner)
	require.NoError(t, k.SetPlayerIndexForAddress(ctx, joinerAcc.String(), joiner.Index))

	testAppendInfusion(k, ctx, types.Infusion{
		DestinationId:   reactor.Id,
		DestinationType: types.ObjectType_reactor,
		Address:         joinerAcc.String(),
		PlayerId:        joiner.Id,
		Fuel:            infusionFuel,
	})
	infusionId := reactor.Id + "-" + joinerAcc.String()

	t.Run("one infusion is genuinely short of the minimum", func(t *testing.T) {
		_, err := ms.GuildMembershipJoin(wctx, &types.MsgGuildMembershipJoin{
			Creator:    joiner.Creator,
			GuildId:    gs.Guild.Id,
			PlayerId:   joiner.Id,
			InfusionId: []string{infusionId},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "join infusion minimum not met")
	})

	t.Run("naming it twice must not clear the minimum", func(t *testing.T) {
		_, err := ms.GuildMembershipJoin(wctx, &types.MsgGuildMembershipJoin{
			Creator:    joiner.Creator,
			GuildId:    gs.Guild.Id,
			PlayerId:   joiner.Id,
			InfusionId: []string{infusionId, infusionId},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "listed more than once")

		p, playerFound := k.GetPlayer(ctx, joiner.Id)
		require.True(t, playerFound)
		require.Empty(t, p.GuildId, "the join must not have gone through")
	})

	t.Run("a duplicate anywhere in a longer list is caught", func(t *testing.T) {
		_, err := ms.GuildMembershipJoin(wctx, &types.MsgGuildMembershipJoin{
			Creator:    joiner.Creator,
			GuildId:    gs.Guild.Id,
			PlayerId:   joiner.Id,
			InfusionId: []string{infusionId, infusionId, infusionId},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "listed more than once")
	})

	// Enough genuine fuel still joins, so the fix rejects repetition rather than
	// the infusion list itself.
	t.Run("a single sufficient infusion still joins", func(t *testing.T) {
		guild, guildFound := k.GetGuild(ctx, gs.Guild.Id)
		require.True(t, guildFound)
		guild.JoinInfusionMinimum = infusionFuel
		k.SetGuild(ctx, guild)

		_, err := ms.GuildMembershipJoin(wctx, &types.MsgGuildMembershipJoin{
			Creator:    joiner.Creator,
			GuildId:    gs.Guild.Id,
			PlayerId:   joiner.Id,
			InfusionId: []string{infusionId},
		})
		require.NoError(t, err)

		p, playerFound := k.GetPlayer(ctx, joiner.Id)
		require.True(t, playerFound)
		require.Equal(t, gs.Guild.Id, p.GuildId)
	})
}

/* TestGuildMembershipJoinMigrationRequiresTokenMigratePermission is the
 * regression on a membership permission moving somebody's stake.
 *
 * When an infusion's reactor sits outside the destination guild, joining
 * redelegates that stake to the guild's validator. MsgGuildMembershipJoin names
 * only Creator as a signer, so the infusion's address never authorizes the move
 * - the handler is the whole authorization, and it demanded only
 * PermGuildMembership. ReactorBeginMigration gates the identical staking call
 * behind PermTokenMigrate.
 *
 * PermissionCheck's Layer 1 tests the signing key's own bits before the owner
 * shortcut, so the same check covers both shapes the report names: a restricted
 * associated address moving its own player's stake, and a caller granted
 * membership rights over somebody else moving theirs.
 */
func TestGuildMembershipJoinMigrationRequiresTokenMigratePermission(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	wctx := sdk.UnwrapSDKContext(ctx)

	gs := testCreateGuild(k, ctx)

	// A reactor in some other guild, so the join takes the migration branch.
	outsideAcc := sdk.AccAddress("outside_reactor_pad1")
	outsideValidator := sdk.ValAddress(outsideAcc.Bytes())
	outsideReactor := testAppendReactor(k, ctx, types.Reactor{
		RawAddress: outsideValidator.Bytes(),
		Validator:  outsideValidator.String(),
		GuildId:    "0-999",
	})
	testAddValidator(k, outsideValidator, math.NewInt(10_000))

	guild, found := k.GetGuild(ctx, gs.Guild.Id)
	require.True(t, found)
	guild.JoinInfusionMinimum = 100
	k.SetGuild(ctx, guild)

	joinerAcc := sdk.AccAddress("migrate_join_pad0001")
	joiner := types.Player{
		Creator:        joinerAcc.String(),
		PrimaryAddress: joinerAcc.String(),
	}
	joiner = testAppendPlayer(k, ctx, joiner)
	require.NoError(t, k.SetPlayerIndexForAddress(ctx, joinerAcc.String(), joiner.Index))

	testAppendInfusion(k, ctx, types.Infusion{
		DestinationId:   outsideReactor.Id,
		DestinationType: types.ObjectType_reactor,
		Address:         joinerAcc.String(),
		PlayerId:        joiner.Id,
		Fuel:            500,
	})
	infusionId := outsideReactor.Id + "-" + joinerAcc.String()

	// The stake behind the infusion has to actually exist for the redelegation
	// to be validated; the mock fires no hooks, so this is the delegation the
	// handler would find on a live chain.
	mock := k.StakingKeeper().(*keepertest.MockStakingKeeper)
	require.NoError(t, mock.SetDelegation(ctx, stakingtypes.Delegation{
		DelegatorAddress: joinerAcc.String(),
		ValidatorAddress: outsideValidator.String(),
		Shares:           math.LegacyNewDecFromInt(math.NewInt(500)),
	}))

	// A secondary key of the joiner, deliberately holding membership rights and
	// nothing that authorizes moving tokens.
	limitedAcc := sdk.AccAddress("migrate_limited_pad1")
	require.NoError(t, k.SetPlayerIndexForAddress(ctx, limitedAcc.String(), joiner.Index))
	testPermissionAdd(k, ctx, keeperlib.GetAddressPermissionIDBytes(limitedAcc.String()), types.PermGuildMembership)

	t.Run("membership permission alone cannot migrate the stake", func(t *testing.T) {
		_, err := ms.GuildMembershipJoin(wctx, &types.MsgGuildMembershipJoin{
			Creator:    limitedAcc.String(),
			GuildId:    gs.Guild.Id,
			PlayerId:   joiner.Id,
			InfusionId: []string{infusionId},
		})
		require.Error(t, err, "a key without PermTokenMigrate must not redelegate this stake")
		require.Contains(t, err.Error(), "permission")

		p, playerFound := k.GetPlayer(ctx, joiner.Id)
		require.True(t, playerFound)
		require.Empty(t, p.GuildId, "and the join must not have gone through either")
	})

	t.Run("adding the token migrate bit is what unblocks it", func(t *testing.T) {
		testPermissionAdd(k, ctx, keeperlib.GetAddressPermissionIDBytes(limitedAcc.String()), types.PermTokenMigrate)

		_, err := ms.GuildMembershipJoin(wctx, &types.MsgGuildMembershipJoin{
			Creator:    limitedAcc.String(),
			GuildId:    gs.Guild.Id,
			PlayerId:   joiner.Id,
			InfusionId: []string{infusionId},
		})
		require.NoError(t, err)

		p, playerFound := k.GetPlayer(ctx, joiner.Id)
		require.True(t, playerFound)
		require.Equal(t, gs.Guild.Id, p.GuildId)
	})
}
