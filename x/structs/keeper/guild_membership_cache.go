package keeper

import (


	"structs/x/structs/types"
	//sdk "github.com/cosmos/cosmos-sdk/types"
	// Used in Randomness Orb
	//"cosmossdk.io/math"
	//authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

type GuildMembershipApplicationCache struct {
    GuildMembershipApplicationId string
	CC  *CurrentContext

	Changed bool
	Ready   bool

	GuildMembershipApplication          types.GuildMembershipApplication
	GuildMembershipApplicationLoaded    bool

	CallingPlayer *PlayerCache
}


func (cache *GuildMembershipApplicationCache) Commit() {
	if cache.Changed {

    	cache.CC.k.logger.Info("Updating Guild Membership Application From Cache", "guildId", cache.GetGuildMembershipApplication().GuildId, "playerId", cache.GetGuildMembershipApplication().PlayerId)
		cache.CC.k.EventGuildMembershipApplication(cache.CC.ctx, cache.GuildMembershipApplication)

		switch cache.GetRegistrationStatus() {
            case types.RegistrationStatus_proposed:
                cache.CC.k.SetGuildMembershipApplication(cache.CC.ctx, cache.GuildMembershipApplication)
            case types.RegistrationStatus_approved:
                cache.CC.k.ClearGuildMembershipApplication(cache.CC.ctx, cache.GetGuildId(), cache.GetPlayerId())
            case types.RegistrationStatus_denied:
                cache.CC.k.ClearGuildMembershipApplication(cache.CC.ctx, cache.GetGuildId(), cache.GetPlayerId())
            case types.RegistrationStatus_revoked:
                cache.CC.k.ClearGuildMembershipApplication(cache.CC.ctx, cache.GetGuildId(), cache.GetPlayerId())
		}
	}
	cache.Changed = false
}

func (cache *GuildMembershipApplicationCache) IsChanged() bool {
	return cache.Changed
}

func (cache *GuildMembershipApplicationCache) ID() string {
	return cache.GuildMembershipApplication.PlayerId + "@" + cache.GuildMembershipApplication.GuildId
}

func (cache *GuildMembershipApplicationCache) LoadGuildMembershipApplication() bool {
    	guildMembershipApplication, guildMembershipApplicationFound := cache.CC.k.GetGuildMembershipApplicationById(cache.CC.ctx, cache.ID())

    	if guildMembershipApplicationFound {
    		cache.GuildMembershipApplication = guildMembershipApplication
    		cache.GuildMembershipApplicationLoaded = true
    	}

    	return cache.GuildMembershipApplicationLoaded
}

/* Separate Loading functions for each of the underlying containers */

/* Getters
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */
func (cache *GuildMembershipApplicationCache) GetGuildMembershipApplication() types.GuildMembershipApplication {
	if !cache.GuildMembershipApplicationLoaded {
	    cache.LoadGuildMembershipApplication()
	}

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
func (cache *GuildMembershipApplicationCache) GetGuild() *GuildCache {
    return cache.CC.GetGuild(cache.GetGuildId())
}

// Get the Player data
func (cache *GuildMembershipApplicationCache) GetPlayerId() string {
	return cache.GetGuildMembershipApplication().PlayerId
}

func (cache *GuildMembershipApplicationCache) GetPlayer() (player *PlayerCache) {
    player, _ = cache.CC.GetPlayer(cache.GetPlayerId())
    return
}

// Get the Proposer data
func (cache *GuildMembershipApplicationCache) GetProposerId() string {
	return cache.GetGuildMembershipApplication().Proposer
}
func (cache *GuildMembershipApplicationCache) GetProposer() (player *PlayerCache) {
    player, _ = cache.CC.GetPlayer(cache.GetProposerId())
	return
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

		substation := cache.CC.GetSubstation(substationId)
		if !substation.LoadSubstation() {
			return types.NewObjectNotFoundError("substation", substationId)
		}

		substationPermissionError := substation.CanManagePlayerConnections(cache.CallingPlayer)
		if substationPermissionError != nil {
			return substationPermissionError
		}

		cache.GuildMembershipApplication.SubstationId = substationId
		cache.Changed = true
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
		if !cache.CC.PermissionHasOneOf(GetObjectPermissionIDBytes(cache.GetPlayerId(), cache.CallingPlayer.GetPlayerId()), types.PermissionAssociations) {
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
	cache.Changed = true

	return nil
}

func (cache *GuildMembershipApplicationCache) DenyInvite() error {
	cache.GuildMembershipApplication.RegistrationStatus = types.RegistrationStatus_denied
	cache.Changed = true

	return nil
}

func (cache *GuildMembershipApplicationCache) RevokeInvite() error {
	cache.GuildMembershipApplication.RegistrationStatus = types.RegistrationStatus_revoked
	cache.Changed = true
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
		if !cache.CC.PermissionHasOneOf(GetObjectPermissionIDBytes(cache.GetPlayerId(), cache.CallingPlayer.GetPlayerId()), types.PermissionAssociations) {
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
	cache.Changed = true

	return nil
}

func (cache *GuildMembershipApplicationCache) DenyRequest() error {
	cache.GuildMembershipApplication.RegistrationStatus = types.RegistrationStatus_denied
	cache.Changed = true

	return nil
}

func (cache *GuildMembershipApplicationCache) RevokeRequest() error {
	cache.GuildMembershipApplication.RegistrationStatus = types.RegistrationStatus_revoked
	cache.Changed = true
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

	cache.Changed = true

	return nil
}

func (cache *GuildMembershipApplicationCache) VerifyDirectJoin() error {
	if cache.GetPlayerId() != cache.CallingPlayer.GetPlayerId() {
		if !cache.CC.PermissionHasOneOf(GetObjectPermissionIDBytes(cache.GetPlayerId(), cache.CallingPlayer.GetPlayerId()), types.PermissionAssociations) {
			return types.NewPermissionError("player", cache.CallingPlayer.GetPlayerId(), "player", cache.GetPlayerId(), uint64(types.PermissionAssociations), "guild_register")
		}
	}
	return nil
}

func (cache *GuildMembershipApplicationCache) DirectJoin() error {

	cache.GetPlayer().MigrateGuild(cache.GetGuild())
	cache.GetPlayer().MigrateSubstation(cache.GetSubstationId())

	cache.GuildMembershipApplication.RegistrationStatus = types.RegistrationStatus_approved
	cache.Changed = true

	return nil
}
