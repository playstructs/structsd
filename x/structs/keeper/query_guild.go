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

func (k Keeper) GuildAll(goCtx context.Context, req *types.QueryAllGuildRequest) (*types.QueryAllGuildResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var guilds []types.Guild
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
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
		return nil, types.ErrObjectNotFound
	}

	return &types.QueryGetGuildResponse{Guild: guild}, nil
}



func (k Keeper) GuildAssociationAll(goCtx context.Context, req *types.QueryAllGuildAssociationRequest) (*types.QueryAllGuildAssociationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    var gridAssociation []*types.GridAssociation

	ctx := sdk.UnwrapSDKContext(goCtx)

	gridAssociationStore := prefix.NewStore(store, types.KeyPrefix(types.GuildRegistrationKey))
	pageRes, err = query.Paginate(gridAssociationStore, req.Pagination, func(key []byte, value []byte) error {
		var address types.AddressAssociation

        gridAssociation.GuildId = string(key)
        gridAssociation.Player = binary.BigEndian.Uint64(value)
        gridAssociation.RegistrationStatus = types.RegistrationStatus_proposed

        addresses = append(addresses, &address)

        return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllGuildAssociationResponse{GridAssociation: addresses, Pagination: pageRes}, nil
}