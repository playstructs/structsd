package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
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
        return &types.MsgStructStatusResponse{}, types.NewPlayerRequiredError(msg.Creator, "struct_build_initiate")
    }

    // Check address play permissions
    permissionError := owner.CanBePlayedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgStructStatusResponse{}, permissionError
    }

    if owner.IsHalted() {
        return &types.MsgStructStatusResponse{}, types.NewPlayerHaltedError(msg.PlayerId, "struct_build_initiate")
    }

    // Load the Struct Type
    structType, structTypeFound := k.GetStructType(ctx, msg.StructTypeId)
    if !structTypeFound {
        return &types.MsgStructStatusResponse{}, types.NewObjectNotFoundError("struct_type", "").WithIndex(msg.StructTypeId)
    }

    // Check that the player can build more of this type of Struct
    if (structType.GetBuildLimit() > 0) {
        if (owner.GetBuiltQuantity(msg.StructTypeId) >= structType.GetBuildLimit()) {
            owner.Discharge()
            owner.Commit()
            return &types.MsgStructStatusResponse{}, types.NewPlayerPowerError(owner.GetPlayerId(), "capacity_exceeded").WithCapacity(structType.GetBuildLimit(), owner.GetBuiltQuantity(msg.StructTypeId))
        }
    }

    // Check Player Charge
    if (owner.GetCharge() < structType.BuildCharge) {
        err := types.NewInsufficientChargeError(owner.GetPlayerId(), structType.BuildCharge, owner.GetCharge(), "build").WithStructType(msg.StructTypeId)
        owner.Discharge()
        owner.Commit()
        return &types.MsgStructStatusResponse{}, err
    }


    if !owner.IsOnline(){
        return &types.MsgStructStatusResponse{}, types.NewPlayerPowerError(owner.GetPlayerId(), "offline")
    }


    if !owner.CanSupportLoadAddition(structType.BuildDraw) {
        owner.Discharge()
        owner.Commit()
        return &types.MsgStructStatusResponse{}, types.NewPlayerPowerError(owner.GetPlayerId(), "capacity_exceeded").WithCapacity(structType.BuildDraw, owner.GetAvailableCapacity())
    }


    k.logger.Info("Struct Materializing", "structType", structType.Type, "ambit", msg.OperatingAmbit, "slot", msg.Slot)
    structure, err := k.InitiateStruct(ctx, msg.Creator, &owner, &structType, msg.OperatingAmbit, msg.Slot)
    if (err != nil) {
        return &types.MsgStructStatusResponse{}, err
    }

    owner.Discharge()
    structure.Commit()


	return &types.MsgStructStatusResponse{Struct: structure.GetStruct()}, nil
}
