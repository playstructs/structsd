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
		{"max length", strings.Repeat("a", MaxPfpLength), false},
		{"exceeds max length", strings.Repeat("a", MaxPfpLength+1), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePfp(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNormalizeName(t *testing.T) {
	require.Equal(t, "my guild", NormalizeName("My Guild"))
	require.Equal(t, "my guild", NormalizeName("  My Guild  "))
	require.Equal(t, "test", NormalizeName("TEST"))
}
