package metadata

import (
	"fmt"
	"reflect"
)

type Operator string

const (
	OpEq       Operator = "$eq"
	OpNe       Operator = "$ne"
	OpGt       Operator = "$gt"
	OpGte      Operator = "$gte"
	OpLt       Operator = "$lt"
	OpLte      Operator = "$lte"
	OpIn       Operator = "$in"
	OpNin      Operator = "$nin"
	OpContains Operator = "$contains"

	OpAnd Operator = "$and"
	OpOr  Operator = "$or"
	OpNot Operator = "$not"
)

// Predicate represents a single condition or composite logical expression.
type Predicate struct {
	Field    string
	Op       Operator
	Value    any
	Children []Predicate
}

// And combines multiple predicates with logical AND.
func And(preds ...Predicate) Predicate {
	return Predicate{
		Op:       OpAnd,
		Children: preds,
	}
}

// Or combines multiple predicates with logical OR.
func Or(preds ...Predicate) Predicate {
	return Predicate{
		Op:       OpOr,
		Children: preds,
	}
}

// Not inverts a predicate with logical NOT.
func Not(pred Predicate) Predicate {
	return Predicate{
		Op:       OpNot,
		Children: []Predicate{pred},
	}
}

// Eq creates an equality condition.
func Eq(field string, val any) Predicate {
	return Predicate{Field: field, Op: OpEq, Value: val}
}

// Gt creates a greater-than condition.
func Gt(field string, val any) Predicate {
	return Predicate{Field: field, Op: OpGt, Value: val}
}

// Lt creates a less-than condition.
func Lt(field string, val any) Predicate {
	return Predicate{Field: field, Op: OpLt, Value: val}
}

// Evaluate checks if the document metadata satisfies the predicate.
func (p Predicate) Evaluate(meta map[string]any) bool {
	switch p.Op {
	case OpAnd:
		for _, child := range p.Children {
			if !child.Evaluate(meta) {
				return false
			}
		}
		return true

	case OpOr:
		for _, child := range p.Children {
			if child.Evaluate(meta) {
				return true
			}
		}
		return false

	case OpNot:
		if len(p.Children) == 0 {
			return true
		}
		return !p.Children[0].Evaluate(meta)

	case OpEq:
		val, exists := meta[p.Field]
		if !exists {
			return p.Value == nil
		}
		return reflect.DeepEqual(normalizeVal(val), normalizeVal(p.Value))

	case OpNe:
		val, exists := meta[p.Field]
		if !exists {
			return p.Value != nil
		}
		return !reflect.DeepEqual(normalizeVal(val), normalizeVal(p.Value))

	case OpGt, OpGte, OpLt, OpLte:
		val, exists := meta[p.Field]
		if !exists {
			return false
		}
		cmp, ok := compareValues(val, p.Value)
		if !ok {
			return false
		}
		switch p.Op {
		case OpGt:
			return cmp > 0
		case OpGte:
			return cmp >= 0
		case OpLt:
			return cmp < 0
		case OpLte:
			return cmp <= 0
		}

	case OpIn:
		val, exists := meta[p.Field]
		if !exists {
			return false
		}
		normVal := normalizeVal(val)
		targetSlice, ok := toInterfaceSlice(p.Value)
		if !ok {
			return false
		}
		for _, item := range targetSlice {
			if reflect.DeepEqual(normVal, normalizeVal(item)) {
				return true
			}
		}
		return false

	case OpContains:
		val, exists := meta[p.Field]
		if !exists {
			return false
		}
		valSlice, ok := toInterfaceSlice(val)
		if !ok {
			return false
		}
		normTarget := normalizeVal(p.Value)
		for _, item := range valSlice {
			if reflect.DeepEqual(normalizeVal(item), normTarget) {
				return true
			}
		}
		return false
	}

	return false
}

func normalizeVal(v any) any {
	if v == nil {
		return nil
	}
	switch n := v.(type) {
	case int:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case float32:
		return float64(n)
	case float64:
		return n
	default:
		return v
	}
}

func compareValues(a, b any) (int, bool) {
	normA := normalizeVal(a)
	normB := normalizeVal(b)

	switch valA := normA.(type) {
	case float64:
		valB, ok := normB.(float64)
		if !ok {
			return 0, false
		}
		if valA < valB {
			return -1, true
		} else if valA > valB {
			return 1, true
		}
		return 0, true

	case string:
		valB, ok := normB.(string)
		if !ok {
			return 0, false
		}
		if valA < valB {
			return -1, true
		} else if valA > valB {
			return 1, true
		}
		return 0, true
	}

	return 0, false
}

func toInterfaceSlice(slice any) ([]any, bool) {
	if slice == nil {
		return nil, false
	}
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice && s.Kind() != reflect.Array {
		return nil, false
	}
	res := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		res[i] = s.Index(i).Interface()
	}
	return res, true
}

func StringifyPredicate(p Predicate) string {
	if p.Field != "" {
		return fmt.Sprintf("%s %s %v", p.Field, p.Op, p.Value)
	}
	return fmt.Sprintf("%s (%d children)", p.Op, len(p.Children))
}
