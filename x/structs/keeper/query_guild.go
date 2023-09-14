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

    "encoding/binary"
    "strings"
    "strconv"
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


func (k Keeper) GuildPermission(goCtx context.Context, req *types.QueryGetGuildPermissionRequest) (*types.QueryGetMultiplePermissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    guildId := strconv.FormatUint(req.GuildId, 10)


    var permissions []*types.QueryPermissionResponse

	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	guildPermissionStore := prefix.NewStore(store, types.KeyPrefix(types.GuildPermissionKey))

	pageRes, err := query.Paginate(guildPermissionStore, req.Pagination, func(key []byte, value []byte) error {
		var permission types.QueryPermissionResponse

	    keys := strings.Split(string(key), "-")

        if (keys[0] == guildId) {
            permission.ObjectId = keys[0]
            permission.PlayerId = keys[1]
            permission.PermissionRecord = binary.BigEndian.Uint64(value)

        	permissions = append(permissions, &permission)
        }
        return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetMultiplePermissionResponse{Permission: permissions, Pagination: pageRes}, nil
}

func (k Keeper) GuildPlayerPermission(goCtx context.Context, req *types.QueryGetGuildPlayerPermissionRequest) (*types.QueryPermissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    guildId := strconv.FormatUint(req.GuildId, 10)
    playerId := strconv.FormatUint(req.PlayerId, 10)

	ctx := sdk.UnwrapSDKContext(goCtx)

    recordId := GetGuildPermissionIDBytes(req.GuildId, req.PlayerId)
    permissionRecord := uint64(k.GuildGetPlayerPermissionsByBytes(ctx, recordId))

	var permission types.QueryPermissionResponse
    permission.ObjectId = guildId
    permission.PlayerId = playerId
    permission.PermissionRecord = permissionRecord

	return &permission, nil
}

func (k Keeper) GuildPermissionAll(goCtx context.Context, req *types.QueryAllGuildPermissionRequest) (*types.QueryGetMultiplePermissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    var permissions []*types.QueryPermissionResponse

	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	guildPermissionStore := prefix.NewStore(store, types.KeyPrefix(types.GuildPermissionKey))

	pageRes, err := query.Paginate(guildPermissionStore, req.Pagination, func(key []byte, value []byte) error {
		var permission types.QueryPermissionResponse

	    keys := strings.Split(string(key), "-")

        permission.ObjectId = keys[0]
        permission.PlayerId = keys[1]
        permission.PermissionRecord = binary.BigEndian.Uint64(value)

        permissions = append(permissions, &permission)

        return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetMultiplePermissionResponse{Permission: permissions, Pagination: pageRes}, nil
}
