package graph

type TraversalSubgraph struct {
	Nodes []*Node `json:"nodes"`
	Edges []Edge  `json:"edges"`
	Hops  map[string]int `json:"hops"`
}

// TraverseBFS traverses the knowledge graph starting from seed node IDs up to maxHops.
func (kg *KnowledgeGraph) TraverseBFS(startIDs []string, maxHops int, minWeight float32) TraversalSubgraph {
	kg.mu.RLock()
	defer kg.mu.RUnlock()

	visited := make(map[string]int) // nodeID -> hop level
	edgeSet := make(map[string]bool)
	var resultEdges []Edge

	type queueItem struct {
		id  string
		hop int
	}

	queue := make([]queueItem, 0)
	for _, id := range startIDs {
		if _, ok := kg.nodes[id]; ok {
			visited[id] = 0
			queue = append(queue, queueItem{id: id, hop: 0})
		}
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr.hop >= maxHops {
			continue
		}

		// Traverse outgoing edges
		for _, edge := range kg.outgoing[curr.id] {
			if edge.Weight < minWeight {
				continue
			}
			edgeKey := edge.From + "->" + edge.Relation + "->" + edge.To
			if !edgeSet[edgeKey] {
				edgeSet[edgeKey] = true
				resultEdges = append(resultEdges, edge)
			}

			if _, seen := visited[edge.To]; !seen {
				visited[edge.To] = curr.hop + 1
				queue = append(queue, queueItem{id: edge.To, hop: curr.hop + 1})
			}
		}

		// Traverse incoming edges (bidirectional relational discovery)
		for _, edge := range kg.incoming[curr.id] {
			if edge.Weight < minWeight {
				continue
			}
			edgeKey := edge.From + "->" + edge.Relation + "->" + edge.To
			if !edgeSet[edgeKey] {
				edgeSet[edgeKey] = true
				resultEdges = append(resultEdges, edge)
			}

			if _, seen := visited[edge.From]; !seen {
				visited[edge.From] = curr.hop + 1
				queue = append(queue, queueItem{id: edge.From, hop: curr.hop + 1})
			}
		}
	}

	resultNodes := make([]*Node, 0, len(visited))
	for id := range visited {
		if node, ok := kg.nodes[id]; ok {
			resultNodes = append(resultNodes, node)
		}
	}

	return TraversalSubgraph{
		Nodes: resultNodes,
		Edges: resultEdges,
		Hops:  visited,
	}
}
