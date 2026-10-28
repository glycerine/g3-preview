package polynomial

// Source: algebraic_numbers.cpp:2709-2714; mpq.cpp:130-139.

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicDisplayIntervalBasicReachedFields(t *testing.T) {
	for _, tc := range []struct {
		name     string
		words    map[uint64]uint32
		want     string
		complete bool
	}{
		{"none", nil, "[", false},
		{"num-only", map[uint64]uint32{0: 7, 4: 0, 8: 0}, "[", false},
		{"den-only", map[uint64]uint32{12: 3, 16: 0, 20: 0}, "[", false},
		// The nonnull DEN pointer makes is_one false without reading DEN.Val.
		// Display then reaches NUM and slash before the missing DEN.Val guard.
		{"big-den-missing-val", map[uint64]uint32{0: 7, 4: 0, 8: 0, 16: 1, 20: 0}, "[7/", false},
		{"fraction", map[uint64]uint32{0: 7, 4: 0, 8: 0, 12: 3, 16: 0, 20: 0}, "[7/3, 7/3]", true},
		{"integer", map[uint64]uint32{0: 7, 4: 0, 8: 0, 12: 1, 16: 0, 20: 0}, "[7, 7]", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			a := algebraicQueryRawCell(t, c, 0)
			l, e := c.location(&a)
			serviceMust(t, e)
			for offset, word := range tc.words {
				serviceMust(t, c.config.Allocator.WriteUint32(l.raw, offset, word))
			}
			var out bytes.Buffer
			e = c.DisplayInterval(&out, &a)
			if tc.complete {
				serviceMust(t, e)
			} else if !errors.Is(e, umemory.ErrRuntimeUninitialized) {
				t.Fatalf("missing source member guard: %v", e)
			}
			if out.String() != tc.want {
				t.Fatalf("source prefix got %q want %q", out.String(), tc.want)
			}
			serviceMust(t, c.CleanupRetained())
		})
	}
}

type algebraicBasicIntervalHookWriter struct {
	bytes.Buffer
	hook func(string)
}

func (w *algebraicBasicIntervalHookWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}
func (w *algebraicBasicIntervalHookWriter) Write(p []byte) (int, error) {
	if w.hook != nil {
		w.hook(string(p))
	}
	return w.Buffer.Write(p)
}

func TestAlgebraicDisplayIntervalBasicRereadsAtSourceInsertions(t *testing.T) {
	for _, at := range []string{"/", ", "} {
		t.Run(at, func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			a := basicValueRawPrefix(t, c, 2)
			l, e := c.location(&a)
			serviceMust(t, e)
			calls := 0
			out := &algebraicBasicIntervalHookWriter{hook: func(s string) {
				if s != at {
					return
				}
				calls++
				if at == "/" {
					serviceMust(t, c.config.Allocator.WriteUint32(l.raw, 12, 5))
				} else {
					serviceMust(t, c.config.Allocator.WriteUint32(l.raw, 0, 9))
					serviceMust(t, c.config.Allocator.WriteUint32(l.raw, 12, 1))
				}
			}}
			serviceMust(t, c.DisplayInterval(out, &a))
			want := "[7/5, 7/5]"
			wantCalls := 2
			if at == ", " {
				want = "[7/3, 9]"
				wantCalls = 1
			}
			if out.String() != want || calls != wantCalls {
				t.Fatalf("source reread got %q calls%d want %q calls%d", out.String(), calls, want, wantCalls)
			}
		})
	}
}

func TestAlgebraicDisplayIntervalBasicGetterAssertionPrecedesNumeral(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions compiled out")
	}
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	a := basicValueRawPrefix(t, c, 2)
	polynomialAssertionActionsFixture(t, "T")
	serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	var out bytes.Buffer
	e := c.DisplayInterval(&out, &a)
	if !errors.Is(e, arena.ErrClosed) || out.String() != "[" {
		t.Fatalf("basic getter assertion prefix %q error %v", out.String(), e)
	}
}

func TestAlgebraicDisplayIntervalBasicMultiwordNumerals(t *testing.T) {
	for _, text := range []string{"-3/2", "18446744073709551619", "-18446744073709551619/7"} {
		t.Run(text, func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			q := c.config.Rationals
			value := unumeral.NewRational()
			serviceMust(t, q.SetString(&value, text))
			var a Algebraic
			serviceMust(t, c.SetRational(&a, &value))
			defer func() { serviceMust(t, c.Del(&a)); serviceMust(t, q.Del(&value)) }()
			var out bytes.Buffer
			serviceMust(t, c.DisplayInterval(&out, &a))
			if want := "[" + text + ", " + text + "]"; out.String() != want {
				t.Fatalf("signed/multiword output got %q want %q", out.String(), want)
			}
			for i := range c.intervalDisplayInsertions {
				if c.intervalDisplayInsertions[i].Active() {
					t.Fatal("completed formatter retained an owning child", i)
				}
			}
			// The predicate's post-refinement interval callee also accepts a
			// newly basic cell. Its actual formatter belongs to that caller.
			l, e := c.location(&a)
			serviceMust(t, e)
			out.Reset()
			serviceMust(t, c.predicateIntervalTrace(algebraicPredicateTraceStream{&out}, l))
			if text != "-3/2" {
				for i := 1; i <= 2; i++ {
					if c.predicateTrace.insertions[i] == (unumeral.NumeralInsertionReceipt{}) {
						t.Fatal("predicate basic formatter omitted its caller receipt", i)
					}
				}
			}
			for i := range c.intervalDisplayInsertions {
				if c.intervalDisplayInsertions[i].Active() {
					t.Fatal("predicate basic formatter acquired public display receipt", i)
				}
			}
			serviceMust(t, c.CleanupRetained())
		})
	}
}
