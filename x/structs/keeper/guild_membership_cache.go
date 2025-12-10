package keeper

import (
	"context"

	"structs/x/structs/types"
	//sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkerrors "cosmossdk.io/errors"

	// Used in Randomness Orb

	//"cosmossdk.io/math"
    //authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)


type GuildMembershipApplicationCache struct {
    ProposerId string
	GuildId string
	PlayerId string

	K          *Keeper
	Ctx        context.Context

	AnyChange bool
	Ready bool

	GuildMembershipApplicationLoaded  bool
	GuildMembershipApplicationFound   bool
	GuildMembershipApplicationChanged bool
	GuildMembershipApplication        types.GuildMembershipApplication

	GuildLoaded  bool
	Guild        *GuildCache

	PlayerLoaded bool
	Player       *PlayerCache

	ProposerLoaded bool
	Proposer       *PlayerCache

    SubstationLoaded bool
    Substation       *SubstationCache
}

// Build this initial Guild Membership Application Cache object
func (k *Keeper) GetGuildMembershipApplicationCache(ctx context.Context, proposerId string, guildId string, playerId string) GuildMembershipApplicationCache {
	return GuildMembershipApplicationCache{
		ProposerId: proposerId,
		GuildId: guildId,
		PlayerId: playerId,

		K:          k,
		Ctx:        ctx,

		AnyChange: false,

        GuildMembershipApplicationLoaded: false,
        GuildMembershipApplicationFound: false,
        GuildMembershipApplicationChanged: false,

		PlayerLoaded: false,
        ProposerLoaded: false,
		GuildLoaded:  false,
		SubstationLoaded: false,

	}
}

func (cache *GuildMembershipApplicationCache) Commit() {
	cache.AnyChange = false

    cache.K.logger.Info("Updating Guild Membership Application From Cache", "guildId", cache.GuildId)

	if cache.Substation != nil && cache.GetSubstation().IsChanged() {
		cache.GetSubstation().Commit()
	}

	if cache.Player != nil && cache.GetPlayer().IsChanged() {
		cache.GetPlayer().Commit()
	}

    /* These two should never change during this process...
        if cache.Guild != nil && cache.GetGuild().IsChanged() {
            cache.Guild.Commit()
        }

        if cache.Proposer != nil && cache.GetProposer().IsChanged() {
            cache.GetProposer().Commit()
        }
    */

    if cache.GuildMembershipApplicationChanged {
        cache.K.EventGuildMembershipApplication(cache.Ctx, cache.GuildMembershipApplication)

        switch cache.GetRegistrationStatus() {
            case types.RegistrationStatus_proposed:
                cache.K.SetGuildMembershipApplication(cache.Ctx, cache.GuildMembershipApplication)
            case types.RegistrationStatus_approved:
                if cache.GuildMembershipApplicationFound {
                    cache.K.ClearGuildMembershipApplication(cache.Ctx, cache.GuildMembershipApplication)
                }
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

func (cache *GuildMembershipApplicationCache) LoadGuildMembershipApplication() (bool) {
	guildMembershipApplication, guildMembershipApplicationFound := cache.K.GetGuildMembershipApplication(cache.Ctx, cache.GuildId, cache.PlayerId)

    cache.GuildMembershipApplication = guildMembershipApplication
    cache.GuildMembershipApplicationFound = guildMembershipApplicationFound

	if !cache.GuildMembershipApplicationFound {
        cache.GuildMembershipApplication.Proposer   = cache.GetProposer().GetPlayerId()
        cache.GuildMembershipApplication.PlayerId   = cache.GetPlayer().GetPlayerId()
        cache.GuildMembershipApplication.GuildId    = cache.GetGuildId()
        cache.GuildMembershipApplication.JoinType   = types.GuildJoinType_unspecified
        cache.GuildMembershipApplication.RegistrationStatus = types.RegistrationStatus_proposed
	}

	cache.GuildMembershipApplicationLoaded = true

	return cache.GuildMembershipApplicationLoaded
}

// Load the Player data
func (cache *GuildMembershipApplicationCache) LoadPlayer() bool {
	newPlayer, _ := cache.K.GetPlayerCacheFromId(cache.Ctx, cache.GetPlayerId())
	cache.Player = &newPlayer
	cache.PlayerLoaded = true
	return cache.PlayerLoaded
}

func (cache *GuildMembershipApplicationCache) ManualLoadPlayer(player *PlayerCache) {
    cache.Player = player
    cache.PlayerLoaded = true
}

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


// Load the Guild record
func (cache *GuildMembershipApplicationCache) LoadGuild() (bool) {
	newGuild := cache.K.GetGuildCacheFromId(cache.Ctx, cache.GuildId)

	if newGuild.LoadGuild() {
		cache.Guild = &newGuild
		cache.GuildLoaded = true
	}

	return cache.GuildLoaded
}

func (cache *GuildMembershipApplicationCache) ManualLoadGuild(guild *GuildCache) {
    cache.Guild = guild
    cache.GuildLoaded = true
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
func (cache *GuildMembershipApplicationCache) GetGuildMembershipApplication() types.GuildMembershipApplication { if !cache.GuildMembershipApplicationLoaded { cache.LoadGuildMembershipApplication() }; return cache.GuildMembershipApplication }
func (cache *GuildMembershipApplicationCache) GetRegistrationStatus() types.RegistrationStatus { return cache.GetGuildMembershipApplication().RegistrationStatus }
func (cache *GuildMembershipApplicationCache) GetJoinType() types.GuildJoinType { return cache.GetGuildMembershipApplication().JoinType }

func (cache *GuildMembershipApplicationCache) IsGuildMembershipApplicationFound() bool { if !cache.GuildMembershipApplicationLoaded { cache.LoadGuildMembershipApplication() }; return cache.GuildMembershipApplicationFound }

func (cache *GuildMembershipApplicationCache) GetGuildId() string { return cache.GuildId }
func (cache *GuildMembershipApplicationCache) GetGuild() *GuildCache { if !cache.GuildLoaded { cache.LoadGuild() }; return cache.Guild }

// Get the Player data
func (cache *GuildMembershipApplicationCache) GetPlayerId() string { return cache.PlayerId }
func (cache *GuildMembershipApplicationCache) GetPlayer() *PlayerCache { if !cache.PlayerLoaded { cache.LoadPlayer() }; return cache.Player }

// Get the Proposer data
func (cache *GuildMembershipApplicationCache) GetProposerId() string { return cache.ProposerId }
func (cache *GuildMembershipApplicationCache) GetProposer() *PlayerCache { if !cache.ProposerLoaded { cache.LoadProposer() }; return cache.Proposer }

// Get the Proposer data
func (cache *GuildMembershipApplicationCache) GetSubstationId() string { if !cache.GuildMembershipApplicationLoaded { cache.LoadGuildMembershipApplication() }; return cache.GuildMembershipApplication.SubstationId }
func (cache *GuildMembershipApplicationCache) GetSubstation() *SubstationCache { if !cache.SubstationLoaded { cache.LoadSubstation() }; return cache.Substation }


func (cache *GuildMembershipApplicationCache) SetRegistrationStatus(registrationStatus types.RegistrationStatus) {
    if !cache.GuildMembershipApplicationLoaded {
        cache.LoadGuildMembershipApplication()
    }

    if cache.GuildMembershipApplication.RegistrationStatus != registrationStatus {
        cache.GuildMembershipApplication.RegistrationStatus = registrationStatus
        cache.GuildMembershipApplicationChanged = true
        cache.Changed()

        if registrationStatus == types.RegistrationStatus_approved {

           // TODO Fix or move
            //cache.GetPlayer().MigrateGuild(cache.GetGuild())

        }
    }
}


func (cache *GuildMembershipApplicationCache) SetJoinType(joinType types.GuildJoinType)  {
    if !cache.GuildMembershipApplicationLoaded {
        cache.LoadGuildMembershipApplication()
    }

    cache.GuildMembershipApplication.JoinType = joinType
}

func (cache *GuildMembershipApplicationCache) SetSubstationId(substationId string)  {
    if !cache.GuildMembershipApplicationLoaded {
        cache.LoadGuildMembershipApplication()
    }

    if cache.GuildMembershipApplication.SubstationId != substationId {
        cache.GuildMembershipApplication.SubstationId = substationId
        cache.GuildMembershipApplicationChanged = true
        cache.Changed()
    }
}
