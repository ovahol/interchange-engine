package classify

// Input is one raw device record from any external system. Every field is
// optional and free-text — the engine normalizes it.
type Input struct {
	DeviceName string `json:"device_name,omitempty"`
	SourceType string `json:"source_type,omitempty"`
	EMDNCode   string `json:"emdn_code,omitempty"`
	EMDNTerm   string `json:"emdn_term,omitempty"`
}

// Field keys for the pluggable Fields map. These are conventional keys a
// vendor taxonomy may or may not declare — the engine never hardcodes a fixed
// dimension set. FieldDeviceType remains the most common key, but a taxonomy is
// free to use any keys it likes.
const (
	FieldDeviceType            = "device_type"
	FieldDeviceCategory        = "device_category"
	FieldDeviceFunction        = "device_function"
	FieldDeviceApplicationRisk = "device_application_risk"
	FieldDeviceFamily          = "device_family"
)

// Result is the minimal system-agnostic classification for one input. Fields is
// the sole storage for taxonomy dimensions; keys come from the vendor taxonomy.
// Name, the input echo, and the diagnostics are always available.
type Result struct {
	// Name is the normalized device name. Example: "ECG machine".
	Name string `json:"name"`

	// Fields holds the taxonomy dimensions resolved for this input, keyed by
	// the vendor taxonomy's field keys (see Field* constants for the
	// conventional ones). This is the primary storage; there are no fixed
	// dimension fields to mirror into.
	Fields map[string]string `json:"fields,omitempty"`

	// Input echo — kept for interchange/audit, not part of the taxonomy.
	LegacySourceName string `json:"legacy_source_name,omitempty"`
	SourceType       string `json:"source_type,omitempty"`
	EMDNCode         string `json:"emdn_code,omitempty"`
	EMDNTerm         string `json:"emdn_term,omitempty"`

	// Confidence and MappingSource are optional diagnostics
	// (high/medium/low/none).
	Confidence    string `json:"confidence,omitempty"`
	MappingSource string `json:"mapping_source,omitempty"`
}

// GetField returns Fields[key], or "" when absent.
func (r Result) GetField(key string) string {
	if r.Fields == nil {
		return ""
	}
	return r.Fields[key]
}

// SetField sets Fields[key]. It is a pure map write; there are no fixed fields
// to mirror into.
func (r *Result) SetField(key, value string) {
	if r.Fields == nil {
		r.Fields = make(map[string]string)
	}
	r.Fields[key] = value
}

// IsValid reports whether normalization actually resolved fields for this
// input. It doesn't check any specific dimension by name — a vendor whose
// taxonomy declares entirely different fields still gets a correct answer.
func (r Result) IsValid() bool {
	return len(r.Fields) > 0 && r.MappingSource != "unsupported_source_type"
}
