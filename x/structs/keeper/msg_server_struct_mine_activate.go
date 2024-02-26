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

    playerId := k.GetPlayerIdFromAddress(ctx, msg.Creator)
    if (playerId == 0) {
        return &types.MsgStructMineActivateResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct build initialization requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayer(ctx, playerId)

    if (!k.SubstationIsOnline(ctx, player.SubstationId)){
        return &types.MsgStructMineActivateResponse{}, sdkerrors.Wrapf(types.ErrSubstationOffline, "The players substation (%d) is offline ",player.SubstationId)
    }

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    // Make sure the address calling this has Play permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.Permission(types.AddressPermissionPlay))) {
        return &types.MsgStructActivateResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }

    structure, structureFound := k.GetStruct(ctx, msg.StructId)
    if (!structureFound) {
        return &types.MsgStructMineActivateResponse{}, sdkerrors.Wrapf(types.ErrStructNotFound, "Struct (%d) not found", msg.StructId)
    }

    if (structure.Type != "Mining Rig") {
        return &types.MsgStructMineActivateResponse{}, sdkerrors.Wrapf(types.ErrStructMineActivate, "This struct (%d) has no mining systems", msg.StructId)
    }

    if (structure.Status != "ACTIVE") {
        return &types.MsgStructMineActivateResponse{}, sdkerrors.Wrapf(types.ErrStructMineActivate, "This struct (%d) is not online", msg.StructId)
    }


    if (structure.MiningSystemStatus != "INACTIVE") {
        return &types.MsgStructMineActivateResponse{}, sdkerrors.Wrapf(types.ErrStructMineActivate, "This Mining System for struct (%d) is already active", msg.StructId)
    }

    /*
     * Until we let players give out Play permissions, this can't happened
     */
    if (player.Id != structure.Owner) {
       return &types.MsgStructMineActivateResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlayerPlay, "For now you can't sudo structs, no permission for action on Struct (%d)", structure.Id)
    }

    _, err = k.PlayerIncrementLoad(ctx, player, structure.ActiveMiningSystemDraw)

    if (err != nil) {
        return &types.MsgStructMineActivateResponse{}, sdkerrors.Wrapf(types.ErrStructMineActivate, "Not enough power to bring Mining System online for Struct (%s)", msg.StructId)
    }

    structure.SetMiningSystemStatus("ACTIVE")
    structure.SetMiningSystemActivationBlock(uint64(ctx.BlockHeight()))
    k.SetStruct(ctx, structure)

	return &types.MsgStructMineActivateResponse{Struct: structure}, nil
}
