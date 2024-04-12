package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) GuildMembershipRequestDeny(goCtx context.Context, msg *types.MsgGuildMembershipRequestDeny) (*types.MsgGuildMembershipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	// Look up requesting account
	player := k.UpsertPlayer(ctx, msg.Creator, true)

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

    // Does the guild currently allow for requests?
    if (guild.JoinInfusionMinimumBypassByRequest == types.GuildJoinBypassLevel_closed) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Guild not currently allowing requests")

    } else if (guild.JoinInfusionMinimumBypassByRequest == types.GuildJoinBypassLevel_permissioned) {
        if (!k.PermissionHasOneOf(ctx, GetObjectPermissionIDBytes(guild.Id, player.Id), types.PermissionAssociations)) {
            return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrPermissionGuildRegister, "Calling player (%s) has no Player Association permissions with the Guild (%s) ", player.Id, guild.Id)
        }

    // Otherwise, just make sure they're in the guild
    } else if (guild.JoinInfusionMinimumBypassByRequest == types.GuildJoinBypassLevel_member) {
        if (player.GuildId != guild.Id) {
            return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Calling player (%s) must be a member of Guild (%s) to invite others", player.Id, guild.Id)
        }
    }

    guildMembershipApplication, guildMembershipApplicationFound := k.GetGuildMembershipApplication(ctx, msg.GuildId, msg.PlayerId)
    if (!guildMembershipApplicationFound) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Membership Application not found")
    }

    if (guildMembershipApplication.JoinType != types.GuildJoinType_request) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Membership Application is incorrect type for request rejection")
    }

    guildMembershipApplication.RegistrationStatus   = types.RegistrationStatus_denied
    k.ClearGuildMembershipApplication(ctx, guildMembershipApplication)

	return &types.MsgGuildMembershipResponse{GuildMembershipApplication: &guildMembershipApplication}, nil
}
