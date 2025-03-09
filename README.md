# queryutil

A Go library that implements query parsing utilities compatible with [Refine's Simple REST Data Provider](https://www.npmjs.com/package/@refinedev/simple-rest) ([source](https://github.com/refinedev/refine/tree/main/packages/simple-rest)) for filtering, sorting, and pagination.

## Features

- Filter parsing compatible with Refine's data provider spec
- Sort handling with multiple fields and directions
- Pagination with server/client modes
- GORM integration helpers

## Installation

```bash
go get go.lumeweb.com/queryutil
```

## Usage

### Request Parsing

```go
// Parse all query parameters from request
filters, sorts, pagination, err := queryutil.ParseRequest(r)
if err != nil {
    // Handle error
}

// Apply to GORM query
db = queryutil.ApplyFilters(db, filters)
db = queryutil.ApplySort(db, sorts)
db = db.Offset(pagination.GetOffset()).Limit(pagination.GetLimit())
```

With global search support:

```go
// Configure searchable columns
searchConfig := &queryutil.GlobalSearchConfig{
    SearchableColumns: []string{"name", "email", "description"},
}

// Parse request with search config
filters, sorts, pagination, err := queryutil.ParseRequestWithSearch(r, searchConfig)
if err != nil {
    // Handle error
}

// Apply to GORM query - pass searchConfig for global search support
db = queryutil.ApplyFilters(db, filters, searchConfig)
db = queryutil.ApplySort(db, sorts)
db = db.Offset(pagination.GetOffset()).Limit(pagination.GetLimit())
```

### Individual Parameter Parsing

You can also parse parameters individually if needed:

```go
// Parse filters
filters, err := query.ParseQueryFilters(r.URL.Query())
if err != nil {
    // Handle error
}

// Parse sort
sorts := query.ParseQuerySort(r.URL.Query())

// Parse pagination
pagination := query.ParseQueryPagination(r.URL.Query())
```

Supported operators:
- Default (no suffix): Exact match
- `_ne`: Not equals
- `_gte`: Greater than or equal
- `_lte`: Less than or equal  
- `_like`: Contains search

Special parameter `q` is reserved for global search.

### Sorting

```go
// Parse sort parameters
sorts := query.ParseQuerySort(
    r.URL.Query().Get("_sort"),
    r.URL.Query().Get("_order"),
)

// Apply to GORM query
db = query.ApplySort(db, sorts)
```

### Pagination

```go
// Parse pagination parameters
pagination := query.ParseQueryPagination(r.URL.Query())

// Get SQL LIMIT/OFFSET
limit := pagination.GetLimit()
offset := pagination.GetOffset()

// Apply to GORM query
db = db.Offset(offset).Limit(limit)
```

## Compatibility

This library implements the query parameter specification used by Refine's [Simple REST Data Provider](https://refine.dev/docs/api-reference/core/providers/simple-rest/).

## License

MIT License - see LICENSE file
