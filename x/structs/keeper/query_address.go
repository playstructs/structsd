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
)

func (k Keeper) AddressAll(goCtx context.Context, req *types.QueryAllAddressRequest) (*types.QueryAllAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var addresss []types.Address
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	addressStore := prefix.NewStore(store, types.KeyPrefix(types.AddressKey))

	pageRes, err := query.Paginate(addressStore, req.Pagination, func(key []byte, value []byte) error {
		var address types.Address
		if err := k.cdc.Unmarshal(value, &address); err != nil {
			return err
		}

		addresss = append(addresss, address)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllAddressResponse{Address: addresss, Pagination: pageRes}, nil
}

func (k Keeper) Address(goCtx context.Context, req *types.QueryGetAddressRequest) (*types.QueryGetAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	address, found := k.GetAddress(ctx, req.Id)
	if !found {
		return nil, sdkerrors.ErrKeyNotFound
	}

	return &types.QueryGetAddressResponse{Address: address}, nil
}
