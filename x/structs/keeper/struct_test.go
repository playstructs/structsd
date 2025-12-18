package keeper_test

import (
	"testing"

	keepertest "structs/testutil/keeper"
	"structs/testutil/nullify"
	"structs/x/structs/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func createNStruct(keeper keeper.Keeper, ctx sdk.Context, n int) []types.Struct {
	items := make([]types.Struct, n)
	for i := range items {
		items[i] = types.Struct{
			Creator: "cosmos1creator" + string(rune(i)),
			Owner:   "cosmos1owner" + string(rune(i)),
			Type:    uint64(i % 3), // Different types for variety
		}
		items[i] = keeper.AppendStruct(ctx, items[i])
	}
	return items
}

func TestStructGet(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNStruct(keeper, ctx, 10)
	for _, item := range items {
		got, found := keeper.GetStruct(ctx, item.Id)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&got),
		)
	}
}

func TestStructRemove(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNStruct(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveStruct(ctx, item.Id)
		_, found := keeper.GetStruct(ctx, item.Id)
		require.False(t, found)
	}
}

func TestStructGetAll(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNStruct(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllStruct(ctx)),
	)
}

func TestStructCount(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	// Get initial count to account for any previous test state
	initialCount := keeper.GetStructCount(ctx)
	items := createNStruct(keeper, ctx, 10)
	expectedCount := initialCount + uint64(len(items))
	actualCount := keeper.GetStructCount(ctx)
	require.Equal(t, expectedCount, actualCount)
}

func TestStructAttributes(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Create a test struct
	struct1 := types.Struct{
		Creator: "cosmos1creator",
		Owner:   "cosmos1owner",
		Type:    1,
	}
	struct1 = keeper.AppendStruct(ctx, struct1)

	// Test setting and getting attributes
	healthAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_health, struct1.Id)
	statusAttrId := keeperlib.GetStructAttributeIDByObjectId(types.StructAttributeType_status, struct1.Id)

	// Test SetStructAttribute
	keeper.SetStructAttribute(ctx, healthAttrId, 100)
	require.Equal(t, uint64(100), keeper.GetStructAttribute(ctx, healthAttrId))

	// Test SetStructAttributeDelta
	// Delta calculation: if oldAmount < currentAmount, resetAmount = currentAmount - oldAmount
	// Otherwise resetAmount = 0. Then amount = resetAmount + newAmount
	// Here: current=100, old=100, new=50 -> resetAmount=0, amount=0+50=50
	newHealth, err := keeper.SetStructAttributeDelta(ctx, healthAttrId, 100, 50)
	require.NoError(t, err)
	require.Equal(t, uint64(50), newHealth)

	// Test delta with oldAmount < currentAmount
	keeper.SetStructAttribute(ctx, healthAttrId, 200)
	newHealth, err = keeper.SetStructAttributeDelta(ctx, healthAttrId, 150, 100)
	require.NoError(t, err)
	// current=200, old=150, new=100 -> resetAmount=200-150=50, amount=50+100=150
	require.Equal(t, uint64(150), newHealth)

	// Test SetStructAttributeDecrement
	// Reset to a known value first
	keeper.SetStructAttribute(ctx, healthAttrId, 150)
	newHealth, err = keeper.SetStructAttributeDecrement(ctx, healthAttrId, 30)
	require.NoError(t, err)
	require.Equal(t, uint64(120), newHealth)

	// Test SetStructAttributeIncrement
	newHealth = keeper.SetStructAttributeIncrement(ctx, healthAttrId, 80)
	require.Equal(t, uint64(200), newHealth)

	// Test ClearStructAttribute
	keeper.ClearStructAttribute(ctx, healthAttrId)
	require.Equal(t, uint64(0), keeper.GetStructAttribute(ctx, healthAttrId))

	// Test flag operations
	materializedFlag := uint64(types.StructStateMaterialized)
	keeper.SetStructAttributeFlagAdd(ctx, statusAttrId, materializedFlag)
	require.True(t, keeper.StructAttributeFlagHasAll(ctx, statusAttrId, materializedFlag))
	require.True(t, keeper.StructAttributeFlagHasOneOf(ctx, statusAttrId, materializedFlag))

	keeper.SetStructAttributeFlagRemove(ctx, statusAttrId, materializedFlag)
	require.False(t, keeper.StructAttributeFlagHasAll(ctx, statusAttrId, materializedFlag))
	require.False(t, keeper.StructAttributeFlagHasOneOf(ctx, statusAttrId, materializedFlag))
}

func TestStructDefender(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Create test structs
	protectedStruct := types.Struct{
		Creator: "cosmos1creator1",
		Owner:   "cosmos1owner1",
		Type:    1,
	}
	protectedStruct = keeper.AppendStruct(ctx, protectedStruct)

	defenderStruct := types.Struct{
		Creator: "cosmos1creator2",
		Owner:   "cosmos1owner2",
		Type:    2,
	}
	defenderStruct = keeper.AppendStruct(ctx, defenderStruct)

	// Test SetStructDefender
	defender := keeper.SetStructDefender(ctx, protectedStruct.Id, protectedStruct.Index, defenderStruct.Id)
	require.Equal(t, protectedStruct.Id, defender.ProtectedStructId)
	require.Equal(t, defenderStruct.Id, defender.DefendingStructId)

	// Test GetStructDefender
	got, found := keeper.GetStructDefender(ctx, protectedStruct.Id, defenderStruct.Id)
	require.True(t, found)
	require.Equal(t, defender, got)

	// Test GetAllStructDefender
	defenders := keeper.GetAllStructDefender(ctx, protectedStruct.Id)
	require.Len(t, defenders, 1)
	require.Equal(t, defenderStruct.Id, defenders[0])

	// Test ClearStructDefender
	keeper.ClearStructDefender(ctx, protectedStruct.Id, defenderStruct.Id)
	_, found = keeper.GetStructDefender(ctx, protectedStruct.Id, defenderStruct.Id)
	require.False(t, found)

	// Test DestroyStructDefender
	defender = keeper.SetStructDefender(ctx, protectedStruct.Id, protectedStruct.Index, defenderStruct.Id)
	keeper.DestroyStructDefender(ctx, defenderStruct.Id)
	_, found = keeper.GetStructDefender(ctx, protectedStruct.Id, defenderStruct.Id)
	require.False(t, found)
}

func TestStructDestructionQueue(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Create test structs
	struct1 := types.Struct{
		Creator: "cosmos1creator1",
		Owner:   "cosmos1owner1",
		Type:    1,
	}
	struct1 = keeper.AppendStruct(ctx, struct1)

	struct2 := types.Struct{
		Creator: "cosmos1creator2",
		Owner:   "cosmos1owner2",
		Type:    2,
	}
	struct2 = keeper.AppendStruct(ctx, struct2)

	// Add structs to destruction queue
	// Note: AppendStructDestructionQueue adds to blockHeight + StructSweepDelay
	// StructSweepDestroyed reads from current blockHeight
	// So we need to advance the block height or the structs won't be in the current block's queue
	keeper.AppendStructDestructionQueue(ctx, struct1.Id)
	keeper.AppendStructDestructionQueue(ctx, struct2.Id)

	// Test StructSweepDestroyed - this will only process structs queued for the current block
	// Since we just queued them for a future block, they won't be processed yet
	keeper.StructSweepDestroyed(ctx)

	// Verify structs are still present (not yet processed due to delay)
	_, found := keeper.GetStruct(ctx, struct1.Id)
	require.True(t, found, "Struct should still exist as it's queued for a future block")
	_, found = keeper.GetStruct(ctx, struct2.Id)
	require.True(t, found, "Struct should still exist as it's queued for a future block")

	// Note: To actually test destruction, we would need to advance the block height
	// by StructSweepDelay blocks, which requires more complex test setup
}
