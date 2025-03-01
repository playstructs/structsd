package keeper

import (
	"context"
    //"strconv"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) StructBuildCancel(goCtx context.Context, msg *types.MsgStructBuildCancel) (*types.MsgStructStatusResponse, error) {
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

    if !structure.LoadStruct(){
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Struct (%s) does not exist", msg.StructId)
    }

    if structure.IsBuilt() {
        structure.GetOwner().Discharge()
        structure.GetOwner().Commit()
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct (%s) already built", msg.StructId)
    }

    // Check Player Charge
    if (structure.GetOwner().GetCharge() < structure.GetStructType().ActivateCharge) {
        err := sdkerrors.Wrapf(types.ErrInsufficientCharge, "Struct Type (%d) required a charge of %d to build, but player (%s) only had %d", structure.GetStructType().Id, structure.GetStructType().ActivateCharge, structure.GetOwnerId(), structure.GetOwner().GetCharge() )
        structure.GetOwner().Discharge()
        structure.GetOwner().Commit()
        return &types.MsgStructStatusResponse{}, err
    }

    if structure.GetOwner().IsOffline(){
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "The player (%s) is offline ",structure.GetOwnerId())
    }


    structure.DestroyAndCommit()

	return &types.MsgStructStatusResponse{Struct: structure.GetStruct()}, nil
}
