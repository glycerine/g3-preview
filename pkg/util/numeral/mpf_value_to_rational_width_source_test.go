package numeral

// Source: src/util/mpf.cpp:1695-1711.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

// Source -a.exponent is undefined for INT64_MIN. This represented Go input
// refuses at that reached expression, after the literal preceding SET/Power;
// it does not claim a C++ assertion or mathematical overflow prefix.
func TestArbitraryFloatToRationalUndefinedSignedNegationGuard(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	cfg := IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}
	m, e := NewArbitraryFloatManager(cfg)
	serviceMust(t, e)
	var x ArbitraryFloat
	one := NewInt(1)
	serviceMust(t, m.SetComponents(&x, 8, 4, false, -1<<63, &one))
	out := NewRational()
	e = m.ToRational(&x, m.rational, &out)
	s := &m.rationalConversion
	if !errors.Is(e, arena.ErrBounds) || s.numericPhase != 5 || s.constructed != 3 || s.sourceCursor != 3 || s.bodyCause != e {
		t.Fatal("reached signed-domain guard", e, s.numericPhase, s.constructed, s.sourceCursor, s.bodyCause)
	}
	if out != NewRational() {
		t.Fatal("unreached output SET", out)
	}
	serviceMust(t, m.CleanupRationalConversion())
	serviceMust(t, m.Del(&x))
	if e := m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
		t.Fatal(e)
	}
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
