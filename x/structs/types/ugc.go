package types

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// Name validation regexes operate AFTER NFC normalization. \p{L} matches
// any Unicode letter (any script). Combining marks (\p{M}) are explicitly
// excluded by these character classes -- they're rejected by the
// containsCombiningMark check below to prevent Zalgo/stacking-mark abuse
// even though some are technically letters in their script.
var playerNameRegex = regexp.MustCompile(`^[\p{L}0-9\-_]{3,20}$`)
var entityNameRegex = regexp.MustCompile(`^[\p{L}0-9\-_' ]{3,20}$`)
var planetNameRegex = regexp.MustCompile(`^[\p{L}0-9\-_' ]{3,25}$`)
var objectIdRegex = regexp.MustCompile(`^[0-9]+-[0-9]+$`)
var doubleSpaceRegex = regexp.MustCompile(`  `)

// opaquePfpRegex matches a non-URL PFP identifier (hash, CID, asset id, etc).
// Used only when the value contains no ':' so we can be certain we're not
// staring at a URI scheme.
var opaquePfpRegex = regexp.MustCompile(`^[a-zA-Z0-9._/\-]{1,256}$`)

const MaxPfpLength = 256

// allowedPfpSchemes is the strict allow-list of URI schemes accepted for
// profile pictures. Any value containing a ':' that does not start with one
// of these schemes is rejected. This avoids open-ended scheme handling
// (file:, javascript:, data:, vbscript:, ftp:, etc.) while leaving room for
// the most common decentralized-storage and HTTPS workflows.
var allowedPfpSchemes = map[string]struct{}{
	"https": {},
	"http":  {},
	"ipfs":  {},
	"ipns":  {},
	"ar":    {},
}

// containsCombiningMark returns true if any rune is in Unicode category Mn
// (non-spacing mark) or Me (enclosing mark). These are the runes used in
// Zalgo / stacked diacritic abuse and are rejected to keep names visually
// stable.
func containsCombiningMark(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) {
			return true
		}
	}
	return false
}

// containsBidiOrInvisible returns true if the string includes bidi-override
// or other invisible / format runes that could be used to spoof display
// order or hide characters.
func containsBidiOrInvisible(s string) bool {
	for _, r := range s {
		switch r {
		// Bidi overrides
		case 0x202A, 0x202B, 0x202C, 0x202D, 0x202E,
			// Isolates
			0x2066, 0x2067, 0x2068, 0x2069,
			// Zero-width joiners / non-joiners and word joiner
			0x200B, 0x200C, 0x200D, 0x2060,
			// Soft hyphen
			0x00AD,
			// BOM / zero-width no-break space
			0xFEFF:
			return true
		}
		// Reject format and surrogate categories outright.
		if unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Cs, r) {
			return true
		}
	}
	return false
}

// normalizeAndValidateRunes applies NFC normalization and runs the structural
// rune-level checks shared by every name validator. Returns the normalized
// form on success.
func normalizeAndValidateRunes(name string) (string, error) {
	if !utf8.ValidString(name) {
		return "", fmt.Errorf("name contains invalid UTF-8")
	}
	normalized := norm.NFC.String(name)
	if containsCombiningMark(normalized) {
		return "", fmt.Errorf("name contains combining marks (stacked diacritics not allowed)")
	}
	if containsBidiOrInvisible(normalized) {
		return "", fmt.Errorf("name contains bidi-override, zero-width, or other invisible characters")
	}
	return normalized, nil
}

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
	normalized, err := normalizeAndValidateRunes(name)
	if err != nil {
		return err
	}
	if err := validateNameCommon(normalized); err != nil {
		return err
	}
	if !playerNameRegex.MatchString(normalized) {
		return fmt.Errorf("player name must be 3-20 characters of letters, digits, hyphens, or underscores")
	}
	return nil
}

func ValidateEntityName(name string) error {
	normalized, err := normalizeAndValidateRunes(name)
	if err != nil {
		return err
	}
	if err := validateRelaxedName(normalized); err != nil {
		return err
	}
	if !entityNameRegex.MatchString(normalized) {
		return fmt.Errorf("name must be 3-20 characters of letters, digits, hyphens, underscores, apostrophes, or spaces")
	}
	return nil
}

func ValidatePlanetName(name string) error {
	normalized, err := normalizeAndValidateRunes(name)
	if err != nil {
		return err
	}
	if err := validateRelaxedName(normalized); err != nil {
		return err
	}
	if !planetNameRegex.MatchString(normalized) {
		return fmt.Errorf("planet name must be 3-25 characters of letters, digits, hyphens, underscores, apostrophes, or spaces")
	}
	return nil
}

// ValidatePfp accepts:
//   - The empty string (clears the pfp).
//   - A URL whose scheme is in the strict allow-list (https, http, ipfs,
//     ipns, ar) and that parses cleanly with a non-empty authority/path.
//   - An opaque identifier (no ':' anywhere) consisting of [A-Za-z0-9._/-]
//     up to MaxPfpLength runes -- intended for content-addressed hashes or
//     CIDs without a scheme prefix.
//
// Anything else (data:, javascript:, vbscript:, file:, ftp:, gopher:,
// arbitrary control / bracket / backtick characters, or unparseable URLs)
// is rejected.
func ValidatePfp(pfp string) error {
	if pfp == "" {
		return nil
	}
	if utf8.RuneCountInString(pfp) > MaxPfpLength {
		return fmt.Errorf("pfp must be at most %d characters", MaxPfpLength)
	}
	if !utf8.ValidString(pfp) {
		return fmt.Errorf("pfp contains invalid UTF-8")
	}
	for _, r := range pfp {
		if r < 0x20 || r == 0x7F {
			return fmt.Errorf("pfp contains forbidden control character (0x%02X)", r)
		}
	}
	if containsBidiOrInvisible(pfp) {
		return fmt.Errorf("pfp contains bidi-override, zero-width, or other invisible characters")
	}
	if strings.ContainsAny(pfp, "<>`\"\\ ") {
		return fmt.Errorf("pfp must not contain <, >, backtick, quote, backslash, or whitespace characters")
	}

	if !strings.Contains(pfp, ":") {
		// No scheme: must be an opaque identifier matching the strict charset.
		if !opaquePfpRegex.MatchString(pfp) {
			return fmt.Errorf("pfp opaque identifier must be 1-%d characters of letters, digits, dot, slash, hyphen, or underscore", MaxPfpLength)
		}
		return nil
	}

	// Has a colon -- treat as URL. Extract the lowercase scheme directly
	// (don't depend on url.Parse for this since some malformed inputs accept
	// arbitrary scheme content). url.Parse is then used for structural
	// validation of the rest of the URL.
	colonIdx := strings.Index(pfp, ":")
	scheme := strings.ToLower(pfp[:colonIdx])
	if scheme == "" {
		return fmt.Errorf("pfp URL must have a scheme")
	}
	if _, ok := allowedPfpSchemes[scheme]; !ok {
		return fmt.Errorf("pfp URL scheme %q is not allowed (permitted: https, http, ipfs, ipns, ar)", scheme)
	}

	u, err := url.Parse(pfp)
	if err != nil {
		return fmt.Errorf("pfp URL is malformed: %w", err)
	}
	if !strings.EqualFold(u.Scheme, scheme) {
		return fmt.Errorf("pfp URL scheme inconsistent after parsing")
	}

	switch scheme {
	case "https", "http":
		if u.Host == "" {
			return fmt.Errorf("pfp %s URL must include a host", scheme)
		}
	case "ipfs", "ipns", "ar":
		// These schemes encode the resource id either in Host (ipfs://CID)
		// or directly in Opaque (ipfs:CID). Either is fine, but at least
		// one must be non-empty to identify a resource.
		if u.Host == "" && u.Opaque == "" && u.Path == "" {
			return fmt.Errorf("pfp %s URL must include a content identifier", scheme)
		}
	}

	return nil
}

// NormalizeName produces the canonical comparison form for a name: NFC
// normalized, lowercased, surrounding whitespace trimmed. All uniqueness
// indexes (e.g. the guild name index) MUST key off this form so that
// visually-identical names cannot be re-registered via case or normalization
// tricks.
func NormalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(norm.NFC.String(name)))
}
