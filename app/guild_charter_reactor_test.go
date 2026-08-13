package app_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	"structs/app"
	structskeeper "structs/x/structs/keeper"
	structstypes "structs/x/structs/types"
)

/* TestGuildCharterReactorPathAgainstRealStaking is the release gate for the
 * proof-free route to founding a guild.
 *
 * The whole reason that route is safe to offer is that it costs a bonded
 * validator: ReactorInitialize runs from AfterValidatorCreated, so a reactor
 * exists for every validator ever created, and MsgCreateValidator picks its own
 * min_self_delegation — a validator that never joins the active set is nearly
 * free. Requiring *currently bonded* is what caps the free supply of guilds at
 * the size of the active set instead of at the number of throwaway identities an
 * attacker cares to register.
 *
 * The keeper suite drives that gate through testutil's MockStakingKeeper, which
 * returns whatever validator the test handed it. That proves our comparison and
 * says nothing about whether real staking ever reports these states the way we
 * read them, which is exactly the gap this test exists to close: the same
 * validator is refused before staking bonds it and accepted after, with nothing
 * between the two attempts but staking's own validator-set update.
 */
func TestGuildCharterReactorPathAgainstRealStaking(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)

	// The production eligibility delay is a month of blocks. Shrink it before the
	// validator exists, because AppendReactor stamps the height off the param at
	// creation time and never revisits it.
	params := bApp.StructsKeeper.GetParams(ctx)
	params.GuildCharterReactorAge = 2
	require.NoError(t, bApp.StructsKeeper.SetParams(ctx, params))

	bondDenom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)

	operatorAcc := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
	stake := math.NewInt(10_000_000)
	funding := sdk.NewCoins(sdk.NewCoin(bondDenom, stake.MulRaw(2)))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, structstypes.ModuleName, funding))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, structstypes.ModuleName, operatorAcc, funding))

	valAddr := sdk.ValAddress(operatorAcc)
	createMsg, err := stakingtypes.NewMsgCreateValidator(
		valAddr.String(),
		ed25519.GenPrivKey().PubKey(),
		sdk.NewCoin(bondDenom, stake),
		stakingtypes.NewDescription("charter", "", "", "", ""),
		stakingtypes.NewCommissionRates(
			math.LegacyNewDecWithPrec(1, 1),
			math.LegacyNewDecWithPrec(2, 1),
			math.LegacyNewDecWithPrec(1, 2),
		),
		math.OneInt(),
	)
	require.NoError(t, err)

	stakingMsgServer := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	_, err = stakingMsgServer.CreateValidator(ctx, createMsg)
	require.NoError(t, err)

	reactorId := jailGateReactorId(t, bApp, ctx, valAddr)

	reactor, found := bApp.StructsKeeper.GetReactor(ctx, reactorId)
	require.True(t, found)
	require.NotZero(t, reactor.GuildCharterEligibleHeight, "AppendReactor must stamp an eligibility height, since zero is read as never eligible")

	// ReactorInitialize upserts a player for the operator's account address and
	// grants it PermReactorAll, which carries the reactor permission the free
	// path needs.
	operatorPlayerId := charterPlayerId(t, bApp, ctx, operatorAcc.String())

	structsMsgServer := structskeeper.NewMsgServerImpl(bApp.StructsKeeper)

	// Before the eligibility height, whatever staking thinks of the validator.
	_, err = structsMsgServer.GuildCreate(ctx, &structstypes.MsgGuildCreate{
		Creator:   operatorAcc.String(),
		ReactorId: reactorId,
		Endpoint:  "too-soon.energy",
	})
	require.ErrorContains(t, err, "not_yet_eligible")

	ctx = ctx.WithBlockHeight(int64(reactor.GuildCharterEligibleHeight))

	/* A validator that has been created but never bonded. This is the cheap
	 * identity the gate exists to refuse, and staking reports it as Unbonded
	 * until ApplyAndReturnValidatorSetUpdates runs.
	 */
	created, err := bApp.StakingKeeper.GetValidator(ctx, valAddr)
	require.NoError(t, err)
	require.False(t, created.IsBonded(), "a freshly created validator should not be bonded yet")

	_, err = structsMsgServer.GuildCreate(ctx, &structstypes.MsgGuildCreate{
		Creator:   operatorAcc.String(),
		ReactorId: reactorId,
		Endpoint:  "unbonded.energy",
	})
	require.ErrorContains(t, err, "validator_inactive")

	anchorBefore := bApp.StructsKeeper.CharterAnchor(ctx)

	// This is the call staking makes in its own EndBlock.
	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)

	bonded, err := bApp.StakingKeeper.GetValidator(ctx, valAddr)
	require.NoError(t, err)
	require.True(t, bonded.IsBonded(), "the validator should be bonded after the set update")

	response, err := structsMsgServer.GuildCreate(ctx, &structstypes.MsgGuildCreate{
		Creator:   operatorAcc.String(),
		ReactorId: reactorId,
		Endpoint:  "bonded.energy",
	})
	require.NoError(t, err, "a bonded validator past its eligibility height may found a guild without a proof")

	guild, found := bApp.StructsKeeper.GetGuild(ctx, response.GuildId)
	require.True(t, found)
	require.Equal(t, operatorPlayerId, guild.Owner)
	require.Empty(t, guild.CharterSolverId, "the free path solves nothing, so it credits nobody")

	/* The two routes must not interfere. Wiping every mining pool's in-flight
	 * work because a validator collected its perk would make the free path an
	 * attack on the paid one.
	 */
	require.Equal(t, anchorBefore, bApp.StructsKeeper.CharterAnchor(ctx),
		"the reactor path must leave the charter puzzle alone")

	// GuildId is what spends the entitlement, so it has to be written here.
	reactor, found = bApp.StructsKeeper.GetReactor(ctx, reactorId)
	require.True(t, found)
	require.Equal(t, response.GuildId, reactor.GuildId)

	/* And the entitlement is one per reactor. A second player is needed to show
	 * it, because the operator is now an owner and would be refused for that
	 * instead — which is the ordering that makes this worth asserting.
	 */
	otherAcc := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
	cc := bApp.StructsKeeper.NewCurrentContext(ctx)
	cc.UpsertPlayer(otherAcc.String())
	cc.CommitAll()

	permissionId := structskeeper.GetObjectPermissionIDBytes(reactorId, charterPlayerId(t, bApp, ctx, otherAcc.String()))
	bApp.StructsKeeper.SetPermissionsByBytes(ctx, permissionId, structstypes.PermReactorAll)

	_, err = structsMsgServer.GuildCreate(ctx, &structstypes.MsgGuildCreate{
		Creator:   otherAcc.String(),
		ReactorId: reactorId,
		Endpoint:  "seconds.energy",
	})
	require.ErrorContains(t, err, "entitlement_spent")
}

func charterPlayerId(t *testing.T, bApp *app.App, ctx sdk.Context, address string) string {
	t.Helper()

	playerIndex := bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, address)
	require.NotZero(t, playerIndex, "address %s should be indexed to a player", address)

	player, found := bApp.StructsKeeper.GetPlayerFromIndex(ctx, playerIndex)
	require.True(t, found)
	return player.Id
}
