package text

import "testing"

func TestNormKey(t *testing.T) {
	cases := map[string]string{
		"Equipment Name(common)": "equipment name",
		"  Model_No.  ":          "model no.",
		"Dräger":                 "drager",
		"Łóżko":                  "lozko",
		"Ștefan":                 "stefan",
		"Straße":                 "strasse",
		"Æther":                  "aether",
		"Œuvre":                  "oeuvre",
		"Þór":                    "thor",
	}
	for in, want := range cases {
		if got := NormKey(in); got != want {
			t.Errorf("NormKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFoldRuneExpandsLigatures(t *testing.T) {
	cases := map[rune]string{
		'ß': "ss",
		'æ': "ae",
		'œ': "oe",
		'þ': "th",
		'A': "a",
		'5': "5",
	}
	for in, want := range cases {
		if got := FoldRune(in); got != want {
			t.Errorf("FoldRune(%q) = %q, want %q", in, got, want)
		}
	}
}
