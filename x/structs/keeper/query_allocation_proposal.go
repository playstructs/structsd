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

func (k Keeper) AllocationProposalAll(goCtx context.Context, req *types.QueryAllAllocationProposalRequest) (*types.QueryAllAllocationProposalResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var allocationProposals []types.AllocationProposal
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	allocationProposalStore := prefix.NewStore(store, types.KeyPrefix(types.AllocationProposalKey))

	pageRes, err := query.Paginate(allocationProposalStore, req.Pagination, func(key []byte, value []byte) error {
		var allocationProposal types.AllocationProposal
		if err := k.cdc.Unmarshal(value, &allocationProposal); err != nil {
			return err
		}

		allocationProposals = append(allocationProposals, allocationProposal)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllAllocationProposalResponse{AllocationProposal: allocationProposals, Pagination: pageRes}, nil
}

func (k Keeper) AllocationProposal(goCtx context.Context, req *types.QueryGetAllocationProposalRequest) (*types.QueryGetAllocationProposalResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	allocationProposal, found := k.GetAllocationProposal(ctx, req.Id)
	if !found {
		return nil, sdkerrors.ErrKeyNotFound
	}

	return &types.QueryGetAllocationProposalResponse{AllocationProposal: allocationProposal}, nil
}
