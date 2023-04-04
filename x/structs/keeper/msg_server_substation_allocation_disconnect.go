package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationAllocationDisconnect(goCtx context.Context, msg *types.MsgSubstationAllocationDisconnect) (*types.MsgSubstationAllocationDisconnectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgSubstationAllocationDisconnectResponse{}, nil
}
