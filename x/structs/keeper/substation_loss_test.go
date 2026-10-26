package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestSubstationLossAllocationAdmission(t *testing.T) {
	for _, kind := range []types.AllocationType{types.AllocationType_static, types.AllocationType_dynamic, types.AllocationType_providerAgreement, types.AllocationType_automated} {
		t.Run(kind.String(), func(t *testing.T) {
			k, ctx := keepertest.StructsKeeper(t)
			k.SetSubstation(ctx, types.Substation{Id: "4-1", Owner: "1-1"})
			k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, "4-1"), 1000)
			cc := k.NewCurrentContext(ctx)
			if kind != types.AllocationType_automated {
				_, err := cc.NewAllocation(kind, "4-1", "", "", "1-1", 901)
				require.Error(t, err, "cannot export the station's operating power")
			}
			allocation, err := cc.NewAllocation(kind, "4-1", "", "", "1-1", 900)
			require.NoError(t, err)
			require.Equal(t, uint64(900), allocation.GetPower())
			if kind == types.AllocationType_dynamic {
				_, err = allocation.SetDynamicPower(901)
				require.Error(t, err)
				_, err = allocation.SetDynamicPower(800)
				require.NoError(t, err)
				_, err = allocation.SetDynamicPower(900)
				require.NoError(t, err)
			}
			require.Zero(t, cc.GetSubstation("4-1").GetAvailableCapacity())
			require.Equal(t, uint64(1000), cc.GetSubstation("4-1").GetCapacity())
		})
	}
}

func TestSubstationLossCompoundsAndSharesRemainingPower(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	for _, id := range []string{"4-1", "4-2", "4-3"} {
		k.SetSubstation(ctx, types.Substation{Id: id, Owner: "1-1"})
	}
	k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, "4-1"), 1000)
	k.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCount, "4-3"), 3)
	cc := k.NewCurrentContext(ctx)
	_, err := cc.NewAllocation(types.AllocationType_dynamic, "4-1", "4-2", "", "1-1", 900)
	require.NoError(t, err)
	_, err = cc.NewAllocation(types.AllocationType_automated, "4-2", "4-3", "", "1-1", 0)
	require.NoError(t, err)
	last := cc.GetSubstation("4-3")
	require.Equal(t, uint64(810), last.GetCapacity())
	require.Equal(t, uint64(729), last.GetUsableCapacity())
	_, err = cc.NewAllocation(types.AllocationType_dynamic, "4-3", "", "", "1-1", 600)
	require.NoError(t, err)
	require.Equal(t, uint64(43), last.GetConnectionCapacity(), "(729 - 600) / 3")
	cc.CommitAll()
	fresh := k.NewCurrentContext(ctx).GetSubstation("4-3")
	require.Equal(t, uint64(810), fresh.GetCapacity(), "loss is never stored back over raw input")
	require.Equal(t, uint64(43), fresh.GetConnectionCapacity())
}

func TestSubstationLossRoundsAggregateInput(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	k.SetSubstation(ctx, types.Substation{Id: "4-1", Owner: "1-1"})
	cc := k.NewCurrentContext(ctx)
	for _, id := range []string{"6-1", "6-2"} {
		cc.GenesisImportAllocation(types.Allocation{Id: id, SourceObjectId: "1-1", DestinationId: "4-1", Type: types.AllocationType_static}, 1)
	}
	require.Equal(t, uint64(2), cc.GetSubstation("4-1").GetCapacity())
	require.Equal(t, uint64(1), cc.GetSubstation("4-1").GetAvailableCapacity(), "do not round each one-unit input to zero")
}
