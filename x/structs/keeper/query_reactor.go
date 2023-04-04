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

func (k Keeper) ReactorAll(goCtx context.Context, req *types.QueryAllReactorRequest) (*types.QueryAllReactorResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var reactors []types.Reactor
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	reactorStore := prefix.NewStore(store, types.KeyPrefix(types.ReactorKey))

	pageRes, err := query.Paginate(reactorStore, req.Pagination, func(key []byte, value []byte) error {
		var reactor types.Reactor
		if err := k.cdc.Unmarshal(value, &reactor); err != nil {
			return err
		}

		reactors = append(reactors, reactor)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllReactorResponse{Reactor: reactors, Pagination: pageRes}, nil
}

func (k Keeper) Reactor(goCtx context.Context, req *types.QueryGetReactorRequest) (*types.QueryGetReactorResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	reactor, found := k.GetReactor(ctx, req.Id)
	if !found {
		return nil, sdkerrors.ErrKeyNotFound
	}

	return &types.QueryGetReactorResponse{Reactor: reactor}, nil
}
