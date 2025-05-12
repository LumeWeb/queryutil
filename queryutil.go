package queryutil

import (
	"fmt"
	"go.lumeweb.com/queryutil/filter"
	"go.lumeweb.com/queryutil/filter/parser"
)

type (
	// RequestParser re-exports the parser.Parser interface
	RequestParser      = parser.Parser
	Sort               = filter.Sort
	SortConfig         = filter.SortConfig
	SortOrder          = filter.SortOrder  
	CrudFilter         = filter.CrudFilter
	GlobalSearchConfig = filter.GlobalSearchConfig
	Pagination         = filter.Pagination
	PaginationError    = filter.PaginationError
	Filter             = filter.CrudFilter
	Operator           = filter.Operator
)

// Re-export filter operators as typed constants
const (
	OpEq           = filter.OpEq
	OpNe           = filter.OpNe
	OpLt           = filter.OpLt
	OpGt           = filter.OpGt
	OpLte          = filter.OpLte
	OpGte          = filter.OpGte
	OpContains     = filter.OpContains
	OpContainss    = filter.OpContainss
	OpNcontains    = filter.OpNcontains
	OpNcontainss   = filter.OpNcontainss
	OpIn           = filter.OpIn
	OpNin          = filter.OpNin
	OpIna          = filter.OpIna
	OpNina         = filter.OpNina
	OpBetween      = filter.OpBetween
	OpNbetween     = filter.OpNbetween
	OpNull         = filter.OpNull
	OpNnull        = filter.OpNnull
	OpStartswith   = filter.OpStartswith
	OpStartswiths  = filter.OpStartswiths
	OpNstartswith  = filter.OpNstartswith
	OpNstartswiths = filter.OpNstartswiths
	OpEndswith     = filter.OpEndswith
	OpEndswiths    = filter.OpEndswiths
	OpNendswith    = filter.OpNendswith
	OpNendswiths   = filter.OpNendswiths
)

// Re-export operator map as a variable
var OperatorMap = filter.OperatorMap

// QueryBuilder defines the interface for building query clauses
type QueryBuilder interface {
	Apply(tx interface{}, filters []filter.CrudFilter) (interface{}, error)
}

// ParseFromSource is the unified parsing entry point using the RequestParser interface
func ParseFromSource(parser RequestParser) ([]CrudFilter, []Sort, Pagination, error) {
	filters, err := parser.ParseFilters()
	if err != nil {
		return nil, nil, Pagination{}, fmt.Errorf("error parsing filters: %w", err)
	}

	sorts, err := parser.ParseSorts(nil) // TODO: Pass appropriate SortConfig
	if err != nil {
		return filters, nil, Pagination{}, fmt.Errorf("error parsing sorts: %w", err)
	}

	pagination, err := parser.ParsePagination()
	if err != nil {
		return filters, sorts, Pagination{}, fmt.Errorf("error parsing pagination: %w", err)
	}

	return filters, sorts, pagination, nil
}

var ParseQuerySort = filter.ParseQuerySort
var GetResultCount = filter.GetResultCount
var FormatContentRange = filter.FormatContentRange
var NewPaginationError = filter.NewPaginationError
var NewSortError = filter.NewSortError
var NewFilterError = filter.NewFilterError
var ParseQueryPagination = filter.ParseQueryPagination
