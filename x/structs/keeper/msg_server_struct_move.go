package keeper

import (
	"context"
    /*
    "strconv"

    //"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	*/
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
    cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
    k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
       return &types.MsgStructStatusResponse{}, err
    }

    structure := cc.GetStruct(msg.StructId)

    // Check to see if the caller has permissions to proceed
    permissionError := structure.CanBePlayedBy(callingPlayer)
    if (permissionError != nil) {
        return &types.MsgStructStatusResponse{}, permissionError
    }

    // Check Player Charge
    if structure.GetOwner().GetCharge() < structure.GetStructType().MoveCharge {
        err := types.NewInsufficientChargeError(structure.GetOwnerId(), structure.GetStructType().MoveCharge, structure.GetOwner().GetCharge(), "move").WithStructType(structure.GetTypeId())
        return &types.MsgStructStatusResponse{}, err
    }

    moveErr := structure.AttemptMove( msg.LocationType, msg.Ambit, msg.Slot)
    if (moveErr != nil) {
        return &types.MsgStructStatusResponse{}, moveErr
    }

    structure.GetOwner().Discharge()

	cc.CommitAll()
	return &types.MsgStructStatusResponse{}, nil
}
