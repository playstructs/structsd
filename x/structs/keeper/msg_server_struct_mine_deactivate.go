package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) StructMineDeactivate(goCtx context.Context, msg *types.MsgStructMineDeactivate) (*types.MsgStructMineDeactivateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    playerIndex := k.GetPlayerIndexFromAddress(ctx, msg.Creator)
    if (playerIndex == 0) {
        return &types.MsgStructMineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct build initialization requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayerFromIndex(ctx, playerIndex, true)

    if (!player.IsOnline()){
        return &types.MsgStructMineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "The player (%s) is offline ",player.Id)
    }

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    // Make sure the address calling this has Play permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionPlay)) {
        return &types.MsgStructMineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }

    structure, structureFound := k.GetStruct(ctx, msg.StructId)
    if (!structureFound) {
        return &types.MsgStructMineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrStructNotFound, "Struct (%s) not found", msg.StructId)
    }

    if (structure.Type != "Mining Rig") {
        return &types.MsgStructMineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrStructMineDeactivate, "This struct (%s) has no mining systems", msg.StructId)
    }

    if (structure.Status != "ACTIVE") {
        return &types.MsgStructMineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrStructMineDeactivate, "This struct (%s) is not online", msg.StructId)
    }


    if (structure.MiningSystemStatus == "INACTIVE") {
        return &types.MsgStructMineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrStructMineDeactivate, "This Mining System for struct (%s) is already inactive", msg.StructId)
    }

    /*
     * Until we let players give out Play permissions, this can't happened
     */
    if (player.Id != structure.Owner) {
       return &types.MsgStructMineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlayerPlay, "For now you (%s) can't sudo structs, no permission for action on Struct (%s)", structure.Owner, msg.StructId)
    }

    k.SetGridAttributeDecrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, player.Id), structure.ActiveMiningSystemDraw)

    structure.SetMiningSystemStatus("INACTIVE")
    structure.SetMiningSystemActivationBlock(0)
    k.SetStruct(ctx, structure)

	return &types.MsgStructMineDeactivateResponse{Struct: structure}, nil
}
