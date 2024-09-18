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

func (k Keeper) PlanetAll(goCtx context.Context, req *types.QueryAllPlanetRequest) (*types.QueryAllPlanetResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var planets []types.Planet
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	planetStore := prefix.NewStore(store, types.KeyPrefix(types.PlanetKey))

	pageRes, err := query.Paginate(planetStore, req.Pagination, func(key []byte, value []byte) error {
		var planet types.Planet
		if err := k.cdc.Unmarshal(value, &planet); err != nil {
			return err
		}

		planets = append(planets, planet)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllPlanetResponse{Planet: planets, Pagination: pageRes}, nil
}

func (k Keeper) Planet(goCtx context.Context, req *types.QueryGetPlanetRequest) (*types.QueryGetPlanetResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	planet, found := k.GetPlanet(ctx, req.Id)
	if !found {
		return nil, types.ErrObjectNotFound
	}

    planetAttributes := k.GetPlanetAttributesByObject(ctx, req.Id)

	return &types.QueryGetPlanetResponse{Planet: planet, PlanetAttributes: &planetAttributes}, nil
}


func (k Keeper) PlanetAllByPlayer(goCtx context.Context, req *types.QueryAllPlanetByPlayerRequest) (*types.QueryAllPlanetResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var planets []types.Planet
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	planetStore := prefix.NewStore(store, types.KeyPrefix(types.PlanetKey))

	pageRes, err := query.Paginate(planetStore, req.Pagination, func(key []byte, value []byte) error {
		var planet types.Planet
		if err := k.cdc.Unmarshal(value, &planet); err != nil {
			return err
		}

        if (req.PlayerId == planet.Owner) {
            planets = append(planets, planet)
        }

		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllPlanetResponse{Planet: planets, Pagination: pageRes}, nil
}


func (k Keeper) PlanetAttribute(goCtx context.Context, req *types.QueryGetPlanetAttributeRequest) (*types.QueryGetPlanetAttributeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	planetAttribute := k.GetPlanetAttribute(ctx, GetPlanetAttributeIDByObjectId(types.PlanetAttributeType_enum[req.AttributeType], req.PlanetId))

	return &types.QueryGetPlanetAttributeResponse{Attribute: planetAttribute}, nil
}
