package app_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
)

// A historical slash follows redelegations into their destination. The cap
// must allow that SDK Unbond hook even if the destination is already over cap
// and its auxiliary infusion repair cannot currently complete.
func TestReactorStakeCapCannotVetoHistoricalRedelegationSlash(t *testing.T) {
	for _, brokenInfusion := range []bool{false, true} {
		t.Run(map[bool]string{false: "healthy", true: "repair-pending"}[brokenInfusion], func(t *testing.T) {
			bApp, ctx := setupJailGateAppWithValidators(t, 4)
			validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
			require.NoError(t, err)
			source, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
			require.NoError(t, err)
			target, err := sdk.ValAddressFromBech32(validators[1].OperatorAddress)
			require.NoError(t, err)
			address, _, _ := newTransferPlayer(t, bApp, ctx)
			denom, err := bApp.StakingKeeper.BondDenom(ctx)
			require.NoError(t, err)
			server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
			delegate(t, bApp, ctx, server, denom, address, source, math.NewInt(2000000))
			_, err = server.BeginRedelegate(ctx, stakingtypes.NewMsgBeginRedelegate(address.String(), source.String(), target.String(), sdk.NewInt64Coin(denom, 1000000)))
			require.NoError(t, err)
			activateTestStakeCap(t, bApp, ctx)
			reactor := jailGateReactorId(t, bApp, ctx, target)
			share, err := bApp.StructsKeeper.ReactorStakeShare(ctx, &types.QueryReactorStakeShareRequest{ReactorId: reactor})
			require.NoError(t, err)
			require.Equal(t, "0", share.ExternalDepositHeadroom, "destination is already above the cap")
			if brokenInfusion {
				row, found := bApp.StructsKeeper.GetInfusion(ctx, reactor, address.String())
				require.True(t, found)
				row.DestinationType = types.ObjectType(500)
				bApp.StructsKeeper.SetInfusion(ctx, row)
				bApp.StructsKeeper.EnqueueReactorRestart(ctx, reactor)
			}
			before, err := bApp.StakingKeeper.GetDelegation(ctx, address, target)
			require.NoError(t, err)
			val, err := bApp.StakingKeeper.GetValidator(ctx, source)
			require.NoError(t, err)
			consensus, err := val.GetConsAddr()
			require.NoError(t, err)
			infractionHeight := ctx.BlockHeight()
			ctx = ctx.WithBlockHeight(infractionHeight + 1)
			burned, err := bApp.StakingKeeper.Slash(ctx, consensus, infractionHeight, val.ConsensusPower(bApp.StakingKeeper.PowerReduction(ctx)), math.LegacyNewDecWithPrec(1, 1))
			require.NoError(t, err)
			require.True(t, burned.IsPositive())
			after, err := bApp.StakingKeeper.GetDelegation(ctx, address, target)
			require.NoError(t, err)
			require.True(t, after.Shares.LT(before.Shares), "slash must reach the redelegated stake")
			if brokenInfusion {
				require.NotEmpty(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx))
			}
			requireStakingAndDistributionHealthy(t, bApp, ctx)
		})
	}
}
