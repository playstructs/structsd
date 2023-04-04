package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationPlayerConnect(goCtx context.Context, msg *types.MsgSubstationPlayerConnect) (*types.MsgSubstationPlayerConnectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgSubstationPlayerConnectResponse{}, nil
}
