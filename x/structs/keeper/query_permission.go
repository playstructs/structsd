package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"structs/x/structs/types"

    "encoding/binary"
    //"strconv"
)


func (k Keeper) Permission(goCtx context.Context, req *types.QueryGetPermissionRequest) (*types.QueryGetPermissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

    var permissions types.PermissionRecord

    permissionValue  := k.GetPermissionsByBytes(ctx, []byte(req.PermissionId))
    permissions.PermissionId    = req.PermissionId
    permissions.Value           = uint64(permissionValue)

	return &types.QueryGetPermissionResponse{PermissionRecord: &permissions}, nil
}


func (k Keeper) PermissionByObject(goCtx context.Context, req *types.QueryAllPermissionByObjectRequest) (*types.QueryAllPermissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    var permissions []*types.PermissionRecord

	ctx := sdk.UnwrapSDKContext(goCtx)

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	permissionStore := prefix.NewStore(store, types.KeyPrefix(types.PermissionKey))
	objectPrefix := req.ObjectId + "@"
	objectStore := prefix.NewStore(permissionStore, []byte(objectPrefix))

	pagination, err := boundedPagination(req.Pagination)
	if err != nil {
		return nil, err
	}
	pageRes, err := query.Paginate(objectStore, pagination, func(key []byte, value []byte) error {

        permissions = append(permissions, &types.PermissionRecord{PermissionId: objectPrefix + string(key), Value: binary.BigEndian.Uint64(value)})

        return nil
	})

	return &types.QueryAllPermissionResponse{PermissionRecords: permissions, Pagination: pageRes}, err
}



func (k Keeper) PermissionByPlayer(goCtx context.Context, req *types.QueryAllPermissionByPlayerRequest) (*types.QueryAllPermissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    var permissions []*types.PermissionRecord

	ctx := sdk.UnwrapSDKContext(goCtx)

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	permissionStore := prefix.NewStore(store, types.KeyPrefix(types.PermissionKey))

	pagination, err := boundedPagination(req.Pagination)
	if err != nil {
		return nil, err
	}
	pageRes, err := query.Paginate(permissionPlayerStore(k, ctx, req.PlayerId), pagination, func(key []byte, value []byte) error {

        player, validKey := permissionPlayer(key)
        row := permissionStore.Get(key)
        if !validKey || player != req.PlayerId || len(row) != 8 {
            return status.Error(codes.Internal, "permission player index is inconsistent")
        }
        permissions = append(permissions, &types.PermissionRecord{PermissionId: string(key), Value: binary.BigEndian.Uint64(row)})

        return nil
	})

	return &types.QueryAllPermissionResponse{PermissionRecords: permissions, Pagination: pageRes}, err
}

func (k Keeper) PermissionAll(goCtx context.Context, req *types.QueryAllPermissionRequest) (*types.QueryAllPermissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

     var permissions []*types.PermissionRecord

 	ctx := sdk.UnwrapSDKContext(goCtx)

 	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
 	permissionStore := prefix.NewStore(store, types.KeyPrefix(types.PermissionKey))

	pagination, err := boundedPagination(req.Pagination)
	if err != nil {
		return nil, err
	}
	pageRes, err := query.Paginate(permissionStore, pagination, func(key []byte, value []byte) error {

        permissions = append(permissions, &types.PermissionRecord{PermissionId: string(key), Value: binary.BigEndian.Uint64(value)})

         return nil
 	})

	return &types.QueryAllPermissionResponse{PermissionRecords: permissions, Pagination: pageRes}, err
}