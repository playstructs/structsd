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
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func (k Keeper) ProviderAll(goCtx context.Context, req *types.QueryAllProviderRequest) (*types.QueryAllProviderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var providers []types.Provider
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	providerStore := prefix.NewStore(store, types.KeyPrefix(types.ProviderKey))

	pageRes, err := query.Paginate(providerStore, req.Pagination, func(key []byte, value []byte) error {
		var provider types.Provider
		if err := k.cdc.Unmarshal(value, &provider); err != nil {
			return err
		}

		providers = append(providers, provider)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllProviderResponse{Provider: providers, Pagination: pageRes}, nil
}

func (k Keeper) Provider(goCtx context.Context, req *types.QueryGetProviderRequest) (*types.QueryGetProviderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	provider, found := k.GetProvider(ctx, req.Id)
	if !found {
		return nil, types.ErrObjectNotFound
	}

    gridAttributes := k.GetGridAttributesByObject(ctx, req.Id)
	return &types.QueryGetProviderResponse{Provider: provider, GridAttributes: &gridAttributes}, nil
}



func (k Keeper) ProviderCollateralAddress(goCtx context.Context, req *types.QueryGetProviderCollateralAddressRequest) (*types.QueryAllProviderCollateralAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    var addresses []*types.InternalAddressAssociation
    address := authtypes.NewModuleAddress(types.ProviderCollateralPool + req.ProviderId).String()
    addressAssociation := types.InternalAddressAssociation{Address: address, ObjectId: req.ProviderId}
    addresses = append(addresses, &addressAssociation)

    return &types.QueryAllProviderCollateralAddressResponse{InternalAddressAssociation: addresses}, nil
}


func (k Keeper) ProviderCollateralAddressAll(goCtx context.Context, req *types.QueryAllProviderCollateralAddressRequest) (*types.QueryAllProviderCollateralAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    var addresses []*types.InternalAddressAssociation
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	providerStore := prefix.NewStore(store, types.KeyPrefix(types.ProviderKey))

	pageRes, err := query.Paginate(providerStore, req.Pagination, func(key []byte, value []byte) error {
		var provider types.Provider
		if err := k.cdc.Unmarshal(value, &provider); err != nil {
			return err
		}

        address := authtypes.NewModuleAddress(types.ProviderCollateralPool + provider.Id).String()
        addressAssociation := types.InternalAddressAssociation{Address: address, ObjectId: provider.Id}
        addresses = append(addresses, &addressAssociation)

		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllProviderCollateralAddressResponse{InternalAddressAssociation: addresses, Pagination: pageRes}, nil
}



func (k Keeper) ProviderEarningsAddress(goCtx context.Context, req *types.QueryGetProviderEarningsAddressRequest) (*types.QueryAllProviderEarningsAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    var addresses []*types.InternalAddressAssociation
    address := authtypes.NewModuleAddress(types.ProviderEarningsPool + req.ProviderId).String()
    addressAssociation := types.InternalAddressAssociation{Address: address, ObjectId: req.ProviderId}
    addresses = append(addresses, &addressAssociation)

    return &types.QueryAllProviderEarningsAddressResponse{InternalAddressAssociation: addresses}, nil
}


func (k Keeper) ProviderEarningsAddressAll(goCtx context.Context, req *types.QueryAllProviderEarningsAddressRequest) (*types.QueryAllProviderEarningsAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    var addresses []*types.InternalAddressAssociation
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	providerStore := prefix.NewStore(store, types.KeyPrefix(types.ProviderKey))

	pageRes, err := query.Paginate(providerStore, req.Pagination, func(key []byte, value []byte) error {
		var provider types.Provider
		if err := k.cdc.Unmarshal(value, &provider); err != nil {
			return err
		}

        address := authtypes.NewModuleAddress(types.ProviderEarningsPool + provider.Id).String()
        addressAssociation := types.InternalAddressAssociation{Address: address, ObjectId: provider.Id}
        addresses = append(addresses, &addressAssociation)

		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllProviderEarningsAddressResponse{InternalAddressAssociation: addresses, Pagination: pageRes}, nil
}

