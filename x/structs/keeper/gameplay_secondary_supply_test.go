package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestGameplayHandlersRejectSecondarySupplyAfterCascade(t *testing.T) {
	for _, action := range []string{"activation", "stealth"} {
		for _, lost := range []bool{false, true} {
			name := action + "/healthy"
			if lost {
				name = action + "/upstream_lost"
			}
			t.Run(name, func(t *testing.T) {
				f := setupDestroyedStructFixture(t)
				f.ctx = f.ctx.WithBlockHeight(1000)
				f.defenseType.UnitDefenses = types.TechUnitDefenses_stealthMode
				f.k.SetStructType(f.ctx, f.defenseType)
				structure := f.appendStruct(true)
				station, _, err := testAppendSubstation(f.k, f.ctx, types.Allocation{}, f.player)
				require.NoError(t, err)
				testSupplySubstation(t, f.k, f.ctx, station.Id, 1_000_000)
				incoming := f.k.GetAllAllocationIdByDestinationIndex(f.ctx, station.Id)
				require.Len(t, incoming, 1)
				cc := f.k.NewCurrentContext(f.ctx)
				owner := cc.GetPlayer(f.player.Id)
				forward, err := cc.NewAllocation(types.AllocationType_dynamic, station.Id, "", f.player.Creator, f.player.Id, 900_000)
				require.NoError(t, err)
				tail, err := cc.NewSubstation(f.player.Creator, owner, forward)
				require.NoError(t, err)
				owner.MigrateSubstation(tail.ID())
				cc.SetGridAttribute(keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, owner.ID()), 0)
				if action == "stealth" {
					cc.GetStruct(structure.Id).GoOnline()
				}
				cc.CommitAll()
				require.True(t, owner.IsOnline())
				if lost {
					cc = f.k.NewCurrentContext(f.ctx)
					root, found := cc.GetAllocation(incoming[0])
					require.True(t, found)
					require.NoError(t, root.Destroy())
					cc.CommitAll()
					require.True(t, f.k.HasPendingGridCascade(f.ctx))
				}
				fresh := f.k.NewCurrentContext(f.ctx)
				require.Equal(t, uint64(810_000), fresh.GetPlayer(f.player.Id).GetCapacitySecondary())
                if lost {
                    for i := 0; f.k.HasPendingGridCascade(f.ctx) && i < 10; i++ { cascade := f.k.NewCurrentContext(f.ctx); cascade.GridCascade(); cascade.CommitAll() }
                    require.False(t, f.k.HasPendingGridCascade(f.ctx))
                }
				beforeGrid := f.k.GetAllGridExport(f.ctx)
				beforeStruct := f.k.GetAllStructAttributeExport(f.ctx)
				beforePlanet := f.k.GetAllPlanetAttributeExport(f.ctx)
				f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
				var response *types.MsgStructStatusResponse
				if action == "activation" {
					response, err = f.ms.StructActivate(f.ctx, &types.MsgStructActivate{Creator: f.player.Creator, StructId: structure.Id})
				} else {
					response, err = f.ms.StructStealthActivate(f.ctx, &types.MsgStructStealthActivate{Creator: f.player.Creator, StructId: structure.Id})
				}
				require.NotNil(t, response)
				if lost {
					require.ErrorContains(t, err, "offline")
					require.Equal(t, beforeGrid, f.k.GetAllGridExport(f.ctx), "refusal cannot discharge or add load")
					require.Equal(t, beforeStruct, f.k.GetAllStructAttributeExport(f.ctx))
					require.Equal(t, beforePlanet, f.k.GetAllPlanetAttributeExport(f.ctx))
				} else {
					require.NoError(t, err)
					fresh = f.k.NewCurrentContext(f.ctx)
					require.Equal(t, uint64(1000), fresh.GetPlayer(f.player.Id).GetLastAction())
					if action == "activation" {
						require.True(t, fresh.GetStruct(structure.Id).IsOnline())
					} else {
						require.True(t, fresh.GetStruct(structure.Id).IsHidden())
					}
				}
			})
		}
	}
}
