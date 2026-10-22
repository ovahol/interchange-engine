package match

// This file is the public surface of the engine. The implementation stays
// unexported so the behaviour pinned by the tests cannot be bypassed.

// Workspace reuses Levenshtein DP buffers across comparisons, so a
// whole-catalog edit pass allocates nothing per entry.
type Workspace = levenshteinWorkspace

// TokenIndex maps content tokens to candidate indices. The returned
// shortlist is reused between calls; consume it before the next call.
type TokenIndex = tokenIndex

// Shortlist returns the indices of entries sharing at least one token with
// rawTokens.
func (x *tokenIndex) Shortlist(n int, rawTokens []string) []int {
	return x.shortlist(n, rawTokens)
}

// RankDevice picks the best catalog candidate for raw text. It returns
// ok=false unless the winner strictly beats the runner-up and is
// convincing: ties, weak matches, and fragments stay unresolved.
func RankDevice(raw string, candidates []Candidate) (Candidate, float64, bool) {
	return rankDeviceCandidates(raw, candidates)
}

// PrepareDeviceCandidates tokenizes every alias once and builds the flat
// name/key lists the edit fallback needs.
func PrepareDeviceCandidates(candidates []Candidate) (aliasTokens [][][]string, names []string, keys [][]rune) {
	return prepareDeviceCandidates(candidates)
}

// RankDeviceShortlist ranks only the candidates whose token set intersects
// the query.
func RankDeviceShortlist(rawTokens []string, candidates []Candidate, aliasTokens [][][]string, indices []int) (Candidate, float64, bool) {
	return rankDeviceCandidatesShortlist(rawTokens, candidates, aliasTokens, indices)
}

// RankName picks the best plain-name candidate by token overlap (brands,
// manufacturers, models). Single candidates are scored, never auto-trusted.
func RankName(raw string, names []string) (string, bool) {
	return rankNames(raw, names)
}

// PrepareNames tokenizes and content-keys a plain catalog list once.
func PrepareNames(names []string) (tokens [][]string, keys [][]rune) {
	return prepareNames(names)
}

// NewNameTokenIndex builds a token index over already-prepared name tokens.
func NewNameTokenIndex(tokenSets [][]string) *TokenIndex {
	return newNameTokenIndex(tokenSets)
}

// RankNameShortlist ranks only the names whose token set intersects the
// query.
func RankNameShortlist(rawTokens []string, names []string, tokenSets [][]string, indices []int) (string, bool) {
	return rankNamesTokensShortlist(rawTokens, names, tokenSets, indices)
}

// RankNameEdit picks the best plain-name candidate by edit similarity.
func RankNameEdit(raw string, names []string) (string, bool) {
	return rankNamesEdit(raw, names)
}

// RankNameEditKeys is RankNameEdit over precomputed content keys.
func RankNameEditKeys(raw string, rawKey []rune, names []string, keys [][]rune, ws *Workspace) (string, bool) {
	return rankNamesEditKeys(raw, rawKey, names, keys, ws)
}

// NewDeviceTokenIndex builds a token index over prepared device alias sets.
func NewDeviceTokenIndex(aliasTokens [][][]string) *TokenIndex {
	return newDeviceTokenIndex(aliasTokens)
}

// ContentTokens returns a name's identity-carrying words.
func ContentTokens(s string) []string {
	return contentTokens(s)
}

// ContentKey joins a name's identity words for edit comparison.
func ContentKey(s string) string {
	return contentKey(s)
}

// StrongMatch reports whether raw and alias share at least one distinctive
// token and clear the score bar, i.e. a single candidate is worth trusting.
func StrongMatch(rawTokens, aliasTokens []string) bool {
	score := matchQuality(rawTokens, aliasTokens)
	return score.convincing() && sharedDistinctive(rawTokens, aliasTokens)
}
