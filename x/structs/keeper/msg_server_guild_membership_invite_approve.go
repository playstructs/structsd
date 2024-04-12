package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) GuildMembershipInviteApprove(goCtx context.Context, msg *types.MsgGuildMembershipInviteApprove) (*types.MsgGuildMembershipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	// Look up requesting account
	player := k.UpsertPlayer(ctx, msg.Creator, true)

	if (msg.PlayerId == "") {
	    msg.PlayerId = player.Id
	}

    addressPermissionId     := GetAddressPermissionIDBytes(msg.Creator)
    // Make sure the address calling this has Associate permissions
    if (!k.PermissionHasOneOf(ctx, addressPermissionId, types.PermissionAssociations)) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrPermissionManageGuild, "Calling address (%s) has no Guild Management permissions ", msg.Creator)
    }

	// look up destination guild
	guild, guildFound := k.GetGuild(ctx, msg.GuildId)

    if (!guildFound) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Guild (%s) not found", msg.GuildId)
    }


    if (player.Id != msg.PlayerId) {
        if (!k.PermissionHasOneOf(ctx, GetObjectPermissionIDBytes(msg.PlayerId, player.Id), types.PermissionAssociations)) {
            return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrPermissionGuildRegister, "Calling player (%s) has no Player Association permissions with the Guild (%s) ", player.Id, guild.Id)
        }
    }

    guildMembershipApplication, guildMembershipApplicationFound := k.GetGuildMembershipApplication(ctx, msg.GuildId, msg.PlayerId)
    if (!guildMembershipApplicationFound) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Membership Application not found")
    }

    if (guildMembershipApplication.JoinType != types.GuildJoinType_invite) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Membership Application is incorrect type for invitation approval")
    }

    /*
     * We're either going to load up the substation provided as an
     * override, or we're going to default to using the guild entry substation
     */

    var substation types.Substation
    var substationFound bool

    if (msg.SubstationId != "") {
        // look up destination substation
        substation, substationFound = k.GetSubstation(ctx, msg.SubstationId, true)

        // Does the substation provided for override exist?
        if (!substationFound) {
            return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Substation (%s) not found", msg.SubstationId)
        }

        // Since the Guild Entry Substation is being overridden, let's make
        // sure the player actually have authority over this substation
        substationObjectPermissionId := GetObjectPermissionIDBytes(substation.Id, player.Id)
        if (!k.PermissionHasOneOf(ctx, substationObjectPermissionId, types.PermissionGrid)) {
            return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrPermissionGuildRegister, "Calling player (%s) has no Player Connect permissions on Substation (%s) used as override", player.Id, substation.Id)
        }

        guildMembershipApplication.SubstationId = substation.Id

    } else {
        if (guildMembershipApplication.SubstationId == "") {
            guildMembershipApplication.SubstationId = guild.EntrySubstationId
        }

        substation, substationFound = k.GetSubstation(ctx, guildMembershipApplication.SubstationId, true)
    }

    // Look up requesting account
    targetPlayer := k.UpsertPlayer(ctx, msg.Creator, true)
    targetPlayer.GuildId = msg.GuildId
    k.SubstationConnectPlayer(ctx, substation, targetPlayer)

    // TODO (Possibly) - One thing we're not doing here yet is clearing out any
    // permissions related to the previous guild. This could get messy so doing it
    // manually might be best. That said, perhaps it could be a configuration option
    // for guilds to define what happens on leave.

    guildMembershipApplication.RegistrationStatus   = types.RegistrationStatus_approved
    k.ClearGuildMembershipApplication(ctx, guildMembershipApplication)

	return &types.MsgGuildMembershipResponse{GuildMembershipApplication: &guildMembershipApplication}, nil
}
