package classify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// CurrentSchemaVersion is the taxonomy schema version this package understands.
// Vendors define their own Taxonomy.ID/Version; this is only used for semver
// compatibility checks when a taxonomy is loaded.
const CurrentSchemaVersion = "1.0.0"

// FieldDef declares one vendor-defined classification dimension. A taxonomy is
// nothing more than a list of these — there is no dimension the engine treats
// as special. Vendors with 2 dimensions declare 2; vendors with 8 declare 8.
// The Field* constants in schema.go are just conventional keys the built-in
// vocabularies happen to use — nothing in the engine requires them.
type FieldDef struct {
	Key           string   `json:"key"`
	Label         string   `json:"label,omitempty"`
	Required      bool     `json:"required,omitempty"`
	AllowedValues []string `json:"allowed_values,omitempty"`
}

// Rule is one inference rule: if the input matches When, assign Set's
// key/value pairs to the result's fields (first rule to set a given field
// wins — later rules only fill in fields still unset).
//
// Requires gates the rule on fields *already resolved by earlier rules* in the
// same taxonomy's rule list, so vendors express multi-stage inference (e.g.
// "resolve device_type, then derive device_function/risk from it, then derive
// device_category from device_function") purely as ordering: list
// dimension-resolving rules before the rules that depend on them.
//
// A rule matches when: Requires is satisfied (AND, if present) AND at least one
// of Keywords/SourceTypes matches (OR between the two), OR — if neither
// Keywords nor SourceTypes is set — Requires alone is enough (a pure
// "defaults" rule).
type Rule struct {
	ID              string            `json:"id,omitempty"`
	Keywords        []string          `json:"keywords,omitempty"`
	ExcludeKeywords []string          `json:"exclude_keywords,omitempty"`
	SourceTypes     []string          `json:"source_types,omitempty"`
	Requires        map[string]string `json:"requires,omitempty"`
	Set             map[string]string `json:"set,omitempty"`
	Name            string            `json:"name,omitempty"`
	CanonicalName   string            `json:"canonical_name,omitempty"`
}

// NamingRule defines a controlled-vocabulary naming convention.
type NamingRule struct {
	Field       string `json:"field"`
	Rule        string `json:"rule"`
	GoodExample string `json:"goodExample,omitempty"`
	Avoid       string `json:"avoid,omitempty"`
}

// NameRefinementRule defines a keyword-based common/canonical name
// transformation rule.
type NameRefinementRule struct {
	TargetName    string   `json:"targetName"`
	Keywords      []string `json:"keywords,omitempty"`
	CommonName    string   `json:"commonName"`
	CanonicalName string   `json:"canonicalName,omitempty"`
}

// SearchAliasRule defines keyword-triggered search aliases.
type SearchAliasRule struct {
	Keywords []string `json:"keywords"`
	Aliases  []string `json:"aliases"`
}

// InferenceRules contains a vendor's ordered rule list plus generic
// text-processing configuration (word lists, not dimension-specific) used for
// name cleanup/humanization and search alias generation.
type InferenceRules struct {
	Rules []Rule `json:"rules,omitempty"`

	// Text-processing config. None of these presuppose any particular
	// classification dimension — they operate on the free-text device name.
	GenericLegacyHeads      []string             `json:"genericLegacyHeads,omitempty"`
	LegacyDescriptorPhrases []string             `json:"legacyDescriptorPhrases,omitempty"`
	NameRefinementRules     []NameRefinementRule `json:"nameRefinementRules,omitempty"`
	Acronyms                []string             `json:"acronyms,omitempty"`
	WordReplacements        map[string]string    `json:"wordReplacements,omitempty"`
	SearchAliasRules        []SearchAliasRule    `json:"searchAliasRules,omitempty"`
}

// Taxonomy is the complete specification of a vendor's controlled vocabulary
// (Fields) and inference logic (Inference.Rules). It is pure data — vendors
// author it as JSON in their own codebase/repo and load it with LoadTaxonomy.
type Taxonomy struct {
	ID            string     `json:"id"`
	Name          string     `json:"name,omitempty"`
	Version       string     `json:"version,omitempty"`
	SchemaVersion string     `json:"schemaVersion,omitempty"`
	Fields        []FieldDef `json:"fields"`

	NamingRules []NamingRule    `json:"namingRules,omitempty"`
	Inference   *InferenceRules `json:"inference,omitempty"`
	Source      string          `json:"source,omitempty"`
}

// Field returns the FieldDef for key, or nil if the taxonomy doesn't declare it.
func (t *Taxonomy) Field(key string) *FieldDef {
	if t == nil {
		return nil
	}
	for i := range t.Fields {
		if t.Fields[i].Key == key {
			return &t.Fields[i]
		}
	}
	return nil
}

// RequiredFieldKeys returns the keys of all fields marked Required.
func (t *Taxonomy) RequiredFieldKeys() []string {
	if t == nil {
		return nil
	}
	var keys []string
	for _, f := range t.Fields {
		if f.Required {
			keys = append(keys, f.Key)
		}
	}
	return keys
}

// parseSemver parses a strict semver "MAJOR.MINOR.PATCH" (no leading "v").
func parseSemver(v string) (major, minor, patch int, err error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, 0, 0, fmt.Errorf("empty version")
	}
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid semver %q: want MAJOR.MINOR.PATCH", v)
	}
	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid semver %q: %w", v, err)
	}
	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid semver %q: %w", v, err)
	}
	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid semver %q: %w", v, err)
	}
	if major < 0 || minor < 0 || patch < 0 {
		return 0, 0, 0, fmt.Errorf("invalid semver %q: negative component", v)
	}
	return major, minor, patch, nil
}

// isBreakingMismatch reports whether loaded vs current is a breaking change
// (different major version).
func isBreakingMismatch(loaded, current string) (bool, error) {
	ma, _, _, err := parseSemver(loaded)
	if err != nil {
		return false, err
	}
	mb, _, _, err := parseSemver(current)
	if err != nil {
		return false, err
	}
	return ma != mb, nil
}

// Validate checks a taxonomy for well-formedness: valid id/version, at least
// one field, and every required field carrying its controlled vocabulary
// (allowed_values).
func (t *Taxonomy) Validate() error {
	if strings.TrimSpace(t.ID) == "" {
		return fmt.Errorf("taxonomy: missing id")
	}
	ver := t.effectiveVersion()
	if strings.TrimSpace(ver) == "" {
		return fmt.Errorf("taxonomy: missing version/schemaVersion")
	}
	if _, _, _, err := parseSemver(ver); err != nil {
		return fmt.Errorf("taxonomy: %w", err)
	}
	if len(t.Fields) == 0 {
		return fmt.Errorf("taxonomy: at least one field must be defined")
	}
	hasRequired := false
	seen := make(map[string]bool, len(t.Fields))
	for _, f := range t.Fields {
		if strings.TrimSpace(f.Key) == "" {
			return fmt.Errorf("taxonomy: field missing key")
		}
		if seen[f.Key] {
			return fmt.Errorf("taxonomy: duplicate field key %q", f.Key)
		}
		seen[f.Key] = true
		if f.Required {
			hasRequired = true
			if len(f.AllowedValues) == 0 {
				return fmt.Errorf("taxonomy: required field %q missing allowed_values", f.Key)
			}
		}
	}
	if !hasRequired {
		return fmt.Errorf("taxonomy: at least one field must be required")
	}
	return nil
}

func (t *Taxonomy) effectiveVersion() string {
	if strings.TrimSpace(t.SchemaVersion) != "" {
		return strings.TrimSpace(t.SchemaVersion)
	}
	return strings.TrimSpace(t.Version)
}

func (t *Taxonomy) normalizeVersionAlias() {
	v := strings.TrimSpace(t.Version)
	sv := strings.TrimSpace(t.SchemaVersion)
	if v == "" && sv != "" {
		t.Version = sv
	}
	if sv == "" && v != "" {
		t.SchemaVersion = v
	}
	if v != "" && sv != "" && v != sv {
		// Prefer schemaVersion as canonical when both present but differ.
		t.Version = sv
	}
}

// Migrate upgrades a taxonomy decoded from an older minor/patch to the current
// schema. It returns an error on breaking (major) mismatch. On success the
// returned taxonomy has its version fields set to CurrentSchemaVersion.
func Migrate(t *Taxonomy) (*Taxonomy, error) {
	if t == nil {
		return nil, fmt.Errorf("taxonomy: nil")
	}
	t.normalizeVersionAlias()
	ver := t.effectiveVersion()
	if ver == "" {
		return nil, fmt.Errorf("taxonomy: missing version/schemaVersion")
	}
	if _, _, _, err := parseSemver(ver); err != nil {
		return nil, fmt.Errorf("taxonomy: %w", err)
	}
	breaking, err := isBreakingMismatch(ver, CurrentSchemaVersion)
	if err != nil {
		return nil, err
	}
	if breaking {
		return nil, fmt.Errorf("taxonomy: breaking version mismatch: got %q, want %q (major version differs)", ver, CurrentSchemaVersion)
	}
	t.Version = CurrentSchemaVersion
	t.SchemaVersion = CurrentSchemaVersion
	return t, nil
}

// LoadTaxonomy decodes, validates, and migrates a taxonomy JSON document.
func LoadTaxonomy(data []byte) (*Taxonomy, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("taxonomy: empty document")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var t Taxonomy
	if err := dec.Decode(&t); err != nil {
		return nil, fmt.Errorf("taxonomy: %w", err)
	}
	if dec.More() {
		return nil, fmt.Errorf("taxonomy: trailing content after JSON object")
	}
	t.normalizeVersionAlias()
	if strings.TrimSpace(t.ID) == "" {
		return nil, fmt.Errorf("taxonomy: missing id")
	}
	if strings.TrimSpace(t.effectiveVersion()) == "" {
		return nil, fmt.Errorf("taxonomy: missing version/schemaVersion")
	}
	ver := t.effectiveVersion()
	if _, _, _, err := parseSemver(ver); err != nil {
		return nil, fmt.Errorf("taxonomy: %w", err)
	}
	if breaking, err := isBreakingMismatch(ver, CurrentSchemaVersion); err != nil {
		return nil, err
	} else if breaking {
		return nil, fmt.Errorf("taxonomy: breaking version mismatch: got %q, want %q (major version differs)", ver, CurrentSchemaVersion)
	}
	if err := t.Validate(); err != nil {
		return nil, err
	}
	return Migrate(&t)
}

// LoadTaxonomyFile reads a file and delegates to LoadTaxonomy.
func LoadTaxonomyFile(path string) (*Taxonomy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("taxonomy: read %s: %w", path, err)
	}
	return LoadTaxonomy(data)
}
