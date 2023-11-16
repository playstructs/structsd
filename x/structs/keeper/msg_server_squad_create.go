package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"structs/x/structs/types"
)

func (k msgServer) SquadCreate(goCtx context.Context, msg *types.MsgSquadCreate) (*types.MsgSquadCreateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

    squadLeader := msg.Leader

    // look up transaction creator player object
    txPlayerId := k.GetPlayerIdFromAddress(ctx, msg.Creator)
    if (txPlayerId == 0) {
        return &types.MsgSquadCreateResponse{}, sdkerrors.Wrapf(types.ErrPlayerRequired, "Squad creation requires Player account but none associated with %s", msg.Creator)
    }
    txPlayer, _ := k.GetPlayer(ctx, txPlayerId)

    // look up squad leader player object
    leaderPlayer, _ := k.GetPlayer(ctx, msg.Leader)

	// look up destination guild
	guild, guildFound := k.GetGuild(ctx, msg.GuildId)

    if (!guildFound) {
        return &types.MsgSquadCreateResponse{}, sdkerrors.Wrapf(types.ErrGuildNotFound, "Referenced Guild (%d) not found", guild.Id)
    }


    if (txPlayer.Id == leaderPlayer.Id) {
        if (guild.OpenSquadCreation) {
            if ( (txPlayer.GuildId != guild.Id) && (!k.GuildPermissionHasOneOf(ctx, guild.Id, txPlayer.Id, types.GuildPermissionSquadCreate))) {
                return &types.MsgSquadCreateResponse{}, sdkerrors.Wrapf(types.ErrPermissionSquadCreation, "Calling player (%d) must be a member of Guild (%d) and does not have Squad Create permissions", txPlayer.Id, guild.Id)
            }

        } else {

           if (!k.GuildPermissionHasOneOf(ctx, guild.Id, txPlayer.Id, types.GuildPermissionSquadCreate)) {
                return &types.MsgSquadCreateResponse{}, sdkerrors.Wrapf(types.ErrPermissionSquadCreation, "Calling player (%d) must have Squad Create permissions in Guild (%d)", txPlayer.Id, guild.Id)
            }
        }


    } else {
        // If creating a squad for another player, the Open Squad Creation is not considered
        if (!k.GuildPermissionHasOneOf(ctx, guild.Id, txPlayer.Id, types.GuildPermissionSquadCreate)) {
            return &types.MsgSquadCreateResponse{}, sdkerrors.Wrapf(types.ErrPermissionSquadCreation, "Calling player (%d) must have Squad Create permissions to create Squad for other guild members", txPlayer.Id, guild.Id)
        }

        // Make sure the player is in the Guild
        // Otherwise they cannot be added to a squad, especially as leader
        if (leaderPlayer.GuildId != guild.Id) {
            return &types.MsgSquadCreateResponse{}, sdkerrors.Wrapf(types.ErrPermissionSquadCreation, "Proposed Leader player (%d) for Squad must be a member of Guild (%d)", leaderPlayer.Id, guild.Id)
        }

        // Make sure the transaction creator player has the authority
        // over the leader player account to manage their squad status
        //
        // If they do not, then create a proposal rather than forcing their
        // ascent to squad leader
        if (!k.PlayerPermissionHasOneOf(ctx, leaderPlayer.Id, txPlayer.Id, types.PlayerPermissionSquad)) {
            squadLeader = 0
        }

    }

    if (msg.EntrySubstationId > 0) {
        // look up destination substation
    	_, substationFound := k.GetSubstation(ctx, guild.EntrySubstationId, true)

        if (!substationFound) {
            return &types.MsgSquadCreateResponse{}, sdkerrors.Wrapf(types.ErrPermissionSquadCreation, "Proposed Entry Substation (%d) for Squad does not exist", msg.EntrySubstationId)
        }

        if (!k.SubstationPermissionHasOneOf(ctx, msg.EntrySubstationId, txPlayer.Id, types.SubstationPermissionRouteSquad)) {
            return &types.MsgSquadCreateResponse{}, sdkerrors.Wrapf(types.ErrPermissionSquadCreation, "Player (%d) does not have permission to route to this Substation (%d)", txPlayer.Id, msg.EntrySubstationId)
        }

    }

    // Compare the current Guild configuration with the proposed Squad
    if (msg.SquadJoinType > guild.SquadJoinTypeMinimum) {
            return &types.MsgSquadCreateResponse{}, sdkerrors.Wrapf(types.ErrPermissionSquadCreation, "Guild (%d) has a Squad Join Type Minimum of (%d) but proposed Squad has a Join Type of (%d)", guild.Id, guild.SquadJoinTypeMinimum, msg.SquadJoinType)
    }


    if (squadLeader > 0) {
        // Is the player already a squad leader?
        if (leaderPlayer.SquadId > 0) {
            // Check their old squad
            oldSquad, _ := k.GetSquad(ctx, leaderPlayer.SquadId)

            if (oldSquad.Leader == leaderPlayer.Id) {
                return &types.MsgSquadCreateResponse{}, sdkerrors.Wrapf(types.ErrPermissionSquadCreation, "Proposed Leader Player (%d) is already a Squad (%d) Leader ", leaderPlayer.Id, oldSquad.Leader)
            }
        }
    }

    // Commit the new Squad to the keeper
    squad := k.AppendSquad(ctx, msg.Creator, msg.GuildId, msg.Leader, msg.SquadJoinType, msg.EntrySubstationId)

    if (squad.Leader > 0 ) {
        // Migrate the new Leader to the Squad
        //
        // Note: This does not change which substation
        //       the player is connected to!
        leaderPlayer.SetSquad(squad.Id)
        k.SetPlayer(ctx, leaderPlayer)

        k.SquadPermissionAdd(ctx, squad.Id, leaderPlayer.Id, types.SquadPermissionAll)
    }

    // If the leader player needed a proposal sent, add this now
    if ((msg.Leader > 0) && (squad.Leader == 0)) {
        k.SquadSetLeaderProposalRequest(ctx, squad, leaderPlayer)
    }

	return &types.MsgSquadCreateResponse{}, nil
}
