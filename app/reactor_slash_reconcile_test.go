package app_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	structstypes "structs/x/structs/types"
)

/* TestSlashDefersReconciliationToTheEndBlocker runs the real staking module.
 *
 * Every keeper-level test of this path drives our own handler directly, which
 * proves the batching and says nothing about whether staking still reaches us or
 * whether the deferral lands in the right place. Two things about the ordering
 * are load-bearing and only a real chain shows them:
 *
 *   - BeforeValidatorSlashed fires *before* staking applies the slash, which is
 *     why the old inline code had to derive the post-slash token figure by hand.
 *     The queue captures nothing, so it depends on the reconciliation running
 *     later, once staking has written the reduced tokens.
 *   - The hook itself must now be O(1). If it were still walking the delegator
 *     set, the queue would be beside the point.
 */
func TestSlashDefersReconciliationToTheEndBlocker(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)

	bondDenom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)

	operatorAcc := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())
	stake := math.NewInt(10_000_000)
	funding := sdk.NewCoins(sdk.NewCoin(bondDenom, stake.MulRaw(2)))
	require.NoError(t, bApp.BankKeeper.MintCoins(ctx, structstypes.ModuleName, funding))
	require.NoError(t, bApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, structstypes.ModuleName, operatorAcc, funding))

	valAddr := sdk.ValAddress(operatorAcc)
	consPubKey := ed25519.GenPrivKey().PubKey()

	createMsg, err := stakingtypes.NewMsgCreateValidator(
		valAddr.String(),
		consPubKey,
		sdk.NewCoin(bondDenom, stake),
		stakingtypes.NewDescription("slash-reconcile", "", "", "", ""),
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

	_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)

	reactorId := jailGateReactorId(t, bApp, ctx, valAddr)
	_, fuelBefore := jailGateInfusion(t, bApp, ctx, reactorId, operatorAcc)
	require.NotZero(t, fuelBefore, "the self-delegation must be infused before we slash it")

	require.Empty(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx),
		"nothing should be queued before the slash")

	// The real thing: staking's Slash, which is what x/slashing and x/evidence
	// both call, and which fires BeforeValidatorSlashed on the way through.
	validator, err := bApp.StakingKeeper.GetValidator(ctx, valAddr)
	require.NoError(t, err)
	power := validator.ConsensusPower(bApp.StakingKeeper.PowerReduction(ctx))

	_, err = bApp.StakingKeeper.Slash(ctx, sdk.ConsAddress(consPubKey.Address()), ctx.BlockHeight(), power,
		math.LegacyNewDecWithPrec(5, 2))
	require.NoError(t, err)

	// The hook queued the reactor rather than reconciling anything, so the
	// infusion still reads its pre-slash fuel at this point. Asserting that is
	// what stops this test passing for the wrong reason: it pins the reconcile to
	// the EndBlocker rather than to the hook.
	queue := bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx)
	require.Len(t, queue, 1, "the slash must queue its reactor")
	require.Equal(t, reactorId, queue[0].ReactorId)

	_, fuelAfterHook := jailGateInfusion(t, bApp, ctx, reactorId, operatorAcc)
	require.Equal(t, fuelBefore, fuelAfterHook,
		"the hook must do no per-delegation work; the reconcile belongs to the EndBlocker")

	// Now the EndBlocker, where staking has already written the reduced tokens.
	bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)

	require.Empty(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx),
		"one delegator is well inside a block's budget")

	_, fuelAfterReconcile := jailGateInfusion(t, bApp, ctx, reactorId, operatorAcc)
	require.Less(t, fuelAfterReconcile, fuelBefore,
		"the reconciliation must pick up the slash from live staking state")
}
