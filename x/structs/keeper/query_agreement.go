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

func (k Keeper) AgreementAll(goCtx context.Context, req *types.QueryAllAgreementRequest) (*types.QueryAllAgreementResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var agreements []types.Agreement
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	agreementStore := prefix.NewStore(store, types.KeyPrefix(types.AgreementKey))

	pageRes, err := query.Paginate(agreementStore, req.Pagination, func(key []byte, value []byte) error {
		var agreement types.Agreement
		if err := k.cdc.Unmarshal(value, &agreement); err != nil {
			return err
		}

		agreements = append(agreements, agreement)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllAgreementResponse{Agreement: agreements, Pagination: pageRes}, nil
}

func (k Keeper) Agreement(goCtx context.Context, req *types.QueryGetAgreementRequest) (*types.QueryGetAgreementResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	agreement, found := k.GetAgreement(ctx, req.Id)
	if !found {
		return nil, types.ErrObjectNotFound
	}

	return &types.QueryGetAgreementResponse{Agreement: agreement}, nil
}

func (k Keeper) AgreementAllByProvider(goCtx context.Context, req *types.QueryAllAgreementByProviderRequest) (*types.QueryAllAgreementResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var agreements []types.Agreement
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	agreementStore := prefix.NewStore(store, AgreementProviderKeyPrefix(req.ProviderId))

	pageRes, err := query.Paginate(agreementStore, req.Pagination, func(key []byte, value []byte) error {
		agreement, found := k.GetAgreement(ctx, string(key))

        if found {
            agreements = append(agreements, agreement)
        }

		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllAgreementResponse{Agreement: agreements, Pagination: pageRes}, nil
}

