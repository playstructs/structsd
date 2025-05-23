package keeper_test

import (
	"testing"

	keepertest "structs/testutil/keeper"
	kpr "structs/x/structs/keeper"
	"structs/x/structs/types"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func createNInfusion(keeper kpr.Keeper, ctx sdk.Context, n int, destinationId string) []types.Infusion {
	items := make([]types.Infusion, n)
	for i := range items {
		items[i] = types.CreateNewInfusion(
			types.ObjectType_struct,
			destinationId,
			"address"+string(rune(i)),
			"player"+string(rune(i)),
			uint64(100+i),
			math.LegacyNewDec(int64(i+1)),
			uint64(1),
		)
		keeper.SetInfusion(ctx, items[i])
	}
	return items
}

func TestInfusionCRUD(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	destinationId := "dest1"
	infusions := createNInfusion(keeper, ctx, 5, destinationId)

	// Test GetInfusion
	for _, infusion := range infusions {
		got, found := keeper.GetInfusion(ctx, destinationId, infusion.Address)
		require.True(t, found)
		require.Equal(t, infusion, got)
	}

	// Test GetInfusionByID
	for _, infusion := range infusions {
		id := destinationId + "-" + infusion.Address
		got, found := keeper.GetInfusionByID(ctx, id)
		require.True(t, found)
		require.Equal(t, infusion, got)
	}

	// Test RemoveInfusion
	for _, infusion := range infusions {
		keeper.RemoveInfusion(ctx, destinationId, infusion.Address)
		_, found := keeper.GetInfusion(ctx, destinationId, infusion.Address)
		require.False(t, found)
	}
}

func TestAppendAndSetInfusion(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	infusion := types.CreateNewInfusion(
		types.ObjectType_struct,
		"dest2",
		"addressX",
		"playerX",
		200,
		math.LegacyNewDec(2),
		1,
	)
	err := keeper.AppendInfusion(ctx, infusion)
	require.NoError(t, err)
	got, found := keeper.GetInfusion(ctx, "dest2", "addressX")
	require.True(t, found)
	require.Equal(t, infusion, got)

	// Test SetInfusion
	infusion.Fuel = 300
	keeper.SetInfusion(ctx, infusion)
	got, found = keeper.GetInfusion(ctx, "dest2", "addressX")
	require.True(t, found)
	require.Equal(t, uint64(300), got.Fuel)
}

func TestGetAllInfusion(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	_ = createNInfusion(keeper, ctx, 3, "dest3")
	all := keeper.GetAllInfusion(ctx)
	require.NotEmpty(t, all)
}

func TestGetAllReactorAndStructInfusions(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	dest := "reactor1"
	_ = createNInfusion(keeper, ctx, 2, dest)
	reactorInfusions := keeper.GetAllReactorInfusions(ctx, dest)
	structInfusions := keeper.GetAllStructInfusions(ctx, dest)
	require.NotEmpty(t, reactorInfusions)
	require.NotEmpty(t, structInfusions)
}

func TestUpsertInfusion(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	player := types.Player{Id: "playerY"}
	infusion, newFuel, oldFuel, newPower, oldPower, newCommission, oldCommission, newPlayerPower, oldPlayerPower, err := keeper.UpsertInfusion(
		ctx,
		types.ObjectType_struct,
		"dest4",
		"addressY",
		player,
		500,
		math.LegacyNewDec(3),
		2,
	)
	require.NoError(t, err)
	require.Equal(t, uint64(500), newFuel)
	require.Equal(t, uint64(0), oldFuel)
	require.NotNil(t, infusion)
	_ = newPower
	_ = oldPower
	_ = newCommission
	_ = oldCommission
	_ = newPlayerPower
	_ = oldPlayerPower
}

func TestDestroyInfusion(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	infusion := types.CreateNewInfusion(
		types.ObjectType_struct,
		"dest5",
		"addressZ",
		"playerZ",
		1000,
		math.LegacyNewDec(4),
		3,
	)
	keeper.SetInfusion(ctx, infusion)
	keeper.DestroyInfusion(ctx, infusion)
	_, found := keeper.GetInfusion(ctx, "dest5", "addressZ")
	require.False(t, found)
}

func TestDestroyAllInfusions(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	infusions := createNInfusion(keeper, ctx, 2, "dest6")
	keeper.DestroyAllInfusions(ctx, infusions)
	for _, infusion := range infusions {
		_, found := keeper.GetInfusion(ctx, "dest6", infusion.Address)
		require.False(t, found)
	}
}
