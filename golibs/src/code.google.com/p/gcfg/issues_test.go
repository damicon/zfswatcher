package gcfg

import (
	"math/big"
	"strings"
	"testing"
)

type Config1 struct {
	Section struct {
		Int    int
		BigInt big.Int
	}
}

var testsIssue1 = []struct {
	cfg      string
	typename string
}{
	{"[section]\nint=X", "int"},
	{"[section]\nint=", "int"},
	{"[section]\nint=1A", "int"},
	{"[section]\nbigint=X", "big.Int"},
	{"[section]\nbigint=", "big.Int"},
	{"[section]\nbigint=1A", "big.Int"},
}

// Value parse error should:
//  - include plain type name
//  - not include reflect internals
func TestIssue1(t *testing.T) {
	for i, tt := range testsIssue1 {
		var c Config1
		err := ReadStringInto(&c, tt.cfg)
		switch {
		case err == nil:
			t.Errorf("%d fail: got ok; wanted error", i)
		case !strings.Contains(err.Error(), tt.typename):
			t.Errorf("%d fail: error message doesn't contain type name %q: %v",
				i, tt.typename, err)
		case strings.Contains(err.Error(), "reflect"):
			t.Errorf("%d fail: error message includes reflect internals: %v",
				i, err)
		default:
			t.Logf("%d pass: %v", i, err)
		}
	}
}
