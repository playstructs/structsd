package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"structs/x/structs/types"
)

const (
	StructsMessageTypeURLPrefix = "/structs.structs.Msg"
	MsgUpdateParamsTypeURL      = "/structs.structs.MsgUpdateParams"
)

// KnownStructsMessages is the complete set of registered Structs message type
// URLs. Any message matching StructsMessageTypeURLPrefix but absent from this
// set is rejected by the StructsDecorator (deny-by-default).
var KnownStructsMessages = map[string]bool{
	"/structs.structs.MsgAddressRegister":                                true,
	"/structs.structs.MsgAddressRevoke":                                  true,
	"/structs.structs.MsgAgreementCapacityDecrease":                      true,
	"/structs.structs.MsgAgreementCapacityIncrease":                      true,
	"/structs.structs.MsgAgreementClose":                                 true,
	"/structs.structs.MsgAgreementDurationIncrease":                      true,
	"/structs.structs.MsgAgreementOpen":                                  true,
	"/structs.structs.MsgAllocationCreate":                               true,
	"/structs.structs.MsgAllocationDelete":                               true,
	"/structs.structs.MsgAllocationTransfer":                             true,
	"/structs.structs.MsgAllocationUpdate":                               true,
	"/structs.structs.MsgFleetMove":                                      true,
	"/structs.structs.MsgGuildBankConfiscateAndBurn":                     true,
	"/structs.structs.MsgGuildBankConvert":                               true,
	"/structs.structs.MsgGuildBankConvertToken":                          true,
	"/structs.structs.MsgGuildBankMint":                                  true,
	"/structs.structs.MsgGuildBankRedeem":                                true,
	"/structs.structs.MsgGuildCreate":                                    true,
	"/structs.structs.MsgGuildMembershipInvite":                          true,
	"/structs.structs.MsgGuildMembershipInviteApprove":                   true,
	"/structs.structs.MsgGuildMembershipInviteDeny":                      true,
	"/structs.structs.MsgGuildMembershipInviteRevoke":                    true,
	"/structs.structs.MsgGuildMembershipJoin":                            true,
	"/structs.structs.MsgGuildMembershipJoinProxy":                       true,
	"/structs.structs.MsgGuildMembershipKick":                            true,
	"/structs.structs.MsgGuildMembershipRequest":                         true,
	"/structs.structs.MsgGuildMembershipRequestApprove":                  true,
	"/structs.structs.MsgGuildMembershipRequestDeny":                     true,
	"/structs.structs.MsgGuildMembershipRequestRevoke":                   true,
	"/structs.structs.MsgGuildUpdateBankConvertInFee":                    true,
	"/structs.structs.MsgGuildUpdateBankConvertOutFee":                   true,
	"/structs.structs.MsgGuildUpdateEndpoint":                            true,
	"/structs.structs.MsgGuildUpdateEntryRank":                           true,
	"/structs.structs.MsgGuildUpdateEntrySubstationId":                   true,
	"/structs.structs.MsgGuildUpdateJoinInfusionMinimum":                 true,
	"/structs.structs.MsgGuildUpdateJoinInfusionMinimumBypassByInvite":   true,
	"/structs.structs.MsgGuildUpdateJoinInfusionMinimumBypassByRequest":  true,
	"/structs.structs.MsgGuildUpdateName":                                true,
	"/structs.structs.MsgGuildUpdatePfp":                                 true,
	"/structs.structs.MsgGuildUpdateOwnerId":                             true,
	"/structs.structs.MsgGuildUpdatePrimaryReactor":                      true,
	"/structs.structs.MsgPermissionGrantOnAddress":                       true,
	"/structs.structs.MsgPermissionGrantOnObject":                        true,
	"/structs.structs.MsgPermissionGuildRankRevoke":                      true,
	"/structs.structs.MsgPermissionGuildRankSet":                         true,
	"/structs.structs.MsgPermissionRevokeOnAddress":                      true,
	"/structs.structs.MsgPermissionRevokeOnObject":                       true,
	"/structs.structs.MsgPermissionSetOnAddress":                         true,
	"/structs.structs.MsgPermissionSetOnObject":                          true,
	"/structs.structs.MsgPlanetExplore":                                  true,
	"/structs.structs.MsgPlanetRaidComplete":                             true,
	"/structs.structs.MsgPlanetUpdateName":                               true,
	"/structs.structs.MsgPlayerSend":                                     true,
	"/structs.structs.MsgPlayerUpdateGuildRank":                          true,
	"/structs.structs.MsgPlayerUpdateName":                               true,
	"/structs.structs.MsgPlayerUpdatePfp":                                true,
	"/structs.structs.MsgPlayerUpdatePfpClientRenderAttributes":          true,
	"/structs.structs.MsgPlayerUpdatePrimaryAddress":                     true,
	"/structs.structs.MsgProviderCreate":                                 true,
	"/structs.structs.MsgProviderDelete":                                 true,
	"/structs.structs.MsgProviderUpdateAccessPolicy":                     true,
	"/structs.structs.MsgProviderUpdateCapacityMaximum":                  true,
	"/structs.structs.MsgProviderUpdateCapacityMinimum":                  true,
	"/structs.structs.MsgProviderUpdateDurationMaximum":                  true,
	"/structs.structs.MsgProviderUpdateDurationMinimum":                  true,
	"/structs.structs.MsgProviderWithdrawBalance":                        true,
	"/structs.structs.MsgReactorBeginMigration":                          true,
	"/structs.structs.MsgReactorCancelDefusion":                          true,
	"/structs.structs.MsgReactorDefuse":                                  true,
	"/structs.structs.MsgReactorInfuse":                                  true,
	"/structs.structs.MsgReactorRestart":                                 true,
	"/structs.structs.MsgStructActivate":                                 true,
	"/structs.structs.MsgStructAttack":                                   true,
	"/structs.structs.MsgStructBuildCancel":                              true,
	"/structs.structs.MsgStructBuildComplete":                            true,
	"/structs.structs.MsgStructBuildInitiate":                            true,
	"/structs.structs.MsgStructDeactivate":                               true,
	"/structs.structs.MsgStructDeactivateBatch":                          true,
	"/structs.structs.MsgStructDefenseClear":                             true,
	"/structs.structs.MsgStructDefenseSet":                               true,
	"/structs.structs.MsgStructGeneratorInfuse":                          true,
	"/structs.structs.MsgStructMove":                                     true,
	"/structs.structs.MsgStructOreMinerComplete":                         true,
	"/structs.structs.MsgStructOreRefineryComplete":                      true,
	"/structs.structs.MsgStructStealthActivate":                          true,
	"/structs.structs.MsgStructStealthDeactivate":                        true,
	"/structs.structs.MsgStructTrash":                                    true,
	"/structs.structs.MsgSubstationAllocationConnect":                    true,
	"/structs.structs.MsgSubstationAllocationDisconnect":                 true,
	"/structs.structs.MsgSubstationCreate":                               true,
	"/structs.structs.MsgSubstationDelete":                               true,
	"/structs.structs.MsgSubstationUpdateName":                           true,
	"/structs.structs.MsgSubstationUpdatePfp":                            true,
	"/structs.structs.MsgSubstationPlayerConnect":                        true,
	"/structs.structs.MsgSubstationPlayerDisconnect":                     true,
	"/structs.structs.MsgSubstationPlayerMigrate":                        true,
}

// PermissionMap maps Structs message type URLs to the address-level permission
// bits required (Layer 1 only). Messages absent from this map have dynamic or
// policy-dependent permissions -- those skip the ante permission check and rely
// on handler-level enforcement.
var PermissionMap = map[string]types.Permission{
	// Gameplay actions
	"/structs.structs.MsgFleetMove":              types.PermPlay,
	"/structs.structs.MsgPlanetExplore":          types.PermPlay,
	"/structs.structs.MsgStructActivate":         types.PermPlay,
	"/structs.structs.MsgStructAttack":           types.PermPlay,
	"/structs.structs.MsgStructBuildCancel":      types.PermPlay,
	"/structs.structs.MsgStructBuildInitiate":    types.PermPlay,
	"/structs.structs.MsgStructDeactivate":       types.PermPlay,
	"/structs.structs.MsgStructDeactivateBatch":  types.PermPlay,
	"/structs.structs.MsgStructDefenseClear":     types.PermPlay,
	"/structs.structs.MsgStructDefenseSet":       types.PermPlay,
	"/structs.structs.MsgStructMove":             types.PermPlay,
	"/structs.structs.MsgStructStealthActivate":  types.PermPlay,
	"/structs.structs.MsgStructStealthDeactivate": types.PermPlay,
	"/structs.structs.MsgStructTrash":            types.PermPlay,

	// Proof-of-work actions (require specific hash permissions)
	"/structs.structs.MsgStructBuildComplete":      types.PermHashBuild,
	"/structs.structs.MsgStructOreMinerComplete":   types.PermHashMine,
	"/structs.structs.MsgStructOreRefineryComplete": types.PermHashRefine,
	"/structs.structs.MsgPlanetRaidComplete":       types.PermHashRaid,

	// Token operations
	"/structs.structs.MsgGuildBankRedeem":      types.PermTokenTransfer,
	"/structs.structs.MsgGuildBankConvert":     types.PermTokenTransfer,
	"/structs.structs.MsgGuildBankConvertToken": types.PermTokenTransfer,
	"/structs.structs.MsgPlayerSend":           types.PermTokenTransfer,
	// Opening an agreement debits the player's primary address for the collateral.
	// The access policy is dynamic and stays in the handler, but the spend bit is
	// required under every policy, so it belongs here as the hard ceiling.
	"/structs.structs.MsgAgreementOpen":        types.PermTokenTransfer,
	"/structs.structs.MsgReactorInfuse":        types.PermTokenInfuse,
	"/structs.structs.MsgReactorCancelDefusion": types.PermTokenInfuse,
	"/structs.structs.MsgStructGeneratorInfuse": types.PermTokenInfuse,
	"/structs.structs.MsgReactorDefuse":        types.PermTokenDefuse,
	"/structs.structs.MsgReactorBeginMigration": types.PermTokenMigrate,

	// Guild banking
	"/structs.structs.MsgGuildBankConfiscateAndBurn": types.PermGuildTokenBurn,
	"/structs.structs.MsgGuildBankMint":              types.PermGuildTokenMint,

	// Guild settings
	"/structs.structs.MsgGuildUpdateEndpoint":                            types.PermGuildEndpointUpdate,
	"/structs.structs.MsgGuildUpdateJoinInfusionMinimum":                 types.PermGuildJoinConstraintsUpdate,
	"/structs.structs.MsgGuildUpdateJoinInfusionMinimumBypassByInvite":   types.PermGuildJoinConstraintsUpdate,
	"/structs.structs.MsgGuildUpdateJoinInfusionMinimumBypassByRequest":  types.PermGuildJoinConstraintsUpdate,
	"/structs.structs.MsgGuildUpdateEntrySubstationId":                   types.PermGuildSubstationUpdate,

	// Guild membership (fixed permission, not policy-dependent)
	"/structs.structs.MsgGuildMembershipJoin":             types.PermGuildMembership,
	"/structs.structs.MsgGuildMembershipInviteApprove":    types.PermGuildMembership,
	"/structs.structs.MsgGuildMembershipInviteDeny":       types.PermGuildMembership,
	"/structs.structs.MsgGuildMembershipRequestRevoke":    types.PermGuildMembership,

	// Admin operations
	"/structs.structs.MsgAllocationTransfer":           types.PermAdmin,
	"/structs.structs.MsgGuildUpdateOwnerId":           types.PermAdmin,
	"/structs.structs.MsgGuildUpdatePrimaryReactor":    types.PermAdmin,
	"/structs.structs.MsgGuildUpdateBankConvertInFee":  types.PermAdmin,
	"/structs.structs.MsgGuildUpdateBankConvertOutFee": types.PermAdmin,
	"/structs.structs.MsgGuildUpdateEntryRank":        types.PermUpdate,

	// Primary address swap grants PermAll to the incoming address and moves
	// balances/delegations with it, so the caller must already hold every bit.
	"/structs.structs.MsgPlayerUpdatePrimaryAddress": types.PermAll,

	// Object updates
	"/structs.structs.MsgAgreementCapacityDecrease":   types.PermUpdate,
	"/structs.structs.MsgAgreementCapacityIncrease":   types.PermUpdate,
	"/structs.structs.MsgAgreementClose":              types.PermUpdate,
	// Extending a duration buys the extra blocks out of the player's primary
	// address, so update rights on the agreement are not enough on their own.
	"/structs.structs.MsgAgreementDurationIncrease":   types.PermUpdate | types.PermTokenTransfer,
	"/structs.structs.MsgProviderUpdateAccessPolicy":  types.PermUpdate,
	"/structs.structs.MsgProviderUpdateCapacityMaximum": types.PermUpdate,
	"/structs.structs.MsgProviderUpdateCapacityMinimum": types.PermUpdate,
	"/structs.structs.MsgProviderUpdateDurationMaximum": types.PermUpdate,
	"/structs.structs.MsgProviderUpdateDurationMinimum": types.PermUpdate,

	// Object deletion
	"/structs.structs.MsgAddressRevoke":    types.PermDelete,
	"/structs.structs.MsgProviderDelete":   types.PermDelete,
	"/structs.structs.MsgSubstationDelete": types.PermDelete,

	// Source allocation
	"/structs.structs.MsgAllocationCreate":  types.PermSourceAllocation,
	"/structs.structs.MsgAllocationUpdate":  types.PermSourceAllocation,
	"/structs.structs.MsgAllocationDelete":  types.PermSourceAllocation,
	"/structs.structs.MsgProviderCreate":    types.PermSourceAllocation,

	// Connection management
	"/structs.structs.MsgSubstationAllocationConnect":    types.PermAllocationConnection,
	"/structs.structs.MsgSubstationAllocationDisconnect": types.PermAllocationConnection,
	"/structs.structs.MsgSubstationCreate":               types.PermAllocationConnection,
	"/structs.structs.MsgSubstationPlayerConnect":        types.PermSubstationConnection,
	"/structs.structs.MsgSubstationPlayerDisconnect":     types.PermSubstationConnection,
	"/structs.structs.MsgSubstationPlayerMigrate":        types.PermSubstationConnection,

	// Provider
	"/structs.structs.MsgProviderWithdrawBalance": types.PermProviderWithdraw,

	// Reactor
	"/structs.structs.MsgGuildCreate": types.PermReactorGuildCreate,
}

// DynamicPermissionMessages are messages where the required permission bits
// come from the message fields themselves or depend on runtime policy. These
// skip the ante-level permission check; the handler enforces the full check.
// The ante handler still verifies the address is registered as a player.
var DynamicPermissionMessages = map[string]bool{
	"/structs.structs.MsgAddressRegister":                       true,
	"/structs.structs.MsgGuildMembershipJoinProxy":              true,
	"/structs.structs.MsgPermissionGrantOnAddress":              true,
	"/structs.structs.MsgPermissionGrantOnObject":               true,
	"/structs.structs.MsgPermissionGuildRankRevoke":             true,
	"/structs.structs.MsgPermissionGuildRankSet":                true,
	"/structs.structs.MsgPermissionRevokeOnAddress":             true,
	"/structs.structs.MsgPermissionRevokeOnObject":              true,
	"/structs.structs.MsgPermissionSetOnAddress":                true,
	"/structs.structs.MsgPermissionSetOnObject":                 true,
	"/structs.structs.MsgPlayerUpdateGuildRank":                 true,
	"/structs.structs.MsgGuildMembershipInvite":                 true,
	"/structs.structs.MsgGuildMembershipInviteRevoke":           true,
	"/structs.structs.MsgGuildMembershipKick":                   true,
	"/structs.structs.MsgGuildMembershipRequest":                true,
	"/structs.structs.MsgGuildMembershipRequestApprove":         true,
	"/structs.structs.MsgGuildMembershipRequestDeny":            true,
	"/structs.structs.MsgGuildUpdateName":                       true,
	"/structs.structs.MsgGuildUpdatePfp":                        true,
	"/structs.structs.MsgPlayerUpdateName":                      true,
	"/structs.structs.MsgPlayerUpdatePfp":                       true,
	"/structs.structs.MsgPlayerUpdatePfpClientRenderAttributes": true,
	"/structs.structs.MsgSubstationUpdateName":                  true,
	"/structs.structs.MsgSubstationUpdatePfp":                   true,
	"/structs.structs.MsgPlanetUpdateName":                      true,

	// MsgReactorRestart requires no permission at all. It only writes state
	// derived from the staking module, so any player may reconcile any reactor;
	// the ante-level player registration check is the whole gate.
	"/structs.structs.MsgReactorRestart":                        true,
}

// ChargeMessages are messages that check charge (blockHeight - lastAction) in
// their handlers. The ante handler verifies charge > 0 as an early rejection
// for same-block double-actions.
var ChargeMessages = map[string]bool{
	"/structs.structs.MsgStructActivate":         true,
	"/structs.structs.MsgStructAttack":           true,
	"/structs.structs.MsgStructBuildInitiate":    true,
	"/structs.structs.MsgStructDefenseClear":     true,
	"/structs.structs.MsgStructDefenseSet":       true,
	"/structs.structs.MsgStructMove":             true,
	"/structs.structs.MsgStructStealthActivate":  true,
	"/structs.structs.MsgStructStealthDeactivate": true,
}

// ProofMessages are PoW messages that get per-object throttling via transient
// store. The extractor returns the object ID used as the throttle key.
var ProofMessages = map[string]func(sdk.Msg) string{
	"/structs.structs.MsgStructBuildComplete": func(msg sdk.Msg) string {
		if m, ok := msg.(*types.MsgStructBuildComplete); ok {
			return m.StructId
		}
		return ""
	},
	"/structs.structs.MsgStructOreMinerComplete": func(msg sdk.Msg) string {
		if m, ok := msg.(*types.MsgStructOreMinerComplete); ok {
			return m.StructId
		}
		return ""
	},
	"/structs.structs.MsgStructOreRefineryComplete": func(msg sdk.Msg) string {
		if m, ok := msg.(*types.MsgStructOreRefineryComplete); ok {
			return m.StructId
		}
		return ""
	},
	"/structs.structs.MsgPlanetRaidComplete": func(msg sdk.Msg) string {
		if m, ok := msg.(*types.MsgPlanetRaidComplete); ok {
			return m.FleetId
		}
		return ""
	},
}

// SignatureMessages are messages with application-level secp256k1 proof
// (proofPubKey + proofSignature) that get pubkey-to-address derivation
// validation in the PubKeyDerivationDecorator.
var SignatureMessages = map[string]bool{
	"/structs.structs.MsgAddressRegister":          true,
	"/structs.structs.MsgGuildMembershipJoinProxy": true,
}

// CreatorExtractors provides direct field access for messages that have
// goproto_getters = false and therefore lack a GetCreator() method.
var CreatorExtractors = map[string]func(sdk.Msg) string{
	"/structs.structs.MsgReactorInfuse": func(msg sdk.Msg) string {
		if m, ok := msg.(*types.MsgReactorInfuse); ok {
			return m.Creator
		}
		return ""
	},
	"/structs.structs.MsgReactorDefuse": func(msg sdk.Msg) string {
		if m, ok := msg.(*types.MsgReactorDefuse); ok {
			return m.Creator
		}
		return ""
	},
	"/structs.structs.MsgReactorBeginMigration": func(msg sdk.Msg) string {
		if m, ok := msg.(*types.MsgReactorBeginMigration); ok {
			return m.Creator
		}
		return ""
	},
	"/structs.structs.MsgReactorCancelDefusion": func(msg sdk.Msg) string {
		if m, ok := msg.(*types.MsgReactorCancelDefusion); ok {
			return m.Creator
		}
		return ""
	},
	"/structs.structs.MsgReactorRestart": func(msg sdk.Msg) string {
		if m, ok := msg.(*types.MsgReactorRestart); ok {
			return m.Creator
		}
		return ""
	},
	"/structs.structs.MsgPlayerSend": func(msg sdk.Msg) string {
		if m, ok := msg.(*types.MsgPlayerSend); ok {
			return m.Creator
		}
		return ""
	},
}

// ThrottleKeyExtractors maps message type URLs to functions that return the
// transient store throttle key for per-object-per-block rate limiting.
var ThrottleKeyExtractors = map[string]func(sdk.Msg) string{
	"/structs.structs.MsgFleetMove": func(msg sdk.Msg) string {
		if m, ok := msg.(*types.MsgFleetMove); ok {
			return "fleet/" + m.FleetId
		}
		return ""
	},
	"/structs.structs.MsgPlanetExplore": func(msg sdk.Msg) string {
		if m, ok := msg.(*types.MsgPlanetExplore); ok {
			return "explore/" + m.PlayerId
		}
		return ""
	},
	"/structs.structs.MsgAddressRegister": func(msg sdk.Msg) string {
		if m, ok := msg.(*types.MsgAddressRegister); ok {
			return "register/" + m.PlayerId
		}
		return ""
	},
}

// ThrottleTarget names the object a throttled message acts on, together with
// the permission its handler demands over that object.
type ThrottleTarget struct {
	Kind       types.ObjectType
	TargetId   string
	Permission types.Permission
}

// ThrottleTargetAuth mirrors, for every throttled message, the target-object
// authorization its handler performs. The keys of ProofMessages and
// ThrottleKeyExtractors both name objects the transaction chooses, and the
// throttle those keys drive is object-global, so a signer with no standing on
// the named object must not be allowed to reserve one. Layer 1 does not catch
// this: a primary address holds PermAll and so passes PermissionMap while
// naming somebody else's struct.
//
// The permission here must be the one the handler's Can*By call resolves to;
// TestArch_ThrottleTargetAuthMatchesHandlers reads the handler sources and
// fails if the two drift. An entry missing from this map means that message
// reserves no throttle key at all, which under-throttles rather than erroring —
// TestThrottleTargetAuthCompleteness is what catches it.
var ThrottleTargetAuth = map[string]func(sdk.Msg) (ThrottleTarget, bool){
	// structure.GetOwner().CanBuildHashedBy(callingPlayer)
	"/structs.structs.MsgStructBuildComplete": func(msg sdk.Msg) (ThrottleTarget, bool) {
		if m, ok := msg.(*types.MsgStructBuildComplete); ok {
			return ThrottleTarget{types.ObjectType_struct, m.StructId, types.PermHashBuild}, true
		}
		return ThrottleTarget{}, false
	},
	// structure.GetOwner().CanMineHashedBy(callingPlayer)
	"/structs.structs.MsgStructOreMinerComplete": func(msg sdk.Msg) (ThrottleTarget, bool) {
		if m, ok := msg.(*types.MsgStructOreMinerComplete); ok {
			return ThrottleTarget{types.ObjectType_struct, m.StructId, types.PermHashMine}, true
		}
		return ThrottleTarget{}, false
	},
	// structure.GetOwner().CanRefineHashedBy(callingPlayer)
	"/structs.structs.MsgStructOreRefineryComplete": func(msg sdk.Msg) (ThrottleTarget, bool) {
		if m, ok := msg.(*types.MsgStructOreRefineryComplete); ok {
			return ThrottleTarget{types.ObjectType_struct, m.StructId, types.PermHashRefine}, true
		}
		return ThrottleTarget{}, false
	},
	// fleet.GetOwner().CanRaidHashedBy(callingPlayer)
	"/structs.structs.MsgPlanetRaidComplete": func(msg sdk.Msg) (ThrottleTarget, bool) {
		if m, ok := msg.(*types.MsgPlanetRaidComplete); ok {
			return ThrottleTarget{types.ObjectType_fleet, m.FleetId, types.PermHashRaid}, true
		}
		return ThrottleTarget{}, false
	},
	// fleet.GetOwner().CanBePlayedBy(activePlayer)
	"/structs.structs.MsgFleetMove": func(msg sdk.Msg) (ThrottleTarget, bool) {
		if m, ok := msg.(*types.MsgFleetMove); ok {
			return ThrottleTarget{types.ObjectType_fleet, m.FleetId, types.PermPlay}, true
		}
		return ThrottleTarget{}, false
	},
	// player.CanBePlayedBy(callingPlayer)
	"/structs.structs.MsgPlanetExplore": func(msg sdk.Msg) (ThrottleTarget, bool) {
		if m, ok := msg.(*types.MsgPlanetExplore); ok {
			return ThrottleTarget{types.ObjectType_player, m.PlayerId, types.PermPlay}, true
		}
		return ThrottleTarget{}, false
	},
	// player.CanRegisterAddressBy(activePlayer, types.Permission(msg.Permissions)).
	// The bit is whatever the message asks to grant, so a registration that
	// grants nothing authorizes against Permissionless, which PermissionCheck
	// always denies — and denial here only skips the reservation.
	"/structs.structs.MsgAddressRegister": func(msg sdk.Msg) (ThrottleTarget, bool) {
		if m, ok := msg.(*types.MsgAddressRegister); ok {
			return ThrottleTarget{types.ObjectType_player, m.PlayerId, types.Permission(m.Permissions)}, true
		}
		return ThrottleTarget{}, false
	},
}

// FreeStakingMessages enumerates the x/staking message type URLs that receive
// free gas treatment. These are the operations players must perform to
// participate in the network — delegation is how they power gameplay.
var FreeStakingMessages = map[string]bool{
	"/cosmos.staking.v1beta1.MsgDelegate":                  true,
	"/cosmos.staking.v1beta1.MsgUndelegate":                true,
	"/cosmos.staking.v1beta1.MsgBeginRedelegate":           true,
	"/cosmos.staking.v1beta1.MsgCancelUnbondingDelegation": true,
	"/cosmos.staking.v1beta1.MsgCreateValidator":           true,
	"/cosmos.staking.v1beta1.MsgEditValidator":             true,
}

const StakingMessageTypeURLPrefix = "/cosmos.staking.v1beta1.Msg"

// IsStructsMessage checks if a message type URL belongs to the Structs module.
func IsStructsMessage(typeURL string) bool {
	return len(typeURL) > len(StructsMessageTypeURLPrefix) &&
		typeURL[:len(StructsMessageTypeURLPrefix)] == StructsMessageTypeURLPrefix
}

// IsStakingMessage checks if a message type URL is a known free staking message.
func IsStakingMessage(typeURL string) bool {
	return FreeStakingMessages[typeURL]
}

// IsFreeTransaction returns true if all messages in the tx are Structs gameplay
// messages (excluding MsgUpdateParams, which is a governance operation).
func IsFreeTransaction(msgs []sdk.Msg) bool {
	if len(msgs) == 0 {
		return false
	}
	for _, msg := range msgs {
		typeURL := sdk.MsgTypeURL(msg)
		if !IsStructsMessage(typeURL) || typeURL == MsgUpdateParamsTypeURL {
			return false
		}
	}
	return true
}

// IsFreeStakingTransaction returns true if all messages in the tx are known
// free staking messages. Does not allow mixing with Structs messages.
func IsFreeStakingTransaction(msgs []sdk.Msg) bool {
	if len(msgs) == 0 {
		return false
	}
	for _, msg := range msgs {
		if !FreeStakingMessages[sdk.MsgTypeURL(msg)] {
			return false
		}
	}
	return true
}

// IsAnyFreeTransaction returns true if the tx qualifies for free gas treatment
// (either pure-Structs or pure-staking).
func IsAnyFreeTransaction(msgs []sdk.Msg) bool {
	return IsFreeTransaction(msgs) || IsFreeStakingTransaction(msgs)
}

// ContainsGatedStructsMessage returns true if any message in the tx is a Structs
// gameplay message, and is what decides whether the Structs ante checks run.
//
// Gating must follow the message, not the fee. IsFreeTransaction requires EVERY
// message to be a Structs message, so a tx pairing one gameplay message with any
// non-Structs message (a bank send, say) is not "free", and keying the Structs
// decorators off free-ness let such a tx buy its way past the player
// registration, permission, charge and throttle checks for the price of a normal
// fee. Paying a fee is not authorization.
//
// MsgUpdateParams is excluded: it is signed by the governance authority rather
// than a player, so there is no address to resolve or permission to check.
func ContainsGatedStructsMessage(msgs []sdk.Msg) bool {
	for _, msg := range msgs {
		typeURL := sdk.MsgTypeURL(msg)
		if IsStructsMessage(typeURL) && typeURL != MsgUpdateParamsTypeURL {
			return true
		}
	}
	return false
}

// StakingSignerExtractors provides direct field access for the signer address
// of each free staking message type (goproto_getters = false on all of them).
var StakingSignerExtractors = map[string]func(sdk.Msg) string{
	"/cosmos.staking.v1beta1.MsgDelegate": func(msg sdk.Msg) string {
		return msg.(*stakingtypes.MsgDelegate).DelegatorAddress
	},
	"/cosmos.staking.v1beta1.MsgUndelegate": func(msg sdk.Msg) string {
		return msg.(*stakingtypes.MsgUndelegate).DelegatorAddress
	},
	"/cosmos.staking.v1beta1.MsgBeginRedelegate": func(msg sdk.Msg) string {
		return msg.(*stakingtypes.MsgBeginRedelegate).DelegatorAddress
	},
	"/cosmos.staking.v1beta1.MsgCancelUnbondingDelegation": func(msg sdk.Msg) string {
		return msg.(*stakingtypes.MsgCancelUnbondingDelegation).DelegatorAddress
	},
	"/cosmos.staking.v1beta1.MsgCreateValidator": func(msg sdk.Msg) string {
		return msg.(*stakingtypes.MsgCreateValidator).ValidatorAddress
	},
	"/cosmos.staking.v1beta1.MsgEditValidator": func(msg sdk.Msg) string {
		return msg.(*stakingtypes.MsgEditValidator).ValidatorAddress
	},
}
