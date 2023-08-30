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

func (k Keeper) PlanetAll(goCtx context.Context, req *types.QueryAllPlanetRequest) (*types.QueryAllPlanetResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var planets []types.Planet
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	planetStore := prefix.NewStore(store, types.KeyPrefix(types.PlanetKey))

	pageRes, err := query.Paginate(planetStore, req.Pagination, func(key []byte, value []byte) error {
		var planet types.Planet
		if err := k.cdc.Unmarshal(value, &planet); err != nil {
			return err
		}

        planet.OreRemaining = planet.MaxOre - k.GetPlanetRefinementCount(ctx, planet.Id)
        planet.OreStored    = k.GetPlanetOreCount(ctx, planet.Id)

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
		return nil, sdkerrors.ErrKeyNotFound
	}

	return &types.QueryGetPlanetResponse{Planet: planet}, nil
}
