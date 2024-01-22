package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) SquadApproveJoinRequest(goCtx context.Context, msg *types.MsgSquadApproveJoinRequest) (*types.MsgSquadApproveJoinRequestResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    txPlayer, txPlayerFound := k.GetPlayer(ctx, k.GetPlayerIdFromAddress(ctx, msg.Creator))


    if (!txPlayerFound) {
        return &types.MsgSquadApproveJoinRequestResponse{}, sdkerrors.Wrapf(types.ErrPlayerNotFound, "Could not perform squad action with non-player address (%s)", msg.Creator)
    }

    squad, squadFound := k.GetSquad(ctx, msg.SquadId)
    if (!squadFound) {
        return &types.MsgSquadApproveJoinRequestResponse{}, sdkerrors.Wrapf(types.ErrSquadNotFound, "Referenced Squad (%d) not found", squad.Id)
    }

    targetPlayer, targetPlayerFound := k.GetPlayer(ctx, msg.PlayerId)
    if (!targetPlayerFound) {
        return &types.MsgSquadApproveJoinRequestResponse{}, sdkerrors.Wrapf(types.ErrPlayerNotFound, "Proposed player (%d) does not exist", msg.PlayerId)
    }

    if (targetPlayer.GuildId != squad.GuildId) {
        return &types.MsgSquadApproveJoinRequestResponse{}, sdkerrors.Wrapf(types.ErrSquadPlayerGuildMismatch, "Squad (%d) cannot have a player (%d) from a different Guild", squad.Id, targetPlayer.Id, targetPlayer.GuildId)
    }

    // Check Player Permissions
    if (txPlayer.Id != targetPlayer.Id) {
        // Does the calling player have sudo on the player?
        if (!k.PlayerPermissionHasOneOf(ctx, leaderPlayer.Id, txPlayer.Id, types.PlayerPermissionSquad)) {
            return &types.MsgSquadApproveJoinRequestResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageSquad, "Calling player (%d) has no Sudo Squad permissions on Player (%d)", txPlayer.Id, msg.PlayerId)
        }
    }

    // Check Address Permissions
    // AddressPermissionManageSquad
    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)
    if ((playerPermissions&types.AddressPermissionManageSquad) == 0) {
        return &types.MsgSquadApproveJoinRequestResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageSquad, "Calling address (%s) has no Squad permissions ", msg.Creator)
    }

    // Calling player (txPlayer) needs to have certain permissions to complete the task
    // either GuildPermissionSquadUpdate or SquadInvite
    if ((!k.GuildPermissionHasOneOf(ctx, squad.GuildID, txPlayer.Id, types.GuildPermissionSquadUpdate)) && (!k.SquadPermissionHasOneOf(ctx, squad.Id, txPlayer.Id, types.SquadInvite))) {
        return &types.MsgSquadApproveJoinRequestResponse{}, sdkerrors.Wrapf(types.ErrPermissionSquadLeaderProposal, "Calling player (%d) does not have Squad Update permissions from Guild (%d) or Squad (%d)", txPlayer.Id, squad.GuildID, squad.Id)
    }

    // this could probably be cleaned up a bit
    // the first variable is always SquadJoinRequestStatus_Pending atm, as long as the second variable is true
    _, joinRequestFound := k.SquadGetJoinRequest(ctx, squad, targetPlayer)
    if (!joinRequestFound) {
        return &types.MsgSquadApproveJoinRequestResponse{}, sdkerrors.Wrapf(types.ErrSquadLeaderProposalNotFound, "Squad (%d) has no join request for Player (%d)", squad.Id, targetPlayer.Id)
    }


    // Check to see if the player is part of a squad already
    if ((targetPlayer.SquadId > 0) && (targetPlayer.SquadId != squad.Id)) {
        oldSquad, _ := k.GetSquad(ctx, targetPlayer.SquadId)
        if (oldSquad.Leader == targetPlayer.Id) {
            return &types.MsgSquadApproveJoinRequestResponse{}, sdkerrors.Wrapf(types.ErrSquadLeaderProposalPlayerIneligible, "Player (%d) already a squad leader (%d). Demote first", targetPlayer.Id, targetPlayer.SquadId)
        }
    }

    // Confirm the Results
    if (msg.Approve) {
        k.SquadApproveJoinRequest(ctx, squad, targetPlayer)
    } else {
        k.SquadDenyJoinRequest(ctx, squad, targetPlayer)
    }

	return &types.MsgSquadApproveJoinRequestResponse{}, nil
}
