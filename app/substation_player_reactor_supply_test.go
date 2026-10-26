package app_test

import (
	"testing"
 

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
)

func TestSubstationRejectsStalePlayerReactorSupply(t *testing.T) {
	for _, tc := range []struct {
		name         string
		slash, spare bool
	}{
		{"jail", false, false},
		{"slash", true, false},
		{"slash-with-spare-backing", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
			require.NoError(t, err)
			validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
			require.NoError(t, err)
			primary, _, ownerID := newTransferPlayer(t, bApp, ctx)
			_, _, consumerID := newTransferPlayer(t, bApp, ctx)
			denom, err := bApp.StakingKeeper.BondDenom(ctx)
			require.NoError(t, err)
			delegate(t, bApp, ctx, stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper), denom, primary, validator, math.NewInt(1000000))

			stationID := "4-99999"
			bApp.StructsKeeper.SetSubstation(ctx, types.Substation{Id: stationID, Owner: ownerID, Creator: primary.String()})
			cc := bApp.StructsKeeper.NewCurrentContext(ctx)
			power := cc.GetPlayer(ownerID).GetCapacity()
			if tc.spare {
				power /= 2
			}
			_, err = cc.NewAllocation(types.AllocationType_dynamic, ownerID, stationID, primary.String(), ownerID, power)
			require.NoError(t, err)
			cc.GetPlayer(consumerID).MigrateSubstation(stationID)
			cc.CommitAll()
			cc = bApp.StructsKeeper.NewCurrentContext(ctx)
			require.Equal(t, power*9/10, cc.GetSubstation(stationID).GetAvailableCapacity())
			require.True(t, cc.GetPlayer(consumerID).IsOnline())
			value, err := bApp.StakingKeeper.GetValidator(ctx, validator)
			require.NoError(t, err)
			consensus, err := value.GetConsAddr()
			require.NoError(t, err)
			if tc.slash {
				_, err = bApp.StakingKeeper.Slash(ctx, consensus, ctx.BlockHeight(), value.ConsensusPower(bApp.StakingKeeper.PowerReduction(ctx)), math.LegacyNewDecWithPrec(5, 2))
				require.NoError(t, err)
			} else {
				require.NoError(t, bApp.StakingKeeper.Jail(ctx, consensus))
			}
			beforeGrid := bApp.StructsKeeper.GetAllGridExport(ctx)
			beforeInfusions := bApp.StructsKeeper.GetAllInfusion(ctx)
			beforeCount := bApp.StructsKeeper.GetAllocationCount(ctx)
			require.Equal(t, power, cc.GetSubstation(stationID).GetCapacity())
			require.Equal(t, power*9/10, cc.GetPlayer(consumerID).GetCapacitySecondary())
			require.Equal(t, power*9/10, cc.GetSubstation(stationID).GetAvailableCapacity())
            require.True(t, cc.GetPlayer(consumerID).IsOnline(), "secondary readiness does not walk upstream positions")
			cc.CommitAll()
			require.Equal(t, beforeGrid, bApp.StructsKeeper.GetAllGridExport(ctx))
			require.Equal(t, beforeInfusions, bApp.StructsKeeper.GetAllInfusion(ctx))
			require.Equal(t, beforeCount, bApp.StructsKeeper.GetAllocationCount(ctx))
		})
	}
}
