package types

import (
	"fmt"
)

// =============================================================================
// Structured Error Interface
// =============================================================================

// StructuredError provides programmatic error details for client handling,
// structured logging, and testing assertions.
type StructuredError interface {
	error
	Code() uint32
	LogFields() []interface{}
	Unwrap() error
}

// =============================================================================
// 1. ObjectNotFoundError
// =============================================================================

// ObjectNotFoundError indicates a requested object does not exist.
type ObjectNotFoundError struct {
	ObjectType  string // "player", "struct", "guild", "planet", "fleet", "substation", "allocation", "reactor", "infusion", "provider", "agreement"
	ObjectId    string
	ObjectIndex uint64 // For indexed lookups (optional)
	Context     string // Additional context (optional)
}

func NewObjectNotFoundError(objectType, objectId string) *ObjectNotFoundError {
	return &ObjectNotFoundError{
		ObjectType: objectType,
		ObjectId:   objectId,
	}
}

func (e *ObjectNotFoundError) WithIndex(index uint64) *ObjectNotFoundError {
	e.ObjectIndex = index
	return e
}

func (e *ObjectNotFoundError) WithContext(ctx string) *ObjectNotFoundError {
	e.Context = ctx
	return e
}

func (e *ObjectNotFoundError) Error() string {
	if e.Context != "" {
		return fmt.Sprintf("%s (%s) not found: %s", e.ObjectType, e.ObjectId, e.Context)
	}
	return fmt.Sprintf("%s (%s) not found", e.ObjectType, e.ObjectId)
}

func (e *ObjectNotFoundError) Code() uint32 { return 1050 }

func (e *ObjectNotFoundError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "object_not_found",
		"object_type", e.ObjectType,
		"object_id", e.ObjectId,
		"object_index", e.ObjectIndex,
		"context", e.Context,
	}
}

func (e *ObjectNotFoundError) Unwrap() error { return ErrObjectNotFound }

// =============================================================================
// 2. InsufficientChargeError
// =============================================================================

// InsufficientChargeError indicates a player lacks charge for an action.
type InsufficientChargeError struct {
	PlayerId   string
	Required   uint64
	Available  uint64
	Action     string // "build", "activate", "attack", "move", "defend", "stealth", "resume"
	StructType uint64 // Optional
	StructId   string // Optional
}

func NewInsufficientChargeError(playerId string, required, available uint64, action string) *InsufficientChargeError {
	return &InsufficientChargeError{
		PlayerId:  playerId,
		Required:  required,
		Available: available,
		Action:    action,
	}
}

func (e *InsufficientChargeError) WithStructType(structType uint64) *InsufficientChargeError {
	e.StructType = structType
	return e
}

func (e *InsufficientChargeError) WithStructId(structId string) *InsufficientChargeError {
	e.StructId = structId
	return e
}

func (e *InsufficientChargeError) Error() string {
	if e.StructType > 0 {
		return fmt.Sprintf("struct type (%d) required charge of %d for %s, but player (%s) only had %d",
			e.StructType, e.Required, e.Action, e.PlayerId, e.Available)
	}
	return fmt.Sprintf("action %s required charge of %d, but player (%s) only had %d",
		e.Action, e.Required, e.PlayerId, e.Available)
}

func (e *InsufficientChargeError) Code() uint32 { return 1200 }

func (e *InsufficientChargeError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "insufficient_charge",
		"player_id", e.PlayerId,
		"required", e.Required,
		"available", e.Available,
		"action", e.Action,
		"struct_type", e.StructType,
		"struct_id", e.StructId,
	}
}

func (e *InsufficientChargeError) Unwrap() error { return ErrInsufficientCharge }

// =============================================================================
// 3. PermissionError
// =============================================================================

// PermissionError indicates permission was denied for an action.
type PermissionError struct {
	CallerType          string // "address" or "player"
	CallerId            string
	TargetType          string // "player", "substation", "guild", "provider", "agreement", "struct"
	TargetId            string
	Permission          uint64
	Action              string
	RequiredLevel       uint64 // For grant/set level checks (optional)
	GuildId             string // For guild market access (optional)
	AssociationTargetId string // For player association (optional)
}

func NewPermissionError(callerType, callerId, targetType, targetId string, permission uint64, action string) *PermissionError {
	return &PermissionError{
		CallerType: callerType,
		CallerId:   callerId,
		TargetType: targetType,
		TargetId:   targetId,
		Permission: permission,
		Action:     action,
	}
}

func (e *PermissionError) WithRequiredLevel(level uint64) *PermissionError {
	e.RequiredLevel = level
	return e
}

func (e *PermissionError) WithGuildId(guildId string) *PermissionError {
	e.GuildId = guildId
	return e
}

func (e *PermissionError) WithAssociationTarget(targetId string) *PermissionError {
	e.AssociationTargetId = targetId
	return e
}

func (e *PermissionError) Error() string {
	if e.TargetId != "" {
		return fmt.Sprintf("calling %s (%s) has no %s permission on %s (%s)",
			e.CallerType, e.CallerId, e.Action, e.TargetType, e.TargetId)
	}
	return fmt.Sprintf("calling %s (%s) has no %s permission", e.CallerType, e.CallerId, e.Action)
}

func (e *PermissionError) Code() uint32 { return 1100 }

func (e *PermissionError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "permission_denied",
		"caller_type", e.CallerType,
		"caller_id", e.CallerId,
		"target_type", e.TargetType,
		"target_id", e.TargetId,
		"permission", e.Permission,
		"action", e.Action,
	}
}

func (e *PermissionError) Unwrap() error { return ErrPermission }

// =============================================================================
// 4. PlayerPowerError
// =============================================================================

// PlayerPowerError indicates a player power/capacity issue.
type PlayerPowerError struct {
	PlayerId  string
	Reason    string // "offline", "capacity_exceeded", "load_exceeded"
	Required  uint64 // For capacity checks (optional)
	Available uint64 // For capacity checks (optional)
}

func NewPlayerPowerError(playerId, reason string) *PlayerPowerError {
	return &PlayerPowerError{
		PlayerId: playerId,
		Reason:   reason,
	}
}

func (e *PlayerPowerError) WithCapacity(required, available uint64) *PlayerPowerError {
	e.Required = required
	e.Available = available
	return e
}

func (e *PlayerPowerError) Error() string {
	switch e.Reason {
	case "offline":
		return fmt.Sprintf("player (%s) is offline due to power", e.PlayerId)
	case "capacity_exceeded":
		return fmt.Sprintf("player (%s) cannot handle new load requirements (required: %d, available: %d)",
			e.PlayerId, e.Required, e.Available)
	default:
		return fmt.Sprintf("player (%s) power error: %s", e.PlayerId, e.Reason)
	}
}

func (e *PlayerPowerError) Code() uint32 { return 1201 }

func (e *PlayerPowerError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "player_power",
		"player_id", e.PlayerId,
		"reason", e.Reason,
		"required", e.Required,
		"available", e.Available,
	}
}

func (e *PlayerPowerError) Unwrap() error { return ErrPlayerPowerOffline }

// =============================================================================
// 5. PlayerAffordabilityError
// =============================================================================

// PlayerAffordabilityError indicates a player cannot afford an action.
type PlayerAffordabilityError struct {
	PlayerId string
	Action   string // "mint", "agreement", "increase_duration", "refine"
	Required string // Amount/description of what's needed
}

func NewPlayerAffordabilityError(playerId, action, required string) *PlayerAffordabilityError {
	return &PlayerAffordabilityError{
		PlayerId: playerId,
		Action:   action,
		Required: required,
	}
}

func (e *PlayerAffordabilityError) Error() string {
	if e.Required != "" {
		return fmt.Sprintf("player (%s) cannot afford %s: requires %s", e.PlayerId, e.Action, e.Required)
	}
	return fmt.Sprintf("player (%s) cannot afford %s", e.PlayerId, e.Action)
}

func (e *PlayerAffordabilityError) Code() uint32 { return 1203 }

func (e *PlayerAffordabilityError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "player_affordability",
		"player_id", e.PlayerId,
		"action", e.Action,
		"required", e.Required,
	}
}

func (e *PlayerAffordabilityError) Unwrap() error { return ErrPlayerAffordability }

// =============================================================================
// 6. StructStateError
// =============================================================================

// StructStateError indicates an invalid struct state for an operation.
type StructStateError struct {
	StructId      string
	CurrentState  string // "offline", "building", "online", "destroyed"
	RequiredState string // "online", "built", "offline"
	Action        string
}

func NewStructStateError(structId, currentState, requiredState, action string) *StructStateError {
	return &StructStateError{
		StructId:      structId,
		CurrentState:  currentState,
		RequiredState: requiredState,
		Action:        action,
	}
}

func (e *StructStateError) Error() string {
	return fmt.Sprintf("struct (%s) is %s but must be %s for %s",
		e.StructId, e.CurrentState, e.RequiredState, e.Action)
}

func (e *StructStateError) Code() uint32 { return 1250 }

func (e *StructStateError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "struct_state",
		"struct_id", e.StructId,
		"current_state", e.CurrentState,
		"required_state", e.RequiredState,
		"action", e.Action,
	}
}

func (e *StructStateError) Unwrap() error { return ErrStructState }

// =============================================================================
// 7. StructCapabilityError
// =============================================================================

// StructCapabilityError indicates a struct lacks a required capability.
type StructCapabilityError struct {
	StructId   string
	Capability string // "mining", "refining", "stealth", "generation", "defense"
}

func NewStructCapabilityError(structId, capability string) *StructCapabilityError {
	return &StructCapabilityError{
		StructId:   structId,
		Capability: capability,
	}
}

func (e *StructCapabilityError) Error() string {
	return fmt.Sprintf("struct (%s) has no %s system", e.StructId, e.Capability)
}

func (e *StructCapabilityError) Code() uint32 { return 1260 }

func (e *StructCapabilityError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "struct_capability",
		"struct_id", e.StructId,
		"capability", e.Capability,
	}
}

func (e *StructCapabilityError) Unwrap() error { return ErrStructCapability }

// =============================================================================
// 8. FleetCommandError
// =============================================================================

// FleetCommandError indicates a fleet command struct issue.
type FleetCommandError struct {
	FleetId  string
	StructId string // The struct attempting action (optional)
	Reason   string // "no_command_struct", "command_offline"
}

func NewFleetCommandError(fleetId, reason string) *FleetCommandError {
	return &FleetCommandError{
		FleetId: fleetId,
		Reason:  reason,
	}
}

func (e *FleetCommandError) WithStructId(structId string) *FleetCommandError {
	e.StructId = structId
	return e
}

func (e *FleetCommandError) Error() string {
	switch e.Reason {
	case "no_command_struct":
		return fmt.Sprintf("fleet (%s) needs a command struct before deploy", e.FleetId)
	case "command_offline":
		return fmt.Sprintf("fleet (%s) needs an online command struct before deploy", e.FleetId)
	default:
		return fmt.Sprintf("fleet (%s) command error: %s", e.FleetId, e.Reason)
	}
}

func (e *FleetCommandError) Code() uint32 { return 1450 }

func (e *FleetCommandError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "fleet_command",
		"fleet_id", e.FleetId,
		"struct_id", e.StructId,
		"reason", e.Reason,
	}
}

func (e *FleetCommandError) Unwrap() error { return ErrFleetCommand }

// =============================================================================
// 9. FleetStateError
// =============================================================================

// FleetStateError indicates an invalid fleet state for an operation.
type FleetStateError struct {
	FleetId  string
	State    string // "on_station", "in_transit", "in_queue"
	Action   string // "build", "raid", "deploy"
	Position uint64 // Queue position for raid errors (optional)
}

func NewFleetStateError(fleetId, state, action string) *FleetStateError {
	return &FleetStateError{
		FleetId: fleetId,
		State:   state,
		Action:  action,
	}
}

func (e *FleetStateError) WithPosition(pos uint64) *FleetStateError {
	e.Position = pos
	return e
}

func (e *FleetStateError) Error() string {
	return fmt.Sprintf("fleet (%s) cannot %s while %s", e.FleetId, e.Action, e.State)
}

func (e *FleetStateError) Code() uint32 { return 1453 }

func (e *FleetStateError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "fleet_state",
		"fleet_id", e.FleetId,
		"state", e.State,
		"action", e.Action,
		"position", e.Position,
	}
}

func (e *FleetStateError) Unwrap() error { return ErrFleetState }

// =============================================================================
// 10. StructOwnershipError
// =============================================================================

// StructOwnershipError indicates a struct ownership mismatch.
type StructOwnershipError struct {
	StructId      string
	ExpectedOwner string
	ActualOwner   string
	LocationType  string // "fleet" or "planet"
	LocationId    string
}

func NewStructOwnershipError(structId, expectedOwner, actualOwner string) *StructOwnershipError {
	return &StructOwnershipError{
		StructId:      structId,
		ExpectedOwner: expectedOwner,
		ActualOwner:   actualOwner,
	}
}

func (e *StructOwnershipError) WithLocation(locationType, locationId string) *StructOwnershipError {
	e.LocationType = locationType
	e.LocationId = locationId
	return e
}

func (e *StructOwnershipError) Error() string {
	return fmt.Sprintf("struct owner must match %s", e.LocationType)
}

func (e *StructOwnershipError) Code() uint32 { return 1301 }

func (e *StructOwnershipError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "struct_ownership",
		"struct_id", e.StructId,
		"expected_owner", e.ExpectedOwner,
		"actual_owner", e.ActualOwner,
		"location_type", e.LocationType,
		"location_id", e.LocationId,
	}
}

func (e *StructOwnershipError) Unwrap() error { return ErrStructOwnership }

// =============================================================================
// 11. StructLocationError
// =============================================================================

// StructLocationError indicates an invalid struct location/ambit.
type StructLocationError struct {
	StructId     string
	StructType   uint64
	Ambit        string
	LocationType string
	LocationId   string
	Reason       string // "invalid_ambit", "invalid_location", "command_struct_fleet_only", "outside_planet"
}

func NewStructLocationError(structType uint64, ambit, reason string) *StructLocationError {
	return &StructLocationError{
		StructType: structType,
		Ambit:      ambit,
		Reason:     reason,
	}
}

func (e *StructLocationError) WithStruct(structId string) *StructLocationError {
	e.StructId = structId
	return e
}

func (e *StructLocationError) WithLocation(locationType, locationId string) *StructLocationError {
	e.LocationType = locationType
	e.LocationId = locationId
	return e
}

func (e *StructLocationError) Error() string {
	switch e.Reason {
	case "invalid_ambit":
		return fmt.Sprintf("struct cannot exist in ambit (%s) based on struct type (%d)", e.Ambit, e.StructType)
	case "command_struct_fleet_only":
		return "command structs can only be built directly in the fleet"
	case "outside_planet":
		return "struct type cannot exist outside a planet"
	default:
		return fmt.Sprintf("struct type (%d) cannot exist in this location: %s", e.StructType, e.Reason)
	}
}

func (e *StructLocationError) Code() uint32 { return 1302 }

func (e *StructLocationError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "struct_location",
		"struct_id", e.StructId,
		"struct_type", e.StructType,
		"ambit", e.Ambit,
		"location_type", e.LocationType,
		"location_id", e.LocationId,
		"reason", e.Reason,
	}
}

func (e *StructLocationError) Unwrap() error { return ErrStructLocation }

// =============================================================================
// 12. CombatTargetingError
// =============================================================================

// CombatTargetingError indicates a combat targeting issue.
type CombatTargetingError struct {
	AttackerId    string
	TargetId      string
	WeaponSystem  string
	Reason        string // "destroyed", "unreachable", "blocked", "hidden", "out_of_range", "incomplete_targeting"
	IsCounter     bool   // True if counter-attack check
	AttackerAmbit string
	TargetAmbit   string
}

func NewCombatTargetingError(attackerId, targetId, weaponSystem, reason string) *CombatTargetingError {
	return &CombatTargetingError{
		AttackerId:   attackerId,
		TargetId:     targetId,
		WeaponSystem: weaponSystem,
		Reason:       reason,
	}
}

func (e *CombatTargetingError) AsCounter() *CombatTargetingError {
	e.IsCounter = true
	return e
}

func (e *CombatTargetingError) WithAmbits(attackerAmbit, targetAmbit string) *CombatTargetingError {
	e.AttackerAmbit = attackerAmbit
	e.TargetAmbit = targetAmbit
	return e
}

func (e *CombatTargetingError) Error() string {
	attackerType := "attacker"
	if e.IsCounter {
		attackerType = "counter-attacker"
	}
	return fmt.Sprintf("target struct (%s) is %s from %s struct (%s)", e.TargetId, e.Reason, attackerType, e.AttackerId)
}

func (e *CombatTargetingError) Code() uint32 { return 1400 }

func (e *CombatTargetingError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "combat_targeting",
		"attacker_id", e.AttackerId,
		"target_id", e.TargetId,
		"weapon_system", e.WeaponSystem,
		"reason", e.Reason,
		"is_counter", e.IsCounter,
		"attacker_ambit", e.AttackerAmbit,
		"target_ambit", e.TargetAmbit,
	}
}

func (e *CombatTargetingError) Unwrap() error { return ErrCombatTargeting }

// =============================================================================
// 13. StructBuildError
// =============================================================================

// StructBuildError indicates a struct build issue.
type StructBuildError struct {
	StructType       uint64
	LocationType     string // "fleet" or "planet"
	LocationId       string
	Slot             uint64
	Ambit            string
	Reason           string // "slot_occupied", "slot_unavailable", "invalid_ambit", "command_exists", "type_unsupported"
	ExistingStructId string // When slot is occupied (optional)
}

func NewStructBuildError(structType uint64, locationType, locationId, reason string) *StructBuildError {
	return &StructBuildError{
		StructType:   structType,
		LocationType: locationType,
		LocationId:   locationId,
		Reason:       reason,
	}
}

func (e *StructBuildError) WithSlot(slot uint64) *StructBuildError {
	e.Slot = slot
	return e
}

func (e *StructBuildError) WithAmbit(ambit string) *StructBuildError {
	e.Ambit = ambit
	return e
}

func (e *StructBuildError) WithExistingStruct(structId string) *StructBuildError {
	e.ExistingStructId = structId
	return e
}

func (e *StructBuildError) Error() string {
	switch e.Reason {
	case "slot_occupied":
		return fmt.Sprintf("the %s (%s) already has a struct on that slot", e.LocationType, e.LocationId)
	case "slot_unavailable":
		return fmt.Sprintf("the %s (%s) doesn't have that slot available to build on", e.LocationType, e.LocationId)
	case "invalid_ambit":
		return "struct build was initiated on a non-existent ambit"
	case "command_exists":
		return fmt.Sprintf("the %s (%s) already has a command struct", e.LocationType, e.LocationId)
	case "type_unsupported":
		return "we're not building these yet"
	default:
		return fmt.Sprintf("struct build failed on %s (%s): %s", e.LocationType, e.LocationId, e.Reason)
	}
}

func (e *StructBuildError) Code() uint32 { return 1350 }

func (e *StructBuildError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "struct_build",
		"struct_type", e.StructType,
		"location_type", e.LocationType,
		"location_id", e.LocationId,
		"slot", e.Slot,
		"ambit", e.Ambit,
		"reason", e.Reason,
		"existing_struct_id", e.ExistingStructId,
	}
}

func (e *StructBuildError) Unwrap() error { return ErrStructBuild }

// =============================================================================
// 14. PlayerHaltedError
// =============================================================================

// PlayerHaltedError indicates a player is halted and cannot perform actions.
type PlayerHaltedError struct {
	PlayerId string
	StructId string // Optional, when action is struct-related
	Action   string
}

func NewPlayerHaltedError(playerId, action string) *PlayerHaltedError {
	return &PlayerHaltedError{
		PlayerId: playerId,
		Action:   action,
	}
}

func (e *PlayerHaltedError) WithStruct(structId string) *PlayerHaltedError {
	e.StructId = structId
	return e
}

func (e *PlayerHaltedError) Error() string {
	if e.StructId != "" {
		return fmt.Sprintf("struct (%s) cannot perform actions while player (%s) is halted", e.StructId, e.PlayerId)
	}
	return fmt.Sprintf("cannot perform actions while player (%s) is halted", e.PlayerId)
}

func (e *PlayerHaltedError) Code() uint32 { return 1151 }

func (e *PlayerHaltedError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "player_halted",
		"player_id", e.PlayerId,
		"struct_id", e.StructId,
		"action", e.Action,
	}
}

func (e *PlayerHaltedError) Unwrap() error { return ErrPlayerHalted }

// =============================================================================
// 15. PlayerRequiredError
// =============================================================================

// PlayerRequiredError indicates a player account is required for an action.
type PlayerRequiredError struct {
	Address string
	Action  string // "build", "guild_update", "guild_create", "substation_action"
}

func NewPlayerRequiredError(address, action string) *PlayerRequiredError {
	return &PlayerRequiredError{
		Address: address,
		Action:  action,
	}
}

func (e *PlayerRequiredError) Error() string {
	return fmt.Sprintf("%s requires player account but none associated with %s", e.Action, e.Address)
}

func (e *PlayerRequiredError) Code() uint32 { return 1150 }

func (e *PlayerRequiredError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "player_required",
		"address", e.Address,
		"action", e.Action,
	}
}

func (e *PlayerRequiredError) Unwrap() error { return ErrPlayerRequired }

// =============================================================================
// 16. AddressValidationError
// =============================================================================

// AddressValidationError indicates an address validation failure.
type AddressValidationError struct {
	Address          string
	ExpectedPlayerId string
	ActualPlayerId   string
	Reason           string // "invalid_format", "not_registered", "wrong_player", "proof_mismatch", "signature_invalid", "already_registered"
}

func NewAddressValidationError(address, reason string) *AddressValidationError {
	return &AddressValidationError{
		Address: address,
		Reason:  reason,
	}
}

func (e *AddressValidationError) WithPlayers(expected, actual string) *AddressValidationError {
	e.ExpectedPlayerId = expected
	e.ActualPlayerId = actual
	return e
}

func (e *AddressValidationError) Error() string {
	switch e.Reason {
	case "invalid_format":
		return fmt.Sprintf("address (%s) couldn't be validated as a real address", e.Address)
	case "not_registered":
		return fmt.Sprintf("address (%s) is not associated with a player", e.Address)
	case "wrong_player":
		return fmt.Sprintf("address (%s) is associated with player %s instead of player %s",
			e.Address, e.ActualPlayerId, e.ExpectedPlayerId)
	case "proof_mismatch":
		return fmt.Sprintf("proof mismatch for address %s", e.Address)
	case "signature_invalid":
		return "proof signature verification failure"
	case "already_registered":
		return fmt.Sprintf("address (%s) already has an account", e.Address)
	default:
		return fmt.Sprintf("address validation failed for %s: %s", e.Address, e.Reason)
	}
}

func (e *AddressValidationError) Code() uint32 { return 1750 }

func (e *AddressValidationError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "address_validation",
		"address", e.Address,
		"expected_player_id", e.ExpectedPlayerId,
		"actual_player_id", e.ActualPlayerId,
		"reason", e.Reason,
	}
}

func (e *AddressValidationError) Unwrap() error { return ErrAddressValidation }

// =============================================================================
// 17. GuildMembershipError
// =============================================================================

// GuildMembershipError indicates a guild membership issue.
type GuildMembershipError struct {
	GuildId         string
	PlayerId        string
	JoinType        string // "invite", "request", "proxy", "direct"
	Reason          string // "wrong_type", "already_member", "not_allowed", "minimum_not_met", "not_member", "is_owner"
	InfusionId      string // For infusion-related errors (optional)
	ReactorId       string // For reactor-related errors (optional)
	MinimumRequired uint64 // For minimum not met (optional)
	ActualAmount    uint64 // For minimum not met (optional)
}

func NewGuildMembershipError(guildId, playerId, reason string) *GuildMembershipError {
	return &GuildMembershipError{
		GuildId:  guildId,
		PlayerId: playerId,
		Reason:   reason,
	}
}

func (e *GuildMembershipError) WithJoinType(joinType string) *GuildMembershipError {
	e.JoinType = joinType
	return e
}

func (e *GuildMembershipError) WithInfusion(infusionId string) *GuildMembershipError {
	e.InfusionId = infusionId
	return e
}

func (e *GuildMembershipError) WithReactor(reactorId string) *GuildMembershipError {
	e.ReactorId = reactorId
	return e
}

func (e *GuildMembershipError) WithMinimum(required, actual uint64) *GuildMembershipError {
	e.MinimumRequired = required
	e.ActualAmount = actual
	return e
}

func (e *GuildMembershipError) Error() string {
	switch e.Reason {
	case "already_member":
		return fmt.Sprintf("player (%s) already a member of guild (%s)", e.PlayerId, e.GuildId)
	case "not_member":
		return fmt.Sprintf("player (%s) not a member of guild (%s)", e.PlayerId, e.GuildId)
	case "is_owner":
		return fmt.Sprintf("player (%s) is the owner of guild (%s), cannot be removed", e.PlayerId, e.GuildId)
	case "wrong_type":
		return fmt.Sprintf("membership application is incorrect type for %s approval", e.JoinType)
	case "minimum_not_met":
		return "join infusion minimum not met"
	case "not_allowed":
		return fmt.Sprintf("guild (%s) is not currently allowing %s", e.GuildId, e.JoinType)
	default:
		return fmt.Sprintf("guild membership error for player (%s) in guild (%s): %s", e.PlayerId, e.GuildId, e.Reason)
	}
}

func (e *GuildMembershipError) Code() uint32 { return 1501 }

func (e *GuildMembershipError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "guild_membership",
		"guild_id", e.GuildId,
		"player_id", e.PlayerId,
		"join_type", e.JoinType,
		"reason", e.Reason,
		"infusion_id", e.InfusionId,
		"reactor_id", e.ReactorId,
		"minimum_required", e.MinimumRequired,
		"actual_amount", e.ActualAmount,
	}
}

func (e *GuildMembershipError) Unwrap() error { return ErrGuildMembership }

// =============================================================================
// 18. GuildUpdateError
// =============================================================================

// GuildUpdateError indicates a guild update failure.
type GuildUpdateError struct {
	GuildId  string
	PlayerId string
	Field    string // "owner", "endpoint", "entry_substation", "join_minimum"
	Reason   string
	NewOwner string // For owner change errors (optional)
}

func NewGuildUpdateError(guildId, playerId, field, reason string) *GuildUpdateError {
	return &GuildUpdateError{
		GuildId:  guildId,
		PlayerId: playerId,
		Field:    field,
		Reason:   reason,
	}
}

func (e *GuildUpdateError) WithNewOwner(newOwner string) *GuildUpdateError {
	e.NewOwner = newOwner
	return e
}

func (e *GuildUpdateError) Error() string {
	if e.Field == "owner" && e.NewOwner != "" {
		return fmt.Sprintf("guild could not change to new owner (%s) because they weren't found", e.NewOwner)
	}
	return fmt.Sprintf("calling player (%s) has no permissions to update guild (%s)", e.PlayerId, e.GuildId)
}

func (e *GuildUpdateError) Code() uint32 { return 1500 }

func (e *GuildUpdateError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "guild_update",
		"guild_id", e.GuildId,
		"player_id", e.PlayerId,
		"field", e.Field,
		"reason", e.Reason,
		"new_owner", e.NewOwner,
	}
}

func (e *GuildUpdateError) Unwrap() error { return ErrGuildUpdate }

// =============================================================================
// 19. AllocationError
// =============================================================================

// AllocationError indicates an allocation operation failure.
type AllocationError struct {
	AllocationId      string
	SourceId          string
	DestinationId     string // For destination-related errors (optional)
	Reason            string // "capacity_exceeded", "automated_conflict", "immutable_source", "immutable_index", "immutable_type", "same_destination", "not_exists", "not_connected"
	Field             string // Which field was problematic (optional)
	OldValue          string
	NewValue          string
	AvailableCapacity uint64 // For capacity errors (optional)
	RequestedPower    uint64 // For capacity errors (optional)
}

func NewAllocationError(sourceId, reason string) *AllocationError {
	return &AllocationError{
		SourceId: sourceId,
		Reason:   reason,
	}
}

func (e *AllocationError) WithAllocation(allocationId string) *AllocationError {
	e.AllocationId = allocationId
	return e
}

func (e *AllocationError) WithDestination(destinationId string) *AllocationError {
	e.DestinationId = destinationId
	return e
}

func (e *AllocationError) WithCapacity(available, requested uint64) *AllocationError {
	e.AvailableCapacity = available
	e.RequestedPower = requested
	return e
}

func (e *AllocationError) WithFieldChange(field, oldVal, newVal string) *AllocationError {
	e.Field = field
	e.OldValue = oldVal
	e.NewValue = newVal
	return e
}

func (e *AllocationError) Error() string {
	switch e.Reason {
	case "capacity_exceeded":
		return fmt.Sprintf("allocation source (%s) does not have capacity (%d) for power (%d)",
			e.SourceId, e.AvailableCapacity, e.RequestedPower)
	case "automated_conflict":
		return fmt.Sprintf("allocation source (%s) cannot have automated allocation with other allocations in place", e.SourceId)
	case "immutable_source":
		return fmt.Sprintf("source object (%s vs %s) should never change during allocation update", e.OldValue, e.NewValue)
	case "immutable_index":
		return fmt.Sprintf("allocation index (%s vs %s) should never change during allocation update", e.OldValue, e.NewValue)
	case "immutable_type":
		return fmt.Sprintf("allocation type (%s vs %s) should never change during allocation update", e.OldValue, e.NewValue)
	case "same_destination":
		return fmt.Sprintf("destination substation (%s) cannot change to same destination", e.DestinationId)
	case "not_exists":
		return "trying to set an allocation that doesn't exist yet"
	case "not_connected":
		return fmt.Sprintf("allocation (%s) must not be connected to a substation during transfer", e.AllocationId)
	default:
		return fmt.Sprintf("allocation error on source (%s): %s", e.SourceId, e.Reason)
	}
}

func (e *AllocationError) Code() uint32 { return 1550 }

func (e *AllocationError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "allocation",
		"allocation_id", e.AllocationId,
		"source_id", e.SourceId,
		"destination_id", e.DestinationId,
		"reason", e.Reason,
		"field", e.Field,
		"old_value", e.OldValue,
		"new_value", e.NewValue,
		"available_capacity", e.AvailableCapacity,
		"requested_power", e.RequestedPower,
	}
}

func (e *AllocationError) Unwrap() error { return ErrAllocationCreate }

// =============================================================================
// 20. ReactorError
// =============================================================================

// ReactorError indicates a reactor operation failure.
type ReactorError struct {
	Operation     string // "infuse", "defuse", "migrate", "cancel_defusion", "required"
	ReactorId     string
	Address       string // Delegator or validator address (optional)
	AddressType   string // "delegator" or "validator" (optional)
	Denom         string
	ExpectedDenom string
	Amount        uint64
	Balance       uint64 // For balance comparison (optional)
	Height        int64  // For height-related errors (optional)
	Reason        string
}

func NewReactorError(operation, reason string) *ReactorError {
	return &ReactorError{
		Operation: operation,
		Reason:    reason,
	}
}

func (e *ReactorError) WithReactor(reactorId string) *ReactorError {
	e.ReactorId = reactorId
	return e
}

func (e *ReactorError) WithAddress(address, addressType string) *ReactorError {
	e.Address = address
	e.AddressType = addressType
	return e
}

func (e *ReactorError) WithDenom(denom, expected string) *ReactorError {
	e.Denom = denom
	e.ExpectedDenom = expected
	return e
}

func (e *ReactorError) WithAmount(amount uint64) *ReactorError {
	e.Amount = amount
	return e
}

func (e *ReactorError) WithBalance(balance uint64) *ReactorError {
	e.Balance = balance
	return e
}

func (e *ReactorError) WithHeight(height int64) *ReactorError {
	e.Height = height
	return e
}

func (e *ReactorError) Error() string {
	switch e.Reason {
	case "invalid_address":
		return fmt.Sprintf("invalid %s address: %s", e.AddressType, e.Address)
	case "invalid_amount":
		return "invalid delegation amount"
	case "invalid_denom":
		return fmt.Sprintf("invalid coin denomination: got %s, expected %s", e.Denom, e.ExpectedDenom)
	case "invalid_height":
		return "invalid height"
	case "balance_exceeded":
		return "amount is greater than the unbonding delegation entry balance"
	case "already_processed":
		return "unbonding delegation is already processed"
	case "entry_not_found":
		return fmt.Sprintf("unbonding delegation entry not found at block height %d", e.Height)
	case "required":
		return fmt.Sprintf("reactor required but none associated with %s", e.Address)
	default:
		return fmt.Sprintf("reactor %s failed: %s", e.Operation, e.Reason)
	}
}

func (e *ReactorError) Code() uint32 { return 1600 }

func (e *ReactorError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "reactor",
		"operation", e.Operation,
		"reactor_id", e.ReactorId,
		"address", e.Address,
		"address_type", e.AddressType,
		"denom", e.Denom,
		"expected_denom", e.ExpectedDenom,
		"amount", e.Amount,
		"balance", e.Balance,
		"height", e.Height,
		"reason", e.Reason,
	}
}

func (e *ReactorError) Unwrap() error { return ErrReactor }

// =============================================================================
// 21. WorkFailureError
// =============================================================================

// WorkFailureError indicates a proof-of-work verification failure.
type WorkFailureError struct {
	Operation string // "mine", "refine", "build", "raid"
	StructId  string
	PlanetId  string // Optional
	HashInput string
}

func NewWorkFailureError(operation, structId, hashInput string) *WorkFailureError {
	return &WorkFailureError{
		Operation: operation,
		StructId:  structId,
		HashInput: hashInput,
	}
}

func (e *WorkFailureError) WithPlanet(planetId string) *WorkFailureError {
	e.PlanetId = planetId
	return e
}

func (e *WorkFailureError) Error() string {
	return fmt.Sprintf("work failure for input (%s) when trying to %s on struct %s", e.HashInput, e.Operation, e.StructId)
}

func (e *WorkFailureError) Code() uint32 { return 1800 }

func (e *WorkFailureError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "work_failure",
		"operation", e.Operation,
		"struct_id", e.StructId,
		"planet_id", e.PlanetId,
		"hash_input", e.HashInput,
	}
}

func (e *WorkFailureError) Unwrap() error { return ErrWorkFailure }

// =============================================================================
// 22. ProviderAccessError
// =============================================================================

// ProviderAccessError indicates a provider access denial.
type ProviderAccessError struct {
	ProviderId   string
	AccessPolicy string // "openMarket", "guildMarket", "closedMarket"
	PlayerId     string
	GuildId      string
	Reason       string // "no_account", "guild_not_approved", "market_closed"
}

func NewProviderAccessError(providerId, reason string) *ProviderAccessError {
	return &ProviderAccessError{
		ProviderId: providerId,
		Reason:     reason,
	}
}

func (e *ProviderAccessError) WithPlayer(playerId string) *ProviderAccessError {
	e.PlayerId = playerId
	return e
}

func (e *ProviderAccessError) WithGuild(guildId string) *ProviderAccessError {
	e.GuildId = guildId
	return e
}

func (e *ProviderAccessError) WithPolicy(policy string) *ProviderAccessError {
	e.AccessPolicy = policy
	return e
}

func (e *ProviderAccessError) Error() string {
	switch e.Reason {
	case "no_account":
		return fmt.Sprintf("calling address has no account for provider (%s)", e.ProviderId)
	case "guild_not_approved":
		return fmt.Sprintf("calling account is not a member of an approved guild for provider (%s)", e.ProviderId)
	case "market_closed":
		return fmt.Sprintf("provider (%s) is not accepting new agreements", e.ProviderId)
	default:
		return fmt.Sprintf("provider (%s) access denied: %s", e.ProviderId, e.Reason)
	}
}

func (e *ProviderAccessError) Code() uint32 { return 1700 }

func (e *ProviderAccessError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "provider_access",
		"provider_id", e.ProviderId,
		"access_policy", e.AccessPolicy,
		"player_id", e.PlayerId,
		"guild_id", e.GuildId,
		"reason", e.Reason,
	}
}

func (e *ProviderAccessError) Unwrap() error { return ErrProviderAccess }

// =============================================================================
// 23. ParameterValidationError
// =============================================================================

// ParameterValidationError indicates a parameter validation failure.
type ParameterValidationError struct {
	Parameter    string // "capacity", "duration"
	Value        uint64
	Minimum      uint64 // Optional
	Maximum      uint64 // Optional
	ProviderId   string // Optional
	SubstationId string // Optional, for capacity checks
	Reason       string // "below_minimum", "above_maximum", "exceeds_available"
}

func NewParameterValidationError(parameter string, value uint64, reason string) *ParameterValidationError {
	return &ParameterValidationError{
		Parameter: parameter,
		Value:     value,
		Reason:    reason,
	}
}

func (e *ParameterValidationError) WithRange(min, max uint64) *ParameterValidationError {
	e.Minimum = min
	e.Maximum = max
	return e
}

func (e *ParameterValidationError) WithProvider(providerId string) *ParameterValidationError {
	e.ProviderId = providerId
	return e
}

func (e *ParameterValidationError) WithSubstation(substationId string) *ParameterValidationError {
	e.SubstationId = substationId
	return e
}

func (e *ParameterValidationError) Error() string {
	switch e.Reason {
	case "below_minimum":
		return fmt.Sprintf("%s (%d) cannot be lower than minimum %s (%d)", e.Parameter, e.Value, e.Parameter, e.Minimum)
	case "above_maximum":
		return fmt.Sprintf("%s (%d) cannot be greater than maximum %s (%d)", e.Parameter, e.Value, e.Parameter, e.Maximum)
	case "exceeds_available":
		return fmt.Sprintf("desired %s (%d) is beyond what substation (%s) can support (%d)",
			e.Parameter, e.Value, e.SubstationId, e.Maximum)
	default:
		return fmt.Sprintf("parameter %s validation failed: %s", e.Parameter, e.Reason)
	}
}

func (e *ParameterValidationError) Code() uint32 { return 1710 }

func (e *ParameterValidationError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "parameter_validation",
		"parameter", e.Parameter,
		"value", e.Value,
		"minimum", e.Minimum,
		"maximum", e.Maximum,
		"provider_id", e.ProviderId,
		"substation_id", e.SubstationId,
		"reason", e.Reason,
	}
}

func (e *ParameterValidationError) Unwrap() error { return ErrParameterValidation }

// =============================================================================
// 24. PlanetStateError
// =============================================================================

// PlanetStateError indicates an invalid planet state for an operation.
type PlanetStateError struct {
	PlanetId string
	State    string // "complete", "empty", "has_ore"
	Action   string // "mine", "explore", "infuse"
}

func NewPlanetStateError(planetId, state, action string) *PlanetStateError {
	return &PlanetStateError{
		PlanetId: planetId,
		State:    state,
		Action:   action,
	}
}

func (e *PlanetStateError) Error() string {
	switch e.State {
	case "complete":
		return fmt.Sprintf("planet (%s) is already complete, move on bud, no work to be done here", e.PlanetId)
	case "empty":
		return fmt.Sprintf("planet (%s) is empty, nothing to mine", e.PlanetId)
	case "has_ore":
		return fmt.Sprintf("new planet cannot be explored while current planet (%s) has ore available for mining", e.PlanetId)
	default:
		return fmt.Sprintf("planet (%s) cannot %s while %s", e.PlanetId, e.Action, e.State)
	}
}

func (e *PlanetStateError) Code() uint32 { return 1650 }

func (e *PlanetStateError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "planet_state",
		"planet_id", e.PlanetId,
		"state", e.State,
		"action", e.Action,
	}
}

func (e *PlanetStateError) Unwrap() error { return ErrPlanetState }

// =============================================================================
// 25. FuelInfuseError
// =============================================================================

// FuelInfuseError indicates a fuel infusion failure.
type FuelInfuseError struct {
	StructId string
	Amount   string
	Denom    string
	Reason   string // "invalid_amount", "invalid_denom", "transfer_failed"
	Details  string // Additional error details (optional)
}

func NewFuelInfuseError(structId, amount, reason string) *FuelInfuseError {
	return &FuelInfuseError{
		StructId: structId,
		Amount:   amount,
		Reason:   reason,
	}
}

func (e *FuelInfuseError) WithDenom(denom string) *FuelInfuseError {
	e.Denom = denom
	return e
}

func (e *FuelInfuseError) WithDetails(details string) *FuelInfuseError {
	e.Details = details
	return e
}

func (e *FuelInfuseError) Error() string {
	switch e.Reason {
	case "invalid_amount":
		return fmt.Sprintf("infuse amount (%s) is invalid", e.Amount)
	case "invalid_denom":
		return fmt.Sprintf("infuse amount (%s) is invalid, %s is not a fuel", e.Amount, e.Denom)
	case "transfer_failed":
		if e.Details != "" {
			return fmt.Sprintf("infuse failed: %s", e.Details)
		}
		return "infuse failed"
	default:
		return fmt.Sprintf("fuel infuse error on struct (%s): %s", e.StructId, e.Reason)
	}
}

func (e *FuelInfuseError) Code() uint32 { return 1270 }

func (e *FuelInfuseError) LogFields() []interface{} {
	return []interface{}{
		"error_type", "fuel_infuse",
		"struct_id", e.StructId,
		"amount", e.Amount,
		"denom", e.Denom,
		"reason", e.Reason,
		"details", e.Details,
	}
}

func (e *FuelInfuseError) Unwrap() error { return ErrStructInfuse }
