package hnsw

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
)

var (
	ErrDuplicateID = errors.New("hnsw: document id already exists")
	ErrNodeNotFound = errors.New("hnsw: node not found")
)

type Config struct {
	M              int
	M0             int
	EfConstruction int
	EfSearch       int
	Dimension      int
	Metric         uint32
	Quantize       bool
}

func DefaultConfig(dim int) Config {
	return Config{
		M:              16,
		M0:             32,
		EfConstruction: 100,
		EfSearch:       64,
		Dimension:      dim,
		Metric:         0, // Cosine
		Quantize:       false,
	}
}

type Node struct {
	ID        uint64
	Vector    []float32
	SQ8       SQ8Vector
	Level     int
	Neighbors [][]uint64 // Neighbors[level] = []uint64
	mu        sync.RWMutex
}

type Graph struct {
	mu         sync.RWMutex
	config     Config
	mL         float64
	entryPoint *Node
	maxLevel   int
	nodes      map[uint64]*Node
	count      atomic.Int64
	distFunc   DistanceFunc
}

func NewGraph(cfg Config) *Graph {
	if cfg.M <= 0 {
		cfg.M = 16
	}
	if cfg.M0 <= 0 {
		cfg.M0 = 2 * cfg.M
	}
	if cfg.EfConstruction <= 0 {
		cfg.EfConstruction = 100
	}
	if cfg.EfSearch <= 0 {
		cfg.EfSearch = 64
	}

	dist := CosineDistance
	if cfg.Metric == 1 {
		dist = L2Distance
	} else if cfg.Metric == 2 {
		dist = DotProductDistance
	}

	return &Graph{
		config:   cfg,
		mL:       1.0 / math.Log(float64(cfg.M)),
		maxLevel: -1,
		nodes:    make(map[uint64]*Node),
		distFunc: dist,
	}
}

func (g *Graph) Len() int {
	return int(g.count.Load())
}

func (g *Graph) randomLevel() int {
	r := rand.Float64()
	if r == 0 {
		r = 0.0000001
	}
	lvl := int(-math.Log(r) * g.mL)
	return lvl
}

func (g *Graph) calculateDistance(a, b []float32) float32 {
	d, err := g.distFunc(a, b)
	if err != nil {
		return math.MaxFloat32
	}
	return d
}

func (g *Graph) nodeDistance(query []float32, node *Node) float32 {
	if g.config.Quantize && len(node.SQ8.Data) > 0 {
		d, err := node.SQ8.AsymmetricCosineDistance(query)
		if err == nil {
			return d
		}
	}
	return g.calculateDistance(query, node.Vector)
}

// Insert adds a vector with unique id into the HNSW graph.
func (g *Graph) Insert(id uint64, vec []float32) error {
	if len(vec) != g.config.Dimension {
		return ErrDimensionMismatch
	}

	g.mu.Lock()
	if _, exists := g.nodes[id]; exists {
		g.mu.Unlock()
		return ErrDuplicateID
	}

	level := g.randomLevel()
	node := &Node{
		ID:        id,
		Vector:    make([]float32, len(vec)),
		Level:     level,
		Neighbors: make([][]uint64, level+1),
	}
	copy(node.Vector, vec)
	if g.config.Quantize {
		node.SQ8 = QuantizeSQ8(vec)
	}

	currObj := g.entryPoint
	maxLvl := g.maxLevel

	if currObj == nil {
		g.entryPoint = node
		g.maxLevel = level
		g.nodes[id] = node
		g.count.Add(1)
		g.mu.Unlock()
		return nil
	}

	g.nodes[id] = node
	g.count.Add(1)
	g.mu.Unlock()

	// 1. Greedy search from top down to level+1
	currDist := g.nodeDistance(vec, currObj)
	for lc := maxLvl; lc > level; lc-- {
		changed := true
		for changed {
			changed = false
			currObj.mu.RLock()
			var neighbors []uint64
			if lc < len(currObj.Neighbors) {
				neighbors = append(neighbors, currObj.Neighbors[lc]...)
			}
			currObj.mu.RUnlock()

			for _, neighborID := range neighbors {
				g.mu.RLock()
				neighbor := g.nodes[neighborID]
				g.mu.RUnlock()
				if neighbor == nil {
					continue
				}

				d := g.nodeDistance(vec, neighbor)
				if d < currDist {
					currDist = d
					currObj = neighbor
					changed = true
				}
			}
		}
	}

	// 2. Search and connect from min(maxLvl, level) down to 0
	topLevel := level
	if maxLvl < topLevel {
		topLevel = maxLvl
	}

	entryPoints := []Item{{ID: currObj.ID, Distance: currDist}}
	for lc := topLevel; lc >= 0; lc-- {
		candidates := g.searchLayer(vec, entryPoints, g.config.EfConstruction, lc)

		m := g.config.M
		if lc == 0 {
			m = g.config.M0
		}

		selectedNeighbors := g.selectNeighbors(candidates, m)
		node.Neighbors[lc] = selectedNeighbors

		// Connect bidirectional edges
		for _, neighborID := range selectedNeighbors {
			g.mu.RLock()
			neighbor := g.nodes[neighborID]
			g.mu.RUnlock()
			if neighbor == nil {
				continue
			}

			neighbor.mu.Lock()
			if lc < len(neighbor.Neighbors) {
				neighbor.Neighbors[lc] = append(neighbor.Neighbors[lc], node.ID)
				if len(neighbor.Neighbors[lc]) > m {
					neighbor.Neighbors[lc] = g.shrinkNeighbors(neighbor, neighbor.Neighbors[lc], m)
				}
			}
			neighbor.mu.Unlock()
		}

		entryPoints = candidates
	}

	g.mu.Lock()
	if level > g.maxLevel {
		g.maxLevel = level
		g.entryPoint = node
	}
	g.mu.Unlock()

	return nil
}

func (g *Graph) selectNeighbors(candidates []Item, m int) []uint64 {
	limit := m
	if len(candidates) < limit {
		limit = len(candidates)
	}
	res := make([]uint64, limit)
	for i := 0; i < limit; i++ {
		res[i] = candidates[i].ID
	}
	return res
}

func (g *Graph) shrinkNeighbors(node *Node, neighbors []uint64, m int) []uint64 {
	type distNode struct {
		id   uint64
		dist float32
	}
	dn := make([]distNode, 0, len(neighbors))
	for _, nid := range neighbors {
		g.mu.RLock()
		nbr := g.nodes[nid]
		g.mu.RUnlock()
		if nbr == nil {
			continue
		}
		d := g.nodeDistance(node.Vector, nbr)
		dn = append(dn, distNode{id: nid, dist: d})
	}

	// Simple selection of m closest
	for i := 0; i < len(dn); i++ {
		for j := i + 1; j < len(dn); j++ {
			if dn[j].dist < dn[i].dist {
				dn[i], dn[j] = dn[j], dn[i]
			}
		}
	}

	if len(dn) > m {
		dn = dn[:m]
	}

	res := make([]uint64, len(dn))
	for i := range dn {
		res[i] = dn[i].id
	}
	return res
}

// Serialize writes the graph to a writer.
func (g *Graph) Serialize(w io.Writer) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Write metadata
	if err := binary.Write(w, binary.LittleEndian, int32(g.config.Dimension)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, int32(g.config.M)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, int32(g.config.M0)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, int32(g.maxLevel)); err != nil {
		return err
	}

	var entryID uint64
	if g.entryPoint != nil {
		entryID = g.entryPoint.ID
	}
	if err := binary.Write(w, binary.LittleEndian, entryID); err != nil {
		return err
	}

	count := uint32(len(g.nodes))
	if err := binary.Write(w, binary.LittleEndian, count); err != nil {
		return err
	}

	// Write each node
	for _, n := range g.nodes {
		if err := binary.Write(w, binary.LittleEndian, n.ID); err != nil {
			return err
		}
		if err := binary.Write(w, binary.LittleEndian, int32(n.Level)); err != nil {
			return err
		}
		// Write vector
		for _, f := range n.Vector {
			if err := binary.Write(w, binary.LittleEndian, f); err != nil {
				return err
			}
		}
		// Write neighbors per level
		for l := 0; l <= n.Level; l++ {
			n.mu.RLock()
			nbrs := n.Neighbors[l]
			n.mu.RUnlock()

			nbrCount := uint32(len(nbrs))
			if err := binary.Write(w, binary.LittleEndian, nbrCount); err != nil {
				return err
			}
			for _, nid := range nbrs {
				if err := binary.Write(w, binary.LittleEndian, nid); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// Deserialize loads the graph from a reader.
func (g *Graph) Deserialize(r io.Reader) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	var dim, m, m0, maxLvl int32
	var entryID uint64
	var count uint32

	if err := binary.Read(r, binary.LittleEndian, &dim); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &m); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &m0); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &maxLvl); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &entryID); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
		return err
	}

	g.config.Dimension = int(dim)
	g.config.M = int(m)
	g.config.M0 = int(m0)
	g.maxLevel = int(maxLvl)
	g.nodes = make(map[uint64]*Node, count)

	for i := uint32(0); i < count; i++ {
		var id uint64
		var lvl int32
		if err := binary.Read(r, binary.LittleEndian, &id); err != nil {
			return err
		}
		if err := binary.Read(r, binary.LittleEndian, &lvl); err != nil {
			return err
		}

		vec := make([]float32, dim)
		for d := 0; d < int(dim); d++ {
			if err := binary.Read(r, binary.LittleEndian, &vec[d]); err != nil {
				return err
			}
		}

		node := &Node{
			ID:        id,
			Vector:    vec,
			Level:     int(lvl),
			Neighbors: make([][]uint64, int(lvl)+1),
		}
		if g.config.Quantize {
			node.SQ8 = QuantizeSQ8(vec)
		}

		for l := 0; l <= int(lvl); l++ {
			var nbrCount uint32
			if err := binary.Read(r, binary.LittleEndian, &nbrCount); err != nil {
				return err
			}
			nbrs := make([]uint64, nbrCount)
			for j := uint32(0); j < nbrCount; j++ {
				if err := binary.Read(r, binary.LittleEndian, &nbrs[j]); err != nil {
					return err
				}
			}
			node.Neighbors[l] = nbrs
		}

		g.nodes[id] = node
	}

	g.entryPoint = g.nodes[entryID]
	g.count.Store(int64(count))
	return nil
}
