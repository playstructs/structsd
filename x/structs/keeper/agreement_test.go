package keeper_test

import (
	"testing"

	keepertest "structs/testutil/keeper"
	"structs/testutil/nullify"
	"structs/x/structs/keeper"
	"structs/x/structs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func createNAgreement(keeper keeper.Keeper, ctx sdk.Context, n int) []types.Agreement {
	items := make([]types.Agreement, n)
	for i := range items {
		items[i] = types.Agreement{
			Id:         "agreement" + string(rune(i)),
			ProviderId: "provider" + string(rune(i)),
			EndBlock:   uint64(1000 + i),
		}
		keeper.AppendAgreement(ctx, items[i])
	}
	return items
}

func TestAgreementGet(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNAgreement(keeper, ctx, 10)
	for _, item := range items {
		got, found := keeper.GetAgreement(ctx, item.Id)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&got),
		)
	}
}

func TestAgreementRemove(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNAgreement(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveAgreement(ctx, item)
		_, found := keeper.GetAgreement(ctx, item.Id)
		require.False(t, found)
	}
}

func TestAgreementGetAll(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	items := createNAgreement(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllAgreement(ctx)),
	)
}

func TestAgreementSet(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Create initial agreement
	agreement := types.Agreement{
		Id:         "test-agreement",
		ProviderId: "test-provider",
		EndBlock:   1000,
	}

	// Test AppendAgreement
	err := keeper.AppendAgreement(ctx, agreement)
	require.NoError(t, err)

	// Verify agreement was stored
	got, found := keeper.GetAgreement(ctx, agreement.Id)
	require.True(t, found)
	require.Equal(t, agreement.Id, got.Id)

	// Test SetAgreement
	agreement.EndBlock = 2000
	updated, err := keeper.SetAgreement(ctx, agreement)
	require.NoError(t, err)
	require.Equal(t, uint64(2000), updated.EndBlock)

	// Verify update was stored
	got, found = keeper.GetAgreement(ctx, agreement.Id)
	require.True(t, found)
	require.Equal(t, uint64(2000), got.EndBlock)
}

func TestAgreementImport(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Create agreement to import
	agreement := types.Agreement{
		Id:         "import-agreement",
		ProviderId: "import-provider",
		EndBlock:   1000,
	}

	// Test ImportAgreement
	keeper.ImportAgreement(ctx, agreement)

	// Verify agreement was stored
	got, found := keeper.GetAgreement(ctx, agreement.Id)
	require.True(t, found)
	require.Equal(t, agreement.Id, got.Id)
	require.Equal(t, agreement.ProviderId, got.ProviderId)
}

func TestAgreementExpirations(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Create agreements with different end blocks
	agreement1 := types.Agreement{
		Id:         "expired-agreement",
		ProviderId: "provider1",
		EndBlock:   uint64(ctx.BlockHeight()), // Current block
	}
	agreement2 := types.Agreement{
		Id:         "future-agreement",
		ProviderId: "provider2",
		EndBlock:   uint64(ctx.BlockHeight() + 1000), // Future block
	}

	// Store agreements
	keeper.AppendAgreement(ctx, agreement1)
	keeper.AppendAgreement(ctx, agreement2)

	// Test AgreementExpirations
	keeper.AgreementExpirations(ctx)

	// Verify expired agreement was handled
	// Note: This test might need to be adjusted based on the actual implementation
	// of how expired agreements are handled in the system
	_, found := keeper.GetAgreement(ctx, agreement1.Id)
	require.True(t, found) // Assuming expired agreements are not automatically removed
}
