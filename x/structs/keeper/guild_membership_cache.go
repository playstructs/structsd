package keeper

import (
	"context"

	"structs/x/structs/types"
	//sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"

	// Used in Randomness Orb

	//"cosmossdk.io/math"
    //authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)


// TODO Proposer is actually different than calling

type GuildMembershipApplicationCache struct {
	K          *Keeper
	Ctx        context.Context

	AnyChange bool
	Ready bool

	GuildMembershipApplicationChanged bool
	GuildMembershipApplication        types.GuildMembershipApplication

	CallingPlayer       *PlayerCache

	Guild        *GuildCache
	Player       *PlayerCache

	ProposerLoaded bool
	Proposer       *PlayerCache

    SubstationLoaded bool
    Substation       *SubstationCache
}

// Build this initial Guild Membership Application Cache object
func (k *Keeper) GetGuildMembershipApplicationCache(ctx context.Context, callingPlayer *PlayerCache, joinType types.GuildJoinType, guildId string, playerId string) (GuildMembershipApplicationCache, error) {

    targetPlayer, err := k.GetPlayerCacheFromId(ctx, playerId)
    if err != nil {
        return GuildMembershipApplicationCache{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Player (%s) not found", playerId)
    }

    if targetPlayer.GetGuildId() == guildId {
        return GuildMembershipApplicationCache{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Player (%s) already a member of Guild (%s)", playerId, guildId)
    }

    guild := k.GetGuildCacheFromId(ctx, guildId)
    if !guild.LoadGuild() {
        return GuildMembershipApplicationCache{}, sdkerrors.Wrapf(types.ErrObjectNotFound, "Guild (%s) not found", guildId)
    }

    guildMembershipApplication, guildMembershipApplicationFound := k.GetGuildMembershipApplication(ctx, guildId, playerId)

    guildMembershipApplicationChanged := false
    proposerLoaded := false
    var proposer *PlayerCache

    if guildMembershipApplicationFound {

        if guildMembershipApplication.JoinType != joinType {
            return GuildMembershipApplicationCache{}, sdkerrors.Wrapf(types.ErrGuildMembershipApplication, "Application cannot change join type")
        }

    } else {

        var guildPermissionError error
        switch guildMembershipApplication.JoinType {
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

        guildMembershipApplication.Proposer   = callingPlayer.GetPlayerId()
        proposer = callingPlayer
        proposerLoaded = true

        guildMembershipApplication.PlayerId   = playerId
        guildMembershipApplication.GuildId    = guildId
        guildMembershipApplication.JoinType   = joinType
        guildMembershipApplication.RegistrationStatus = types.RegistrationStatus_proposed

        guildMembershipApplicationChanged = true
    }


	return GuildMembershipApplicationCache{
		CallingPlayer: callingPlayer,

		K:          k,
		Ctx:        ctx,

		AnyChange: false,

        GuildMembershipApplication: guildMembershipApplication,
        GuildMembershipApplicationChanged: guildMembershipApplicationChanged,

        Player: &targetPlayer,

        Proposer: proposer,
        ProposerLoaded: proposerLoaded,

		Guild:  &guild,

		SubstationLoaded: false,

	}, nil
}

func (cache *GuildMembershipApplicationCache) Commit() {
	cache.AnyChange = false

    cache.K.logger.Info("Updating Guild Membership Application From Cache", "guildId", cache.GetGuildMembershipApplication().GuildId, "playerId", cache.GetGuildMembershipApplication().PlayerId)

	if cache.Substation != nil && cache.GetSubstation().IsChanged() {
		cache.GetSubstation().Commit()
	}

	if cache.Player != nil && cache.GetPlayer().IsChanged() {
		cache.GetPlayer().Commit()
	}

    if cache.GuildMembershipApplicationChanged {
        cache.K.EventGuildMembershipApplication(cache.Ctx, cache.GuildMembershipApplication)

        switch cache.GetRegistrationStatus() {
            case types.RegistrationStatus_proposed:
                cache.K.SetGuildMembershipApplication(cache.Ctx, cache.GuildMembershipApplication)
            case types.RegistrationStatus_approved:
                cache.K.ClearGuildMembershipApplication(cache.Ctx, cache.GuildMembershipApplication)
            case types.RegistrationStatus_denied:
                cache.K.ClearGuildMembershipApplication(cache.Ctx, cache.GuildMembershipApplication)
            case types.RegistrationStatus_revoked:
                cache.K.ClearGuildMembershipApplication(cache.Ctx, cache.GuildMembershipApplication)
        }
    }

}

func (cache *GuildMembershipApplicationCache) IsChanged() bool {
	return cache.AnyChange
}

func (cache *GuildMembershipApplicationCache) Changed() {
	cache.AnyChange = true
}

/* Separate Loading functions for each of the underlying containers */

// Load the Proposer Player data
func (cache *GuildMembershipApplicationCache) LoadProposer() bool {
	newProposer, _ := cache.K.GetPlayerCacheFromId(cache.Ctx, cache.GetProposerId())
	cache.Proposer = &newProposer
	cache.ProposerLoaded = true
	return cache.ProposerLoaded
}

func (cache *GuildMembershipApplicationCache) ManualLoadProposer(player *PlayerCache) {
    cache.Proposer = player
    cache.ProposerLoaded = true
}

// Load the Substation data
func (cache *GuildMembershipApplicationCache) LoadSubstation() bool {
	newSubstation := cache.K.GetSubstationCacheFromId(cache.Ctx, cache.GetSubstationId())
	cache.Substation = &newSubstation
	cache.SubstationLoaded = cache.Substation.LoadSubstation()

	return cache.SubstationLoaded
}

func (cache *GuildMembershipApplicationCache) ManualLoadSubstation(substation *SubstationCache) {
    cache.Substation = substation

    if cache.Substation.SubstationLoaded {
       cache.SubstationLoaded = true
    } else {
       cache.SubstationLoaded = cache.Substation.LoadSubstation()
    }

}

/* Getters
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */
func (cache *GuildMembershipApplicationCache) GetGuildMembershipApplication() types.GuildMembershipApplication { return cache.GuildMembershipApplication }
func (cache *GuildMembershipApplicationCache) GetRegistrationStatus() types.RegistrationStatus { return cache.GetGuildMembershipApplication().RegistrationStatus }
func (cache *GuildMembershipApplicationCache) GetJoinType() types.GuildJoinType { return cache.GetGuildMembershipApplication().JoinType }

func (cache *GuildMembershipApplicationCache) GetGuildId() string { return cache.GetGuildMembershipApplication().GuildId }
func (cache *GuildMembershipApplicationCache) GetGuild() *GuildCache { return cache.Guild }

// Get the Player data
func (cache *GuildMembershipApplicationCache) GetPlayerId() string { return cache.GetGuildMembershipApplication().PlayerId }
func (cache *GuildMembershipApplicationCache) GetPlayer() *PlayerCache { return cache.Player }

// Get the Proposer data
func (cache *GuildMembershipApplicationCache) GetProposerId() string { return cache.GetGuildMembershipApplication().Proposer }
func (cache *GuildMembershipApplicationCache) GetProposer() *PlayerCache { if !cache.ProposerLoaded { cache.LoadProposer() }; return cache.Proposer }

// Get the Proposer data
func (cache *GuildMembershipApplicationCache) GetSubstationId() string { return cache.GetGuildMembershipApplication().SubstationId }
func (cache *GuildMembershipApplicationCache) GetSubstation() *SubstationCache { if !cache.SubstationLoaded { cache.LoadSubstation() }; return cache.Substation }


func (cache *GuildMembershipApplicationCache) SetSubstationIdOverride(substationId string) (error) {

    if cache.GuildMembershipApplication.SubstationId != substationId {

        substation := cache.K.GetSubstationCacheFromId(cache.Ctx, substationId)
        if !substation.LoadSubstation() {
            return sdkerrors.Wrapf(types.ErrObjectNotFound, "Substation (%s) not found", substationId)
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
