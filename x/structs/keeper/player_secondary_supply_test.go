package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestPlayerSecondaryPowerFollowsBoundedCascade(t *testing.T) {
	for _, direct := range []uint64{0, 3} {
		t.Run(sdk.NewInt64Coin("power", int64(direct)).String(), func(t *testing.T) {
			chain := buildCascadeChain(t, 3)
			address := sdk.AccAddress("secondary-supply-consumer-000001")
			player := testAppendPlayer(chain.k, chain.ctx, types.Player{Creator: address.String(), PrimaryAddress: address.String()})
			tail := chain.substationIds[len(chain.substationIds)-1]
			cc := chain.k.NewCurrentContext(chain.ctx)
			consumer := cc.GetPlayer(player.Id)
			consumer.MigrateSubstation(tail)
			cc.SetGridAttribute(keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id), direct)
			cc.SetGridAttribute(keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, player.Id), 1)
			cc.CommitAll()
			require.Equal(t, uint64(1), consumer.GetCapacitySecondary())
			require.True(t, consumer.IsOnline())
			require.Equal(t, direct, consumer.GetAvailableCapacity())
			chain.pullTheRoot(t)
			require.True(t, chain.k.HasPendingGridCascade(chain.ctx))
			before := chain.k.GetAllGridExport(chain.ctx)
			cc = chain.k.NewCurrentContext(chain.ctx)
			consumer = cc.GetPlayer(player.Id)
			require.Equal(t, uint64(2), cc.GetSubstation(tail).GetCapacity(), "the loss has not reached the tail's stored accounting")
			require.Equal(t, uint64(1), consumer.GetCapacitySecondary(), "the raw connection share remains for cascade accounting")
			require.True(t, consumer.IsOnline(), "cascade has not reached this station yet")
            require.Equal(t, direct, consumer.GetAvailableCapacity())

			cc.CommitAll()
			require.Equal(t, before, chain.k.GetAllGridExport(chain.ctx), "readiness checks cannot apply the cascade's deltas early")
			for i := 0; chain.k.HasPendingGridCascade(chain.ctx) && i < 10; i++ {
				chain.cascadeBlock(t)
			}
			require.False(t, chain.k.HasPendingGridCascade(chain.ctx))
			consumer = chain.k.NewCurrentContext(chain.ctx).GetPlayer(player.Id)
			require.Zero(t, consumer.GetCapacitySecondary())
			require.Equal(t, direct > 0, consumer.IsOnline())
		})
	}
}
