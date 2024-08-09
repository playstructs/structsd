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

func (k Keeper) FleetAll(goCtx context.Context, req *types.QueryAllFleetRequest) (*types.QueryAllFleetResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var fleets []types.Fleet
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	fleetStore := prefix.NewStore(store, types.KeyPrefix(types.FleetKey))

	pageRes, err := query.Paginate(fleetStore, req.Pagination, func(key []byte, value []byte) error {
		var fleet types.Fleet
		if err := k.cdc.Unmarshal(value, &fleet); err != nil {
			return err
		}

		fleets = append(fleets, fleet)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllFleetResponse{Fleet: fleets, Pagination: pageRes}, nil
}

func (k Keeper) Fleet(goCtx context.Context, req *types.QueryGetFleetRequest) (*types.QueryGetFleetResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	fleet, found := k.GetFleet(ctx, req.Id)
	if !found {
		return nil, types.ErrObjectNotFound
	}

	return &types.QueryGetFleetResponse{Fleet: fleet}, nil
}


func (k Keeper) FleetByIndex(goCtx context.Context, req *types.QueryGetFleetByIndexRequest) (*types.QueryGetFleetResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	fleet, found := k.GetFleet(ctx, GetObjectID(types.ObjectType_fleet, req.Index))
	if !found {
		return nil, types.ErrObjectNotFound
	}

	return &types.QueryGetFleetResponse{Fleet: fleet}, nil
}

