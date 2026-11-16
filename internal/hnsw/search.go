package hnsw

import (
	"container/heap"
)

// searchLayer searches for nearest neighbors at a specific layer lc.
func (g *Graph) searchLayer(query []float32, entryPoints []Item, ef int, lc int) []Item {
	visited := make(map[uint64]bool)
	candidates := &MinHeap{}
	w := &MaxHeap{}
	heap.Init(candidates)
	heap.Init(w)

	for _, ep := range entryPoints {
		visited[ep.ID] = true
		heap.Push(candidates, ep)
		heap.Push(w, ep)
	}

	for candidates.Len() > 0 {
		curr := heap.Pop(candidates).(Item)
		furthest := w.Peek()

		if curr.Distance > furthest.Distance {
			break
		}

		g.mu.RLock()
		node := g.nodes[curr.ID]
		g.mu.RUnlock()
		if node == nil {
			continue
		}

		node.mu.RLock()
		var neighbors []uint64
		if lc < len(node.Neighbors) {
			neighbors = append(neighbors, node.Neighbors[lc]...)
		}
		node.mu.RUnlock()

		for _, nbrID := range neighbors {
			if visited[nbrID] {
				continue
			}
			visited[nbrID] = true

			g.mu.RLock()
			nbr := g.nodes[nbrID]
			g.mu.RUnlock()
			if nbr == nil {
				continue
			}

			d := g.nodeDistance(query, nbr)
			furthest = w.Peek()

			if d < furthest.Distance || w.Len() < ef {
				heap.Push(candidates, Item{ID: nbrID, Distance: d})
				heap.Push(w, Item{ID: nbrID, Distance: d})

				if w.Len() > ef {
					heap.Pop(w)
				}
			}
		}
	}

	res := make([]Item, w.Len())
	for i := len(res) - 1; i >= 0; i-- {
		res[i] = heap.Pop(w).(Item)
	}
	return res
}

// Search finds the k nearest neighbors to the query vector, applying an optional predicate filter.
func (g *Graph) Search(query []float32, k int, filter func(id uint64) bool) ([]Item, error) {
	if len(query) != g.config.Dimension {
		return nil, ErrDimensionMismatch
	}

	g.mu.RLock()
	currObj := g.entryPoint
	maxLvl := g.maxLevel
	g.mu.RUnlock()

	if currObj == nil {
		return nil, nil
	}

	currDist := g.nodeDistance(query, currObj)

	// 1. Greedy descent from top layer down to layer 1
	for lc := maxLvl; lc > 0; lc-- {
		changed := true
		for changed {
			changed = false
			currObj.mu.RLock()
			var neighbors []uint64
			if lc < len(currObj.Neighbors) {
				neighbors = append(neighbors, currObj.Neighbors[lc]...)
			}
			currObj.mu.RUnlock()

			for _, nbrID := range neighbors {
				g.mu.RLock()
				nbr := g.nodes[nbrID]
				g.mu.RUnlock()
				if nbr == nil {
					continue
				}

				d := g.nodeDistance(query, nbr)
				if d < currDist {
					currDist = d
					currObj = nbr
					changed = true
				}
			}
		}
	}

	// 2. Beam search at layer 0
	ef := g.config.EfSearch
	if ef < k {
		ef = k
	}
	if filter != nil {
		// When filtering, expand search horizon to avoid premature pruning
		ef = ef * 2
	}

	candidates := g.searchLayer(query, []Item{{ID: currObj.ID, Distance: currDist}}, ef, 0)

	// Filter and select top-k
	var results []Item
	for _, item := range candidates {
		if filter != nil && !filter(item.ID) {
			continue
		}
		results = append(results, item)
		if len(results) >= k {
			break
		}
	}

	return results, nil
}
