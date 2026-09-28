package app_test

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	"structs/app/ante"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

// This is an opt-in cost measurement, not a performance acceptance threshold.
// Real staking/distribution and distinct players expose the per-infusion writes
// hidden by a one-owner fixture or a staking mock. The full-scan helper costs
// are legacy baselines; TestNativeReactorShutdownQueuesBoundedCleanupAndRebond
// measures the active native departure path. Setup is outside each timing.
func TestReactorEnergyFanoutMeasurement(t *testing.T) {
	if os.Getenv("STRUCTS_REACTOR_FANOUT_MEASURE") != "1" {
		t.Skip("set STRUCTS_REACTOR_FANOUT_MEASURE=1 to measure reactor lifecycle costs")
	}
	for _, count := range []int{100, 1000, 5000} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			bApp, ctx := setupJailGateApp(t)
			validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
			require.NoError(t, err)
			validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
			require.NoError(t, err)
			reactorID := jailGateReactorId(t, bApp, ctx, validator)
			denom, err := bApp.StakingKeeper.BondDenom(ctx)
			require.NoError(t, err)


			staking := stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper)
			addresses := make([]sdk.AccAddress, count)
			for i := range addresses {
				address := make([]byte, 20)
				address[0] = 0x79
				binary.BigEndian.PutUint64(address[12:], uint64(i+1))
				addresses[i] = sdk.AccAddress(address)
				delegate(t, bApp, ctx, staking, denom, addresses[i], validator, math.NewInt(1000))
			}
			checkSamples := func(power uint64) {
				for _, address := range []sdk.AccAddress{addresses[0], addresses[len(addresses)-1]} {
					row, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, address.String())
					require.True(t, found)
					require.Equal(t, uint64(1000), row.Fuel)

					require.Equal(t, power, row.Power)
				}
			}
			measure := func(name string, operation func(sdk.Context) error) {
				metered := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
				started := time.Now()
				err := operation(metered)
				elapsed := time.Since(started)
				require.NoError(t, err)
				encoded, err := json.Marshal(metered.EventManager().Events())
				require.NoError(t, err)
				t.Logf("players=%d operation=%s gas=%d elapsed_ms=%.3f events=%d event_json_bytes=%d",
					count, name, metered.GasMeter().GasConsumed(), float64(elapsed.Microseconds())/1000,
					len(metered.EventManager().Events()), len(encoded))
			}
			checkSamples(1000)
			measure("legacy_gate_helper", func(ctx sdk.Context) error {
				bApp.StructsKeeper.ReactorGateEnergy(ctx, validator)
				return nil
			})
			checkSamples(0)
			measure("legacy_restore_helper", func(ctx sdk.Context) error {
				bApp.StructsKeeper.ReactorRestoreEnergy(ctx, validator)
				return nil
			})
			checkSamples(1000)
			ms := keeperlib.NewMsgServerImpl(bApp.StructsKeeper)
			restart := &types.MsgReactorRestart{Creator: addresses[0].String(), ValidatorAddress: validator.String()}
			require.False(t, ante.IsFreeTransaction([]sdk.Msg{restart}), "interpret restart cost against paid gas, not DefaultFreeGasCap")
			measure("unchanged_restart", func(ctx sdk.Context) error {
				_, err := ms.ReactorRestart(ctx, restart)
				return err
			})
			checkSamples(1000)
			requireStakingAndDistributionHealthy(t, bApp, ctx)
		})
	}
}
