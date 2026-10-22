// Package text holds the engine's pure text primitives: unicode folding
// and the normalized key used by headers, names, and vocabulary lookups.
package text

import (
	"strings"
	"unicode"
)

// foldTable maps lowercase Latin letters with marks (and ligatures) to
// their unaccented spelling. Ligatures expand to more than one letter
// (ß→ss, æ→ae, þ→th), which a rune-to-rune fold cannot express.
var foldTable = map[rune]string{
	'ä': "a", 'à': "a", 'á': "a", 'â': "a", 'ã': "a", 'å': "a",
	'ā': "a", 'ă': "a", 'ą': "a", 'ǎ': "a", 'ạ': "a", 'ả': "a",
	'ấ': "a", 'ầ': "a", 'ẩ': "a", 'ẫ': "a", 'ậ': "a",
	'ắ': "a", 'ằ': "a", 'ẳ': "a", 'ẵ': "a", 'ặ': "a",
	'ë': "e", 'è': "e", 'é': "e", 'ê': "e", 'ẽ': "e", 'ē': "e",
	'ĕ': "e", 'ė': "e", 'ę': "e", 'ě': "e", 'ẹ': "e", 'ẻ': "e",
	'ế': "e", 'ề': "e", 'ể': "e", 'ễ': "e", 'ệ': "e",
	'ï': "i", 'ì': "i", 'í': "i", 'î': "i", 'ĩ': "i", 'ī': "i",
	'ĭ': "i", 'į': "i", 'ı': "i", 'ǐ': "i", 'ị': "i", 'ỉ': "i",
	'ö': "o", 'ò': "o", 'ó': "o", 'ô': "o", 'õ': "o", 'ø': "o",
	'ō': "o", 'ŏ': "o", 'ő': "o", 'ǒ': "o", 'ọ': "o", 'ỏ': "o",
	'ố': "o", 'ồ': "o", 'ổ': "o", 'ỗ': "o", 'ộ': "o",
	'ớ': "o", 'ờ': "o", 'ở': "o", 'ỡ': "o", 'ợ': "o",
	'ü': "u", 'ù': "u", 'ú': "u", 'û': "u", 'ũ': "u", 'ū': "u",
	'ŭ': "u", 'ů': "u", 'ű': "u", 'ų': "u", 'ǔ': "u", 'ụ': "u",
	'ủ': "u", 'ư': "u", 'ứ': "u", 'ừ': "u", 'ử': "u", 'ữ': "u", 'ự': "u",
	'ç': "c", 'ć': "c", 'ĉ': "c", 'ċ': "c", 'č': "c",
	'ñ': "n", 'ń': "n", 'ņ': "n", 'ň': "n", 'ŋ': "n",
	'ÿ': "y", 'ý': "y", 'ỳ': "y", 'ỵ': "y", 'ỷ': "y", 'ỹ': "y",
	'š': "s", 'ș': "s", 'ş': "s", 'ś': "s", 'ṣ': "s", 'ſ': "s",
	'ž': "z", 'ż': "z", 'ź': "z", 'ẓ': "z",
	'ł': "l", 'ĺ': "l", 'ļ': "l", 'ľ': "l",
	'đ': "d", 'ð': "d", 'ḍ': "d",
	'ř': "r", 'ŕ': "r", 'ŗ': "r",
	'ț': "t", 'ţ': "t", 'ť': "t", 'ṭ': "t", 'ŧ': "t",
	'ğ': "g", 'ģ': "g", 'ġ': "g",
	'ħ': "h", 'ḥ': "h",
	'ķ': "k",
	// Ligatures expand rather than lose the second letter.
	'ß': "ss", 'æ': "ae", 'œ': "oe", 'þ': "th", 'ĳ': "ij",
	'ﬀ': "ff", 'ﬁ': "fi", 'ﬂ': "fl", 'ﬃ': "ffi", 'ﬄ': "ffl",
	'ﬅ': "st", 'ﬆ': "st",
}

// FoldRune returns r lowercased and unaccented. A ligature may expand to
// more than one rune.
func FoldRune(r rune) string {
	r = unicode.ToLower(r)
	if r <= unicode.MaxASCII {
		return string(r)
	}
	if s, ok := foldTable[r]; ok {
		return s
	}
	return string(r)
}

// WriteFolded appends r lowercased and unaccented to b.
func WriteFolded(b *strings.Builder, r rune) {
	b.WriteString(FoldRune(r))
}

// NormKey normalizes a name or header for comparison: lowercase, trimmed,
// separators folded to spaces, parenthetical qualifiers dropped, and
// diacritics folded ("Dräger" and "Drager" meet).
func NormKey(k string) string {
	var b strings.Builder
	depth := 0
	for _, r := range strings.ToLower(strings.TrimSpace(k)) {
		if r == '(' || r == '[' {
			depth++
			continue
		}
		if r == ')' || r == ']' {
			if depth > 0 {
				depth--
			}
			continue
		}
		if depth > 0 {
			continue
		}
		WriteFolded(&b, r)
	}
	k = b.String()
	k = strings.ReplaceAll(k, "_", " ")
	k = strings.ReplaceAll(k, "-", " ")
	k = strings.ReplaceAll(k, "–", " ")
	k = strings.ReplaceAll(k, "—", " ")
	return strings.Join(strings.Fields(k), " ")
}
