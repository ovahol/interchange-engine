// Package vocab is the single source of Ovahol's device status vocabulary.
// It projects one synonym table two ways: a canonical status (used by
// imports) and, where the term describes an operational condition, an
// interchange condition code.
//
// The projections are deliberately not identical: a term may resolve to a
// canonical status without describing an operational condition (workflow
// context), and condition-only WHO phrasing may not be a status synonym.
package vocab

import "github.com/ovahol/interchange-engine/text"

// DefaultStatus is the conservative landing status for a freshly imported
// record whose source status is missing or unrecognized.
const DefaultStatus = "New / Commissioning"

var canonicalStatuses = []string{
	"In-Service",
	"Standby / Spare",
	"Under Maintenance",
	"Out of Service",
	"Decommissioned",
	"Disposed",
	"Transferred",
	"New / Commissioning",
}

// statusSynonyms maps each canonical status to the free-text terms a source
// inventory might use for it. Every term appears under exactly one status;
// TestNoTermAppearsUnderTwoStatuses enforces that structurally.
var statusSynonyms = map[string][]string{
	"In-Service": {
		"functional", "functioning", "active", "in active service", "working",
		"in service", "in-service", "inservice", "operational", "operating",
		"in use", "in-use", "currently in use", "deployed", "live", "running",
		"good", "good condition", "ok", "normal",
	},
	"Under Maintenance": {
		"faulty", "not working", "broken down", "broken down and repairable", "maintenance",
		"broken but repairable", "under repair", "in repair", "being repaired",
		"out for repair", "awaiting repair", "awaiting parts", "waiting on parts",
		"down for maintenance", "in for service", "in for maintenance", "being serviced",
		"malfunctioning", "defective", "needs repair", "repair needed", "pm due",
		"preventive maintenance due", "calibration due", "under calibration",
		"being calibrated", "service required", "repair in progress", "servicing",
	},
	"Out of Service": {
		"out-of-service", "out of service", "down", "offline", "non-operational",
		"not operational", "unusable", "inoperable", "disabled", "shut down",
		"shutdown", "taken out of service", "on hold", "hold", "unavailable",
		"not available", "quarantined", "quarantine", "suspended", "blocked",
		"deactivated",
	},
	"Standby / Spare": {
		"standby", "spare", "reserve", "inactive - in storage", "in storage",
		"backup", "back-up", "not in use", "idle", "available", "on standby",
		"held in reserve", "stored", "in warehouse", "in stock", "spare unit",
		"reserve unit", "warehouse", "unassigned", "in inventory",
	},
	"Decommissioned": {
		"retired - available for parts", "retired for parts", "available for parts",
		"decommissioned", "retired", "end of life", "eol", "end-of-life",
		"taken out of commission", "pulled from service", "withdrawn from service",
		"obsolete", "phased out", "removed from service", "condemned",
	},
	"Disposed": {
		"discarded", "scrapped", "sold as scrap", "disposed", "disposed of",
		"destroyed", "junked", "salvaged", "recycled", "written off",
		"condemned and disposed", "scrap", "trashed",
	},
	"Transferred": {
		"relocated", "moved", "transferred out", "sent to another facility",
		"shipped out", "reassigned", "moved to another facility", "relocated out",
	},
	"New / Commissioning": {
		"new", "brand new", "commissioning", "being commissioned",
		"pending commissioning", "not yet commissioned", "awaiting installation",
		"pending installation", "installed - not commissioned", "incoming",
		"received", "newly received", "unboxed", "pending acceptance testing",
		"awaiting acceptance testing", "acceptance testing", "pre-service",
		"not yet in service", "newly acquired", "just delivered",
	},
}

// conditionCodes maps normalized free-text to interchange operational-
// condition codes (the five WHO reference codes from the interchange design
// requirements). Terms whose meaning is pure workflow — Out of Service,
// Transferred, Decommissioned, New / Commissioning — are deliberately
// absent: workflow context is not an operational condition, and the
// no-guess rule says to omit and annotate instead of inventing one.
var conditionCodes = map[string]string{
	// In-Service + WHO "available for intended use".
	"functional": "in_active_service", "functioning": "in_active_service",
	"active": "in_active_service", "in active service": "in_active_service",
	"working": "in_active_service", "in service": "in_active_service",
	"inservice":   "in_active_service",
	"operational": "in_active_service", "operating": "in_active_service",
	"in use":   "in_active_service",
	"deployed": "in_active_service", "live": "in_active_service",
	"running": "in_active_service", "good": "in_active_service",
	"good condition": "in_active_service", "ok": "in_active_service",
	"normal": "in_active_service", "available": "in_active_service",
	"available for intended use":           "in_active_service",
	"device is available for intended use": "in_active_service",

	// Under Maintenance.
	"faulty": "broken_but_repairable", "not working": "broken_but_repairable",
	"not functional": "broken_but_repairable", "non functional": "broken_but_repairable",
	"nonfunctional": "broken_but_repairable",
	"broken":        "broken_but_repairable", "broken but repairable": "broken_but_repairable",
	"broken down":                "broken_but_repairable",
	"broken down and repairable": "broken_but_repairable",
	"under maintenance":          "broken_but_repairable", "maintenance": "broken_but_repairable",
	"under repair": "broken_but_repairable", "under repairs": "broken_but_repairable", "in repair": "broken_but_repairable",
	"being repaired": "broken_but_repairable", "out for repair": "broken_but_repairable",
	"awaiting repair": "broken_but_repairable", "awaiting parts": "broken_but_repairable",
	"waiting on parts":     "broken_but_repairable",
	"down for maintenance": "broken_but_repairable",
	"in for service":       "broken_but_repairable", "in for maintenance": "broken_but_repairable",
	"being serviced": "broken_but_repairable", "malfunctioning": "broken_but_repairable",
	"defective": "broken_but_repairable", "needs repair": "broken_but_repairable",
	"repair needed": "broken_but_repairable", "pm due": "broken_but_repairable",
	"preventive maintenance due": "broken_but_repairable",
	"calibration due":            "broken_but_repairable", "under calibration": "broken_but_repairable",
	"being calibrated": "broken_but_repairable", "service required": "broken_but_repairable",
	"repair in progress": "broken_but_repairable", "servicing": "broken_but_repairable",
	"nonfunctional but considered repairable":           "broken_but_repairable",
	"non functional but considered repairable":          "broken_but_repairable",
	"device is nonfunctional but considered repairable": "broken_but_repairable",

	// Standby / Spare.
	"standby": "inactive_in_storage", "spare": "inactive_in_storage",
	"reserve": "inactive_in_storage", "in storage": "inactive_in_storage",
	"inactive in storage": "inactive_in_storage",
	"backup":              "inactive_in_storage", "back up": "inactive_in_storage",
	"idle":       "inactive_in_storage",
	"on standby": "inactive_in_storage", "held in reserve": "inactive_in_storage",
	"stored": "inactive_in_storage", "in warehouse": "inactive_in_storage",
	"in stock": "inactive_in_storage", "spare unit": "inactive_in_storage",
	"reserve unit": "inactive_in_storage", "warehouse": "inactive_in_storage",
	"unassigned": "inactive_in_storage", "in inventory": "inactive_in_storage",
	"standby / spare":               "inactive_in_storage",
	"inactive and stored":           "inactive_in_storage",
	"device is inactive and stored": "inactive_in_storage",

	// WHO retired-and-retained-for-parts.
	"retired for parts":                        "retired_available_for_parts",
	"retired available for parts":              "retired_available_for_parts",
	"available for parts":                      "retired_available_for_parts",
	"retained for parts":                       "retired_available_for_parts",
	"retired and retained for parts":           "retired_available_for_parts",
	"device is retired and retained for parts": "retired_available_for_parts",

	// Disposed + WHO retired-and-disposed-of.
	"disposed": "retired_disposed", "destroyed": "retired_disposed",
	"discarded": "retired_disposed",
	"scrapped":  "retired_disposed", "recycled": "retired_disposed",
	"condemned":                         "retired_disposed",
	"retired and disposed of":           "retired_disposed",
	"device is retired and disposed of": "retired_disposed",
}

// Entry is one resolved status term.
type Entry struct {
	Status    string
	Condition string // interchange operational-condition code; "" when workflow-only
}

var synonymOwner map[string]string

func init() {
	synonymOwner = make(map[string]string, len(statusSynonyms)*4)
	// Iterate canonicalStatuses, not the map, so duplicate terms resolve
	// deterministically (the uniqueness test should prevent them anyway).
	for _, status := range canonicalStatuses {
		synonymOwner[text.NormKey(status)] = status
		for _, term := range statusSynonyms[status] {
			key := text.NormKey(term)
			if key == "" {
				continue
			}
			if _, exists := synonymOwner[key]; !exists {
				synonymOwner[key] = status
			}
		}
	}
}

// Lookup resolves free text to its canonical status and, when the term
// describes an operational condition, its interchange condition code.
func Lookup(raw string) (Entry, bool) {
	key := text.NormKey(raw)
	if key == "" {
		return Entry{}, false
	}
	status, ok := synonymOwner[key]
	if !ok {
		return Entry{}, false
	}
	return Entry{Status: status, Condition: conditionCodes[key]}, true
}

// Condition resolves free text to an interchange operational-condition
// code. ok=false means the term is workflow-only or unknown; callers must
// omit the condition rather than guess.
func Condition(raw string) (string, bool) {
	code, ok := conditionCodes[text.NormKey(raw)]
	return code, ok
}

// IsStatusText reports whether free text is a status term, which keeps
// status words out of name columns.
func IsStatusText(raw string) bool {
	if _, ok := Lookup(raw); ok {
		return true
	}
	_, ok := Condition(raw)
	return ok
}

// Statuses returns the canonical statuses in display order.
func Statuses() []string {
	out := make([]string, len(canonicalStatuses))
	copy(out, canonicalStatuses)
	return out
}

// StatusSynonyms returns a deep copy of the canonical status → synonyms map,
// suitable for building an external conventions value.
func StatusSynonyms() map[string][]string {
	out := make(map[string][]string, len(canonicalStatuses))
	for _, status := range canonicalStatuses {
		terms := statusSynonyms[status]
		cp := make([]string, len(terms))
		copy(cp, terms)
		out[status] = cp
	}
	return out
}
