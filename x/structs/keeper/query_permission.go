package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"structs/x/structs/types"

    "encoding/binary"
    "strings"
    //"strconv"
)


func (k Keeper) Permission(goCtx context.Context, req *types.QueryGetPermissionRequest) (*types.QueryGetPermissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

    var permissions *types.PermissionRecord

    permissionValue  := k.GetPermissionsByBytes(ctx, []byte(req.PermissionId))
    permissions.PermissionId    = req.PermissionId
    permissions.Value           = uint64(permissionValue)

	return &types.QueryGetPermissionResponse{PermissionRecord: permissions}, nil
}


func (k Keeper) PermissionByObject(goCtx context.Context, req *types.QueryAllPermissionByObjectRequest) (*types.QueryAllPermissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    var permissions []*types.PermissionRecord

	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	permissionStore := prefix.NewStore(store, types.KeyPrefix(types.PermissionKey))

	pageRes, err := query.Paginate(permissionStore, req.Pagination, func(key []byte, value []byte) error {

        extractedId := strings.Split(string(key), "@")
        if (extractedId[0] == req.ObjectId) {
            permissions = append(permissions, &types.PermissionRecord{PermissionId: string(key), Value: binary.BigEndian.Uint64(value)})
        }

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

	store := ctx.KVStore(k.storeKey)
	permissionStore := prefix.NewStore(store, types.KeyPrefix(types.PermissionKey))

	pageRes, err := query.Paginate(permissionStore, req.Pagination, func(key []byte, value []byte) error {

        extractedId := strings.Split(string(key), "@")
        if (extractedId[1] == req.PlayerId) {
            permissions = append(permissions, &types.PermissionRecord{PermissionId: string(key), Value: binary.BigEndian.Uint64(value)})
        }

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

 	store := ctx.KVStore(k.storeKey)
 	permissionStore := prefix.NewStore(store, types.KeyPrefix(types.PermissionKey))

 	pageRes, err := query.Paginate(permissionStore, req.Pagination, func(key []byte, value []byte) error {

        permissions = append(permissions, &types.PermissionRecord{PermissionId: string(key), Value: binary.BigEndian.Uint64(value)})

         return nil
 	})

	return &types.QueryAllPermissionResponse{PermissionRecords: permissions, Pagination: pageRes}, err
}