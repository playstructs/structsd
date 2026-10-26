package app

import (
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

/* Regression suite for an allowlist entry that validates and then matches
 * nothing.
 *
 * Bech32 is case-insensitive: an all-uppercase address decodes fine and
 * re-encodes lowercase. The allowlist was keyed by the text the operator
 * supplied, while the lookup in prepForZeroHeightGenesis is against
 * addr.String() on an address rebuilt from validator-key bytes - always the
 * canonical lowercase form. So an uppercase entry passed validation, stored
 * itself under a key nothing would ever match, and jailed the very validator it
 * was written to protect. Silently, because the entry was valid.
 *
 * Internal, and against the builder rather than the export, because the export
 * needs IBC transfer state the test app does not build. The builder is where the
 * decision lives, which is what makes that acceptable rather than a dodge.
 */

func testValAddr(t *testing.T) sdk.ValAddress {
	t.Helper()
	return sdk.ValAddress(ed25519.GenPrivKey().PubKey().Address())
}

func TestBuildJailAllowlist_UppercaseMatchesCanonical(t *testing.T) {
	valAddr := testValAddr(t)
	canonical := valAddr.String()
	upper := strings.ToUpper(canonical)
	require.NotEqual(t, canonical, upper, "fixture sanity: the two spellings must differ")

	allowed, err := buildJailAllowlist([]string{upper})
	require.NoError(t, err, "uppercase bech32 is valid and must be accepted")

	require.True(t, allowed[canonical],
		"the lookup is by canonical address, so an uppercase entry must land under that key or it protects nobody")
}

// Mixed case is the same problem in a form an operator is more likely to
// produce by hand.
func TestBuildJailAllowlist_MixedCaseMatchesCanonical(t *testing.T) {
	valAddr := testValAddr(t)
	canonical := valAddr.String()

	// Upper-case only the data part; the HRP stays lowercase.
	idx := strings.LastIndex(canonical, "1")
	mixed := canonical[:idx+1] + strings.ToUpper(canonical[idx+1:])

	allowed, err := buildJailAllowlist([]string{mixed})
	if err != nil {
		// Bech32 forbids mixing cases across the whole string; if the decoder
		// refuses this, that is a correct rejection and there is nothing to key.
		t.Skipf("mixed-case bech32 is rejected by the decoder: %v", err)
	}
	require.True(t, allowed[canonical])
}

// Two spellings of one validator must be one entry, not two.
func TestBuildJailAllowlist_AliasesCollapse(t *testing.T) {
	valAddr := testValAddr(t)
	canonical := valAddr.String()

	allowed, err := buildJailAllowlist([]string{canonical, strings.ToUpper(canonical)})
	require.NoError(t, err)
	require.Len(t, allowed, 1, "one validator named twice is one allowlist entry")
	require.True(t, allowed[canonical])
}

// The ordinary case must be untouched by the canonicalisation.
func TestBuildJailAllowlist_CanonicalUnchanged(t *testing.T) {
	first, second := testValAddr(t), testValAddr(t)

	allowed, err := buildJailAllowlist([]string{first.String(), second.String()})
	require.NoError(t, err)
	require.Len(t, allowed, 2)
	require.True(t, allowed[first.String()])
	require.True(t, allowed[second.String()])
}

// A typo is an operator error and must be reported, not silently dropped.
func TestBuildJailAllowlist_RejectsMalformed(t *testing.T) {
	_, err := buildJailAllowlist([]string{"not-a-validator-address"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "allowlist")
}
