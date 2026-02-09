package keeper

import (
	"context"

	"structs/x/structs/types"
	//sdk "github.com/cosmos/cosmos-sdk/types"
	// Used in Randomness Orb
	//"cosmossdk.io/math"
	//authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

type GuildMembershipApplicationCache struct {
	K   *Keeper
	Ctx context.Context
	CC  *CurrentContext

	AnyChange bool
	Ready     bool

	GuildMembershipApplicationChanged bool
	GuildMembershipApplication        types.GuildMembershipApplication

	CallingPlayer *PlayerCache

	Guild  *GuildCache
	Player *PlayerCache

	ProposerLoaded bool
	Proposer       *PlayerCache

	SubstationLoaded bool
	Substation       *SubstationCache
}

// Build this initial Guild Membership Application Cache object
func (k *Keeper) GetGuildMembershipApplicationCache(ctx context.Context, callingPlayer *PlayerCache, joinType types.GuildJoinType, guildId string, playerId string) (GuildMembershipApplicationCache, error) {

	targetPlayer, err := k.GetPlayerCacheFromId(ctx, playerId)
	if err != nil {
		return GuildMembershipApplicationCache{}, types.NewObjectNotFoundError("player", playerId)
	}

	if targetPlayer.GetGuildId() == guildId {
		k.ClearGuildMembershipApplication(ctx, guildId, playerId)
		return GuildMembershipApplicationCache{}, types.NewGuildMembershipError(guildId, playerId, "already_member")
	}

	guild := k.GetGuildCacheFromId(ctx, guildId)
	if !guild.LoadGuild() {
		return GuildMembershipApplicationCache{}, types.NewObjectNotFoundError("guild", guildId)
	}

	guildMembershipApplication, guildMembershipApplicationFound := k.GetGuildMembershipApplication(ctx, guildId, playerId)

	guildMembershipApplicationChanged := false
	proposerLoaded := false
	var proposer *PlayerCache

	if guildMembershipApplicationFound {

		if guildMembershipApplication.JoinType != joinType {
			return GuildMembershipApplicationCache{}, types.NewGuildMembershipError(guildId, playerId, "join_type_mismatch")
		}

	} else {

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
		}
		if guildPermissionError != nil {
			return GuildMembershipApplicationCache{}, guildPermissionError
		}

		guildMembershipApplication.Proposer = callingPlayer.GetPlayerId()
		proposer = callingPlayer
		proposerLoaded = true

		guildMembershipApplication.PlayerId = playerId
		guildMembershipApplication.GuildId = guildId
		guildMembershipApplication.JoinType = joinType
		guildMembershipApplication.RegistrationStatus = types.RegistrationStatus_proposed

		guildMembershipApplicationChanged = true
	}

	return GuildMembershipApplicationCache{
		CallingPlayer: callingPlayer,

		K:   k,
		Ctx: ctx,

		AnyChange: guildMembershipApplicationChanged,

		GuildMembershipApplication:        guildMembershipApplication,
		GuildMembershipApplicationChanged: guildMembershipApplicationChanged,

		Player: &targetPlayer,

		Proposer:       proposer,
		ProposerLoaded: proposerLoaded,

		Guild: &guild,

		SubstationLoaded: false,
	}, nil
}

func (k *Keeper) GetGuildMembershipKickCache(ctx context.Context, callingPlayer *PlayerCache, guildId string, playerId string) (GuildMembershipApplicationCache, error) {

	targetPlayer, err := k.GetPlayerCacheFromId(ctx, playerId)
	if err != nil {
		return GuildMembershipApplicationCache{}, types.NewObjectNotFoundError("player", playerId)
	}

	if targetPlayer.GetGuildId() != guildId {
		return GuildMembershipApplicationCache{}, types.NewGuildMembershipError(guildId, playerId, "not_member")
	}

	guild := k.GetGuildCacheFromId(ctx, guildId)
	if !guild.LoadGuild() {
		return GuildMembershipApplicationCache{}, types.NewObjectNotFoundError("guild", guildId)
	}

	if guild.GetOwnerId() == playerId {
		return GuildMembershipApplicationCache{}, types.NewGuildMembershipError(guildId, playerId, "cannot_kick_owner")
	}

	guildMembershipApplication, guildMembershipApplicationFound := k.GetGuildMembershipApplication(ctx, guildId, playerId)

	if guildMembershipApplicationFound {
		k.ClearGuildMembershipApplication(ctx, guildId, playerId)
	}

	guildPermissionError := guild.CanKickMembers(callingPlayer)
	if guildPermissionError != nil {
		return GuildMembershipApplicationCache{}, guildPermissionError
	}

	guildMembershipApplication.Proposer = callingPlayer.GetPlayerId()
	guildMembershipApplication.PlayerId = playerId
	guildMembershipApplication.GuildId = guildId
	guildMembershipApplication.JoinType = types.GuildJoinType_direct
	guildMembershipApplication.RegistrationStatus = types.RegistrationStatus_revoked

	// Not true until kicked
	guildMembershipApplicationChanged := false

	return GuildMembershipApplicationCache{
		CallingPlayer: callingPlayer,

		K:   k,
		Ctx: ctx,

		AnyChange: false,

		GuildMembershipApplication:        guildMembershipApplication,
		GuildMembershipApplicationChanged: guildMembershipApplicationChanged,

		Player: &targetPlayer,

		Proposer:       callingPlayer,
		ProposerLoaded: true,

		Guild: &guild,

		SubstationLoaded: false,
	}, nil
}

func (cache *GuildMembershipApplicationCache) Commit() {
	cache.AnyChange = false

	cache.K.logger.Info("Updating Guild Membership Application From Cache", "guildId", cache.GetGuildMembershipApplication().GuildId, "playerId", cache.GetGuildMembershipApplication().PlayerId)

	if cache.GuildMembershipApplicationChanged {
		cache.K.EventGuildMembershipApplication(cache.Ctx, cache.GuildMembershipApplication)

		switch cache.GetRegistrationStatus() {
		case types.RegistrationStatus_proposed:
			cache.K.SetGuildMembershipApplication(cache.Ctx, cache.GuildMembershipApplication)
		case types.RegistrationStatus_approved:
			cache.K.ClearGuildMembershipApplication(cache.Ctx, cache.GetGuildId(), cache.GetPlayerId())
		case types.RegistrationStatus_denied:
			cache.K.ClearGuildMembershipApplication(cache.Ctx, cache.GetGuildId(), cache.GetPlayerId())
		case types.RegistrationStatus_revoked:
			cache.K.ClearGuildMembershipApplication(cache.Ctx, cache.GetGuildId(), cache.GetPlayerId())
		}
	}

}

func (cache *GuildMembershipApplicationCache) IsChanged() bool {
	return cache.AnyChange
}

func (cache *GuildMembershipApplicationCache) ID() string {
	return cache.GuildMembershipApplication.GuildId + "/" + cache.GuildMembershipApplication.PlayerId
}

func (cache *GuildMembershipApplicationCache) Changed() {
	cache.AnyChange = true
}

/* Separate Loading functions for each of the underlying containers */

// Load the Proposer Player data
func (cache *GuildMembershipApplicationCache) LoadProposer() bool {
	if cache.CC != nil {
		proposer, _ := cache.CC.GetPlayer(cache.GetProposerId())
		cache.Proposer = proposer
	} else {
		newProposer, _ := cache.K.GetPlayerCacheFromId(cache.Ctx, cache.GetProposerId())
		cache.Proposer = &newProposer
	}
	cache.ProposerLoaded = true
	return cache.ProposerLoaded
}

/* Getters
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */
func (cache *GuildMembershipApplicationCache) GetGuildMembershipApplication() types.GuildMembershipApplication {
	return cache.GuildMembershipApplication
}
func (cache *GuildMembershipApplicationCache) GetRegistrationStatus() types.RegistrationStatus {
	return cache.GetGuildMembershipApplication().RegistrationStatus
}
func (cache *GuildMembershipApplicationCache) GetJoinType() types.GuildJoinType {
	return cache.GetGuildMembershipApplication().JoinType
}

func (cache *GuildMembershipApplicationCache) GetGuildId() string {
	return cache.GetGuildMembershipApplication().GuildId
}
func (cache *GuildMembershipApplicationCache) GetGuild() *GuildCache { return cache.Guild }

// Get the Player data
func (cache *GuildMembershipApplicationCache) GetPlayerId() string {
	return cache.GetGuildMembershipApplication().PlayerId
}
func (cache *GuildMembershipApplicationCache) GetPlayer() *PlayerCache { return cache.Player }

// Get the Proposer data
func (cache *GuildMembershipApplicationCache) GetProposerId() string {
	return cache.GetGuildMembershipApplication().Proposer
}
func (cache *GuildMembershipApplicationCache) GetProposer() *PlayerCache {
	if !cache.ProposerLoaded {
		cache.LoadProposer()
	}
	return cache.Proposer
}

func (cache *GuildMembershipApplicationCache) GetSubstationId() (substationId string) {
	substationId = cache.GetGuildMembershipApplication().SubstationId
	if substationId == "" {
		substationId = cache.GetGuild().GetEntrySubstationId()
	}
	return
}

func (cache *GuildMembershipApplicationCache) SetSubstationIdOverride(substationId string) error {

	if cache.GuildMembershipApplication.SubstationId != substationId {

		substation := cache.K.GetSubstationCacheFromId(cache.Ctx, substationId)
		if !substation.LoadSubstation() {
			return types.NewObjectNotFoundError("substation", substationId)
		}

		substationPermissionError := substation.CanManagePlayerConnections(cache.CallingPlayer)
		if substationPermissionError != nil {
			return substationPermissionError
		}

		cache.Substation = &substation
		cache.SubstationLoaded = true

		cache.GuildMembershipApplication.SubstationId = substationId
		cache.GuildMembershipApplicationChanged = true
		cache.Changed()
	}

	return nil
}

func (cache *GuildMembershipApplicationCache) VerifyInviteAsGuild() error {
	guildPermissionError := cache.GetGuild().CanInviteMembers(cache.CallingPlayer)
	if guildPermissionError != nil {
		return guildPermissionError
	}

	if cache.GetJoinType() != types.GuildJoinType_invite {
		return types.NewGuildMembershipError(cache.GetGuildId(), cache.GetPlayerId(), "wrong_join_type").WithJoinType("invite")
	}

	return nil
}

func (cache *GuildMembershipApplicationCache) VerifyInviteAsPlayer() error {
	if cache.GetPlayerId() != cache.CallingPlayer.GetPlayerId() {
		if !cache.K.PermissionHasOneOf(cache.Ctx, GetObjectPermissionIDBytes(cache.GetPlayerId(), cache.CallingPlayer.GetPlayerId()), types.PermissionAssociations) {
			return types.NewPermissionError("player", cache.CallingPlayer.GetPlayerId(), "player", cache.GetPlayerId(), uint64(types.PermissionAssociations), "guild_register")
		}
	}

	if cache.GetJoinType() != types.GuildJoinType_invite {
		return types.NewGuildMembershipError(cache.GetGuildId(), cache.GetPlayerId(), "wrong_join_type").WithJoinType("invite")
	}

	return nil
}

func (cache *GuildMembershipApplicationCache) ApproveInvite() error {
	cache.GetPlayer().MigrateGuild(cache.GetGuild())
	cache.GetPlayer().MigrateSubstation(cache.GetSubstationId())

	cache.GuildMembershipApplication.RegistrationStatus = types.RegistrationStatus_approved
	cache.GuildMembershipApplicationChanged = true
	cache.Changed()

	return nil
}

func (cache *GuildMembershipApplicationCache) DenyInvite() error {
	cache.GuildMembershipApplication.RegistrationStatus = types.RegistrationStatus_denied
	cache.GuildMembershipApplicationChanged = true
	cache.Changed()

	return nil
}

func (cache *GuildMembershipApplicationCache) RevokeInvite() error {
	cache.GuildMembershipApplication.RegistrationStatus = types.RegistrationStatus_revoked
	cache.GuildMembershipApplicationChanged = true
	cache.Changed()

	return nil
}

func (cache *GuildMembershipApplicationCache) VerifyRequestAsGuild() error {
	guildPermissionError := cache.GetGuild().CanApproveMembershipRequest(cache.CallingPlayer)
	if guildPermissionError != nil {
		return guildPermissionError
	}

	if cache.GetJoinType() != types.GuildJoinType_request {
		return types.NewGuildMembershipError(cache.GetGuildId(), cache.GetPlayerId(), "wrong_join_type").WithJoinType("request")
	}

	return nil
}

func (cache *GuildMembershipApplicationCache) VerifyRequestAsPlayer() error {
	if cache.GetPlayerId() != cache.CallingPlayer.GetPlayerId() {
		if !cache.K.PermissionHasOneOf(cache.Ctx, GetObjectPermissionIDBytes(cache.GetPlayerId(), cache.CallingPlayer.GetPlayerId()), types.PermissionAssociations) {
			return types.NewPermissionError("player", cache.CallingPlayer.GetPlayerId(), "player", cache.GetPlayerId(), uint64(types.PermissionAssociations), "guild_register")
		}
	}

	if cache.GetJoinType() != types.GuildJoinType_request {
		return types.NewGuildMembershipError(cache.GetGuildId(), cache.GetPlayerId(), "wrong_join_type").WithJoinType("request")
	}

	return nil
}

func (cache *GuildMembershipApplicationCache) ApproveRequest() error {
	cache.GetPlayer().MigrateGuild(cache.GetGuild())
	cache.GetPlayer().MigrateSubstation(cache.GetSubstationId())

	cache.GuildMembershipApplication.RegistrationStatus = types.RegistrationStatus_approved
	cache.GuildMembershipApplicationChanged = true
	cache.Changed()

	return nil
}

func (cache *GuildMembershipApplicationCache) DenyRequest() error {
	cache.GuildMembershipApplication.RegistrationStatus = types.RegistrationStatus_denied
	cache.GuildMembershipApplicationChanged = true
	cache.Changed()

	return nil
}

func (cache *GuildMembershipApplicationCache) RevokeRequest() error {
	cache.GuildMembershipApplication.RegistrationStatus = types.RegistrationStatus_revoked
	cache.GuildMembershipApplicationChanged = true
	cache.Changed()

	return nil
}

func (cache *GuildMembershipApplicationCache) Kick() error {
	cache.GetPlayer().LeaveGuild()

	substationPermissionCheck := cache.GetPlayer().GetSubstation().CanManagePlayerConnections(cache.CallingPlayer)
	if substationPermissionCheck == nil {
		cache.GetPlayer().DisconnectSubstation()
	} else if cache.GetPlayer().GetSubstation().GetOwnerId() == cache.GetGuild().GetOwnerId() {
		cache.GetPlayer().DisconnectSubstation()
	}

	cache.GuildMembershipApplicationChanged = true
	cache.Changed()

	return nil
}

func (cache *GuildMembershipApplicationCache) VerifyDirectJoin() error {
	if cache.GetPlayerId() != cache.CallingPlayer.GetPlayerId() {
		if !cache.K.PermissionHasOneOf(cache.Ctx, GetObjectPermissionIDBytes(cache.GetPlayerId(), cache.CallingPlayer.GetPlayerId()), types.PermissionAssociations) {
			return types.NewPermissionError("player", cache.CallingPlayer.GetPlayerId(), "player", cache.GetPlayerId(), uint64(types.PermissionAssociations), "guild_register")
		}
	}
	return nil
}

func (cache *GuildMembershipApplicationCache) DirectJoin() error {

	cache.GetPlayer().MigrateGuild(cache.GetGuild())
	cache.GetPlayer().MigrateSubstation(cache.GetSubstationId())

	cache.GuildMembershipApplication.RegistrationStatus = types.RegistrationStatus_approved
	cache.GuildMembershipApplicationChanged = true
	cache.Changed()

	return nil
}
