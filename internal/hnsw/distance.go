package hnsw

import (
	"errors"
	"math"
)

var (
	ErrDimensionMismatch = errors.New("hnsw: vector dimension mismatch")
	ErrEmptyVector        = errors.New("hnsw: empty vector")
)

type DistanceFunc func(a, b []float32) (float32, error)

// CosineDistance calculates cosine distance (1.0 - cosine similarity).
// Lower distance means higher similarity.
func CosineDistance(a, b []float32) (float32, error) {
	n := len(a)
	if n == 0 {
		return 0, ErrEmptyVector
	}
	if n != len(b) {
		return 0, ErrDimensionMismatch
	}

	var dot, normA, normB float32

	// Loop unrolling (8 elements per step) for compiler auto-vectorization
	i := 0
	for ; i+7 < n; i += 8 {
		dot += a[i]*b[i] + a[i+1]*b[i+1] + a[i+2]*b[i+2] + a[i+3]*b[i+3] +
			a[i+4]*b[i+4] + a[i+5]*b[i+5] + a[i+6]*b[i+6] + a[i+7]*b[i+7]

		normA += a[i]*a[i] + a[i+1]*a[i+1] + a[i+2]*a[i+2] + a[i+3]*a[i+3] +
			a[i+4]*a[i+4] + a[i+5]*a[i+5] + a[i+6]*a[i+6] + a[i+7]*a[i+7]

		normB += b[i]*b[i] + b[i+1]*b[i+1] + b[i+2]*b[i+2] + b[i+3]*b[i+3] +
			b[i+4]*b[i+4] + b[i+5]*b[i+5] + b[i+6]*b[i+6] + b[i+7]*b[i+7]
	}

	// Remainder
	for ; i < n; i++ {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 1.0, nil
	}

	similarity := dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
	// Clamp similarity between -1 and 1 due to floating point precision
	if similarity > 1.0 {
		similarity = 1.0
	} else if similarity < -1.0 {
		similarity = -1.0
	}

	return 1.0 - similarity, nil
}

// L2Distance computes squared Euclidean distance.
func L2Distance(a, b []float32) (float32, error) {
	n := len(a)
	if n == 0 {
		return 0, ErrEmptyVector
	}
	if n != len(b) {
		return 0, ErrDimensionMismatch
	}

	var sum float32
	i := 0
	for ; i+7 < n; i += 8 {
		d0 := a[i] - b[i]
		d1 := a[i+1] - b[i+1]
		d2 := a[i+2] - b[i+2]
		d3 := a[i+3] - b[i+3]
		d4 := a[i+4] - b[i+4]
		d5 := a[i+5] - b[i+5]
		d6 := a[i+6] - b[i+6]
		d7 := a[i+7] - b[i+7]

		sum += d0*d0 + d1*d1 + d2*d2 + d3*d3 + d4*d4 + d5*d5 + d6*d6 + d7*d7
	}

	for ; i < n; i++ {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return float32(math.Sqrt(float64(sum))), nil
}

// DotProductDistance computes negative dot product (for Maximum Inner Product Search).
func DotProductDistance(a, b []float32) (float32, error) {
	n := len(a)
	if n == 0 {
		return 0, ErrEmptyVector
	}
	if n != len(b) {
		return 0, ErrDimensionMismatch
	}

	var dot float32
	i := 0
	for ; i+7 < n; i += 8 {
		dot += a[i]*b[i] + a[i+1]*b[i+1] + a[i+2]*b[i+2] + a[i+3]*b[i+3] +
			a[i+4]*b[i+4] + a[i+5]*b[i+5] + a[i+6]*b[i+6] + a[i+7]*b[i+7]
	}

	for ; i < n; i++ {
		dot += a[i] * b[i]
	}

	return -dot, nil
}

// Normalize normalizes a vector to unit length (L2 norm = 1).
func Normalize(v []float32) []float32 {
	var norm float32
	for _, val := range v {
		norm += val * val
	}
	if norm == 0 {
		return v
	}
	invNorm := float32(1.0 / math.Sqrt(float64(norm)))
	res := make([]float32, len(v))
	for i, val := range v {
		res[i] = val * invNorm
	}
	return res
}
