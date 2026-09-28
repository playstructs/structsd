package app_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestTopUpDuringSlashRepairMatchesLiveFlatBacking(t *testing.T) {
	t.Run("native-staking", func(t *testing.T) { testTopUpDuringSlashRepair(t, false) })
	t.Run("structs-handler", func(t *testing.T) { testTopUpDuringSlashRepair(t, true) })
}

func testTopUpDuringSlashRepair(t *testing.T, structsHandler bool) {
	bApp, ctx := setupJailGateApp(t)
	validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	require.NoError(t, err)
	primary, _, _ := newTransferPlayer(t, bApp, ctx)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	delegate(t, bApp, ctx, server, denom, primary, validator, math.NewInt(1000000))
	id := jailGateReactorId(t, bApp, ctx, validator)


	value, err := bApp.StakingKeeper.GetValidator(ctx, validator)
	require.NoError(t, err)
	consensus, err := value.GetConsAddr()
	require.NoError(t, err)
	_, err = bApp.StakingKeeper.Slash(ctx, consensus, ctx.BlockHeight(), value.ConsensusPower(bApp.StakingKeeper.PowerReduction(ctx)), math.LegacyNewDecWithPrec(5, 2))
	require.NoError(t, err)
	backing := func() uint64 {
		t.Helper()
		value, err := bApp.StakingKeeper.GetValidator(ctx, validator)
		require.NoError(t, err)
		delegation, err := bApp.StakingKeeper.GetDelegation(ctx, primary, validator)
		require.NoError(t, err)
		return delegation.Shares.MulInt(value.Tokens).QuoTruncate(value.DelegatorShares).TruncateInt().Uint64()
	}
	oldBacking := backing()
	require.Equal(t, uint64(950000), oldBacking)
	stale, found := bApp.StructsKeeper.GetInfusion(ctx, id, primary.String())
	require.True(t, found)
	require.Equal(t, uint64(1000000), stale.Fuel, "the repair has not reached this position")
	if structsHandler {
		fundRedelegationAccount(t, bApp, ctx, primary, sdk.NewCoins(sdk.NewInt64Coin(denom, 50000)))
		_, err = keeperlib.NewMsgServerImpl(bApp.StructsKeeper).ReactorInfuse(ctx, &types.MsgReactorInfuse{
			Creator: primary.String(), DelegatorAddress: primary.String(), ValidatorAddress: validator.String(), Amount: sdk.NewInt64Coin(denom, 50000),
		})
		require.NoError(t, err)
	} else {
		delegate(t, bApp, ctx, server, denom, primary, validator, math.NewInt(50000))
	}
	fuel := backing()
	row, found := bApp.StructsKeeper.GetInfusion(ctx, id, primary.String())
	require.True(t, found)
	require.Equal(t, fuel, row.Fuel)
	require.Equal(t, fuel, row.Power)

	bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
	repaired, found := bApp.StructsKeeper.GetInfusion(ctx, id, primary.String())
	require.True(t, found)
	require.Equal(t, row, repaired, "repair cannot reprice the already settled top-up")
	requireStakingAndDistributionHealthy(t, bApp, ctx)
}
