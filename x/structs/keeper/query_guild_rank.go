package keeper

import (
	"context"
	"encoding/binary"
	"strings"

	"github.com/cosmos/cosmos-sdk/runtime"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"structs/x/structs/types"
)

const maxQueryIDLength = 256

func (k Keeper) GuildRankPermissionByObject(goCtx context.Context, req *types.QueryGuildRankPermissionByObjectRequest) (*types.QueryGuildRankPermissionByObjectResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.ObjectId == "" {
		return nil, status.Error(codes.InvalidArgument, "object_id is required")
	}
	if len(req.ObjectId) > maxQueryIDLength {
		return nil, status.Error(codes.InvalidArgument, "object_id exceeds max length")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	// Store keys: objectId + "/" + guildId + "/" + 8-byte permission; value: 8-byte rank
	objectPrefix := append(types.KeyPrefix(types.PermissionGuildRank), []byte(req.ObjectId+"/")...)
	prefixStore := prefix.NewStore(store, objectPrefix)

	var records []*types.GuildRankPermissionRecord
	pageRes, err := query.Paginate(prefixStore, req.Pagination, func(key []byte, value []byte) error {
		if len(key) < 9 { // at least "x/"+ 8 bytes
			return nil
		}
		guildId := strings.TrimSuffix(string(key[:len(key)-9]), "/")
		permVal := binary.BigEndian.Uint64(key[len(key)-8:])
		rank := binary.BigEndian.Uint64(value)
		records = append(records, &types.GuildRankPermissionRecord{
			ObjectId:    req.ObjectId,
			GuildId:     guildId,
			Permissions: permVal,
			Rank:        rank,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryGuildRankPermissionByObjectResponse{
		GuildRankPermissionRecords: records,
		Pagination:                 pageRes,
	}, nil
}

func (k Keeper) GuildRankPermissionByObjectAndGuild(goCtx context.Context, req *types.QueryGuildRankPermissionByObjectAndGuildRequest) (*types.QueryGuildRankPermissionByObjectAndGuildResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.ObjectId == "" {
		return nil, status.Error(codes.InvalidArgument, "object_id is required")
	}
	if req.GuildId == "" {
		return nil, status.Error(codes.InvalidArgument, "guild_id is required")
	}
	if len(req.ObjectId) > maxQueryIDLength || len(req.GuildId) > maxQueryIDLength {
		return nil, status.Error(codes.InvalidArgument, "object_id or guild_id exceeds max length")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	guildRankStore := prefix.NewStore(store, GuildRankKeyPrefix(req.ObjectId, req.GuildId))
	// Keys in this store: 8-byte permission; value: 8-byte rank

	var records []*types.GuildRankPermissionRecord
	pageRes, err := query.Paginate(guildRankStore, req.Pagination, func(key []byte, value []byte) error {
		if len(key) != 8 {
			return nil
		}
		permVal := binary.BigEndian.Uint64(key)
		rank := binary.BigEndian.Uint64(value)
		records = append(records, &types.GuildRankPermissionRecord{
			ObjectId:    req.ObjectId,
			GuildId:     req.GuildId,
			Permissions: permVal,
			Rank:        rank,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryGuildRankPermissionByObjectAndGuildResponse{
		GuildRankPermissionRecords: records,
		Pagination:                 pageRes,
	}, nil
}
