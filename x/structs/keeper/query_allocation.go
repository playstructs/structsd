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

func (k Keeper) AllocationAll(goCtx context.Context, req *types.QueryAllAllocationRequest) (*types.QueryAllAllocationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var allocations []types.Allocation
	var statuses []uint64
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	allocationStore := prefix.NewStore(store, types.KeyPrefix(types.AllocationKey))

	pageRes, err := query.Paginate(allocationStore, req.Pagination, func(key []byte, value []byte) error {
		var allocation types.Allocation
		if err := k.cdc.Unmarshal(value, &allocation); err != nil {
			return err
		}

        statuses = append(statuses, k.GetAllocationStatus(ctx, allocation.Id))
		allocations = append(allocations, allocation)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllAllocationResponse{Allocation: allocations, Status: statuses, Pagination: pageRes}, nil
}

func (k Keeper) Allocation(goCtx context.Context, req *types.QueryGetAllocationRequest) (*types.QueryGetAllocationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	allocation, found := k.GetAllocation(ctx, req.Id)
	if !found {
		return nil, sdkerrors.ErrKeyNotFound
	}

	return &types.QueryGetAllocationResponse{Allocation: allocation}, nil
}
