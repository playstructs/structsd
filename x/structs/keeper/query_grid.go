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

    "encoding/binary"
    //"strings"
    //"strconv"
)


func (k Keeper) Grid(goCtx context.Context, req *types.QueryGetGridRequest) (*types.QueryGetGridResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

    var grids types.GridRecord

    gridsValue       := k.GetGridAttribute(ctx, req.AttributeId)
    grids.AttributeId   = req.AttributeId
    grids.Value         = uint64(gridsValue)

	return &types.QueryGetGridResponse{GridRecord: &grids}, nil
}


func (k Keeper) GridAll(goCtx context.Context, req *types.QueryAllGridRequest) (*types.QueryAllGridResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

     var grids []*types.GridRecord

 	ctx := sdk.UnwrapSDKContext(goCtx)

 	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
 	gridStore := prefix.NewStore(store, types.KeyPrefix(types.GridAttributeKey))

 	pageRes, err := query.Paginate(gridStore, req.Pagination, func(key []byte, value []byte) error {

        grids = append(grids, &types.GridRecord{AttributeId: string(key), Value: binary.BigEndian.Uint64(value)})

         return nil
 	})

	return &types.QueryAllGridResponse{GridRecords: grids, Pagination: pageRes}, err
}