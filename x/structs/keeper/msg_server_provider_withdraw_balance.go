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
    ctx := sdk.UnwrapSDKContext(goCtx)
    cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)
    activePlayer, _ := cc.GetPlayerByAddress(msg.Creator)

    provider := cc.GetProvider(msg.ProviderId)

    permissionError := provider.CanWithdrawBalance(activePlayer)
    if (permissionError != nil) {
        return &types.MsgProviderResponse{}, permissionError
    }

    err := provider.WithdrawBalanceAndCommit(msg.DestinationAddress)
    if (err != nil) {
        return &types.MsgProviderResponse{}, err
    }

	cc.CommitAll()
	return &types.MsgProviderResponse{}, nil
}
