package keeper_test

import (
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func TestSubstationDeletionBoundsPermissionlessDustAndClosesAdmission(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	address := sdk.AccAddress("deletion_owner_addr_")
	owner := testAppendPlayer(k, ctx, types.Player{Creator: address.String(), PrimaryAddress: address.String()})
	stationID := "4-900"
	k.SetSubstation(ctx, types.Substation{Id: stationID, Owner: owner.Id})
	cc := k.NewCurrentContext(ctx)
	const count = 2200
	for i := 0; i < count; i++ {
		cc.GenesisImportAllocation(types.Allocation{Id: fmt.Sprintf("6-%d", 10000+i), SourceObjectId: owner.Id, DestinationId: stationID, Controller: owner.Id, Type: types.AllocationType_static}, 1)
	}
	cc.CommitAll()
	metered := ctx.WithGasMeter(storetypes.NewGasMeter(20_000_000))
	_, err := keeperlib.NewMsgServerImpl(k).SubstationDelete(metered, &types.MsgSubstationDelete{Creator: owner.Creator, SubstationId: stationID})
	require.NoError(t, err)
	t.Logf("delete admission gas with %d inbound edges: %d", count, metered.GasMeter().GasConsumed())
	station, found := k.GetSubstation(ctx, stationID)
	require.True(t, found)
	require.True(t, station.DeletionPending)
	pendingEvents := 0
	for _, event := range ctx.EventManager().ABCIEvents() {
		if event.Type == "structs.structs.EventSubstation" {
			parsed, parseErr := sdk.ParseTypedEvent(event)
			require.NoError(t, parseErr)
			row := parsed.(*types.EventSubstation).Substation
			if row.Id == stationID && row.DeletionPending {
				require.Equal(t, station, *row)
				require.Len(t, event.Attributes, 1)
				require.Contains(t, event.Attributes[0].Value, `"deletion_pending":true`, "document the actual chain JSON spelling")
				pendingEvents++
			}
		}
		if event.Type == "structs.structs.EventDelete" {
			parsed, parseErr := sdk.ParseTypedEvent(event)
			require.NoError(t, parseErr)
			require.NotEqual(t, stationID, parsed.(*types.EventDelete).ObjectId, "pending is not deleted")
		}
	}
	require.Equal(t, 1, pendingEvents)
	require.Len(t, k.GetAllAllocationIdByDestinationIndex(ctx, stationID), count-8)
	// A new context simulates the next operation; the persisted closing flag is
	// sufficient to block replenishment without an in-memory deny list.
	cc = k.NewCurrentContext(ctx)
	destination := cc.GetSubstation(stationID)
	require.Zero(t, destination.GetAvailableCapacity())
	require.Zero(t, destination.GetConnectionCapacity())
	require.Error(t, destination.CheckAcceptingConnections())
	require.Error(t, destination.CanManageConnectionsBy(cc.GetPlayer(owner.Id)))
	cc.GetPlayer(owner.Id).MigrateSubstation(stationID)
	require.Empty(t, cc.GetPlayer(owner.Id).GetSubstationId())
	allocation, found := cc.GetAllocation("6-10020")
	require.True(t, found)
	require.NoError(t, allocation.SetDestination(""))
	require.Error(t, allocation.SetDestination(stationID))
	_, err = cc.NewAllocation(types.AllocationType_automated, stationID, "", owner.Creator, owner.Id, 0)
	require.Error(t, err)
	cc.CommitAll()
	for block := 0; block < 30; block++ {
		before := len(k.GetAllAllocationIdByDestinationIndex(ctx, stationID))
		cc = k.NewCurrentContext(ctx)
		cc.GridCascade()
		cc.CommitAll()
		after := len(k.GetAllAllocationIdByDestinationIndex(ctx, stationID))
		require.LessOrEqual(t, before-after, types.GridCascadeBlockBudget)
		_, found = k.GetSubstation(ctx, stationID)
		if !found {
			require.Zero(t, after)
			deletedEvents := 0
			for _, event := range ctx.EventManager().ABCIEvents() {
				if event.Type == "structs.structs.EventDelete" {
					parsed, parseErr := sdk.ParseTypedEvent(event)
					require.NoError(t, parseErr)
					if parsed.(*types.EventDelete).ObjectId == stationID {
						deletedEvents++
					}
				}
			}
			require.Equal(t, 1, deletedEvents, "publish final deletion exactly once")
			return
		}
	}
	t.Fatal("bounded deletion did not finish")
}

func TestSubstationDeletionMigratesPlayersInBatchesAndHandlesClosedTarget(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	address := sdk.AccAddress("deletion_owner_addr_")
	owner := testAppendPlayer(k, ctx, types.Player{Creator: address.String(), PrimaryAddress: address.String()})
	for _, id := range []string{"4-900", "4-901"} {
		k.SetSubstation(ctx, types.Substation{Id: id, Owner: owner.Id})
	}
	cc := k.NewCurrentContext(ctx)
	for i := 0; i < 25; i++ {
		p := testAppendPlayer(k, ctx, types.Player{})
		cc.GetPlayer(p.Id).MigrateSubstation("4-900")
	}
	cc.CommitAll()
	cc = k.NewCurrentContext(ctx)
	cc.GetSubstation("4-900").Delete("4-901")
	cc.CommitAll()
	require.Len(t, k.GetAllPlayerIdBySubstationIndex(ctx, "4-900"), 17)
	require.Len(t, k.GetAllPlayerIdBySubstationIndex(ctx, "4-901"), 8)
	// If the previously authorized target closes before the next batch, its
	// remaining migrants disconnect instead of recreating a deleted station.
	cc = k.NewCurrentContext(ctx)
	cc.GetSubstation("4-901").Delete("")
	cc.CommitAll()
	cc = k.NewCurrentContext(ctx)
	cc.GridCascade()
	cc.CommitAll()
	for _, id := range []string{"4-900", "4-901"} {
		_, found := k.GetSubstation(ctx, id)
		require.False(t, found)
		require.Empty(t, k.GetAllPlayerIdBySubstationIndex(ctx, id))
	}
}
