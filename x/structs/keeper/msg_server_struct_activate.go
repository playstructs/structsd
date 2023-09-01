package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) StructActivate(goCtx context.Context, msg *types.MsgStructActivate) (*types.MsgStructActivateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    playerId := k.GetPlayerIdFromAddress(ctx, msg.Creator)
    if (playerId == 0) {
        return &types.MsgStructActivateResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Struct mining requires Player account but none associated with %s", msg.Creator)
    }
    player, _ := k.GetPlayer(ctx, playerId)

    if (!k.SubstationIsOnline(ctx, player.SubstationId)){
        return &types.MsgStructActivateResponse{}, sdkerrors.Wrapf(types.ErrSubstationOffline, "The players substation (%d) is offline ",player.SubstationId)
    }

    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)
    if ((playerPermissions&types.AddressPermissionPlay) == 0) {
        return &types.MsgStructActivateResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlay, "Calling address (%s) has no play permissions ", msg.Creator)
    }

    structure, structureFound := k.GetStruct(ctx, msg.StructId)
    if (!structureFound) {
        return &types.MsgStructActivateResponse{}, sdkerrors.Wrapf(types.ErrStructNotFound, "Struct (%d) not found", msg.StructId)
    }

    if (structure.Status == "ACTIVE") {
        return &types.MsgStructActivateResponse{}, sdkerrors.Wrapf(types.ErrStructActivate, "This struct (%d) is already online", msg.StructId)
    }


    /*
     * Until we let players give out Play permissions, this can't happened
     */
    if (player.Id != structure.Owner) {
       return &types.MsgStructActivateResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlayerPlay, "For now you can't sudo structs, no permission for action on Struct (%d)", structure.Owner)
    }

     // Try to bring online if there is room in the energy cap
    _, err = k.PlayerIncrementLoad(ctx, player, structure.PassiveDraw)
    if (err != nil) {
        return &types.MsgStructActivateResponse{}, sdkerrors.Wrapf(types.ErrStructActivate, "Could not bring Struct %d online, player %d does not have enough power",structure.Id, player.Id)
    }

    // Reset difficulty block
    structure.SetStatus("ACTIVE")
    k.SetStruct(ctx, structure)

	return &types.MsgStructActivateResponse{Struct: structure}, nil
}
