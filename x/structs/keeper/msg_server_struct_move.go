package keeper

import (
	"context"
    /*
    "strconv"

    //"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	*/
    sdkerrors "cosmossdk.io/errors"
    sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

/*
message MsgStructMove {
  option (cosmos.msg.v1.signer) = "creator";

  string creator      = 1;
  string structId     = 2;
  string locationId   = 3;
  string locationType = 4;
  ambit  ambit        = 5;
  uint64 slot         = 6;
}

*/
func (k msgServer) StructMove(goCtx context.Context, msg *types.MsgStructMove) (*types.MsgStructStatusResponse, error) {
    ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
    k.AddressEmitActivity(ctx, msg.Creator)

    structure := k.GetStructCacheFromId(ctx, msg.StructId)

    // Check to see if the caller has permissions to proceed
    permissionError := structure.CanBePlayedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgStructStatusResponse{}, permissionError
    }

    if structure.GetOwner().IsHalted() {
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrPlayerHalted, "Struct (%s) cannot perform actions while Player (%s) is Halted", msg.StructId, structure.GetOwnerId())
    }

    // Check Player Charge
    if structure.GetOwner().GetCharge() < structure.GetStructType().GetMoveCharge() {
        err := sdkerrors.Wrapf(types.ErrInsufficientCharge, "Struct Type (%d) required a charge of %d for movement, but player (%s) only had %d", structure.GetStructType().GetId(), structure.GetStructType().GetMoveCharge(), structure.GetOwnerId(), structure.GetOwner().GetCharge() )
        structure.GetOwner().Discharge()
        structure.GetOwner().Commit()
        return &types.MsgStructStatusResponse{}, err
    }

    err := structure.AttemptMove( msg.LocationType, msg.Ambit, msg.Slot)
    if (err != nil) {
        return &types.MsgStructStatusResponse{}, err
    }

    structure.GetOwner().Discharge()
    structure.Commit()

	return &types.MsgStructStatusResponse{}, nil
}
