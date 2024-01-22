package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) SquadDeleteJoinRequest(goCtx context.Context, msg *types.MsgSquadDeleteJoinRequest) (*types.MsgSquadDeleteJoinRequestResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    // look up transaction creator player object
    txPlayerId := k.GetPlayerIdFromAddress(ctx, msg.Creator)
    if (txPlayerId == 0) {
        return &types.MsgSquadJoinRequestResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Squad management requires Player account but none associated with %s", msg.Creator)
    }
    txPlayer, _ := k.GetPlayer(ctx, txPlayerId)

    // look up target player object
    targetPlayer, targetPlayerFound := k.GetPlayer(ctx, msg.PlayerId)
    if (!targetPlayerFound) {
        return &types.MsgSquadJoinRequestResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Referenced target player (%d) not found", msg.PlayerId)
    }


	// look up destination squad
	squad, squadFound := k.GetSquad(ctx, msg.SquadId)

    if (!squadFound) {
        return &types.MsgSquadJoinRequestResponse{}, sdkerrors.Wrapf(types.ErrSquadNotFound, "Referenced Squad (%d) not found", msg.SquadId)
    }


    // Calling address (msg.Creator) must have permissions to perform the Squad Management task
    // AddressPermissionManageSquad
    if (!k.AddressPermissionHasOneOf(ctx, msg.Creator, types.AddressPermissionManageSquad)) {
        return &types.MsgSquadJoinRequestResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageSquad, "Calling Address (%s) must have Squad Management permissions", msg.Creator)
    }

    if (txPlayer.Id != msg.PlayerId) {
        // check that the calling player has target player permissions
        if (!k.PlayerPermissionHasOneOf(ctx, msg.PlayerId, txPlayer.Id, types.PlayerPermissionSquad)) {
           return &types.MsgSquadJoinRequestResponse{}, sdkerrors.Wrapf(types.ErrPermissionPlayerSquad, "Calling player (%d) has no Player Squad permissions ", txPlayer.Id)
        }
    }

    k.SquadDeleteJoinRequest(ctx, squad, targetPlayer)

	return &types.MsgSquadJoinRequestResponse{}, nil
}
