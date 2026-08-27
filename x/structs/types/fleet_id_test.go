package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"structs/x/structs/types"
)

/* ParseFleetId is the single rule behind both GetFleetById and the genesis
 * validation of FleetList. It is one function precisely because two spellings
 * would drift, and the property that matters is that a file which validates also
 * imports.
 */
func TestParseFleetId(t *testing.T) {
	index, err := types.ParseFleetId("9-42")
	require.NoError(t, err)
	require.Equal(t, uint64(42), index)

	for _, malformed := range []string{
		"",                          // empty
		"42",                        // no object-type prefix
		"8-42",                      // some other object type
		"9-",                        // no index
		"9-abc",                     // index is not a number
		"9-42-7",                    // too many segments
		"9--42",                     // empty segment
		"9--1",                      // negative
		" 9-42",                     // leading space
		"9-99999999999999999999999", // does not fit uint64
	} {
		_, err := types.ParseFleetId(malformed)
		require.Error(t, err, "fleet id %q must not parse", malformed)
	}
}
