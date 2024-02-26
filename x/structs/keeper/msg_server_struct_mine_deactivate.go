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

    playerId := k.GetPlayerIdFromAddress(ctx, msg.Creator)
    if (playerId == 0) {
        return &types.MsgStructMineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct build initialization requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayer(ctx, playerId)

    if (!k.SubstationIsOnline(ctx, player.SubstationId)){
        return &types.MsgStructMineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrSubstationOffline, "The players substation (%d) is offline ",player.SubstationId)
    }

    addressPermissionId := GetAddressPermissionIDBytes(msg.Creator)
    // Make sure the address calling this has Play permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.Permission(types.AddressPermissionPlay))) {
        return &types.MsgStructActivateResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }

    structure, structureFound := k.GetStruct(ctx, msg.StructId)
    if (!structureFound) {
        return &types.MsgStructMineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrStructNotFound, "Struct (%d) not found", msg.StructId)
    }

    if (structure.Type != "Mining Rig") {
        return &types.MsgStructMineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrStructMineDeactivate, "This struct (%d) has no mining systems", msg.StructId)
    }

    if (structure.Status != "ACTIVE") {
        return &types.MsgStructMineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrStructMineDeactivate, "This struct (%d) is not online", msg.StructId)
    }


    if (structure.MiningSystemStatus == "INACTIVE") {
        return &types.MsgStructMineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrStructMineDeactivate, "This Mining System for struct (%d) is already inactive", msg.StructId)
    }

    /*
     * Until we let players give out Play permissions, this can't happened
     */
    if (player.Id != structure.Owner) {
       return &types.MsgStructMineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlayerPlay, "For now you (%d) can't sudo structs, no permission for action on Struct (%d)", structure.Owner, msg.StructId)
    }

    _, _ = k.PlayerDecrementLoad(ctx, player.Id, structure.ActiveMiningSystemDraw)


    structure.SetMiningSystemStatus("INACTIVE")
    structure.SetMiningSystemActivationBlock(0)
    k.SetStruct(ctx, structure)

	return &types.MsgStructMineDeactivateResponse{Struct: structure}, nil
}
