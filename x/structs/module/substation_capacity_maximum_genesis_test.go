package structs_test

import (
	"testing"

	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	structs "structs/x/structs/module"
	"structs/x/structs/types"
)

func TestGenesis_SubstationConnectionCapacityMaximumRoundTrip(t *testing.T) {
	for _, maximum := range []uint64{0, 200} {
		genesis := types.DefaultGenesis()
		genesis.SubstationList = []types.Substation{{Id: "4-1", Owner: "1-1"}}
		genesis.AllocationList = []types.Allocation{{Id: "6-1", SourceObjectId: "1-1", DestinationId: "4-1", Type: types.AllocationType_static}}
		maximumID := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacityMaximum, "4-1")
		genesis.GridList = []*types.GridRecord{
			{AttributeId: keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_power, "6-1"), Value: 1000},
			{AttributeId: maximumID, Value: maximum},
			// Derived share must be rebuilt rather than trusted on import.
			{AttributeId: keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, "4-1"), Value: 9999},
		}
		for pass := 0; pass < 2; pass++ {
			k, ctx := keepertest.StructsKeeper(t)
			structs.InitGenesis(ctx, k, *genesis)
			station := k.NewCurrentContext(ctx).GetSubstation("4-1")
			require.Equal(t, maximum, station.GetConnectionCapacityMaximum())
			want := uint64(900)
			if maximum != 0 {
				want = maximum
			}
			require.Equal(t, want, station.GetConnectionCapacity())
			exported := structs.ExportGenesis(ctx, k)
			wire, err := proto.Marshal(exported)
			require.NoError(t, err)
			genesis = &types.GenesisState{}
			require.NoError(t, proto.Unmarshal(wire, genesis))
		}
	}
}
