package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

func (k msgServer) SubstationCreate(goCtx context.Context, msg *types.MsgSubstationCreate) (*types.MsgSubstationCreateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgSubstationCreateResponse{}, nil
}
