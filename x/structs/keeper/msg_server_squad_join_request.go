package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) SquadJoinRequest(goCtx context.Context, msg *types.MsgSquadJoinRequest) (*types.MsgSquadJoinRequestResponse, error) {
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

    if (squad.Id == targetPlayer.SquadId) {
        return &types.MsgSquadJoinRequestResponse{}, sdkerrors.Wrapf(types.ErrSquadPlayerCannotSquadHarder, "Proposed Player (%d) already in the Squad (%d)", msg.PlayerId, msg.SquadId)
    }

    // Check to see if the player is part of a squad already
    if (targetPlayer.SquadId > 0) {
        oldSquad, _ := k.GetSquad(ctx, targetPlayer.SquadId)
        if (oldSquad.Leader == targetPlayer.Id) {
            return &types.MsgSquadApproveLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrSquadLeaderProposalPlayerIneligible, "Player (%d) already Squad Leader of (%d). Demote first", targetPlayer.Id, targetPlayer.SquadId)
        }
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



    // Make sure the player is in the Guild
    // Otherwise they cannot be added to a squad
    if (targetPlayer.GuildId != squad.GuildId) {
        return &types.MsgSquadJoinRequestResponse{}, sdkerrors.Wrapf(types.ErrPermissionSquadCreation, "Proposed player (%d) for Squad must be a member of Guild (%d)", targetPlayer.Id, squad.GuildId)
    }

    k.SquadSetJoinRequest(ctx, squad, targetPlayer)

	return &types.MsgSquadJoinRequestResponse{}, nil
}
