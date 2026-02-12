package keeper

import (
	"context"
    "time"
    "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (k msgServer) GuildMembershipJoin(goCtx context.Context, msg *types.MsgGuildMembershipJoin) (*types.MsgGuildMembershipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayer, err := cc.GetPlayerByAddress(msg.Creator)
    if err != nil {
        return &types.MsgGuildMembershipResponse{}, err
    }

    // Use cache permission methods
    callingPlayerPermissionError := callingPlayer.CanBeAdministratedBy(msg.Creator, types.PermissionAssociations)
    if callingPlayerPermissionError != nil {
        return &types.MsgGuildMembershipResponse{}, callingPlayerPermissionError
    }

    if (msg.PlayerId == "") {
        msg.PlayerId = callingPlayer.GetPlayerId()
    }

	if msg.GuildId == "" {
		msg.GuildId = callingPlayer.GetGuildId()
	}

    guildMembershipApplication, guildMembershipApplicationError := cc.GetGuildMembershipApplicationCache(callingPlayer, types.GuildJoinType_direct, msg.GuildId, msg.PlayerId)
    if guildMembershipApplicationError != nil {
        return &types.MsgGuildMembershipResponse{}, guildMembershipApplicationError
    }

    guildMembershipApplicationError = guildMembershipApplication.VerifyDirectJoin()
    if guildMembershipApplicationError != nil {
        return &types.MsgGuildMembershipResponse{}, guildMembershipApplicationError
    }

    var infusionMigrationList []types.Infusion
    var infusionMigrationShares []math.LegacyDec
    var infusionMigrationReactor []sdk.ValAddress
    var infusionMigrationAmount []math.Int

    destinationReactor, destinationReactorFound := k.GetReactor(ctx, guildMembershipApplication.GetGuild().GetPrimaryReactorId())
    if (!destinationReactorFound) {
        return &types.MsgGuildMembershipResponse{}, types.NewObjectNotFoundError("reactor", guildMembershipApplication.GetGuild().GetPrimaryReactorId())
    }
    destinationValidatorAccount, _ := sdk.ValAddressFromBech32(destinationReactor.Validator)

    if (guildMembershipApplication.GetGuild().GetJoinInfusionMinimum() != 0) {
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
                return &types.MsgGuildMembershipResponse{}, types.NewObjectNotFoundError("infusion", infusionId)
            }

            if (infusion.PlayerId != msg.PlayerId) {
                return &types.MsgGuildMembershipResponse{}, types.NewGuildMembershipError(msg.GuildId, msg.PlayerId, "infusion_ownership").WithInfusion(infusionId)
            }

            if (infusion.DestinationType != types.ObjectType_reactor) {
                return &types.MsgGuildMembershipResponse{}, types.NewGuildMembershipError(msg.GuildId, msg.PlayerId, "invalid_infusion_type").WithInfusion(infusionId)
            }

            sourceReactor, sourceReactorFound := k.GetReactor(ctx, infusion.DestinationId)
            if (!sourceReactorFound) {
                return &types.MsgGuildMembershipResponse{}, types.NewObjectNotFoundError("reactor", infusion.DestinationId)
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

        if (currentFuel < guildMembershipApplication.GetGuild().GetJoinInfusionMinimum()) {
            return &types.MsgGuildMembershipResponse{}, types.NewGuildMembershipError(msg.GuildId, msg.PlayerId, "minimum_not_met").WithMinimum(guildMembershipApplication.GetGuild().GetJoinInfusionMinimum(), currentFuel)
        }
    }

	if msg.SubstationId != "" {
	    substationOverrideError := guildMembershipApplication.SetSubstationIdOverride(msg.SubstationId)
	    if substationOverrideError != nil {
	        return &types.MsgGuildMembershipResponse{}, substationOverrideError
	    }
	}

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

    guildMembershipApplication.DirectJoin()

	return &types.MsgGuildMembershipResponse{GuildMembershipApplication: &guildMembershipApplication.GuildMembershipApplication}, nil
}
