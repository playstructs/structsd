package app_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	"structs/app"
	structskeeper "structs/x/structs/keeper"
	structstypes "structs/x/structs/types"
)

// These are the release gates for the delegation transfer, and they have to run
// against real staking and real distribution for the same reason the
// redelegation gate does: the mock keepers fire no hooks, so they can prove the
// sequence we intend and nothing about whether the SDK accepts it.
//
// What the mock structurally cannot see is that a delegation moved with a bare
// SetDelegation leaves no DelegatorStartingInfo at the destination, which is not
// an error at the time and makes every later withdraw, undelegate and
// redelegate fail forever after. That failure is the assertion below.
//
// If these fail after an SDK upgrade, look first at whether the distribution
// hook contract around initializeDelegation has changed, and second at whether
// staking has grown a real delegation-transfer primitive worth using instead.

// TestDelegationTransferMergesAndKeepsDistributionUsable is the collision case:
// both of a player's addresses delegate to the same validator, so the sweep has
// to merge rather than overwrite. Overwriting loses the destination's own
// shares while Validator.DelegatorShares keeps counting them, which is
// unrecoverable once it happens -- hence the invariant checks at the end.
func TestDelegationTransferMergesAndKeepsDistributionUsable(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)

	bondDenom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)

	stakingMsgServer := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	valAddr := createRedelegationValidator(t, bApp, ctx, stakingMsgServer, bondDenom, "delegationmerge")

	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)

	primaryAcc, secondaryAcc, playerId := newTransferPlayer(t, bApp, ctx)

	// Both addresses delegate to the same validator. Before the merge the
	// second write simply replaced the first.
	primaryStake := math.NewInt(3_000_000)
	secondaryStake := math.NewInt(4_000_000)
	delegate(t, bApp, ctx, stakingMsgServer, bondDenom, primaryAcc, valAddr, primaryStake)
	delegate(t, bApp, ctx, stakingMsgServer, bondDenom, secondaryAcc, valAddr, secondaryStake)

	validatorBefore, err := bApp.StakingKeeper.GetValidator(ctx, valAddr)
	require.NoError(t, err)

	capacityBefore := redelegationCapacity(t, bApp, ctx, playerId)

	structsMsgServer := structskeeper.NewMsgServerImpl(bApp.StructsKeeper)
	_, err = structsMsgServer.AddressRevoke(ctx, &structstypes.MsgAddressRevoke{
		Creator: primaryAcc.String(),
		Address: secondaryAcc.String(),
	})
	require.NoError(t, err)

	merged, err := bApp.StakingKeeper.GetDelegation(ctx, primaryAcc, valAddr)
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDecFromInt(primaryStake.Add(secondaryStake)), merged.Shares,
		"the primary's own stake must survive the arrival of the secondary's")

	_, err = bApp.StakingKeeper.GetDelegation(ctx, secondaryAcc, valAddr)
	require.Error(t, err, "and the revoked address must hold nothing")

	validatorAfter, err := bApp.StakingKeeper.GetValidator(ctx, valAddr)
	require.NoError(t, err)
	require.Equal(t, validatorBefore.DelegatorShares, validatorAfter.DelegatorShares,
		"shares changed delegator, not existence, so the validator's total is untouched")
	require.Equal(t, validatorBefore.Tokens, validatorAfter.Tokens)

	require.Equal(t, capacityBefore, redelegationCapacity(t, bApp, ctx, playerId),
		"and the player keeps the capacity behind all of it")

	// The half of the bug with no symptom at the time. A delegation written by
	// bare SetDelegation has no starting info, and both of these fail on
	// ErrEmptyDelegationDistInfo forever after.
	hasStartingInfo, err := bApp.DistrKeeper.HasDelegatorStartingInfo(ctx, valAddr, primaryAcc)
	require.NoError(t, err)
	require.True(t, hasStartingInfo, "the destination must be able to price its rewards")

	distrMsgServer := distrkeeper.NewMsgServerImpl(bApp.DistrKeeper)
	_, err = distrMsgServer.WithdrawDelegatorReward(ctx, &distrtypes.MsgWithdrawDelegatorReward{
		DelegatorAddress: primaryAcc.String(),
		ValidatorAddress: valAddr.String(),
	})
	require.NoError(t, err, "the merged delegation must still be able to withdraw rewards")

	_, err = stakingMsgServer.Undelegate(ctx, stakingtypes.NewMsgUndelegate(
		primaryAcc.String(), valAddr.String(), sdk.NewCoin(bondDenom, primaryStake)))
	require.NoError(t, err, "and must still be able to undelegate")

	requireStakingAndDistributionHealthy(t, bApp, ctx)
}

// TestDelegationTransferDisownsRedelegatingStakeOnRevoke covers the state the
// transfer cannot move through. A redelegation in flight has queue rows keyed by
// the delegator address that no public keeper API can rewrite, and moving the
// delegation out from under it would make that stake unslashable for the source
// validator's infraction -- SlashRedelegation resolves through the redelegation
// record's own delegator address and skips silently when it finds nothing.
//
// Revoke still has to succeed. It is what a player does about a key they no
// longer trust, and whoever holds that key can keep a redelegation in flight
// indefinitely, so refusing would let an attacker block their own eviction --
// while gaining the player nothing, since that key could always have undelegated
// the stake directly. The stake stays bonded where it is and the game simply
// stops crediting the player for it.
func TestDelegationTransferDisownsRedelegatingStakeOnRevoke(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)

	bondDenom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)

	stakingMsgServer := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	sourceVal := createRedelegationValidator(t, bApp, ctx, stakingMsgServer, bondDenom, "disownsource")
	destinationVal := createRedelegationValidator(t, bApp, ctx, stakingMsgServer, bondDenom, "disowndest")

	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)

	destinationReactorId := jailGateReactorId(t, bApp, ctx, destinationVal)

	primaryAcc, secondaryAcc, playerId := newTransferPlayer(t, bApp, ctx)

	stake := math.NewInt(4_000_000)
	delegate(t, bApp, ctx, stakingMsgServer, bondDenom, secondaryAcc, sourceVal, stake)

	// Put a redelegation in flight into destinationVal, which is what blocks
	// the secondary address's delegation there from moving.
	_, err = stakingMsgServer.BeginRedelegate(ctx, stakingtypes.NewMsgBeginRedelegate(
		secondaryAcc.String(), sourceVal.String(), destinationVal.String(),
		sdk.NewCoin(bondDenom, stake)))
	require.NoError(t, err)

	receiving, err := bApp.StakingKeeper.HasReceivingRedelegation(ctx, secondaryAcc, destinationVal)
	require.NoError(t, err)
	require.True(t, receiving, "precondition: the redelegation is in flight")

	require.NotZero(t, redelegationCapacity(t, bApp, ctx, playerId),
		"precondition: the player is credited for the redelegating stake")

	structsMsgServer := structskeeper.NewMsgServerImpl(bApp.StructsKeeper)
	_, err = structsMsgServer.AddressRevoke(ctx, &structstypes.MsgAddressRevoke{
		Creator: primaryAcc.String(),
		Address: secondaryAcc.String(),
	})
	require.NoError(t, err, "a revoke must never be blockable by the key being revoked")

	require.Zero(t, bApp.StructsKeeper.GetPlayerIndexFromAddress(ctx, secondaryAcc.String()),
		"the address is revoked")

	stranded, err := bApp.StakingKeeper.GetDelegation(ctx, secondaryAcc, destinationVal)
	require.NoError(t, err, "the blocked delegation stays bonded exactly where it was")
	require.Equal(t, math.LegacyNewDecFromInt(stake), stranded.Shares)

	_, err = bApp.StakingKeeper.GetDelegation(ctx, primaryAcc, destinationVal)
	require.Error(t, err, "and did not arrive at the primary")

	if disowned, found := bApp.StructsKeeper.GetInfusion(ctx, destinationReactorId, secondaryAcc.String()); found {
		require.Zero(t, disowned.Fuel, "but the game stops representing it")
		require.Zero(t, disowned.Power)
	}
	require.Zero(t, redelegationCapacity(t, bApp, ctx, playerId),
		"so the player is no longer credited for stake on an address they no longer own")

	requireStakingAndDistributionHealthy(t, bApp, ctx)
}

// requireStakingAndDistributionHealthy is the backstop: it re-checks the two
// module-wide accounting properties a hand-assembled delegation move can break,
// rather than only the specific values each test set up.
//
// These were staking's DelegatorSharesInvariant and distribution's
// CanWithdrawInvariant. SDK v0.53 retired the crisis module and deleted both, so
// they are spelled out here. If a later SDK reinstates them, prefer the SDK's.
func requireStakingAndDistributionHealthy(t *testing.T, bApp *app.App, ctx sdk.Context) {
	t.Helper()

	validators, err := bApp.StakingKeeper.GetAllValidators(ctx)
	require.NoError(t, err)

	for _, validator := range validators {
		valAddr, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
		require.NoError(t, err)

		delegations, err := bApp.StakingKeeper.GetValidatorDelegations(ctx, valAddr)
		require.NoError(t, err)

		// The overwriting rekey's signature failure: shares vanish from a
		// delegation while the validator's total goes on counting them, so the
		// redemption ratio silently favours everyone else.
		total := math.LegacyZeroDec()
		for _, delegation := range delegations {
			total = total.Add(delegation.Shares)
		}
		require.True(t, total.Equal(validator.DelegatorShares),
			"validator %s: delegations sum to %s but DelegatorShares is %s",
			validator.OperatorAddress, total, validator.DelegatorShares)

		// Every delegation must still be able to price its rewards. A
		// delegation whose starting info was never written reads as healthy
		// until the day its owner tries to touch it. Cached, because
		// IncrementValidatorPeriod writes.
		cacheCtx, _ := ctx.CacheContext()
		endingPeriod, err := bApp.DistrKeeper.IncrementValidatorPeriod(cacheCtx, validator)
		require.NoError(t, err)

		for _, delegation := range delegations {
			delAddr, err := sdk.AccAddressFromBech32(delegation.DelegatorAddress)
			require.NoError(t, err)

			hasStartingInfo, err := bApp.DistrKeeper.HasDelegatorStartingInfo(cacheCtx, valAddr, delAddr)
			require.NoError(t, err)
			require.True(t, hasStartingInfo,
				"delegation %s -> %s has no starting info and can never withdraw",
				delegation.DelegatorAddress, validator.OperatorAddress)

			_, err = bApp.DistrKeeper.CalculateDelegationRewards(cacheCtx, validator, delegation, endingPeriod)
			require.NoError(t, err,
				"delegation %s -> %s cannot calculate rewards",
				delegation.DelegatorAddress, validator.OperatorAddress)
		}
	}
}

// newTransferPlayer builds one player with a primary address and a fully
// permissioned secondary address.
func newTransferPlayer(t *testing.T, bApp *app.App, ctx sdk.Context) (primaryAcc, secondaryAcc sdk.AccAddress, playerId string) {
	t.Helper()

	primaryAcc = sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
	secondaryAcc = sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())

	cc := bApp.StructsKeeper.NewCurrentContext(ctx)
	player := cc.UpsertPlayer(primaryAcc.String())
	playerId = player.GetPlayerId()
	playerIndex := player.GetIndex()
	cc.CommitAll()

	require.NoError(t, bApp.StructsKeeper.SetPlayerIndexForAddress(ctx, secondaryAcc.String(), playerIndex))
	bApp.StructsKeeper.SetPermissionsByBytes(ctx,
		structskeeper.GetAddressPermissionIDBytes(secondaryAcc.String()), structstypes.PermAll)

	return primaryAcc, secondaryAcc, playerId
}

func delegate(
	t *testing.T,
	bApp *app.App,
	ctx sdk.Context,
	msgServer stakingtypes.MsgServer,
	bondDenom string,
	delegator sdk.AccAddress,
	valAddr sdk.ValAddress,
	amount math.Int,
) {
	t.Helper()

	fundRedelegationAccount(t, bApp, ctx, delegator, sdk.NewCoins(sdk.NewCoin(bondDenom, amount)))
	_, err := msgServer.Delegate(ctx, stakingtypes.NewMsgDelegate(
		delegator.String(), valAddr.String(), sdk.NewCoin(bondDenom, amount)))
	require.NoError(t, err)
}
