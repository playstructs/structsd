package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
	//"fmt"
)

func (k msgServer) StructStealthDeactivate(goCtx context.Context, msg *types.MsgStructStealthDeactivate) (*types.MsgStructStatusResponse, error) {
    ctx := sdk.UnwrapSDKContext(goCtx)
    cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)


    structure := cc.GetStruct(msg.StructId)

    permissionError := structure.CanBePlayedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgStructStatusResponse{}, permissionError
    }

    // Is the Struct & Owner online?
    readinessError := structure.ReadinessCheck()
    if (readinessError != nil) {
        return &types.MsgStructStatusResponse{}, readinessError
    }

    if !structure.IsCommandable() {
        return &types.MsgStructStatusResponse{}, types.NewFleetCommandError(structure.GetStructId(), "no_command_struct")
    }

    // Is Struct Stealth Mode already activated?
    if !structure.IsHidden() {
        return &types.MsgStructStatusResponse{}, types.NewStructStateError(msg.StructId, "visible", "hidden", "stealth_deactivate")
    }

    if (!structure.GetStructType().HasStealthSystem()) {
        return &types.MsgStructStatusResponse{}, types.NewStructCapabilityError(msg.StructId, "stealth")
    }


    // Check Sudo Player Charge
    if (structure.GetOwner().GetCharge()  < structure.GetStructType().StealthActivateCharge) {
        return &types.MsgStructStatusResponse{}, types.NewInsufficientChargeError(structure.GetOwnerId(), structure.GetStructType().StealthActivateCharge, structure.GetOwner().GetCharge() , "stealth").WithStructType(structure.GetTypeId())
    }

    structure.GetOwner().Discharge()

    // Set the struct status flag to include hidden
    structure.StatusRemoveHidden()

	cc.CommitAll()
	return &types.MsgStructStatusResponse{Struct: structure.GetStruct() }, nil

}
