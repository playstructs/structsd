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

func (k Keeper) StructTypeAll(goCtx context.Context, req *types.QueryAllStructTypeRequest) (*types.QueryAllStructTypeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var structTypes []types.StructType
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	structTypeStore := prefix.NewStore(store, types.KeyPrefix(types.StructTypeKey))

	pageRes, err := query.Paginate(structTypeStore, req.Pagination, func(key []byte, value []byte) error {
		var structType types.StructType
		if err := k.cdc.Unmarshal(value, &structType); err != nil {
			return err
		}

		structTypes = append(structTypes, structType)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllStructTypeResponse{StructType: structTypes, Pagination: pageRes}, nil
}

func (k Keeper) StructType(goCtx context.Context, req *types.QueryGetStructTypeRequest) (*types.QueryGetStructTypeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	structType, found := k.GetStructType(ctx, req.Id)
	if !found {
		return nil, types.ErrObjectNotFound
	}

	return &types.QueryGetStructTypeResponse{StructType: structType}, nil
}
