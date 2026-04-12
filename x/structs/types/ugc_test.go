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
		{"valid url", "https://example.com/image.png", false},
		{"valid http url", "http://example.com/avatar.jpg", false},
		{"valid ipfs cid", "ipfs://QmW2WQi7j6c7UgJTarActp7tDNikE4B2qXtFCfLPdsgaTQ", false},
		{"max length ascii", strings.Repeat("a", MaxPfpLength), false},
		{"exceeds max length ascii", strings.Repeat("a", MaxPfpLength+1), true},
		{"max length multibyte runes", strings.Repeat("日", MaxPfpLength), false},
		{"exceeds max runes multibyte", strings.Repeat("日", MaxPfpLength+1), true},
		{"under rune limit but over byte limit", strings.Repeat("日", 100), false},

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

		// Dangerous URI schemes
		{"javascript scheme", "javascript:alert(1)", true},
		{"javascript uppercase", "JavaScript:alert(1)", true},
		{"javascript mixed case", "JaVaScRiPt:alert(1)", true},
		{"data scheme", "data:text/html,<script>alert(1)</script>", true},
		{"data image", "data:image/png;base64,abc", true},
		{"vbscript scheme", "vbscript:msgbox", true},

		// Valid edge cases
		{"plain text identifier", "avatar-12345", false},
		{"hash identifier", "abc123def456", false},
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
}
