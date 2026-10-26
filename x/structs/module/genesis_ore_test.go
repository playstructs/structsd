package structs_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	structs "structs/x/structs/module"
	"structs/x/structs/types"
)

func TestGenesisPreservesRemainingPlanetOre(t *testing.T) {
	for _, height := range []int64{1, 101} {
		for _, status := range []types.PlanetStatus{types.PlanetStatus_active, types.PlanetStatus_complete} {
			for _, remaining := range []uint64{0, 1, types.PlanetStartingOre} {
				if status == types.PlanetStatus_complete && remaining != 0 {
					continue
				}
				t.Run(fmt.Sprintf("height=%d/status=%s/ore=%d", height, status, remaining), func(t *testing.T) {
					state := types.DefaultGenesis()
					state.PlanetList = []types.Planet{{Id: "2-1", Status: status}}
					oreID := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_ore, "2-1")
					state.GridList = []*types.GridRecord{{AttributeId: oreID, Value: remaining}}
					for restart := 0; restart < 2; restart++ {
						k, ctx := keepertest.StructsKeeper(t)
						ctx = ctx.WithBlockHeight(height)
						structs.InitGenesis(ctx, k, *state)
						require.Equal(t, remaining, k.GetGridAttribute(ctx, oreID), "restart must not replenish ore")
						planet, found := k.GetPlanet(ctx, "2-1")
						require.True(t, found)
						require.Equal(t, status, planet.Status)
						state = structs.ExportGenesis(ctx, k)
					}
				})
			}
		}
	}
}

func TestGenesisPlanetWithoutOreRecordKeepsLegacyDefault(t *testing.T) {
	state := types.DefaultGenesis()
	state.PlanetList = []types.Planet{{Id: "2-1", Status: types.PlanetStatus_active}, {Id: "2-2", Status: types.PlanetStatus_complete}}
	k, ctx := keepertest.StructsKeeper(t)
	structs.InitGenesis(ctx, k, *state)
	require.Equal(t, uint64(types.PlanetStartingOre), k.GetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_ore, "2-1")))
	require.Zero(t, k.GetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_ore, "2-2")))
}
