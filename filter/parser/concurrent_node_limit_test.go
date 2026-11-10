package parser

import (
	"fmt"
	"net/url"
	"sync"
	"testing"

	"go.lumeweb.com/queryutil/filter"
)

func TestFilterNodeLimitIsLocalToConcurrentParses(t *testing.T) {
	cases := []struct {
		name  string
		parse func() error
	}{
		{"query parameters", func() error {
			p := NewQueryParamParser(url.Values{"filters[or][0][id][eq]": {"1"}, "filters[or][1][id][eq]": {"2"}}, WithMaxFilterNodes(3))
			return runConcurrentParses(p.ParseFilters)
		}},
		{"JSON", func() error {
			p := NewJSONParser(`[{"operator":"or","value":[{"field":"id","operator":"eq","value":1},{"field":"id","operator":"eq","value":2}]}]`, WithMaxFilterNodes(3))
			return runConcurrentParses(p.ParseFilters)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.parse(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func runConcurrentParses(parse func() ([]filter.CrudFilter, error)) error {
	var wg sync.WaitGroup
	errs := make(chan error, 32)
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 8; j++ {
				nodes, err := parse()
				if err != nil {
					errs <- err
					return
				}
				if len(nodes) != 1 {
					errs <- fmt.Errorf("unexpected root count %d", len(nodes))
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		return err
	}
	return nil
}
