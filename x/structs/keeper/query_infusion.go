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

func (k Keeper) InfusionAll(goCtx context.Context, req *types.QueryAllInfusionRequest) (*types.QueryAllInfusionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var infusions []types.Infusion
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	infusionStore := prefix.NewStore(store, types.KeyPrefix(types.InfusionKey))

	pageRes, err := query.Paginate(infusionStore, req.Pagination, func(key []byte, value []byte) error {
		var infusion types.Infusion
		if err := k.cdc.Unmarshal(value, &infusion); err != nil {
			return err
		}

		infusions = append(infusions, infusion)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllInfusionResponse{Infusion: infusions, Pagination: pageRes}, nil
}

func (k Keeper) Infusion(goCtx context.Context, req *types.QueryGetInfusionRequest) (*types.QueryGetInfusionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	infusion, found := k.GetInfusion(ctx, types.ObjectType_enum[req.DestinationType], req.DestinationId, req.Address)
	if !found {
		return nil, sdkerrors.ErrKeyNotFound
	}

	return &types.QueryGetInfusionResponse{Infusion: infusion}, nil
}
