package ingest

import (
	"encoding/json"
	"testing"
)

func TestApplyColumnMappingExplicitAndSynonyms(t *testing.T) {
	row := map[string]any{
		"Equipment Name": "Infusion Pump",
		"Model No.":      "IP-2000",
		"Usr":            "ICU",
		"Junk":           "ignored",
	}
	mapped, unmapped, claimed := ApplyColumnMapping(row, map[string]string{"Usr": FieldDepartmentName})

	if got := mapped[FieldDeviceName]; got != "Infusion Pump" {
		t.Fatalf("device name = %q, want Infusion Pump", got)
	}
	if got := mapped[FieldModelName]; got != "IP-2000" {
		t.Fatalf("model = %q, want IP-2000", got)
	}
	if got := mapped[FieldDepartmentName]; got != "ICU" {
		t.Fatalf("department = %q, want ICU", got)
	}
	if len(unmapped) != 1 || unmapped[0] != "Junk" {
		t.Fatalf("unmapped = %v, want [Junk]", unmapped)
	}
	if !claimed["Usr"] || !claimed["Equipment Name"] {
		t.Fatalf("claimed = %v, want Usr and Equipment Name", claimed)
	}
}

func TestApplyColumnMappingFirstSynonymWinsDeterministically(t *testing.T) {
	row := map[string]any{"Manufacturer": "Alpha", "Supplier": "Beta"}
	mapped, _, _ := ApplyColumnMapping(row, nil)
	if got := mapped[FieldManufacturerName]; got != "Alpha" {
		t.Fatalf("manufacturer = %q, want Alpha (sorted first)", got)
	}
}

func TestApplyColumnMappingEmptyCellDoesNotStealSlot(t *testing.T) {
	row := map[string]any{"Asset ID": "", "EquipmentID": "EQ-1"}
	mapped, _, _ := ApplyColumnMapping(row, nil)
	if got := mapped[FieldTagNumber]; got != "EQ-1" {
		t.Fatalf("tag = %q, want EQ-1", got)
	}
}

func TestDetectCSVDelimiter(t *testing.T) {
	cases := map[string]rune{
		"a,b\n1,2":       ',',
		"a;b\n1;2":       ';',
		"a\tb\n1\t2":     '\t',
		"a|b\n1|2":       '|',
		"\n\nsingle-col": ',',
	}
	for in, want := range cases {
		if got := DetectCSVDelimiter(in); got != want {
			t.Errorf("DetectCSVDelimiter(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseCSVRecords(t *testing.T) {
	records, err := ParseCSVRecords("a,b\n1,2\n3,4\n", ',')
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(records) != 3 || records[0][1] != "b" || records[2][0] != "3" {
		t.Fatalf("unexpected records: %v", records)
	}
}

func TestParseCSVRecordsRejectsRaggedRows(t *testing.T) {
	if _, err := ParseCSVRecords("a,b\n1,2,3\n", ','); err == nil {
		t.Fatal("expected an error for a row longer than the header")
	}
}

func TestStringifyCell(t *testing.T) {
	if got := StringifyCell(nil); got != "" {
		t.Fatalf("nil = %q, want empty", got)
	}
	if got := StringifyCell(json.Number("123")); got != "123" {
		t.Fatalf("json.Number = %q, want 123", got)
	}
	if got := StringifyCell(`"00098`); got != "00098" {
		t.Fatalf("force-text quote = %q, want 00098", got)
	}
	if got := StringifyCell("  Ward A  "); got != "Ward A" {
		t.Fatalf("trim = %q, want Ward A", got)
	}
}

func TestFoldUnknownCell(t *testing.T) {
	for _, v := range []string{"Unknown", "unknown", "N/A", "n/a", "-"} {
		if got := FoldUnknownCell(v); got != "" {
			t.Errorf("FoldUnknownCell(%q) = %q, want empty", v, got)
		}
	}
	if got := FoldUnknownCell("Ward A"); got != "Ward A" {
		t.Fatalf("FoldUnknownCell(Ward A) = %q", got)
	}
}

func TestTryParseLooseDate(t *testing.T) {
	for _, v := range []string{"2024-01-02", "01/02/2024", "13/02/2024", "2024-01-02 15:04"} {
		if !TryParseLooseDate(v) {
			t.Errorf("TryParseLooseDate(%q) = false, want true", v)
		}
	}
	for _, v := range []string{"Infusion Pump", "IP-2000", ""} {
		if TryParseLooseDate(v) {
			t.Errorf("TryParseLooseDate(%q) = true, want false", v)
		}
	}
}
