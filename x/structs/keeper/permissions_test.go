package keeper_test

import (
	"testing"

	keepertest "structs/testutil/keeper"
	keeperlib "structs/x/structs/keeper"
	"structs/x/structs/types"

	"github.com/stretchr/testify/require"
)

func TestPermissionBasicOperations(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Test data
	objectId := "test-object"
	playerId := "test-player"
	permissionId := keeperlib.GetObjectPermissionIDBytes(objectId, playerId)

	// Test initial state (should be Permissionless)
	initialPermission := keeper.GetPermissionsByBytes(ctx, permissionId)
	require.Equal(t, types.Permissionless, initialPermission)

	// Test setting permissions
	testPermission := types.Permission(0b1010) // Example permission flags
	setPermission := keeper.SetPermissionsByBytes(ctx, permissionId, testPermission)
	require.Equal(t, testPermission, setPermission)

	// Verify the permission was set
	retrievedPermission := keeper.GetPermissionsByBytes(ctx, permissionId)
	require.Equal(t, testPermission, retrievedPermission)

	// Test clearing permissions
	keeper.PermissionClearAll(ctx, permissionId)
	clearedPermission := keeper.GetPermissionsByBytes(ctx, permissionId)
	require.Equal(t, types.Permissionless, clearedPermission)
}

func TestPermissionBitOperations(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Test data
	objectId := "test-object"
	playerId := "test-player"
	permissionId := keeperlib.GetObjectPermissionIDBytes(objectId, playerId)

	// Test adding permissions
	flag1 := types.Permission(0b0001)
	flag2 := types.Permission(0b0010)

	// Add first flag
	newPermission := keeper.PermissionAdd(ctx, permissionId, flag1)
	require.Equal(t, flag1, newPermission)

	// Add second flag
	newPermission = keeper.PermissionAdd(ctx, permissionId, flag2)
	require.Equal(t, types.Permission(0b0011), newPermission)

	// Test removing permissions
	newPermission = keeper.PermissionRemove(ctx, permissionId, flag1)
	require.Equal(t, flag2, newPermission)

	// Test permission checks
	require.True(t, keeper.PermissionHasAll(ctx, permissionId, flag2))
	require.False(t, keeper.PermissionHasAll(ctx, permissionId, flag1))
	require.True(t, keeper.PermissionHasOneOf(ctx, permissionId, types.Permission(0b0011)))
	require.False(t, keeper.PermissionHasOneOf(ctx, permissionId, types.Permission(0b0100)))
}

func TestGetPermissionsByObject(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Test data
	objectId := "test-object"
	player1 := "player1"
	player2 := "player2"

	// Set permissions for multiple players on the same object
	permission1 := keeperlib.GetObjectPermissionIDBytes(objectId, player1)
	permission2 := keeperlib.GetObjectPermissionIDBytes(objectId, player2)

	keeper.SetPermissionsByBytes(ctx, permission1, types.Permission(0b0001))
	keeper.SetPermissionsByBytes(ctx, permission2, types.Permission(0b0010))

	// Get all permissions for the object
	permissions := keeper.GetPermissionsByObject(ctx, objectId)
	require.Len(t, permissions, 2)

	// Verify permissions
	foundPlayer1 := false
	foundPlayer2 := false
	for _, p := range permissions {
		if p.PermissionId == string(permission1) {
			require.Equal(t, uint64(0b0001), p.Value)
			foundPlayer1 = true
		}
		if p.PermissionId == string(permission2) {
			require.Equal(t, uint64(0b0010), p.Value)
			foundPlayer2 = true
		}
	}
	require.True(t, foundPlayer1)
	require.True(t, foundPlayer2)
}

func TestGetPermissionsByPlayer(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Test data
	playerId := "test-player"
	object1 := "object1"
	object2 := "object2"

	// Set permissions for the player on multiple objects
	permission1 := keeperlib.GetObjectPermissionIDBytes(object1, playerId)
	permission2 := keeperlib.GetObjectPermissionIDBytes(object2, playerId)

	keeper.SetPermissionsByBytes(ctx, permission1, types.Permission(0b0001))
	keeper.SetPermissionsByBytes(ctx, permission2, types.Permission(0b0010))

	// Get all permissions for the player
	permissions := keeper.GetPermissionsByPlayer(ctx, playerId)
	require.Len(t, permissions, 2)

	// Verify permissions
	foundObject1 := false
	foundObject2 := false
	for _, p := range permissions {
		if p.PermissionId == string(permission1) {
			require.Equal(t, uint64(0b0001), p.Value)
			foundObject1 = true
		}
		if p.PermissionId == string(permission2) {
			require.Equal(t, uint64(0b0010), p.Value)
			foundObject2 = true
		}
	}
	require.True(t, foundObject1)
	require.True(t, foundObject2)
}

func TestGetAllPermissionExport(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Test data
	object1 := "object1"
	object2 := "object2"
	player1 := "player1"
	player2 := "player2"

	// Set various permissions
	permissions := []struct {
		objectId   string
		playerId   string
		permission types.Permission
	}{
		{object1, player1, types.Permission(0b0001)},
		{object1, player2, types.Permission(0b0010)},
		{object2, player1, types.Permission(0b0100)},
		{object2, player2, types.Permission(0b1000)},
	}

	for _, p := range permissions {
		permissionId := keeperlib.GetObjectPermissionIDBytes(p.objectId, p.playerId)
		keeper.SetPermissionsByBytes(ctx, permissionId, p.permission)
	}

	// Get all permissions
	allPermissions := keeper.GetAllPermissionExport(ctx)
	require.Len(t, allPermissions, len(permissions))

	// Verify all permissions are present
	for _, p := range permissions {
		permissionId := keeperlib.GetObjectPermissionIDBytes(p.objectId, p.playerId)
		found := false
		for _, exported := range allPermissions {
			if exported.PermissionId == string(permissionId) {
				require.Equal(t, uint64(p.permission), exported.Value)
				found = true
				break
			}
		}
		require.True(t, found)
	}
}

func TestAddressPermissionIDBytes(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	// Test data
	address := "test-address"
	permissionId := keeperlib.GetAddressPermissionIDBytes(address)

	// Test setting and getting address permissions
	testPermission := types.Permission(0b1010)
	keeper.SetPermissionsByBytes(ctx, permissionId, testPermission)

	// Verify the permission was set correctly
	retrievedPermission := keeper.GetPermissionsByBytes(ctx, permissionId)
	require.Equal(t, testPermission, retrievedPermission)

	// Test permission operations
	require.True(t, keeper.PermissionHasAll(ctx, permissionId, testPermission))
	require.False(t, keeper.PermissionHasAll(ctx, permissionId, types.Permission(0b0101)))
}
