package gcfg

import (
	"fmt"
	"strings"
)

const (
	// initial size of buffer for ScanEnum
	scanEnumBufferHint = 16
)

func strEq(s1, s2 string, fold bool) bool {
	return fold && strings.EqualFold(s1, s2) || !fold && s1 == s2
}

// ScanEnum is a helper function to simplify the implementation of fmt.Scanner
// methods for "enum-like" types, that is, user-defined types where the set of
// values and string representations is fixed.
// ScanEnum allows multiple string representations for the same value.
//
// State is the state passed to the implementation of the fmt.Scanner method.
// Values holds as map values the values of the type, with their string
// representations as keys.
// If fold is true, comparison of the string representation uses
// strings.EqualFold, otherwise the equal operator for strings.
//
// On a match, ScanEnum stops after reading the last rune of the matched string,
// and returns the corresponding value together with a nil error.
// On no match, ScanEnum attempts to unread the last rune (the first rune that
// could not potentially match any of the values), and returns a non-nil error,
// together with a nil value for interface{}.
// On I/O error, ScanEnum returns the I/O error, together with a nil value for
// interface{}.
//
func scanEnum(state fmt.ScanState, values map[string]interface{}, fold bool) (
	interface{}, error) {
	//
	rd := make([]rune, 0, scanEnumBufferHint)
	keys := make(map[string]struct{}, len(values)) // potential keys
	for s, _ := range values {
		keys[s] = struct{}{}
	}
	for {
		r, _, err := state.ReadRune()
		if err != nil {
			return nil, err
		}
		rd = append(rd, r)
		srd := string(rd)
		lrd := len(srd)
		for s, _ := range keys {
			if strEq(srd, s, fold) {
				return values[s], nil
			}
			if len(rd) < len(s) && !strEq(srd, s[:lrd], fold) {
				delete(keys, s)
			}
		}
		if len(keys) == 0 {
			state.UnreadRune()
			return nil, fmt.Errorf("unsupported value %q", srd)
		}
	}
	panic("never reached")
}
