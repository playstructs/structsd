package types

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidatePlayerName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid short", "abc", false},
		{"valid 20 chars", "abcdefghij1234567890", false},
		{"valid with hyphens", "my-player", false},
		{"valid with underscores", "my_player", false},
		{"valid unicode", "Ñoño", false},
		{"too short", "ab", true},
		{"too long", "abcdefghij12345678901", true},
		{"contains space", "my player", true},
		{"contains apostrophe", "my'player", true},
		{"looks like object id", "1-23", true},
		{"empty", "", true},

		// Unicode hardening
		// Single combining mark on a base letter NFC-normalizes to a precomposed
		// letter (no combining mark survives) -- legal.
		{"single combining mark normalized", "abc\u0301def", false},
		// Stacked combining marks (zalgo): NFC composes the first one but the
		// extras remain as combining marks and must be rejected.
		{"zalgo stacked combining marks", "ab\u0301\u0302\u0303cd", true},
		{"zero-width joiner", "ab\u200Dcd", true},
		{"bidi override", "ab\u202Ecd", true},
		{"soft hyphen", "ab\u00ADcd", true},
		{"invalid utf-8", "ab\xff\xfecd", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePlayerName(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateEntityName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid short", "abc", false},
		{"valid with spaces", "my guild", false},
		{"valid with apostrophe", "player's guild", false},
		{"valid 20 chars", "abcdefghij1234567890", false},
		{"too short", "ab", true},
		{"too long", "abcdefghij12345678901", true},
		{"leading space", " guild", true},
		{"trailing space", "guild ", true},
		{"double space", "my  guild", true},
		{"looks like object id", "1-23", true},
		{"empty", "", true},

		// Unicode hardening
		// Letter-on-base normalization: "c" + combining acute -> precomposed
		// "ć" (U+0107) is canonical, so the combining mark disappears post-NFC
		// and the name is legal.
		{"composable combining mark normalized", "guildc\u0301a", false},
		// "d" + combining acute has no precomposed form, so the combining mark
		// survives NFC and must be rejected.
		{"non-composable combining mark", "guild\u0301", true},
		{"zalgo stacked combining marks", "gu\u0301\u0302\u0303ild", true},
		{"zero-width space", "gu\u200Bild", true},
		{"bidi override", "gu\u202Eild", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEntityName(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidatePlanetName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid short", "abc", false},
		{"valid 25 chars", "abcdefghij1234567890abcde", false},
		{"valid with spaces", "my planet", false},
		{"too short", "ab", true},
		{"too long", "abcdefghij1234567890abcdef", true},
		{"leading space", " planet", true},
		{"trailing space", "planet ", true},
		{"double space", "my  planet", true},
		{"looks like object id", "1-23", true},
		{"empty", "", true},

		{"zalgo stacked combining marks", "p\u0301\u0302\u0303lanet", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePlanetName(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidatePfp(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty string", "", false},
		{"valid https url", "https://example.com/image.png", false},
		{"valid http url", "http://example.com/avatar.jpg", false},
		{"valid ipfs cid", "ipfs://QmW2WQi7j6c7UgJTarActp7tDNikE4B2qXtFCfLPdsgaTQ", false},
		{"valid ipns", "ipns://k51qzi5uqu5dgutdk6i1ynyzgkqngpha5xpgia3a5qqp4jsh0u4csygvpvuxfvz", false},
		{"valid arweave", "ar://Q1234abcDEF_Some-arweave-id", false},
		{"max length ascii url", "https://example.com/" + strings.Repeat("a", MaxPfpLength-len("https://example.com/")), false},
		{"exceeds max length", strings.Repeat("a", MaxPfpLength+1), true},
		{"max length multibyte runes", strings.Repeat("日", MaxPfpLength), true},  // not URL, not opaque charset
		{"under rune limit but over byte limit", strings.Repeat("日", 100), true}, // not URL, not opaque charset

		// Control character rejection
		{"null byte", "https://example.com/\x00img.png", true},
		{"tab character", "https://example.com/\timg.png", true},
		{"newline", "https://example.com/\nimg.png", true},
		{"carriage return", "https://example.com/\rimg.png", true},
		{"DEL character", "https://example.com/\x7Fimg.png", true},

		// HTML/template injection
		{"angle bracket open", "<script>alert(1)</script>", true},
		{"angle bracket close", "img src=x onerror=alert(1)>", true},
		{"backtick", "https://example.com/`inject`", true},
		{"double quote", "https://example.com/\"inject\"", true},
		{"backslash", "https://example.com\\inject", true},
		{"space inside url", "https://example.com/ image.png", true},

		// Dangerous URI schemes
		{"javascript scheme", "javascript:alert(1)", true},
		{"javascript uppercase", "JavaScript:alert(1)", true},
		{"javascript mixed case", "JaVaScRiPt:alert(1)", true},
		{"data scheme", "data:text/html,<script>alert(1)</script>", true},
		{"data image", "data:image/png;base64,abc", true},
		{"vbscript scheme", "vbscript:msgbox", true},
		{"file scheme", "file:///etc/passwd", true},
		{"ftp scheme", "ftp://example.com/file", true},
		{"gopher scheme", "gopher://example.com/", true},
		{"unknown custom scheme", "weirdscheme:payload", true},

		// Bidi/zero-width spoofing
		{"bidi override in url", "https://exam\u202Eple.com/img.png", true},
		{"zero-width in url", "https://exa\u200Bmple.com/img.png", true},

		// Malformed URLs
		{"https without host", "https://", true},
		{"http without host", "http://", true},
		{"ipfs without identifier", "ipfs://", true},
		{"colon at start (empty scheme)", ":payload", true},

		// Valid opaque identifiers
		{"plain text identifier", "avatar-12345", false},
		{"hash identifier", "abc123def456", false},
		{"dotted hash", "Qm123.png", false},
		{"path-style opaque", "assets/avatar/123.png", false},
		{"opaque with disallowed char", "avatar#fragment", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePfp(tt.input)
			if tt.wantErr {
				require.Error(t, err, "expected error for input: %q", tt.input)
			} else {
				require.NoError(t, err, "unexpected error for input: %q", tt.input)
			}
		})
	}
}

func TestNormalizeName(t *testing.T) {
	require.Equal(t, "my guild", NormalizeName("My Guild"))
	require.Equal(t, "my guild", NormalizeName("  My Guild  "))
	require.Equal(t, "test", NormalizeName("TEST"))

	// NFC normalization: precomposed and decomposed forms must collapse to
	// the same canonical key so a name can't be re-registered just by
	// switching unicode encoding.
	precomposed := "café"      // é = U+00E9
	decomposed := "cafe\u0301" // e + combining acute
	require.Equal(t, NormalizeName(precomposed), NormalizeName(decomposed))
}

func TestValidatePfpClientRenderAttributes(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"empty clears", "", "", false},
		{"valid object", `{"head":1234,"background":1234}`, `{"head":1234,"background":1234}`, false},
		{"valid object with schema hash", `{"head":1234,"schema":"abc123"}`, `{"head":1234,"schema":"abc123"}`, false},
		{"empty object", `{}`, `{}`, false},
		{"whitespace compacted", "{\n  \"head\": 1234\n}", `{"head":1234}`, false},
		{"array rejected", `[1,2,3]`, "", true},
		{"scalar number rejected", `1234`, "", true},
		{"scalar string rejected", `"head"`, "", true},
		{"malformed json rejected", `{"head":}`, "", true},
		{"trailing garbage rejected", `{"head":1}x`, "", true},
		{"over byte cap rejected", `{"x":"` + strings.Repeat("a", MaxPfpClientRenderAttributesBytes) + `"}`, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidatePfpClientRenderAttributes(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// unicode16Letters are code points that Unicode 15.0.0 leaves unassigned and a
// later version classifies as letters. They are the concrete shape of the
// consensus split the pinned tables exist to prevent: under the old \p{L}
// regex a validator built with newer Unicode tables accepted a name made of
// these while every other validator rejected it, and only the accepting side
// wrote the name, so the two disagreed on the application hash.
//
// U+088F is the cheapest demonstration because it sits immediately after
// U+088E, which is a letter in 15.0.0, so the two together show exactly where
// the pinned boundary falls.
var unicode16Letters = map[string]rune{
	"U+088F arabic":        '\u088F',
	"U+105C0 todhri":       '\U000105C0',
	"U+116D0 myanmar extC": '\U000116D0',
	"U+1E5D0 ol onal":      '\U0001E5D0',
}

func TestNameValidation_RejectsPostPinnedUnicodeLetters(t *testing.T) {
	for label, r := range unicode16Letters {
		t.Run(label, func(t *testing.T) {
			name := strings.Repeat(string(r), 3)

			require.False(t, isPinnedLetter(r),
				"%s must not be a letter under the pinned tables; if it is, the tables "+
					"were regenerated against a newer Unicode version", label)

			require.Error(t, ValidatePlayerName(name), "%s must be rejected in a player name", label)
			require.Error(t, ValidateEntityName(name), "%s must be rejected in an entity name", label)
			require.Error(t, ValidatePlanetName(name), "%s must be rejected in a planet name", label)
		})
	}
}

// TestNameValidation_AcceptsPinnedBoundaryLetter is the other half: the pin
// must not be a silent tightening of the character set that was already in use.
func TestNameValidation_AcceptsPinnedBoundaryLetter(t *testing.T) {
	const lastArabicLetter = '\u088E' // assigned and category Lo in Unicode 15.0.0

	require.True(t, isPinnedLetter(lastArabicLetter),
		"U+088E is a letter in Unicode %s and must stay accepted", PinnedUnicodeVersion)

	name := strings.Repeat(string(rune(lastArabicLetter)), 3)
	require.NoError(t, ValidatePlayerName(name))
	require.NoError(t, ValidateEntityName(name))
	require.NoError(t, ValidatePlanetName(name))
}

// TestValidatePfp_FormatCharacterCheckIsPinned covers the second divergent
// site, which differs from the name validators in an important way: a URL path
// has no character allow-list, so whether a pfp is accepted depends directly on
// the format-category answer for arbitrary runes. Under the toolchain's tables
// a code point newly assigned to Cf would be accepted by old binaries and
// rejected by new ones -- the opposite direction from the name path, and just
// as much of a split.
func TestValidatePfp_FormatCharacterCheckIsPinned(t *testing.T) {
	// A format character assigned in Unicode 15.0.0 is rejected, so the check
	// is doing its job rather than passing everything.
	require.True(t, isPinnedFormat('\u2060'), "word joiner is a format character")
	require.Error(t, ValidatePfp("https://example.com/a\u2060b.png"),
		"a format character in the path must be rejected")

	// A code point unassigned in Unicode 15.0.0 is not a format character and
	// the pfp is accepted. The pinned tables are what make that answer the same
	// on every binary, whatever a later Unicode version decides to call it.
	for label, r := range unicode16Letters {
		require.False(t, isPinnedFormat(r), "%s must not be a format character under the pinned tables", label)
		require.NoError(t, ValidatePfp("https://example.com/a"+string(r)+"b.png"),
			"%s in a pfp path must be accepted deterministically", label)
	}
}
