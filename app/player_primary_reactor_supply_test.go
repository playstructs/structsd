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

func TestPlayerReadinessExcludesStalePrimaryReactorPower(t *testing.T) {
	for _, tc := range []struct {
		name   string
		slash  bool
		backup bool
	}{
		{"jail", false, false},
		{"slash", true, false},
		{"jail-with-healthy-backup", false, true},
		{"slash-with-healthy-backup", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bApp, ctx := setupJailGateAppWithValidators(t, 2)
			validators, err := bApp.StakingKeeper.GetLastValidators(ctx)
			require.NoError(t, err)
			validator, err := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
			require.NoError(t, err)
			primary, _, playerID := newTransferPlayer(t, bApp, ctx)
			denom, err := bApp.StakingKeeper.BondDenom(ctx)
			require.NoError(t, err)
			delegate(t, bApp, ctx, stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper), denom, primary, validator, math.NewInt(1000000))
			reactorID := jailGateReactorId(t, bApp, ctx, validator)
			cc := bApp.StructsKeeper.NewCurrentContext(ctx)
			cc.CommitAll()
			backupShare := uint64(0)
			if tc.backup {
				backupValidator, err := sdk.ValAddressFromBech32(validators[1].OperatorAddress)
				require.NoError(t, err)
				delegate(t, bApp, ctx, stakingkeeper.NewMsgServerImpl(bApp.StakingKeeper), denom, primary, backupValidator, math.NewInt(1000000))
				backupID := jailGateReactorId(t, bApp, ctx, backupValidator)
				cc = bApp.StructsKeeper.NewCurrentContext(ctx)

				cc.CommitAll()
				backup, found := bApp.StructsKeeper.GetInfusion(ctx, backupID, primary.String())
				require.True(t, found)
				_, _, backupShare = backup.GetPowerDistribution()
			}

			infusion, found := bApp.StructsKeeper.GetInfusion(ctx, reactorID, primary.String())
			require.True(t, found)
			_, _, originalShare := infusion.GetPowerDistribution()
			cc = bApp.StructsKeeper.NewCurrentContext(ctx)
			existing, err := cc.NewAllocation(types.AllocationType_dynamic, playerID, "", primary.String(), playerID, 2)
			require.NoError(t, err)
			allocationID := existing.ID()
			cc.CommitAll()
			cc = bApp.StructsKeeper.NewCurrentContext(ctx)
			player := cc.GetPlayer(playerID)
			baselineLoad := player.GetLoad() + player.GetStructsLoad()
			storedBonus := uint64(0)
			if tc.backup {
				storedBonus = (originalShare + backupShare) / 18
			} // equal pair: 1/18 of base
			storedTotal := originalShare + backupShare + storedBonus
			require.Equal(t, storedTotal-baselineLoad, player.GetAvailableCapacity())
			// A permissionless restart request is not an eligibility transition.
			bApp.StructsKeeper.EnqueueReactorRestart(ctx, reactorID)
			require.Equal(t, storedTotal-baselineLoad, player.GetAvailableCapacity())
			value, err := bApp.StakingKeeper.GetValidator(ctx, validator)
			require.NoError(t, err)
			consensus, err := value.GetConsAddr()
			require.NoError(t, err)
			expected := uint64(0)
			if tc.slash {
				_, err = bApp.StakingKeeper.Slash(ctx, consensus, ctx.BlockHeight(), value.ConsensusPower(bApp.StakingKeeper.PowerReduction(ctx)), math.LegacyNewDecWithPrec(5, 2))
				require.NoError(t, err)
				// 950,000 remaining fuel uses the unchanged commission split.
				projected := infusion
				projected.Power = 950000
				_, _, expected = projected.GetPowerDistribution()
			} else {
				require.NoError(t, bApp.StakingKeeper.Jail(ctx, consensus))
			}
			if expected != 0 && backupShare != 0 {
				// Fixture values are below one million; these products fit uint64.
				liveBonus := (expected + backupShare) * expected * backupShare / (9 * (expected*expected + backupShare*backupShare))
				expected += min(storedBonus, liveBonus)
			}
			expected += backupShare
			beforeGrid := bApp.StructsKeeper.GetAllGridExport(ctx)
			beforeInfusions := bApp.StructsKeeper.GetAllInfusion(ctx)
			beforeEvents := len(ctx.EventManager().Events())
			// Reuse the context that observed healthy power: no positive result
			// or validator state may hide a subsequent live staking transition.
			require.Equal(t, storedTotal, player.GetCapacity())
			available := uint64(0)
			if expected > baselineLoad {
				available = expected - baselineLoad
			}
			require.Equal(t, available, player.GetAvailableCapacity())
			require.False(t, player.CanSupportLoadAddition(available+1))
			beforeCount := bApp.StructsKeeper.GetAllocationCount(ctx)
			beforeAllocations := bApp.StructsKeeper.GetAllAllocation(ctx)
			_, err = cc.NewAllocation(types.AllocationType_dynamic, playerID, "", primary.String(), playerID, player.GetCapacity()-player.GetLoad())
			require.ErrorContains(t, err, "does not have capacity")
			existing, found = cc.GetAllocation(allocationID)
			require.True(t, found)
			_, err = existing.SetDynamicPower(player.GetCapacity() - player.GetLoad() + existing.GetPower())
			require.ErrorContains(t, err, "does not have capacity")
			require.Equal(t, beforeCount, bApp.StructsKeeper.GetAllocationCount(ctx))
			require.Equal(t, beforeAllocations, bApp.StructsKeeper.GetAllAllocation(ctx))
			cc.CommitAll()
			require.Equal(t, beforeGrid, bApp.StructsKeeper.GetAllGridExport(ctx))
			require.Equal(t, beforeInfusions, bApp.StructsKeeper.GetAllInfusion(ctx))
			require.Equal(t, beforeEvents, len(ctx.EventManager().Events()))
			// Reducing an existing allocation remains available while supply
			// repair is pending. Use isolated SDK caches for positive controls.
			probe, _ := ctx.CacheContext()
			repair := bApp.StructsKeeper.NewCurrentContext(probe)
			shrinking, found := repair.GetAllocation(allocationID)
			require.True(t, found)
			_, err = shrinking.SetDynamicPower(1)
			require.NoError(t, err)
			repair.CommitAll()
			stored, found := bApp.StructsKeeper.NewCurrentContext(probe).GetAllocation(allocationID)
			require.True(t, found)
			require.Equal(t, uint64(1), stored.GetPower())
			// A new automated allocation would consume all stored capacity,
			// so it must wait until the unavailable contribution is settled.
			repair = bApp.StructsKeeper.NewCurrentContext(probe)
			shrinking, found = repair.GetAllocation(allocationID)
			require.True(t, found)
			require.NoError(t, shrinking.Destroy())
			repair.CommitAll()
			repair = bApp.StructsKeeper.NewCurrentContext(probe)
			count := bApp.StructsKeeper.GetAllocationCount(probe)
			_, err = repair.NewAllocation(types.AllocationType_automated, playerID, "", primary.String(), playerID, 0)
			require.ErrorContains(t, err, "unsupported_supply")
			repair.CommitAll()
			require.Equal(t, count, bApp.StructsKeeper.GetAllocationCount(probe))
			_, hooked := bApp.StructsKeeper.GetAutoResizeAllocationBySource(probe, playerID)
			require.False(t, hooked)
			if expected > player.GetLoad() {
				probe, _ = ctx.CacheContext()
				usable := bApp.StructsKeeper.NewCurrentContext(probe)
				power := expected - player.GetLoad()
				created, err := usable.NewAllocation(types.AllocationType_dynamic, playerID, "", primary.String(), playerID, power)
				require.NoError(t, err, "remaining backed power must stay allocatable")
				usable.CommitAll()
				stored, found = bApp.StructsKeeper.NewCurrentContext(probe).GetAllocation(created.ID())
				require.True(t, found)
				require.Equal(t, power, stored.GetPower())
			}
			cc = bApp.StructsKeeper.NewCurrentContext(ctx)
			cc.SetGridAttribute(keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, playerID), storedTotal-cc.GetPlayer(playerID).GetLoad())
			cc.CommitAll()
			require.False(t, bApp.StructsKeeper.NewCurrentContext(ctx).GetPlayer(playerID).IsOnline())
		})
	}
}
