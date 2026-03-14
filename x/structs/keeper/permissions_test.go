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
	testPermissionAdd(keeper, ctx, permissionId, flag1)
	newPermission := keeper.GetPermissionsByBytes(ctx, permissionId)
	require.Equal(t, flag1, newPermission)

	// Add second flag
	testPermissionAdd(keeper, ctx, permissionId, flag2)
	newPermission = keeper.GetPermissionsByBytes(ctx, permissionId)
	require.Equal(t, types.Permission(0b0011), newPermission)

	// Test removing permissions
	testPermissionRemove(keeper, ctx, permissionId, flag1)
	newPermission = keeper.GetPermissionsByBytes(ctx, permissionId)
	require.Equal(t, flag2, newPermission)

	// Test permission checks
	require.True(t, testPermissionHasAll(keeper, ctx, permissionId, flag2))
	require.False(t, testPermissionHasAll(keeper, ctx, permissionId, flag1))
	require.True(t, testPermissionHasOneOf(keeper, ctx, permissionId, types.Permission(0b0011)))
	require.False(t, testPermissionHasOneOf(keeper, ctx, permissionId, types.Permission(0b0100)))
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
	require.True(t, testPermissionHasAll(keeper, ctx, permissionId, testPermission))
	require.False(t, testPermissionHasAll(keeper, ctx, permissionId, types.Permission(0b0101)))
}

func TestGuildRankRegisterReadWrite(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	objectId := "1-1"
	guildId := "2-2"

	// Empty register returns all zeros
	reg := keeper.ReadGuildRankRegister(ctx, objectId, guildId)
	for i := 0; i < types.PermissionBitCount; i++ {
		require.Equal(t, uint64(0), reg[i])
	}

	// Write a register with some slots set
	reg[0] = 5 // PermPlay -> rank 5
	reg[1] = 3 // PermAdmin -> rank 3
	keeper.WriteGuildRankRegister(ctx, objectId, guildId, reg)

	// Read it back
	reg2 := keeper.ReadGuildRankRegister(ctx, objectId, guildId)
	require.Equal(t, uint64(5), reg2[0])
	require.Equal(t, uint64(3), reg2[1])
	for i := 2; i < types.PermissionBitCount; i++ {
		require.Equal(t, uint64(0), reg2[i])
	}

	// Writing all-zero register deletes the key
	var zeroReg [types.PermissionBitCount]uint64
	keeper.WriteGuildRankRegister(ctx, objectId, guildId, zeroReg)
	reg3 := keeper.ReadGuildRankRegister(ctx, objectId, guildId)
	for i := 0; i < types.PermissionBitCount; i++ {
		require.Equal(t, uint64(0), reg3[i])
	}
}

func TestSetGuildRankPermissionStoreOnly(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	objectId := "1-1"
	guildId := "2-2"

	// Set individual permission bits via StoreOnly (genesis path)
	keeper.SetGuildRankPermissionStoreOnly(ctx, objectId, guildId, types.PermPlay, 5)
	keeper.SetGuildRankPermissionStoreOnly(ctx, objectId, guildId, types.PermAdmin, 3)

	reg := keeper.ReadGuildRankRegister(ctx, objectId, guildId)
	require.Equal(t, uint64(5), reg[0]) // bit 0 = PermPlay
	require.Equal(t, uint64(3), reg[1]) // bit 1 = PermAdmin

	// Set combined mask -- both bits get the same rank
	keeper.SetGuildRankPermissionStoreOnly(ctx, objectId, guildId, types.PermUpdate|types.PermDelete, 7)
	reg = keeper.ReadGuildRankRegister(ctx, objectId, guildId)
	require.Equal(t, uint64(5), reg[0]) // PermPlay unchanged
	require.Equal(t, uint64(3), reg[1]) // PermAdmin unchanged
	require.Equal(t, uint64(7), reg[2]) // PermUpdate
	require.Equal(t, uint64(7), reg[3]) // PermDelete
}

func TestClearPermissionGuildRankByObject(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)

	objectId := "1-10"
	guild1 := "2-1"
	guild2 := "2-2"

	keeper.SetGuildRankPermissionStoreOnly(ctx, objectId, guild1, types.PermPlay, 3)
	keeper.SetGuildRankPermissionStoreOnly(ctx, objectId, guild2, types.PermAdmin, 1)

	reg1 := keeper.ReadGuildRankRegister(ctx, objectId, guild1)
	require.Equal(t, uint64(3), reg1[0])

	keeper.ClearPermissionGuildRankByObject(ctx, objectId)

	// After clear, registers are gone
	reg1 = keeper.ReadGuildRankRegister(ctx, objectId, guild1)
	require.Equal(t, uint64(0), reg1[0])
	reg2 := keeper.ReadGuildRankRegister(ctx, objectId, guild2)
	require.Equal(t, uint64(0), reg2[1])

	// Clear on nonexistent object is a no-op
	keeper.ClearPermissionGuildRankByObject(ctx, "nonexistent")
}
