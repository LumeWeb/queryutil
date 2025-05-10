// Package queryutil provides a comprehensive toolkit for building data-driven web services with advanced filtering,
// sorting and pagination capabilities. The module follows a clean architecture pattern with separate packages for:
// - HTTP adapter: Request/response handling
// - Filter: Core filtering logic and query building
// - Parser: Input parsing from different formats
// - Builder: Database-specific query building
//
// Key Features:
// - Full CRUD filter support with complex logical operators
// - Automatic validation of filter/sort/pagination parameters
// - Database-agnostic query building (currently supports GORM)
// - Server-side pagination with Content-Range headers
// - Support for both JSON and query parameter input formats
// - Type-safe value conversions and validation
// - Global search across multiple fields
//
// Usage Example:
//  // Parse HTTP request params
//  filters, sorts, pagination, err := queryutil.ParseRequestHTTP(r)
//  
//  // Build database query
//  query := db.Model(&User{})
//  gormBuilder := builder.NewGORMBuilder(db, nil)
//  filteredQuery, err := gormBuilder.Apply(query, filters)
//  
//  // Apply sorting and pagination
//  filteredQuery = gormBuilder.ApplySorts(filteredQuery, sorts)
//  filteredQuery = filteredQuery.Limit(pagination.GetLimit()).Offset(pagination.GetOffset())
//  
//  // Execute and format response
//  var results []User
//  filteredQuery.Find(&results)
//  total := queryutil.GetResultCount(results)
//  w.Header().Set("Content-Range", queryutil.FormatContentRange("users", pagination, total, total))
//  http.EncodeJSON(w, results)
package queryutil
