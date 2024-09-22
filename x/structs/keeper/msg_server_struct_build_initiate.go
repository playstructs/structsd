package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
	"fmt"
)

/* MsgStructBuildInitiate
  string creator        = 1;
  uint64 structTypeId   = 2;
  string planetId       = 3;
  ambit operatingAmbit  = 4;
  uint64 slot           = 5;
  */

func (k msgServer) StructBuildInitiate(goCtx context.Context, msg *types.MsgStructBuildInitiate) (*types.MsgStructStatusResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    // Load the Owner Player
    owner, err := k.GetPlayerCacheFromId(ctx, msg.PlayerId)
    if (err != nil) {
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct build initialization requires Player account but none associated with %s", msg.Creator)
    }

    // Check address play permissions
    permissionError := owner.CanBePlayedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgStructStatusResponse{}, permissionError
    }

    // Load the Struct Type
    structType, structTypeFound := k.GetStructType(ctx, msg.StructTypeId)
    if !structTypeFound {
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Struct Type (%d) was not found. Building a Struct with schematics might be tough", msg.StructTypeId)
    }

    // Check that the player can build more of this type of Struct
    if (structType.GetBuildLimit() > 0) {
        if (owner.GetBuiltQuantity(msg.StructTypeId) >= structType.GetBuildLimit()) {
            owner.Discharge()
            owner.Commit()
            return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "The player (%s) cannot build more of this type ",owner.GetPlayerId())
        }
    }

    // Check Player Charge
    if (owner.GetCharge() < structType.BuildCharge) {
        owner.Discharge()
        owner.Commit()
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrInsufficientCharge, "Struct Type (%d) required a charge of %d to build, but player (%s) only had %d", msg.StructTypeId, structType.BuildCharge, owner.GetPlayerId(), owner.GetCharge() )
    }


    if !owner.IsOnline(){
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "The player (%s) is offline ",owner.GetPlayerId())
    }


    if !owner.CanSupportLoadAddition(structType.BuildDraw) {
        owner.Discharge()
        owner.Commit()
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct Type (%d) required a draw of %d during build, but player (%s) has %d available", msg.StructTypeId, structType.BuildDraw, owner.GetPlayerId(),owner.GetAvailableCapacity())
    }

    // todo actually verify the darn location

    fmt.Printf("Trying to materialized a Struct \n")
    fmt.Printf("Struct Type: %s ", structType.Type)
    fmt.Printf("Destination: %s %s %d", msg.LocationId, msg.LocationType, msg.Slot)
    structure, err := k.InitiateStruct(ctx, msg.Creator, &owner, &structType, msg.LocationId, msg.LocationType, msg.OperatingAmbit, msg.Slot)
    if (err != nil) {
        return &types.MsgStructStatusResponse{}, err
    }

    owner.Discharge()
    structure.Commit()
    structure.GetPlanet().Commit()


	return &types.MsgStructStatusResponse{Struct: structure.GetStruct()}, nil
}
