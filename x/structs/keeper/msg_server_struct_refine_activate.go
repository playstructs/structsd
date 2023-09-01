package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) StructRefineActivate(goCtx context.Context, msg *types.MsgStructRefineActivate) (*types.MsgStructRefineActivateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    playerId := k.GetPlayerIdFromAddress(ctx, msg.Creator)
    if (playerId == 0) {
        return &types.MsgStructRefineActivateResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct build initialization requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayer(ctx, playerId)

    if (!k.SubstationIsOnline(ctx, player.SubstationId)){
        return &types.MsgStructRefineActivateResponse{}, sdkerrors.Wrapf(types.ErrSubstationOffline, "The players substation (%d) is offline ",player.SubstationId)
    }

    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)
    if ((playerPermissions&types.AddressPermissionPlay) == 0) {
        return &types.MsgStructRefineActivateResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }

    structure, structureFound := k.GetStruct(ctx, msg.StructId)
    if (!structureFound) {
        return &types.MsgStructRefineActivateResponse{}, sdkerrors.Wrapf(types.ErrStructNotFound, "Struct (%d) not found", msg.StructId)
    }

    if (structure.Type != "Refinery") {
        return &types.MsgStructRefineActivateResponse{}, sdkerrors.Wrapf(types.ErrStructRefineActivate, "This struct (%d) has no refining systems", msg.StructId)
    }

    if (structure.Status != "ACTIVE") {
        return &types.MsgStructRefineActivateResponse{}, sdkerrors.Wrapf(types.ErrStructRefineActivate, "This struct (%d) is not online", msg.StructId)
    }


    if (structure.RefiningSystemStatus != "INACTIVE") {
        return &types.MsgStructRefineActivateResponse{}, sdkerrors.Wrapf(types.ErrStructRefineActivate, "This Refining System for struct (%d) is already active", msg.StructId)
    }

    /*
     * Until we let players give out Play permissions, this can't happened
     */
    if (player.Id != structure.Owner) {
       return &types.MsgStructRefineActivateResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlayerPlay, "For now you (%d) can't sudo structs, no permission for action on Struct (%d)", structure.Owner, msg.StructId)
    }

    _, err = k.PlayerIncrementLoad(ctx, player, structure.ActiveRefiningSystemDraw)

    if (err != nil) {
        return &types.MsgStructRefineActivateResponse{}, sdkerrors.Wrapf(types.ErrStructRefineActivate, "Not enough power to bring Refining System online for Struct (%d)", msg.StructId)
    }

    structure.SetRefiningSystemStatus("ACTIVE")
    structure.SetRefiningSystemActivationBlock(uint64(ctx.BlockHeight()))
    k.SetStruct(ctx, structure)

	return &types.MsgStructRefineActivateResponse{Struct: structure}, nil
}