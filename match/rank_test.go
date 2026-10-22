package match

import (
	"testing"
)

func TestRankSlitLampMicroscope(t *testing.T) {
	candidates := []Candidate{
		{Name: "Slit lamp", CommonNames: []string{"Slit lamp microscope"}},
		{Name: "Microscope, laboratory", CommonNames: []string{"Microscope"}},
	}
	best, score, ok := rankDeviceCandidates("Slit Lamp Microscope", candidates)
	if !ok {
		t.Fatal("expected a strict winner")
	}
	if best.Name != "Slit lamp" {
		t.Fatalf("expected Slit lamp, got %q (score %.2f)", best.Name, score)
	}
}

func TestRankTieStaysUnresolved(t *testing.T) {
	candidates := []Candidate{
		{Name: "Infusion pump"},
		{Name: "Suction pump"},
	}
	if _, _, ok := rankDeviceCandidates("Pump", candidates); ok {
		t.Fatal("tied candidates must stay unresolved")
	}
}

func TestRankWeakMatchStaysUnresolved(t *testing.T) {
	candidates := []Candidate{
		{Name: "Sterilizing Units, Steam"},
	}
	if _, _, ok := rankDeviceCandidates("Cart (Sterilization)", candidates); ok {
		t.Fatal("weak overlap must stay unresolved")
	}
}

func TestRankEmpty(t *testing.T) {
	if _, _, ok := rankDeviceCandidates("", []Candidate{{Name: "Pump"}}); ok {
		t.Fatal("empty raw text must not resolve")
	}
	if _, _, ok := rankDeviceCandidates("Pump", nil); ok {
		t.Fatal("no candidates must not resolve")
	}
}

func TestRankNamesBrandStrictWinner(t *testing.T) {
	best, ok := rankNames("GE Healthcare Systems", []string{"GE Healthcare", "GE Medical"})
	if !ok || best != "GE Healthcare" {
		t.Fatalf("expected GE Healthcare, got %q (ok=%v)", best, ok)
	}
}

func TestRankNamesSharedTokenOnlyTie(t *testing.T) {
	// "GE Health" shares only "ge" with both candidates — a tie that
	// must stay unresolved rather than guess a division.
	if best, ok := rankNames("GE Health", []string{"GE Healthcare", "GE Medical"}); ok {
		t.Fatalf("must stay unresolved, got %q", best)
	}
}

func TestRankNamesTieStaysUnresolved(t *testing.T) {
	if best, ok := rankNames("Abi", []string{"Abi Medical", "Abi Surgical"}); ok {
		t.Fatalf("tied names must stay unresolved, got %q", best)
	}
}

func TestRankNamesSingleResolves(t *testing.T) {
	best, ok := rankNames("Shenzhen Luckcome", []string{"Shenzhen Luckcome Technology"})
	if !ok || best != "Shenzhen Luckcome Technology" {
		t.Fatalf("expected single candidate, got %q (ok=%v)", best, ok)
	}
}

func TestRankRejectsFragmentMatches(t *testing.T) {
	// "220" shares a short fragment with "BioPlex 2200" but covers
	// little of either side: must stay unresolved, never auto-trust
	// a lone containment candidate.
	if best, ok := rankNames("220", []string{"BioPlex 2200"}); ok {
		t.Fatalf("fragment must stay unresolved, got %q", best)
	}
	// "YP-GM1" shares only the 2-letter fragment "yp" with "YP-100".
	if best, ok := rankNames("YP-GM1", []string{"YP-100"}); ok {
		t.Fatalf("fragment must stay unresolved, got %q", best)
	}
}

func TestRankDeviceRejectsFragmentMatch(t *testing.T) {
	candidates := []Candidate{
		{Name: "BioPlex 2200"},
	}
	if _, _, ok := rankDeviceCandidates("220", candidates); ok {
		t.Fatal("fragment must stay unresolved")
	}
}

func TestEditSimilarityValues(t *testing.T) {
	if s := editSimilarity("Samsun", "Samsung"); s < 0.8 {
		t.Fatalf("Samsun/Samsung should score >= 0.8, got %.3f", s)
	}
	if s := editSimilarity("Omron Heathcare Co. Ltd", "Omron Healthcare Co., Ltd"); s < 0.8 {
		t.Fatalf("heathcare typo should score >= 0.8, got %.3f", s)
	}
	if s := editSimilarity("Pump", "Ventilator"); s >= 0.8 {
		t.Fatalf("unrelated names should score low, got %.3f", s)
	}
	if s := editSimilarity("", "Pump"); s != 0 {
		t.Fatalf("empty input should score 0, got %.3f", s)
	}
}

func TestRankNamesEditTypoWinner(t *testing.T) {
	best, ok := rankNamesEdit("Samsun", []string{"Samsung", "Siemens"})
	if !ok || best != "Samsung" {
		t.Fatalf("expected Samsung, got %q (ok=%v)", best, ok)
	}
}

func TestRankNamesEditTieStaysUnresolved(t *testing.T) {
	if best, ok := rankNamesEdit("Hospivax", []string{"Hospivac", "Hospivak"}); ok {
		t.Fatalf("near-tie must stay unresolved, got %q", best)
	}
}

func TestTokenizeSplitsPunctuationAndFoldsDiacritics(t *testing.T) {
	assertTokens(t, "Phoenix Medical System (Pl)Ltd", []string{"phoenix", "medical", "system", "pl", "ltd"})
	assertTokens(t, "Dräger", []string{"drager"})
}

func assertTokens(t *testing.T, in string, want []string) {
	t.Helper()
	got := tokenize(in)
	if len(got) != len(want) {
		t.Fatalf("tokenize(%q) = %q, want %q", in, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tokenize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestContentTokensDropLegalSuffixes(t *testing.T) {
	got := contentTokens("Shenzhen Comen Medical Instruments Co., Ltd.")
	for _, stop := range []string{"co", "ltd"} {
		for _, tok := range got {
			if tok == stop {
				t.Fatalf("stop-word %q survived in %q", stop, got)
			}
		}
	}
	if len(got) == 0 {
		t.Fatal("content tokens must not be empty for a real name")
	}
}

func TestBogusCorporateWinnerRejected(t *testing.T) {
	// CIGA shares only generic words with Trivitron: no strict winner.
	if best, ok := rankNames("CIGA Healthcare Ltd", []string{"Trivitron Healthcare Pvt. Ltd.", "Unrelated Devices"}); ok {
		t.Fatalf("generic-word overlap must not resolve, got %q", best)
	}
}

// TestRankDevicePrefersShortestContainer pins the Patient Monitor
// case: among several catalog names containing the raw text, the
// shortest container ("Multiparametric patient monitor") strictly
// outscores its longer qualified siblings and resolves — provided
// the candidate query returns it (ORDER BY LENGTH ASC, wider LIMIT).
func TestRankDevicePrefersShortestContainer(t *testing.T) {
	candidates := []Candidate{
		{Name: "Multiparametric advanced patient monitor"},
		{Name: "Multiparametric basic patient monitor"},
		{Name: "Multiparametric intermediate patient monitor"},
		{Name: "Multiparametric patient monitor"},
		{Name: "Multiparametric transport patient monitor"},
	}
	best, _, ok := rankDeviceCandidates("Patient Monitor", candidates)
	if !ok {
		t.Fatal("shortest container must resolve")
	}
	if best.Name != "Multiparametric patient monitor" {
		t.Fatalf("expected Multiparametric patient monitor, got %q", best.Name)
	}
}

// TestRankDeviceRejectsSingleTokenAliasPartialWin pins the hypernym
// trap: "Medical Trolley/Cart" shares only container words ("cart"
// via alias, "trolley") with "Clean linen cart" — no distinctive
// token, so it must stay unresolved even as strict winner. Bare
// hypernyms ("Cart") are equally unresolvable: refusing is the
// no-guess rule, not a miss.
func TestRankDeviceRejectsSingleTokenAliasPartialWin(t *testing.T) {
	cart := Candidate{
		Name:        "Clean linen cart",
		CommonNames: []string{"linen trolley", "cart"},
	}
	other := Candidate{Name: "Defibrillator trolley"}
	if _, _, ok := rankDeviceCandidates("Medical Trolley/Cart", []Candidate{cart, other}); ok {
		t.Fatal("hypernym alias must stay unresolved")
	}
	if _, _, ok := rankDeviceCandidates("Cart", []Candidate{cart, other}); ok {
		t.Fatal("bare hypernym must stay unresolved")
	}
}

func TestMatchQualityCountsDuplicateTokensOnce(t *testing.T) {
	// Repeated tokens must not inflate F1 or recall above 1.
	if s := matchQuality([]string{"pump"}, []string{"pump", "pump"}); s.f1 != 1 || s.recall != 1 || s.matched != 1 {
		t.Fatalf("duplicate alias tokens: %+v, want f1=1 recall=1 matched=1", s)
	}
	if s := matchQuality([]string{"pump", "pump"}, []string{"pump"}); s.f1 != 1 || s.recall != 1 || s.matched != 1 {
		t.Fatalf("duplicate raw tokens: %+v, want f1=1 recall=1 matched=1", s)
	}
	if s := matchQuality([]string{"infusion", "pump"}, []string{"pump", "pump"}); s.f1 > 1 || s.recall > 1 {
		t.Fatalf("scores must stay within [0,1]: %+v", s)
	}
}

func TestRankRejectsBrandOnlyOverlap(t *testing.T) {
	// "Siemens CT" and "Siemens MRI" share only the brand word. A lone
	// candidate must not win on that overlap alone.
	candidates := []Candidate{{Name: "Siemens MRI"}}
	if best, _, ok := rankDeviceCandidates("Siemens CT", candidates); ok {
		t.Fatalf("brand-only overlap must stay unresolved, got %q", best.Name)
	}
	if best, ok := rankNames("Siemens CT", []string{"Siemens MRI"}); ok {
		t.Fatalf("brand-only overlap must stay unresolved, got %q", best)
	}
}

func TestRankKeepsGenericOnlyDifference(t *testing.T) {
	// The differing words are generic, so the shared distinctive word
	// still resolves: "x-ray system" → "x-ray machine".
	if best, ok := rankNames("x-ray system", []string{"x-ray machine"}); !ok || best != "x-ray machine" {
		t.Fatalf("generic-only difference should resolve, got %q (ok=%v)", best, ok)
	}
}

func TestRankNamesEditIgnoresCorporateSuffixNoise(t *testing.T) {
	// A one-letter typo plus "Co Ltd" noise must still resolve: the
	// edit path compares identity words, not the raw strings.
	best, ok := rankNamesEdit("Samsun", []string{"Samsung Co Ltd", "Siemens"})
	if !ok || best != "Samsung Co Ltd" {
		t.Fatalf("expected suffix-tolerant typo match, got %q (ok=%v)", best, ok)
	}
}

func TestContentTokensFoldPlurals(t *testing.T) {
	cases := map[string]string{
		"Pumps":     "pump",
		"Batteries": "battery",
		"Devices":   "device",
		"Trolleys":  "trolley",
		"Systems":   "system",
		"Boxes":     "box",
	}
	for in, want := range cases {
		if got := contentTokens(in); len(got) != 1 || got[0] != want {
			t.Fatalf("contentTokens(%q) = %q, want [%s]", in, got, want)
		}
	}
}

func TestRankMatchesPluralVariant(t *testing.T) {
	// "Pumps" and "Pump" must meet after plural folding.
	if best, ok := rankNames("Infusion Pumps", []string{"Infusion Pump"}); !ok || best != "Infusion Pump" {
		t.Fatalf("plural variant should resolve, got %q (ok=%v)", best, ok)
	}
}

func TestStemTokenFoldsDerivations(t *testing.T) {
	cases := map[string]string{
		"Sterilizing":   "sterilize",
		"Sterilization": "sterilize",
		"Calibrating":   "calibrate",
		"Calibration":   "calibrate",
		"Organizations": "organize",
		"Station":       "station", // length guard: not folded
	}
	for in, want := range cases {
		if got := contentTokens(in); len(got) != 1 || got[0] != want {
			t.Fatalf("contentTokens(%q) = %q, want [%s]", in, got, want)
		}
	}
}

func TestRankMatchesDerivationalVariant(t *testing.T) {
	if best, ok := rankNames("Sterilizing Unit", []string{"Sterilization unit"}); !ok || best != "Sterilization unit" {
		t.Fatalf("derivational variant should resolve, got %q (ok=%v)", best, ok)
	}
}

func TestRankAcceptsShorterFullyCoveredName(t *testing.T) {
	// The candidate is fully explained by the raw text (the extra raw
	// word is a qualifier), so the shorter name wins strictly over the
	// longer one, which leaves a word unexplained on both sides.
	if best, ok := rankNames("Luckcome Tech", []string{"Luckcome", "Luckcome Technology"}); !ok || best != "Luckcome" {
		t.Fatalf("expected Luckcome, got %q (ok=%v)", best, ok)
	}
}

func TestFoldMarkFoldsUppercaseAndExtendedLatin(t *testing.T) {
	assertTokens(t, "Łóżko", []string{"lozko"})
	assertTokens(t, "Ștefan", []string{"stefan"})
	assertTokens(t, "Ţeavă", []string{"teava"})
	assertTokens(t, "DrÄger", []string{"drager"})
}

func TestFoldMarkExpandsLigatures(t *testing.T) {
	assertTokens(t, "Straße", []string{"strasse"})
	assertTokens(t, "Æther", []string{"aether"})
	assertTokens(t, "Œuvre", []string{"oeuvre"})
	assertTokens(t, "Þór", []string{"thor"})
}

func TestStemTokenKeepsNonPlurals(t *testing.T) {
	for _, w := range []string{"lens", "bias", "series", "species", "news", "diabetes", "mumps"} {
		if got := contentTokens(w); len(got) != 1 || got[0] != w {
			t.Fatalf("contentTokens(%q) = %q, want [%s]", w, got, w)
		}
	}
}

func TestContentKeyDedupesTokens(t *testing.T) {
	if got := contentKey("Pump Pump Co Ltd"); got != "pump" {
		t.Fatalf("contentKey = %q, want %q", got, "pump")
	}
}

func TestRankMatchesLigatureVariant(t *testing.T) {
	if best, ok := rankNames("Straße", []string{"Strasse"}); !ok || best != "Strasse" {
		t.Fatalf("ligature variant should resolve, got %q (ok=%v)", best, ok)
	}
}

func TestDeviceShortlistMatchesFullScan(t *testing.T) {
	candidates := []Candidate{
		{Name: "Slit lamp", CommonNames: []string{"Slit lamp microscope"}},
		{Name: "Microscope, laboratory", CommonNames: []string{"Microscope"}},
		{Name: "Infusion pump"},
		{Name: "Suction pump"},
		{Name: "X-ray machine", CommonNames: []string{"X-ray machine"}},
	}
	tokens, _, _ := prepareDeviceCandidates(candidates)
	index := newDeviceTokenIndex(tokens)

	for _, raw := range []string{"Slit Lamp Microscope", "Pump", "X-ray system", "Siemens CT", "Patient Monitor", "Infusion Pumps"} {
		rawTokens := contentTokens(raw)
		wantCand, wantScore, wantOK := rankDeviceCandidatesTokens(rawTokens, candidates, tokens)
		indices := index.shortlist(len(candidates), rawTokens)
		gotCand, gotScore, gotOK := rankDeviceCandidatesShortlist(rawTokens, candidates, tokens, indices)
		if gotOK != wantOK || gotCand.Name != wantCand.Name || gotScore != wantScore {
			t.Fatalf("shortlist diverged for %q: full=(%q,%.3f,%v) short=(%q,%.3f,%v)",
				raw, wantCand.Name, wantScore, wantOK, gotCand.Name, gotScore, gotOK)
		}
	}

	// The index must actually narrow the field.
	if n := len(index.shortlist(len(candidates), contentTokens("X-ray system"))); n >= len(candidates) {
		t.Fatalf("expected a strict shortlist, got %d of %d", n, len(candidates))
	}
}

func TestNameShortlistMatchesFullScan(t *testing.T) {
	names := []string{"Luckcome", "Luckcome Technology", "GE Healthcare", "GE Medical", "Samsung", "Siemens", "Abi Medical", "Abi Surgical"}
	tokens, _ := prepareNames(names)
	index := newNameTokenIndex(tokens)

	for _, raw := range []string{"Luckcome Tech", "GE Health", "Samsun", "Abi", "Siemens", "Unknown"} {
		rawTokens := contentTokens(raw)
		wantName, wantOK := rankNamesTokens(rawTokens, names, tokens)
		indices := index.shortlist(len(names), rawTokens)
		gotName, gotOK := rankNamesTokensShortlist(rawTokens, names, tokens, indices)
		if gotOK != wantOK || gotName != wantName {
			t.Fatalf("shortlist diverged for %q: full=(%q,%v) short=(%q,%v)", raw, wantName, wantOK, gotName, gotOK)
		}
	}
}
