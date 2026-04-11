package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

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
	"/structs.structs.MsgGuildUpdateEndpoint":                            true,
	"/structs.structs.MsgGuildUpdateEntryRank":                           true,
	"/structs.structs.MsgGuildUpdateEntrySubstationId":                   true,
	"/structs.structs.MsgGuildUpdateJoinInfusionMinimum":                 true,
	"/structs.structs.MsgGuildUpdateJoinInfusionMinimumBypassByInvite":   true,
	"/structs.structs.MsgGuildUpdateJoinInfusionMinimumBypassByRequest":  true,
	"/structs.structs.MsgGuildUpdateOwnerId":                             true,
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
	"/structs.structs.MsgPlayerSend":                                     true,
	"/structs.structs.MsgPlayerUpdateGuildRank":                          true,
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
	"/structs.structs.MsgStructActivate":                                 true,
	"/structs.structs.MsgStructAttack":                                   true,
	"/structs.structs.MsgStructBuildCancel":                              true,
	"/structs.structs.MsgStructBuildComplete":                            true,
	"/structs.structs.MsgStructBuildInitiate":                            true,
	"/structs.structs.MsgStructDeactivate":                               true,
	"/structs.structs.MsgStructDefenseClear":                             true,
	"/structs.structs.MsgStructDefenseSet":                               true,
	"/structs.structs.MsgStructGeneratorInfuse":                          true,
	"/structs.structs.MsgStructMove":                                     true,
	"/structs.structs.MsgStructOreMinerComplete":                         true,
	"/structs.structs.MsgStructOreRefineryComplete":                      true,
	"/structs.structs.MsgStructStealthActivate":                          true,
	"/structs.structs.MsgStructStealthDeactivate":                        true,
	"/structs.structs.MsgSubstationAllocationConnect":                    true,
	"/structs.structs.MsgSubstationAllocationDisconnect":                 true,
	"/structs.structs.MsgSubstationCreate":                               true,
	"/structs.structs.MsgSubstationDelete":                               true,
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
	"/structs.structs.MsgStructDefenseClear":     types.PermPlay,
	"/structs.structs.MsgStructDefenseSet":       types.PermPlay,
	"/structs.structs.MsgStructMove":             types.PermPlay,
	"/structs.structs.MsgStructStealthActivate":  types.PermPlay,
	"/structs.structs.MsgStructStealthDeactivate": types.PermPlay,

	// Proof-of-work actions (require specific hash permissions)
	"/structs.structs.MsgStructBuildComplete":      types.PermHashBuild,
	"/structs.structs.MsgStructOreMinerComplete":   types.PermHashMine,
	"/structs.structs.MsgStructOreRefineryComplete": types.PermHashRefine,
	"/structs.structs.MsgPlanetRaidComplete":       types.PermHashRaid,

	// Token operations
	"/structs.structs.MsgGuildBankRedeem":      types.PermTokenTransfer,
	"/structs.structs.MsgPlayerSend":           types.PermTokenTransfer,
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
	"/structs.structs.MsgAllocationTransfer":          types.PermAdmin,
	"/structs.structs.MsgGuildUpdateOwnerId":          types.PermAdmin,
	"/structs.structs.MsgPlayerUpdatePrimaryAddress":  types.PermAdmin,
	"/structs.structs.MsgGuildUpdateEntryRank":        types.PermUpdate,

	// Object updates
	"/structs.structs.MsgAgreementCapacityDecrease":   types.PermUpdate,
	"/structs.structs.MsgAgreementCapacityIncrease":   types.PermUpdate,
	"/structs.structs.MsgAgreementClose":              types.PermUpdate,
	"/structs.structs.MsgAgreementDurationIncrease":   types.PermUpdate,
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
	"/structs.structs.MsgAgreementOpen":                         true,
	"/structs.structs.MsgGuildMembershipInvite":                 true,
	"/structs.structs.MsgGuildMembershipInviteRevoke":           true,
	"/structs.structs.MsgGuildMembershipKick":                   true,
	"/structs.structs.MsgGuildMembershipRequest":                true,
	"/structs.structs.MsgGuildMembershipRequestApprove":         true,
	"/structs.structs.MsgGuildMembershipRequestDeny":            true,
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
		return msg.(*types.MsgStructBuildComplete).StructId
	},
	"/structs.structs.MsgStructOreMinerComplete": func(msg sdk.Msg) string {
		return msg.(*types.MsgStructOreMinerComplete).StructId
	},
	"/structs.structs.MsgStructOreRefineryComplete": func(msg sdk.Msg) string {
		return msg.(*types.MsgStructOreRefineryComplete).StructId
	},
	"/structs.structs.MsgPlanetRaidComplete": func(msg sdk.Msg) string {
		return msg.(*types.MsgPlanetRaidComplete).FleetId
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
		return msg.(*types.MsgReactorInfuse).Creator
	},
	"/structs.structs.MsgReactorDefuse": func(msg sdk.Msg) string {
		return msg.(*types.MsgReactorDefuse).Creator
	},
	"/structs.structs.MsgReactorBeginMigration": func(msg sdk.Msg) string {
		return msg.(*types.MsgReactorBeginMigration).Creator
	},
	"/structs.structs.MsgReactorCancelDefusion": func(msg sdk.Msg) string {
		return msg.(*types.MsgReactorCancelDefusion).Creator
	},
	"/structs.structs.MsgPlayerSend": func(msg sdk.Msg) string {
		return msg.(*types.MsgPlayerSend).Creator
	},
}

// ThrottleKeyExtractors maps message type URLs to functions that return the
// transient store throttle key for per-object-per-block rate limiting.
var ThrottleKeyExtractors = map[string]func(sdk.Msg) string{
	"/structs.structs.MsgFleetMove": func(msg sdk.Msg) string {
		return "fleet/" + msg.(*types.MsgFleetMove).FleetId
	},
	"/structs.structs.MsgPlanetExplore": func(msg sdk.Msg) string {
		return "explore/" + msg.(*types.MsgPlanetExplore).PlayerId
	},
	"/structs.structs.MsgAddressRegister": func(msg sdk.Msg) string {
		return "register/" + msg.(*types.MsgAddressRegister).PlayerId
	},
}

// IsStructsMessage checks if a message type URL belongs to the Structs module.
func IsStructsMessage(typeURL string) bool {
	return len(typeURL) > len(StructsMessageTypeURLPrefix) &&
		typeURL[:len(StructsMessageTypeURLPrefix)] == StructsMessageTypeURLPrefix
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
