package classify

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		name string
		in   Input
		want string // expected DeviceType
	}{
		{
			name: "ECG via monitoring equipment",
			in:   Input{DeviceName: "ECG machine, portable", SourceType: "monitoring equipment"},
			want: "Monitoring & Measurement Devices",
		},
		{
			name: "infusion pump",
			in:   Input{DeviceName: "Infusion pump, volumetric", SourceType: "infusion devices"},
			want: "Treatment, Surgical & Life Support Devices",
		},
		{
			name: "ultrasound via EMDN",
			in:   Input{DeviceName: "Ultrasound scanner", SourceType: "imaging nuclear medicine equipment", EMDNTerm: "Diagnostic ultrasound systems"},
			want: "Diagnostic & Imaging Devices",
		},
		{
			name: "unsupported source type",
			in:   Input{DeviceName: "Something", SourceType: "unknown category xyz"},
			want: "",
		},
		{
			name: "lab analyzer",
			in:   Input{DeviceName: "Chemistry analyzer", SourceType: "laboratory equipment"},
			want: "Laboratory & IVD Equipment",
		},
		{
			name: "oxygen concentrator via specific rule",
			in:   Input{DeviceName: "Oxygen concentrator, 5L", SourceType: "medical gas equipment"},
			want: "Medical Gas & Respiratory Devices",
		},
	}
	tax, err := LoadTaxonomyFile("testdata/fixture.json")
	if err != nil {
		t.Fatalf("load fixture taxonomy: %v", err)
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := Normalize(tc.in, tax)
			if got := result.GetField(FieldDeviceType); got != tc.want {
				t.Errorf("Normalize(%+v) device_type = %q, want %q (full result: %+v)", tc.in, got, tc.want, result)
			}
			if tc.want == "" && result.IsValid() {
				t.Errorf("expected invalid result for unsupported source type, got valid: %+v", result)
			}
			if tc.want != "" && !result.IsValid() {
				t.Errorf("expected valid result, got invalid: %+v", result)
			}
		})
	}
}

func TestNormalizeConfidence(t *testing.T) {
	tax := loadFixture(t)
	r := Normalize(Input{DeviceName: "ECG machine", SourceType: "monitoring equipment"}, tax)
	if r.Confidence == "none" {
		t.Errorf("expected non-none confidence for valid input, got %q", r.Confidence)
	}
	r2 := Normalize(Input{DeviceName: "Widget", SourceType: "unknown xyz"}, tax)
	if r2.Confidence != "none" {
		t.Errorf("expected none confidence for unsupported, got %q", r2.Confidence)
	}
}

func TestNormalizeBatch(t *testing.T) {
	tax := loadFixture(t)
	inputs := []Input{
		{DeviceName: "ECG machine", SourceType: "monitoring equipment"},
		{DeviceName: "Infusion pump", SourceType: "infusion devices"},
	}
	results := NormalizeBatch(inputs, tax)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].GetField(FieldDeviceType) == "" || results[1].GetField(FieldDeviceType) == "" {
		t.Errorf("expected both rows classified: %+v", results)
	}
}

func TestNormalizeStreamlinedFields(t *testing.T) {
	tax := loadFixture(t)
	r := Normalize(Input{DeviceName: "Infusion pump", SourceType: "infusion devices"}, tax)
	if got := r.GetField(FieldDeviceType); got != "Treatment, Surgical & Life Support Devices" {
		t.Errorf("unexpected device_type: %q", got)
	}
	if got := r.GetField(FieldDeviceCategory); got != "Therapeutic" {
		t.Errorf("device_category = %q, want Therapeutic", got)
	}
	if r.GetField(FieldDeviceFunction) == "" || r.GetField(FieldDeviceApplicationRisk) == "" {
		t.Errorf("expected function/risk, got %q / %q", r.GetField(FieldDeviceFunction), r.GetField(FieldDeviceApplicationRisk))
	}
}

func TestNormalizeEchoesInput(t *testing.T) {
	tax := loadFixture(t)
	r := Normalize(Input{DeviceName: "  ECG machine  ", EMDNCode: "X1", EMDNTerm: "Monitor"}, tax)
	if r.LegacySourceName != "  ECG machine  " {
		t.Errorf("LegacySourceName = %q, want raw input echo", r.LegacySourceName)
	}
	if r.EMDNCode != "X1" || r.EMDNTerm != "Monitor" {
		t.Errorf("EMDN echo lost: %+v", r)
	}
}

func TestNilTaxonomyDoesNotPanic(t *testing.T) {
	r := Normalize(Input{DeviceName: "ECG machine"}, nil)
	if r.IsValid() {
		t.Errorf("expected invalid result without a taxonomy: %+v", r)
	}
	if r.Confidence != "none" {
		t.Errorf("Confidence = %q, want none", r.Confidence)
	}
}

func TestLoadTaxonomyRejectsGarbage(t *testing.T) {
	for _, data := range [][]byte{
		nil,
		[]byte("   "),
		[]byte("{"),
		[]byte(`{"id":"","version":"1.0.0","fields":[]}`),
		[]byte(`{"id":"x","version":"1.0.0","fields":[]}`),
		[]byte(`{"id":"x","version":"2.0.0","fields":[{"key":"a","required":true,"allowed_values":["A"]}]}`),
	} {
		if _, err := LoadTaxonomy(data); err == nil {
			t.Errorf("LoadTaxonomy(%q) unexpectedly succeeded", data)
		}
	}
}

func loadFixture(t *testing.T) *Taxonomy {
	t.Helper()
	tax, err := LoadTaxonomyFile("testdata/fixture.json")
	if err != nil {
		t.Fatalf("load fixture taxonomy: %v", err)
	}
	return tax
}
