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

    //"encoding/binary"
    //"strings"
    //"strconv"
)

func (k Keeper) SubstationAll(goCtx context.Context, req *types.QueryAllSubstationRequest) (*types.QueryAllSubstationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var substations []types.Substation
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	substationStore := prefix.NewStore(store, types.KeyPrefix(types.SubstationKey))

	pageRes, err := query.Paginate(substationStore, req.Pagination, func(key []byte, value []byte) error {
		var substation types.Substation
		if err := k.cdc.Unmarshal(value, &substation); err != nil {
			return err
		}

		substations = append(substations, substation)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllSubstationResponse{Substation: substations, Pagination: pageRes}, nil
}

func (k Keeper) Substation(goCtx context.Context, req *types.QueryGetSubstationRequest) (*types.QueryGetSubstationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	substation, found := k.GetSubstation(ctx, req.Id)
	if !found {
		return nil, types.ErrObjectNotFound
	}

    gridAttributes := k.GetGridAttributesByObject(ctx, req.Id)

	return &types.QueryGetSubstationResponse{Substation: substation, GridAttributes: &gridAttributes}, nil
}

