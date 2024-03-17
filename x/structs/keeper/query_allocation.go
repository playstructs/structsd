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
)

func (k Keeper) AllocationAll(goCtx context.Context, req *types.QueryAllAllocationRequest) (*types.QueryAllAllocationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var allocations []types.Allocation
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	allocationStore := prefix.NewStore(store, types.KeyPrefix(types.AllocationKey))

	pageRes, err := query.Paginate(allocationStore, req.Pagination, func(key []byte, value []byte) error {
		var allocation types.Allocation
		if err := k.cdc.Unmarshal(value, &allocation); err != nil {
			return err
		}

		allocations = append(allocations, allocation)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllAllocationResponse{Allocation: allocations, Pagination: pageRes}, nil
}

func (k Keeper) Allocation(goCtx context.Context, req *types.QueryGetAllocationRequest) (*types.QueryGetAllocationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	allocation, found := k.GetAllocation(ctx, req.Id, true)
	if !found {
		return nil, types.ErrObjectNotFound
	}

	return &types.QueryGetAllocationResponse{Allocation: allocation}, nil
}
