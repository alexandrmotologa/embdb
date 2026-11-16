package engine

import (
	"sort"
	"strings"
	"unicode"
)

type RankItem struct {
	DocID       uint64  `json:"doc_id"`
	DenseRank   int     `json:"dense_rank"`
	SparseRank  int     `json:"sparse_rank"`
	DenseScore  float32 `json:"dense_score"`
	SparseScore float32 `json:"sparse_score"`
	FinalScore  float32 `json:"final_score"`
}

// DetectQueryIntent dynamically adjusts dense and sparse weights based on query characteristics.
func DetectQueryIntent(query string) (float32, float32) {
	q := strings.TrimSpace(query)
	if q == "" {
		return 0.5, 0.5
	}

	tokens := strings.Fields(q)
	numTokens := len(tokens)

	var lexicalScore int
	var semanticScore int

	// Check for exact quotes
	if strings.Contains(q, "\"") || strings.Contains(q, "'") {
		lexicalScore += 3
	}

	// Check for code symbols, punctuation, or technical identifiers
	for _, tok := range tokens {
		if strings.ContainsAny(tok, "_-.:/=()[]{}#@$%&*<>") {
			lexicalScore += 2
		}
		hasDigit := false
		hasLetter := false
		for _, r := range tok {
			if unicode.IsDigit(r) {
				hasDigit = true
			}
			if unicode.IsLetter(r) {
				hasLetter = true
			}
		}
		if hasDigit && hasLetter { // alphanumeric token like AES256, IPv4, SHA256
			lexicalScore += 2
		}
	}

	// Check for natural language question words
	lower := strings.ToLower(q)
	questionWords := []string{"what", "how", "why", "when", "where", "who", "which", "explain", "describe", "difference", "comparison"}
	for _, qw := range questionWords {
		if strings.HasPrefix(lower, qw+" ") || strings.Contains(lower, " "+qw+" ") {
			semanticScore += 3
		}
	}

	// Sentence length heuristic
	if numTokens >= 6 {
		semanticScore += 2
	} else if numTokens <= 2 {
		lexicalScore += 1
	}

	if lexicalScore > semanticScore {
		diff := lexicalScore - semanticScore
		if diff >= 3 {
			return 0.2, 0.8
		}
		return 0.35, 0.65
	} else if semanticScore > lexicalScore {
		diff := semanticScore - lexicalScore
		if diff >= 3 {
			return 0.8, 0.2
		}
		return 0.65, 0.35
	}

	return 0.5, 0.5
}

// ReciprocalRankFusion combines dense and sparse rankings using RRF formula.
func ReciprocalRankFusion(
	denseResults []RankItem,
	sparseResults []RankItem,
	denseWeight, sparseWeight float32,
	rrfK int,
) []RankItem {
	if rrfK <= 0 {
		rrfK = 60
	}
	if denseWeight <= 0 && sparseWeight <= 0 {
		denseWeight = 0.5
		sparseWeight = 0.5
	}

	merged := make(map[uint64]*RankItem)

	for rank, item := range denseResults {
		entry := &RankItem{
			DocID:      item.DocID,
			DenseRank:  rank + 1,
			DenseScore: item.DenseScore,
		}
		merged[item.DocID] = entry
	}

	for rank, item := range sparseResults {
		entry, exists := merged[item.DocID]
		if !exists {
			entry = &RankItem{
				DocID: item.DocID,
			}
			merged[item.DocID] = entry
		}
		entry.SparseRank = rank + 1
		entry.SparseScore = item.SparseScore
	}

	results := make([]RankItem, 0, len(merged))
	for _, item := range merged {
		var score float32

		if item.DenseRank > 0 {
			score += denseWeight / float32(rrfK+item.DenseRank)
		}
		if item.SparseRank > 0 {
			score += sparseWeight / float32(rrfK+item.SparseRank)
		}

		item.FinalScore = score
		results = append(results, *item)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].FinalScore > results[j].FinalScore
	})

	return results
}
