package vocab

import (
	"strings"
	"testing"
)

// TestNoTermAppearsUnderTwoStatuses is structural: lookup must be
// deterministic, so a free-text term may belong to exactly one canonical
// status.
func TestNoTermAppearsUnderTwoStatuses(t *testing.T) {
	seen := make(map[string]string, 128)
	for status, terms := range StatusSynonyms() {
		for _, term := range terms {
			key := strings.ToLower(strings.TrimSpace(term))
			if key == "" {
				t.Errorf("StatusSynonyms[%q] contains a blank term", status)
				continue
			}
			if owner, dup := seen[key]; dup {
				t.Errorf("term %q appears under both %q and %q", term, owner, status)
				continue
			}
			seen[key] = status
		}
	}
}

func TestEveryCanonicalStatusHasSynonyms(t *testing.T) {
	synonyms := StatusSynonyms()
	for _, status := range Statuses() {
		if len(synonyms[status]) == 0 {
			t.Errorf("canonical status %q has no synonyms", status)
		}
	}
}

func TestLookupResolvesEveryStatusAndSynonym(t *testing.T) {
	for status, terms := range StatusSynonyms() {
		entry, ok := Lookup(status)
		if !ok || entry.Status != status {
			t.Errorf("Lookup(%q) = %+v, %v; want status %q", status, entry, ok, status)
		}
		for _, term := range terms {
			entry, ok := Lookup(term)
			if !ok || entry.Status != status {
				t.Errorf("Lookup(%q) = %+v, %v; want status %q", term, entry, ok, status)
			}
		}
	}
}

func TestLookupRejectsUnknownText(t *testing.T) {
	for _, term := range []string{"", "Infusion Pump", "n/a", "unknown"} {
		if entry, ok := Lookup(term); ok {
			t.Errorf("Lookup(%q) = %+v, want no match", term, entry)
		}
	}
}

func TestConditionMapping(t *testing.T) {
	cases := map[string]string{
		"functional":        "in_active_service",
		"in service":        "in_active_service",
		"faulty":            "broken_but_repairable",
		"awaiting parts":    "broken_but_repairable",
		"standby":           "inactive_in_storage",
		"spare":             "inactive_in_storage",
		"retired for parts": "retired_available_for_parts",
		"disposed":          "retired_disposed",
		"scrapped":          "retired_disposed",
	}
	for term, want := range cases {
		got, ok := Condition(term)
		if !ok || got != want {
			t.Errorf("Condition(%q) = %q, %v; want %q", term, got, ok, want)
		}
	}
}

// TestWorkflowOnlyTermsHaveNoCondition pins the deliberate omissions:
// workflow context is not an operational condition, so the import path
// annotates and omits instead of guessing.
func TestWorkflowOnlyTermsHaveNoCondition(t *testing.T) {
	for _, term := range []string{"out of service", "transferred", "decommissioned", "new"} {
		if code, ok := Condition(term); ok {
			t.Errorf("Condition(%q) = %q, want no condition", term, code)
		}
	}
}

func TestIsStatusText(t *testing.T) {
	for _, term := range []string{"functional", "Faulty", "on hold", "retired for parts", "brand new"} {
		if !IsStatusText(term) {
			t.Errorf("IsStatusText(%q) = false, want true", term)
		}
	}
	for _, term := range []string{"Infusion Pump", "BD", "C451", ""} {
		if IsStatusText(term) {
			t.Errorf("IsStatusText(%q) = true, want false", term)
		}
	}
}

func TestDefaultStatus(t *testing.T) {
	if DefaultStatus != "New / Commissioning" {
		t.Fatalf("DefaultStatus = %q", DefaultStatus)
	}
}
