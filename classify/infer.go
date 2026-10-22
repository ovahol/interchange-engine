package classify

import (
	"regexp"
	"strings"
)

var nonAlnumRe = regexp.MustCompile(`[^a-z0-9]+`)
var nonAlnumPreserveCaseRe = regexp.MustCompile(`[^a-zA-Z0-9]+`)
var parenRe = regexp.MustCompile(`\([^)]*\)`)
var slashRe = regexp.MustCompile(`/`)

// Normalized converts text to lowercase alphanumeric tokens separated by single
// spaces.
func Normalized(text string) string {
	if text == "" {
		return ""
	}
	lower := strings.ToLower(text)
	replaced := nonAlnumRe.ReplaceAllString(lower, " ")
	parts := strings.Fields(replaced)
	return strings.Join(parts, " ")
}

// HasAny reports whether text contains any of the candidate tokens.
func HasAny(text string, tokens []string) bool {
	padded := " " + text + " "
	for _, token := range tokens {
		candidate := Normalized(token)
		if candidate == "" {
			continue
		}
		if strings.Contains(candidate, " ") || len(candidate) <= 4 {
			if strings.Contains(padded, " "+candidate+" ") {
				return true
			}
			continue
		}
		if strings.Contains(text, candidate) {
			return true
		}
	}
	return false
}

// inference returns tax's inference config, or an empty one when tax is nil.
func inference(tax *Taxonomy) *InferenceRules {
	if tax == nil || tax.Inference == nil {
		return &InferenceRules{}
	}
	return tax.Inference
}

// ruleMatches reports whether rule r fires given the text/source extracted from
// the input and the fields resolved so far by earlier rules.
func ruleMatches(r *Rule, text, source string, fields map[string]string) bool {
	for k, v := range r.Requires {
		if fields[k] != v {
			return false
		}
	}
	if len(r.ExcludeKeywords) > 0 && text != "" && HasAny(text, r.ExcludeKeywords) {
		return false
	}
	// A rule with only Requires (no Keywords/SourceTypes) is a pure "defaults"
	// rule: it fires whenever its Requires are satisfied.
	matched := len(r.Requires) > 0 && len(r.Keywords) == 0 && len(r.SourceTypes) == 0
	if len(r.Keywords) > 0 && text != "" && HasAny(text, r.Keywords) {
		matched = true
	}
	if len(r.SourceTypes) > 0 && source != "" && HasAny(source, r.SourceTypes) {
		matched = true
	}
	return matched
}

// ApplyRulesFor runs tax's ordered rule list against one input, once, top to
// bottom. Rules are generic: they know nothing about "device type" or "family"
// — they just assign whatever field keys the vendor's Set map names. The first
// rule to set a given field wins; later rules only fill in fields still unset,
// which is how a vendor expresses multi-stage inference (resolve dimension A,
// then derive B from A via Requires, then C from B) — purely through rule
// ordering, not engine code.
//
// After the ordered pass, any field still unset is given one more chance: if
// the taxonomy declares AllowedValues for that field and the normalized source
// type matches one of them directly, that value is used. This lets a vendor
// whose source-type strings already line up with their own vocabulary get
// classification for free, with zero rules.
func ApplyRulesFor(deviceName, sourceType, emdnTerm string, tax *Taxonomy) (fields map[string]string, name, canonicalName string, specific bool) {
	fields = map[string]string{}
	if tax == nil {
		return fields, "", "", false
	}
	parts := []string{}
	if deviceName != "" {
		parts = append(parts, deviceName)
	}
	if emdnTerm != "" {
		parts = append(parts, emdnTerm)
	}
	text := Normalized(strings.Join(parts, " "))
	source := Normalized(sourceType)

	if tax.Inference != nil {
		for i := range tax.Inference.Rules {
			r := &tax.Inference.Rules[i]
			if !ruleMatches(r, text, source, fields) {
				continue
			}
			for k, v := range r.Set {
				if fields[k] == "" {
					fields[k] = v
				}
			}
			if r.Name != "" && name == "" {
				name = r.Name
				// Only a rule that matches independent of any prior resolution
				// (no Requires) counts as "specific" — a rule gated by
				// Requires is by definition a fallback/default for whatever it
				// requires, and shouldn't outrank a legacy-derived name just
				// because it also happens to have Keywords/SourceTypes.
				if len(r.Requires) == 0 && (len(r.Keywords) > 0 || len(r.SourceTypes) > 0) {
					specific = true
				}
			}
			if r.CanonicalName != "" && canonicalName == "" {
				canonicalName = r.CanonicalName
			}
		}
	}

	if source != "" {
		for _, f := range tax.Fields {
			if fields[f.Key] != "" || len(f.AllowedValues) == 0 {
				continue
			}
			for _, val := range f.AllowedValues {
				if Normalized(val) == source {
					fields[f.Key] = val
					break
				}
			}
		}
	}

	return fields, name, canonicalName, specific
}

// CleanLegacySegmentFor removes descriptor phrases and unit artifacts under the
// given taxonomy.
func CleanLegacySegmentFor(text string, tax *Taxonomy) string {
	if text == "" {
		return ""
	}
	inf := inference(tax)
	cleaned := text
	for _, phrase := range inf.LegacyDescriptorPhrases {
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(phrase) + `\b`)
		cleaned = re.ReplaceAllString(cleaned, " ")
	}
	cleaned = regexp.MustCompile(`\b\d+(?:[.,]\d+)?\s*(?:mm|cm|m|ml|l|g|kg|v|w|hz|fr|gauge|ch|french|f|kva|kva|mhz|khz)\b`).ReplaceAllString(cleaned, " ")
	cleaned = regexp.MustCompile(`\b\d+\s*x\s*\d+\b`).ReplaceAllString(cleaned, " ")
	cleaned = regexp.MustCompile(`\b(?:set|pack|box|pair|kit)\s+of\s+\d+\b`).ReplaceAllString(cleaned, " ")
	cleaned = regexp.MustCompile(`\b\d+\s*(?:pieces?|pcs?|units?|tests?)\b`).ReplaceAllString(cleaned, " ")
	cleaned = nonAlnumPreserveCaseRe.ReplaceAllString(cleaned, " ")
	return strings.Join(strings.Fields(cleaned), " ")
}

// HumanizeNameFor title-cases text and preserves acronyms under the given
// taxonomy.
func HumanizeNameFor(text string, tax *Taxonomy) string {
	if text == "" {
		return ""
	}
	inf := inference(tax)
	t := text
	for k, v := range inf.WordReplacements {
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(k) + `\b`)
		t = re.ReplaceAllString(t, v)
	}
	words := strings.Fields(t)
	if len(words) == 0 {
		return ""
	}
	acronymMap := make(map[string]bool, len(inf.Acronyms))
	for _, a := range inf.Acronyms {
		acronymMap[strings.ToLower(a)] = true
	}
	loweredWords := []string{"and", "or", "for", "with", "of", "in", "on", "at", "to", "by", "the", "a", "an"}
	isLowered := func(w string) bool {
		for _, lw := range loweredWords {
			if w == lw {
				return true
			}
		}
		return false
	}
	for i, w := range words {
		lw := strings.ToLower(w)
		if acronymMap[lw] {
			words[i] = strings.ToUpper(lw)
		} else if i > 0 && isLowered(lw) {
			words[i] = lw
		} else {
			r := []rune(lw)
			words[i] = strings.ToUpper(string(r[0])) + string(r[1:])
		}
	}
	return strings.Join(words, " ")
}

// InferCommonNameFromLegacyFor extracts a common name from legacy text under the
// given taxonomy.
func InferCommonNameFromLegacyFor(legacySourceName string, tax *Taxonomy) string {
	if legacySourceName == "" {
		return ""
	}
	inf := inference(tax)
	raw := parenRe.ReplaceAllString(legacySourceName, " ")
	raw = slashRe.ReplaceAllString(raw, " ")
	segments := strings.Split(raw, ",")
	first := CleanLegacySegmentFor(segments[0], tax)
	if first == "" {
		for _, seg := range segments[1:] {
			c := CleanLegacySegmentFor(seg, tax)
			if c != "" {
				first = c
				break
			}
		}
	}
	if first == "" {
		return ""
	}
	modifiers := []string{}
	for _, seg := range segments[1:] {
		c := CleanLegacySegmentFor(seg, tax)
		if c != "" && len(strings.Fields(c)) <= 2 {
			modifiers = append(modifiers, c)
		}
	}
	firstWords := strings.Fields(first)
	head := strings.ToLower(firstWords[len(firstWords)-1])

	heads := make(map[string]struct{}, len(inf.GenericLegacyHeads))
	for _, h := range inf.GenericLegacyHeads {
		heads[Normalized(h)] = struct{}{}
	}
	var candidate string
	if _, ok := heads[head]; ok && len(modifiers) > 0 {
		candidate = strings.Join(append(modifiers, first), " ")
	} else {
		candidate = first
	}
	candidate = HumanizeNameFor(candidate, tax)
	wc := len(strings.Fields(candidate))
	if wc >= 1 && wc <= 6 {
		return candidate
	}
	return ""
}

// RefineDescriptiveNamesFor refines common and canonical names under the given
// taxonomy.
func RefineDescriptiveNamesFor(commonName, canonicalName, legacySourceName, emdnTerm string, tax *Taxonomy) (string, string) {
	if commonName == "" {
		return commonName, canonicalName
	}
	rules := inference(tax).NameRefinementRules
	if len(rules) == 0 {
		return commonName, canonicalName
	}
	parts := []string{}
	if legacySourceName != "" {
		parts = append(parts, legacySourceName)
	}
	if emdnTerm != "" {
		parts = append(parts, emdnTerm)
	}
	text := Normalized(strings.Join(parts, " "))

	for _, r := range rules {
		if strings.EqualFold(r.TargetName, commonName) {
			if len(r.Keywords) == 0 || HasAny(text, r.Keywords) {
				cName := r.CommonName
				canName := r.CanonicalName
				if canName == "" {
					canName = cName
				}
				return cName, canName
			}
		}
	}
	return commonName, canonicalName
}

// ResolvedRow holds intermediate resolution data for a row.
type ResolvedRow struct {
	Fields        map[string]string
	Name          string
	CanonicalName string
	CommonNames   []string
	NamingSource  string
}

// GetField returns the taxonomy value for key from Fields.
func (r ResolvedRow) GetField(key string) string {
	if r.Fields == nil {
		return ""
	}
	return r.Fields[key]
}

// requiredFieldsResolved reports whether every field the taxonomy marks Required
// has a non-empty value in fields. If the taxonomy declares no required fields,
// any non-empty resolution counts as a match.
func requiredFieldsResolved(fields map[string]string, tax *Taxonomy) bool {
	required := false
	if tax != nil {
		for _, f := range tax.Fields {
			if f.Required {
				required = true
				if fields[f.Key] == "" {
					return false
				}
			}
		}
	}
	if !required {
		return len(fields) > 0
	}
	return true
}

// ResolveRowNamingFor resolves names and taxonomy dimensions under the given
// taxonomy.
func ResolveRowNamingFor(row map[string]string, tax *Taxonomy) ResolvedRow {
	legacySourceName := row["Legacy source name"]
	sourceDeviceType := row["Source device type"]
	emdnTerm := row["EMDN term"]

	fields, ruleName, ruleCanonical, specific := ApplyRulesFor(legacySourceName, sourceDeviceType, emdnTerm, tax)

	if !requiredFieldsResolved(fields, tax) {
		return ResolvedRow{NamingSource: "unsupported_source_type"}
	}

	generatedCommon := InferCommonNameFromLegacyFor(legacySourceName, tax)

	var commonName, namingSource string
	switch {
	case specific && ruleName != "":
		commonName = ruleName
		namingSource = "specific_rule"
	case generatedCommon != "":
		commonName = generatedCommon
		namingSource = "legacy_derived"
	case ruleName != "":
		commonName = ruleName
		namingSource = "family_fallback"
	default:
		namingSource = "family_fallback"
	}

	canonicalName := ruleCanonical
	if canonicalName == "" {
		canonicalName = commonName
	}

	commonName, canonicalName = RefineDescriptiveNamesFor(commonName, canonicalName, legacySourceName, emdnTerm, tax)

	return ResolvedRow{
		Fields:        fields,
		Name:          commonName,
		CanonicalName: canonicalName,
		CommonNames:   BuildSearchAliasesFor(commonName, canonicalName, tax),
		NamingSource:  namingSource,
	}
}

// BuildSearchAliasesFor builds search aliases under the given taxonomy.
func BuildSearchAliasesFor(commonName, canonicalName string, tax *Taxonomy) []string {
	inf := inference(tax)
	values := []string{}
	add := func(v string) {
		if v == "" {
			return
		}
		cleaned := strings.Join(strings.Fields(strings.TrimSpace(v)), " ")
		if cleaned == "" {
			return
		}
		lowered := strings.ToLower(cleaned)
		for _, existing := range values {
			if strings.ToLower(existing) == lowered {
				return
			}
		}
		values = append(values, cleaned)
	}
	add(commonName)
	add(canonicalName)

	if len(inf.SearchAliasRules) > 0 {
		combined := Normalized(strings.Join([]string{commonName, canonicalName}, " "))
		for _, r := range inf.SearchAliasRules {
			if HasAny(combined, r.Keywords) {
				for _, a := range r.Aliases {
					add(a)
				}
			}
		}
	}
	if len(values) == 0 {
		return nil
	}
	return values
}
