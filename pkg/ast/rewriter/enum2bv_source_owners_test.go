package rewriter

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestEnum2BVIndependentUtilityCachesAndCleanupReconstruction(t *testing.T) {
	m, sort, cs := enum2BVFixture(t)
	handles := append([]ast.ASTHandle{sort}, cs...)
	before := make([]uint32, len(handles))
	for i, h := range handles {
		astTestPin(t, m, h)
		var err error
		before[i], err = m.GetRefCount(h)
		astTestMust(t, err)
	}
	r, err := enum2BVNewForTest(t, m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	outer, inner := r.utility, r.config.utility
	if outer == inner {
		t.Fatal("source imp and rw_cfg share datatype utility")
	}
	calls := 0
	astTestMust(t, r.SetSortPredicate(func(h ast.ASTHandle) (bool, error) {
		calls++
		for i, h := range handles {
			count, err := m.GetRefCount(h)
			if err != nil {
				return false, err
			}
			want := before[i] + 1
			if i == 0 {
				want++
			}
			if count != want {
				t.Fatal("source enum cache was not populated before sort predicate", i, count, want)
			}
		}
		return false, nil
	}))
	for i := 0; i < 2; i++ {
		yes, err := r.isFinite(sort)
		astTestMust(t, err)
		if yes {
			t.Fatal("sort predicate rejection ignored")
		}
	}
	if calls != 2 {
		t.Fatal("predicate result incorrectly cached")
	}
	_, err = inner.Constructors(sort)
	astTestMust(t, err)
	for i, h := range handles {
		count, err := m.GetRefCount(h)
		astTestMust(t, err)
		want := before[i] + 2
		if i == 0 {
			want++
		}
		if count != want {
			t.Fatal("configuration cache borrowed outer constructor owner", i, count, want)
		}
	}
	astTestMust(t, r.Pop(0))
	astTestMust(t, r.Cleanup())
	if r.utility == outer || r.config.utility == inner || r.predicate != nil || r.config.owner != r {
		t.Fatal("Cleanup did not reconstruct source imp/cfg owners")
	}
	if _, err := outer.IsEnumSort(sort); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("old utility survived Cleanup", err)
	}
	for i, h := range handles {
		count, err := m.GetRefCount(h)
		astTestMust(t, err)
		if count != before[i] {
			t.Fatal("Cleanup retained datatype cache pin", i, count, before[i])
		}
	}
}

func enum2BVBoundFixture(t *testing.T) (*ast.ASTManager, *Enum2BVRewriter) {
	t.Helper()
	m, sort, _ := enum2BVFixture(t)
	r, err := enum2BVNewForTest(t, m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	a, err := m.MkConst(astTestSymbol(t, m, "enum-output-a"), sort)
	astTestMust(t, err)
	b, err := m.MkConst(astTestSymbol(t, m, "enum-output-b"), sort)
	astTestMust(t, err)
	eq, err := m.MkEq(a, b)
	astTestMust(t, err)
	out, proof := enum2BVRefs(t, m)
	astTestMust(t, r.Rewrite(eq, out, proof))
	if len(r.bounds) != 2 {
		t.Fatal("fixture did not create two actual bounds")
	}
	return m, r
}

func TestEnum2BVFlushedBoundsOwnedByCaller(t *testing.T) {
	for _, mode := range []string{"close", "cleanup"} {
		t.Run(mode, func(t *testing.T) {
			m, r := enum2BVBoundFixture(t)
			values := append([]ast.ASTHandle(nil), r.bounds...)
			var out []ast.ASTRef
			t.Cleanup(func() {
				for i := range out {
					astTestMust(t, m.DelRef(&out[i]))
				}
			})
			before, err := m.GetRefCount(values[0])
			astTestMust(t, err)
			astTestMust(t, r.FlushSideConstraints(&out))
			after, err := m.GetRefCount(values[0])
			astTestMust(t, err)
			if len(out) != 2 || len(r.bounds) != 0 || len(r.boundRefs) != 0 || before != after {
				t.Fatal("flush did not append owning outputs then reset source bounds", len(out), before, after)
			}
			if mode == "close" {
				astTestMust(t, r.Close())
			} else {
				astTestMust(t, r.Cleanup())
			}
			for i, ref := range out {
				value, err := m.RefGet(ref)
				astTestMust(t, err)
				if value != values[i] {
					t.Fatal("output owner changed after rewriter retirement")
				}
				kind, err := m.GetDeclKind(value)
				astTestMust(t, err)
				if kind != 22 {
					t.Fatal("flushed bound no longer usable", kind)
				}
			}
		})
	}
}

func TestEnum2BVFlushFailurePreservesCallerPrefix(t *testing.T) {
	m, r := enum2BVBoundFixture(t)
	first := r.bounds[0]
	old := append([]ast.ASTRef(nil), r.boundRefs...)
	var pins, out []ast.ASTRef
	t.Cleanup(func() {
		for i := range out {
			astTestMust(t, m.DelRef(&out[i]))
		}
		for i := range pins {
			if pins[i] != (ast.ASTRef{}) {
				astTestMust(t, m.DelRef(&pins[i]))
			}
		}
	})
	seed, err := m.NewRef(first)
	astTestMust(t, err)
	pins = append(pins, seed)
	// Reach a real reference-page frontier through public NewRef, then free
	// exactly one slot so output1 succeeds and output2 refuses allocation.
	for i := 0; i < 4096; i++ {
		m.Budget().FailAfter(0)
		pin, err := m.NewRef(first)
		m.Budget().FailAfter(-1)
		if errors.Is(err, arena.ErrInjectedFailure) {
			break
		}
		astTestMust(t, err)
		pins = append(pins, pin)
		if i == 4095 {
			t.Fatal("reference-page frontier not reached")
		}
	}
	if len(pins) == 0 {
		t.Fatal("fixture needs a releasable reference")
	}
	astTestMust(t, m.DelRef(&pins[len(pins)-1]))
	m.Budget().FailAfter(0)
	err = r.FlushSideConstraints(&out)
	m.Budget().FailAfter(-1)
	if !errors.Is(err, arena.ErrInjectedFailure) || len(out) != 1 || !reflect.DeepEqual(old, r.boundRefs) {
		t.Fatal("failed flush lost caller prefix or reset source before append completed", err, len(out))
	}
	astTestMust(t, r.Close())
	value, err := m.RefGet(out[0])
	astTestMust(t, err)
	if value != first {
		t.Fatal("rewriter Close invalidated failed-flush caller prefix")
	}
}

type enum2BVStopWriter struct {
	action func() error
	err    error
	called bool
}

func (w *enum2BVStopWriter) Write(p []byte) (int, error) {
	if !w.called {
		w.called = true
		w.err = w.action()
	}
	return len(p), nil
}

func TestEnum2BVCloseStopsAtConfigurationUtility(t *testing.T) {
	s, q, b := astTestOwners(t)
	writer := &enum2BVStopWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: q, Budget: b, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.Close()) })
	astTestMust(t, m.RegisterDeclarationPlugins())
	r, err := enum2BVNewForTest(t, m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	traversal := r.traversal
	header := r.header
	sort, err := m.FloatingPointSort(3, 4)
	astTestMust(t, err)
	tags.Enable("ast")
	if !tags.ShouldTrace("ast") {
		t.Skip("requires source trace build")
	}
	writer.action = r.Close
	_, err = m.MkFPAZeroOfSort(sort, false)
	astTestMust(t, err)
	if !writer.called || !errors.Is(writer.err, ast.ErrASTPending) || r.traversal != traversal || r.header != header || r.closed {
		t.Fatal("configuration utility failure entered later member destruction", writer.called, writer.err)
	}
	if _, err := r.config.utility.IsEnumSort(sort); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("configuration utility did not enter its Close barrier", err)
	}
	if _, err := r.NumSteps(); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("stopped owner remained usable", err)
	}
	astTestMust(t, r.Close())
	astTestMust(t, r.Close())
}

func TestEnum2BVConstructionAndCleanupAllocationRefusals(t *testing.T) {
	m, _, _ := enum2BVFixture(t)
	c, p := enum2BVParameters(t, m)
	astTestMust(t, c.SetUInt(p, astTestSymbol(t, m, "constructor-owned-params"), 17))
	warm, err := NewEnum2BVRewriter(m, c, p)
	astTestMust(t, err)
	astTestMust(t, warm.Close())
	baseline := m.Budget().Used()
	completed := false
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		m.Budget().FailAfter(ordinal)
		r, err := NewEnum2BVRewriter(m, c, p)
		m.Budget().FailAfter(-1)
		if r != nil {
			astTestMust(t, r.Close())
		}
		if m.Budget().Used() != baseline {
			t.Fatal("constructor refusal retained partial owners", ordinal, m.Budget().Used(), baseline)
		}
		if c.Stats().References != 1 {
			t.Fatal("constructor refusal lost parameter member prefix", ordinal, c.Stats())
		}
		if err == nil {
			completed = true
			break
		}
		if !errors.Is(err, arena.ErrInjectedFailure) {
			t.Fatal("unexpected constructor refusal", ordinal, err)
		}
	}
	if !completed {
		t.Fatal("constructor allocation census did not complete")
	}
	r, err := NewEnum2BVRewriter(m, c, p)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	m.Budget().FailAfter(0)
	err = r.Cleanup()
	m.Budget().FailAfter(-1)
	if !errors.Is(err, arena.ErrInjectedFailure) || r.header != 0 || r.traversal != nil || r.utility != nil || r.config.utility != nil {
		t.Fatal("Cleanup refusal did not retire old imp before replacement allocation", err)
	}
	if _, err := r.NumSteps(); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("failed replacement remained usable", err)
	}
	astTestMust(t, r.Cleanup())
	if _, err := r.NumSteps(); err != nil {
		t.Fatal("Cleanup replacement retry did not become ready", err)
	}
}

func TestEnum2BVTranslationWarmsConfigurationCache(t *testing.T) {
	m, r := enum2BVBoundFixture(t)
	sort, err := m.GetRange(r.entries[0].original)
	astTestMust(t, err)
	before, err := m.GetRefCount(sort)
	astTestMust(t, err)
	_, err = r.config.utility.Constructors(sort)
	astTestMust(t, err)
	after, err := m.GetRefCount(sort)
	astTestMust(t, err)
	if before != after {
		t.Fatal("translation bypassed rw_cfg constructor utility", before, after)
	}
}
