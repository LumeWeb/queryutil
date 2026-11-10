package parser

import (
	"fmt"
	"go.lumeweb.com/queryutil/filter"
)

func countFilterNode(count *int, max int) error {
	(*count)++
	if max > 0 && *count > max {
		return fmt.Errorf("%w: too many filter nodes", filter.ErrInvalidQuery)
	}
	return nil
}
