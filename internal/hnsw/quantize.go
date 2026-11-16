package hnsw

import (
	"math"
)

// SQ8Vector represents a vector quantized to 8-bit unsigned integers.
type SQ8Vector struct {
	Min  float32
	Max  float32
	Data []uint8
}

// QuantizeSQ8 compresses a float32 vector into an 8-bit SQ8Vector.
func QuantizeSQ8(v []float32) SQ8Vector {
	n := len(v)
	if n == 0 {
		return SQ8Vector{}
	}

	minVal := v[0]
	maxVal := v[0]
	for _, val := range v[1:] {
		if val < minVal {
			minVal = val
		}
		if val > maxVal {
			maxVal = val
		}
	}

	data := make([]uint8, n)
	diff := maxVal - minVal

	if diff == 0 {
		return SQ8Vector{
			Min:  minVal,
			Max:  maxVal,
			Data: data, // all zeros
		}
	}

	scale := 255.0 / diff
	for i, val := range v {
		q := float32(math.Round(float64((val - minVal) * scale)))
		if q < 0 {
			q = 0
		} else if q > 255 {
			q = 255
		}
		data[i] = uint8(q)
	}

	return SQ8Vector{
		Min:  minVal,
		Max:  maxVal,
		Data: data,
	}
}

// Dequantize reconstructs the float32 vector approximation.
func (q *SQ8Vector) Dequantize() []float32 {
	n := len(q.Data)
	res := make([]float32, n)
	diff := q.Max - q.Min
	if diff == 0 {
		for i := range res {
			res[i] = q.Min
		}
		return res
	}

	invScale := diff / 255.0
	for i, b := range q.Data {
		res[i] = q.Min + float32(b)*invScale
	}
	return res
}

// AsymmetricCosineDistance computes cosine distance between float32 query and SQ8 document.
func (q *SQ8Vector) AsymmetricCosineDistance(query []float32) (float32, error) {
	n := len(query)
	if n != len(q.Data) {
		return 0, ErrDimensionMismatch
	}

	diff := q.Max - q.Min
	invScale := diff / 255.0

	var dot, normQ, normD float32
	for i := 0; i < n; i++ {
		valD := q.Min + float32(q.Data[i])*invScale
		dot += query[i] * valD
		normQ += query[i] * query[i]
		normD += valD * valD
	}

	if normQ == 0 || normD == 0 {
		return 1.0, nil
	}

	sim := dot / (float32(math.Sqrt(float64(normQ))) * float32(math.Sqrt(float64(normD))))
	if sim > 1.0 {
		sim = 1.0
	} else if sim < -1.0 {
		sim = -1.0
	}
	return 1.0 - sim, nil
}
