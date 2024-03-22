package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
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
    permissionValue := uint64(k.GetPermissionsByBytes(ctx, addressPermissionId))

	var permission types.QueryAddressResponse
    permission.Address  = req.Address
    permission.PlayerId = GetObjectID(types.ObjectType_player, k.GetPlayerIndexFromAddress(ctx, permission.Address))
    permission.Permissions = permissionValue

	return &permission, nil
}


func (k Keeper) AddressAll(goCtx context.Context, req *types.QueryAllAddressRequest) (*types.QueryAllAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    var permissions []*types.QueryAddressResponse

	ctx := sdk.UnwrapSDKContext(goCtx)

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	addressPlayerStore := prefix.NewStore(store, types.KeyPrefix(types.AddressPlayerKey))

	pageRes, err := query.Paginate(addressPlayerStore, req.Pagination, func(key []byte, value []byte) error {
		var permission types.QueryAddressResponse

        permission.Address = string(key)
        permission.PlayerId = GetObjectID(types.ObjectType_player, k.GetPlayerIndexFromAddress(ctx, permission.Address))

        addressPermissionId := GetAddressPermissionIDBytes(permission.Address)
        permission.Permissions = uint64(k.GetPermissionsByBytes(ctx, addressPermissionId))

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

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	addressPlayerStore := prefix.NewStore(store, types.KeyPrefix(types.AddressPlayerKey))

	pageRes, err := query.Paginate(addressPlayerStore, req.Pagination, func(key []byte, value []byte) error {
		var permission types.QueryAddressResponse


        if (GetObjectID(types.ObjectType_player, binary.BigEndian.Uint64(value)) == req.PlayerId) {
            permission.Address = string(key)
            permission.PlayerId = GetObjectID(types.ObjectType_player, binary.BigEndian.Uint64(value))

            addressPermissionId := GetAddressPermissionIDBytes(permission.Address)
            permission.Permissions = uint64(k.GetPermissionsByBytes(ctx, addressPermissionId))

            permissions = append(permissions, &permission)
        }

        return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllAddressResponse{Address: permissions, Pagination: pageRes}, nil
}


func (k Keeper) AddressAssociationAll(goCtx context.Context, req *types.QueryAllAddressAssociationRequest) (*types.QueryAllAddressAssociationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    var addresses []*types.AddressAssociation

	ctx := sdk.UnwrapSDKContext(goCtx)

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	addressPlayerStore := prefix.NewStore(store, types.KeyPrefix(types.AddressPlayerKey))

	pageRes, err := query.Paginate(addressPlayerStore, req.Pagination, func(key []byte, value []byte) error {
		var address types.AddressAssociation

        address.Address = string(key)
        address.PlayerIndex = binary.BigEndian.Uint64(value)
        address.RegistrationStatus = types.RegistrationStatus_approved

        addresses = append(addresses, &address)

        return nil
	})

	addressAssociationStore := prefix.NewStore(store, types.KeyPrefix(types.AddressRegistrationKey))
	pageRes, err = query.Paginate(addressAssociationStore, req.Pagination, func(key []byte, value []byte) error {
		var address types.AddressAssociation

        address.Address = string(key)
        address.PlayerIndex = binary.BigEndian.Uint64(value)
        address.RegistrationStatus = types.RegistrationStatus_proposed

        addresses = append(addresses, &address)

        return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllAddressAssociationResponse{AddressAssociation: addresses, Pagination: pageRes}, nil
}