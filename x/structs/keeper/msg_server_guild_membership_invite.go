package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) GuildMembershipInvite(goCtx context.Context, msg *types.MsgGuildMembershipInvite) (*types.MsgGuildMembershipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	// Look up requesting account
	player := k.UpsertPlayer(ctx, msg.Creator)

    addressPermissionId     := GetAddressPermissionIDBytes(msg.Creator)
    // Make sure the address calling this has Associate permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionAssociations)) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageGuild, "Calling address (%s) has no Guild Management permissions ", msg.Creator)
    }

    if (msg.GuildId == "") {
        msg.GuildId = player.GuildId
    }

	// look up destination guild
	guild, guildFound := k.GetGuild(ctx, msg.GuildId)

    if (!guildFound) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Guild (%s) not found", msg.GuildId)
    }


    // Invitations not needed. Have the player perform a request
    if (guild.JoinInfusionMinimum == 0) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Guild not currently requiring invitation")
    }

    // Does the guild currently allow for invitations?
    if (guild.JoinInfusionMinimumBypassByInvite == types.GuildJoinBypassLevel_closed) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Guild not currently allowing invitations")

    // If the invitations require a permissioned player, check for it
    } else if (guild.JoinInfusionMinimumBypassByInvite == types.GuildJoinBypassLevel_permissioned) {
        if (!k.PermissionHasOneOf(ctx, GetObjectPermissionIDBytes(guild.Id, player.Id), types.PermissionAssociations)) {
            return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrPermissionGuildRegister, "Calling player (%s) has no Player Association permissions with the Guild (%s) ", player.Id, guild.Id)
        }

    // Otherwise, just make sure they're in the guild
    } else if (guild.JoinInfusionMinimumBypassByInvite == types.GuildJoinBypassLevel_member) {
        if (player.GuildId != guild.Id) {
            return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Calling player (%s) must be a member of Guild (%s) to invite others", player.Id, guild.Id)
        }
    }


    guildMembershipApplication, guildMembershipApplicationFound := k.GetGuildMembershipApplication(ctx, msg.GuildId, msg.PlayerId)
    if (guildMembershipApplicationFound) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Membership Application already pending")
    }

    /*
     * We're either going to load up the substation provided as an
     * override, or we're going to default to using the guild entry substation
     */
    if (msg.SubstationId != "") {
        // look up destination substation
        substation, substationFound := k.GetSubstation(ctx, msg.SubstationId)

        // Does the substation provided for override exist?
        if (!substationFound) {
            return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Substation (%s) not found", msg.SubstationId)
        }

        // Since the Guild Entry Substation is being overridden, let's make
        // sure the player actually have authority over this substation
        substationObjectPermissionId := GetObjectPermissionIDBytes(substation.Id, player.Id)
        if (!k.PermissionHasOneOf(ctx, substationObjectPermissionId, types.PermissionGrid)) {
            return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrPermissionGuildRegister, "Calling player (%d) has no Player Connect permissions on Substation (%s) used as override", player.Id, substation.Id)
        }

        guildMembershipApplication.SubstationId = substation.Id
    }

    guildMembershipApplication.Proposer             = player.Id
    guildMembershipApplication.PlayerId             = msg.PlayerId
    guildMembershipApplication.GuildId              = guild.Id
    guildMembershipApplication.JoinType             = types.GuildJoinType_invite
    guildMembershipApplication.RegistrationStatus   = types.RegistrationStatus_proposed

    k.SetGuildMembershipApplication(ctx, guildMembershipApplication)

	return &types.MsgGuildMembershipResponse{GuildMembershipApplication: &guildMembershipApplication}, nil
}
