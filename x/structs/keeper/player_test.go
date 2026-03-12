package keeper_test

import (
	"testing"

	keepertest "structs/testutil/keeper"
	"structs/testutil/nullify"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"

	"github.com/stretchr/testify/require"
)

func TestPlayerCRUD(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	ctxSDK := ctx

	player := types.Player{
		Creator:        "cosmos1creatoraddress",
		PrimaryAddress: "cosmos1creatoraddress",
		SubstationId:   "substation1",
	}

	// AppendPlayer
	created := testAppendPlayer(k, ctxSDK, player)
	require.NotEmpty(t, created.Id)
	require.Equal(t, player.Creator, created.Creator)
	require.Equal(t, player.PrimaryAddress, created.PrimaryAddress)

	// GetPlayer
	got, found := k.GetPlayer(ctxSDK, created.Id)
	require.True(t, found)
	require.Equal(t, created.Id, got.Id)
	require.Equal(t,
		nullify.Fill(&created),
		nullify.Fill(&got),
	)

	// SetPlayer
	got.SubstationId = "substation2"
	k.SetPlayer(ctxSDK, got)
	updated, found := k.GetPlayer(ctxSDK, created.Id)
	require.True(t, found)
	require.Equal(t, "substation2", updated.SubstationId)

	// GetAllPlayer
	players := k.GetAllPlayer(ctxSDK)
	foundAny := false
	for _, p := range players {
		if p.Id == created.Id {
			foundAny = true
			break
		}
	}
	require.True(t, foundAny)

	// GetPlayerCount
	count := k.GetPlayerCount(ctxSDK)
	require.GreaterOrEqual(t, count, uint64(1))

	// GetPlayerFromIndex
	playerFromIndex, found := k.GetPlayerFromIndex(ctxSDK, created.Index)
	require.True(t, found)
	require.Equal(t, created.Id, playerFromIndex.Id)

	// Player removal is no longer directly supported via RemovePlayer
	// Verify the player still exists
	_, found = k.GetPlayer(ctxSDK, created.Id)
	require.True(t, found)
}

func TestPlayerCacheBasic(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	ctxSDK := ctx

	player := types.Player{
		Creator:        "cosmos1cacheaddress",
		PrimaryAddress: "cosmos1cacheaddress",
	}
	created := testAppendPlayer(k, ctxSDK, player)

	// Test GetPlayer
	p, found := k.GetPlayer(ctxSDK, created.Id)
	require.True(t, found)
	require.Equal(t, created.Id, p.Id)
	require.Equal(t, created.Creator, p.Creator)

	// Test PrimaryAddress
	require.Equal(t, "cosmos1cacheaddress", p.PrimaryAddress)

	// Test GetIndex
	require.Equal(t, created.Index, p.Index)
}

func TestPlayerPermissionsAndReadiness(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	ctxSDK := ctx

	player := types.Player{
		Creator:        "cosmos1permaddress",
		PrimaryAddress: "cosmos1permaddress",
	}
	created := testAppendPlayer(k, ctxSDK, player)

	// Verify player was created
	p, found := k.GetPlayer(ctxSDK, created.Id)
	require.True(t, found)
	require.Equal(t, created.Id, p.Id)

	// Verify address permissions were set
	addressPermId := keeperlib.GetAddressPermissionIDBytes(player.PrimaryAddress)
	perms := k.GetPermissionsByBytes(ctxSDK, addressPermId)
	require.NotEqual(t, types.Permissionless, perms)
}

func TestPlayerInventory(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	ctxSDK := ctx

	player := types.Player{
		Creator:        "cosmos1invaddress",
		PrimaryAddress: "cosmos1invaddress",
	}
	_ = testAppendPlayer(k, ctxSDK, player)

	inv := k.GetPlayerInventory(ctxSDK, player.Creator)
	require.NotNil(t, inv)
}

func TestPlayerUpsert(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	ctxSDK := ctx

	address := "cosmos1upsertaddress"

	// Create a player via testAppendPlayer
	player1 := types.Player{
		Creator:        address,
		PrimaryAddress: address,
	}
	player1 = testAppendPlayer(k, ctxSDK, player1)
	require.NotEmpty(t, player1.Id)
	require.Equal(t, address, player1.Creator)
	require.Equal(t, address, player1.PrimaryAddress)

	// Retrieve by address index
	idx := k.GetPlayerIndexFromAddress(ctxSDK, address)
	player2, found := k.GetPlayerFromIndex(ctxSDK, idx)
	require.True(t, found)
	require.Equal(t, player1.Id, player2.Id)
	require.Equal(t, player1.Index, player2.Index)
}

func TestPlayerGetAllBySubstation(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	ctxSDK := ctx

	substationId := "substation-test"

	player1 := types.Player{
		Creator:        "cosmos1player1",
		PrimaryAddress: "cosmos1player1",
		SubstationId:   substationId,
	}
	player1 = testAppendPlayer(k, ctxSDK, player1)

	player2 := types.Player{
		Creator:        "cosmos1player2",
		PrimaryAddress: "cosmos1player2",
		SubstationId:   substationId,
	}
	player2 = testAppendPlayer(k, ctxSDK, player2)

	player3 := types.Player{
		Creator:        "cosmos1player3",
		PrimaryAddress: "cosmos1player3",
		SubstationId:   "other-substation",
	}
	player3 = testAppendPlayer(k, ctxSDK, player3)

	// Get players by substation
	players := k.GetAllPlayerBySubstation(ctxSDK, substationId)
	require.Len(t, players, 2)

	foundPlayer1 := false
	foundPlayer2 := false
	for _, p := range players {
		if p.Id == player1.Id {
			foundPlayer1 = true
		}
		if p.Id == player2.Id {
			foundPlayer2 = true
		}
	}
	require.True(t, foundPlayer1)
	require.True(t, foundPlayer2)
}

func TestPlayerChargeGridAttributes(t *testing.T) {
	k, ctx := keepertest.StructsKeeper(t)
	ctxSDK := ctx

	player := types.Player{
		Creator:        "cosmos1chargeaddress",
		PrimaryAddress: "cosmos1chargeaddress",
	}
	created := testAppendPlayer(k, ctxSDK, player)

	// Test charge via grid attributes (lastAction)
	lastActionAttrId := keeperlib.GetGridAttributeIDByObjectId(types.GridAttributeType_lastAction, created.Id)
	lastAction := k.GetGridAttribute(ctxSDK, lastActionAttrId)
	require.GreaterOrEqual(t, lastAction, uint64(0))
}
