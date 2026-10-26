package types

import (
	"fmt"
	"strconv"
	"strings"
)

// ValidatePermissionCleanupQueue permits only deleted objects from namespaces
// whose teardown schedules cleanup. A live object must never be imported into
// this queue, since the worker deletes permission rows without authorizing them.
func (gs GenesisState) ValidatePermissionCleanupQueue() error {
	if len(gs.PermissionCleanupQueue) == 0 {
		return nil
	}
	live := make(map[string]bool)
	for _, object := range gs.AllocationList {
		live[object.Id] = true
	}
	for _, object := range gs.AgreementList {
		live[object.Id] = true
	}
	for _, object := range gs.ProviderList {
		live[object.Id] = true
	}
	for _, object := range gs.SubstationList {
		live[object.Id] = true
	}
	seen := make(map[string]bool)
	for _, id := range gs.PermissionCleanupQueue {
		parts := strings.Split(id, "-")
		if len(parts) != 2 {
			return fmt.Errorf("invalid permission cleanup object %q", id)
		}
		kind, kindErr := strconv.ParseUint(parts[0], 10, 32)
		index, indexErr := strconv.ParseUint(parts[1], 10, 64)
		if kindErr != nil || indexErr != nil || id != fmt.Sprintf("%d-%d", kind, index) {
			return fmt.Errorf("invalid permission cleanup object %q", id)
		}
		var nextIndex uint64
		switch ObjectType(kind) {
		case ObjectType_allocation, ObjectType_agreement:
			// Agreements share the index of their backing allocation.
			nextIndex = gs.AllocationCount
		case ObjectType_provider:
			nextIndex = gs.ProviderCount
		case ObjectType_substation:
			nextIndex = gs.SubstationCount
		default:
			return fmt.Errorf("unsupported permission cleanup object %q", id)
		}
		if live[id] {
			return fmt.Errorf("permission cleanup targets live object %q", id)
		}
		if index >= nextIndex {
			return fmt.Errorf("permission cleanup object %q has not been allocated below next index %d", id, nextIndex)
		}
		if seen[id] {
			return fmt.Errorf("duplicate permission cleanup object %q", id)
		}
		seen[id] = true
	}
	return nil
}
