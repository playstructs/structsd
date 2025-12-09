package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) GuildMembershipKick(goCtx context.Context, msg *types.MsgGuildMembershipKick) (*types.MsgGuildMembershipResponse, error) {
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

    if (!k.PermissionHasOneOf(ctx, GetObjectPermissionIDBytes(guild.Id, player.Id), types.PermissionAssociations)) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrPermissionGuildRegister, "Calling player (%s) has no Player Association permissions with the Guild (%s) ", player.Id, guild.Id)
    }

    guildMembershipApplication, guildMembershipApplicationFound := k.GetGuildMembershipApplication(ctx, msg.GuildId, msg.PlayerId)

    guildMembershipApplication.RegistrationStatus   = types.RegistrationStatus_revoked
    if (!guildMembershipApplicationFound) {
        guildMembershipApplication.Proposer             = player.Id
        guildMembershipApplication.PlayerId             = msg.PlayerId
        guildMembershipApplication.GuildId              = guild.Id
        guildMembershipApplication.JoinType             = types.GuildJoinType_direct
    }

    // Look up requesting account
    targetPlayer := k.UpsertPlayer(ctx, msg.PlayerId)
    targetPlayer.GuildId = ""

    targetPlayerUpdated := false
    if (player.SubstationId != "") {

        substation, substationFound := k.GetSubstation(ctx, targetPlayer.SubstationId)

        if (substationFound) {
            if (substation.Owner != targetPlayer.Id) {
                if (!k.PermissionHasOneOf(ctx, GetObjectPermissionIDBytes(substation.Id, player.Id), types.PermissionGrid)) {
                     k.SubstationDisconnectPlayer(ctx, targetPlayer)
                     targetPlayerUpdated = true
                }
            }
        }
    }

    if (!targetPlayerUpdated) {
        k.SetPlayer(ctx, targetPlayer)
    }

    // TODO (Possibly) - One thing we're not doing here yet is clearing out any
    // permissions related to the previous guild. This could get messy so doing it
    // manually might be best. That said, perhaps it could be a configuration option
    // for guilds to define what happens on leave.

    k.ClearGuildMembershipApplication(ctx, guildMembershipApplication)

	return &types.MsgGuildMembershipResponse{GuildMembershipApplication: &guildMembershipApplication}, nil
}
