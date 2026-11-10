package filter

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// ValidateJSONPathField rejects bytes that cannot be represented safely in a
// database JSON path. Empty path segments are allowed: they name empty keys.
func ValidateJSONPathField(field string) error {
	if !utf8.ValidString(field) || strings.IndexFunc(field, func(r rune) bool { return r < 0x20 }) >= 0 {
		return fmt.Errorf("%w: invalid JSON path field", ErrInvalidQuery)
	}
	return nil
}
