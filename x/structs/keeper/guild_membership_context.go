package keeper

import (
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

// Build this initial Guild Membership Application Cache object
func (cc *CurrentContext) GetGuildMembershipApplicationCache(callingPlayer *PlayerCache, joinType types.GuildJoinType, guildId string, playerId string) (*GuildMembershipApplicationCache, error) {

	targetPlayer, err := cc.GetPlayer(playerId)
	if err != nil {
		return &GuildMembershipApplicationCache{}, types.NewObjectNotFoundError("player", playerId)
	}

	if targetPlayer.GetGuildId() == guildId {
		cc.k.ClearGuildMembershipApplication(cc.ctx, guildId, playerId)
		return &GuildMembershipApplicationCache{}, types.NewGuildMembershipError(guildId, playerId, "already_member")
	}

	guild := cc.GetGuild(guildId)
	if !guild.LoadGuild() {
		return &GuildMembershipApplicationCache{}, types.NewObjectNotFoundError("guild", guildId)
	}

	guildMembershipApplication := cc.GetGuildMembershipApp(guildId, playerId)
    guildMembershipApplicationFound := guildMembershipApplication.LoadGuildMembershipApplication()
    guildMembershipApplication.CallingPlayer = callingPlayer

	if guildMembershipApplicationFound {

		if guildMembershipApplication.GuildMembershipApplication.JoinType != joinType {
			return &GuildMembershipApplicationCache{}, types.NewGuildMembershipError(guildId, playerId, "join_type_mismatch")
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
			return &GuildMembershipApplicationCache{}, guildPermissionError
		}

		guildMembershipApplication.GuildMembershipApplication.Proposer = callingPlayer.GetPlayerId()

		guildMembershipApplication.GuildMembershipApplication.PlayerId = playerId
		guildMembershipApplication.GuildMembershipApplication.GuildId = guildId
		guildMembershipApplication.GuildMembershipApplication.JoinType = joinType
		guildMembershipApplication.GuildMembershipApplication.RegistrationStatus = types.RegistrationStatus_proposed

		guildMembershipApplication.Changed = true
	}

	return guildMembershipApplication, nil
}

func (cc *CurrentContext) GetGuildMembershipKickCache(callingPlayer *PlayerCache, guildId string, playerId string) (*GuildMembershipApplicationCache, error) {

	targetPlayer, err := cc.GetPlayer(playerId)
	if err != nil {
		return &GuildMembershipApplicationCache{}, types.NewObjectNotFoundError("player", playerId)
	}

	if targetPlayer.GetGuildId() != guildId {
		return &GuildMembershipApplicationCache{}, types.NewGuildMembershipError(guildId, playerId, "not_member")
	}

	guild := cc.GetGuild(guildId)
	if !guild.LoadGuild() {
		return &GuildMembershipApplicationCache{}, types.NewObjectNotFoundError("guild", guildId)
	}

	if guild.GetOwnerId() == playerId {
		return &GuildMembershipApplicationCache{}, types.NewGuildMembershipError(guildId, playerId, "cannot_kick_owner")
	}

	guildMembershipApplication := cc.GetGuildMembershipApp(guildId, playerId)
    guildMembershipApplicationFound := guildMembershipApplication.LoadGuildMembershipApplication()
    guildMembershipApplication.CallingPlayer = callingPlayer

	if guildMembershipApplicationFound {
		cc.k.ClearGuildMembershipApplication(cc.ctx, guildId, playerId)
	}

	guildPermissionError := guild.CanKickMembers(callingPlayer)
	if guildPermissionError != nil {
		return &GuildMembershipApplicationCache{}, guildPermissionError
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