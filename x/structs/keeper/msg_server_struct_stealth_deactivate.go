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
    defer cc.CommitAll()

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)


    structure := cc.GetStruct(msg.StructId)

    permissionError := structure.CanBePlayedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgStructStatusResponse{}, permissionError
    }

    if structure.GetOwner().IsHalted() {
        return &types.MsgStructStatusResponse{}, types.NewPlayerHaltedError(structure.GetOwnerId(), "stealth_deactivate").WithStruct(msg.StructId)
    }

    // Is the Struct & Owner online?
    readinessError := structure.ReadinessCheck()
    if (readinessError != nil) {
        structure.GetOwner().Discharge()
        return &types.MsgStructStatusResponse{}, readinessError
    }

    if !structure.IsCommandable() {
        structure.GetOwner().Discharge()
        return &types.MsgStructStatusResponse{}, types.NewFleetCommandError(structure.GetStructId(), "no_command_struct")
    }

    // Is Struct Stealth Mode already activated?
    if !structure.IsHidden() {
        structure.GetOwner().Discharge()
        return &types.MsgStructStatusResponse{}, types.NewStructStateError(msg.StructId, "visible", "hidden", "stealth_deactivate")
    }


    if (!structure.GetStructType().HasStealthSystem()) {
        structure.GetOwner().Discharge()
        return &types.MsgStructStatusResponse{}, types.NewStructCapabilityError(msg.StructId, "stealth")
    }


    // Check Sudo Player Charge
    playerCharge := k.GetPlayerCharge(ctx, structure.GetOwnerId())
    if (playerCharge < structure.GetStructType().GetStealthActivateCharge()) {
        structure.GetOwner().Discharge()
        return &types.MsgStructStatusResponse{}, types.NewInsufficientChargeError(structure.GetOwnerId(), structure.GetStructType().GetStealthActivateCharge(), playerCharge, "stealth").WithStructType(structure.GetTypeId())
    }

    structure.GetOwner().Discharge()

    // Set the struct status flag to include hidden
    structure.StatusRemoveHidden()

	return &types.MsgStructStatusResponse{Struct: structure.GetStruct() }, nil

}
