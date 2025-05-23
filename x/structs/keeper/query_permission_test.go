package keeper_test

import (
	"testing"

	keepertest "structs/testutil/keeper"
	"structs/x/structs/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPermissionQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	// Test data
	objectId := "test-object"
	playerId := "test-player"
	permissionId := objectId + "@" + playerId
	testPermission := types.Permission(0b1010)

	// Set up test permission
	keeper.SetPermissionsByBytes(ctx, []byte(permissionId), testPermission)

	for _, tc := range []struct {
		desc     string
		request  *types.QueryGetPermissionRequest
		response *types.QueryGetPermissionResponse
		err      error
	}{
		{
			desc: "ValidPermission",
			request: &types.QueryGetPermissionRequest{
				PermissionId: permissionId,
			},
			response: &types.QueryGetPermissionResponse{
				PermissionRecord: &types.PermissionRecord{
					PermissionId: permissionId,
					Value:        uint64(testPermission),
				},
			},
		},
		{
			desc: "NonExistentPermission",
			request: &types.QueryGetPermissionRequest{
				PermissionId: "non-existent@permission",
			},
			response: &types.QueryGetPermissionResponse{
				PermissionRecord: &types.PermissionRecord{
					PermissionId: "non-existent@permission",
					Value:        0, // Permissionless
				},
			},
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.Permission(wctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.response, response)
			}
		})
	}
}

func TestPermissionByObjectQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	// Test data
	objectId := "test-object"
	player1 := "player1"
	player2 := "player2"

	// Set up test permissions
	permission1 := objectId + "@" + player1
	permission2 := objectId + "@" + player2
	keeper.SetPermissionsByBytes(ctx, []byte(permission1), types.Permission(0b0001))
	keeper.SetPermissionsByBytes(ctx, []byte(permission2), types.Permission(0b0010))

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllPermissionByObjectRequest {
		return &types.QueryAllPermissionByObjectRequest{
			ObjectId: objectId,
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}

	t.Run("ByOffset", func(t *testing.T) {
		step := 1
		for i := 0; i < 2; i += step {
			resp, err := keeper.PermissionByObject(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.PermissionRecords), step)
		}
	})

	t.Run("ByKey", func(t *testing.T) {
		step := 1
		var next []byte
		for i := 0; i < 2; i += step {
			resp, err := keeper.PermissionByObject(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.PermissionRecords), step)
			next = resp.Pagination.NextKey
		}
	})

	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.PermissionByObject(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, 2, int(resp.Pagination.Total))
		require.Len(t, resp.PermissionRecords, 2)
	})

	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.PermissionByObject(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}

func TestPermissionByPlayerQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	// Test data
	playerId := "test-player"
	object1 := "object1"
	object2 := "object2"

	// Set up test permissions
	permission1 := object1 + "@" + playerId
	permission2 := object2 + "@" + playerId
	keeper.SetPermissionsByBytes(ctx, []byte(permission1), types.Permission(0b0001))
	keeper.SetPermissionsByBytes(ctx, []byte(permission2), types.Permission(0b0010))

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllPermissionByPlayerRequest {
		return &types.QueryAllPermissionByPlayerRequest{
			PlayerId: playerId,
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}

	t.Run("ByOffset", func(t *testing.T) {
		step := 1
		for i := 0; i < 2; i += step {
			resp, err := keeper.PermissionByPlayer(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.PermissionRecords), step)
		}
	})

	t.Run("ByKey", func(t *testing.T) {
		step := 1
		var next []byte
		for i := 0; i < 2; i += step {
			resp, err := keeper.PermissionByPlayer(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.PermissionRecords), step)
			next = resp.Pagination.NextKey
		}
	})

	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.PermissionByPlayer(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, 2, int(resp.Pagination.Total))
		require.Len(t, resp.PermissionRecords, 2)
	})

	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.PermissionByPlayer(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}

func TestPermissionAllQuery(t *testing.T) {
	keeper, ctx := keepertest.StructsKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)

	// Test data
	object1 := "object1"
	object2 := "object2"
	player1 := "player1"
	player2 := "player2"

	// Set up test permissions
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
		permissionId := p.objectId + "@" + p.playerId
		keeper.SetPermissionsByBytes(ctx, []byte(permissionId), p.permission)
	}

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllPermissionRequest {
		return &types.QueryAllPermissionRequest{
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}

	t.Run("ByOffset", func(t *testing.T) {
		step := 2
		for i := 0; i < len(permissions); i += step {
			resp, err := keeper.PermissionAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.PermissionRecords), step)
		}
	})

	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(permissions); i += step {
			resp, err := keeper.PermissionAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.PermissionRecords), step)
			next = resp.Pagination.NextKey
		}
	})

	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.PermissionAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(permissions), int(resp.Pagination.Total))
		require.Len(t, resp.PermissionRecords, len(permissions))
	})

	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.PermissionAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
