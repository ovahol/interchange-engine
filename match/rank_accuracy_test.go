package match

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// rankAccuracyFixture is a labeled set of ranking examples. Extend it
// from real migration data whenever a matching rule changes; the
// metrics below are a regression guard, not a tuning knob.
type rankAccuracyFixture struct {
	Devices []struct {
		Note       string `json:"note"`
		Raw        string `json:"raw"`
		Candidates []struct {
			Name        string   `json:"name"`
			CommonNames []string `json:"common_names"`
		} `json:"candidates"`
		Want string `json:"want"`
	} `json:"devices"`
	Names []struct {
		Note    string   `json:"note"`
		Raw     string   `json:"raw"`
		Catalog []string `json:"catalog"`
		Want    string   `json:"want"`
	} `json:"names"`
}

func TestRankingAccuracy(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "rank_accuracy.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture rankAccuracyFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	var tp, fp, fn, tn int
	record := func(raw, got string, matched bool, want string) {
		switch {
		case matched && got == want:
			tp++
		case matched && want == "":
			fp++
			t.Errorf("%q: unexpected match %q", raw, got)
		case matched:
			fp++
			fn++
			t.Errorf("%q: matched %q, want %q", raw, got, want)
		case want == "":
			tn++
		default:
			fn++
			t.Errorf("%q: unresolved, want %q", raw, want)
		}
	}

	for _, c := range fixture.Devices {
		candidates := make([]Candidate, 0, len(c.Candidates))
		for _, cc := range c.Candidates {
			candidates = append(candidates, Candidate{
				Name:        cc.Name,
				CommonNames: cc.CommonNames,
			})
		}
		best, _, ok := rankDeviceCandidates(c.Raw, candidates)
		record(c.Raw, best.Name, ok, c.Want)
	}
	for _, c := range fixture.Names {
		// Mirror the production sequence: token ranking first, then
		// edit similarity for typos that share no token.
		got, ok := rankNames(c.Raw, c.Catalog)
		if !ok {
			got, ok = rankNamesEdit(c.Raw, c.Catalog)
		}
		record(c.Raw, got, ok, c.Want)
	}

	precision, recall, accuracy := rankingMetrics(tp, fp, fn, tn)
	t.Logf("ranking accuracy: precision=%.3f recall=%.3f accuracy=%.3f (tp=%d fp=%d fn=%d tn=%d)",
		precision, recall, accuracy, tp, fp, fn, tn)
	if precision < 0.95 {
		t.Errorf("precision %.3f below 0.95", precision)
	}
	if recall < 0.85 {
		t.Errorf("recall %.3f below 0.85", recall)
	}
	if accuracy < 0.90 {
		t.Errorf("accuracy %.3f below 0.90", accuracy)
	}
}

func rankingMetrics(tp, fp, fn, tn int) (precision, recall, accuracy float64) {
	if tp+fp > 0 {
		precision = float64(tp) / float64(tp+fp)
	}
	if tp+fn > 0 {
		recall = float64(tp) / float64(tp+fn)
	}
	if total := tp + fp + fn + tn; total > 0 {
		accuracy = float64(tp+tn) / float64(total)
	}
	return precision, recall, accuracy
}
