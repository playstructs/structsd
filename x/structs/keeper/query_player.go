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

func (k Keeper) PlayerAll(goCtx context.Context, req *types.QueryAllPlayerRequest) (*types.QueryAllPlayerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var players []types.Player
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	playerStore := prefix.NewStore(store, types.KeyPrefix(types.PlayerKey))

	pageRes, err := query.Paginate(playerStore, req.Pagination, func(key []byte, value []byte) error {
		var player types.Player
		if err := k.cdc.Unmarshal(value, &player); err != nil {
			return err
		}

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
	player, found := k.GetPlayer(ctx, req.Id)
	if !found {
		return nil, types.ErrObjectNotFound
	}

	gridAttributes := k.GetGridAttributesByObject(ctx, req.Id)
	playerInventory := k.GetPlayerInventory(ctx, player.PrimaryAddress)

	return &types.QueryGetPlayerResponse{Player: player, GridAttributes: &gridAttributes, PlayerInventory: &playerInventory}, nil
}
