package app_test

import (
	"encoding/binary"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// This uses actual staking transitions, not a direct call to the hook or the
// legacy full-scan fixture helper. Delegator count must not size the hook.
func TestNativeReactorShutdownQueuesBoundedCleanupAndRebond(t *testing.T) {
	var costs []uint64
	for _, count := range []int{types.ReactorSlashReconcileBudget + 2, 8*types.ReactorSlashReconcileBudget + 2} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			bApp, ctx := setupJailGateAppWithValidators(t, 2)
			ctx = ctx.WithBlockHeight(1000)
			validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
			require.NoError(t, err)
			validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
			require.NoError(t, err)
			consensus, err := validators[0].GetConsAddr()
			require.NoError(t, err)
			id := jailGateReactorId(t, bApp, ctx, validator)


			denom, err := bApp.StakingKeeper.BondDenom(ctx)
			require.NoError(t, err)
			staking := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
			addresses := make([]sdk.AccAddress, count)
			for i := range addresses {
				address := make([]byte, 20)
				address[0] = 0x73
				binary.BigEndian.PutUint64(address[12:], uint64(i+1))
				addresses[i] = sdk.AccAddress(address)
				delegate(t, bApp, ctx, staking, denom, addresses[i], validator, math.NewInt(1000000))
			}
			// Ensure the sampled owners have real powered structures. A zero-load
			// player being "online" with zero headroom proves nothing about play.
			samples := []sdk.AccAddress{addresses[0], addresses[len(addresses)-1]}
			var commandID string
			for i, address := range samples {
				cc := bApp.StructsKeeper.NewCurrentContext(ctx)
				player, err := cc.GetPlayerByAddress(address.String())
				require.NoError(t, err)
				command := player.GetFleet().GetCommandStruct()
				if i == 0 {
					commandID = command.ID()
					kind := command.GetStructType()
					kind.UnitDefenses = types.TechUnitDefenses_stealthMode
					kind.StealthActivateCharge = 1
					bApp.StructsKeeper.SetStructType(ctx, kind)
				}
				cc.CommitAll()
				require.True(t, player.IsOnline())
				require.Positive(t, player.GetStructsLoad())
			}
			message := &types.MsgStructStealthActivate{Creator: samples[0].String(), StructId: commandID}
			server := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
			healthy, _ := ctx.CacheContext()
			_, err = server.StructStealthActivate(healthy, message)
			require.NoError(t, err, "the exact gameplay action must work before the outage")
			original := bApp.StructsKeeper.GetAllReactorInfusions(ctx, id)
			beforeGrid := bApp.StructsKeeper.GetAllGridExport(ctx)
			require.NoError(t, bApp.StakingKeeper.Jail(ctx, consensus))
			metered := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
			_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(metered)
			require.NoError(t, err)
			costs = append(costs, metered.GasMeter().GasConsumed())
			t.Logf("positions=%d native_departure_gas=%d", count, metered.GasMeter().GasConsumed())
			require.Equal(t, original, bApp.StructsKeeper.GetAllReactorInfusions(ctx, id), "native departure may not walk and rewrite all positions")
			require.Equal(t, beforeGrid, bApp.StructsKeeper.GetAllGridExport(ctx))
			for _, address := range samples {
				cc := bApp.StructsKeeper.NewCurrentContext(ctx)
				player, err := cc.GetPlayerByAddress(address.String())
				require.NoError(t, err)
				require.Positive(t, player.GetCapacity(), "stored accounting has not caught up")
				require.False(t, player.IsOnline())
				require.Zero(t, player.GetAvailableCapacity())
				_, err = cc.NewAllocation(types.AllocationType_dynamic, id, "", address.String(), player.ID(), 1)
				require.ErrorContains(t, err, "unsupported_supply")
			}
			beforeStruct := bApp.StructsKeeper.GetAllStructAttributeExport(ctx)
			beforePlanet := bApp.StructsKeeper.GetAllPlanetAttributeExport(ctx)
			_, err = server.StructStealthActivate(ctx, message)
			require.ErrorContains(t, err, "offline")
			require.Equal(t, beforeGrid, bApp.StructsKeeper.GetAllGridExport(ctx))
			require.Equal(t, beforeStruct, bApp.StructsKeeper.GetAllStructAttributeExport(ctx))
			require.Equal(t, beforePlanet, bApp.StructsKeeper.GetAllPlanetAttributeExport(ctx))

			snapshot, found, err := bApp.StructsKeeper.GetReactorRepairSnapshot(ctx, id)
			require.NoError(t, err)
			require.True(t, found)
			require.Zero(t, snapshot.After)
			require.Equal(t, uint64(len(original)), snapshot.Through)
			bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
			gated := 0
			for _, before := range original {
				row, found := bApp.StructsKeeper.GetInfusion(ctx, id, before.Address)
				require.True(t, found)
				require.Equal(t, before.Fuel, row.Fuel)

				if row.Power == 0 {
					gated++
				}
			}
			require.Equal(t, types.ReactorSlashReconcileBudget, gated)
			require.Len(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx), 1)
			// A later deposit starts disabled and cannot enlarge this shutdown
			// pass, even though it is included in the later bonding transition.
			late := sdk.AccAddress("later_shutdown_entry")
			delegate(t, bApp, ctx, staking, denom, late, validator, math.NewInt(1000000))
			lateRow, found := bApp.StructsKeeper.GetInfusion(ctx, id, late.String())
			require.True(t, found)
			require.Zero(t, lateRow.Power)
			pending, _, err := bApp.StructsKeeper.GetReactorRepairSnapshot(ctx, id)
			require.NoError(t, err)
			require.Equal(t, snapshot.Through, pending.Through)
			require.NoError(t, bApp.StakingKeeper.Unjail(ctx, consensus))
			_, err = bApp.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
			require.NoError(t, err)
			rebond, _, err := bApp.StructsKeeper.GetReactorRepairSnapshot(ctx, id)
			require.NoError(t, err)
			require.Zero(t, rebond.After)
			require.Greater(t, rebond.Through, snapshot.Through)
			for i := 0; i < (len(original)+1+types.ReactorSlashReconcileBudget-1)/types.ReactorSlashReconcileBudget; i++ {
				bApp.StructsKeeper.ProcessReactorSlashReconcileQueue(ctx)
			}
			require.Empty(t, bApp.StructsKeeper.GetReactorSlashReconcileQueue(ctx))
			for _, before := range original {
				row, found := bApp.StructsKeeper.GetInfusion(ctx, id, before.Address)
				require.True(t, found)
				require.Equal(t, before, row, "rebond restores the original blend without stale shutdown deltas")
			}
			lateRow, found = bApp.StructsKeeper.GetInfusion(ctx, id, late.String())
			require.True(t, found)
			require.Equal(t, uint64(1000000), lateRow.Power)
			requireStakingAndDistributionHealthy(t, bApp, ctx)
		})
	}
	require.Len(t, costs, 2)
	// Validator token encodings may differ by a few bytes. Allow that constant
	// overhead, while rejecting even one extra per-delegator read/write loop.
	require.LessOrEqual(t, costs[1], costs[0]+1000)
}
