package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// Extra characters each name kind allows alongside letters and ASCII digits.
// Combining marks are never allowed: containsCombiningMark rejects them below
// to prevent Zalgo/stacking-mark abuse even though some are technically
// letters in their script.
const (
	playerNameExtra  = "-_"
	relaxedNameExtra = "-_' "
)

// The remaining regexes are pure ASCII character classes with no case-folding
// flag, so they carry no dependency on the toolchain's Unicode tables.
var objectIdRegex = regexp.MustCompile(`^[0-9]+-[0-9]+$`)
var doubleSpaceRegex = regexp.MustCompile(`  `)

// nameCharsetOK reports whether s is between minRunes and maxRunes long and
// every rune is a letter under the pinned tables, an ASCII digit, or one of
// extra.
//
// This replaces three regexes of the form ^[\p{L}0-9\-_]{3,20}$. A regexp
// cannot be pointed at a range table we control, and Go resolves \p{L} against
// the tables of whichever toolchain compiled the binary, so the charset test
// had to become an explicit loop to be deterministic. See unicode_pinned.go.
//
// The bounds are rune counts because that is what a repeat count on a
// single-rune character class meant: the regexes it replaces counted runes,
// not bytes. Callers run this after NFC normalization, as the regexes did.
func nameCharsetOK(s string, minRunes int, maxRunes int, extra string) bool {
	count := 0
	for _, r := range s {
		count++
		if count > maxRunes {
			return false
		}
		if isPinnedLetter(r) || ('0' <= r && r <= '9') || strings.ContainsRune(extra, r) {
			continue
		}
		return false
	}
	return count >= minRunes
}

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
		if isPinnedCombiningMark(r) {
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
		// Reject the format category outright.
		if isPinnedFormat(r) {
			return true
		}
		// Surrogates, kept as a literal range rather than a table because the
		// range is fixed by the encoding rather than by any Unicode version.
		// Both callers check utf8.ValidString first and a surrogate cannot
		// appear in a valid UTF-8 string, so this is unreachable; it stays as
		// a belt in case a future caller skips that check.
		if r >= 0xD800 && r <= 0xDFFF {
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
	if !nameCharsetOK(normalized, 3, 20, playerNameExtra) {
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
	if !nameCharsetOK(normalized, 3, 20, relaxedNameExtra) {
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
	if !nameCharsetOK(normalized, 3, 25, relaxedNameExtra) {
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
	//
	// Lowercased ASCII-only rather than with strings.ToLower, whose case
	// tables come from the compiling toolchain. Every allowed scheme is ASCII,
	// so a scheme carrying a non-ASCII rune stays non-ASCII and fails the
	// allow-list below, which is what we want anyway.
	colonIdx := strings.Index(pfp, ":")
	scheme := asciiToLower(pfp[:colonIdx])
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
	// Compared with ASCII folding rather than strings.EqualFold, which reads
	// the toolchain's case tables and would treat, for instance, U+017F as
	// equal to "s". url.Parse already lowercases the scheme by ASCII rules and
	// scheme is ASCII-lowered above, so this is a plain comparison.
	if asciiToLower(u.Scheme) != scheme {
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

// MaxPfpClientRenderAttributesBytes bounds the stored client render
// attributes blob. The cap is byte-denominated (not rune count) because the
// value is re-marshaled into the Player record and emitted on every player
// write, so the byte size is what actually drives state and event growth.
const MaxPfpClientRenderAttributesBytes = 512

// ValidatePfpClientRenderAttributes validates the client render attributes
// blob attached to a player's locally-rendered profile picture and returns
// the compacted (whitespace-stripped) JSON to store.
//
// The empty string is allowed and clears the value. Otherwise the value must
// be a JSON object within the byte cap. Validation is intentionally loose --
// object-only, with no key/value schema -- so the contents stay flexible as
// the client render model evolves. Storing the compacted form prevents
// padding the cap with whitespace and keeps the per-write/event footprint
// minimal.
func ValidatePfpClientRenderAttributes(attributes string) (string, error) {
	if attributes == "" {
		return "", nil
	}
	if len(attributes) > MaxPfpClientRenderAttributesBytes {
		return "", fmt.Errorf("pfpClientRenderAttributes must be at most %d bytes", MaxPfpClientRenderAttributesBytes)
	}

	// Must decode as a JSON object (rejects arrays, scalars, and malformed
	// JSON). RawMessage values avoid imposing any schema on the contents.
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(attributes), &obj); err != nil {
		return "", fmt.Errorf("pfpClientRenderAttributes must be a valid JSON object")
	}

	var compacted bytes.Buffer
	if err := json.Compact(&compacted, []byte(attributes)); err != nil {
		return "", fmt.Errorf("pfpClientRenderAttributes must be a valid JSON object")
	}
	return compacted.String(), nil
}

// NormalizeName produces the canonical comparison form for a name: NFC
// normalized, lowercased, surrounding whitespace trimmed. All uniqueness
// indexes (e.g. the guild name index) MUST key off this form so that
// visually-identical names cannot be re-registered via case or normalization
// tricks.
//
// The output is a KV key, not just a comparison value: SetGuildNameIndex and
// RemoveGuildNameIndex write and delete "Guild/name/" + this string. That
// makes it the one place in the package where a Unicode table disagreement
// between two binaries would not merely accept different names but write
// different keys, and RemoveGuildNameIndex is called with a name already in
// state rather than one that just passed validation. So the case folding and
// the space trimming both come from the pinned tables. norm.NFC stays, being
// pinned by go.sum rather than by the toolchain.
func NormalizeName(name string) string {
	return pinnedToLowerString(pinnedTrimSpace(norm.NFC.String(name)))
}
