package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"structs/x/structs/types"
)

func (k Keeper) Vault(ctx context.Context, req *types.QueryVaultRequest) (*types.QueryVaultResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	cc := k.NewCurrentContext(ctx)
	vault, err := cc.GetVault(req.OwnerId)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	address := vault.Address()
	_, indexed := k.GetVaultOwner(ctx, address)
	return &types.QueryVaultResponse{
		OwnerId:     vault.ID(),
		Address:     address.String(),
		Provisioned: indexed && k.accountKeeper.GetAccount(ctx, address) != nil,
	}, nil
}
