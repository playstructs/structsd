package app_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestCancellationRestoresFlatBacking(t *testing.T) {
	for _, native := range []bool{false, true} {
		t.Run(map[bool]string{true: "/native", false: "/structs"}[native], func(t *testing.T) {
			bApp, ctx := setupJailGateAppWithValidators(t, 2)
			primary, _, id := newTransferPlayer(t, bApp, ctx)
			vals, err := bApp.StakingKeeper.GetLastValidators(ctx)
			require.NoError(t, err)
			val, err := sdk.ValAddressFromBech32(vals[0].OperatorAddress)
			require.NoError(t, err)
			denom, err := bApp.StakingKeeper.BondDenom(ctx)
			require.NoError(t, err)
			address := primary
			if !native {
				address, err = types.VaultAddress(id)
				require.NoError(t, err)
				_, err = bApp.StructsKeeper.NewCurrentContext(ctx).EnsureVaults(id)
				require.NoError(t, err)
			}
			staking := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
			delegate(t, bApp, ctx, staking, denom, address, val, math.NewInt(1000000))
			backup, err := sdk.ValAddressFromBech32(vals[1].OperatorAddress)
			require.NoError(t, err)
			delegate(t, bApp, ctx, staking, denom, address, backup, math.NewInt(1000000))
			before, err := bApp.StructsKeeper.OwnerReactorDiversity(ctx, &types.QueryOwnerReactorDiversityRequest{OwnerId: id})
			require.NoError(t, err)
			require.Positive(t, before.Stored.AppliedBonus)
			reactor := jailGateReactorId(t, bApp, ctx, val)
			server := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
			for _, amount := range []int64{400000, 600000} {
				if native {
					_, err = staking.Undelegate(ctx, stakingtypes.NewMsgUndelegate(address.String(), val.String(), sdk.NewInt64Coin(denom, amount)))
				} else {
					_, err = server.ReactorDefuse(ctx, &types.MsgReactorDefuse{Creator: primary.String(), ValidatorAddress: val.String(), Amount: sdk.NewInt64Coin(denom, amount)})
				}
				require.NoError(t, err)
				require.NoError(t, bApp.StructsKeeper.CheckReactorDiversityAccounting(ctx))
			}
			pending, found := bApp.StructsKeeper.GetInfusion(ctx, reactor, address.String())
			require.True(t, found)
			require.Zero(t, pending.Fuel)
			require.Equal(t, id, pending.AssetOwnerID())
			remaining, found := bApp.StructsKeeper.GetOwnerReactorDiversity(ctx, id)
			require.True(t, found)
			require.Zero(t, remaining.AppliedBonus, "one remaining reactor has no diversity")
			ubd, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, address, val)
			require.NoError(t, err)
			require.Len(t, ubd.Entries, 1, "same-block native unbonding entries merge")
			require.Equal(t, math.NewInt(1000000), ubd.Entries[0].Balance)
			for i, amount := range []int64{250000, 750000} {
				if native {
					_, err = staking.CancelUnbondingDelegation(ctx, &stakingtypes.MsgCancelUnbondingDelegation{DelegatorAddress: address.String(), ValidatorAddress: val.String(), Amount: sdk.NewInt64Coin(denom, amount), CreationHeight: ctx.BlockHeight()})
				} else {
					_, err = server.ReactorCancelDefusion(ctx, &types.MsgReactorCancelDefusion{Creator: primary.String(), ValidatorAddress: val.String(), Amount: sdk.NewInt64Coin(denom, amount), CreationHeight: ctx.BlockHeight()})
				}
				require.NoError(t, err)
				bApp.StructsKeeper.ProcessInfusionMaturitySweep(ctx)
				row, found := bApp.StructsKeeper.GetInfusion(ctx, reactor, address.String())
				require.True(t, found)

				require.NoError(t, row.ValidateAssetOwner())
				require.Equal(t, id, row.AssetOwnerID())
				require.NoError(t, bApp.StructsKeeper.CheckReactorDiversityAccounting(ctx))
				if i == 1 {
					require.Equal(t, uint64(1000000), row.Power)
					require.Zero(t, row.Defusing)
					require.Equal(t, uint64(1000000), row.Fuel)
				}
			}
			after, err := bApp.StructsKeeper.OwnerReactorDiversity(ctx, &types.QueryOwnerReactorDiversityRequest{OwnerId: id})
			require.NoError(t, err)
			require.Equal(t, before, after, "full cancellation restores the portfolio and its bonus without historical metadata")
			requireStakingAndDistributionHealthy(t, bApp, ctx)
		})
	}
}

func TestCancellationSelectsTheOriginalEntryAndPrunesMaturity(t *testing.T) {
	bApp, ctx := setupJailGateApp(t)
	address, _, _ := newTransferPlayer(t, bApp, ctx)
	vals, err := bApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	val, err := sdk.ValAddressFromBech32(vals[0].OperatorAddress)
	require.NoError(t, err)
	denom, err := bApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
	delegate(t, bApp, ctx, server, denom, address, val, math.NewInt(1000000))
	reactor := jailGateReactorId(t, bApp, ctx, val)

	firstHeight := ctx.BlockHeight()
	_, err = server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(address.String(), val.String(), sdk.NewInt64Coin(denom, 500000)))
	require.NoError(t, err)
	ctx = ctx.WithBlockHeight(firstHeight + 1)
	delegate(t, bApp, ctx, server, denom, address, val, math.NewInt(500000))
	_, err = server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(address.String(), val.String(), sdk.NewInt64Coin(denom, 1000000)))
	require.NoError(t, err)
	row, found := bApp.StructsKeeper.GetInfusion(ctx, reactor, address.String())
	require.True(t, found)
	require.Equal(t, uint64(1500000), row.Defusing)
	_, err = server.CancelUnbondingDelegation(ctx, &stakingtypes.MsgCancelUnbondingDelegation{DelegatorAddress: address.String(), ValidatorAddress: val.String(), Amount: sdk.NewInt64Coin(denom, 500000), CreationHeight: firstHeight})
	require.NoError(t, err)
	row, found = bApp.StructsKeeper.GetInfusion(ctx, reactor, address.String())
	require.True(t, found)

	require.Equal(t, uint64(500000), row.Fuel)
	require.Equal(t, uint64(500000), row.Power)
	pending, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, address, val)
	require.NoError(t, err)
	ctx = ctx.WithBlockTime(pending.Entries[0].CompletionTime.Add(time.Second))
	_, err = bApp.StakingKeeper.CompleteUnbonding(ctx, address, val)
	require.NoError(t, err)
	bApp.StructsKeeper.ProcessInfusionMaturitySweep(ctx)
	row, found = bApp.StructsKeeper.GetInfusion(ctx, reactor, address.String())
	require.True(t, found)
	require.Equal(t, uint64(500000), row.Power)
	require.Zero(t, row.Defusing)

	requireStakingAndDistributionHealthy(t, bApp, ctx)
}
