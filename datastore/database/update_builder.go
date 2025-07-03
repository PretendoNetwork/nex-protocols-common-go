package database

import (
	"errors"
	"fmt"
	"strings"
)

// UpdateBuilder is an SQL `UPDATE` statement builder.
// Made to be simple and intuitive, without the need
// for complex features like code generation and while
// reading as closely to raw SQL queries as possible
type UpdateBuilder struct {
	table     string
	set       map[string]any
	where     map[string]string
	lastWhere string
	params    []any
	err       error
}

// Set sets one or more fields to be updated in the UPDATE statement
//
// Supports the following formats:
// - `Set("key", "value")`
// - `Set(map[string]any{"key": "value"})`
//
// If `Set("key", "value")` is used, only 1 `value` parameter is allowed
//
// If `Set(map[string]any{"key": "value"})` is used, the `value` parameter is ignored
func (ub *UpdateBuilder) Set(field any, value ...any) *UpdateBuilder {
	switch f := field.(type) {
	case map[string]any:
		for k, v := range f {
			ub.set[k] = v
		}
	case string:
		if len(value) == 0 {
			ub.err = errors.New("invalid field value")
			break
		}

		ub.set[f] = value[0]
	default:
		// TODO - Maybe just use %v here or something instead, rather than erroring?
		ub.err = errors.New("unknown field type")
	}

	return ub
}

// Where sets the key to be used
func (ub *UpdateBuilder) Where(field string) *UpdateBuilder {
	ub.lastWhere = field

	return ub
}

// And is an alias for `Where` to make constructing queries
// read more like raw SQL
func (ub *UpdateBuilder) And(field string) *UpdateBuilder {
	return ub.Where(field)
}

// Is adds a new `key=value` statement for the WHERE clause.
// Uses the last set key from either `Where` or `And`
func (ub *UpdateBuilder) Is(value any) *UpdateBuilder {
	ub.params = append(ub.params, value)
	ub.where[ub.lastWhere] = "=?" // * Placeholder value. Gets replaced with $N in ub.Build

	return ub
}

// Build constructs and returns the paramterized SQL query
// and the parameters used for it.
//
// Note: The `SET` clause will NOT have the columns in the
// same order as they were defined in! Go does not guarantee
// the order of `map` keys!
func (ub UpdateBuilder) Build() (string, []any, error) {
	updateStatements := make([]string, 0)
	whereStatements := make([]string, 0)
	allParams := make([]any, 0)
	paramIndex := 1

	for k, v := range ub.set {
		paramName := fmt.Sprintf("$%d", paramIndex)
		updateStatements = append(updateStatements, fmt.Sprintf("%s=%s", k, paramName))
		allParams = append(allParams, v)
		paramIndex++
	}

	for k, v := range ub.where {
		paramName := fmt.Sprintf("$%d", paramIndex)
		whereClause := strings.Replace(v, "?", paramName, 1)
		whereStatements = append(whereStatements, fmt.Sprintf("%s%s", k, whereClause))
		paramIndex++
	}

	allParams = append(allParams, ub.params...)

	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("UPDATE %s SET ", ub.table))
	builder.WriteString(strings.Join(updateStatements, ", "))

	if len(whereStatements) != 0 {
		builder.WriteString(fmt.Sprintf(" WHERE %s", strings.Join(whereStatements, " AND ")))
	}

	return builder.String(), allParams, ub.err
}

// Update creates and returns a new UpdateBuilder.
// Not named `NewUpdateBuilder` to make constructing
// queries read more like raw SQL
func Update(table string) *UpdateBuilder {
	return &UpdateBuilder{
		table:  table,
		set:    make(map[string]any),
		where:  make(map[string]string),
		params: make([]any, 0),
	}
}
