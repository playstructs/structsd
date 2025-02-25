package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) ProviderDelete(goCtx context.Context, msg *types.MsgProviderDelete) (*types.MsgProviderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    // TODO Provider Delete
        // Need to finish Agreements first so we know what all to delete

	return &types.MsgProviderResponse{}, nil
}
