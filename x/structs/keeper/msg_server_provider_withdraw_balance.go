package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

/*
message MsgProviderWithdrawBalance {
  option (cosmos.msg.v1.signer) = "creator";

  string creator            = 1;
  string providerId         = 2;
  string destinationAddress = 3;
}
*/
func (k msgServer) ProviderWithdrawBalance(goCtx context.Context, msg *types.MsgProviderWithdrawBalance) (*types.MsgProviderResponse, error) {
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

    permissionError := provider.CanWithdrawBalanceBy(activePlayer)
    if (permissionError != nil) {
        return emptyResponse, permissionError
    }

    err := provider.WithdrawBalanceAndCommit(msg.DestinationAddress)
    if (err != nil) {
        return emptyResponse, err
    }

	cc.CommitAll()
	return &types.MsgProviderResponse{}, nil
}
