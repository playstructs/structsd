package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
	//"fmt"
)

func (k msgServer) StructStealthDeactivate(goCtx context.Context, msg *types.MsgStructStealthDeactivate) (*types.MsgStructStatusResponse, error) {
    ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)


    structure := k.GetStructCacheFromId(ctx, msg.StructId)

    permissionError := structure.CanBePlayedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgStructStatusResponse{}, permissionError
    }

    if structure.GetOwner().IsHalted() {
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrPlayerHalted, "Struct (%s) cannot perform actions while Player (%s) is Halted", msg.StructId, structure.GetOwnerId())
    }

    // Is the Struct & Owner online?
    readinessError := structure.ReadinessCheck()
    if (readinessError != nil) {
        structure.GetOwner().Discharge()
        structure.GetOwner().Commit()
        return &types.MsgStructStatusResponse{}, readinessError
    }

    if !structure.IsCommandable() {
        structure.GetOwner().Discharge()
        structure.GetOwner().Commit()
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrInsufficientCharge, "Commanding a Fleet Struct (%s) requires a Command Struct be Online", structure.GetStructId())
    }

    // Is Struct Stealth Mode already activated?
    if !structure.IsHidden() {
        structure.GetOwner().Discharge()
        structure.GetOwner().Commit()
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) already in out of stealth", msg.StructId)
    }


    if (!structure.GetStructType().HasStealthSystem()) {
        structure.GetOwner().Discharge()
        structure.GetOwner().Commit()
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) has no stealth system", msg.StructId)
    }


    // Check Sudo Player Charge
    playerCharge := k.GetPlayerCharge(ctx, structure.GetOwnerId())
    if (playerCharge < structure.GetStructType().GetStealthActivateCharge()) {
        structure.GetOwner().Discharge()
        structure.GetOwner().Commit()
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrInsufficientCharge, "Struct Type (%d) required a charge of %d for stealth mode, but player (%s) only had %d",  structure.GetTypeId(),  structure.GetStructType().GetStealthActivateCharge(), structure.GetOwnerId(), playerCharge)
    }

    structure.GetOwner().Discharge()

    // Set the struct status flag to include hidden
    structure.StatusRemoveHidden()
    structure.Commit()

	return &types.MsgStructStatusResponse{Struct: structure.GetStruct() }, nil

}
