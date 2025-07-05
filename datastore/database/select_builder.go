package database

import (
	"fmt"
	"github.com/lib/pq"
	"strings"

	"github.com/wk8/go-ordered-map/v2"
)

// SelectBuilder is an SQL `SELECT` statement builder.
// Made to be simple and intuitive, without the need
// for complex features like code generation and while
// reading as closely to raw SQL queries as possible
type SelectBuilder struct {
	table     string
	columns   []string
	where     *orderedmap.OrderedMap[string, string]
	lastWhere string
	params    []any
	limit     int
	offset    int
	err       error
}

// From specifies the table to select from
func (sb *SelectBuilder) From(table string) *SelectBuilder {
	sb.table = table
	return sb
}

// Where sets the key to be used
func (sb *SelectBuilder) Where(field string) *SelectBuilder {
	sb.lastWhere = field
	return sb
}

// And is an alias for `Where` to make constructing queries
// read more like raw SQL
func (sb *SelectBuilder) And(field string) *SelectBuilder {
	return sb.Where(field)
}

// Is adds a new `key=value` statement for the WHERE clause.
// Uses the last set key from either `Where` or `And`
func (sb *SelectBuilder) Is(value any) *SelectBuilder {
	sb.params = append(sb.params, value)
	sb.where.Set(sb.lastWhere, "=?") // * Placeholder value. ? Gets replaced with $N in sb.Build
	return sb
}

// IsAnyOf adds a new `key=ANY(value)` statement for the WHERE clause.
// Uses the last set key from either `Where` or `And`
// Please make sure to provide a slice!
func (sb *SelectBuilder) IsAnyOf(value any) *SelectBuilder {
	sb.params = append(sb.params, pq.Array(value))
	sb.where.Set(sb.lastWhere, "=ANY(?)") // * Placeholder value. ? Gets replaced with $N in sb.Build
	return sb
}

// ArrayContains adds a new `key ?? ARRAY[value]` statement for the WHERE clause.
// Uses the last set key from either `Where` or `And`
func (sb *SelectBuilder) ArrayContains(value any) *SelectBuilder {
	sb.params = append(sb.params, []any{value})
	sb.where.Set(sb.lastWhere, "&&?") // * Placeholder value. ? Gets replaced with $N in sb.Build
	return sb
}

// Limit sets the LIMIT clause
func (sb *SelectBuilder) Limit(limit int) *SelectBuilder {
	sb.limit = limit
	return sb
}

// Offset sets the OFFSET clause
func (sb *SelectBuilder) Offset(offset int) *SelectBuilder {
	sb.offset = offset
	return sb
}

// Build constructs and returns the parameterized SQL query
// and the parameters used for it.
func (sb SelectBuilder) Build() (string, []any, error) {
	if sb.err != nil {
		return "", nil, sb.err
	}

	whereStatements := make([]string, 0)
	allParams := make([]any, 0)
	paramIndex := 1

	for pair := sb.where.Oldest(); pair != nil; pair = pair.Next() {
		k, v := pair.Key, pair.Value

		paramName := fmt.Sprintf("$%d", paramIndex)
		whereClause := strings.Replace(v, "?", paramName, 1)
		whereStatements = append(whereStatements, fmt.Sprintf("%s%s", k, whereClause))
		paramIndex++
	}

	allParams = append(allParams, sb.params...)

	var builder strings.Builder

	builder.WriteString("SELECT ")
	builder.WriteString(strings.Join(sb.columns, ", "))
	builder.WriteString(fmt.Sprintf(" FROM %s", sb.table))

	if len(whereStatements) > 0 {
		builder.WriteString(" WHERE ")
		builder.WriteString(strings.Join(whereStatements, " AND "))
	}

	if sb.limit > 0 {
		builder.WriteString(fmt.Sprintf(" LIMIT %d", sb.limit))
	}

	if sb.offset > 0 {
		builder.WriteString(fmt.Sprintf(" OFFSET %d", sb.offset))
	}

	return builder.String(), allParams, nil
}

// Select creates and returns a new SelectBuilder with specified columns.
// Not named `NewSelectBuilder` to make constructing
// queries read more like raw SQL
func Select(columns ...string) *SelectBuilder {
	sb := &SelectBuilder{
		columns: make([]string, 0),
		where:   orderedmap.New[string, string](),
		params:  make([]any, 0),
	}

	// * If there are no input columns
	// * assume every column
	if len(columns) == 0 {
		sb.columns = append(sb.columns, "*")
	} else {
		sb.columns = columns
	}

	return sb
}
