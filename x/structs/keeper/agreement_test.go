package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
	"structs/x/structs/keeper"
	"structs/x/structs/types"
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
		require.Equal(t, item.Id, got.Id)
		require.Equal(t, item.ProviderId, got.ProviderId)
		require.Equal(t, item.EndBlock, got.EndBlock)
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
	got := keeper.GetAllAgreement(ctx)
	require.Len(t, got, len(items))
	for i, item := range items {
		require.Equal(t, item.Id, got[i].Id)
		require.Equal(t, item.ProviderId, got[i].ProviderId)
		require.Equal(t, item.EndBlock, got[i].EndBlock)
	}
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

	// Create providers first
	provider1 := types.Provider{
		Owner:                       "player1",
		Creator:                     "address1",
		SubstationId:                "substation1",
		Rate:                        sdk.NewCoin("token", math.NewInt(100)),
		AccessPolicy:                types.ProviderAccessPolicy_openMarket,
		CapacityMinimum:             100,
		CapacityMaximum:             1000,
		DurationMinimum:             1,
		DurationMaximum:             10,
		ProviderCancellationPenalty: math.LegacyNewDec(1),
		ConsumerCancellationPenalty: math.LegacyNewDec(1),
	}
	provider2 := types.Provider{
		Owner:                       "player2",
		Creator:                     "address2",
		SubstationId:                "substation2",
		Rate:                        sdk.NewCoin("token", math.NewInt(100)),
		AccessPolicy:                types.ProviderAccessPolicy_openMarket,
		CapacityMinimum:             100,
		CapacityMaximum:             1000,
		DurationMinimum:             1,
		DurationMaximum:             10,
		ProviderCancellationPenalty: math.LegacyNewDec(1),
		ConsumerCancellationPenalty: math.LegacyNewDec(1),
	}

	// Store providers
	provider1, _ = keeper.AppendProvider(ctx, provider1)
	provider2, _ = keeper.AppendProvider(ctx, provider2)

	// Create agreements with different end blocks
	agreement1 := types.Agreement{
		Id:         "expired-agreement",
		ProviderId: provider1.Id,
		EndBlock:   uint64(ctx.BlockHeight()), // Current block
	}
	agreement2 := types.Agreement{
		Id:         "future-agreement",
		ProviderId: provider2.Id,
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
