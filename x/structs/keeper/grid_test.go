package keeper_test

import (
	"testing"

	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"

	"github.com/stretchr/testify/require"
)

func TestGridAttributeOperations(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Test setting and getting grid attributes
	objectId := "test-object"
	attributeId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_power, objectId)

	// Test initial value (should be 0)
	initialValue := keeper.GetGridAttribute(ctx, attributeId)
	require.Equal(t, uint64(0), initialValue)

	// Test setting value
	expectedValue := uint64(100)
	keeper.SetGridAttribute(ctx, attributeId, expectedValue)
	actualValue := keeper.GetGridAttribute(ctx, attributeId)
	require.Equal(t, expectedValue, actualValue)

	// Test clearing value
	keeper.ClearGridAttribute(ctx, attributeId)
	clearedValue := keeper.GetGridAttribute(ctx, attributeId)
	require.Equal(t, uint64(0), clearedValue)
}

func TestGridAttributeDelta(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	objectId := "test-object"
	attributeId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_power, objectId)

	// Set initial value
	initialValue := uint64(100)
	keeper.SetGridAttribute(ctx, attributeId, initialValue)

	// Test delta update
	oldAmount := uint64(50)
	newAmount := uint64(75)
	amount, err := keeper.SetGridAttributeDelta(ctx, attributeId, oldAmount, newAmount)
	require.NoError(t, err)
	require.Equal(t, uint64(125), amount) // 100 - 50 + 75 = 125

	// Verify final value
	finalValue := keeper.GetGridAttribute(ctx, attributeId)
	require.Equal(t, uint64(125), finalValue)
}

func TestGridAttributeIncrementDecrement(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	objectId := "test-object"
	attributeId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_power, objectId)

	// Set initial value
	initialValue := uint64(100)
	keeper.SetGridAttribute(ctx, attributeId, initialValue)

	// Test increment
	incrementAmount := uint64(50)
	amount := keeper.SetGridAttributeIncrement(ctx, attributeId, incrementAmount)
	require.Equal(t, uint64(150), amount)

	// Test decrement
	decrementAmount := uint64(30)
	amount, err := keeper.SetGridAttributeDecrement(ctx, attributeId, decrementAmount)
	require.NoError(t, err)
	require.Equal(t, uint64(120), amount)
}

func TestGridCache(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	objectId := "test-object"
	cache := keeper.GetGridCacheFromId(ctx, objectId)

	// Test initial state
	require.Equal(t, objectId, cache.GetObjectId())
	require.False(t, cache.IsChanged())

	// Test loading attributes
	require.Equal(t, uint64(0), cache.GetPower())
	require.Equal(t, uint64(0), cache.GetCapacity())
	require.Equal(t, uint64(0), cache.GetLoad())

	// Test setting attributes
	cache.LoadPower()
	cache.LoadCapacity()
	cache.LoadLoad()

	// Verify attributes are loaded
	require.True(t, cache.PowerLoaded)
	require.True(t, cache.CapacityLoaded)
	require.True(t, cache.LoadLoaded)
}

func TestGridCascadeQueue(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Test appending to queue
	queueId := "test-queue"
	err := keeper.AppendGridCascadeQueue(ctx, queueId)
	require.NoError(t, err)

	// Test getting queue
	queue := keeper.GetGridCascadeQueue(ctx, false)
	require.Contains(t, queue, queueId)

	// Test clearing queue
	clearedQueue := keeper.GetGridCascadeQueue(ctx, true)
	require.Contains(t, clearedQueue, queueId)

	// Verify queue is empty after clearing
	emptyQueue := keeper.GetGridCascadeQueue(ctx, false)
	require.Empty(t, emptyQueue)
}

func TestGridAttributesByObject(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	objectId := "test-object"

	// Set some test values
	keeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_power, objectId), 100)
	keeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, objectId), 200)
	keeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_load, objectId), 50)

	// Get all attributes
	attributes := keeper.GetGridAttributesByObject(ctx, objectId)

	// Verify values
	require.Equal(t, uint64(100), attributes.Power)
	require.Equal(t, uint64(200), attributes.Capacity)
	require.Equal(t, uint64(50), attributes.Load)
}

func TestGridConnectionCapacity(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	objectId := "test-object"

	// Set initial values
	keeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, objectId), 1000)
	keeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_load, objectId), 400)
	keeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCount, objectId), 2)

	// Update connection capacity
	keeper.UpdateGridConnectionCapacity(ctx, objectId)

	// Verify connection capacity
	connectionCapacity := keeper.GetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, objectId))
	require.Equal(t, uint64(300), connectionCapacity) // (1000 - 400) / 2 = 300
}
