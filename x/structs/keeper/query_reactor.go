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

func (k Keeper) ReactorAll(goCtx context.Context, req *types.QueryAllReactorRequest) (*types.QueryAllReactorResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var reactors []types.Reactor
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	reactorStore := prefix.NewStore(store, types.KeyPrefix(types.ReactorKey))

	pageRes, err := query.Paginate(reactorStore, req.Pagination, func(key []byte, value []byte) error {
		var reactor types.Reactor
		if err := k.cdc.Unmarshal(value, &reactor); err != nil {
			return err
		}

        reactor.Load        = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_load, reactor.Id))
        reactor.Capacity    = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, reactor.Id))
        reactor.Fuel        = k.GetGridAttribute(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_fuel, reactor.Id))


		reactors = append(reactors, reactor)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllReactorResponse{Reactor: reactors, Pagination: pageRes}, nil
}

func (k Keeper) Reactor(goCtx context.Context, req *types.QueryGetReactorRequest) (*types.QueryGetReactorResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	reactor, found := k.GetReactor(ctx, req.Id, true)
	if !found {
		return nil, sdkerrors.ErrKeyNotFound
	}

	return &types.QueryGetReactorResponse{Reactor: reactor}, nil
}

/*
func (k Keeper) ReactorPermission(goCtx context.Context, req *types.QueryGetReactorPermissionRequest) (*types.QueryGetMultiplePermissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    reactorId := strconv.FormatUint(req.ReactorId, 10)


    var permissions []*types.QueryPermissionResponse

	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	reactorPermissionStore := prefix.NewStore(store, types.KeyPrefix(types.ReactorPermissionKey))

	pageRes, err := query.Paginate(reactorPermissionStore, req.Pagination, func(key []byte, value []byte) error {
		var permission types.QueryPermissionResponse

	    keys := strings.Split(string(key), "-")

        if (keys[0] == reactorId) {
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

func (k Keeper) ReactorPlayerPermission(goCtx context.Context, req *types.QueryGetReactorPlayerPermissionRequest) (*types.QueryPermissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    reactorId := strconv.FormatUint(req.ReactorId, 10)
    playerId := strconv.FormatUint(req.PlayerId, 10)

	ctx := sdk.UnwrapSDKContext(goCtx)

    recordId := GetReactorPermissionIDBytes(req.ReactorId, req.PlayerId)
    permissionRecord := uint64(k.ReactorGetPlayerPermissionsByBytes(ctx, recordId))

	var permission types.QueryPermissionResponse
    permission.ObjectId = reactorId
    permission.PlayerId = playerId
    permission.PermissionRecord = permissionRecord

	return &permission, nil
}

func (k Keeper) ReactorPermissionAll(goCtx context.Context, req *types.QueryAllReactorPermissionRequest) (*types.QueryGetMultiplePermissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

    var permissions []*types.QueryPermissionResponse

	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	reactorPermissionStore := prefix.NewStore(store, types.KeyPrefix(types.ReactorPermissionKey))

	pageRes, err := query.Paginate(reactorPermissionStore, req.Pagination, func(key []byte, value []byte) error {
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