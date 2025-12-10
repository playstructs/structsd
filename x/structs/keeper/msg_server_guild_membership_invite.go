package keeper

import (
	"context"

	"structs/x/structs/types"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) GuildMembershipInvite(goCtx context.Context, msg *types.MsgGuildMembershipInvite) (*types.MsgGuildMembershipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Add an Active Address record to the
	// indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayer, err := k.GetPlayerCacheFromAddress(ctx, msg.Creator)
    if err != nil {
        return &types.MsgGuildMembershipResponse{}, err
    }

    // Use cache permission methods
    callingPlayerPermissionError := callingPlayer.CanBeAdministratedBy(msg.Creator, types.PermissionAssociations)
    if callingPlayerPermissionError != nil {
        return &types.MsgGuildMembershipResponse{}, callingPlayerPermissionError
    }

    // targetPlayer
    _, err = k.GetPlayerCacheFromId(ctx, msg.PlayerId)
    if err != nil {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Player (%s) not found", msg.PlayerId)
    }

	if msg.GuildId == "" {
		msg.GuildId = callingPlayer.GetGuildId()
	}

    guild := k.GetGuildCacheFromId(ctx, msg.GuildId)
    if !guild.LoadGuild() {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Guild (%s) not found", msg.GuildId)
    }

    // For guild permissions
    guildPermissionError := guild.CanAdministrateMembers(&callingPlayer)
    if guildPermissionError != nil {
        return &types.MsgGuildMembershipResponse{}, guildPermissionError
    }

	guildMembershipApplication := k.GetGuildMembershipApplicationCache(ctx, callingPlayer.GetPlayerId(), msg.GuildId, msg.PlayerId)

	if guildMembershipApplication.IsGuildMembershipApplicationFound() {
		return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Membership Application already pending")
	}

	guildMembershipApplication.SetJoinType(types.GuildJoinType_invite)

	/*
	 * We're either going to load up the substation provided as an
	 * override, or we're going to default to using the guild entry substation
	 */
	if msg.SubstationId != "" {

        substation := k.GetSubstationCacheFromId(ctx, msg.SubstationId)
        if !substation.LoadSubstation() {
            return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Substation (%s) not found", msg.SubstationId)
        }

        substationPermissionError := substation.CanManagePlayerConnections(&callingPlayer)
        if substationPermissionError != nil {
            return &types.MsgGuildMembershipResponse{}, substationPermissionError
        }

		guildMembershipApplication.SetSubstationId(substation.GetSubstationId())
	} else {
	    guildMembershipApplication.SetSubstationId(guild.GetEntrySubstationId())
	}

	guildMembershipApplication.Commit()

	return &types.MsgGuildMembershipResponse{GuildMembershipApplication: &guildMembershipApplication.GuildMembershipApplication}, nil
}
