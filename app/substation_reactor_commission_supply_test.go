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

func TestSubstationRejectsJailedReactorCommissionSupply(t *testing.T) {
	for _, tc := range []struct {
		name   string
		jailed bool
		slashed bool
	}{
		{"healthy", false, false},
		{"jailed", true, false},
		{"slashed", false, true},
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
			reactorID := jailGateReactorId(t, bApp, ctx, validator)
			stationID := "4-99999"
			bApp.StructsKeeper.SetSubstation(ctx, types.Substation{Id: stationID, Owner: ownerID, Creator: primary.String()})
			cc := bApp.StructsKeeper.NewCurrentContext(ctx)
			capacityBefore := cc.GetGridAttribute(keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, reactorID))
			power := capacityBefore
			power /= 2
			feeding, err := cc.NewAllocation(types.AllocationType_dynamic, reactorID, stationID, primary.String(), ownerID, power)
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
			bApp.StructsKeeper.EnqueueReactorRestart(ctx, reactorID)
			require.Equal(t, power*9/10, cc.GetSubstation(stationID).GetAvailableCapacity())
			if tc.jailed {
				require.NoError(t, bApp.StakingKeeper.Jail(ctx, consensus))
			}
			if tc.slashed {
				_, err = bApp.StakingKeeper.Slash(ctx, consensus, ctx.BlockHeight(), value.ConsensusPower(bApp.StakingKeeper.PowerReduction(ctx)), math.LegacyNewDecWithPrec(5, 2))
				require.NoError(t, err)
				revision, err := bApp.StructsKeeper.GetReactorCommissionRevision(ctx, reactorID)
				require.NoError(t, err)
				require.Equal(t, keeperlib.ReactorCommissionRevision{Issued: 1}, revision)
			}
			beforeGrid := bApp.StructsKeeper.GetAllGridExport(ctx)
			beforeInfusions := bApp.StructsKeeper.GetAllInfusion(ctx)
			beforeCount := bApp.StructsKeeper.GetAllocationCount(ctx)
			require.Equal(t, power, cc.GetSubstation(stationID).GetCapacity())
			require.Equal(t, power*9/10, cc.GetPlayer(consumerID).GetCapacitySecondary())
			if !tc.jailed && !tc.slashed {
				require.Equal(t, power*9/10, cc.GetSubstation(stationID).GetAvailableCapacity())
				require.True(t, cc.GetPlayer(consumerID).IsOnline())
			} else {
				require.Equal(t, power*9/10, cc.GetSubstation(stationID).GetAvailableCapacity())
				require.True(t, cc.GetPlayer(consumerID).IsOnline())
				_, err = cc.NewAllocation(types.AllocationType_dynamic, reactorID, "", primary.String(), ownerID, 1)
				require.ErrorContains(t, err, "unsupported_supply")
				current, found := cc.GetAllocation(feeding.ID())
				require.True(t, found)
				_, err = current.SetDynamicPower(power + 1)
				require.ErrorContains(t, err, "unsupported_supply")
			}
			cc.CommitAll()
			require.Equal(t, beforeGrid, bApp.StructsKeeper.GetAllGridExport(ctx))
			require.Equal(t, beforeInfusions, bApp.StructsKeeper.GetAllInfusion(ctx))
			require.Equal(t, beforeCount, bApp.StructsKeeper.GetAllocationCount(ctx))
            if tc.slashed {
                bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
                revision, err := bApp.StructsKeeper.GetReactorCommissionRevision(ctx, reactorID)
                require.NoError(t, err)
                require.Equal(t, keeperlib.ReactorCommissionRevision{Issued: 1, Certified: 1}, revision)
                require.Empty(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx))
                repaired := bApp.StructsKeeper.NewCurrentContext(ctx)
                capacityAfter := repaired.GetGridAttribute(keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, reactorID))
                require.Equal(t, capacityBefore*95/100, capacityAfter)
                require.Equal(t, power*9/10, repaired.GetSubstation(stationID).GetAvailableCapacity())
                require.True(t, repaired.GetPlayer(consumerID).IsOnline())
                _, err = repaired.NewAllocation(types.AllocationType_dynamic, reactorID, "", primary.String(), ownerID, 1)
                require.NoError(t, err, "admission recovers after slash accounting commits")
                beforeRepeat := bApp.StructsKeeper.GetAllGridExport(ctx)
                beforeEvents := len(ctx.EventManager().Events())
                bApp.StructsKeeper.EnqueueReactorRestart(ctx, reactorID)
                bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
                require.Equal(t, beforeRepeat, bApp.StructsKeeper.GetAllGridExport(ctx))
                require.Len(t, ctx.EventManager().Events(), beforeEvents)
            }
		})
	}
}
