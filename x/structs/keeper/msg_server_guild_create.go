package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

/* GuildCreate founds a guild, either by solving the chain-global charter
 * proof-of-work or by spending a reactor's one-time entitlement.
 *
 * Two identities, and confusing them is the sharpest hazard in this handler.
 * The signer is the *solver*: its only roles are to be half of the work preimage
 * and to be credited on the guild record. The *founder* owns the resulting
 * guild, and is therefore the subject of every authorization and every mutation
 * below — the membership move, the substation rights, the ownership. The two are
 * the same player in the ordinary case, which is exactly why a mistake here
 * would not show up in the common path.
 *
 * Nothing but the work is a gate on creation any more. A guild used to require
 * PermReactorGuildCreate on a reactor, which made guilds a validator perk;
 * now anyone who solves the puzzle may found one, and that permission survives
 * only as the gate on binding the reactor's own GuildId (see below).
 */
func (k msgServer) GuildCreate(goCtx context.Context, msg *types.MsgGuildCreate) (*types.MsgGuildCreateResponse, error) {
	emptyResponse := &types.MsgGuildCreateResponse{}
	ctx := sdk.UnwrapSDKContext(goCtx)
	cc := k.NewCurrentContext(ctx)

	// Add an Active Address record to the
	// indexer for UI requirements
	k.AddressEmitActivity(ctx, msg.Creator)

	solver, playerErr := cc.GetSigningPlayer(msg.Creator)
	if playerErr != nil {
		return emptyResponse, types.NewPlayerRequiredError(msg.Creator, "guild_create")
	}

	/* The founder is a message-supplied player id, so it resolves through
	 * GetExistingPlayer: GetPlayer is an allocator and would hand back a phantom
	 * that reads as a zero value and silently reverts everything written to it.
	 */
	founder := solver
	if msg.FounderPlayerId != "" && msg.FounderPlayerId != solver.GetPlayerId() {
		var founderErr error
		founder, founderErr = cc.GetExistingPlayer(msg.FounderPlayerId)
		if founderErr != nil {
			return emptyResponse, founderErr
		}
	}

	reactor := cc.GetReactor(msg.ReactorId)
	if reactor.CheckReactor() != nil {
		return emptyResponse, types.NewReactorError("guild_create", "required").WithAddress(msg.Creator, "validator")
	}

	/* Founding a guild leaves the one you are in, and an owner cannot leave, so
	 * they have to hand their guild over first. Checked before anything mutates:
	 * the old handler checked nothing at all, which let one player accumulate
	 * guilds while their Player record pointed only at the newest.
	 */
	oldGuild, membershipErr := guildCharterLeavable(cc, founder)
	if membershipErr != nil {
		return emptyResponse, membershipErr
	}

	/* The entry substation is validated against the founder, not the signer. A
	 * third-party solver holds no rights on the founder's substation and never
	 * will; what vouches for the substation named is the founder's consent
	 * signature, which covers it.
	 */
	if msg.EntrySubstationId != "" {
		substation := cc.GetSubstation(msg.EntrySubstationId)
		if substation.CheckSubstation() != nil {
			return emptyResponse, types.NewObjectNotFoundError("substation", msg.EntrySubstationId)
		}

		substationPermissionErr := substation.CanManageConnectionsBy(founder)
		if substationPermissionErr != nil {
			return emptyResponse, substationPermissionErr
		}
	}

	anchor := k.CharterAnchor(ctx)

	/* Which path. A proof present means the puzzle was solved; absent means the
	 * reactor entitlement is being spent. Only the proof path moves the anchor:
	 * wiping every pool's in-flight mining because a validator collected a perk
	 * would make the two paths interfere for no reason.
	 */
	provenByWork := msg.Proof != "" || msg.Nonce != ""

	if founder.GetPlayerId() != solver.GetPlayerId() {
		/* A consent is single-use *because* founding moves the anchor, so only a
		 * path that moves it may spend one. The entitlement path deliberately
		 * does not, which would leave the signature live until somebody else
		 * proof-founded — long enough for the solver to mine a fresh proof and
		 * replay it, dragging the founder out of whatever guild they had since
		 * joined and into a second one at the entry rank.
		 *
		 * Refusing costs nothing real. guildCharterReactorEligible below already
		 * demands CanCreateGuildBy(founder), so a founder who can use that path
		 * at all holds reactor permission and can simply sign for themselves;
		 * there is no race on that path to sign ahead of.
		 */
		if !provenByWork {
			return emptyResponse, types.NewReactorError("guild_charter", "consent_needs_proof").
				WithReactor(reactor.GetReactorId())
		}

		if consentErr := k.verifyGuildCharterConsent(cc, msg, solver, founder, anchor); consentErr != nil {
			return emptyResponse, consentErr
		}
	}

	charterSolverId := ""
	achievedDifficulty := uint64(0)

	if provenByWork {
		// The proof binds no reactor, so the reactor is gated on its own account.
		// The entitlement path below is strictly stronger and checks for itself.
		if liveErr := k.guildCharterReactorLive(ctx, reactor); liveErr != nil {
			return emptyResponse, liveErr
		}

		hashInput := types.GuildCharterWorkInput(solver.GetPlayerId(), founder.GetPlayerId(), anchor, msg.Nonce)

		var valid bool
		valid, achievedDifficulty = types.HashBuildAndCheckDifficulty(hashInput, msg.Proof, k.CharterAge(ctx), k.GetParams(ctx).CharterDifficultyRange())
		if !valid {
			return emptyResponse, types.NewWorkFailureError(types.GuildCharterErrorOperation, solver.GetPlayerId(), hashInput)
		}

		charterSolverId = solver.GetPlayerId()
	} else if eligibilityErr := k.guildCharterReactorEligible(ctx, reactor, founder); eligibilityErr != nil {
		return emptyResponse, eligibilityErr
	}

	guild := k.AppendGuild(ctx, msg.Endpoint, msg.EntrySubstationId, reactor.GetReactor(), founder.GetPlayer(), charterSolverId)

	guildCharterLeaveAndJoin(cc, founder, oldGuild, guild.Id, msg.EntrySubstationId)

	if provenByWork {
		k.SetGuildCharterAnchor(ctx, uint64(ctx.BlockHeight()))

		_ = ctx.EventManager().EmitTypedEvent(&types.EventHashSuccess{EventHashSuccessDetail: &types.EventHashSuccessDetail{
			CallerAddress: msg.Creator,
			Category:      types.GuildCharterErrorOperation,
			Difficulty:    achievedDifficulty,
			ObjectId:      guild.Id,
		}})
	}

	/* Binding the reactor's own GuildId is what the old creation permission
	 * becomes. It matters more than it looks: GuildId is also the marker that
	 * spends a reactor's free-guild entitlement, so without a permission check
	 * here anyone could name a stranger's reactor and burn its entitlement while
	 * making their guild that reactor's official one. On the entitlement path the
	 * check has already passed, so this always binds and always spends.
	 */
	if reactor.GetReactor().GuildId == "" && reactor.CanCreateGuildBy(founder) == nil {
		reactor.SetGuild(guild.Id)
	}

	cc.CommitAll()
	return &types.MsgGuildCreateResponse{GuildId: guild.Id}, nil
}

/* guildCharterLeavable reports whether a founder is free to be moved into a new
 * guild, returning the guild they are leaving when there is one.
 *
 * An owner is refused rather than migrated. Leaving a guild you own would strand
 * it with an owner who is not a member, which is the state the old handler
 * created on every repeat call, and there is a proper way to do it:
 * MsgGuildUpdateOwnerId, which now also revokes the outgoing owner's rights.
 *
 * A GuildId pointing at a guild that will not load is nothing to leave, so it is
 * silently overwritten rather than treated as a failure.
 */
func guildCharterLeavable(cc *CurrentContext, founder *PlayerCache) (*GuildCache, error) {
	if founder.GetGuildId() == "" {
		return nil, nil
	}

	oldGuild := cc.GetGuild(founder.GetGuildId())
	if oldGuild.CheckGuild() != nil {
		return nil, nil
	}

	if oldGuild.GetOwnerId() == founder.GetPlayerId() {
		return nil, types.NewGuildMembershipError(oldGuild.GetGuildId(), founder.GetPlayerId(), "is_owner")
	}

	return oldGuild, nil
}

/* guildCharterLeaveAndJoin moves the founder out of their old guild and into the
 * new one.
 *
 * Mirrors GuildMembershipJoinProxy's handling deliberately, including both of
 * its conservative choices. The old substation is only dropped when it is that
 * guild's entry substation, because otherwise it is likely the player's own and
 * not something to sever; and the new entry substation is only connected to when
 * the player has no substation at all, so an existing connection is never
 * silently rerouted.
 */
func guildCharterLeaveAndJoin(cc *CurrentContext, founder *PlayerCache, oldGuild *GuildCache, guildId string, entrySubstationId string) {
	if oldGuild != nil {
		if founder.GetSubstationId() != "" && founder.GetSubstationId() == oldGuild.GetEntrySubstationId() {
			founder.DisconnectSubstation()
		}
	}

	founder.SetGuild(guildId)
	founder.SetGuildRank(1)

	if entrySubstationId != "" && founder.GetSubstationId() == "" {
		founder.MigrateSubstation(entrySubstationId)
	}
}
