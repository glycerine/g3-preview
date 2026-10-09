// Source: src/util/sorting_network.h:197-252,983-1019; src/util/debug.h:60-65.
package ast

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func psortActionFixture(t *testing.T, text string) (*PsortASTNetwork, *util.DebugContext, *util.NumericCharInput, *os.File) {
	t.Helper()
	b, r, d, input, streams := astPermutationActionFixture(t, text)
	symbols, e := util.NewSymbolManager(util.SymbolConfig{Budget: b})
	astTestMust(t, e)
	q, e := util.NewRationalService(util.RationalServiceConfig{Integers: util.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	astTestMust(t, e)
	m, e := NewASTManager(ASTManagerConfig{Symbols: symbols, Numerals: q, Budget: b})
	astTestMust(t, e)
	c, e := NewPsortASTContext(m)
	astTestMust(t, e)
	s, e := NewPsortASTNetwork(c, r, b, d)
	astTestMust(t, e)
	t.Cleanup(func() {
		b.FailAfter(-1)
		astTestMust(t, s.Close())
		astTestMust(t, c.Close())
		astTestMust(t, m.Close())
		astTestMust(t, q.Close())
		astTestMust(t, symbols.Close())
	})
	return s, d, input, streams
}
func TestPsortASTNetworkBoundContinueThenThrowActualSourcePrefix(t *testing.T) {
	s, _, input, streams := psortActionFixture(t, "CT")
	m := s.context.manager
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	x, e := m.MkFreshConst("a", sort)
	astTestMust(t, e)
	in, e := NewRuntimeASTLiteralVector(s.runtime, s.budget, s.debug)
	astTestMust(t, e)
	astTestMust(t, in.Push(x))
	p, e := in.CPtr()
	astTestMust(t, e)
	_, e = s.Eq(true, 0, 1, p)
	astTestMust(t, e)
	out, e := NewRuntimeASTLiteralVector(s.runtime, s.budget, s.debug)
	astTestMust(t, e)
	s.running = true
	cause := s.directSort(2, 1, psortASTPointer{span: p}, out)
	s.running = false
	data, e := os.ReadFile(streams.Name())
	astTestMust(t, e)
	state, e := input.State()
	astTestMust(t, e)
	if util.RuntimeVectorSourceDebug {
		var exception z3exception.DefaultException
		if !errors.As(cause, &exception) || !strings.Contains(string(data), "Line: 983\nm <= n\n") || !strings.Contains(string(data), "Line: 1019\nk + offset <= n\n") || state.Position != 2 {
			t.Fatal("source bound action chain", cause, string(data), state)
		}
		n, e := out.Size()
		astTestMust(t, e)
		if n != 2 {
			t.Fatal("C must reach two actual fresh/output publications before later T", n)
		}
	} else {
		if cause != nil || len(data) != 0 || state.Position != 0 {
			t.Fatal("release erased assertions", cause, string(data), state)
		}
	}
	astTestMust(t, in.Close())
	astTestMust(t, out.Close())
}
func TestPsortASTNetworkUnusedClosedAssertionProviderAndDisabledBound(t *testing.T) {
	s, d, input, streams := psortActionFixture(t, "T")
	astTestMust(t, d.EnableAssertions(false))
	s.running = true
	_, e := s.power2(10)
	s.running = false
	astTestMust(t, e)
	state, e := input.State()
	astTestMust(t, e)
	data, e := os.ReadFile(streams.Name())
	astTestMust(t, e)
	if state.Position != 0 || len(data) != 0 {
		t.Fatal("disabled assertion must not reach actions", state, string(data))
	}
	astTestMust(t, d.Close())
	before := s.budget.Used()
	_, e = s.Ge(true, 1, 0, ASTLiteralVectorSpan{})
	astTestMust(t, e)
	_, e = s.Le(true, 0, 0, ASTLiteralVectorSpan{})
	astTestMust(t, e)
	_, e = s.Eq(true, 1, 0, ASTLiteralVectorSpan{})
	astTestMust(t, e)
	if s.budget.Used() != before {
		t.Fatal("unused provider fast paths reserved storage")
	}
	s.running = true
	_, e = s.power2(1)
	s.running = false
	if util.RuntimeVectorSourceDebug {
		if !errors.Is(e, arena.ErrClosed) {
			t.Fatal("actual reached flag read", e)
		}
	} else {
		astTestMust(t, e)
	}
}
