package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) AddressRegister(goCtx context.Context, msg *types.MsgAddressRegister) (*types.MsgAddressRegisterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    player, playerFound := k.GetPlayer(ctx, msg.PlayerId, false)
    if (playerFound) {
        // TODO Add address proof signature verification
        k.AddressSetRegisterRequest(ctx, player, msg.Address)
    } else {
        return &types.MsgAddressRegisterResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Could not associate an address with a non-existent player")
    }

	return &types.MsgAddressRegisterResponse{}, nil
}
