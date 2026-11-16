package engine

import (
	"sort"
)

type RankItem struct {
	DocID       uint64  `json:"doc_id"`
	DenseRank   int     `json:"dense_rank"`
	SparseRank  int     `json:"sparse_rank"`
	DenseScore  float32 `json:"dense_score"`
	SparseScore float32 `json:"sparse_score"`
	FinalScore  float32 `json:"final_score"`
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
