package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) GuildMembershipRequest(goCtx context.Context, msg *types.MsgGuildMembershipRequest) (*types.MsgGuildMembershipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	// Look up requesting account
	player := k.UpsertPlayer(ctx, msg.Creator)

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

    // Requests not needed. Join Automatically
    if (guild.JoinInfusionMinimum == 0) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Guild not currently requiring requests")
    }

    // Does the guild currently allow for requests?
    if (guild.JoinInfusionMinimumBypassByRequest == types.GuildJoinBypassLevel_closed) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Guild not currently allowing requests")
    }


    if (player.Id != msg.PlayerId) {
        if (!k.PermissionHasOneOf(ctx, GetObjectPermissionIDBytes(msg.PlayerId, player.Id), types.PermissionAssociations)) {
            return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrPermissionGuildRegister, "Calling player (%s) has no Player Registration permissions ", player.Id)
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
            return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrPermissionGuildRegister, "Calling player (%s) has no Player Connect permissions on Substation (%s) used as override", player.Id, substation.Id)
        }

        guildMembershipApplication.SubstationId = substation.Id
    }

    guildMembershipApplication.Proposer             = player.Id
    guildMembershipApplication.PlayerId             = msg.PlayerId
    guildMembershipApplication.GuildId              = guild.Id
    guildMembershipApplication.JoinType             = types.GuildJoinType_request
    guildMembershipApplication.RegistrationStatus   = types.RegistrationStatus_proposed

    k.SetGuildMembershipApplication(ctx, guildMembershipApplication)

	return &types.MsgGuildMembershipResponse{GuildMembershipApplication: &guildMembershipApplication}, nil
}
