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

func (k Keeper) SubstationAll(goCtx context.Context, req *types.QueryAllSubstationRequest) (*types.QueryAllSubstationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var substations []types.Substation
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	substationStore := prefix.NewStore(store, types.KeyPrefix(types.SubstationKey))

	pageRes, err := query.Paginate(substationStore, req.Pagination, func(key []byte, value []byte) error {
		var substation types.Substation
		if err := k.cdc.Unmarshal(value, &substation); err != nil {
			return err
		}

        substation.Load = k.SubstationGetLoad(ctx, substation.Id)
        substation.Energy = k.SubstationGetEnergy(ctx, substation.Id)
        substation.ConnectedPlayerCount = k.SubstationGetConnectedPlayerCount(ctx, substation.Id)

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
	substation, found := k.GetSubstation(ctx, req.Id, true)
	if !found {
		return nil, sdkerrors.ErrKeyNotFound
	}

	return &types.QueryGetSubstationResponse{Substation: substation}, nil
}
