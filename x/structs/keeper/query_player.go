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

    //"encoding/binary"
    //"strings"
    //"strconv"
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
