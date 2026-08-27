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

		/* Re-spellings of a real fleet.
		 *
		 * ParseUint accepts leading zeros, so these all reached fleet 9-1 while
		 * remaining distinct strings - and the per-fleet throttle keys off the
		 * string, so three spellings bought three moves of one fleet in a block
		 * where the rule is one. A fleet is the only object identified by a
		 * parsed number rather than by its id text, which is why nothing else
		 * aliases this way.
		 */
		"9-01",
		"9-001",
		"9-0000000001",
	} {
		_, err := types.ParseFleetId(malformed)
		require.Error(t, err, "fleet id %q must not parse", malformed)
	}
}

// Zero is a legitimate index and its canonical spelling has no padding, so the
// leading-zero rule must not swallow it.
func TestParseFleetId_ZeroIndex(t *testing.T) {
	index, err := types.ParseFleetId("9-0")
	require.NoError(t, err)
	require.Equal(t, uint64(0), index)

	_, err = types.ParseFleetId("9-00")
	require.Error(t, err, "a padded zero is still a re-spelling")
}

// CanonicalFleetId is what ParseFleetId measures against, so every id it
// produces must parse back to the same index.
func TestCanonicalFleetId_RoundTrips(t *testing.T) {
	for _, index := range []uint64{0, 1, 9, 10, 1000, 18446744073709551615} {
		id := types.CanonicalFleetId(index)
		parsed, err := types.ParseFleetId(id)
		require.NoError(t, err, "canonical id %q must parse", id)
		require.Equal(t, index, parsed)
	}
}
