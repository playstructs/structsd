package keeper

import (
	errorsmod "cosmossdk.io/errors"

	"structs/x/structs/types"
)

func (cc *CurrentContext) GetGuildMembershipApp(guildId string, playerId string) *GuildMembershipApplicationCache {
	appKey := playerId + "@" + guildId

	if cache, exists := cc.guildMembershipApps[appKey]; exists {
		return cache
	}

    cc.guildMembershipApps[appKey] = &GuildMembershipApplicationCache{
                                        GuildMembershipApplicationId: appKey,
                                        CC: cc,
                                        Changed: false,
                                    }

	return cc.guildMembershipApps[appKey]
}

func (cc *CurrentContext) GenesisImportGuildMembershipApplication(app types.GuildMembershipApplication) {
	cache := cc.GetGuildMembershipApp(app.GuildId, app.PlayerId)
	cache.GuildMembershipApplication = app
	cache.GuildMembershipApplicationLoaded = true
	cache.Changed = true
}

/* Membership applications are two-sided. Creation paths use GetOrCreate;
 * approval and acceptance paths use GetPending so a missing first leg can
 * never be synthesized as evidence of consent.
 */

// resolveGuildMembershipApplication is the part both constructors share:
// resolve the target player, refuse one who is already a member, load the guild,
// attach the caller, and report whether an application is on file.
func (cc *CurrentContext) resolveGuildMembershipApplication(callingPlayer *PlayerCache, joinType types.GuildJoinType, guildId string, playerId string) (*GuildMembershipApplicationCache, bool, error) {

	targetPlayer, err := cc.GetExistingPlayer(playerId)
	if err != nil {
		return &GuildMembershipApplicationCache{}, false, err
	}

	if targetPlayer.GetGuildId() == guildId {
		// This delete does not survive: every caller turns the error below into
		// a failed message and the SDK discards the message cache with it. Kept
		// as-is because a leftover row naming a player who is already a member
		// is inert, but do not read it as cleanup that lands.
		cc.k.ClearGuildMembershipApplication(cc.ctx, guildId, playerId)
		return &GuildMembershipApplicationCache{}, false, types.NewGuildMembershipError(guildId, playerId, "already_member")
	}

	guild := cc.GetGuild(guildId)
	if err := guild.CheckGuild(); err != nil {
		return &GuildMembershipApplicationCache{}, false, err
	}

	guildMembershipApplication := cc.GetGuildMembershipApp(guildId, playerId)
	guildMembershipApplicationFound := guildMembershipApplication.LoadGuildMembershipApplication()
	guildMembershipApplication.CallingPlayer = callingPlayer

	if guildMembershipApplicationFound && guildMembershipApplication.GetJoinType() != joinType {
		return &GuildMembershipApplicationCache{}, false, types.NewGuildMembershipError(guildId, playerId, "join_type_mismatch")
	}

	return guildMembershipApplication, guildMembershipApplicationFound, nil
}

// GetOrCreateGuildMembershipApplicationCache is for the handlers that open an
// application: GuildMembershipRequest, GuildMembershipInvite and
// GuildMembershipJoin. It creates a proposed application when none exists.
//
// Never use it on a path that approves, denies or revokes — the record it
// invents is exactly the consent such a path is supposed to be verifying.
func (cc *CurrentContext) GetOrCreateGuildMembershipApplicationCache(callingPlayer *PlayerCache, joinType types.GuildJoinType, guildId string, playerId string) (*GuildMembershipApplicationCache, error) {

	guildMembershipApplication, guildMembershipApplicationFound, err := cc.resolveGuildMembershipApplication(callingPlayer, joinType, guildId, playerId)
	if err != nil {
		return &GuildMembershipApplicationCache{}, err
	}

	// The guild-side gate runs whether or not a row already exists. It used to
	// sit inside the creation branch only, which left GuildMembershipInvite —
	// whose sole authorization this is, the handler having no Verify call of its
	// own — completely unchecked against any invite already on file. An outsider
	// could name another player's pending invite and carry on into
	// SetSubstationIdOverride, which validates rights on the destination
	// substation and not on the guild, redirecting the invite to a substation
	// they own; the invitee then connected there on accept.
	//
	// Every caller passes a compile-time constant, so no transaction can reach
	// the default today. It is the callers that make it unreachable, not the
	// switch: a joinType read from a message or a stored record would be an open
	// proto3 enum, and falling through left guildPermissionError nil, skipping
	// the guild-side check entirely.
	guild := cc.GetGuild(guildId)

	var guildPermissionError error
	switch joinType {
	case types.GuildJoinType_invite:
		guildPermissionError = guild.CanInviteMembers(callingPlayer)
	case types.GuildJoinType_request:
		guildPermissionError = guild.CanRequestMembership()
	case types.GuildJoinType_proxy:
		guildPermissionError = guild.CanAddMembersByProxy(callingPlayer)
	case types.GuildJoinType_direct:
		// Check on Infusion
	default:
		guildPermissionError = types.NewGuildMembershipError(guildId, playerId, "invalid_join_type")
	}
	if guildPermissionError != nil {
		return &GuildMembershipApplicationCache{}, guildPermissionError
	}

	if !guildMembershipApplicationFound {
		guildMembershipApplication.GuildMembershipApplication.Proposer = callingPlayer.GetPlayerId()

		guildMembershipApplication.GuildMembershipApplication.PlayerId = playerId
		guildMembershipApplication.GuildMembershipApplication.GuildId = guildId
		guildMembershipApplication.GuildMembershipApplication.JoinType = joinType
		guildMembershipApplication.GuildMembershipApplication.RegistrationStatus = types.RegistrationStatus_proposed

		guildMembershipApplication.Changed = true
	}

	return guildMembershipApplication, nil
}

// GetPendingGuildMembershipApplicationCache is for the handlers that act on an
// application already on file: the approve, deny and revoke paths on both the
// request and the invite side. It refuses when the store has nothing, so the
// application stands as proof that the other side consented.
//
// The guild-side and player-side checks stay in the handlers, which is where the
// asymmetry lives: an invite is approved by the player and revoked by the guild,
// a request the other way around.
func (cc *CurrentContext) GetPendingGuildMembershipApplicationCache(callingPlayer *PlayerCache, joinType types.GuildJoinType, guildId string, playerId string) (*GuildMembershipApplicationCache, error) {

	guildMembershipApplication, guildMembershipApplicationFound, err := cc.resolveGuildMembershipApplication(callingPlayer, joinType, guildId, playerId)
	if err != nil {
		return &GuildMembershipApplicationCache{}, err
	}

	if !guildMembershipApplicationFound {
		return &GuildMembershipApplicationCache{}, errorsmod.Wrapf(types.ErrGuildMembershipApplication, "no application on file (%s)", GetGuildMembershipApplicationID(guildId, playerId))
	}

	if statusError := guildMembershipApplication.requirePending(); statusError != nil {
		return &GuildMembershipApplicationCache{}, statusError
	}

	return guildMembershipApplication, nil
}

func (cc *CurrentContext) GetGuildMembershipKickCache(callingPlayer *PlayerCache, guildId string, playerId string) (*GuildMembershipApplicationCache, error) {

	targetPlayer, err := cc.GetExistingPlayer(playerId)
	if err != nil {
		return &GuildMembershipApplicationCache{}, err
	}

	if targetPlayer.GetGuildId() != guildId {
		return &GuildMembershipApplicationCache{}, types.NewGuildMembershipError(guildId, playerId, "not_member")
	}

	guild := cc.GetGuild(guildId)
	if err := guild.CheckGuild(); err != nil {
		return &GuildMembershipApplicationCache{}, err
	}

	if guild.GetOwnerId() == playerId {
		return &GuildMembershipApplicationCache{}, types.NewGuildMembershipError(guildId, playerId, "cannot_kick_owner")
	}

	guildMembershipApplication := cc.GetGuildMembershipApp(guildId, playerId)
    guildMembershipApplication.LoadGuildMembershipApplication()
    guildMembershipApplication.CallingPlayer = callingPlayer

	// No direct ClearGuildMembershipApplication here: the revoked status set
	// below drives the cache commit to clear the row on the success path, so a
	// raw delete would be a second, unordered deletion that also runs ahead of
	// the permission checks.

	guildPermissionError := guild.CanKickMembers(callingPlayer)
	if guildPermissionError != nil {
		return &GuildMembershipApplicationCache{}, guildPermissionError
	}

	// Rank is standing inside this guild. A caller who is not a member has
	// only CanKickMembers, which the owner already passed; their rank in
	// some other guild is an unrelated number.
	if callingPlayer.GetGuildId() == guildId && callingPlayer.GetGuildRank() >= targetPlayer.GetGuildRank() {
		return &GuildMembershipApplicationCache{}, types.NewPermissionError(
			"player", callingPlayer.GetPlayerId(),
			"guild", guildId,
			uint64(types.PermGuildMembership), "guild_membership_kick",
		)
	}

	guildMembershipApplication.GuildMembershipApplication.Proposer = callingPlayer.GetPlayerId()
	guildMembershipApplication.GuildMembershipApplication.PlayerId = playerId
	guildMembershipApplication.GuildMembershipApplication.GuildId = guildId
	guildMembershipApplication.GuildMembershipApplication.JoinType = types.GuildJoinType_direct
	guildMembershipApplication.GuildMembershipApplication.RegistrationStatus = types.RegistrationStatus_revoked

	// Not true until kicked
	guildMembershipApplication.Changed = true

	return guildMembershipApplication, nil
}