package types

import (
	"errors"
	"testing"
)

// =============================================================================
// Test Helpers
// =============================================================================

// assertContains checks if substr is in s
func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if len(substr) == 0 {
		return
	}
	found := false
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected %q to contain %q", s, substr)
	}
}

// assertErrorIs checks if err wraps target
func assertErrorIs(t *testing.T, err, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Errorf("expected error to wrap %v, got %v", target, err)
	}
}

// assertCode checks if the error code matches
func assertCode(t *testing.T, got, expected uint32) {
	t.Helper()
	if got != expected {
		t.Errorf("expected code %d, got %d", expected, got)
	}
}

// assertLogFieldsContain checks if LogFields contains the expected keys
func assertLogFieldsContain(t *testing.T, fields []interface{}, keys ...string) {
	t.Helper()
	fieldMap := make(map[string]bool)
	for i := 0; i < len(fields)-1; i += 2 {
		if key, ok := fields[i].(string); ok {
			fieldMap[key] = true
		}
	}
	for _, key := range keys {
		if !fieldMap[key] {
			t.Errorf("expected LogFields to contain key %q", key)
		}
	}
}

// =============================================================================
// ObjectNotFoundError Tests
// =============================================================================

func TestObjectNotFoundError(t *testing.T) {
	err := NewObjectNotFoundError("player", "player-123")

	// Test Error() message
	assertContains(t, err.Error(), "player")
	assertContains(t, err.Error(), "player-123")
	assertContains(t, err.Error(), "not found")

	// Test Code()
	assertCode(t, err.Code(), 1050)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "object_type", "object_id")

	// Test Unwrap()
	assertErrorIs(t, err, ErrObjectNotFound)
}

func TestObjectNotFoundError_WithMethods(t *testing.T) {
	err := NewObjectNotFoundError("struct", "struct-456").
		WithIndex(42).
		WithContext("during migration")

	assertContains(t, err.Error(), "struct")
	assertContains(t, err.Error(), "during migration")

	if err.ObjectIndex != 42 {
		t.Errorf("expected ObjectIndex 42, got %d", err.ObjectIndex)
	}
}

// =============================================================================
// InsufficientChargeError Tests
// =============================================================================

func TestInsufficientChargeError(t *testing.T) {
	err := NewInsufficientChargeError("player-1", 100, 50, "build")

	// Test Error() message
	assertContains(t, err.Error(), "player-1")
	assertContains(t, err.Error(), "100")
	assertContains(t, err.Error(), "50")

	// Test Code()
	assertCode(t, err.Code(), 1200)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "player_id", "required", "available", "action")

	// Test Unwrap()
	assertErrorIs(t, err, ErrInsufficientCharge)
}

func TestInsufficientChargeError_WithStructType(t *testing.T) {
	err := NewInsufficientChargeError("player-1", 200, 100, "activate").
		WithStructType(5)

	assertContains(t, err.Error(), "struct type (5)")
	assertContains(t, err.Error(), "activate")

	if err.StructType != 5 {
		t.Errorf("expected StructType 5, got %d", err.StructType)
	}
}

// =============================================================================
// PermissionError Tests
// =============================================================================

func TestPermissionError(t *testing.T) {
	err := NewPermissionError("address", "cosmos1abc", "substation", "sub-1", 8, "manage")

	// Test Error() message
	assertContains(t, err.Error(), "address")
	assertContains(t, err.Error(), "cosmos1abc")
	assertContains(t, err.Error(), "substation")
	assertContains(t, err.Error(), "sub-1")

	// Test Code()
	assertCode(t, err.Code(), 1100)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "caller_type", "caller_id", "target_type", "target_id", "permission", "action")

	// Test Unwrap()
	assertErrorIs(t, err, ErrPermission)
}

func TestPermissionError_NoTarget(t *testing.T) {
	err := NewPermissionError("player", "player-1", "", "", 4, "play")

	// Should not contain target info when empty
	assertContains(t, err.Error(), "player")
	assertContains(t, err.Error(), "play permission")
}

func TestPermissionError_WithMethods(t *testing.T) {
	err := NewPermissionError("player", "player-1", "guild", "guild-1", 16, "invite").
		WithRequiredLevel(3).
		WithGuildId("guild-2").
		WithAssociationTarget("player-2")

	if err.RequiredLevel != 3 {
		t.Errorf("expected RequiredLevel 3, got %d", err.RequiredLevel)
	}
	if err.GuildId != "guild-2" {
		t.Errorf("expected GuildId guild-2, got %s", err.GuildId)
	}
	if err.AssociationTargetId != "player-2" {
		t.Errorf("expected AssociationTargetId player-2, got %s", err.AssociationTargetId)
	}
}

// =============================================================================
// PlayerPowerError Tests
// =============================================================================

func TestPlayerPowerError(t *testing.T) {
	err := NewPlayerPowerError("player-1", "offline")

	// Test Error() message
	assertContains(t, err.Error(), "player-1")
	assertContains(t, err.Error(), "offline")

	// Test Code()
	assertCode(t, err.Code(), 1201)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "player_id", "reason")

	// Test Unwrap()
	assertErrorIs(t, err, ErrPlayerPowerOffline)
}

func TestPlayerPowerError_WithCapacity(t *testing.T) {
	err := NewPlayerPowerError("player-1", "capacity_exceeded").
		WithCapacity(100, 50)

	if err.Required != 100 {
		t.Errorf("expected Required 100, got %d", err.Required)
	}
	if err.Available != 50 {
		t.Errorf("expected Available 50, got %d", err.Available)
	}
}

// =============================================================================
// PlayerAffordabilityError Tests
// =============================================================================

func TestPlayerAffordabilityError(t *testing.T) {
	err := NewPlayerAffordabilityError("player-1", "agreement_open", "1000alpha")

	// Test Error() message
	assertContains(t, err.Error(), "player-1")
	assertContains(t, err.Error(), "agreement_open")

	// Test Code()
	assertCode(t, err.Code(), 1203)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "player_id", "action", "required")

	// Test Unwrap()
	assertErrorIs(t, err, ErrPlayerAffordability)
}

// =============================================================================
// StructStateError Tests
// =============================================================================

func TestStructStateError(t *testing.T) {
	err := NewStructStateError("struct-1", "built", "building", "build_complete")

	// Test Error() message
	assertContains(t, err.Error(), "struct-1")
	assertContains(t, err.Error(), "built")
	assertContains(t, err.Error(), "building")

	// Test Code()
	assertCode(t, err.Code(), 1250)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "struct_id", "current_state", "required_state", "action")

	// Test Unwrap()
	assertErrorIs(t, err, ErrStructState)
}

// =============================================================================
// StructCapabilityError Tests
// =============================================================================

func TestStructCapabilityError(t *testing.T) {
	err := NewStructCapabilityError("struct-1", "stealth")

	// Test Error() message
	assertContains(t, err.Error(), "struct-1")
	assertContains(t, err.Error(), "stealth")

	// Test Code()
	assertCode(t, err.Code(), 1260)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "struct_id", "capability")

	// Test Unwrap()
	assertErrorIs(t, err, ErrStructCapability)
}

// =============================================================================
// FleetCommandError Tests
// =============================================================================

func TestFleetCommandError(t *testing.T) {
	err := NewFleetCommandError("fleet-1", "no_command_struct")

	// Test Error() message
	assertContains(t, err.Error(), "fleet-1")

	// Test Code()
	assertCode(t, err.Code(), 1450)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "fleet_id", "reason")

	// Test Unwrap()
	assertErrorIs(t, err, ErrFleetCommand)
}

// =============================================================================
// FleetStateError Tests
// =============================================================================

func TestFleetStateError(t *testing.T) {
	err := NewFleetStateError("fleet-1", "raiding", "build")

	// Test Error() message
	assertContains(t, err.Error(), "fleet-1")
	assertContains(t, err.Error(), "raiding")

	// Test Code()
	assertCode(t, err.Code(), 1453)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "fleet_id", "state", "action")

	// Test Unwrap()
	assertErrorIs(t, err, ErrFleetState)
}

// =============================================================================
// StructOwnershipError Tests
// =============================================================================

func TestStructOwnershipError(t *testing.T) {
	err := NewStructOwnershipError("struct-1", "fleet-1", "fleet-2")

	// Test Error() message contains location type
	assertContains(t, err.Error(), "struct owner must match")

	// Test Code()
	assertCode(t, err.Code(), 1301)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "struct_id", "expected_owner", "actual_owner")

	// Test Unwrap()
	assertErrorIs(t, err, ErrStructOwnership)
}

// =============================================================================
// StructLocationError Tests
// =============================================================================

func TestStructLocationError(t *testing.T) {
	err := NewStructLocationError(5, "space", "invalid_ambit")

	// Test Error() message
	assertContains(t, err.Error(), "struct type")
	assertContains(t, err.Error(), "5")

	// Test Code()
	assertCode(t, err.Code(), 1302)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "struct_type", "ambit", "reason")

	// Test Unwrap()
	assertErrorIs(t, err, ErrStructLocation)
}

func TestStructLocationError_WithMethods(t *testing.T) {
	err := NewStructLocationError(3, "land", "out_of_range").
		WithStruct("struct-1").
		WithLocation("planet", "planet-1")

	if err.StructId != "struct-1" {
		t.Errorf("expected StructId struct-1, got %s", err.StructId)
	}
	if err.LocationType != "planet" {
		t.Errorf("expected LocationType planet, got %s", err.LocationType)
	}
	if err.LocationId != "planet-1" {
		t.Errorf("expected LocationId planet-1, got %s", err.LocationId)
	}
}

// =============================================================================
// CombatTargetingError Tests
// =============================================================================

func TestCombatTargetingError(t *testing.T) {
	err := NewCombatTargetingError("struct-1", "struct-2", "laser", "out_of_range")

	// Test Error() message
	assertContains(t, err.Error(), "struct-1")
	assertContains(t, err.Error(), "struct-2")

	// Test Code()
	assertCode(t, err.Code(), 1400)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "attacker_id", "target_id", "weapon_system", "reason")

	// Test Unwrap()
	assertErrorIs(t, err, ErrCombatTargeting)
}

func TestCombatTargetingError_AsCounter(t *testing.T) {
	err := NewCombatTargetingError("struct-1", "struct-2", "cannon", "destroyed").
		AsCounter()

	if !err.IsCounter {
		t.Errorf("expected IsCounter true, got false")
	}
}

// =============================================================================
// StructBuildError Tests
// =============================================================================

func TestStructBuildError(t *testing.T) {
	err := NewStructBuildError(5, "fleet", "fleet-1", "slot_occupied")

	// Test Error() message
	assertContains(t, err.Error(), "fleet")
	assertContains(t, err.Error(), "fleet-1")

	// Test Code()
	assertCode(t, err.Code(), 1350)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "struct_type", "location_type", "location_id", "reason")

	// Test Unwrap()
	assertErrorIs(t, err, ErrStructBuild)
}

func TestStructBuildError_WithSlot(t *testing.T) {
	err := NewStructBuildError(7, "fleet", "fleet-1", "command_exists").
		WithSlot(3)

	if err.Slot != 3 {
		t.Errorf("expected Slot 3, got %d", err.Slot)
	}
}

// =============================================================================
// PlayerHaltedError Tests
// =============================================================================

func TestPlayerHaltedError(t *testing.T) {
	err := NewPlayerHaltedError("player-1", "struct_build")

	// Test Error() message
	assertContains(t, err.Error(), "player-1")
	assertContains(t, err.Error(), "halted")

	// Test Code()
	assertCode(t, err.Code(), 1151)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "player_id", "action")

	// Test Unwrap()
	assertErrorIs(t, err, ErrPlayerHalted)
}

func TestPlayerHaltedError_WithStruct(t *testing.T) {
	err := NewPlayerHaltedError("player-1", "activate").
		WithStruct("struct-1")

	if err.StructId != "struct-1" {
		t.Errorf("expected StructId struct-1, got %s", err.StructId)
	}
}

// =============================================================================
// PlayerRequiredError Tests
// =============================================================================

func TestPlayerRequiredError(t *testing.T) {
	err := NewPlayerRequiredError("cosmos1abc", "guild_create")

	// Test Error() message
	assertContains(t, err.Error(), "cosmos1abc")
	assertContains(t, err.Error(), "player")

	// Test Code()
	assertCode(t, err.Code(), 1150)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "address", "action")

	// Test Unwrap()
	assertErrorIs(t, err, ErrPlayerRequired)
}

// =============================================================================
// AddressValidationError Tests
// =============================================================================

func TestAddressValidationError(t *testing.T) {
	err := NewAddressValidationError("cosmos1abc", "not_registered")

	// Test Error() message
	assertContains(t, err.Error(), "cosmos1abc")
	assertContains(t, err.Error(), "not associated with a player")

	// Test Code()
	assertCode(t, err.Code(), 1750)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "address", "reason")

	// Test Unwrap()
	assertErrorIs(t, err, ErrAddressValidation)
}

func TestAddressValidationError_WithMethods(t *testing.T) {
	err := NewAddressValidationError("cosmos1abc", "wrong_player").
		WithPlayers("player-1", "player-2")

	if err.ExpectedPlayerId != "player-1" {
		t.Errorf("expected ExpectedPlayerId player-1, got %s", err.ExpectedPlayerId)
	}
	if err.ActualPlayerId != "player-2" {
		t.Errorf("expected ActualPlayerId player-2, got %s", err.ActualPlayerId)
	}
}

// =============================================================================
// GuildMembershipError Tests
// =============================================================================

func TestGuildMembershipError(t *testing.T) {
	err := NewGuildMembershipError("guild-1", "player-1", "already_member")

	// Test Error() message
	assertContains(t, err.Error(), "guild-1")
	assertContains(t, err.Error(), "player-1")

	// Test Code()
	assertCode(t, err.Code(), 1501)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "guild_id", "player_id", "reason")

	// Test Unwrap()
	assertErrorIs(t, err, ErrGuildMembership)
}

func TestGuildMembershipError_WithJoinType(t *testing.T) {
	err := NewGuildMembershipError("guild-1", "player-1", "wrong_join_type").
		WithJoinType("invite")

	if err.JoinType != "invite" {
		t.Errorf("expected JoinType invite, got %s", err.JoinType)
	}
}

// =============================================================================
// GuildUpdateError Tests
// =============================================================================

func TestGuildUpdateError(t *testing.T) {
	err := NewGuildUpdateError("guild-1", "player-1", "owner_id", "invalid_player")

	// Test Error() message
	assertContains(t, err.Error(), "guild-1")
	assertContains(t, err.Error(), "player-1")

	// Test Code()
	assertCode(t, err.Code(), 1500)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "guild_id", "player_id", "field", "reason")

	// Test Unwrap()
	assertErrorIs(t, err, ErrGuildUpdate)
}

// =============================================================================
// AllocationError Tests
// =============================================================================

func TestAllocationError(t *testing.T) {
	err := NewAllocationError("sub-1", "capacity_exceeded")

	// Test Error() message
	assertContains(t, err.Error(), "sub-1")
	assertContains(t, err.Error(), "capacity")

	// Test Code()
	assertCode(t, err.Code(), 1550)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "source_id", "reason")

	// Test Unwrap()
	assertErrorIs(t, err, ErrAllocationCreate)
}

func TestAllocationError_WithMethods(t *testing.T) {
	err := NewAllocationError("sub-1", "capacity").
		WithAllocation("alloc-1").
		WithCapacity(50, 100)

	if err.AllocationId != "alloc-1" {
		t.Errorf("expected AllocationId alloc-1, got %s", err.AllocationId)
	}
	if err.RequestedPower != 100 {
		t.Errorf("expected RequestedPower 100, got %d", err.RequestedPower)
	}
	if err.AvailableCapacity != 50 {
		t.Errorf("expected AvailableCapacity 50, got %d", err.AvailableCapacity)
	}
}

// =============================================================================
// ReactorError Tests
// =============================================================================

func TestReactorError(t *testing.T) {
	err := NewReactorError("infuse", "invalid_amount")

	// Test Error() message
	assertContains(t, err.Error(), "invalid delegation amount")

	// Test Code()
	assertCode(t, err.Code(), 1600)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "operation", "reason")

	// Test Unwrap()
	assertErrorIs(t, err, ErrReactor)
}

func TestReactorError_WithMethods(t *testing.T) {
	err := NewReactorError("defuse", "insufficient_balance").
		WithReactor("reactor-1").
		WithAmount(1000).
		WithDenom("alpha", "beta")

	if err.ReactorId != "reactor-1" {
		t.Errorf("expected ReactorId reactor-1, got %s", err.ReactorId)
	}
	if err.Amount != 1000 {
		t.Errorf("expected Amount 1000, got %d", err.Amount)
	}
	if err.Denom != "alpha" {
		t.Errorf("expected Denom alpha, got %s", err.Denom)
	}
	if err.ExpectedDenom != "beta" {
		t.Errorf("expected ExpectedDenom beta, got %s", err.ExpectedDenom)
	}
}

// =============================================================================
// WorkFailureError Tests
// =============================================================================

func TestWorkFailureError(t *testing.T) {
	err := NewWorkFailureError("build", "struct-1", "input-hash")

	// Test Error() message
	assertContains(t, err.Error(), "build")
	assertContains(t, err.Error(), "struct-1")

	// Test Code()
	assertCode(t, err.Code(), 1800)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "operation", "struct_id", "hash_input")

	// Test Unwrap()
	assertErrorIs(t, err, ErrWorkFailure)
}

// =============================================================================
// ProviderAccessError Tests
// =============================================================================

func TestProviderAccessError(t *testing.T) {
	err := NewProviderAccessError("provider-1", "closed_market")

	// Test Error() message
	assertContains(t, err.Error(), "provider-1")
	assertContains(t, err.Error(), "closed_market")

	// Test Code()
	assertCode(t, err.Code(), 1700)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "provider_id", "reason")

	// Test Unwrap()
	assertErrorIs(t, err, ErrProviderAccess)
}

func TestProviderAccessError_WithMethods(t *testing.T) {
	err := NewProviderAccessError("provider-1", "guild_not_allowed").
		WithPlayer("player-1").
		WithGuild("guild-1")

	if err.PlayerId != "player-1" {
		t.Errorf("expected PlayerId player-1, got %s", err.PlayerId)
	}
	if err.GuildId != "guild-1" {
		t.Errorf("expected GuildId guild-1, got %s", err.GuildId)
	}
}

// =============================================================================
// ParameterValidationError Tests
// =============================================================================

func TestParameterValidationError(t *testing.T) {
	err := NewParameterValidationError("capacity", 500, "above_maximum")

	// Test Error() message
	assertContains(t, err.Error(), "capacity")
	assertContains(t, err.Error(), "500")

	// Test Code()
	assertCode(t, err.Code(), 1710)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "parameter", "value", "reason")

	// Test Unwrap()
	assertErrorIs(t, err, ErrParameterValidation)
}

func TestParameterValidationError_WithMethods(t *testing.T) {
	err := NewParameterValidationError("duration", 100, "below_minimum").
		WithRange(200, 1000).
		WithSubstation("sub-1")

	if err.Minimum != 200 {
		t.Errorf("expected Minimum 200, got %d", err.Minimum)
	}
	if err.Maximum != 1000 {
		t.Errorf("expected Maximum 1000, got %d", err.Maximum)
	}
	if err.SubstationId != "sub-1" {
		t.Errorf("expected SubstationId sub-1, got %s", err.SubstationId)
	}
}

// =============================================================================
// PlanetStateError Tests
// =============================================================================

func TestPlanetStateError(t *testing.T) {
	err := NewPlanetStateError("planet-1", "complete", "explore")

	// Test Error() message
	assertContains(t, err.Error(), "planet-1")
	assertContains(t, err.Error(), "complete")

	// Test Code()
	assertCode(t, err.Code(), 1650)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "planet_id", "state", "action")

	// Test Unwrap()
	assertErrorIs(t, err, ErrPlanetState)
}

// =============================================================================
// FuelInfuseError Tests
// =============================================================================

func TestFuelInfuseError(t *testing.T) {
	err := NewFuelInfuseError("struct-1", "100", "invalid_fuel_type")

	// Test Error() message
	assertContains(t, err.Error(), "struct-1")
	assertContains(t, err.Error(), "invalid_fuel_type")

	// Test Code()
	assertCode(t, err.Code(), 1270)

	// Test LogFields()
	fields := err.LogFields()
	assertLogFieldsContain(t, fields, "error_type", "struct_id", "amount", "reason")

	// Test Unwrap()
	assertErrorIs(t, err, ErrStructInfuse)
}

// =============================================================================
// Interface Implementation Tests
// =============================================================================

func TestStructuredErrorInterface(t *testing.T) {
	// Verify all error types implement StructuredError
	var _ StructuredError = &ObjectNotFoundError{}
	var _ StructuredError = &InsufficientChargeError{}
	var _ StructuredError = &PermissionError{}
	var _ StructuredError = &PlayerPowerError{}
	var _ StructuredError = &PlayerAffordabilityError{}
	var _ StructuredError = &StructStateError{}
	var _ StructuredError = &StructCapabilityError{}
	var _ StructuredError = &FleetCommandError{}
	var _ StructuredError = &FleetStateError{}
	var _ StructuredError = &StructOwnershipError{}
	var _ StructuredError = &StructLocationError{}
	var _ StructuredError = &CombatTargetingError{}
	var _ StructuredError = &StructBuildError{}
	var _ StructuredError = &PlayerHaltedError{}
	var _ StructuredError = &PlayerRequiredError{}
	var _ StructuredError = &AddressValidationError{}
	var _ StructuredError = &GuildMembershipError{}
	var _ StructuredError = &GuildUpdateError{}
	var _ StructuredError = &AllocationError{}
	var _ StructuredError = &ReactorError{}
	var _ StructuredError = &WorkFailureError{}
	var _ StructuredError = &ProviderAccessError{}
	var _ StructuredError = &ParameterValidationError{}
	var _ StructuredError = &PlanetStateError{}
	var _ StructuredError = &FuelInfuseError{}
}

// =============================================================================
// Error Chaining Tests
// =============================================================================

func TestMethodChaining(t *testing.T) {
	// Test that method chaining returns the same pointer and allows fluent calls
	err := NewObjectNotFoundError("player", "player-1").
		WithIndex(42).
		WithContext("test context")

	if err.ObjectIndex != 42 {
		t.Errorf("expected ObjectIndex 42, got %d", err.ObjectIndex)
	}
	if err.Context != "test context" {
		t.Errorf("expected Context 'test context', got %s", err.Context)
	}

	// Verify error message still works after chaining
	assertContains(t, err.Error(), "player")
	assertContains(t, err.Error(), "test context")
}

// =============================================================================
// Code Uniqueness Tests
// =============================================================================

func TestErrorCodesAreUnique(t *testing.T) {
	codes := map[uint32]string{
		NewObjectNotFoundError("", "").Code():                "ObjectNotFoundError",
		NewInsufficientChargeError("", 0, 0, "").Code():      "InsufficientChargeError",
		NewPermissionError("", "", "", "", 0, "").Code():     "PermissionError",
		NewPlayerPowerError("", "").Code():                   "PlayerPowerError",
		NewPlayerAffordabilityError("", "", "").Code():       "PlayerAffordabilityError",
		NewStructStateError("", "", "", "").Code():           "StructStateError",
		NewStructCapabilityError("", "").Code():              "StructCapabilityError",
		NewFleetCommandError("", "").Code():                  "FleetCommandError",
		NewFleetStateError("", "", "").Code():                "FleetStateError",
		NewStructOwnershipError("", "", "").Code():           "StructOwnershipError",
		NewStructLocationError(0, "", "").Code():             "StructLocationError",
		NewCombatTargetingError("", "", "", "").Code():       "CombatTargetingError",
		NewStructBuildError(0, "", "", "").Code():            "StructBuildError",
		NewPlayerHaltedError("", "").Code():                  "PlayerHaltedError",
		NewPlayerRequiredError("", "").Code():                "PlayerRequiredError",
		NewAddressValidationError("", "").Code():             "AddressValidationError",
		NewGuildMembershipError("", "", "").Code():           "GuildMembershipError",
		NewGuildUpdateError("", "", "", "").Code():           "GuildUpdateError",
		NewAllocationError("", "").Code():                    "AllocationError",
		NewReactorError("", "").Code():                       "ReactorError",
		NewWorkFailureError("", "", "").Code():               "WorkFailureError",
		NewProviderAccessError("", "").Code():                "ProviderAccessError",
		NewParameterValidationError("", 0, "").Code():        "ParameterValidationError",
		NewPlanetStateError("", "", "").Code():               "PlanetStateError",
		NewFuelInfuseError("", "", "").Code():                "FuelInfuseError",
	}

	// This test verifies we have 25 unique codes
	if len(codes) != 25 {
		t.Errorf("expected 25 unique error codes, got %d", len(codes))
	}
}
