//go:build g3checked && g3debug

package ast

// Source: datatype::decl::plugin::finalize and datatype::util::~util.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestDatatypePluginRealNumeralReleaseFailureRetry(t *testing.T) {
	b := arena.NewBudget(1 << 28)
	rt, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	astTestMust(t, err)
	cfg := unumeral.IntegerConfig{Memory: rt, Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}
	q, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: cfg})
	astTestMust(t, err)
	symbols, err := usymbol.NewManager(usymbol.Config{Budget: b})
	astTestMust(t, err)
	m, err := NewASTManager(ASTManagerConfig{Symbols: symbols, Numerals: q, Budget: b})
	astTestMust(t, err)
	_, err = m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterDatatypePlugin()
	astTestMust(t, err)
	scoped, err := unumeral.NewScopedInt(q.Manager.Z)
	astTestMust(t, err)
	astTestMust(t, scoped.SetInt64(1))
	raw, err := scoped.Get()
	astTestMust(t, err)
	astTestMust(t, q.Manager.Z.Mul2kInPlace(raw, 1600000))
	rational, err := q.NewFromScopedInt(scoped)
	astTestMust(t, err)
	parameter, err := m.NewParameterRational(&rational)
	astTestMust(t, err)
	info, err := m.NewSortInfo(NullFamilyID, 0, []ASTParameter{parameter}, InfiniteSortSize(), false)
	astTestMust(t, err)
	payload, err := m.MkSort(astTestSymbol(t, m, "NumericPayload"), info)
	astTestMust(t, err)
	astTestMust(t, m.DelSortInfo(&info))
	astTestMust(t, m.DelParameter(&parameter))
	sorts, err := m.MkDatatypes([]DatatypeDefinition{
		{Name: astTestSymbol(t, m, "NumericCache"), Constructors: []DatatypeConstructorDefinition{{Name: astTestSymbol(t, m, "numeric-constructor"), Recognizer: astTestSymbol(t, m, "is-numeric-constructor"), Accessors: []DatatypeAccessorDefinition{{Name: astTestSymbol(t, m, "numeric-field"), Range: payload}}}}},
		{Name: astTestSymbol(t, m, "LaterCache"), Constructors: []DatatypeConstructorDefinition{{Name: astTestSymbol(t, m, "later-constructor"), Recognizer: astTestSymbol(t, m, "is-later-constructor")}}},
	})
	astTestMust(t, err)
	_, found, err := m.DatatypeNonRecursiveConstructor(sorts[0])
	astTestMust(t, err)
	if !found {
		t.Fatal("missing numeric constructor")
	}
	later, found, err := m.DatatypeNonRecursiveConstructor(sorts[1])
	astTestMust(t, err)
	if !found {
		t.Fatal("missing later constructor")
	}
	heldSort, err := m.NewRef(sorts[1])
	astTestMust(t, err)
	heldConstructor, err := m.NewRef(later)
	astTestMust(t, err)
	astTestMust(t, q.Del(&rational))
	astTestMust(t, scoped.Close())
	p := m.datatypePlugin
	u := p.utility
	if len(u.refs) != 8 {
		t.Fatal("actual cache/preference owners missing", len(u.refs))
	}
	laterSortCount, err := m.GetRefCount(sorts[1])
	astTestMust(t, err)
	laterConstructorCount, err := m.GetRefCount(later)
	astTestMust(t, err)
	beforeBlocks, beforeRefs := rt.Stats().LiveBlocks, m.Stats().LiveReferences
	astMpfFormatTLSNearMinimum(t, rt)
	// Rational-valued parameters are supported by source sort substitution.
	// Reach the actual plugin finalizer at its normal locked call boundary.
	// Definition owners are retired before the concrete cached utility.
	astTestMust(t, m.lock())
	stopped := m.finalizeDatatypePlugin()
	m.mu.Unlock()
	if !errors.Is(stopped, arena.ErrBounds) || rt.Stats().LiveBlocks >= beforeBlocks {
		t.Fatal("did not reach consumed NUM failure", stopped, rt.State())
	}
	if m.Stats().LiveReferences+4 != beforeRefs || p.membersEnded || p.utility != u || u.manager != m || len(u.refs) != 8 || len(p.definitionRefs) != 0 || p.definitions != nil || u.operation != 2 {
		t.Fatal("plugin lost its real utility continuation", m.Stats(), beforeRefs, u.refs)
	}
	for i, ref := range u.refs {
		if (ref == (ASTRef{})) != (i < 4) {
			t.Fatal("wrong consumed reference prefix", i, ref)
		}
	}
	// The later sort loses its definition-owned pin before utility teardown.
	laterSortCount--
	for _, entry := range []struct {
		h     ASTHandle
		count uint32
	}{{sorts[1], laterSortCount}, {later, laterConstructorCount}} {
		count, err := m.GetRefCount(entry.h)
		astTestMust(t, err)
		if count != entry.count {
			t.Fatal("failure consumed a later utility owner", count, entry.count)
		}
	}
	recovery, err := rt.Allocate(400000)
	astTestMust(t, err)
	state := rt.State()
	blocks := rt.Stats().LiveBlocks
	astTestMust(t, m.lock())
	retry := m.finalizeDatatypePlugin()
	m.mu.Unlock()
	astTestMust(t, retry)
	if rt.State() != state || rt.Stats().LiveBlocks != blocks || !p.membersEnded || p.utility != nil || u.manager != nil || len(u.refs) != 0 || m.Stats().LiveReferences != beforeRefs-8 {
		t.Fatal("utility retry replayed NUM deletion or retained source owners")
	}
	for _, entry := range []struct {
		h     ASTHandle
		count uint32
	}{{sorts[1], laterSortCount}, {later, laterConstructorCount}} {
		count, err := m.GetRefCount(entry.h)
		astTestMust(t, err)
		if count+2 != entry.count {
			t.Fatal("utility retry did not retire exactly its remaining cache/preference pins", count, entry.count)
		}
	}
	astTestMust(t, m.lock())
	again := m.finalizeDatatypePlugin()
	m.mu.Unlock()
	astTestMust(t, again)
	finish := func(close func() error) {
		t.Helper()
		for attempts := 0; attempts < 128; attempts++ {
			before := rt.Stats().LiveBlocks
			err := close()
			if err == nil {
				return
			}
			if !errors.Is(err, arena.ErrBounds) || rt.Stats().LiveBlocks >= before {
				t.Fatal("cleanup made no physical progress", err, rt.State())
			}
		}
		t.Fatal("cleanup horizon")
	}
	astTestMust(t, m.DelRef(&heldConstructor))
	astTestMust(t, m.DelRef(&heldSort))
	finish(m.Close)
	finish(q.Close)
	finish(symbols.Close)
	if err = rt.Deallocate(&recovery); err != nil && (!errors.Is(err, arena.ErrBounds) || recovery != (umemory.Block{})) {
		t.Fatal(err)
	}
	finish(rt.Close)
	if b.Used() != 0 {
		t.Fatal("fixture retained bytes", b.Used())
	}
}
