package keeper

import (
	"context"
    "strconv"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) StructBuildComplete(goCtx context.Context, msg *types.MsgStructBuildComplete) (*types.MsgStructStatusResponse, error) {
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
        structure.GetOwner().Discharge()
        structure.GetOwner().Commit()
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrInsufficientCharge, "Struct Type (%d) required a charge of %d to build, but player (%s) only had %d", structure.GetStructType().Id, structure.GetStructType().ActivateCharge, structure.GetOwnerId(), structure.GetOwner().GetCharge() )
    }

    if structure.GetOwner().IsOffline(){
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "The player (%s) is offline ",structure.GetOwnerId())
    }

    // Remove the BuildDraw load
    structure.GetOwner().StructsLoadDecrement(structure.GetStructType().BuildDraw)

    if !structure.GetOwner().CanSupportLoadAddition(structure.GetStructType().PassiveDraw) {
        structure.GetOwner().StructsLoadIncrement(structure.GetStructType().BuildDraw)
        structure.GetOwner().Discharge()
        structure.GetOwner().Commit()
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "Struct Type (%d) required a draw of %d to activate, but player (%s) has %d available", structure.GetStructType().Id, structure.GetStructType().PassiveDraw, structure.GetOwnerId(), structure.GetOwner().GetAvailableCapacity())
    }

    // Check the Proof
    buildStartBlockString           := strconv.FormatUint(structure.GetBlockStartBuild() , 10)
    hashInput                       := structure.GetStructId() + "BUILD" + buildStartBlockString + "NONCE" + msg.Nonce

    currentAge := uint64(ctx.BlockHeight()) - structure.GetBlockStartBuild()

    if (!types.HashBuildAndCheckDifficulty(hashInput, msg.Proof, currentAge, structure.GetStructType().BuildDifficulty)) {
        structure.GetOwner().StructsLoadIncrement(structure.GetStructType().BuildDraw)
        structure.GetOwner().Discharge()
        structure.GetOwner().Commit()
        return &types.MsgStructStatusResponse{}, sdkerrors.Wrapf(types.ErrStructBuildComplete, "Work failure for input (%s) when trying to build Struct %s", hashInput, structure.GetStructId())
    }

    structure.GoOnline()
    structure.Commit()

	return &types.MsgStructStatusResponse{Struct: structure.GetStruct()}, nil
}
