package classify

// Normalize is the single-record entry point: it takes one raw device record
// from any external system and returns the canonical system-agnostic result.
// Callers supply their own taxonomy (LoadTaxonomy / LoadTaxonomyFile); a nil
// taxonomy yields an invalid result rather than panicking.
//
// This is the function any migrator calls per row.
func Normalize(in Input, tax *Taxonomy) Result {
	row := map[string]string{
		"Legacy source name": in.DeviceName,
		"Source device type": in.SourceType,
		"EMDN term":          in.EMDNTerm,
	}

	resolved := ResolveRowNamingFor(row, tax)

	fields := resolved.Fields
	if len(fields) == 0 {
		fields = nil
	}
	return Result{
		Name:             resolved.Name,
		Fields:           fields,
		LegacySourceName: in.DeviceName,
		SourceType:       in.SourceType,
		EMDNCode:         in.EMDNCode,
		EMDNTerm:         in.EMDNTerm,
		MappingSource:    resolved.NamingSource,
		Confidence:       confidenceFor(resolved),
	}
}

func confidenceFor(r ResolvedRow) string {
	switch r.NamingSource {
	case "unsupported_source_type":
		return "none"
	case "specific_rule":
		return "high"
	case "legacy_derived":
		if r.GetField(FieldDeviceCategory) != "" {
			return "high"
		}
		return "medium"
	case "family_fallback":
		if r.GetField(FieldDeviceCategory) != "" {
			return "medium"
		}
		return "low"
	default:
		return "low"
	}
}

// NormalizeBatch normalizes a slice of inputs, preserving order. Invalid
// results (unsupported source type) are included with Confidence "none" so the
// caller can decide how to handle them — filter, report, or override.
func NormalizeBatch(inputs []Input, tax *Taxonomy) []Result {
	out := make([]Result, 0, len(inputs))
	for _, in := range inputs {
		out = append(out, Normalize(in, tax))
	}
	return out
}
