package match

import (
	"fmt"
	"testing"
)

func benchmarkCatalog(n int) []Candidate {
	out := make([]Candidate, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, Candidate{
			Name:        fmt.Sprintf("Multiparametric patient monitor model %d", i),
			CommonNames: []string{fmt.Sprintf("Monitor %d", i), "Patient monitor"},
		})
	}
	return out
}

func benchmarkNameList(n int) []string {
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, fmt.Sprintf("Samsung Medical Device Co Ltd %d", i))
	}
	return out
}

func BenchmarkRankDeviceCandidates(b *testing.B) {
	candidates := benchmarkCatalog(1000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rankDeviceCandidates("Patient Monitor", candidates)
	}
}

func BenchmarkRankDeviceCandidatesPrepared(b *testing.B) {
	candidates := benchmarkCatalog(1000)
	tokens, _, _ := prepareDeviceCandidates(candidates)
	rawTokens := contentTokens("Patient Monitor")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rankDeviceCandidatesTokens(rawTokens, candidates, tokens)
	}
}

// benchmarkWideCatalog models a large catalog where only one entry
// shares words with the query — the case the token index is for.
func benchmarkWideCatalog(n int) []Candidate {
	out := make([]Candidate, 0, n+1)
	for i := 0; i < n; i++ {
		out = append(out, Candidate{
			Name: fmt.Sprintf("Catalog entry number %d", i),
		})
	}
	out = append(out, Candidate{Name: "Multiparametric patient monitor"})
	return out
}

func BenchmarkRankDeviceCandidatesWide(b *testing.B) {
	candidates := benchmarkWideCatalog(1000)
	tokens, _, _ := prepareDeviceCandidates(candidates)
	rawTokens := contentTokens("Patient Monitor")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rankDeviceCandidatesTokens(rawTokens, candidates, tokens)
	}
}

func BenchmarkRankDeviceCandidatesWideIndexed(b *testing.B) {
	candidates := benchmarkWideCatalog(1000)
	tokens, _, _ := prepareDeviceCandidates(candidates)
	index := newDeviceTokenIndex(tokens)
	rawTokens := contentTokens("Patient Monitor")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rankDeviceCandidatesShortlist(rawTokens, candidates, tokens, index.shortlist(len(candidates), rawTokens))
	}
}

func BenchmarkRankNamesEdit(b *testing.B) {
	names := benchmarkNameList(1000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rankNamesEdit("Samsun", names)
	}
}

func BenchmarkRankNamesEditPrepared(b *testing.B) {
	names := benchmarkNameList(1000)
	keys := make([][]rune, len(names))
	for i, n := range names {
		keys[i] = []rune(contentKey(n))
	}
	rawKey := []rune(contentKey("Samsun"))
	var ws levenshteinWorkspace
	rankNamesEditKeys("Samsun", rawKey, names, keys, &ws) // warm row buffers
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rankNamesEditKeys("Samsun", rawKey, names, keys, &ws)
	}
}
