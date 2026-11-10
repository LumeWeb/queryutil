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

#### JSON Input Support
```go
// Parse filters from JSON string
jsonFilters := `[
  {
    "field": "name",
    "operator": "contains",
    "value": "john"
  },
  {
    "operator": "or",
    "value": [
      {
        "field": "age",
        "operator": "gte",
        "value": 30
      }
    ]
  }
]`

filters, _, _, err := queryutil.ParseRequestFromJSON(jsonFilters)
if err != nil {
    // Handle error
}
```

#### Mixed Input Handling
```go
// If using both parameter and JSON parsing:
paramsFilters, paramSorts, pagination, err := queryutil.ParseRequest(r)
jsonFilters, _, _, err := queryutil.ParseRequestFromJSON(body)

// Merge filter sets
allFilters := append(paramsFilters, jsonFilters...)
```

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

### Preserve string-valued query fields

URL query values normally undergo numeric and boolean coercion. Use
`WithStringFields` when a field's exact text must reach validation unchanged:

```go
filters, sorts, _, err := queryutil.ParseRequestWithOptions(
    r, queryutil.WithStringFields("fingerprint"),
)
if err != nil {
    return err
}
// Validate filters and sorts with a QueryPolicy before building a query.
```

The option also applies to nested filters and array values. It does not change
other fields or JSON input. The reserved search field `q` always retains its
query-parameter text, even without the option. Numeric comparison and range
operators on a string-preserved field return an error rather than coercing their
values. For direct query-parameter parsing, pass the option
to `parser.NewQueryParamParser`; `NewHTTPRequestParser` accepts it as an
optional fourth argument.

### Restricting untrusted queries

Parsing alone does not authorize filter or sort fields. Validate at the service
boundary as well as at HTTP entry points, before calling `ApplyFilters` or
`ApplySort`:

```go
policy := queryutil.QueryPolicy{
    Fields: map[string]queryutil.FieldRule{
        "id":          {Operators: []queryutil.Operator{queryutil.OpEq}},
        "fingerprint": {Operators: []queryutil.Operator{queryutil.OpEq}, Convert: decodeFingerprint},
    },
    SortFields: []string{"id", "fingerprint"},
}
safeFilters, err := policy.Normalize(filters, sorts)
if err != nil {
    return err // reject the request; do not build a query
}
// Use safeFilters (not filters) for ApplyFilters; sorts have been validated too.
```

`Convert` has signature `func(queryutil.Operator, any) (any, error)` and receives
the full value (including arrays for IN/BETWEEN). It runs for matching fields
at every depth of AND/OR/NOT groups. The returned filter nodes are new;
mutable values without a converter may still be shared with the input.
If every operator supported for scalar fields is appropriate, use
`queryutil.AllScalarFieldRules("id", "created_at")` to build those field rules,
or `queryutil.ScalarOperators()` for an individual rule. Both return fresh
operator slices; prefer explicit per-field lists when fewer operators are safe.
For equality, range, IN, and BETWEEN without pattern or null operators, use
`queryutil.ComparableFieldRules("id", "created_at")` or
`queryutil.ComparableOperators()` for a single rule. These also return
independent operator slices. Sort fields must still be declared separately.

Configure permitted operators and sort fields explicitly. The special `q`
field requires `OpContains` in the policy; `q[eq]` is accepted as an alias
and normalized to `OpContains`. Search needs a configured global search;
dotted fields cannot use array or range operators because they may be JSON paths.

### Policy-aware HTTP lists

`http.ProcessListRequestWithPolicy` is a separate entry point; the existing
`ProcessListRequest` signature and behavior are unchanged. Supply a policy,
the usual service and DTO converter, and parser options such as
`queryutil.WithStringFields("external_id")`. It rejects malformed/unauthorized
queries before calling the service and maps service `ErrInvalidQuery` to HTTP
400. It retains the existing JSON response and `Content-Range` behavior.

The policy-aware entry point reads already-decoded `r.URL.Query()` values and
never unescapes text a second time. The legacy parser retains its original
pre-escaped-value behavior; direct callers can use `parser.WithRawQueryValues()`
for decoded `url.Values`. `FieldRule.Convert` runs during HTTP normalization;
if the service checks the same policy again, converters must accept their own
already-converted output (for example, a string-to-binary converter must also
accept that binary type). Service entry points called without HTTP must still
normalize their filters. Keep ownership constraints independent of client
filters and apply them to the query separately.

### Application-specific GORM predicates

For a virtual field with no direct database column, register a trusted
predicate with `queryutil.ApplyFiltersWithOptions(db, safeFilters, searchConfig,
queryutil.WithFieldPredicate("domain", predicate))`, or use
`builder.NewGORMBuilderWithOptions` directly. The callback receives the
normalized operator and value and returns a SQL condition, bound arguments,
and an error. For example, an `EXISTS (SELECT ... WHERE domain = ?)` fragment
binds the domain in its arguments. The builder places the condition inside
AND/OR/NOT groups; the application supplies the SQL and its `QueryPolicy`
allowlist. Always normalize untrusted filters before applying them; the
builder does not authorize a field or infer ownership from the predicate.

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
