// Package filter provides utilities for parsing, validating, and applying filters, sorts, 
// and pagination parameters from HTTP requests to database queries. The package implements
// a full-stack filtering solution with these key features:
//
//  - Unified query parameter parsing for both URL parameters and JSON input
//  - Type-safe validation of filter operators and values
//  - Database-agnostic query building with multiple driver implementations
//  - Complex logical combinations using AND/OR/NOT operators
//  - Server-side pagination with Content-Range header support
//  - Secure whitelist validation for sortable fields
//  - Global search capabilities across multiple columns
//
// The package follows a visitor pattern architecture for easy extension to different
// database systems while maintaining a consistent API for filter processing.
package filter
