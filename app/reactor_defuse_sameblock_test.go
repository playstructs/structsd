package app_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	structskeeper "structs/x/structs/keeper"
	structstypes "structs/x/structs/types"
)

/* TestDefuseTwiceInOneBlockLeavesNoStalePower pins two Cosmos SDK behaviours
 * that this module's infusion accounting rides on, against real staking.
 *
 * Undelegate derives an unbonding entry's creation height and completion time
 * from the block, so two defuses in one block produce the *same* entry:
 * UnbondingDelegation.AddEntry merges them, and SetUnbondingDelegationEntry
 * fires AfterUnbondingInitiated only for a genuinely new one. Separately, Unbond
 * routes a delegation whose shares reach zero through RemoveDelegation and
 * deliberately skips AfterDelegationModified.
 *
 * So a small defuse followed by a draining one triggers neither of the two hooks
 * that would normally reconcile - which is exactly the shape that used to leave
 * Fuel, and therefore reactor power, standing on stake that was already
 * unbonding. BeforeDelegationRemoved covers it now.
 *
 * The mock staking keeper fires no hooks at all, so only a real chain can show
 * any of this.
 */
func TestDefuseTwiceInOneBlockLeavesNoStalePower(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)

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
		stakingtypes.NewDescription("defuse-same-block", "", "", "", ""),
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
	require.NotZero(t, fuelBefore, "the self-delegation must be infused before we defuse it")

	delegation, err := bApp.StakingKeeper.GetDelegation(ctx, operatorAcc, valAddr)
	require.NoError(t, err)
	totalShares := delegation.Shares

	// First defuse: a token amount, so the remainder still backs real power.
	small := math.LegacyNewDec(1000)
	_, _, err = bApp.StakingKeeper.Undelegate(ctx, operatorAcc, valAddr, small)
	require.NoError(t, err)

	_, fuelAfterFirst := jailGateInfusion(t, bApp, ctx, reactorId, operatorAcc)
	require.Less(t, fuelAfterFirst, fuelBefore, "the first defuse must reduce fuel")
	require.NotZero(t, fuelAfterFirst, "and must leave the remainder infused")

	// Second defuse, same block: everything that is left. This is the call that
	// fires neither hook - the unbonding entry merges into the first, and the
	// delegation is removed rather than modified.
	_, _, err = bApp.StakingKeeper.Undelegate(ctx, operatorAcc, valAddr, totalShares.Sub(small))
	require.NoError(t, err)

	_, err = bApp.StakingKeeper.GetDelegation(ctx, operatorAcc, valAddr)
	require.Error(t, err, "the delegation should be gone; that is what makes the hooks skip")

	unbonding, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, operatorAcc, valAddr)
	require.NoError(t, err)
	require.Len(t, unbonding.Entries, 1,
		"both defuses must land in one merged entry; without the merge this test proves nothing")

	// The property that matters: no fuel, and therefore no power or grid
	// capacity, standing on stake that is unbonding.
	_, fuelAfterSecond := jailGateInfusion(t, bApp, ctx, reactorId, operatorAcc)
	require.Zero(t, fuelAfterSecond,
		"the whole stake is unbonding, so no fuel may remain backing reactor power")

	capacity := bApp.StructsKeeper.GetGridAttribute(ctx,
		structskeeper.GetGridAttributeIDByObjectId(structstypes.GridAttributeType_capacity, reactorId))
	require.Zero(t, capacity, "and no reactor capacity may remain either")
}
