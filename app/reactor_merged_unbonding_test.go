package app_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
)

func TestMergedUnbondingReconcilesAfterStakingWithoutDuplicateQueueWork(t *testing.T) {
	for _, full := range []bool{false, true} {
		t.Run(fmt.Sprintf("full=%t", full), func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
			require.NoError(t, err)
			validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
			require.NoError(t, err)
			denom, err := bApp.StakingKeeper.BondDenom(ctx)
			require.NoError(t, err)
			primary, _, _ := newTransferPlayer(t, bApp, ctx)
			server := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
			stake := int64(3000000)
			if full {
				stake = 2000000
			}
			delegate(t, bApp, ctx, server, denom, primary, validator, math.NewInt(stake))
			reactorBytes, found := bApp.StructsKeeper.GetReactorBytesFromValidator(ctx, validator)
			require.True(t, found)
			reactor, found := bApp.StructsKeeper.GetReactorByBytes(ctx, reactorBytes)
			require.True(t, found)
			for i := 0; i < 2; i++ {
				_, err := server.Undelegate(ctx, stakingtypes.NewMsgUndelegate(primary.String(), validator.String(), sdk.NewInt64Coin(denom, 1000000)))
				require.NoError(t, err)
			}
			ubd, err := bApp.StakingKeeper.GetUnbondingDelegation(ctx, primary, validator)
			require.NoError(t, err)
			require.Len(t, ubd.Entries, 1, "SDK merges these operations without a second unbonding hook")
			require.Equal(t, int64(2000000), ubd.Entries[0].Balance.Int64())
			// One current-block reconciliation plus one future maturity row,
			// regardless of how many reductions touched this infusion.
			require.Len(t, bApp.StructsKeeper.GetInfusionMaturitySweepQueueExport(ctx), 2)
			_, err = bApp.StructsKeeper.EndBlocker(ctx)
			require.NoError(t, err)
			infusion, found := bApp.StructsKeeper.GetInfusion(ctx, reactor.Id, primary.String())
			require.True(t, found)
			require.Equal(t, uint64(2000000), infusion.Defusing)
			require.Equal(t, uint64(stake-2000000), infusion.Fuel)
			require.NoError(t, infusion.ValidateAssetOwner())
			require.Equal(t, infusion.Fuel, infusion.Power)
			require.Len(t, bApp.StructsKeeper.GetInfusionMaturitySweepQueueExport(ctx), 1, "reconciliation must not re-enqueue itself")
			_, err = bApp.StructsKeeper.EndBlocker(ctx)
			require.NoError(t, err)
			repeated, found := bApp.StructsKeeper.GetInfusion(ctx, reactor.Id, primary.String())
			require.True(t, found)
			require.Equal(t, infusion, repeated)
			requireStakingAndDistributionHealthy(t, bApp, ctx)
		})
	}
}
