package types

import (
	"strings"
	"unicode"
)

//go:generate go run ./internal/maketables -out unicode_tables.go

// PinnedUnicodeVersion is the Unicode version that governs every
// consensus-visible character decision in this package.
//
// Standard-library Unicode tables vary by Go release, so consensus code uses
// the generated tables in unicode_tables.go instead. NFC remains pinned through
// golang.org/x/text in go.sum. Changing either version changes accepted names
// and requires a chain upgrade.
const PinnedUnicodeVersion = "15.0.0"

// pinnedCaseRange is one range of the simple-lowercase mapping. It mirrors
// unicode.CaseRange reduced to the single case we need, because
// unicode.CaseRange's Delta field has an unexported type.
type pinnedCaseRange struct {
	Lo    uint32
	Hi    uint32
	Delta rune
}

// upperLowerSentinel marks a range that is an alternating sequence of upper
// and lower case runes rather than a fixed delta, matching the meaning of
// unicode.UpperLower. It is a constant expression, not table data.
const upperLowerSentinel = unicode.MaxRune + 1

// isPinnedLetter reports whether r is a letter under the pinned tables. It is
// the replacement for \p{L} in a regexp, which cannot be pointed at a custom
// table and so forced the name charset checks to become explicit rune loops.
func isPinnedLetter(r rune) bool {
	return unicode.Is(pinnedL, r)
}

// isPinnedCombiningMark reports whether r is a non-spacing or enclosing mark
// under the pinned tables: the runes used for Zalgo and stacked-diacritic
// abuse.
func isPinnedCombiningMark(r rune) bool {
	return unicode.Is(pinnedMn, r) || unicode.Is(pinnedMe, r)
}

// isPinnedFormat reports whether r is a format character under the pinned
// tables.
func isPinnedFormat(r rune) bool {
	return unicode.Is(pinnedCf, r)
}

// isPinnedSpace mirrors unicode.IsSpace against the pinned White_Space table.
// The Latin-1 runes are spelled out for the same reason the standard library
// spells them out: the property does not match category Z there.
func isPinnedSpace(r rune) bool {
	if uint32(r) <= unicode.MaxLatin1 {
		switch r {
		case '\t', '\n', '\v', '\f', '\r', ' ', 0x85, 0xA0:
			return true
		}
		return false
	}
	return unicode.Is(pinnedWhiteSpace, r)
}

// pinnedToLower is unicode.ToLower against the pinned case ranges. The
// structure follows unicode.ToLower, unicode.lookupCaseRange and
// unicode.convertCase exactly, including the alternating-sequence handling,
// so that the two agree for every rune while the pinned version is current.
func pinnedToLower(r rune) rune {
	if r <= unicode.MaxASCII {
		if 'A' <= r && r <= 'Z' {
			r += 'a' - 'A'
		}
		return r
	}

	lo, hi := 0, len(pinnedLowerCaseRanges)
	for lo < hi {
		m := int(uint(lo+hi) >> 1)
		cr := &pinnedLowerCaseRanges[m]
		switch {
		case r < rune(cr.Lo):
			hi = m
		case r > rune(cr.Hi):
			lo = m + 1
		case cr.Delta == upperLowerSentinel:
			// An alternating upper/lower sequence starting on an upper case
			// rune. Setting the low bit of the offset selects the lower case
			// member of each pair.
			return rune(cr.Lo) + ((r-rune(cr.Lo))&^1 | 1)
		default:
			return r + cr.Delta
		}
	}
	return r
}

// pinnedToLowerString is strings.ToLower against the pinned case ranges.
// strings.Map is itself table-free, so composing pinnedToLower over it
// reproduces strings.ToLower — including its replacement of invalid UTF-8
// with U+FFFD, which the guild name index depends on for stable keys.
func pinnedToLowerString(s string) string {
	return strings.Map(pinnedToLower, s)
}

// pinnedTrimSpace is strings.TrimSpace against the pinned White_Space table.
// strings.TrimFunc does no classification of its own.
func pinnedTrimSpace(s string) string {
	return strings.TrimFunc(s, isPinnedSpace)
}

// asciiToLower lowercases only A-Z and leaves every other byte alone.
//
// Used for URI schemes, where the alternative would be strings.ToLower or
// strings.EqualFold reaching for the toolchain's case tables. Every allowed
// scheme is ASCII, so a scheme containing a non-ASCII rune stays non-ASCII
// here and fails the allow-list, which is both stricter and deterministic.
// strings.EqualFold would instead have folded, for instance, U+017F to "s".
func asciiToLower(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		b.WriteByte(c)
	}
	return b.String()
}
