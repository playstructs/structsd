package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) SquadInvite(goCtx context.Context, msg *types.MsgSquadInvite) (*types.MsgSquadInviteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    // look up transaction creator player object
    txPlayerId := k.GetPlayerIdFromAddress(ctx, msg.Creator)
    if (txPlayerId == 0) {
        return &types.MsgSquadInviteResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Squad management requires Player account but none associated with %s", msg.Creator)
    }
    txPlayer, _ := k.GetPlayer(ctx, txPlayerId)

    // look up target player object
    targetPlayer, targetPlayerFound := k.GetPlayer(ctx, msg.PlayerId)
    if (!targetPlayerFound) {
        return &types.MsgSquadInviteResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Referenced target player (%d) not found", msg.PlayerId)
    }


	// look up destination squad
	squad, squadFound := k.GetSquad(ctx, msg.SquadId)

    if (!squadFound) {
        return &types.MsgSquadInviteResponse{}, sdkerrors.Wrapf(types.ErrSquadNotFound, "Referenced Squad (%d) not found", msg.SquadId)
    }

    if (squad.Id == targetPlayer.SquadId) {
        return &types.MsgSquadInviteResponse{}, sdkerrors.Wrapf(types.ErrSquadPlayerCannotSquadHarder, "Proposed Player (%d) already in the Squad (%d)", msg.PlayerId, msg.SquadId)
    }


	// look up destination guild
	// Can likely be removed
	guild, guildFound := k.GetGuild(ctx, squad.GuildId)

    if (!guildFound) {
        return &types.MsgSquadInviteResponse{}, sdkerrors.Wrapf(types.ErrGuildNotFound, "Referenced Guild (%d) not found", squad.GuildId)
    }


    // Calling address (msg.Creator) must have permissions to perform the Squad Management task
    // AddressPermissionManageSquad
    if (!k.AddressPermissionHasOneOf(ctx, msg.Creator, types.AddressPermissionManageSquad)) {
        return &types.MsgSquadInviteResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageSquad, "Calling Address (%s) must have Squad Management permissions", msg.Creator)
    }

    // Calling player (txPlayer) needs to have certain permissions to complete the task
    // either GuildPermissionSquadUpdate or SquadInvite
    if ((!k.GuildPermissionHasOneOf(ctx, guild.Id, txPlayer.Id, types.GuildPermissionSquadUpdate)) && (!k.SquadPermissionHasOneOf(ctx, squad.Id, txPlayer.Id, types.SquadInvite))) {
        return &types.MsgSquadInviteResponse{}, sdkerrors.Wrapf(types.ErrPermissionSquadLeaderProposal, "Calling player (%d) does not have Squad Update permissions from Guild (%d) or Squad (%d)", txPlayer.Id, guild.Id, squad.Id)
    }

    // Make sure the player is in the Guild
    // Otherwise they cannot be added to a squad
    if (targetPlayer.GuildId != guild.Id) {
        return &types.MsgSquadInviteResponse{}, sdkerrors.Wrapf(types.ErrPermissionSquadCreation, "Proposed player (%d) for Squad must be a member of Guild (%d)", targetPlayer.Id, guild.Id)
    }



    if ((txPlayer.Id == targetPlayer.Id) || (!k.PlayerPermissionHasOneOf(ctx, targetPlayer.Id, txPlayer.Id, types.PlayerPermissionSquad))){
        // At this point, we can just add them to the squad
        // They're permissed enough, no reason to make it a proposal.

        targetPlayer.SetSquad(squad.Id)
        k.SetPlayer(ctx, targetPlayer)

    } else {
        // Create a proposal rather than forcing
        // their ascent to squad leader

        k.SquadAddInvite(ctx, squad, targetPlayer)
    }


	return &types.MsgSquadInviteResponse{}, nil
}
