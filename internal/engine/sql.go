package engine

import (
	"errors"
	"strconv"
	"strings"

	"github.com/alexandrmotologa/embdb/internal/metadata"
)

type OrderMode int

const (
	OrderModeNone OrderMode = iota
	OrderModeHybrid
	OrderModeVector
	OrderModeBM25
	OrderModeField
)

type OrderClause struct {
	Mode      OrderMode
	QueryText string
	Field     string
	Desc      bool
}

type ParsedQuery struct {
	Fields     []string
	Collection string
	Where      *metadata.Predicate
	Order      OrderClause
	Limit      int
	Offset     int
}

var (
	ErrInvalidSQL = errors.New("sql: invalid or unsupported sql query")
)

// ParseSQL parses a lightweight SQL statement for semantic and hybrid search.
func ParseSQL(query string) (*ParsedQuery, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, ErrInvalidSQL
	}

	upper := strings.ToUpper(q)
	if !strings.HasPrefix(upper, "SELECT") {
		return nil, errors.New("sql: query must start with SELECT")
	}

	fromIdx := strings.Index(upper, " FROM ")
	if fromIdx == -1 {
		return nil, errors.New("sql: missing FROM clause")
	}

	fieldsPart := strings.TrimSpace(q[6:fromIdx])
	fields := parseFields(fieldsPart)

	remainder := strings.TrimSpace(q[fromIdx+6:])
	parsed := &ParsedQuery{
		Fields: fields,
		Limit:  10,
	}

	// Tokenize remaining clauses: WHERE, ORDER BY, LIMIT, OFFSET
	remUpper := strings.ToUpper(remainder)

	whereIdx := strings.Index(remUpper, " WHERE ")
	orderIdx := strings.Index(remUpper, " ORDER BY ")
	limitIdx := strings.Index(remUpper, " LIMIT ")
	offsetIdx := strings.Index(remUpper, " OFFSET ")

	// Find end of collection name
	endColl := len(remainder)
	for _, idx := range []int{whereIdx, orderIdx, limitIdx, offsetIdx} {
		if idx != -1 && idx < endColl {
			endColl = idx
		}
	}
	parsed.Collection = strings.TrimSpace(remainder[:endColl])

	// Parse WHERE
	if whereIdx != -1 {
		startWhere := whereIdx + 7
		endWhere := len(remainder)
		for _, idx := range []int{orderIdx, limitIdx, offsetIdx} {
			if idx > whereIdx && idx < endWhere {
				endWhere = idx
			}
		}
		whereStr := strings.TrimSpace(remainder[startWhere:endWhere])
		pred, err := parseWhereClause(whereStr)
		if err != nil {
			return nil, err
		}
		parsed.Where = pred
	}

	// Parse ORDER BY
	if orderIdx != -1 {
		startOrder := orderIdx + 10
		endOrder := len(remainder)
		for _, idx := range []int{limitIdx, offsetIdx} {
			if idx > orderIdx && idx < endOrder {
				endOrder = idx
			}
		}
		orderStr := strings.TrimSpace(remainder[startOrder:endOrder])
		parsed.Order = parseOrderClause(orderStr)
	}

	// Parse LIMIT
	if limitIdx != -1 {
		startLimit := limitIdx + 7
		endLimit := len(remainder)
		if offsetIdx > limitIdx && offsetIdx < endLimit {
			endLimit = offsetIdx
		}
		limitStr := strings.TrimSpace(remainder[startLimit:endLimit])
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			parsed.Limit = l
		}
	}

	// Parse OFFSET
	if offsetIdx != -1 {
		startOffset := offsetIdx + 8
		offsetStr := strings.TrimSpace(remainder[startOffset:])
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			parsed.Offset = o
		}
	}

	return parsed, nil
}

func parseFields(fieldsStr string) []string {
	if fieldsStr == "*" {
		return []string{"*"}
	}
	parts := strings.Split(fieldsStr, ",")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		clean := strings.TrimSpace(p)
		if clean != "" {
			res = append(res, clean)
		}
	}
	return res
}

func parseWhereClause(whereStr string) (*metadata.Predicate, error) {
	// Support simple conditions: field = 'val', field > 10, field = true, joined with AND
	parts := strings.Split(whereStr, " AND ")
	var preds []metadata.Predicate

	for _, part := range parts {
		cond := strings.TrimSpace(part)
		if cond == "" {
			continue
		}

		var field, opStr, valStr string
		for _, op := range []string{">=", "<=", "!=", "=", ">", "<"} {
			if idx := strings.Index(cond, op); idx != -1 {
				field = strings.TrimSpace(cond[:idx])
				opStr = op
				valStr = strings.TrimSpace(cond[idx+len(op):])
				break
			}
		}

		if field == "" {
			continue
		}

		val := parseValue(valStr)
		var op metadata.Operator
		switch opStr {
		case "=":
			op = metadata.OpEq
		case "!=":
			op = metadata.OpNe
		case ">":
			op = metadata.OpGt
		case ">=":
			op = metadata.OpGte
		case "<":
			op = metadata.OpLt
		case "<=":
			op = metadata.OpLte
		default:
			op = metadata.OpEq
		}

		preds = append(preds, metadata.Predicate{
			Field: field,
			Op:    op,
			Value: val,
		})
	}

	if len(preds) == 0 {
		return nil, nil
	}
	if len(preds) == 1 {
		return &preds[0], nil
	}
	combined := metadata.And(preds...)
	return &combined, nil
}

func parseValue(valStr string) any {
	clean := strings.Trim(valStr, "'\"")
	if strings.EqualFold(clean, "true") {
		return true
	}
	if strings.EqualFold(clean, "false") {
		return false
	}
	if n, err := strconv.ParseFloat(clean, 64); err == nil {
		return n
	}
	return clean
}

func parseOrderClause(orderStr string) OrderClause {
	orderUpper := strings.ToUpper(orderStr)
	desc := strings.HasSuffix(orderUpper, " DESC")
	clean := strings.TrimSuffix(strings.TrimSuffix(orderStr, " DESC"), " ASC")
	clean = strings.TrimSpace(clean)

	lower := strings.ToLower(clean)
	if strings.HasPrefix(lower, "hybrid(") && strings.HasSuffix(lower, ")") {
		queryArg := clean[7 : len(clean)-1]
		queryArg = strings.Trim(queryArg, "'\"")
		return OrderClause{
			Mode:      OrderModeHybrid,
			QueryText: queryArg,
			Desc:      desc,
		}
	}

	if strings.HasPrefix(lower, "bm25(") && strings.HasSuffix(lower, ")") {
		queryArg := clean[5 : len(clean)-1]
		queryArg = strings.Trim(queryArg, "'\"")
		return OrderClause{
			Mode:      OrderModeBM25,
			QueryText: queryArg,
			Desc:      desc,
		}
	}

	if lower == "vector" {
		return OrderClause{
			Mode: OrderModeVector,
			Desc: desc,
		}
	}

	return OrderClause{
		Mode:  OrderModeField,
		Field: clean,
		Desc:  desc,
	}
}
