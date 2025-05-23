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
	items := createNStruct(keeper, ctx, 10)
	count := uint64(len(items))
	require.Equal(t, count, keeper.GetStructCount(ctx))
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
	newHealth, err := keeper.SetStructAttributeDelta(ctx, healthAttrId, 100, 50)
	require.NoError(t, err)
	require.Equal(t, uint64(150), newHealth)

	// Test SetStructAttributeDecrement
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
	keeper.AppendStructDestructionQueue(ctx, struct1.Id)
	keeper.AppendStructDestructionQueue(ctx, struct2.Id)

	// Test StructSweepDestroyed
	keeper.StructSweepDestroyed(ctx)

	// Verify structs are removed
	_, found := keeper.GetStruct(ctx, struct1.Id)
	require.False(t, found)
	_, found = keeper.GetStruct(ctx, struct2.Id)
	require.False(t, found)
}
