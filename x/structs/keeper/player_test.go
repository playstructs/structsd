package keeper_test

import (
	keepertest "structs/testutil/keeper"
	"structs/x/structs/types"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlayerCRUD(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)

	player := types.Player{
		Creator:        "cosmos1creatoraddress",
		PrimaryAddress: "cosmos1creatoraddress",
		SubstationId:   "substation1",
	}

	// AppendPlayer
	created := k.AppendPlayer(ctx, player)
	require.NotEmpty(t, created.Id)
	require.Equal(t, player.Creator, created.Creator)

	// GetPlayer
	got, found := k.GetPlayer(ctx, created.Id)
	require.True(t, found)
	require.Equal(t, created.Id, got.Id)

	// SetPlayer
	got.SubstationId = "substation2"
	k.SetPlayer(ctx, got)
	updated, found := k.GetPlayer(ctx, created.Id)
	require.True(t, found)
	require.Equal(t, "substation2", updated.SubstationId)

	// GetAllPlayer
	players := k.GetAllPlayer(ctx)
	foundAny := false
	for _, p := range players {
		if p.Id == created.Id {
			foundAny = true
		}
	}
	require.True(t, foundAny)

	// GetPlayerCount
	count := k.GetPlayerCount(ctx)
	require.GreaterOrEqual(t, count, uint64(1))

	// RemovePlayer
	k.RemovePlayer(ctx, created.Id)
	_, found = k.GetPlayer(ctx, created.Id)
	require.False(t, found)
}

func TestPlayerCacheBasic(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	player := types.Player{
		Creator:        "cosmos1cacheaddress",
		PrimaryAddress: "cosmos1cacheaddress",
	}
	created := k.AppendPlayer(ctx, player)

	cache, err := k.GetPlayerCacheFromId(ctx, created.Id)
	require.NoError(t, err)
	require.Equal(t, created.Id, cache.GetPlayerId())

	// Test GetPlayer
	p, err := cache.GetPlayer()
	require.NoError(t, err)
	require.Equal(t, created.Id, p.Id)

	// Test SetActiveAddress
	cache.SetActiveAddress("cosmos1cacheaddress")
	require.Equal(t, "cosmos1cacheaddress", cache.GetActiveAddress())
}

func TestPlayerCachePermissionsAndReadiness(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	player := types.Player{
		Creator:        "cosmos1permaddress",
		PrimaryAddress: "cosmos1permaddress",
	}
	created := k.AppendPlayer(ctx, player)
	cache, err := k.GetPlayerCacheFromId(ctx, created.Id)
	require.NoError(t, err)

	err = cache.CanBePlayedBy(player.Creator)
	require.NoError(t, err)

	err = cache.CanBeUpdatedBy(player.Creator)
	require.NoError(t, err)

	err = cache.ReadinessCheck()
	require.NoError(t, err)
}

func TestPlayerInventoryAndCharge(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	player := types.Player{
		Creator:        "cosmos1invaddress",
		PrimaryAddress: "cosmos1invaddress",
	}
	created := k.AppendPlayer(ctx, player)

	inv := k.GetPlayerInventory(ctx, player.Creator)
	require.NotNil(t, inv)

	// Test charge/discharge
	cache, err := k.GetPlayerCacheFromId(ctx, created.Id)
	require.NoError(t, err)
	initialCharge := cache.GetCharge()
	cache.Discharge()
	newCharge := cache.GetCharge()
	require.LessOrEqual(t, newCharge, initialCharge)
}
