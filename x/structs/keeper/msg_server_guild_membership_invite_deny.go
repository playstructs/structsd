package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) GuildMembershipInviteDeny(goCtx context.Context, msg *types.MsgGuildMembershipInviteDeny) (*types.MsgGuildMembershipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

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

    if (player.Id != msg.PlayerId) {
        if (!k.PermissionHasOneOf(ctx, GetObjectPermissionIDBytes(msg.PlayerId, player.Id), types.PermissionAssociations)) {
            return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrPermissionGuildRegister, "Calling player (%s) has no Player Association permissions with the Guild (%s) ", player.Id, msg.GuildId)
        }
    }

    guildMembershipApplication, guildMembershipApplicationFound := k.GetGuildMembershipApplication(ctx, msg.GuildId, msg.PlayerId)
    if (!guildMembershipApplicationFound) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Membership Application not found")
    }

    if (guildMembershipApplication.JoinType != types.GuildJoinType_invite) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Membership Application is incorrect type for invitation denial")
    }

    guildMembershipApplication.RegistrationStatus   = types.RegistrationStatus_denied
    k.ClearGuildMembershipApplication(ctx, guildMembershipApplication)

	return &types.MsgGuildMembershipResponse{GuildMembershipApplication: &guildMembershipApplication}, nil
}
