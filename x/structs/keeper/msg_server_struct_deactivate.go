package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
	//"fmt"
)

func (k msgServer) StructDeactivate(goCtx context.Context, msg *types.MsgStructDeactivate) (*types.MsgStructStatusResponse, error) {
    ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    // load struct
    structure := k.GetStructCacheFromId(ctx, msg.StructId)

    // Check to see if the caller has permissions to proceed
    permissionError := structure.CanBePlayedBy(msg.Creator)
    if (permissionError != nil) {
        return &types.MsgStructStatusResponse{}, permissionError
    }

    if structure.GetOwner().IsHalted() {
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrPlayerHalted, "Struct (%s) cannot perform actions while Player (%s) is Halted", msg.StructId, structure.GetOwnerId())
    }

    if !structure.LoadStruct(){
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Struct (%s) does not exist", msg.StructId)
    }

    if !structure.IsBuilt() {
        structure.GetOwner().Discharge()
        structure.GetOwner().Commit()
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) isn't built yet", msg.StructId)
    }

    if structure.IsOffline() {
        structure.GetOwner().Discharge()
        structure.GetOwner().Commit()
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) already offline", msg.StructId)
    }

    if structure.GetOwner().IsOffline(){
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "The player (%s) is offline ",structure.GetOwnerId())
    }


    structure.GoOffline()
    structure.Commit()

	return &types.MsgStructStatusResponse{Struct: structure.GetStruct()}, nil
}
