package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"structs/x/structs/types"

    "encoding/binary"

)


func (k Keeper) Address(goCtx context.Context, req *types.QueryGetAddressRequest) (*types.QueryAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

    addressPermissionId := GetAddressPermissionIDBytes(req.Address)
    permissionRecord := uint64(k.GetPermissionsByBytes(ctx, addressPermissionId))

	var permission types.QueryAddressResponse
    permission.Address  = req.Address
    permission.PlayerId = GetObjectID(types.ObjectType_player, k.GetPlayerIndexFromAddress(ctx, permission.Address))
    permission.PermissionRecord = permissionRecord

	return &permission, nil
}


// TODO this function is broken
// It once relied on the address permission store, but in the unified permission store this is no longer effective
func (k Keeper) AddressAll(goCtx context.Context, req *types.QueryAllAddressRequest) (*types.QueryAllAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    var permissions []*types.QueryAddressResponse

	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	permissionStore := prefix.NewStore(store, types.KeyPrefix(types.PermissionKey))

	pageRes, err := query.Paginate(permissionStore, req.Pagination, func(key []byte, value []byte) error {
		var permission types.QueryAddressResponse

        permission.Address = string(key)
        permission.PlayerId = GetObjectID(types.ObjectType_player, k.GetPlayerIndexFromAddress(ctx, permission.Address))
        permission.PermissionRecord = binary.BigEndian.Uint64(value)

        permissions = append(permissions, &permission)

        return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllAddressResponse{Address: permissions, Pagination: pageRes}, nil
}

// TODO re-write this function
// This function is broken for similar reasons as the above function
func (k Keeper) AddressAllByPlayer(goCtx context.Context, req *types.QueryAllAddressByPlayerRequest) (*types.QueryAllAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    var permissions []*types.QueryAddressResponse

	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	permissionStore := prefix.NewStore(store, types.KeyPrefix(types.PermissionKey))

	pageRes, err := query.Paginate(permissionStore, req.Pagination, func(key []byte, value []byte) error {
		var permission types.QueryAddressResponse

        if (GetObjectID(types.ObjectType_player, k.GetPlayerIndexFromAddress(ctx, string(key))) == req.PlayerId) {
            permission.Address = string(key)
            permission.PlayerId = GetObjectID(types.ObjectType_player, k.GetPlayerIndexFromAddress(ctx, permission.Address))
            permission.PermissionRecord = binary.BigEndian.Uint64(value)

            permissions = append(permissions, &permission)
        }

        return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllAddressResponse{Address: permissions, Pagination: pageRes}, nil
}
