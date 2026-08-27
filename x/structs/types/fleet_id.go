package types

import (
	"fmt"
	"strconv"
	"strings"
)

/* ParseFleetId splits a fleet id into its index, or reports why it cannot.
 *
 * This lives in types rather than in the keeper because GenesisState.Validate
 * has to apply exactly the same rule the keeper does, and it cannot import the
 * keeper. Two spellings of "is this a fleet id" would drift, and the whole point
 * of the check is that a file which validates must also import.
 *
 * The prefix is derived from ObjectType_fleet rather than written out, so a
 * renumbered enum moves both ends at once.
 */
func ParseFleetId(fleetId string) (uint64, error) {
	prefix := fmt.Sprintf("%d-", ObjectType_fleet)

	if !strings.HasPrefix(fleetId, prefix) {
		return 0, NewObjectNotFoundError("fleet", fleetId)
	}

	parts := strings.Split(fleetId, "-")
	if len(parts) != 2 {
		return 0, NewObjectNotFoundError("fleet", fleetId)
	}

	index, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return 0, NewObjectNotFoundError("fleet", fleetId)
	}

	/* Only the canonical spelling resolves.
	 *
	 * A fleet is the one object identified by a parsed number rather than by its
	 * id string: GetFleetById parses the suffix and GetFleet keys the cache and
	 * the store by that index. Everything else - structs, players, substations -
	 * is keyed by the raw string, so a re-spelt id simply fails to load.
	 *
	 * ParseUint accepts leading zeros, so "9-1", "9-01" and "9-001" all reached
	 * the same fleet while remaining three distinct strings. That mattered
	 * wherever the string, not the fleet, is the identity: the per-fleet throttle
	 * keys off the wire value, so three spellings bought three moves of one fleet
	 * in a block where the rule is one.
	 *
	 * Rejecting rather than normalising, so there is a single spelling of a fleet
	 * everywhere - in a throttle key, in an event, in a client's records - rather
	 * than one canonical form and an unknown number of accepted aliases.
	 */
	if fleetId != CanonicalFleetId(index) {
		return 0, NewObjectNotFoundError("fleet", fleetId)
	}

	return index, nil
}

// CanonicalFleetId is the one spelling of a fleet id, and the only one
// ParseFleetId accepts. It matches keeper.GetObjectID for ObjectType_fleet.
func CanonicalFleetId(index uint64) string {
	return fmt.Sprintf("%d-%d", ObjectType_fleet, index)
}
