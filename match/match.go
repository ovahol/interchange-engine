package match

import (
	"strings"
	"unicode"

	"github.com/ovahol/interchange-engine/text"
)

// Candidate is one catalog entry to rank against: a canonical name plus
// its aliases.
type Candidate struct {
	Name        string
	CommonNames []string
}

// MinMatchLen is the shortest raw text that qualifies for name matching.
const MinMatchLen = 3

// minRankScore is the lowest token-overlap F1 a ranked device match
// must reach. Below it, even a strict winner is too weak to trust
// (e.g. sharing one generic word like "unit").
const minRankScore = 0.5

// corpStopWords are legal-entity tokens that carry no identity:
// without them, shared suffixes ("Co., Ltd.", "GmbH") inflate
// overlap scores and crown bogus strict winners (e.g. any "X Co.
// Ltd" outscoring its true match). Single letters go too ("S.P.A"
// contributes nothing; "GE" stays).
var corpStopWords = map[string]bool{
	"ltd": true, "limited": true, "inc": true, "incorporated": true,
	"corp": true, "corporation": true, "co": true, "company": true,
	"gmbh": true, "ag": true, "sa": true, "spa": true, "srl": true,
	"pvt": true, "plc": true, "llc": true, "nv": true, "kg": true,
	"sarl": true, "gbr": true, "ug": true,
}

// genericWords are domain-common tokens: fine as supporting overlap,
// but a match resting ONLY on these is not distinctive enough to
// trust (e.g. "CIGA Healthcare Ltd" sharing just "healthcare" with
// "Trivitron Healthcare"). Winners must share at least one
// non-generic token with the raw text. Plurals are folded by
// stemToken, so only the singular form is listed.
var genericWords = map[string]bool{
	"medical": true, "healthcare": true, "health": true,
	"technology": true,
	"system":     true,
	"equipment":  true, "device": true,
	"product":       true,
	"international": true, "global": true,
	"industry": true,
	"group":    true, "solution": true,
	"service": true, "care": true,
	"unit": true,
	// Container/furniture hypernyms: sharing "cart" proves nothing
	// about WHICH cart ("Medical Trolley/Cart" must not resolve to
	// "Clean linen cart" on that word alone). Every legitimate
	// match keeps another distinctive token ("Operating Table" has
	// "operating", "Infusion Pump" has "infusion"), while bare
	// hypernyms ("Trolley", "Bed", "Monitor") correctly stay
	// unresolved for a human to place.
	"cart": true, "trolley": true,
	"table": true, "stand": true,
	"bed": true, "lamp": true,
	"light": true, "chair": true,
	"pump": true, "scale": true,
	"monitor": true, "machine": true,
	// Entity words: sharing "brand"/"model" proves nothing ("Rec
	// Brand 1" must not resolve to "Unknown Brand" on that word
	// alone).
	"brand":        true,
	"manufacturer": true,
	"model":        true,
}

// containsToken reports whether t appears in tokens. Token lists are
// short (a handful of words), so a linear scan beats allocating a map
// per comparison.
func containsToken(tokens []string, t string) bool {
	for _, x := range tokens {
		if x == t {
			return true
		}
	}
	return false
}

// sharedDistinctive reports whether raw and alias share at least one
// token that is neither a stop-word nor generic.
func sharedDistinctive(rawTokens, aliasTokens []string) bool {
	for _, t := range aliasTokens {
		if !genericWords[t] && containsToken(rawTokens, t) {
			return true
		}
	}
	return false
}

// tokenize splits text into lowercase alphanumeric words: punctuation
// marks word boundaries ("(Pl)Ltd" → pl + ltd, "S.P.A" → s + p + a)
// and diacritics fold ("Dräger" → drager).
func tokenize(s string) []string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			text.WriteFolded(&b, r)
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.Fields(b.String())
}

// contentTokens are a name's identity-carrying words: stop-words and
// single letters dropped ("S.P.A" contributes nothing; "GE" stays).
func contentTokens(s string) []string {
	var out []string
	for _, t := range tokenize(s) {
		if len(t) < 2 {
			continue
		}
		t = stemToken(t)
		if corpStopWords[t] {
			continue
		}
		out = append(out, t)
	}
	return out
}

// nonPluralWords end in "s" but are not plurals; stemming them would
// create false collisions ("lens" → "len", "series" → "serie").
var nonPluralWords = map[string]bool{
	"lens": true, "bias": true, "series": true, "species": true,
	"news": true, "atlas": true, "canvas": true, "alias": true,
	"chaos": true, "cosmos": true, "ethos": true, "pathos": true,
	"diabetes": true, "herpes": true, "rabies": true, "scabies": true,
	"caries": true, "measles": true, "mumps": true,
}

// stemToken folds conservative English inflection: plurals and the
// common -ies case. It deliberately stops short of a full Porter
// stemmer, which would over-merge proper nouns and model numbers.
func stemToken(t string) string {
	if len(t) < 4 || nonPluralWords[t] {
		return t
	}
	return stemDerivation(stemPlural(t))
}

func stemPlural(t string) string {
	switch {
	case strings.HasSuffix(t, "ies"):
		return t[:len(t)-3] + "y"
	case strings.HasSuffix(t, "sses"), strings.HasSuffix(t, "shes"),
		strings.HasSuffix(t, "ches"), strings.HasSuffix(t, "xes"),
		strings.HasSuffix(t, "zes"):
		return t[:len(t)-2]
	case strings.HasSuffix(t, "s") && !strings.HasSuffix(t, "ss") &&
		!strings.HasSuffix(t, "us") && !strings.HasSuffix(t, "is"):
		return t[:len(t)-1]
	}
	return t
}

// stemDerivation folds a few high-confidence -ation/-ization pairs
// ("sterilizing"/"sterilization" → "sterilize", "calibration"/
// "calibrating" → "calibrate"). The length guard keeps shorter nouns
// ("station", "ration") out of the fold.
func stemDerivation(t string) string {
	if len(t) < 8 {
		return t
	}
	switch {
	case strings.HasSuffix(t, "ization"), strings.HasSuffix(t, "isation"):
		return t[:len(t)-7] + "ize"
	case strings.HasSuffix(t, "izing"), strings.HasSuffix(t, "ising"):
		return t[:len(t)-5] + "ize"
	case strings.HasSuffix(t, "ation"), strings.HasSuffix(t, "ating"):
		return t[:len(t)-5] + "ate"
	}
	return t
}

// contentKey joins a name's identity words for edit comparison. Legal
// suffixes ("Co., Ltd") and bracketed qualifiers are dropped first, so
// noise does not dilute the distance between two otherwise-equal names
// ("Samsun" vs "Samsung Co Ltd").
func contentKey(s string) string {
	tokens := contentTokens(text.NormKey(s))
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if !containsToken(out, t) {
			out = append(out, t)
		}
	}
	return strings.Join(out, " ")
}

// rankDeviceCandidates picks the best catalog candidate for raw text by
// token overlap over each candidate's canonical name and aliases.
// Returns ok=false unless the winner strictly beats the runner-up and
// is convincing (score bar plus full raw coverage, two shared words, or
// one name fully explained by the other): ties, weak and fragment
// matches stay unresolved (the no-guess rule). Single candidates are
// scored, never auto-trusted.
func rankDeviceCandidates(raw string, candidates []Candidate) (Candidate, float64, bool) {
	rawTokens := contentTokens(raw)
	if len(rawTokens) == 0 {
		return Candidate{}, 0, false
	}
	tokens, _, _ := prepareDeviceCandidates(candidates)
	return rankDeviceCandidatesTokens(rawTokens, candidates, tokens)
}

// rankDeviceCandidatesTokens is rankDeviceCandidates over precomputed
// alias token sets (one []string per alias, canonical name first). The
// matcher reuses these across rows instead of re-tokenizing the catalog
// for every field.
func rankDeviceCandidatesTokens(rawTokens []string, candidates []Candidate, aliasTokens [][][]string) (Candidate, float64, bool) {
	return rankDeviceCandidatesCore(rawTokens, candidates, aliasTokens, nil, true)
}

// rankDeviceCandidatesShortlist is the indexed variant: it scores only
// the candidates whose token set intersects the query. Candidates
// outside the shortlist share no token and therefore score zero, so
// they can neither win nor change the runner-up.
func rankDeviceCandidatesShortlist(rawTokens []string, candidates []Candidate, aliasTokens [][][]string, indices []int) (Candidate, float64, bool) {
	return rankDeviceCandidatesCore(rawTokens, candidates, aliasTokens, indices, false)
}

func rankDeviceCandidatesCore(rawTokens []string, candidates []Candidate, aliasTokens [][][]string, indices []int, considerAll bool) (Candidate, float64, bool) {
	if len(rawTokens) == 0 {
		return Candidate{}, 0, false
	}
	var best matchScore
	var bestTokens []string
	var bestCand Candidate
	runnerUp := 0.0
	consider := func(i int) {
		score, winner := bestAliasScore(rawTokens, aliasTokens[i])
		if score.f1 > best.f1 {
			runnerUp, best, bestCand, bestTokens = best.f1, score, candidates[i], winner
		} else if score.f1 > runnerUp {
			runnerUp = score.f1
		}
	}
	if considerAll {
		for i := range candidates {
			consider(i)
		}
	} else {
		for _, i := range indices {
			consider(i)
		}
	}
	if best.f1 <= runnerUp || !best.convincing() {
		return Candidate{}, best.f1, false
	}
	if !sharedDistinctive(rawTokens, bestTokens) {
		return Candidate{}, best.f1, false
	}
	return bestCand, best.f1, true
}

// prepareDeviceCandidates tokenizes every alias once and builds the
// flat name/key lists the edit fallback needs.
func prepareDeviceCandidates(candidates []Candidate) (aliasTokens [][][]string, names []string, keys [][]rune) {
	aliasTokens = make([][][]string, len(candidates))
	total := 0
	for _, c := range candidates {
		total += 1 + len(c.CommonNames)
	}
	names = make([]string, 0, total)
	keys = make([][]rune, 0, total)
	for i, c := range candidates {
		sets := make([][]string, 0, 1+len(c.CommonNames))
		sets = append(sets, contentTokens(c.Name))
		names = append(names, c.Name)
		keys = append(keys, []rune(contentKey(c.Name)))
		for _, alias := range c.CommonNames {
			sets = append(sets, contentTokens(alias))
			names = append(names, alias)
			keys = append(keys, []rune(contentKey(alias)))
		}
		aliasTokens[i] = sets
	}
	return aliasTokens, names, keys
}

// bestAliasScore is the best match quality across a candidate's alias
// token sets, plus the winning set (used for the distinctiveness gate).
func bestAliasScore(rawTokens []string, aliasTokens [][]string) (matchScore, []string) {
	var best matchScore
	var bestTokens []string
	for _, tokens := range aliasTokens {
		if s := matchQuality(rawTokens, tokens); betterScore(s, best) {
			best, bestTokens = s, tokens
		}
	}
	return best, bestTokens
}

// matchScore is one alias's quality against the raw text.
type matchScore struct {
	f1      float64
	recall  float64
	matched int
	// Distinctive (non-generic) tokens present on only one side. A
	// zero on either side means one name is fully explained by the
	// other ("Luckcome Tech" vs "Luckcome"); zeros on neither means
	// the names disagree on a specific word ("Siemens CT" vs
	// "Siemens MRI").
	rawUnmatchedDistinctive   int
	aliasUnmatchedDistinctive int
}

func (s matchScore) unexplained() int {
	return s.rawUnmatchedDistinctive + s.aliasUnmatchedDistinctive
}

// convincing reports whether a strict-winning score is strong enough
// to trust: the F1 bar plus full raw coverage, two distinct shared
// words, or one side fully explained by the other. This keeps
// "x-ray system" → "x-ray machine" and "Luckcome Tech" → "Luckcome"
// while refusing brand-only overlaps ("Siemens CT" → "Siemens MRI")
// and fragment matches ("220", "YP-GM1").
func (s matchScore) convincing() bool {
	if s.f1 < minRankScore {
		return false
	}
	return s.recall == 1 || s.matched >= 2 ||
		s.rawUnmatchedDistinctive == 0 || s.aliasUnmatchedDistinctive == 0
}

// betterScore orders alias scores: higher F1, then higher raw
// coverage, then more matched words, then fewer unexplained
// distinctive words.
func betterScore(a, b matchScore) bool {
	if a.f1 > b.f1 {
		return true
	}
	if a.f1 < b.f1 {
		return false
	}
	if a.recall > b.recall {
		return true
	}
	if a.recall < b.recall {
		return false
	}
	if a.matched > b.matched {
		return true
	}
	if a.matched < b.matched {
		return false
	}
	return a.unexplained() < b.unexplained()
}

// matchQuality scores raw text against one alias by distinct-token
// overlap. Repeated tokens count once on each side, so "pump pump"
// cannot push a score above 1. Token lists are short, so linear scans
// avoid the per-comparison map allocations a whole-catalog rank would
// otherwise pay for every candidate.
func matchQuality(rawTokens, aliasTokens []string) matchScore {
	if len(rawTokens) == 0 || len(aliasTokens) == 0 {
		return matchScore{}
	}
	rawDistinct := 0
	for i, t := range rawTokens {
		if !containsToken(rawTokens[:i], t) {
			rawDistinct++
		}
	}
	aliasDistinct := 0
	matched := 0
	for i, t := range aliasTokens {
		if containsToken(aliasTokens[:i], t) {
			continue
		}
		aliasDistinct++
		if containsToken(rawTokens, t) {
			matched++
		}
	}
	if matched == 0 || rawDistinct == 0 {
		return matchScore{}
	}
	precision := float64(matched) / float64(aliasDistinct)
	recall := float64(matched) / float64(rawDistinct)
	rawOnly, aliasOnly := unmatchedDistinctive(rawTokens, aliasTokens)
	return matchScore{
		f1:                        2 * precision * recall / (precision + recall),
		recall:                    recall,
		matched:                   matched,
		rawUnmatchedDistinctive:   rawOnly,
		aliasUnmatchedDistinctive: aliasOnly,
	}
}

// unmatchedDistinctive returns the distinctive (non-generic) tokens
// present only in raw and only in alias, respectively.
func unmatchedDistinctive(rawTokens, aliasTokens []string) (rawOnly, aliasOnly int) {
	for i, t := range rawTokens {
		if containsToken(rawTokens[:i], t) {
			continue
		}
		if !containsToken(aliasTokens, t) && !genericWords[t] {
			rawOnly++
		}
	}
	for i, t := range aliasTokens {
		if containsToken(aliasTokens[:i], t) {
			continue
		}
		if !containsToken(rawTokens, t) && !genericWords[t] {
			aliasOnly++
		}
	}
	return rawOnly, aliasOnly
}

// minEditScore is the lowest normalized-Levenshtein similarity a
// typo-tolerant match must reach ("Samsun"/"Samsung" scores ~0.86).
// Strict-winner applies as everywhere: near-ties stay unresolved.
const minEditScore = 0.8

// editSimilarity scores two names by normalized Levenshtein similarity
// over separator-folded lowercase (so "CO, LTD" and "Co. Ltd" compare
// equal before distance runs). Single-token typos ("Samsun" vs
// "Samsung") score here where token overlap sees nothing in common.
func editSimilarity(a, b string) float64 {
	var ws levenshteinWorkspace
	return editSimilarityRunes([]rune(text.NormKey(a)), []rune(text.NormKey(b)), &ws)
}

// editSimilarityRunes is the prepared-key form: the inputs are already
// normalized and the DP rows come from the caller's workspace, so a
// whole-catalog pass allocates nothing per comparison.
func editSimilarityRunes(a, b []rune, ws *levenshteinWorkspace) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	if runesEqual(a, b) {
		return 1
	}
	dist := ws.distance(a, b)
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	return 1 - float64(dist)/float64(maxLen)
}

func runesEqual(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// levenshteinWorkspace reuses the two DP rows across comparisons, so a
// whole-catalog edit pass allocates nothing per entry.
type levenshteinWorkspace struct {
	prev []int
	cur  []int
}

func (w *levenshteinWorkspace) distance(a, b []rune) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	// Distance is symmetric; keep the inner dimension on the shorter
	// string so both buffers are as small as possible.
	if len(a) < len(b) {
		a, b = b, a
	}
	n := len(b) + 1
	if cap(w.prev) < n {
		w.prev = make([]int, n)
	} else {
		w.prev = w.prev[:n]
	}
	if cap(w.cur) < n {
		w.cur = make([]int, n)
	} else {
		w.cur = w.cur[:n]
	}
	prev, cur := w.prev, w.cur
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			cur[j] = min3(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	w.prev, w.cur = prev, cur
	return prev[len(b)]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// rankNamesEdit picks the best plain-name candidate for raw text by
// edit similarity. Same strict-winner + threshold contract as
// rankNames; single candidates must still earn minEditScore (unlike
// containment singles, sheer existence proves nothing here).
func rankNamesEdit(raw string, names []string) (string, bool) {
	if len(strings.TrimSpace(raw)) < MinMatchLen || len(names) == 0 {
		return "", false
	}
	// Compare identity words, not raw strings: legal suffixes
	// ("Co., Ltd") would otherwise drown out a one-letter typo.
	keys := make([][]rune, len(names))
	for i, n := range names {
		keys[i] = []rune(contentKey(n))
	}
	var ws levenshteinWorkspace
	return rankNamesEditKeys(raw, []rune(contentKey(raw)), names, keys, &ws)
}

// rankNamesEditKeys is rankNamesEdit over precomputed content keys.
func rankNamesEditKeys(raw string, rawKey []rune, names []string, keys [][]rune, ws *levenshteinWorkspace) (string, bool) {
	if len(strings.TrimSpace(raw)) < MinMatchLen || len(rawKey) == 0 || len(names) == 0 {
		return "", false
	}
	bestScore, runnerUp := 0.0, 0.0
	best := ""
	for i, n := range names {
		if s := editSimilarityRunes(rawKey, keys[i], ws); s > bestScore {
			runnerUp, bestScore, best = bestScore, s, n
		} else if s > runnerUp {
			runnerUp = s
		}
	}
	if bestScore < minEditScore || bestScore <= runnerUp {
		return "", false
	}
	return best, true
}

// rankNames picks the best plain-name candidate for raw text by token
// overlap (brands, manufacturers, models — single names, no aliases).
// Winners must strictly beat the runner-up and be convincing (score
// bar plus full raw coverage, two shared words, or one name fully
// explained by the other); lone candidates are scored, never
// auto-trusted.
func rankNames(raw string, names []string) (string, bool) {
	rawTokens := contentTokens(raw)
	if len(rawTokens) == 0 || len(names) == 0 {
		return "", false
	}
	tokens, _ := prepareNames(names)
	return rankNamesTokens(rawTokens, names, tokens)
}

// rankNamesTokens is rankNames over precomputed token sets.
func rankNamesTokens(rawTokens []string, names []string, tokenSets [][]string) (string, bool) {
	return rankNamesTokensCore(rawTokens, names, tokenSets, nil, true)
}

// rankNamesTokensShortlist is the indexed variant: it scores only the
// names whose token set intersects the query (all others score zero).
func rankNamesTokensShortlist(rawTokens []string, names []string, tokenSets [][]string, indices []int) (string, bool) {
	return rankNamesTokensCore(rawTokens, names, tokenSets, indices, false)
}

func rankNamesTokensCore(rawTokens []string, names []string, tokenSets [][]string, indices []int, considerAll bool) (string, bool) {
	if len(rawTokens) == 0 || len(names) == 0 {
		return "", false
	}
	var best matchScore
	var bestTokens []string
	bestName := ""
	runnerUp := 0.0
	consider := func(i int) {
		score := matchQuality(rawTokens, tokenSets[i])
		if score.f1 > best.f1 {
			runnerUp, best, bestName, bestTokens = best.f1, score, names[i], tokenSets[i]
		} else if score.f1 > runnerUp {
			runnerUp = score.f1
		}
	}
	if considerAll {
		for i := range names {
			consider(i)
		}
	} else {
		for _, i := range indices {
			consider(i)
		}
	}
	if best.f1 <= runnerUp || !best.convincing() {
		return "", false
	}
	if !sharedDistinctive(rawTokens, bestTokens) {
		return "", false
	}
	return bestName, true
}

// prepareNames tokenizes and content-keys a plain catalog list once.
func prepareNames(names []string) (tokens [][]string, keys [][]rune) {
	tokens = make([][]string, len(names))
	keys = make([][]rune, len(names))
	for i, n := range names {
		tokens[i] = contentTokens(n)
		keys[i] = []rune(contentKey(n))
	}
	return tokens, keys
}

// tokenIndex maps a content token to the catalog entries containing it,
// so whole-catalog matching only scores entries sharing a word with the
// query. Buffer reuse assumes the matcher stays single-threaded.
type tokenIndex struct {
	postings map[string][]int
	seen     []int
	buf      []int
	gen      int
}

func newNameTokenIndex(tokenSets [][]string) *tokenIndex {
	idx := &tokenIndex{postings: make(map[string][]int, 256)}
	for i, tokens := range tokenSets {
		addPostings(idx, i, tokens)
	}
	return idx
}

func newDeviceTokenIndex(aliasTokens [][][]string) *tokenIndex {
	idx := &tokenIndex{postings: make(map[string][]int, 256)}
	for i, sets := range aliasTokens {
		var all []string
		for _, tokens := range sets {
			all = append(all, tokens...)
		}
		addPostings(idx, i, all)
	}
	return idx
}

// addPostings appends i under each distinct token.
func addPostings(idx *tokenIndex, i int, tokens []string) {
	for j, t := range tokens {
		if containsToken(tokens[:j], t) {
			continue
		}
		idx.postings[t] = append(idx.postings[t], i)
	}
}

// shortlist returns the indices of entries sharing at least one token
// with rawTokens. The returned slice is reused; callers must consume it
// before the next call.
func (x *tokenIndex) shortlist(n int, rawTokens []string) []int {
	if x == nil || len(x.postings) == 0 || len(rawTokens) == 0 {
		return nil
	}
	x.gen++
	if x.gen == 0 {
		for i := range x.seen {
			x.seen[i] = 0
		}
		x.gen = 1
	}
	if len(x.seen) != n {
		x.seen = make([]int, n)
	}
	out := x.buf[:0]
	for _, t := range rawTokens {
		for _, i := range x.postings[t] {
			if x.seen[i] == x.gen {
				continue
			}
			x.seen[i] = x.gen
			out = append(out, i)
		}
	}
	x.buf = out
	return out
}
