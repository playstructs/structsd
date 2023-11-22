package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) SquadLeaderProposal(goCtx context.Context, msg *types.MsgSquadLeaderProposal) (*types.MsgSquadLeaderProposalResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    // look up transaction creator player object
    txPlayerId := k.GetPlayerIdFromAddress(ctx, msg.Creator)
    if (txPlayerId == 0) {
        return &types.MsgSquadLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Squad creation requires Player account but none associated with %s", msg.Creator)
    }
    txPlayer, _ := k.GetPlayer(ctx, txPlayerId)

    // look up squad leader player object
    leaderPlayer, _ := k.GetPlayer(ctx, msg.Leader)


	// look up destination squad
	squad, squadFound := k.GetSquad(ctx, msg.SquadId)

    if (!squadFound) {
        return &types.MsgSquadLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrSquadNotFound, "Referenced Squad (%d) not found", msg.SquadId)
    }

    if (squad.Leader == leaderPlayer.Id) {
        return &types.MsgSquadLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrSquadLeaderProposalPlayerCannotLeadHarder, "Proposed Player (%d) already leader of Squad (%d)", msg.Leader, msg.SquadId)
    }


	// look up destination guild
	guild, guildFound := k.GetGuild(ctx, squad.GuildId)

    if (!guildFound) {
        return &types.MsgSquadLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrGuildNotFound, "Referenced Guild (%d) not found", squad.GuildId)
    }


    // Calling address (msg.Creator) must have permissions to perform the Squad Management task
    // AddressPermissionManageSquad
    if (!k.AddressPermissionHasOneOf(ctx, msg.Creator, types.AddressPermissionManageSquad)) {
        return &types.MsgSquadLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageSquad, "Calling Address (%s) must have Squad Management permissions", msg.Creator)
    }

    // Calling player (txPlayer) needs to have certain permissions to complete the task
    // either GuildPermissionSquadUpdate or SquadPermissionUpdateLeader
    if ((!k.GuildPermissionHasOneOf(ctx, guild.Id, txPlayer.Id, types.GuildPermissionSquadUpdate)) && (!k.SquadPermissionHasOneOf(ctx, squad.Id, txPlayer.Id, types.SquadPermissionUpdateLeader))) {
        return &types.MsgSquadLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrPermissionSquadLeaderProposal, "Calling player (%d) does not have Squad Update permissions from Guild (%d) or Squad (%d)", txPlayer.Id, guild.Id, squad.Id)
    }

    // Make sure the player is in the Guild
    // Otherwise they cannot be added to a squad, especially as leader
    if (leaderPlayer.GuildId != guild.Id) {
        return &types.MsgSquadLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrPermissionSquadCreation, "Proposed Leader player (%d) for Squad must be a member of Guild (%d)", leaderPlayer.Id, guild.Id)
    }


    if (leaderPlayer.SquadId > 0) {
        // Check their old squad
        // We know from a previous check that they're
        // not the squad leader of this squad already.
        oldSquad, _ := k.GetSquad(ctx, leaderPlayer.SquadId)

        if (oldSquad.Leader == leaderPlayer.Id) {
            return &types.MsgSquadLeaderProposalResponse{}, sdkerrors.Wrapf(types.ErrSquadLeaderProposalPlayerIneligible, "Proposed Leader Player (%d) is already a Squad (%d) Leader ", leaderPlayer.Id, oldSquad.Leader)
        }

    }

    if ((txPlayer.Id == leaderPlayer.Id) || (!k.PlayerPermissionHasOneOf(ctx, leaderPlayer.Id, txPlayer.Id, types.PlayerPermissionSquad))){
        // At this point, we can just appoint them as leader of the squad
        // They're permissed enough, no reason to make it a proposal.

        squad.SetLeader(leaderPlayer.Id)
        k.SetSquad(ctx, squad)

        leaderPlayer.SetSquad(squad.Id)
        k.SetPlayer(ctx, leaderPlayer)

        k.SquadPermissionAdd(ctx, squad.Id, leaderPlayer.Id, types.SquadPermissionAll)

    } else {
        // Create a proposal rather than forcing
        // their ascent to squad leader

        k.SquadSetLeaderProposalRequest(ctx, squad, leaderPlayer)
    }


	return &types.MsgSquadLeaderProposalResponse{}, nil
}
