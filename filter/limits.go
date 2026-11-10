package filter

import "fmt"

// ValidateFilterNodeLimit checks a filter tree before callers build dynamic
// policies from its fields. Group and leaf occurrences both count; zero means
// unlimited for backward compatibility. A depth bound also rejects cycles.
func ValidateFilterNodeLimit(filters []CrudFilter, maxNodes int) error {
	if maxNodes < 0 {
		return fmt.Errorf("%w: negative filter node limit", ErrInvalidQuery)
	}
	count := 0
	var visit func([]CrudFilter, int) error
	visit = func(nodes []CrudFilter, depth int) error {
		if depth > 32 {
			return fmt.Errorf("%w: filter nesting too deep", ErrInvalidQuery)
		}
		for _, node := range nodes {
			count++
			if maxNodes > 0 && count > maxNodes {
				return fmt.Errorf("%w: too many filter nodes", ErrInvalidQuery)
			}
			switch f := node.(type) {
			case *LogicalFilter:
				if f == nil {
					return fmt.Errorf("%w: nil filter", ErrInvalidQuery)
				}
			case *ConditionalFilter:
				if f == nil {
					return fmt.Errorf("%w: nil group", ErrInvalidQuery)
				}
				if err := visit(f.Filters, depth+1); err != nil {
					return err
				}
			default:
				return fmt.Errorf("%w: unsupported filter type %T", ErrInvalidQuery, node)
			}
		}
		return nil
	}
	return visit(filters, 0)
}
