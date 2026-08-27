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

	return index, nil
}
