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
    emptyResponse := &types.MsgGuildMembershipResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

    // Add an Active Address record to the
    // indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

    callingPlayer, err := cc.GetSigningPlayer(msg.Creator)
    if err != nil {
        return emptyResponse, err
    }

    if (msg.PlayerId == "") {
        msg.PlayerId = callingPlayer.GetPlayerId()
    }

	if msg.GuildId == "" {
		msg.GuildId = callingPlayer.GetGuildId()
	}

    guildMembershipApplication, guildMembershipApplicationError := cc.GetOrCreateGuildMembershipApplicationCache(callingPlayer, types.GuildJoinType_direct, msg.GuildId, msg.PlayerId)
    if guildMembershipApplicationError != nil {
        return emptyResponse, guildMembershipApplicationError
    }

    guildMembershipApplicationError = guildMembershipApplication.VerifyDirectJoin()
    if guildMembershipApplicationError != nil {
        return emptyResponse, guildMembershipApplicationError
    }

    var infusionMigrationList []types.Infusion
    var infusionMigrationShares []math.LegacyDec
    var infusionMigrationReactor []sdk.ValAddress
    var infusionMigrationAmount []math.Int

    destinationReactor, destinationReactorFound := k.GetReactor(ctx, guildMembershipApplication.GetGuild().GetPrimaryReactorId())
    if (!destinationReactorFound) {
        return emptyResponse, types.NewObjectNotFoundError("reactor", guildMembershipApplication.GetGuild().GetPrimaryReactorId())
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
        /* msg.InfusionId is an unconstrained repeated field, and nothing in this
         * loop mutates the infusion it reads. The branch below for a reactor
         * already inside the guild is pure accumulation, so naming one owned
         * infusion N times counted its fuel N times and cleared a minimum the
         * player never actually held. Reject the repeat rather than skipping it:
         * a duplicate in the list is a malformed request, and quietly ignoring
         * it would hide the client bug that produced it.
         */
        seenInfusions := make(map[string]struct{}, len(msg.InfusionId))

        for _, infusionId := range msg.InfusionId {

            if _, duplicate := seenInfusions[infusionId]; duplicate {
                return emptyResponse, types.NewGuildMembershipError(msg.GuildId, msg.PlayerId, "duplicate_infusion").WithInfusion(infusionId)
            }
            seenInfusions[infusionId] = struct{}{}

            infusion, infusionFound := k.GetInfusionByID(ctx, infusionId)
            if (!infusionFound) {
                return emptyResponse, types.NewObjectNotFoundError("infusion", infusionId)
            }

            if (infusion.PlayerId != msg.PlayerId) {
                return emptyResponse, types.NewGuildMembershipError(msg.GuildId, msg.PlayerId, "infusion_ownership").WithInfusion(infusionId)
            }

            /* The stored PlayerId is not enough on its own to authorize what
             * follows. Below we redelegate on infusion.Address's behalf through
             * a raw keeper call, so that address, not the record describing it,
             * is what has to belong to the claimant. An address can be revoked
             * and re-registered to somebody else, and while UpsertInfusion
             * re-homes the record the next time staking touches it, a record
             * holding only a Defusing balance is never touched again.
             *
             * A subject address, so the pure lookup: an unregistered one has no
             * owner and is refused.
             */
            infusionAddressOwner, infusionAddressOwnerErr := cc.GetPlayerByAddress(infusion.Address)
            if infusionAddressOwnerErr != nil {
                return emptyResponse, infusionAddressOwnerErr
            }

            if (infusionAddressOwner.GetPlayerId() != msg.PlayerId) {
                return emptyResponse, types.NewGuildMembershipError(msg.GuildId, msg.PlayerId, "infusion_address_ownership").WithInfusion(infusionId)
            }

            if (infusion.DestinationType != types.ObjectType_reactor) {
                return emptyResponse, types.NewGuildMembershipError(msg.GuildId, msg.PlayerId, "invalid_infusion_type").WithInfusion(infusionId)
            }

            sourceReactor, sourceReactorFound := k.GetReactor(ctx, infusion.DestinationId)
            if (!sourceReactorFound) {
                return emptyResponse, types.NewObjectNotFoundError("reactor", infusion.DestinationId)
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
                    return emptyResponse, validationErr
                }

                // The actual redelegation process will only start after all values are checked
                // Save the validation results for later in the function
                infusionMigrationReactor = append(infusionMigrationReactor, sourceValidatorAccount)
                infusionMigrationShares = append(infusionMigrationShares, shares)
                infusionMigrationList = append(infusionMigrationList, infusion)

                accumulatedFuel, fuelOverflow := addFuel(currentFuel, redelegateAmount.Uint64())
                if fuelOverflow {
                    return emptyResponse, types.NewGuildMembershipError(msg.GuildId, msg.PlayerId, "fuel_overflow").WithInfusion(infusionId)
                }
                currentFuel = accumulatedFuel
            } else {
                accumulatedFuel, fuelOverflow := addFuel(currentFuel, infusion.Fuel)
                if fuelOverflow {
                    return emptyResponse, types.NewGuildMembershipError(msg.GuildId, msg.PlayerId, "fuel_overflow").WithInfusion(infusionId)
                }
                currentFuel = accumulatedFuel
            }

        }

        if (currentFuel < guildMembershipApplication.GetGuild().GetJoinInfusionMinimum()) {
            return emptyResponse, types.NewGuildMembershipError(msg.GuildId, msg.PlayerId, "minimum_not_met").WithMinimum(guildMembershipApplication.GetGuild().GetJoinInfusionMinimum(), currentFuel)
        }
    }

	if msg.SubstationId != "" {
	    substationOverrideError := guildMembershipApplication.SetSubstationIdOverride(msg.SubstationId)
	    if substationOverrideError != nil {
	        return emptyResponse, substationOverrideError
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
            return emptyResponse, redelegationErr
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

    if directJoinError := guildMembershipApplication.DirectJoin(); directJoinError != nil {
        return emptyResponse, directJoinError
    }

	cc.CommitAll()
	return &types.MsgGuildMembershipResponse{GuildMembershipApplication: &guildMembershipApplication.GuildMembershipApplication}, nil
}

// addFuel sums infusion fuel, reporting a wrap rather than performing one.
//
// The total is compared against the guild's join minimum, so a wrap does not
// merely produce a wrong number: it produces a small one, which is the direction
// that lets somebody in.
func addFuel(current uint64, addition uint64) (uint64, bool) {
    total := current + addition
    if total < current {
        return 0, true
    }
    return total, false
}
