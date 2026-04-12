package types

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

var playerNameRegex = regexp.MustCompile(`^[\p{L}0-9\-_]{3,20}$`)
var entityNameRegex = regexp.MustCompile(`^[\p{L}0-9\-_' ]{3,20}$`)
var planetNameRegex = regexp.MustCompile(`^[\p{L}0-9\-_' ]{3,25}$`)
var objectIdRegex = regexp.MustCompile(`^[0-9]+-[0-9]+$`)
var doubleSpaceRegex = regexp.MustCompile(`  `)

const MaxPfpLength = 256

func validateNameCommon(name string) error {
	if objectIdRegex.MatchString(name) {
		return fmt.Errorf("name cannot resemble an object ID")
	}
	return nil
}

func validateRelaxedName(name string) error {
	if err := validateNameCommon(name); err != nil {
		return err
	}
	if strings.HasPrefix(name, " ") || strings.HasSuffix(name, " ") {
		return fmt.Errorf("name cannot have leading or trailing spaces")
	}
	if doubleSpaceRegex.MatchString(name) {
		return fmt.Errorf("name cannot contain consecutive spaces")
	}
	return nil
}

func ValidatePlayerName(name string) error {
	if err := validateNameCommon(name); err != nil {
		return err
	}
	if !playerNameRegex.MatchString(name) {
		return fmt.Errorf("player name must be 3-20 characters of letters, digits, hyphens, or underscores")
	}
	return nil
}

func ValidateEntityName(name string) error {
	if err := validateRelaxedName(name); err != nil {
		return err
	}
	if !entityNameRegex.MatchString(name) {
		return fmt.Errorf("name must be 3-20 characters of letters, digits, hyphens, underscores, apostrophes, or spaces")
	}
	return nil
}

func ValidatePlanetName(name string) error {
	if err := validateRelaxedName(name); err != nil {
		return err
	}
	if !planetNameRegex.MatchString(name) {
		return fmt.Errorf("planet name must be 3-25 characters of letters, digits, hyphens, underscores, apostrophes, or spaces")
	}
	return nil
}

func ValidatePfp(pfp string) error {
	if pfp == "" {
		return nil
	}
	if utf8.RuneCountInString(pfp) > MaxPfpLength {
		return fmt.Errorf("pfp must be at most %d characters", MaxPfpLength)
	}
	for _, r := range pfp {
		if r < 0x20 || r == 0x7F {
			return fmt.Errorf("pfp contains forbidden control character (0x%02X)", r)
		}
	}
	if strings.ContainsAny(pfp, "<>`") {
		return fmt.Errorf("pfp must not contain <, >, or backtick characters")
	}
	lower := strings.ToLower(strings.TrimSpace(pfp))
	for _, scheme := range []string{"javascript:", "data:", "vbscript:"} {
		if strings.HasPrefix(lower, scheme) {
			return fmt.Errorf("pfp must not use %s URI scheme", scheme)
		}
	}
	return nil
}

func NormalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
