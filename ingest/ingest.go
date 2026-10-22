// Package ingest normalizes foreign inventory input before it reaches the
// matcher or a bundle: canonical fields, header synonyms, per-file column
// mapping, cell cleanup, and CSV parsing.
package ingest

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ovahol/interchange-engine/text"
)

// Canonical fields a foreign row can map onto.
const (
	FieldDeviceName       = "device_name"
	FieldModelName        = "model_name"
	FieldBrandName        = "brand_name"
	FieldManufacturerName = "manufacturer_name"
	FieldSerialNumber     = "serial_number"
	FieldTagNumber        = "device_tag_number"
	FieldUDI              = "udi"
	FieldLocationName     = "location_name"
	FieldDepartmentName   = "department_name"
	FieldCondition        = "condition"
	FieldInstallDate      = "installation_date"
	FieldNotes            = "notes"
)

// headerAliases maps normalized source headers to canonical fields. It is
// the zero-config path; explicit per-file mapping is the escape hatch.
var headerAliases = map[string]string{
	"device name": FieldDeviceName, "device": FieldDeviceName, "equipment": FieldDeviceName,
	"equipment name": FieldDeviceName, "equipment common name": FieldDeviceName,
	"common name": FieldDeviceName, "asset name": FieldDeviceName, "device type": FieldDeviceName,
	"nomenclature": FieldDeviceName, "generic name": FieldDeviceName, "devicename": FieldDeviceName,
	// UMDNS nomenclature as device-name fallback: it only wins when
	// no equipment/device column claimed the field (first-wins race,
	// deterministic order), so rows with an empty equipment name but
	// a UMDNS name still get a real definition instead of the model
	// fallback. The losing column stays preserved in SourceData.
	"umdns name": FieldDeviceName, "umdns": FieldDeviceName,
	"model": FieldModelName, "model name": FieldModelName, "model number": FieldModelName,
	"model no": FieldModelName, "model no.": FieldModelName, "modelname": FieldModelName,
	"brand": FieldBrandName, "brand name": FieldBrandName, "make": FieldBrandName,
	"brandname":    FieldBrandName,
	"manufacturer": FieldManufacturerName, "manufacturer name": FieldManufacturerName,
	"manufacturername": FieldManufacturerName,
	"vendor":           FieldManufacturerName, "supplier": FieldManufacturerName, "mfr": FieldManufacturerName,
	"mfr.":   FieldManufacturerName,
	"serial": FieldSerialNumber, "serial number": FieldSerialNumber, "serial no": FieldSerialNumber,
	"serial no.": FieldSerialNumber, "s/n": FieldSerialNumber, "sn": FieldSerialNumber,
	"serialnumber": FieldSerialNumber, "serialno": FieldSerialNumber,
	"tag": FieldTagNumber, "tag number": FieldTagNumber, "tag id": FieldTagNumber, "tag no": FieldTagNumber,
	"asset tag": FieldTagNumber,
	"asset id":  FieldTagNumber, "asset no": FieldTagNumber, "asset number": FieldTagNumber,
	"assetid":    FieldTagNumber,
	"equipment#": FieldTagNumber, "equipment #": FieldTagNumber, "equipment no": FieldTagNumber,
	"equipment number": FieldTagNumber,
	"equipmentid":      FieldTagNumber,
	"inventory no":     FieldTagNumber, "inventory number": FieldTagNumber, "inventory no.": FieldTagNumber,
	"equipment id": FieldTagNumber,
	"udi":          FieldUDI, "udi-di": FieldUDI, "udi di": FieldUDI, "device identifier": FieldUDI,
	"location": FieldLocationName, "ward": FieldLocationName, "room": FieldLocationName,
	"location/room": FieldLocationName, "location room": FieldLocationName,
	"site": FieldLocationName, "placement": FieldLocationName, "current location": FieldLocationName,
	"department": FieldDepartmentName, "dept": FieldDepartmentName, "dept.": FieldDepartmentName,
	"unit": FieldDepartmentName, "division": FieldDepartmentName,
	"status": FieldCondition, "condition": FieldCondition, "state": FieldCondition,
	"operational status": FieldCondition, "functional status": FieldCondition,
	"availability":      FieldCondition,
	"installation date": FieldInstallDate, "installed on": FieldInstallDate,
	"commissioning date": FieldInstallDate, "install date": FieldInstallDate,
	"date of installation": FieldInstallDate, "date installed": FieldInstallDate,
	"dateinstalled": FieldInstallDate,
	"notes":         FieldNotes, "remarks": FieldNotes, "remark": FieldNotes, "comment": FieldNotes, "comments": FieldNotes,
	"comment/remark": FieldNotes,
	"description":    FieldNotes,
}

// ResolveHeader maps one source header to its canonical field.
func ResolveHeader(header string) (string, bool) {
	canon, ok := headerAliases[text.NormKey(header)]
	return canon, ok
}

// IsCanonicalField reports whether f is one of the canonical fields.
func IsCanonicalField(f string) bool {
	switch f {
	case FieldDeviceName, FieldModelName, FieldBrandName, FieldManufacturerName,
		FieldSerialNumber, FieldTagNumber, FieldUDI, FieldLocationName,
		FieldDepartmentName, FieldCondition, FieldInstallDate, FieldNotes:
		return true
	}
	return false
}

// ApplyColumnMapping translates one foreign row to canonical fields.
// Returns the mapped row, the foreign keys nobody claimed (for the
// unmapped-fields report), and the raw keys claimed by either path.
// Callers must use the claimed set (not a re-derived lookup) when
// deciding what to preserve verbatim: only claimed keys are consumed,
// so explicitly mapped columns never duplicate and losing synonym
// headers are never dropped.
func ApplyColumnMapping(row map[string]any, mapping map[string]string) (mapped map[string]string, unmapped []string, claimed map[string]bool) {
	mapped = map[string]string{}
	lowerKeys := map[string]string{}
	for k := range row {
		lowerKeys[text.NormKey(fmt.Sprint(k))] = fmt.Sprint(k)
	}
	claimed = map[string]bool{}
	// Explicit mapping first.
	for foreign, canon := range mapping {
		canon = strings.ToLower(strings.TrimSpace(canon))
		if !IsCanonicalField(canon) {
			continue
		}
		if orig, ok := lowerKeys[text.NormKey(foreign)]; ok {
			mapped[canon] = StringifyCell(row[orig])
			claimed[orig] = true
		}
	}
	// Auto-detect the rest, iterating headers in sorted order so
	// multi-synonym files resolve deterministically: when two headers
	// claim one canonical field ("Manufacturer" and "Supplier", "Asset
	// ID" and "EquipmentID"), the lexicographically first wins every
	// run, and the loser stays preserved in SourceData via the claimed
	// set — never dropped, never random. Two passes: filled cells
	// first, so an empty column never steals a slot from a filled one
	// (an empty "Asset ID" must not block a filled "EquipmentID");
	// empties claim leftovers in pass two so they stay out of
	// SourceData and the unmapped report, as before.
	keys := make([]string, 0, len(row))
	for k := range row {
		keys = append(keys, fmt.Sprint(k))
	}
	sort.Strings(keys)
	keyOf := map[string]string{}
	for k := range row {
		keyOf[fmt.Sprint(k)] = k
	}
	claim := func(sk string, emptyOK bool) {
		k := keyOf[sk]
		v := row[k]
		if claimed[k] {
			return
		}
		sv := StringifyCell(v)
		if sv == "" && !emptyOK {
			return
		}
		if canon, ok := headerAliases[text.NormKey(sk)]; ok {
			if _, taken := mapped[canon]; !taken {
				mapped[canon] = sv
				claimed[k] = true
			}
		}
	}
	for _, sk := range keys {
		claim(sk, false)
	}
	for _, sk := range keys {
		claim(sk, true)
	}
	for k := range row {
		if !claimed[k] {
			unmapped = append(unmapped, k)
		}
	}
	return mapped, unmapped, claimed
}

// StringifyCell renders one foreign cell as text. JSON null (Go nil)
// becomes "" — never the literal "<nil>" fmt.Sprint would produce,
// which would otherwise poison names, locations, and identity keys.
// A leading quote run is stripped: spreadsheets force-text cells by
// prefixing a quote (e.g. `"00098`), which is markup, not content.
func StringifyCell(v any) string {
	if v == nil {
		return ""
	}
	var s string
	switch t := v.(type) {
	case string:
		s = strings.TrimSpace(t)
	case json.Number:
		s = strings.TrimSpace(t.String())
	default:
		s = strings.TrimSpace(fmt.Sprint(v))
	}
	return strings.TrimLeft(s, "\"'")
}

// FoldUnknownCell folds placeholder texts ("Unknown", "N/A", "-")
// to missing. A literal "Unknown" in the source is not a name, a
// location, or a department — folding keeps it from becoming a
// catalog name or a meaningless "Unknown" location/department.
func FoldUnknownCell(v string) string {
	if strings.EqualFold(v, "unknown") || strings.EqualFold(v, "n/a") || v == "-" {
		return ""
	}
	return v
}

// DetectCSVDelimiter picks the delimiter from the first non-empty line.
func DetectCSVDelimiter(content string) rune {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		best, bestCount := ',', 0
		for _, candidate := range []rune{',', ';', '\t', '|'} {
			if n := strings.Count(line, string(candidate)); n > bestCount {
				best, bestCount = candidate, n
			}
		}
		return best
	}
	return ','
}

// ParseCSVRecords decodes CSV content into raw records (header first).
// Two passes, structural disambiguation: hospital exports contain both
// Excel force-text markers (`"00098` — an opener that never closes) and
// legitimate multi-line quoted fields (comments with embedded newlines).
// The two are locally indistinguishable, so the parse that yields a
// consistent shape wins: a LazyQuotes pass first (handles embedded
// newlines natively), and only when records come out ragged does the
// sanitizer path run (drops unterminated openers, then the strict rules
// below apply).
func ParseCSVRecords(content string, delimiter rune) ([][]string, error) {
	if all, _, ok := readCSV(content, delimiter, false); ok {
		return all, nil
	}
	sanitized := sanitizeCSVQuotes(content, delimiter)
	if all, _, ok := readCSV(sanitized, delimiter, true); ok {
		return all, nil
	}
	// Both passes ragged: report the offending physical line from the
	// raw content so the operator can find the unquoted comma.
	if _, line, _ := readCSV(content, delimiter, false); line > 0 {
		return nil, fmt.Errorf("parse CSV: record on line %d has more fields than the header — check for an unquoted comma in that row (wrap the field in double quotes)", line)
	}
	return nil, fmt.Errorf("parse CSV: ragged rows even after quote sanitizing — check for an unquoted comma")
}

// readCSV parses content with LazyQuotes (tolerating mid-field strays).
// It reports ok only when every record matches the header field count:
// short rows (ragged trailing empties) pad out when padding is allowed;
// long rows never silently shift — a stray comma cannot be safely
// assigned to a column, so the offending line number is reported instead.
func readCSV(content string, delimiter rune, padShort bool) ([][]string, int, bool) {
	r := csv.NewReader(strings.NewReader(content))
	r.TrimLeadingSpace = true
	r.LazyQuotes = true
	// Delimiter auto-detect: comma first, then semicolon (locales
	// using comma decimals), tab (spreadsheet copy-paste), pipe.
	r.Comma = delimiter
	r.FieldsPerRecord = -1
	all, err := r.ReadAll()
	if err != nil || len(all) < 2 {
		return nil, 0, false
	}
	headers := all[0]
	out := make([][]string, 0, len(all)-1)
	for i, rec := range all[1:] {
		if len(rec) > len(headers) {
			return nil, i + 2, false
		}
		if len(rec) < len(headers) {
			if !padShort {
				return nil, 0, false
			}
			rec = append(rec, make([]string, len(headers)-len(rec))...)
		}
		out = append(out, rec)
	}
	return append([][]string{headers}, out...), 0, true
}

// sanitizeCSVQuotes drops field-opening quotes that never close (Excel
// force-text markers like `"00098`, or hand damage). It runs only as the
// fallback when the direct parse comes out ragged — legitimate multi-line
// quoted fields are handled by the first pass, never reach here, and must
// not be "repaired".
func sanitizeCSVQuotes(content string, delimiter rune) string {
	lines := strings.Split(content, "\n")
	for li, line := range lines {
		var out strings.Builder
		inQuotes := false
		fieldStart := true
		runes := []rune(line)
		for i := 0; i < len(runes); i++ {
			r := runes[i]
			if r == '"' && !inQuotes && fieldStart {
				// Look ahead on this line for a valid closer.
				closed := false
				for j := i + 1; j < len(runes); j++ {
					if runes[j] == '"' {
						next := j + 1
						if next >= len(runes) || runes[next] == delimiter || runes[next] == '\r' {
							closed = true
						}
						break
					}
				}
				if !closed {
					fieldStart = false
					continue
				}
				inQuotes = true
				fieldStart = false
				out.WriteRune(r)
				continue
			}
			if r == '"' && inQuotes {
				// Escaped ("") or closing quote.
				if i+1 < len(runes) && runes[i+1] == '"' {
					out.WriteRune(r)
					out.WriteRune(runes[i+1])
					i++
					continue
				}
				inQuotes = false
				out.WriteRune(r)
				continue
			}
			if r == delimiter && !inQuotes {
				fieldStart = true
			} else if r != ' ' && r != '\t' && r != '\r' {
				fieldStart = false
			}
			out.WriteRune(r)
		}
		lines[li] = out.String()
	}
	return strings.Join(lines, "\n")
}

// TryParseLooseDate reports whether v is a date in a common inventory
// export format (M/D/YY, D/M/YYYY, ISO, with optional time), trying both
// slash orders so any calendar date is caught.
func TryParseLooseDate(v string) bool {
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"2006/01/02",
		"01/02/2006 15:04",
		"02/01/2006 15:04",
		"01/02/06 15:04",
		"02/01/06 15:04",
		"01/02/2006",
		"02/01/2006",
		"01/02/06",
		"02/01/06",
	}
	for _, l := range layouts {
		if _, err := time.Parse(l, strings.TrimSpace(v)); err == nil {
			return true
		}
	}
	return false
}
