package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
)

func (k msgServer) GuildMembershipJoin(goCtx context.Context, msg *types.MsgGuildMembershipJoin) (*types.MsgGuildMembershipResponse, error) {
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
    if (guildMembershipApplicationFound) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Membership Application already pending. Deny request or invitation first")
    }

    if (guild.JoinInfusionMinimum != 0) {
        var currentFuel uint64

        /* We're going to iterate through all the infusion records
         * that were provided in the message, checking to make sure
         * that they collectively meet the infusion minimum (as defined
         * by the guild), and that the infusion is actually relevant.
         *
         * Infusion is...
         * - A valid infusion record
         * - Owned by the player
         * - Points to a Reactor
         * - The Destination Reactor is part of the Guild
         */
        for _, infusionId := range msg.InfusionId {

            infusion, infusionFound := k.GetInfusionByID(ctx, infusionId)
            if (!infusionFound) {
                return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Infusion (%s) not found", infusionId)
            }

            if (infusion.PlayerId != msg.PlayerId) {
                return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Infusion (%s) does not belong to player (%s)", infusionId, msg.PlayerId)
            }

            if (infusion.DestinationType != types.ObjectType_reactor) {
                return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Only Reactor infusions allowed, Infusion (%s) unacceptable", infusionId)
            }

            reactor, reactorFound := k.GetReactor(ctx, infusion.DestinationId, false)
            if (!reactorFound) {
                return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Somehow this reactor (%s) doesn't exist, you should tell an adult",infusion.DestinationId)
            }

            if (reactor.GuildId != msg.GuildId) {
                return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Infusion (%s) is for a Reactor (%s) of a different Guild (%s)", infusionId, reactor.Id, reactor.GuildId)
            }

            currentFuel = currentFuel + infusion.Fuel

            /* This is an expensive process, so fail fast.
             *
             * This could mean an infusion is provided in the message that doesn't
             * meet these requirements but at this point, we've met the infusion
             * minimum of the guild so that doesn't actually matter.
             */
            if (currentFuel > guild.JoinInfusionMinimum) { break }
        }

        if (currentFuel < guild.JoinInfusionMinimum) {
            return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Join Infusion Minimum not met")
        }
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
        guildMembershipApplication.SubstationId = guild.EntrySubstationId
        substation, substationFound = k.GetSubstation(ctx, guildMembershipApplication.SubstationId, true)
    }

    guildMembershipApplication.Proposer             = player.Id
    guildMembershipApplication.PlayerId             = msg.PlayerId
    guildMembershipApplication.GuildId              = guild.Id
    guildMembershipApplication.JoinType             = types.GuildJoinType_direct
    guildMembershipApplication.RegistrationStatus   = types.RegistrationStatus_approved

    // Look up joining account
    targetPlayer := k.UpsertPlayer(ctx, msg.Creator, true)
    targetPlayer.GuildId = msg.GuildId
    k.SubstationConnectPlayer(ctx, substation, targetPlayer)

    // TODO (Possibly) - One thing we're not doing here yet is clearing out any
    // permissions related to the previous guild. This could get messy so doing it
    // manually might be best. That said, perhaps it could be a configuration option
    // for guilds to define what happens on leave.

    k.EventGuildMembershipApplication(ctx, guildMembershipApplication)

	return &types.MsgGuildMembershipResponse{GuildMembershipApplication: &guildMembershipApplication}, nil
}
