package keeper

import (
	"context"

    //"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"


	"structs/x/structs/types"
)


func (k msgServer) StructDefenseClear(goCtx context.Context, msg *types.MsgStructDefenseClear) (*types.MsgStructStatusResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    // load struct
    structure := k.GetStructCacheFromId(ctx, msg.DefenderStructId)

    // Check to see if the caller has permissions to proceed
    permissionError := structure.CanBePlayedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgStructStatusResponse{}, permissionError
    }

    if !structure.LoadStruct(){
        return &types.MsgStructStatusResponse{}, types.NewObjectNotFoundError("struct", msg.DefenderStructId)
    }

    if structure.GetOwner().IsHalted() {
        return &types.MsgStructStatusResponse{}, types.NewPlayerHaltedError(structure.GetOwnerId(), "defense_clear").WithStruct(msg.DefenderStructId)
    }

    if structure.IsOffline() {
        structure.GetOwner().Discharge()
        structure.GetOwner().Commit()
        return &types.MsgStructStatusResponse{}, types.NewStructStateError(msg.DefenderStructId, "offline", "online", "defense_clear")
    }

    if !structure.IsCommandable() {
        structure.GetOwner().Discharge()
        structure.GetOwner().Commit()
        return &types.MsgStructStatusResponse{}, types.NewFleetCommandError(structure.GetStructId(), "no_command_struct")
    }

    // Check Player Charge
    if (structure.GetOwner().GetCharge() < structure.GetStructType().DefendChangeCharge) {
        err := types.NewInsufficientChargeError(structure.GetOwnerId(), structure.GetStructType().DefendChangeCharge, structure.GetOwner().GetCharge(), "defend").WithStructType(structure.GetStructType().Id)
        structure.GetOwner().Discharge()
        structure.GetOwner().Commit()
        return &types.MsgStructStatusResponse{}, err
    }

    if structure.GetOwner().IsOffline(){
        return &types.MsgStructStatusResponse{}, types.NewPlayerPowerError(structure.GetOwnerId(), "offline")
    }

    protectedStructIndex := k.GetStructAttribute(ctx, GetStructAttributeIDByObjectId(types.StructAttributeType_protectedStructIndex, msg.DefenderStructId))
    if (protectedStructIndex == 0) {
        structure.GetOwner().Discharge()
        structure.GetOwner().Commit()
        return &types.MsgStructStatusResponse{}, types.NewStructStateError(msg.DefenderStructId, "not_defending", "defending", "defense_clear")
    }
    protectedStructId := GetObjectID(types.ObjectType_struct, protectedStructIndex)

    k.ClearStructDefender(ctx, protectedStructId, msg.DefenderStructId)

    structure.GetOwner().Discharge()
    structure.Commit()

	return &types.MsgStructStatusResponse{}, nil
}
