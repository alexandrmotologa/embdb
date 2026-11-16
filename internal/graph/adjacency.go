package graph

import (
	"encoding/json"
	"io"
	"sync"
)

type Node struct {
	ID         string         `json:"id"`
	DocID      uint64         `json:"doc_id,omitempty"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties,omitempty"`
}

type Edge struct {
	From       string         `json:"from"`
	To         string         `json:"to"`
	Relation   string         `json:"relation"`
	Weight     float32        `json:"weight"`
	Properties map[string]any `json:"properties,omitempty"`
}

type KnowledgeGraph struct {
	mu       sync.RWMutex
	nodes    map[string]*Node
	outgoing map[string][]Edge
	incoming map[string][]Edge
}

func NewKnowledgeGraph() *KnowledgeGraph {
	return &KnowledgeGraph{
		nodes:    make(map[string]*Node),
		outgoing: make(map[string][]Edge),
		incoming: make(map[string][]Edge),
	}
}

func (kg *KnowledgeGraph) AddNode(node Node) {
	kg.mu.Lock()
	defer kg.mu.Unlock()
	kg.nodes[node.ID] = &node
}

func (kg *KnowledgeGraph) GetNode(id string) (*Node, bool) {
	kg.mu.RLock()
	defer kg.mu.RUnlock()
	n, ok := kg.nodes[id]
	return n, ok
}

func (kg *KnowledgeGraph) AddEdge(edge Edge) {
	kg.mu.Lock()
	defer kg.mu.Unlock()

	if edge.Weight == 0 {
		edge.Weight = 1.0
	}

	kg.outgoing[edge.From] = append(kg.outgoing[edge.From], edge)
	kg.incoming[edge.To] = append(kg.incoming[edge.To], edge)
}

func (kg *KnowledgeGraph) OutgoingEdges(nodeID string) []Edge {
	kg.mu.RLock()
	defer kg.mu.RUnlock()
	return kg.outgoing[nodeID]
}

func (kg *KnowledgeGraph) IncomingEdges(nodeID string) []Edge {
	kg.mu.RLock()
	defer kg.mu.RUnlock()
	return kg.incoming[nodeID]
}

func (kg *KnowledgeGraph) NodeCount() int {
	kg.mu.RLock()
	defer kg.mu.RUnlock()
	return len(kg.nodes)
}

func (kg *KnowledgeGraph) EdgeCount() int {
	kg.mu.RLock()
	defer kg.mu.RUnlock()
	var total int
	for _, edges := range kg.outgoing {
		total += len(edges)
	}
	return total
}

func (kg *KnowledgeGraph) Nodes() []*Node {
	kg.mu.RLock()
	defer kg.mu.RUnlock()
	res := make([]*Node, 0, len(kg.nodes))
	for _, n := range kg.nodes {
		res = append(res, n)
	}
	return res
}

func (kg *KnowledgeGraph) Edges() []Edge {
	kg.mu.RLock()
	defer kg.mu.RUnlock()
	res := make([]Edge, 0)
	for _, edgeList := range kg.outgoing {
		res = append(res, edgeList...)
	}
	return res
}

// Serialize writes the graph as JSON stream.
func (kg *KnowledgeGraph) Serialize(w io.Writer) error {
	kg.mu.RLock()
	defer kg.mu.RUnlock()

	type serialGraph struct {
		Nodes []*Node `json:"nodes"`
		Edges []Edge  `json:"edges"`
	}

	sg := serialGraph{
		Nodes: make([]*Node, 0, len(kg.nodes)),
		Edges: make([]Edge, 0),
	}
	for _, n := range kg.nodes {
		sg.Nodes = append(sg.Nodes, n)
	}
	for _, elist := range kg.outgoing {
		sg.Edges = append(sg.Edges, elist...)
	}

	enc := json.NewEncoder(w)
	return enc.Encode(sg)
}

// Deserialize loads the graph from a JSON stream.
func (kg *KnowledgeGraph) Deserialize(r io.Reader) error {
	kg.mu.Lock()
	defer kg.mu.Unlock()

	type serialGraph struct {
		Nodes []*Node `json:"nodes"`
		Edges []Edge  `json:"edges"`
	}

	var sg serialGraph
	dec := json.NewDecoder(r)
	if err := dec.Decode(&sg); err != nil {
		return err
	}

	kg.nodes = make(map[string]*Node, len(sg.Nodes))
	kg.outgoing = make(map[string][]Edge)
	kg.incoming = make(map[string][]Edge)

	for _, n := range sg.Nodes {
		kg.nodes[n.ID] = n
	}
	for _, e := range sg.Edges {
		kg.outgoing[e.From] = append(kg.outgoing[e.From], e)
		kg.incoming[e.To] = append(kg.incoming[e.To], e)
	}

	return nil
}
