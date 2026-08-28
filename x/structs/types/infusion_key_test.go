package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"structs/x/structs/types"
)

/* An infusion key round-trips only because of two properties neither half
 * declares: a destination id carries exactly one hyphen, and a bech32 address
 * carries none. Every runtime path satisfies both without trying, which is why
 * nothing checked - and why a genesis file, which assigns the fields directly,
 * is the one way a key that cannot be split back reaches state.
 */
func TestParseInfusionKey(t *testing.T) {
	destination, address, err := types.ParseInfusionKey("3-1-structs1abc")
	require.NoError(t, err)
	require.Equal(t, "3-1", destination)
	require.Equal(t, "structs1abc", address)

	for _, malformed := range []string{
		"",                 // empty
		"-",                // the key an empty destination and address produce
		"3-1",              // no address
		"structs1abc",      // no destination
		"3-1-structs1-abc", // a hyphen in the address: four parts
		"3-1-",             // empty address
		"-1-structs1abc",   // empty object type
		"3--structs1abc",   // empty index
	} {
		_, _, err := types.ParseInfusionKey(malformed)
		require.Error(t, err, "infusion key %q must not parse", malformed)
	}
}

// ValidInfusionParts is what genesis validation asks, and it must agree with
// what the keeper will later be able to resolve.
func TestValidInfusionParts(t *testing.T) {
	require.True(t, types.ValidInfusionParts("3-1", "structs1abc"))

	require.False(t, types.ValidInfusionParts("", ""),
		"empty parts join to \"-\", which is the key that panicked the EndBlocker")
	require.False(t, types.ValidInfusionParts("3-1", ""))
	require.False(t, types.ValidInfusionParts("", "structs1abc"))
	require.False(t, types.ValidInfusionParts("3-1", "has-hyphen"),
		"a hyphen in the address splits the key into four parts")
	require.False(t, types.ValidInfusionParts("no-object-type", "structs1abc"))
}

// Anything BuildInfusionKey produces from valid parts must parse back to them.
func TestInfusionKeyRoundTrips(t *testing.T) {
	for _, pair := range [][2]string{
		{"3-1", "structs1abc"},
		{"6-99999", "structs1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq"},
	} {
		destination, address, err := types.ParseInfusionKey(types.BuildInfusionKey(pair[0], pair[1]))
		require.NoError(t, err)
		require.Equal(t, pair[0], destination)
		require.Equal(t, pair[1], address)
	}
}
