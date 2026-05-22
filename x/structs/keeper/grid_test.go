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

	// Simulate delta: current - oldAmount + newAmount = 100 - 50 + 75 = 125
	current := keeper.GetGridAttribute(ctx, attributeId)
	newVal := current - uint64(50) + uint64(75)
	keeper.SetGridAttribute(ctx, attributeId, newVal)

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

	// Increment manually
	current := keeper.GetGridAttribute(ctx, attributeId)
	keeper.SetGridAttribute(ctx, attributeId, current+uint64(50))
	require.Equal(t, uint64(150), keeper.GetGridAttribute(ctx, attributeId))

	// Decrement manually
	current = keeper.GetGridAttribute(ctx, attributeId)
	keeper.SetGridAttribute(ctx, attributeId, current-uint64(30))
	require.Equal(t, uint64(120), keeper.GetGridAttribute(ctx, attributeId))
}

func TestGridAttributes(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	objectId := "test-object"

	// Test initial state (all attributes should be 0)
	powerAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_power, objectId)
	capacityAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, objectId)
	loadAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_load, objectId)

	require.Equal(t, uint64(0), keeper.GetGridAttribute(ctx, powerAttrId))
	require.Equal(t, uint64(0), keeper.GetGridAttribute(ctx, capacityAttrId))
	require.Equal(t, uint64(0), keeper.GetGridAttribute(ctx, loadAttrId))

	// Set and verify attributes
	keeper.SetGridAttribute(ctx, powerAttrId, 100)
	keeper.SetGridAttribute(ctx, capacityAttrId, 200)
	keeper.SetGridAttribute(ctx, loadAttrId, 50)

	require.Equal(t, uint64(100), keeper.GetGridAttribute(ctx, powerAttrId))
	require.Equal(t, uint64(200), keeper.GetGridAttribute(ctx, capacityAttrId))
	require.Equal(t, uint64(50), keeper.GetGridAttribute(ctx, loadAttrId))
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

	// Compute and set connection capacity directly: (capacity - load) / connectionCount
	connectionCapacity := (uint64(1000) - uint64(400)) / uint64(2)
	keeper.SetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, objectId), connectionCapacity)

	// Verify connection capacity
	storedCapacity := keeper.GetGridAttribute(ctx, keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, objectId))
	require.Equal(t, uint64(300), storedCapacity)
}

// TestIsValidGridAttributeID locks down the validation rule used by both the
// SetGridAttribute backstop and the upgrade-handler pruning sweep. The "2-"
// case is the testnet orphan from incident-2026-05-grid-orphan.md.
func TestIsValidGridAttributeID(t *testing.T) {
	cases := []struct {
		id    string
		valid bool
	}{
		// Canonical three-segment ids built by GetGridAttributeID.
		{"0-1-15", true},
		{"2-4-100", true},
		{"14-2-7", true},

		// GetGridAttributeIDByObjectId(type, objectId) shapes from tests
		// that pass single-token objectIds. Both segments non-empty so the
		// row is well-formed even if it is not three-segment.
		{"2-source1", true},
		{"5-test-object", true},

		// The historical orphan and friends.
		{"2-", false},
		{"0-", false},
		{"", false},
		{"-2", false},
		{"-", false},
		{"2", false},
	}

	for _, tc := range cases {
		require.Equal(t, tc.valid, keeperlib.IsValidGridAttributeID(tc.id), "id=%q", tc.id)
	}
}

// TestSetGridAttributeRejectsMalformedID locks the keeper-level guard that
// makes the "2-" leak unreachable going forward. Any future regression that
// calls GetGridAttributeIDByObjectId(t, "") drops on the floor instead of
// landing as an orphan KV row.
func TestSetGridAttributeRejectsMalformedID(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Reproduce the historical pre-daac34c bug shape: capacity attribute
	// for an empty destination id.
	badID := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, "")
	require.Equal(t, "2-", badID, "regression guard: bad id shape changed")

	keeper.SetGridAttribute(ctx, badID, uint64(840000))

	// The write should be a no-op; nothing leaked into the store.
	require.Equal(t, uint64(0), keeper.GetGridAttribute(ctx, badID))
	for _, rec := range keeper.GetAllGridExport(ctx) {
		require.NotEqual(t, badID, rec.AttributeId, "malformed id leaked into store")
	}
}

// TestPruneMalformedGridAttributes verifies the upgrade-handler sweep
// that recovers historical orphan rows (the "2-" testnet leak): it deletes
// every key that fails IsValidGridAttributeID while leaving canonical rows
// untouched, and reports the deleted keys for upgrade-time logging.
func TestPruneMalformedGridAttributes(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// A canonical row (must survive).
	goodID := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, "1-7")
	keeper.SetGridAttribute(ctx, goodID, uint64(123))

	// Bypass the SetGridAttribute backstop to plant an orphan row that
	// looks exactly like the one that exists on testnet today, simulating
	// pre-upgrade state.
	keepertest.WriteRawGridAttribute(t, keeper, ctx, "2-", uint64(840000))

	pruned := keeper.PruneMalformedGridAttributes(ctx)
	require.Equal(t, []string{"2-"}, pruned)

	// Bad row gone, good row preserved.
	require.Equal(t, uint64(0), keeper.GetGridAttribute(ctx, "2-"))
	require.Equal(t, uint64(123), keeper.GetGridAttribute(ctx, goodID))

	// Idempotent: a second pass finds nothing to prune.
	require.Empty(t, keeper.PruneMalformedGridAttributes(ctx))
}
