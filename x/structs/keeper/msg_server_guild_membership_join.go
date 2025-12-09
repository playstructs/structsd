package keeper

import (
	"context"
    "time"
    "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (k msgServer) GuildMembershipJoin(goCtx context.Context, msg *types.MsgGuildMembershipJoin) (*types.MsgGuildMembershipResponse, error) {
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

    if (player.Id != msg.PlayerId) {
        if (!k.PermissionHasOneOf(ctx, GetObjectPermissionIDBytes(msg.PlayerId, player.Id), types.PermissionAssociations)) {
            return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrPermissionGuildRegister, "Calling player (%s) has no Player Association permissions with the Player (%s) ", msg.PlayerId, player.Id)
        }
    }

    targetPlayer, targetPlayerFound := k.GetPlayer(ctx, msg.PlayerId)
    if !targetPlayerFound {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Player (%s) not found", msg.PlayerId)
    }

	// look up destination guild
	guild, guildFound := k.GetGuild(ctx, msg.GuildId)

    if (!guildFound) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Guild (%s) not found", msg.GuildId)
    }


    guildMembershipApplication, guildMembershipApplicationFound := k.GetGuildMembershipApplication(ctx, msg.GuildId, msg.PlayerId)
    if (guildMembershipApplicationFound) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Membership Application already pending. Deny request or invitation first")
    }

    var infusionMigrationList []types.Infusion
    var infusionMigrationShares []math.LegacyDec
    var infusionMigrationReactor []sdk.ValAddress
    var infusionMigrationAmount []math.Int

    destinationReactor, destinationReactorFound := k.GetReactor(ctx, guild.PrimaryReactorId)
    if (!destinationReactorFound) {
        return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Somehow this reactor (%s) doesn't exist, you should tell an adult",guild.PrimaryReactorId)
    }
    destinationValidatorAccount, _ := sdk.ValAddressFromBech32(destinationReactor.Validator)

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
         * - If the destination reactor is not part of the guild, we need to migrate the assets over.
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

            sourceReactor, sourceReactorFound := k.GetReactor(ctx, infusion.DestinationId)
            if (!sourceReactorFound) {
                return &types.MsgGuildMembershipResponse{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Somehow this reactor (%s) doesn't exist, you should tell an adult",infusion.DestinationId)
            }

            if (sourceReactor.GuildId != msg.GuildId) {
                /*
                    Previously, this would fail at this point but now we'll be migrating the infusion
                    over to the new guilds reactor.

                    Before migrating the infusions, we need to...
                    [] confirm they can be migrated (and are not stuck in a redelegation)
                    [] confirm the total amount of infusions migrating will meet the minimum

                */

                redelegateAmount := math.NewIntFromUint64(infusion.Fuel)
                infusionMigrationAmount = append(infusionMigrationAmount, redelegateAmount)

                // The validation should never fail assuming there isn't a bug in the Infusion system
                // but we can use this function to reliably calculate the shares
                infusionAccount, _ := sdk.AccAddressFromBech32(infusion.Address)
                sourceValidatorAccount, _ := sdk.ValAddressFromBech32(sourceReactor.Validator)
                shares, validationErr := k.stakingKeeper.ValidateUnbondAmount(
                    ctx, infusionAccount, sourceValidatorAccount, redelegateAmount,
                )
                if validationErr != nil {
                    return &types.MsgGuildMembershipResponse{}, validationErr
                }

                // The actual redelegation process will only start after all values are checked
                // Save the validation results for later in the function
                infusionMigrationReactor = append(infusionMigrationReactor, sourceValidatorAccount)
                infusionMigrationShares = append(infusionMigrationShares, shares)
                infusionMigrationList = append(infusionMigrationList, infusion)

                currentFuel = currentFuel + redelegateAmount.Uint64()
            } else {
                currentFuel = currentFuel + infusion.Fuel
            }

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
        substation, substationFound = k.GetSubstation(ctx, msg.SubstationId)

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
        substation, substationFound = k.GetSubstation(ctx, guildMembershipApplication.SubstationId)
    }

    guildMembershipApplication.Proposer             = player.Id
    guildMembershipApplication.PlayerId             = msg.PlayerId
    guildMembershipApplication.GuildId              = guild.Id
    guildMembershipApplication.JoinType             = types.GuildJoinType_direct
    guildMembershipApplication.RegistrationStatus   = types.RegistrationStatus_approved

    // This seems like a safe place for this.
    // We either need to do this basically last, or undo any changes if errors occur.
    for migrationInfusionIndex, migrationInfusion := range infusionMigrationList {
        // Handle the migration of Infusions from previous reactor to new

        infusionAccount, _ := sdk.AccAddressFromBech32(migrationInfusion.Address)
        completionTime, redelegationErr := k.stakingKeeper.BeginRedelegation(
            ctx,
            infusionAccount,
            infusionMigrationReactor[migrationInfusionIndex],
            destinationValidatorAccount,
            infusionMigrationShares[migrationInfusionIndex],
        )

        // This is kinda a problem by now tbh
        // Maybe tell an adult if this happens
        if redelegationErr != nil {
            return &types.MsgGuildMembershipResponse{}, redelegationErr
        }

        ctx.EventManager().EmitEvents(sdk.Events{
            sdk.NewEvent(
                staking.EventTypeRedelegate,
                sdk.NewAttribute(staking.AttributeKeySrcValidator, infusionMigrationReactor[migrationInfusionIndex].String()),
                sdk.NewAttribute(staking.AttributeKeyDstValidator, destinationReactor.Validator),
                sdk.NewAttribute(staking.AttributeKeyDelegator, migrationInfusion.Address),
                sdk.NewAttribute(sdk.AttributeKeyAmount, infusionMigrationAmount[migrationInfusionIndex].String()),
                sdk.NewAttribute(staking.AttributeKeyCompletionTime, completionTime.Format(time.RFC3339)),
            ),
        })

    }

    // Guild Variable gets committed in the SubstationConnectPlayer function
    targetPlayer.GuildId = msg.GuildId
    k.SubstationConnectPlayer(ctx, substation, targetPlayer)

    // TODO (Possibly) - One thing we're not doing here yet is clearing out any
    // permissions related to the previous guild. This could get messy so doing it
    // manually might be best. That said, perhaps it could be a configuration option
    // for guilds to define what happens on leave.

    k.EventGuildMembershipApplication(ctx, guildMembershipApplication)

	return &types.MsgGuildMembershipResponse{GuildMembershipApplication: &guildMembershipApplication}, nil
}
