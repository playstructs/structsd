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

func (k Keeper) StructAll(goCtx context.Context, req *types.QueryAllStructRequest) (*types.QueryAllStructResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var structures []types.Struct
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	structureStore := prefix.NewStore(store, types.KeyPrefix(types.StructKey))

	pageRes, err := query.Paginate(structureStore, req.Pagination, func(key []byte, value []byte) error {
		var structure types.Struct
		if err := k.cdc.Unmarshal(value, &structure); err != nil {
			return err
		}

		structures = append(structures, structure)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllStructResponse{Struct: structures, Pagination: pageRes}, nil
}

func (k Keeper) Struct(goCtx context.Context, req *types.QueryGetStructRequest) (*types.QueryGetStructResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	structure, found := k.GetStruct(ctx, req.Id)
	if !found {
		return nil, sdkerrors.ErrKeyNotFound
	}

	return &types.QueryGetStructResponse{Struct: structure}, nil
}
