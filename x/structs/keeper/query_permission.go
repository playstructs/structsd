package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"structs/x/structs/types"

    "encoding/binary"
    "strings"
    //"strconv"
)


func (k Keeper) Permission(goCtx context.Context, req *types.QueryGetGuildPermissionRequest) (*types.QueryGetMultiplePermissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}



    var permissions []*types.QueryPermissionResponse

	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	guildPermissionStore := prefix.NewStore(store, types.KeyPrefix(types.PermissionKey))

	pageRes, err := query.Paginate(guildPermissionStore, req.Pagination, func(key []byte, value []byte) error {
		var permission types.QueryPermissionResponse

	    keys := strings.Split(string(key), "-")

        if (keys[0] == req.GuildId) {
            permission.ObjectId = keys[0]
            permission.PlayerId = keys[1]
            permission.PermissionRecord = binary.BigEndian.Uint64(value)

        	permissions = append(permissions, &permission)
        }
        return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetMultiplePermissionResponse{Permission: permissions, Pagination: pageRes}, nil
}


func (k Keeper) PermissionByObject(goCtx context.Context, req *types.QueryAllGuildPermissionRequest) (*types.QueryGetMultiplePermissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    var permissions []*types.QueryPermissionResponse

	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	guildPermissionStore := prefix.NewStore(store, types.KeyPrefix(types.PermissionKey))

	pageRes, err := query.Paginate(guildPermissionStore, req.Pagination, func(key []byte, value []byte) error {
		var permission types.PermissionRecord

	    keys := strings.Split(string(key), "-")

        permission.ObjectId = keys[0]
        permission.PlayerId = keys[1]
        permission.PermissionRecord = binary.BigEndian.Uint64(value)

        permissions = append(permissions, &permission)

        return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetMultiplePermissionResponse{Permission: permissions, Pagination: pageRes}, nil
}



func (k Keeper) PermissionByPlayer(goCtx context.Context, req *types.QueryAllGuildPermissionRequest) (*types.QueryGetMultiplePermissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    var permissions []*types.QueryPermissionResponse

	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	guildPermissionStore := prefix.NewStore(store, types.KeyPrefix(types.PermissionKey))

	pageRes, err := query.Paginate(guildPermissionStore, req.Pagination, func(key []byte, value []byte) error {
		var permission types.PermissionRecord

	    keys := strings.Split(string(key), "-")

        permission.ObjectId = keys[0]
        permission.PlayerId = keys[1]
        permission.PermissionRecord = binary.BigEndian.Uint64(value)

        permissions = append(permissions, &permission)

        return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetMultiplePermissionResponse{Permission: permissions, Pagination: pageRes}, nil
}

func (k Keeper) PermissionAll(goCtx context.Context, req *types.QueryAllGuildPermissionRequest) (*types.QueryGetMultiplePermissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    var permissions []*types.QueryPermissionResponse

	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	guildPermissionStore := prefix.NewStore(store, types.KeyPrefix(types.PermissionKey))

	pageRes, err := query.Paginate(guildPermissionStore, req.Pagination, func(key []byte, value []byte) error {
		var permission types.PermissionRecord

	    keys := strings.Split(string(key), "-")

        permission.ObjectId = keys[0]
        permission.PlayerId = keys[1]
        permission.PermissionRecord = binary.BigEndian.Uint64(value)

        permissions = append(permissions, &permission)

        return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetMultiplePermissionResponse{Permission: permissions, Pagination: pageRes}, nil
}