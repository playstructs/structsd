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

// TestFullRedelegationClearsSourceInfusion is the release gate for the
// BeforeDelegationRemoved hook, and it has to run against real staking.
//
// Staking's Unbond routes a delegation whose shares reach zero through
// RemoveDelegation, which fires BeforeDelegationRemoved and deliberately skips
// AfterDelegationModified. While that hook was a no-op, a full redelegation left
// the source infusion's fuel, power and grid capacity installed while the
// destination was granted capacity for the very same stake, so the delegator
// ended up holding two reactors' worth of capacity for one stake.
//
// The keeper's mock staking keeper fires no hooks at all, so no unit test can
// see any of this. If this test fails after an SDK upgrade, energy duplication
// is back: check whether Unbond still takes the RemoveDelegation branch at zero
// shares.
func TestFullRedelegationClearsSourceInfusion(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)

	bondDenom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)

	stakingMsgServer := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)

	sourceVal := createRedelegationValidator(t, bApp, ctx, stakingMsgServer, bondDenom, "source")
	destVal := createRedelegationValidator(t, bApp, ctx, stakingMsgServer, bondDenom, "destination")

	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)

	sourceReactorId := jailGateReactorId(t, bApp, ctx, sourceVal)
	destReactorId := jailGateReactorId(t, bApp, ctx, destVal)

	// Delegate from a third party rather than an operator, so the operators'
	// own self-delegations cannot mask a stale source row.
	delegator := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
	stake := math.NewInt(4_000_000)
	fundRedelegationAccount(t, bApp, ctx, delegator, sdk.NewCoins(sdk.NewCoin(bondDenom, stake)))

	_, err = stakingMsgServer.Delegate(ctx, stakingtypes.NewMsgDelegate(
		delegator.String(), sourceVal.String(), sdk.NewCoin(bondDenom, stake)))
	require.NoError(t, err)

	sourceBefore, found := bApp.StructsKeeper.GetInfusion(ctx, sourceReactorId, delegator.String())
	require.True(t, found, "delegating should have infused the source reactor")
	require.Equal(t, stake.Uint64(), sourceBefore.Fuel)
	require.NotZero(t, sourceBefore.Power, "a bonded validator's infusion should carry power")

	playerId := sourceBefore.PlayerId
	require.NotEmpty(t, playerId, "the delegator should have been upserted into a player")

	_, sourceCommissionPower, sourcePlayerPower := sourceBefore.GetPowerDistribution()
	require.NotZero(t, sourceCommissionPower, "the reactor should hold its commission share")
	require.NotZero(t, sourcePlayerPower, "the delegator should hold the remainder")

	playerCapacityBefore := redelegationCapacity(t, bApp, ctx, playerId)
	sourceReactorCapacityBefore := redelegationCapacity(t, bApp, ctx, sourceReactorId)
	require.Equal(t, sourcePlayerPower, playerCapacityBefore,
		"the delegator's only capacity should be this infusion's player share")

	// Move the entire delegation. Shares reach zero at the source, which is the
	// case that skips AfterDelegationModified.
	_, err = stakingMsgServer.BeginRedelegate(ctx, stakingtypes.NewMsgBeginRedelegate(
		delegator.String(), sourceVal.String(), destVal.String(), sdk.NewCoin(bondDenom, stake)))
	require.NoError(t, err)

	_, err = bApp.StakingKeeper.GetDelegation(ctx, delegator, sourceVal)
	require.Error(t, err, "a full redelegation should leave no source delegation behind")

	// The record itself may survive until the EndBlocker drains the destruction
	// queue, but it must carry nothing.
	if sourceAfter, stillThere := bApp.StructsKeeper.GetInfusion(ctx, sourceReactorId, delegator.String()); stillThere {
		require.Zero(t, sourceAfter.Fuel, "the source infusion must keep no fuel")
		require.Zero(t, sourceAfter.Power, "the source infusion must keep no power")
	}

	destAfter, found := bApp.StructsKeeper.GetInfusion(ctx, destReactorId, delegator.String())
	require.True(t, found, "the destination reactor should have been infused")
	require.Equal(t, stake.Uint64(), destAfter.Fuel, "the stake should have arrived intact")

	require.Equal(t, sourceReactorCapacityBefore-sourceCommissionPower,
		redelegationCapacity(t, bApp, ctx, sourceReactorId),
		"the source reactor must give up the commission capacity it no longer earns")

	// The heart of it. Capacity moved between reactors, so the delegator's own
	// total is unchanged; anything above this figure is capacity backed by no
	// stake at all.
	require.Equal(t, playerCapacityBefore, redelegationCapacity(t, bApp, ctx, playerId),
		"a redelegation moves the delegator's capacity, it does not duplicate it")

	// Net-zero is not the same as never dipping. The source leg subtracts and
	// queues a grid cascade before the destination leg adds it back, and the
	// queue is only drained in the EndBlocker, by which point the delegator is
	// whole again. Reordering the hook so the dip outlives the transaction
	// would shed this player's allocations for no reason.
	cc := bApp.StructsKeeper.NewCurrentContext(ctx)
	require.True(t, cc.GetPlayer(playerId).IsOnline(),
		"a legitimate redelegation must not knock the delegator offline")
}

// TestFullUndelegationClearsInfusionFuel pins the neighbouring case, which was
// never broken and is worth keeping honest for a different reason.
//
// A full undelegate reaches RemoveDelegation exactly as a full redelegation
// does, but it is covered by a second path: SetUnbondingDelegationEntry fires
// AfterUnbondingInitiated with an id that resolves to a real unbonding
// delegation, so ReactorInfusionUnbonding clears the fuel whatever
// BeforeDelegationRemoved does. That is precisely why the redelegation case hid
// for so long, the two differing only in which id staking hands us.
//
// So this passes with the hook reverted. It is here to pin the interaction
// between the two paths: the fuel goes, and the defusing balance the other path
// recorded survives the hook rather than being wiped by it.
func TestFullUndelegationClearsInfusionFuel(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)

	bondDenom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)

	stakingMsgServer := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	valAddr := createRedelegationValidator(t, bApp, ctx, stakingMsgServer, bondDenom, "undelegate")

	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)

	reactorId := jailGateReactorId(t, bApp, ctx, valAddr)

	delegator := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
	stake := math.NewInt(3_000_000)
	fundRedelegationAccount(t, bApp, ctx, delegator, sdk.NewCoins(sdk.NewCoin(bondDenom, stake)))

	_, err = stakingMsgServer.Delegate(ctx, stakingtypes.NewMsgDelegate(
		delegator.String(), valAddr.String(), sdk.NewCoin(bondDenom, stake)))
	require.NoError(t, err)

	before, found := bApp.StructsKeeper.GetInfusion(ctx, reactorId, delegator.String())
	require.True(t, found)
	require.Equal(t, stake.Uint64(), before.Fuel)

	_, err = stakingMsgServer.Undelegate(ctx, stakingtypes.NewMsgUndelegate(
		delegator.String(), valAddr.String(), sdk.NewCoin(bondDenom, stake)))
	require.NoError(t, err)

	after, found := bApp.StructsKeeper.GetInfusion(ctx, reactorId, delegator.String())
	require.True(t, found, "an unbonding balance should keep the record alive")
	require.Zero(t, after.Fuel, "the withdrawn stake must stop producing capacity")
	require.Zero(t, after.Power)
	require.Equal(t, stake.Uint64(), after.Defusing, "the unbonding balance must survive the hook")

	require.Zero(t, redelegationCapacity(t, bApp, ctx, before.PlayerId),
		"a fully undelegated player holds no infusion capacity")
}

func createRedelegationValidator(
	t *testing.T,
	bApp *app.App,
	ctx sdk.Context,
	msgServer stakingtypes.MsgServer,
	bondDenom string,
	moniker string,
) sdk.ValAddress {
	t.Helper()

	operator := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
	stake := math.NewInt(10_000_000)
	fundRedelegationAccount(t, bApp, ctx, operator, sdk.NewCoins(sdk.NewCoin(bondDenom, stake)))

	createMsg, err := stakingtypes.NewMsgCreateValidator(
		sdk.ValAddress(operator).String(),
		ed25519.GenPrivKey().PubKey(),
		sdk.NewCoin(bondDenom, stake),
		stakingtypes.NewDescription(moniker, "", "", "", ""),
		stakingtypes.NewCommissionRates(
			math.LegacyNewDecWithPrec(1, 1),
			math.LegacyNewDecWithPrec(2, 1),
			math.LegacyNewDecWithPrec(1, 2),
		),
		math.OneInt(),
	)
	require.NoError(t, err)

	_, err = msgServer.CreateValidator(ctx, createMsg)
	require.NoError(t, err)

	return sdk.ValAddress(operator)
}

func fundRedelegationAccount(t *testing.T, bApp *app.App, ctx sdk.Context, to sdk.AccAddress, amount sdk.Coins) {
	t.Helper()

	// The structs module account carries Minter, so it is the faucet available
	// in a test chain with no mint module.
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, structstypes.ModuleName, amount))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, structstypes.ModuleName, to, amount))
}

func redelegationCapacity(t *testing.T, bApp *app.App, ctx sdk.Context, objectId string) uint64 {
	t.Helper()

	return bApp.StructsKeeper.GetGridAttribute(ctx,
		structskeeper.GetGridAttributeIDByObjectId(structstypes.GridAttributeType_capacity, objectId))
}
