package keeper_test

import (
	"testing"

	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"

	"github.com/stretchr/testify/require"
)

func createTestAllocation(sourceId string, destId string, allocationType types.AllocationType) types.Allocation {
	return types.Allocation{
		SourceObjectId: sourceId,
		DestinationId:  destId,
		Type:           allocationType,
	}
}

func TestAppendAllocation(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Test static allocation
	sourceId := "source1"
	destId := "dest1"
	power := uint64(100)

	// Set up source capacity
	keeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, sourceId), uint64(200))

	allocation := createTestAllocation(sourceId, destId, types.AllocationType_static)
	appendedAlloc, actualPower, err := keeper.AppendAllocation(ctx, allocation, power)

	require.NoError(t, err)
	require.Equal(t, power, actualPower)
	require.NotEmpty(t, appendedAlloc.Id)
	require.Equal(t, sourceId, appendedAlloc.SourceObjectId)
	require.Equal(t, destId, appendedAlloc.DestinationId)
	require.Equal(t, types.AllocationType_static, appendedAlloc.Type)

	// Test automated allocation
	automatedAlloc := createTestAllocation(sourceId, destId, types.AllocationType_automated)
	appendedAutoAlloc, autoPower, err := keeper.AppendAllocation(ctx, automatedAlloc, 0)

	require.NoError(t, err)
	require.Equal(t, uint64(200), autoPower) // Should use full capacity
	require.NotEmpty(t, appendedAutoAlloc.Id)
	require.Equal(t, types.AllocationType_automated, appendedAutoAlloc.Type)
}

func TestSetAllocation(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Create initial allocation
	sourceId := "source1"
	destId := "dest1"
	power := uint64(100)

	keeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, sourceId), uint64(200))

	allocation := createTestAllocation(sourceId, destId, types.AllocationType_static)
	appendedAlloc, _, err := keeper.AppendAllocation(ctx, allocation, power)
	require.NoError(t, err)

	// Test updating allocation
	newDestId := "dest2"
	appendedAlloc.DestinationId = newDestId
	updatedAlloc, newPower, err := keeper.SetAllocation(ctx, appendedAlloc, power)

	require.NoError(t, err)
	require.Equal(t, power, newPower)
	require.Equal(t, newDestId, updatedAlloc.DestinationId)
}

func TestSetAllocationOnly(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	allocation := createTestAllocation("source1", "dest1", types.AllocationType_static)
	allocation.Id = "test-id"

	updatedAlloc, err := keeper.SetAllocationOnly(ctx, allocation)
	require.NoError(t, err)
	require.Equal(t, allocation.Id, updatedAlloc.Id)
}

func TestImportAllocation(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	allocation := createTestAllocation("source1", "dest1", types.AllocationType_static)
	allocation.Id = "import-id"

	keeper.ImportAllocation(ctx, allocation)

	// Verify allocation was imported
	importedAlloc, found := keeper.GetAllocation(ctx, allocation.Id)
	require.True(t, found)
	require.Equal(t, allocation.Id, importedAlloc.Id)
}

func TestRemoveAndDestroyAllocation(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Create and append allocation
	sourceId := "source1"
	destId := "dest1"
	power := uint64(100)

	keeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, sourceId), uint64(200))

	allocation := createTestAllocation(sourceId, destId, types.AllocationType_static)
	appendedAlloc, _, err := keeper.AppendAllocation(ctx, allocation, power)
	require.NoError(t, err)

	// Test RemoveAllocation
	keeper.RemoveAllocation(ctx, appendedAlloc.Id)
	_, found := keeper.GetAllocation(ctx, appendedAlloc.Id)
	require.False(t, found)

	// Test DestroyAllocation
	allocation = createTestAllocation(sourceId, destId, types.AllocationType_static)
	appendedAlloc, _, err = keeper.AppendAllocation(ctx, allocation, power)
	require.NoError(t, err)

	destroyed := keeper.DestroyAllocation(ctx, appendedAlloc.Id)
	require.True(t, destroyed)

	_, found = keeper.GetAllocation(ctx, appendedAlloc.Id)
	require.False(t, found)
}

func TestGetAllocation(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Create test allocation
	sourceId := "source1"
	destId := "dest1"
	power := uint64(100)

	keeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, sourceId), uint64(200))

	allocation := createTestAllocation(sourceId, destId, types.AllocationType_static)
	appendedAlloc, _, err := keeper.AppendAllocation(ctx, allocation, power)
	require.NoError(t, err)

	// Test GetAllocation
	retrievedAlloc, found := keeper.GetAllocation(ctx, appendedAlloc.Id)
	require.True(t, found)
	require.Equal(t, appendedAlloc.Id, retrievedAlloc.Id)
	require.Equal(t, sourceId, retrievedAlloc.SourceObjectId)
	require.Equal(t, destId, retrievedAlloc.DestinationId)
}

func TestGetAllAllocation(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Create multiple allocations
	sourceId := "source1"
	keeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, sourceId), uint64(200))

	allocations := []types.Allocation{
		createTestAllocation(sourceId, "dest1", types.AllocationType_static),
		createTestAllocation(sourceId, "dest2", types.AllocationType_static),
		createTestAllocation(sourceId, "dest3", types.AllocationType_automated),
	}

	for _, alloc := range allocations {
		_, _, err := keeper.AppendAllocation(ctx, alloc, 100)
		require.NoError(t, err)
	}

	// Test GetAllAllocation
	allAllocations := keeper.GetAllAllocation(ctx)
	require.Len(t, allAllocations, len(allocations))
}
