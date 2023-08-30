package keeper

import (
	"context"
    "strconv"

    //"fmt"

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


    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)
    if ((playerPermissions&types.AddressPermissionPlay) == 0) {
        return &types.MsgStructMineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }

    structure, structureFound := k.GetStruct(ctx, msg.StructId)
    if (!structureFound) {
        structIdString := strconv.FormatUint(msg.StructId, 10)
        return &types.MsgStructMineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrStructNotFound, "Struct (%s) not found", structIdString)
    }

    if (structure.Type != "Mining Rig") {
        structIdString := strconv.FormatUint(msg.StructId, 10)
        return &types.MsgStructMineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrStructMineDeactivate, "This struct (%s) has no mining systems", structIdString)
    }

    if (structure.Status != "ACTIVE") {
        structIdString := strconv.FormatUint(msg.StructId, 10)
        return &types.MsgStructMineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrStructMineDeactivate, "This struct (%s) is not online", structIdString)
    }


    if (structure.MiningSystemStatus == "INACTIVE") {
        structIdString := strconv.FormatUint(msg.StructId, 10)
        return &types.MsgStructMineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrStructMineDeactivate, "This Mining System for struct (%s) is already inactive", structIdString)
    }

    /*
     * Until we let players give out Play permissions, this can't happened
     */
    if (player.Id != structure.Owner) {
       structIdString := strconv.FormatUint(structure.Owner, 10)
       return &types.MsgStructMineDeactivateResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlayerPlay, "For now you can't sudo structs, no permission for action on Struct (%s)", structIdString)
    }

    _, _ = k.PlayerDecrementLoad(ctx, player.Id, structure.ActiveMiningSystemDraw)


    structure.SetMiningSystemStatus("INACTIVE")
    structure.SetMiningSystemActivationBlock(0)
    k.SetStruct(ctx, structure)

	return &types.MsgStructMineDeactivateResponse{Struct: structure}, nil
}
