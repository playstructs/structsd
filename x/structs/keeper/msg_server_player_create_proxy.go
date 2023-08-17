package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) PlayerCreateProxy(goCtx context.Context, msg *types.MsgPlayerCreateProxy) (*types.MsgPlayerCreateProxyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx


	// TODO Add a verification process to ensure that the proxy agent had the rights to do this
	// Basically, the player will need to provide some sort of signature that can then be verified here

	// Look up requesting account
	// look up destination guild
	// look up destination substation
	// create new player




	return &types.MsgPlayerCreateProxyResponse{}, nil
}
