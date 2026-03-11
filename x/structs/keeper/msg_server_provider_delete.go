package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) ProviderDelete(goCtx context.Context, msg *types.MsgProviderDelete) (*types.MsgProviderResponse, error) {
    emptyResponse := &types.MsgProviderResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
    cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)
    activePlayer, _ := cc.GetPlayerByAddress(msg.Creator)

    provider := cc.GetProvider(msg.ProviderId)

    permissionError := provider.CanBeDeletedBy(activePlayer)
    if (permissionError != nil) {
        return emptyResponse, permissionError
    }

    paramErr := provider.Delete()
    if paramErr != nil {
        return emptyResponse, paramErr
    }

	cc.CommitAll()
	return &types.MsgProviderResponse{}, nil
}
