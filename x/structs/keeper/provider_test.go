package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "structs/testutil/keeper"
	kpr "structs/x/structs/keeper"
	"structs/x/structs/types"
)

func createNProvider(keeper kpr.Keeper, ctx sdk.Context, n int) []types.Provider {
	items := make([]types.Provider, n)
	for i := range items {
		provider := types.Provider{
			Owner:                       "player" + string(rune(i)),
			Creator:                     "address" + string(rune(i)),
			SubstationId:                "substation" + string(rune(i)),
			Rate:                        sdk.NewCoin("token", math.NewInt(100)),
			AccessPolicy:                types.ProviderAccessPolicy_openMarket,
			CapacityMinimum:             100,
			CapacityMaximum:             1000,
			DurationMinimum:             1,
			DurationMaximum:             10,
			ProviderCancellationPenalty: math.LegacyNewDec(1),
			ConsumerCancellationPenalty: math.LegacyNewDec(1),
		}
		provider = testAppendProvider(keeper, ctx, provider)
		items[i] = provider
	}
	return items
}

func TestProviderCRUD(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	providers := createNProvider(keeper, ctx, 5)

	// Test GetProvider
	for _, provider := range providers {
		got, found := keeper.GetProvider(ctx, provider.Id)
		require.True(t, found)
		require.Equal(t, provider, got)
	}

	// Test GetAllProvider
	allProviders := keeper.GetAllProvider(ctx)
	require.Len(t, allProviders, 5)

	// Test RemoveProvider
	for _, provider := range providers {
		keeper.RemoveProvider(ctx, provider.Id)
		_, found := keeper.GetProvider(ctx, provider.Id)
		require.False(t, found)
	}
}

func TestProviderCount(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	initialCount := keeper.GetProviderCount(ctx)
	require.Equal(t, types.KeeperStartValue, initialCount)

	// Create a provider and check count
	provider := types.Provider{
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
	_ = testAppendProvider(keeper, ctx, provider)
	newCount := keeper.GetProviderCount(ctx)
	require.Equal(t, initialCount+1, newCount)
}

func TestProviderGuildAccess(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	provider := types.Provider{
		Owner:                       "player1",
		Creator:                     "address1",
		SubstationId:                "substation1",
		Rate:                        sdk.NewCoin("token", math.NewInt(100)),
		AccessPolicy:                types.ProviderAccessPolicy_guildMarket,
		CapacityMinimum:             100,
		CapacityMaximum:             1000,
		DurationMinimum:             1,
		DurationMaximum:             10,
		ProviderCancellationPenalty: math.LegacyNewDec(1),
		ConsumerCancellationPenalty: math.LegacyNewDec(1),
	}
	provider = testAppendProvider(keeper, ctx, provider)

	// Test granting guild access
	guildId := "guild1"
	keeper.ProviderGrantGuild(ctx, provider.Id, guildId)
	allowed := keeper.ProviderGuildAccessAllowed(ctx, provider.Id, guildId)
	require.True(t, allowed)

	// Test revoking guild access
	keeper.ProviderRevokeGuild(ctx, provider.Id, guildId)
	allowed = keeper.ProviderGuildAccessAllowed(ctx, provider.Id, guildId)
	require.False(t, allowed)
}

func TestProviderCache(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	provider := types.Provider{
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
	provider = testAppendProvider(keeper, ctx, provider)

	// Test loading provider directly
	loadedProvider, found := keeper.GetProvider(ctx, provider.Id)
	require.True(t, found)
	require.Equal(t, provider.Id, loadedProvider.Id)
	require.Equal(t, provider, loadedProvider)

	// Test grid attribute for provider (using load as general example)
	loadAttrId := kpr.GetGridAttributeIDByObjectId(types.GridAttributeType_load, provider.Id)
	initialLoad := keeper.GetGridAttribute(ctx, loadAttrId)
	require.Equal(t, uint64(0), initialLoad)

	keeper.SetGridAttribute(ctx, loadAttrId, 50)
	require.Equal(t, uint64(50), keeper.GetGridAttribute(ctx, loadAttrId))
}

func TestProviderAgreementVerification(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	provider := types.Provider{
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
	provider = testAppendProvider(keeper, ctx, provider)

	// Verify provider was stored with correct bounds
	loadedProvider, found := keeper.GetProvider(ctx, provider.Id)
	require.True(t, found)
	require.Equal(t, uint64(100), loadedProvider.CapacityMinimum)
	require.Equal(t, uint64(1000), loadedProvider.CapacityMaximum)
	require.Equal(t, uint64(1), loadedProvider.DurationMinimum)
	require.Equal(t, uint64(10), loadedProvider.DurationMaximum)
}

func TestProviderAccessPolicy(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	provider := types.Provider{
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
	provider = testAppendProvider(keeper, ctx, provider)

	// Test initial access policy
	loadedProvider, found := keeper.GetProvider(ctx, provider.Id)
	require.True(t, found)
	require.Equal(t, types.ProviderAccessPolicy_openMarket, loadedProvider.AccessPolicy)

	// Test changing access policy
	loadedProvider.AccessPolicy = types.ProviderAccessPolicy_guildMarket
	keeper.ImportProvider(ctx, loadedProvider)
	updatedProvider, _ := keeper.GetProvider(ctx, provider.Id)
	require.Equal(t, types.ProviderAccessPolicy_guildMarket, updatedProvider.AccessPolicy)
}
