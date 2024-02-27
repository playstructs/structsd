package keeper

import (
	"context"
 	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) StructRefineDeactivate(goCtx context.Context, msg *types.MsgStructRefineDeactivate) (*types.MsgStructRefineDeactivateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    playerIndex := k.GetPlayerIndexFromAddress(ctx, msg.Creator)
    if (playerIndex == 0) {
        return &types.MsgStructRefineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct build initialization requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayerFromIndex(ctx, playerIndex, true)

    if (!k.SubstationIsOnline(ctx, player.SubstationId)){
        return &types.MsgStructRefineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrSubstationOffline, "The players substation (%d) is offline ",player.SubstationId)
    }

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    // Make sure the address calling this has Play permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.Permission(types.AddressPermissionPlay))) {
        return &types.MsgStructActivateResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }

    structure, structureFound := k.GetStruct(ctx, msg.StructId)
    if (!structureFound) {
        return &types.MsgStructRefineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrStructNotFound, "Struct (%d) not found", msg.StructId)
    }

    if (structure.Type != "Refinery") {
        return &types.MsgStructRefineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrStructRefineDeactivate, "This struct (%d) has no refining systems", msg.StructId)
    }

    if (structure.Status != "ACTIVE") {
        return &types.MsgStructRefineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrStructRefineDeactivate, "This struct (%d) is not online", msg.StructId)
    }


    if (structure.RefiningSystemStatus == "INACTIVE") {
        return &types.MsgStructRefineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrStructRefineDeactivate, "This Refining System for struct (%d) is already inactive", msg.StructId)
    }

    /*
     * Until we let players give out Play permissions, this can't happened
     */
    if (player.Id != structure.Owner) {
       return &types.MsgStructRefineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlayerPlay, "For now you (%d) can't sudo structs, no permission for action on Struct (%d)", structure.Owner, structure.Id)
    }

    _, _ = k.PlayerDecrementLoad(ctx, player.Id, structure.ActiveRefiningSystemDraw)


    structure.SetRefiningSystemStatus("INACTIVE")
    structure.SetRefiningSystemActivationBlock(0)
    k.SetStruct(ctx, structure)

	return &types.MsgStructRefineDeactivateResponse{Struct: structure}, nil
}
