package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
)

/* GuildCreate founds a guild, either by solving the chain-global charter
 * proof-of-work or by spending a reactor's one-time entitlement.
 *
 * The signer is the solver; the founder owns the guild and is the subject of
 * membership, substation, reactor, and ownership checks.
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

	// The founder is message-supplied and must resolve to existing state.
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

	// Founding leaves the current guild; an owner must transfer it first.
	oldGuild, membershipErr := guildCharterLeavable(cc, founder)
	if membershipErr != nil {
		return emptyResponse, membershipErr
	}

	// Substation rights belong to the founder, not a third-party solver.
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

	// Only proof-based founding moves the anchor; entitlements do not invalidate
	// work already in progress.
	provenByWork := msg.Proof != "" || msg.Nonce != ""

	if founder.GetPlayerId() != solver.GetPlayerId() {
		// Third-party consent is single-use only when the proof path moves the
		// anchor, so it cannot authorize the entitlement path.
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

	// GuildId also marks the reactor entitlement spent, so binding it requires
	// the founder's reactor permission.
	if reactor.GetReactor().GuildId == "" && reactor.CanCreateGuildBy(founder) == nil {
		reactor.SetGuild(guild.Id)
	}

	cc.CommitAll()
	return &types.MsgGuildCreateResponse{GuildId: guild.Id}, nil
}
