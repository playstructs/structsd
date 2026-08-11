package keeper

import (
	errorsmod "cosmossdk.io/errors"

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
            default:
                // registrationStatus is an open proto3 enum, so a value outside
                // the declared four is storable. Log and write nothing, which is
                // what this switch already did in silence. Clearing would be
                // tidier for a phantom row, but it would mean a status added to
                // the proto without a case here deletes live rows, and with
                // GenesisState.Validate rejecting undeclared values on the one
                // path that assigns a whole record, this branch is unreachable —
                // an unreachable branch should not be the one that deletes data.
                cache.CC.k.logger.Error("guild membership application holds an undeclared registration status; not persisting",
                    "guildId", cache.GetGuildId(), "playerId", cache.GetPlayerId(), "registrationStatus", int32(cache.GetRegistrationStatus()))
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
    	guildMembershipApplication, guildMembershipApplicationFound := cache.CC.k.GetGuildMembershipApplicationById(cache.CC.ctx, cache.GuildMembershipApplicationId)

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

		substationPermissionError := substation.CanManageConnectionsBy(cache.CallingPlayer)
		if substationPermissionError != nil {
			return substationPermissionError
		}

		cache.GuildMembershipApplication.SubstationId = substationId
		cache.Changed = true
	}

	return nil
}

// requirePending refuses a terminal transition on anything but a live, stored
// proposal. It is the last line behind GetPendingGuildMembershipApplicationCache,
// so a handler that reaches for the wrong constructor still cannot approve an
// application nobody filed.
//
// GuildMembershipApplicationLoaded is the load-bearing half: only a successful
// store read sets it, so it means precisely "a player or a guild actually filed
// this". The status test is defence in depth and no public path can reach it
// today, because Commit clears every status except proposed and so only proposed
// rows ever persist. Note also that proposed is the zero value, as invite is for
// join type, which means an empty record reads as a proposed invite and the
// status test alone would wave it through. Tests reach the branch by writing a
// row with SetGuildMembershipApplication directly.
func (cache *GuildMembershipApplicationCache) requirePending() error {
	if !cache.GuildMembershipApplicationLoaded {
		return errorsmod.Wrapf(types.ErrGuildMembershipApplication, "no application on file (%s)", cache.GuildMembershipApplicationId)
	}

	if cache.GetRegistrationStatus() != types.RegistrationStatus_proposed {
		return errorsmod.Wrapf(types.ErrGuildMembershipApplication, "application (%s) is no longer pending (registrationStatus %d)", cache.GuildMembershipApplicationId, int32(cache.GetRegistrationStatus()))
	}

	return nil
}

func (cache *GuildMembershipApplicationCache) VerifyInviteAsGuild() error {
	if cache.GetJoinType() != types.GuildJoinType_invite {
		return types.NewGuildMembershipError(cache.GetGuildId(), cache.GetPlayerId(), "wrong_join_type").WithJoinType("invite")
	}

	return cache.GetGuild().CanInviteMembers(cache.CallingPlayer)
}

func (cache *GuildMembershipApplicationCache) VerifyInviteAsPlayer() error {
	if cache.GetJoinType() != types.GuildJoinType_invite {
		return types.NewGuildMembershipError(cache.GetGuildId(), cache.GetPlayerId(), "wrong_join_type").WithJoinType("invite")
	}

    return cache.CC.PermissionCheck(cache.GetPlayer(), cache.CallingPlayer, types.PermGuildMembership)
}

func (cache *GuildMembershipApplicationCache) ApproveInvite() error {
	if err := cache.requirePending(); err != nil {
		return err
	}

	cache.GetPlayer().MigrateGuild(cache.GetGuild())
	cache.GetPlayer().MigrateSubstation(cache.GetSubstationId())

	cache.GuildMembershipApplication.RegistrationStatus = types.RegistrationStatus_approved
	cache.Changed = true

	return nil
}

func (cache *GuildMembershipApplicationCache) DenyInvite() error {
	if err := cache.requirePending(); err != nil {
		return err
	}

	cache.GuildMembershipApplication.RegistrationStatus = types.RegistrationStatus_denied
	cache.Changed = true

	return nil
}

func (cache *GuildMembershipApplicationCache) RevokeInvite() error {
	if err := cache.requirePending(); err != nil {
		return err
	}

	cache.GuildMembershipApplication.RegistrationStatus = types.RegistrationStatus_revoked
	cache.Changed = true
	return nil
}

func (cache *GuildMembershipApplicationCache) VerifyRequestAsGuild() error {
	if cache.GetJoinType() != types.GuildJoinType_request {
		return types.NewGuildMembershipError(cache.GetGuildId(), cache.GetPlayerId(), "wrong_join_type").WithJoinType("request")
	}

	return cache.GetGuild().CanApproveMembershipRequest(cache.CallingPlayer)
}

func (cache *GuildMembershipApplicationCache) VerifyRequestAsPlayer() error {
	if cache.GetJoinType() != types.GuildJoinType_request {
		return types.NewGuildMembershipError(cache.GetGuildId(), cache.GetPlayerId(), "wrong_join_type").WithJoinType("request")
	}

	return cache.CC.PermissionCheck(cache.GetPlayer(), cache.CallingPlayer, types.PermGuildMembership)
}

func (cache *GuildMembershipApplicationCache) ApproveRequest() error {
	if err := cache.requirePending(); err != nil {
		return err
	}

	cache.GetPlayer().MigrateGuild(cache.GetGuild())
	cache.GetPlayer().MigrateSubstation(cache.GetSubstationId())

	cache.GuildMembershipApplication.RegistrationStatus = types.RegistrationStatus_approved
	cache.Changed = true

	return nil
}

func (cache *GuildMembershipApplicationCache) DenyRequest() error {
	if err := cache.requirePending(); err != nil {
		return err
	}

	cache.GuildMembershipApplication.RegistrationStatus = types.RegistrationStatus_denied
	cache.Changed = true

	return nil
}

func (cache *GuildMembershipApplicationCache) RevokeRequest() error {
	if err := cache.requirePending(); err != nil {
		return err
	}

	cache.GuildMembershipApplication.RegistrationStatus = types.RegistrationStatus_revoked
	cache.Changed = true
	return nil
}

// Kick deliberately does not call requirePending. GetGuildMembershipKickCache
// builds its record from scratch every time — a kick is initiated by the guild
// and there is nothing for a member to have filed — and does its own
// authorization: CanKickMembers, plus a rank comparison, plus refusing the owner.
func (cache *GuildMembershipApplicationCache) Kick() error {
	cache.GetPlayer().LeaveGuild()

	substationPermissionCheck := cache.GetPlayer().GetSubstation().CanManageConnectionsBy(cache.CallingPlayer)
	if substationPermissionCheck == nil {
		cache.GetPlayer().DisconnectSubstation()
	}

	cache.Changed = true

	return nil
}

func (cache *GuildMembershipApplicationCache) VerifyDirectJoin() error {
    return cache.CC.PermissionCheck(cache.GetPlayer(), cache.CallingPlayer, types.PermGuildMembership)
}

// DirectJoin deliberately does not call requirePending. GuildMembershipJoin
// creates the application and consumes it in the same handler, so there is never
// a stored row to find, and consent is not in question: VerifyDirectJoin demands
// PermGuildMembership on the joining player themselves.
func (cache *GuildMembershipApplicationCache) DirectJoin() error {

	cache.GetPlayer().MigrateGuild(cache.GetGuild())
	cache.GetPlayer().MigrateSubstation(cache.GetSubstationId())

	cache.GuildMembershipApplication.RegistrationStatus = types.RegistrationStatus_approved
	cache.Changed = true

	return nil
}
