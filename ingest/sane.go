package ingest

import (
	"strings"
	"unicode"

	"github.com/ovahol/interchange-engine/vocab"
)

// SaneCatalogName reports whether free text is plausible as a
// brand/manufacturer/model name. It rejects three shapes of dirty source
// data seen in real hospital inventories: date-like cells landing in name
// columns (shifted or mislabeled CSV columns), letter-less cells (serials,
// quantities, bare numbers), and condition vocabulary ("Functional" as a
// brand). Callers treat a failure as missing — never recommend it, never
// emit it as a name.
func SaneCatalogName(raw string) bool {
	v := strings.TrimSpace(raw)
	if v == "" {
		return false
	}
	hasLetter := false
	for _, r := range v {
		if unicode.IsLetter(r) {
			hasLetter = true
			break
		}
	}
	if !hasLetter {
		return false
	}
	if TryParseLooseDate(v) {
		return false
	}
	if _, ok := vocab.Condition(v); ok {
		return false
	}
	return true
}

// MisplacedNameValue reports whether free text is a date or a condition word
// sitting where a name should be — a shifted-column signature ("Functional"
// as a model, a date as a brand). Unlike SaneCatalogName it allows
// letter-less text: model codes are legitimately numeric ("220", "C451") and
// stay untouched.
func MisplacedNameValue(raw string) bool {
	v := strings.TrimSpace(raw)
	if v == "" {
		return false
	}
	if TryParseLooseDate(v) {
		return true
	}
	_, ok := vocab.Condition(v)
	return ok
}
