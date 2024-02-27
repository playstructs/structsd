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

    permissionRecord := uint64(k.AddressGetPlayerPermissions(ctx, req.Address))

	var permission types.QueryAddressResponse
    permission.Address  = req.Address
    permission.PlayerId = k.GetPlayerIndexFromAddress(ctx, permission.Address)
    permission.PermissionRecord = permissionRecord

	return &permission, nil
}

func (k Keeper) AddressAll(goCtx context.Context, req *types.QueryAllAddressRequest) (*types.QueryAllAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    var permissions []*types.QueryAddressResponse

	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	addressPermissionStore := prefix.NewStore(store, types.KeyPrefix(types.AddressPermissionKey))

	pageRes, err := query.Paginate(addressPermissionStore, req.Pagination, func(key []byte, value []byte) error {
		var permission types.QueryAddressResponse

        permission.Address = string(key)
        permission.PlayerId = k.GetPlayerIndexFromAddress(ctx, permission.Address)
        permission.PermissionRecord = binary.BigEndian.Uint64(value)

        permissions = append(permissions, &permission)

        return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllAddressResponse{Address: permissions, Pagination: pageRes}, nil
}



func (k Keeper) AddressAllByPlayer(goCtx context.Context, req *types.QueryAllAddressByPlayerRequest) (*types.QueryAllAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    var permissions []*types.QueryAddressResponse

	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	addressPermissionStore := prefix.NewStore(store, types.KeyPrefix(types.AddressPermissionKey))

	pageRes, err := query.Paginate(addressPermissionStore, req.Pagination, func(key []byte, value []byte) error {
		var permission types.QueryAddressResponse

        if (k.GetPlayerIndexFromAddress(ctx, string(key)) == req.PlayerId) {
            permission.Address = string(key)
            permission.PlayerId = k.GetPlayerIndexFromAddress(ctx, permission.Address)
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
