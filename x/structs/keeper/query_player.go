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

func (k Keeper) PlayerAll(goCtx context.Context, req *types.QueryAllPlayerRequest) (*types.QueryAllPlayerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var players []types.Player
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	playerStore := prefix.NewStore(store, types.KeyPrefix(types.PlayerKey))

	pageRes, err := query.Paginate(playerStore, req.Pagination, func(key []byte, value []byte) error {
		var player types.Player
		if err := k.cdc.Unmarshal(value, &player); err != nil {
			return err
		}


        player.Load      = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, player.Id))
        player.Capacity  = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, player.Id))

        player.StructsLoad           = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, player.Id))
        player.CapacitySecondary    = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_connectionCapacity, player.SubstationId))

        playerAcc, _ := sdk.AccAddressFromBech32(player.PrimaryAddress)
        player.Storage = k.bankKeeper.SpendableCoin(ctx, playerAcc, "alpha")

		players = append(players, player)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllPlayerResponse{Player: players, Pagination: pageRes}, nil
}

func (k Keeper) Player(goCtx context.Context, req *types.QueryGetPlayerRequest) (*types.QueryGetPlayerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	player, found := k.GetPlayer(ctx, req.Id, true)
	if !found {
		return nil, sdkerrors.ErrKeyNotFound
	}

	return &types.QueryGetPlayerResponse{Player: player}, nil
}

/*

func (k Keeper) PlayerPermission(goCtx context.Context, req *types.QueryGetPlayerPermissionRequest) (*types.QueryGetMultiplePermissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    playerId := strconv.FormatUint(req.PlayerId, 10)


    var permissions []*types.QueryPermissionResponse

	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	playerPermissionStore := prefix.NewStore(store, types.KeyPrefix(types.PlayerPermissionKey))

	pageRes, err := query.Paginate(playerPermissionStore, req.Pagination, func(key []byte, value []byte) error {
		var permission types.QueryPermissionResponse

	    keys := strings.Split(string(key), "-")

        if (keys[0] == playerId) {
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

func (k Keeper) PlayerPlayerPermission(goCtx context.Context, req *types.QueryGetPlayerPlayerPermissionRequest) (*types.QueryPermissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    targetPlayerId := strconv.FormatUint(req.TargetPlayerId, 10)
    playerId := strconv.FormatUint(req.PlayerId, 10)

	ctx := sdk.UnwrapSDKContext(goCtx)

    recordId := GetPlayerPermissionIDBytes(req.TargetPlayerId, req.PlayerId)
    permissionRecord := uint64(k.PlayerGetPlayerPermissionsByBytes(ctx, recordId))

	var permission types.QueryPermissionResponse
    permission.ObjectId = targetPlayerId
    permission.PlayerId = playerId
    permission.PermissionRecord = permissionRecord

	return &permission, nil
}

func (k Keeper) PlayerPermissionAll(goCtx context.Context, req *types.QueryAllPlayerPermissionRequest) (*types.QueryGetMultiplePermissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    var permissions []*types.QueryPermissionResponse

	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	playerPermissionStore := prefix.NewStore(store, types.KeyPrefix(types.PlayerPermissionKey))

	pageRes, err := query.Paginate(playerPermissionStore, req.Pagination, func(key []byte, value []byte) error {
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
*/