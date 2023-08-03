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

func (k Keeper) GuildAll(goCtx context.Context, req *types.QueryAllGuildRequest) (*types.QueryAllGuildResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var guilds []types.Guild
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	guildStore := prefix.NewStore(store, types.KeyPrefix(types.GuildKey))

	pageRes, err := query.Paginate(guildStore, req.Pagination, func(key []byte, value []byte) error {
		var guild types.Guild
		if err := k.cdc.Unmarshal(value, &guild); err != nil {
			return err
		}

		guilds = append(guilds, guild)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllGuildResponse{Guild: guilds, Pagination: pageRes}, nil
}

func (k Keeper) Guild(goCtx context.Context, req *types.QueryGetGuildRequest) (*types.QueryGetGuildResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	guild, found := k.GetGuild(ctx, req.Id)
	if !found {
		return nil, sdkerrors.ErrKeyNotFound
	}

	return &types.QueryGetGuildResponse{Guild: guild}, nil
}
