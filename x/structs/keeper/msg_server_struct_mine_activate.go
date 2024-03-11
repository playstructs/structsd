package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) StructMineActivate(goCtx context.Context, msg *types.MsgStructMineActivate) (*types.MsgStructMineActivateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    playerIndex := k.GetPlayerIndexFromAddress(ctx, msg.Creator)
    if (playerIndex == 0) {
        return &types.MsgStructMineActivateResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct build initialization requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayerFromIndex(ctx, playerIndex, true)

    if (!player.IsOnline()){
        return &types.MsgStructMineActivateResponse{}, sdkerrors.Wrapf(types.ErrGridMalfunction, "The player (%s) is offline ",player.Id)
    }

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    // Make sure the address calling this has Play permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionPlay)) {
        return &types.MsgStructMineActivateResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }

    structure, structureFound := k.GetStruct(ctx, msg.StructId)
    if (!structureFound) {
        return &types.MsgStructMineActivateResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Struct (%s) not found", msg.StructId)
    }

    if (structure.Type != "Mining Rig") {
        return &types.MsgStructMineActivateResponse{}, sdkerrors.Wrapf(types.ErrStructMineActivate, "This struct (%s) has no mining systems", msg.StructId)
    }

    if (structure.Status != "ACTIVE") {
        return &types.MsgStructMineActivateResponse{}, sdkerrors.Wrapf(types.ErrStructMineActivate, "This struct (%s) is not online", msg.StructId)
    }


    if (structure.MiningSystemStatus != "INACTIVE") {
        return &types.MsgStructMineActivateResponse{}, sdkerrors.Wrapf(types.ErrStructMineActivate, "This Mining System for struct (%s) is already active", msg.StructId)
    }

    /*
     * Until we let players give out Play permissions, this can't happened
     */
    if (player.Id != structure.Owner) {
       return &types.MsgStructMineActivateResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlayerPlay, "For now you can't sudo structs, no permission for action on Struct (%s)", structure.Id)
    }

    // Try to bring online if there is room in the energy cap
    if (!player.CanSupportNewLoad(structure.ActiveMiningSystemDraw)) {
        return &types.MsgStructMineActivateResponse{}, sdkerrors.Wrapf(types.ErrStructMineActivate, "Not enough power to bring Mining System online for Struct (%s)", msg.StructId)
    }

    k.SetGridAttributeIncrement(ctx, GetGridAttributeIDByObjectId(types.GridAttributeType_structsLoad, player.Id), structure.ActiveMiningSystemDraw)

    structure.SetMiningSystemStatus("ACTIVE")
    structure.SetMiningSystemActivationBlock(uint64(ctx.BlockHeight()))
    k.SetStruct(ctx, structure)

	return &types.MsgStructMineActivateResponse{Struct: structure}, nil
}
