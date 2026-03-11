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
    emptyResponse := &types.MsgStructStatusResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
       return emptyResponse, err
    }

    // Load the Owner Player
    owner, err := cc.GetPlayer(msg.PlayerId)
    if (err != nil) {
        return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "struct_build_initiate")
    }

    // Check address play permissions
    permissionError := owner.CanBePlayedBy(callingPlayer)
    if (permissionError != nil) {
        return emptyResponse, permissionError
    }

    // Load the Struct Type
    structType, structTypeFound := cc.GetStructType(msg.StructTypeId)
    if !structTypeFound {
        return emptyResponse, types.NewObjectNotFoundError("struct_type", "").WithIndex(msg.StructTypeId)
    }

    // Check that the player can build more of this type of Struct
    if (structType.GetStructType().BuildLimit > 0) {
        if (owner.GetBuiltQuantity(msg.StructTypeId) >= structType.GetStructType().BuildLimit) {
            return emptyResponse, types.NewPlayerPowerError(owner.GetPlayerId(), "capacity_exceeded").WithCapacity(structType.GetStructType().BuildLimit, owner.GetBuiltQuantity(msg.StructTypeId))
        }
    }

    // Check Player Charge
    if (owner.GetCharge() < structType.GetStructType().BuildCharge) {
        err := types.NewInsufficientChargeError(owner.GetPlayerId(), structType.GetStructType().BuildCharge, owner.GetCharge(), "build").WithStructType(msg.StructTypeId)
        return emptyResponse, err
    }


    if !owner.IsOnline(){
        return emptyResponse, types.NewPlayerPowerError(owner.GetPlayerId(), "offline")
    }


    if !owner.CanSupportLoadAddition(structType.GetStructType().BuildDraw) {
        return emptyResponse, types.NewPlayerPowerError(owner.GetPlayerId(), "capacity_exceeded").WithCapacity(structType.GetStructType().BuildDraw, owner.GetAvailableCapacity())
    }

    k.logger.Info("Struct Materializing", "structType", structType.GetStructType().Type, "ambit", msg.OperatingAmbit, "slot", msg.Slot)
    structure, err := cc.InitiateStruct(msg.Creator, owner, structType, msg.OperatingAmbit, msg.Slot)
    if (err != nil) {
        return emptyResponse, err
    }

    owner.Discharge()

	cc.CommitAll()
	return &types.MsgStructStatusResponse{Struct: structure.GetStruct()}, nil
}
