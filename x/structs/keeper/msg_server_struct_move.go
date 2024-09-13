package keeper

import (
	"context"
    /*
    "strconv"

    //"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
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

    // Add an Active Address record to the
    // indexer for UI requirements
    k.AddressEmitActivity(ctx, msg.Creator)

    structure := k.GetStructCacheFromId(ctx, msg.StructId)

    // Check to see if the caller has permissions to proceed
    permissionError := structure.CanBePlayedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgStructStatusResponse{}, permissionError
    }

    // I think we need to check power, but that might be being checked in AttemptMove
    breaking change for comment above

    err := structure.AttemptMove(msg.LocationId, msg.LocationType, msg.Ambit, msg.Slot)
    if (err != nil) {
        return &types.MsgStructStatusResponse{}, err
    }

    structure.GetOwner().Discharge()
    structure.Commit()

	return &types.MsgStructStatusResponse{}, nil
}
