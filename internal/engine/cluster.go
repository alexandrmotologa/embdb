package engine

import (
	"math/rand"
	"sort"
	"strings"

	"github.com/alexandrmotologa/embdb/internal/bm25"
	"github.com/alexandrmotologa/embdb/internal/hnsw"
)

type Cluster struct {
	ID       int      `json:"id"`
	Label    string   `json:"label"`
	DocCount int      `json:"doc_count"`
	DocKeys  []string `json:"doc_keys"`
}

type ClusterResult struct {
	Clusters []Cluster `json:"clusters"`
	TotalDocs int      `json:"total_docs"`
}

// ComputeClusters runs spherical k-means on the stored vectors to discover semantic topics.
func (db *DB) ComputeClusters(k int, maxIterations int) ClusterResult {
	db.mu.RLock()
	defer db.mu.RUnlock()

	docs := make([]*Document, 0, len(db.docs))
	for _, d := range db.docs {
		if len(d.Vector) > 0 {
			docs = append(docs, d)
		}
	}

	if len(docs) == 0 {
		return ClusterResult{}
	}

	if k <= 0 {
		k = 3
	}
	if k > len(docs) {
		k = len(docs)
	}
	if maxIterations <= 0 {
		maxIterations = 15
	}

	dim := db.dim

	// 1. Initialize centroids randomly from existing documents
	centroids := make([][]float32, k)
	perm := rand.Perm(len(docs))
	for i := 0; i < k; i++ {
		centroids[i] = make([]float32, dim)
		copy(centroids[i], docs[perm[i]].Vector)
		centroids[i] = hnsw.Normalize(centroids[i])
	}

	assignments := make([]int, len(docs))

	// 2. Iterative optimization
	for iter := 0; iter < maxIterations; iter++ {
		changed := false

		// Assignment step: assign each doc to closest centroid
		for i, doc := range docs {
			bestCluster := 0
			bestSim := float32(-1.0)

			for c := 0; c < k; c++ {
				var dot float32
				for d := 0; d < dim; d++ {
					dot += doc.Vector[d] * centroids[c][d]
				}
				if dot > bestSim {
					bestSim = dot
					bestCluster = c
				}
			}

			if assignments[i] != bestCluster {
				assignments[i] = bestCluster
				changed = true
			}
		}

		if !changed {
			break
		}

		// Update step: recompute centroids
		counts := make([]int, k)
		newCentroids := make([][]float32, k)
		for c := 0; c < k; c++ {
			newCentroids[c] = make([]float32, dim)
		}

		for i, doc := range docs {
			c := assignments[i]
			counts[c]++
			for d := 0; d < dim; d++ {
				newCentroids[c][d] += doc.Vector[d]
			}
		}

		for c := 0; c < k; c++ {
			if counts[c] > 0 {
				centroids[c] = hnsw.Normalize(newCentroids[c])
			}
		}
	}

	// 3. Build cluster result and extract top term labels
	clusters := make([]Cluster, k)
	clusterTexts := make([][]string, k)

	for c := 0; c < k; c++ {
		clusters[c].ID = c + 1
	}

	for i, doc := range docs {
		c := assignments[i]
		clusters[c].DocCount++
		clusters[c].DocKeys = append(clusters[c].DocKeys, doc.Key)
		clusterTexts[c] = append(clusterTexts[c], doc.Text)
	}

	for c := 0; c < k; c++ {
		// Extract top frequent non-stop term across cluster texts as label
		termCounts := make(map[string]int)
		for _, text := range clusterTexts[c] {
			toks := bm25.Tokenize(text)
			for _, t := range toks {
				termCounts[t]++
			}
		}

		type termFreq struct {
			term string
			freq int
		}
		var tfList []termFreq
		for t, f := range termCounts {
			tfList = append(tfList, termFreq{term: t, freq: f})
		}
		sort.Slice(tfList, func(i, j int) bool {
			return tfList[i].freq > tfList[j].freq
		})

		var topTerms []string
		for i := 0; i < len(tfList) && i < 2; i++ {
			t := tfList[i].term
			if len(t) > 0 {
				topTerms = append(topTerms, strings.ToUpper(t[:1])+t[1:])
			}
		}
		if len(topTerms) > 0 {
			clusters[c].Label = strings.Join(topTerms, " & ")
		} else {
			clusters[c].Label = "Cluster " + string(rune('A'+c))
		}
	}

	return ClusterResult{
		Clusters:  clusters,
		TotalDocs: len(docs),
	}
}
