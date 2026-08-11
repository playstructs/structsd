package types

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/unicode/norm"
)

// maxReportedMismatches bounds the failure output. A broken table disagrees
// with the toolchain for hundreds of thousands of runes and the first handful
// are enough to identify which one.
const maxReportedMismatches = 10

// TestPinnedUnicodeVersionIsStable pins the version constant itself. Moving it
// changes which names the chain accepts, so it should never move as a side
// effect of an unrelated edit; a deliberate move updates this test alongside
// an upgrade handler.
func TestPinnedUnicodeVersionIsStable(t *testing.T) {
	require.Equal(t, "15.0.0", PinnedUnicodeVersion,
		"the pinned Unicode version is consensus state; moving it needs an upgrade handler")
}

// TestPinnedTablesChecksum fails if the generated tables change at all.
//
// The generator is only allowed to run on a toolchain at the pinned version,
// but that is a guard on the generator rather than on the file, and the file is
// what ships. This is the guard on the file: any regeneration, hand edit or
// merge artifact moves the digest, so the tables cannot change without someone
// updating a constant in a test that says why they must not.
func TestPinnedTablesChecksum(t *testing.T) {
	const want = "f95e95db3facfc6809d10d3088abbd54ed56babea41e48949d9d21cae9155205"

	// Encoded by hand rather than with binary.Write so there is no error to
	// ignore. Every field is big-endian and fixed width, so the digest covers
	// every bound and stride rather than a summary of them.
	var buf []byte
	be16 := func(v uint16) { buf = append(buf, byte(v>>8), byte(v)) }
	be32 := func(v uint32) { buf = append(buf, byte(v>>24), byte(v>>16), byte(v>>8), byte(v)) }

	for _, tbl := range []struct {
		name  string
		table *unicode.RangeTable
	}{
		{"L", pinnedL},
		{"Mn", pinnedMn},
		{"Me", pinnedMe},
		{"Cf", pinnedCf},
		{"WhiteSpace", pinnedWhiteSpace},
	} {
		buf = append(buf, tbl.name...)
		for _, r := range tbl.table.R16 {
			be16(r.Lo)
			be16(r.Hi)
			be16(r.Stride)
		}
		for _, r := range tbl.table.R32 {
			be32(r.Lo)
			be32(r.Hi)
			be32(r.Stride)
		}
		be32(0)
		be32(uint32(tbl.table.LatinOffset))
	}

	buf = append(buf, "lowerCaseRanges"...)
	for _, cr := range pinnedLowerCaseRanges {
		be32(cr.Lo)
		be32(cr.Hi)
		be32(uint32(cr.Delta))
	}

	sum := sha256.Sum256(buf)
	require.Equal(t, want, hex.EncodeToString(sum[:]),
		"the pinned Unicode tables changed. If that was deliberate, it is a "+
			"consensus-breaking change to the accepted character set and needs an "+
			"upgrade handler plus a new digest here.")
}

// TestPinnedTablesMatchToolchain is the proof that the checked-in tables are a
// faithful copy of Unicode 15.0.0, and it is exhaustive on purpose.
//
// Sampling would not do. unicode.isExcludingLatin trusts RangeTable.LatinOffset
// to skip the Latin-1 prefix of R16, so a table with a wrong offset answers
// correctly for ASCII and incorrectly for everything above U+00FF. And
// pinnedToLower has to reproduce the UpperLower sentinel, which only appears in
// alternating upper/lower sequences in a few blocks. Both classes of bug hide
// from spot checks and are caught here.
//
// On a toolchain past the pinned version this logs the drift instead of
// failing. That drift is the whole reason the tables exist: the pinned answer
// is the correct one and the toolchain's is the one that would have split
// consensus. TestPinnedLetterAcrossScripts and TestPinnedTablesChecksum are
// what still guard the tables once this check goes quiet.
func TestPinnedTablesMatchToolchain(t *testing.T) {
	if unicode.Version != PinnedUnicodeVersion {
		t.Logf("toolchain ships Unicode %s but the tables are pinned to %s; "+
			"skipping the exhaustive comparison. This is expected and is the "+
			"divergence the pinned tables exist to absorb.",
			unicode.Version, PinnedUnicodeVersion)
		return
	}

	checks := []struct {
		name   string
		pinned func(rune) bool
		stdlib func(rune) bool
	}{
		{"letter", isPinnedLetter, func(r rune) bool { return unicode.Is(unicode.L, r) }},
		{"combiningMark", isPinnedCombiningMark, func(r rune) bool {
			return unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r)
		}},
		{"format", isPinnedFormat, func(r rune) bool { return unicode.Is(unicode.Cf, r) }},
		{"space", isPinnedSpace, unicode.IsSpace},
	}

	for _, c := range checks {
		var mismatches []string
		for r := rune(0); r <= unicode.MaxRune; r++ {
			if c.pinned(r) != c.stdlib(r) {
				if len(mismatches) < maxReportedMismatches {
					mismatches = append(mismatches, fmt.Sprintf("U+%04X pinned=%v toolchain=%v",
						r, c.pinned(r), c.stdlib(r)))
				}
			}
		}
		require.Empty(t, mismatches, "pinned %s table disagrees with Unicode %s: %s",
			c.name, unicode.Version, strings.Join(mismatches, ", "))
	}

	var caseMismatches []string
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if got, want := pinnedToLower(r), unicode.ToLower(r); got != want {
			if len(caseMismatches) < maxReportedMismatches {
				caseMismatches = append(caseMismatches,
					fmt.Sprintf("U+%04X pinned=U+%04X toolchain=U+%04X", r, got, want))
			}
		}
	}
	require.Empty(t, caseMismatches, "pinnedToLower disagrees with Unicode %s: %s",
		unicode.Version, strings.Join(caseMismatches, ", "))
}

// TestPinnedStringHelpersMatchToolchain covers the string-level wrappers.
//
// pinnedToLower and isPinnedSpace are proven rune by rune above, but
// strings.ToLower and strings.TrimSpace each carry an ASCII fast path that is
// not literally strings.Map or strings.TrimFunc, and NormalizeName's key bytes
// depend on those wrappers agreeing rather than on the predicates alone. One
// single-rune string per code point exercises the fast paths from both sides.
func TestPinnedStringHelpersMatchToolchain(t *testing.T) {
	if unicode.Version != PinnedUnicodeVersion {
		t.Skipf("toolchain ships Unicode %s, tables pinned to %s", unicode.Version, PinnedUnicodeVersion)
	}

	var lowerMismatches, trimMismatches []string
	for r := rune(0); r <= unicode.MaxRune; r++ {
		s := string(r)
		if got, want := pinnedToLowerString(s), strings.ToLower(s); got != want {
			if len(lowerMismatches) < maxReportedMismatches {
				lowerMismatches = append(lowerMismatches, fmt.Sprintf("U+%04X", r))
			}
		}
		// Pad so the trim has something to remove on both sides and so an
		// interior occurrence is left alone.
		padded := s + "a" + s
		if got, want := pinnedTrimSpace(padded), strings.TrimSpace(padded); got != want {
			if len(trimMismatches) < maxReportedMismatches {
				trimMismatches = append(trimMismatches, fmt.Sprintf("U+%04X", r))
			}
		}
	}
	require.Empty(t, lowerMismatches, "pinnedToLowerString disagrees with strings.ToLower at %s",
		strings.Join(lowerMismatches, ", "))
	require.Empty(t, trimMismatches, "pinnedTrimSpace disagrees with strings.TrimSpace at %s",
		strings.Join(trimMismatches, ", "))
}

// TestNormalizeNameMatchesLegacyForm is the reason MigrateGuildNameIndex is
// expected to be a no-op: it shows the new NormalizeName produces the same
// bytes the old one did, so rebuilding the guild name index writes back the
// keys that are already there.
func TestNormalizeNameMatchesLegacyForm(t *testing.T) {
	if unicode.Version != PinnedUnicodeVersion {
		t.Skipf("toolchain ships Unicode %s, tables pinned to %s", unicode.Version, PinnedUnicodeVersion)
	}

	legacy := func(name string) string {
		return strings.ToLower(strings.TrimSpace(norm.NFC.String(name)))
	}

	corpus := []string{
		"",
		" ",
		"Alpha Guild",
		"  Alpha Guild  ",
		"ALPHA",
		"Ñoño",
		"ÑOÑO",
		"cafe\u0301",
		"café",
		"İstanbul",  // dotted capital I, a full-case-mapping edge
		"ǅ",         // U+01C5, in an UpperLower alternating sequence
		"Ǆ",         // U+01C4, same sequence
		"ΣΊΣΥΦΟΣ",   // final sigma territory
		"ΑΣ",        //
		"ЖУРНАЛ",    // Cyrillic
		"ｆｕｌｌｗｉｄｔｈ", // fullwidth Latin
		"\u00A0pad\u00A0",
		"\u3000pad\u3000",
		"\u2028line\u2029",
		"\ttabbed\n",
		"K",        // U+212A Kelvin sign, which NFC folds to ASCII K
		"ſharp",    // U+017F long s, which EqualFold would have folded to "s"
		"ǰ",        // multi-codepoint uppercase, single lowercase
		"\xff\xfe", // invalid UTF-8, which both forms replace with U+FFFD
		"a\xffb",
	}
	for _, name := range corpus {
		require.Equal(t, legacy(name), NormalizeName(name),
			"NormalizeName changed the key bytes for %q; the guild name index would be re-keyed", name)
	}
}

// TestPinnedLetterAcrossScripts holds on any toolchain, which is what makes it
// useful. Once the exhaustive comparison above goes quiet on a newer Go, a
// truncated or mangled table would otherwise only be caught by the checksum,
// and a checksum cannot say whether the new bytes are sane.
func TestPinnedLetterAcrossScripts(t *testing.T) {
	letters := map[string]rune{
		"latin":      'a',
		"latinUpper": 'Z',
		"greek":      'π',
		"cyrillic":   'ж',
		"armenian":   'ա',
		"hebrew":     'א',
		"arabic":     'ب',
		"devanagari": 'क',
		"thai":       'ก',
		"georgian":   'ა',
		"hangul":     '한',
		"hiragana":   'あ',
		"katakana":   'ア',
		"cjk":        '日',
		"cjkExtB":    '\U00020000',
		"ethiopic":   'ሀ',
		"cherokee":   'Ꭰ',
		"tamil":      'அ',
	}
	for name, r := range letters {
		require.True(t, isPinnedLetter(r), "%s U+%04X must classify as a letter", name, r)
	}

	notLetters := map[string]rune{
		"digit":         '5',
		"space":         ' ',
		"hyphen":        '-',
		"underscore":    '_',
		"apostrophe":    '\'',
		"combiningMark": '\u0301',
		"softHyphen":    '\u00AD',
		"zwj":           '\u200D',
		"emoji":         '\U0001F600',
		"cjkPunct":      '、',
	}
	for name, r := range notLetters {
		require.False(t, isPinnedLetter(r), "%s U+%04X must not classify as a letter", name, r)
	}

	// Spot values that would move if the case table were regenerated against a
	// different Unicode version or emitted with a broken sentinel.
	require.Equal(t, 'a', pinnedToLower('A'))
	require.Equal(t, 'ä', pinnedToLower('Ä'))
	require.Equal(t, 'π', pinnedToLower('Π'))
	require.Equal(t, 'ж', pinnedToLower('Ж'))
	require.Equal(t, '\u01C6', pinnedToLower('\u01C4'), "UpperLower sequence start must map to its lower member")
	require.Equal(t, '\u01C6', pinnedToLower('\u01C5'), "UpperLower sequence title case must map to its lower member")
	require.Equal(t, '日', pinnedToLower('日'), "an uncased rune must be returned unchanged")

	require.True(t, isPinnedSpace('\u00A0'))
	require.True(t, isPinnedSpace('\u3000'))
	require.False(t, isPinnedSpace('a'))

	require.True(t, isPinnedFormat('\u200D'), "zero-width joiner is a format character")
	require.True(t, isPinnedCombiningMark('\u0301'), "combining acute is a non-spacing mark")
	require.True(t, isPinnedCombiningMark('\u20DD'), "combining enclosing circle is an enclosing mark")
}

// TestNFCGoldenVectors guards the other half of the determinism story.
//
// norm.NFC is deterministic across toolchains because golang.org/x/text is
// pinned by go.sum, not because anything checks it. A routine `go get -u`
// would bump its Unicode data, change the composition of some code point, and
// silently change both which names validate and what the guild name index is
// keyed on. These vectors turn that into a failing test.
func TestNFCGoldenVectors(t *testing.T) {
	vectors := []struct {
		name string
		in   string
		want string
	}{
		{"already composed", "café", "café"},
		{"decomposed acute composes", "cafe\u0301", "café"},
		{"decomposed umlaut composes", "u\u0308ber", "über"},
		{"angstrom sign normalizes to a-ring", "\u212B", "\u00C5"},
		{"ohm sign normalizes to omega", "\u2126", "\u03A9"},
		{"hangul jamo composes", "\u1100\u1161", "\uAC00"},
		{"multiple marks reorder canonically", "q\u0307\u0323", "q\u0323\u0307"},
		{"singleton composition of dotted I stays", "\u0130", "\u0130"},
		{"kelvin sign canonically decomposes to ascii K", "\u212A", "K"},
		{"compatibility form is left alone by NFC", "\uFB01", "\uFB01"},
		{"fullwidth is left alone by NFC", "\uFF41", "\uFF41"},
		{"ascii is untouched", "Alpha-Guild_1", "Alpha-Guild_1"},
	}
	for _, v := range vectors {
		t.Run(v.name, func(t *testing.T) {
			require.Equal(t, v.want, norm.NFC.String(v.in),
				"NFC output moved, which means golang.org/x/text changed Unicode version. "+
					"That re-keys the guild name index and changes which names validate.")
		})
	}
}
