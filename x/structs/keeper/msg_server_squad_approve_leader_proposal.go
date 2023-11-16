package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) SquadApproveLeaderProposal(goCtx context.Context, msg *types.MsgSquadApproveLeaderProposal) (*types.MsgSquadApproveLeaderProposalResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    txPlayer, txPlayerFound := k.GetPlayer(ctx, k.GetPlayerIdFromAddress(ctx, msg.Creator))


    if (!txPlayerFound) {
        return &types.MsgSquadApproveLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrPlayerNotFound, "Could not perform squad action with non-player address (%s)", msg.Creator)
    }

    squad, squadFound := k.GetSquad(ctx, msg.SquadId)
    if (!squadFound) {
        return &types.MsgSquadApproveLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrSquadNotFound, "Referenced Squad (%d) not found", squad.Id)
    }

    leaderPlayer, leaderPlayerFound := k.GetPlayer(ctx, msg.PlayerId)
    if (!leaderPlayerFound) {
        return &types.MsgSquadApproveLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrPlayerNotFound, "Proposed Leader (%d) does not exist", msg.PlayerId)
    }

    if (leaderPlayer.GuildId != squad.GuildId) {
        return &types.MsgSquadApproveLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrSquadLeaderProposalGuildMismatch, "Squad (%d) cannot have a Leader Player (%d) from a different Guild", squad.Id, leaderPlayer.Id, leaderPlayer.GuildId)
    }

    // Check Player Permissions
    if (txPlayer.Id != leaderPlayer.Id) {
        // Does the calling player have sudo on the leader?
        if (!k.PlayerPermissionHasOneOf(ctx, leaderPlayer.Id, txPlayer.Id, types.PlayerPermissionSquad)) {
            return &types.MsgSquadApproveLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageSquad, "Calling player (%d) has no Sudo Squad permissions on Proposed Leader (%d)", txPlayer.Id, msg.PlayerId)
        }
    }

    // Check Address Permissions
    // AddressPermissionManageSquad
    playerPermissions := k.AddressGetPlayerPermissions(ctx, msg.Creator)
    if ((playerPermissions&types.AddressPermissionManageSquad) == 0) {
        return &types.MsgSquadApproveLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageSquad, "Calling address (%s) has no Squad permissions ", msg.Creator)
    }

    proposalPlayer, proposalFound := k.SquadGetLeaderProposalRequest(ctx, squad)
    if (!proposalFound) {
        return &types.MsgSquadApproveLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrSquadLeaderProposalNotFound, "Squad (%d) has no leader proposal", squad.Id)
    }

    if (proposalPlayer.Id != leaderPlayer.Id) {
        return &types.MsgSquadApproveLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrSquadLeaderProposalPlayerMismatch, "Squad (%d) has no leader proposal for Player (%d)", squad.Id, leaderPlayer.Id)
    }

    // Check to see if the player is part of a squad already
    if ((leaderPlayer.SquadId > 0) && (leaderPlayer.SquadId != squad.Id)) {
        oldSquad, _ := k.GetSquad(ctx, leaderPlayer.SquadId)
        if (oldSquad.Leader == leaderPlayer.Id) {
            return &types.MsgSquadApproveLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrSquadLeaderProposalPlayerIneligible, "Player (%d) already Squad Leader of (%d). Demote first", leaderPlayer.Id, leaderPlayer.SquadId)
        }
    }

    // Confirm the Results
    if (msg.Approve) {
        k.SquadApproveLeaderProposalRequest(ctx, squad, leaderPlayer)
    } else {
        k.SquadDenyLeaderProposalRequest(ctx, squad, leaderPlayer)
    }

	return &types.MsgSquadApproveLeaderProposalResponse{}, nil
}
