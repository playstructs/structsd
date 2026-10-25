package types

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/unicode/norm"
)

// GO-2026-5970 affects norm.Iter on malformed UTF-8. Keep testing our actual
// callers while x/text remains consensus-pinned. A subprocess bounds a future
// regression without leaving a stuck normalization goroutine in the test runner.
func TestPinnedNormalizationMalformedUTF8(t *testing.T) {
	const childEnv = "STRUCTS_NORMALIZATION_SECURITY_CHILD"
	if os.Getenv(childEnv) != "1" {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		binary, err := os.Executable()
		require.NoError(t, err)
		cmd := exec.CommandContext(ctx, binary, "-test.run=^TestPinnedNormalizationMalformedUTF8$", "-test.count=1")
		cmd.Env = append(os.Environ(), childEnv+"=1")
		output, err := cmd.CombinedOutput()
		require.NoError(t, ctx.Err(), "normalization did not terminate: %s", output)
		require.NoError(t, err, "%s", output)
		return
	}
	inputs := []string{
		"\xf3\xcc\x80", // upstream regression vector
		"abc\xff", "abc\xe2\x82", "abc\xc0\xaf", "abc\xed\xa0\x80",
	}
	validators := map[string]func(string) error{
		"player": ValidatePlayerName,
		"entity": ValidateEntityName,
		"planet": ValidatePlanetName,
	}
	for _, input := range inputs {
		for name, validate := range validators {
			err := validate(input)
			require.ErrorContains(t, err, "invalid UTF-8", "%s accepted %x", name, input)
		}
		// These Form APIs, unlike Iter, are used by Structs and the IDNA dependency.
		require.Equal(t, []byte(norm.NFC.String(input)), norm.NFC.Bytes([]byte(input)))
		_ = norm.NFC.IsNormalString(input)
		_ = norm.NFC.QuickSpan([]byte(input))
		_ = NormalizeName(input) // also used for historical/genesis index rebuilding
	}
	// Preserve existing index behavior; do not silently re-key historical names.
	require.Equal(t, "\ufffd\u0300", NormalizeName("\xf3\xcc\x80"))
}
