package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) ProviderUpdateDurationMinimum(goCtx context.Context, msg *types.MsgProviderUpdateDurationMinimum) (*types.MsgProviderResponse, error) {
    emptyResponse := &types.MsgProviderResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
    cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)
    activePlayer, lookupErr := cc.GetSigningPlayer(msg.Creator)
    if lookupErr != nil {
        return emptyResponse, lookupErr
    }

    provider, providerErr := cc.GetExistingProvider(msg.ProviderId)
    if providerErr != nil {
        return emptyResponse, providerErr
    }

    permissionError := provider.CanBeUpdatedBy(activePlayer)
    if (permissionError != nil) {
        return emptyResponse, permissionError
    }

    paramErr := provider.SetDurationMinimum(msg.NewMinimumDuration)
    if paramErr != nil {
        return emptyResponse, paramErr
    }

	cc.CommitAll()
	return &types.MsgProviderResponse{}, nil
}
