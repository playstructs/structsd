package structs_test

import (
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	structs "structs/x/structs/module"
	"structs/x/structs/types"
	"testing"
)

func TestGenesis_SubstationDeletionSurvivesWireExportAndMissingQueue(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	structs.InitGenesis(ctx, k, *types.DefaultGenesis())
	k.SetSubstation(ctx, types.Substation{Id: "4-90", Owner: "1-1", DeletionPending: true, DeletionMigrationSubstationId: "4-91"})
	k.SetSubstationCount(ctx, 92)
	// A stale incoming index is still bounded work and must not strand cleanup.
	k.SetAllocationDestinationIndex(ctx, "4-90", "6-99")
	k.AppendGridCascadeQueue(ctx, "4-90")
	exported := structs.ExportGenesis(ctx, k)
	wire, err := proto.Marshal(exported)
	require.NoError(t, err)
	var decoded types.GenesisState
	require.NoError(t, proto.Unmarshal(wire, &decoded))
	for _, omitQueue := range []bool{false, true} {
		if omitQueue {
			decoded.GridCascadeQueue = nil
		}
		k2, ctx2 := keepertest.StructsKeeper(t)
		structs.InitGenesis(ctx2, k2, decoded)
		station, found := k2.GetSubstation(ctx2, "4-90")
		require.True(t, found)
		require.True(t, station.DeletionPending)
		require.Equal(t, "4-91", station.DeletionMigrationSubstationId)
		require.Contains(t, k2.GetGridCascadeQueueExport(ctx2), "4-90")
		cc := k2.NewCurrentContext(ctx2)
		cc.GridCascade()
		cc.CommitAll()
		_, found = k2.GetSubstation(ctx2, "4-90")
		require.False(t, found)
		require.Zero(t, k2.GetGridAttribute(ctx2, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, "4-90")))
	}
}
