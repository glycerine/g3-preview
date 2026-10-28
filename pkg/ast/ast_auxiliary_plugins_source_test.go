package ast

// Source: src/ast/dl_decl_plugin.cpp:31-48; src/ast/pb_decl_plugin.cpp:23-29;
// src/ast/reg_decl_plugins.cpp:30-55; src/ast/ast.cpp:1635-1638.

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestASTDatalogAndPseudoBooleanActualConstructorMembers(t *testing.T) {
	m := astTestManager(t)
	dl, e := m.RegisterDatalogPlugin()
	astTestMust(t, e)
	pb, e := m.RegisterPseudoBooleanPlugin()
	astTestMust(t, e)
	if dl == pb || !m.pluginAt(dl).present || !m.pluginAt(pb).present {
		t.Fatal("actual independent registration")
	}
	for _, tc := range []struct {
		kind  astPluginKind
		names []string
	}{
		{astDatalogPlugin, []string{"store", "empty", "is_empty", "join", "union", "widen", "project", "filter", "negation_filter", "rename", "complement", "select", "clone", "N", "<", "<=", "R"}},
		{astPseudoBooleanPlugin, []string{"at-most", "at-least", "pble", "pbge", "pbeq"}},
	} {
		p := m.auxiliaryPlugin(tc.kind)
		if p == nil || !p.completed || !p.published || p.initialized != uint32(len(tc.names)) {
			t.Fatal("constructor completion/publication", p)
		}
		for i, name := range tc.names {
			want, e := m.Symbols.Intern(name)
			astTestMust(t, e)
			if p.symbols[i] != want {
				t.Fatal("literal member order", i, name)
			}
		}
	}
}

func TestASTAuxiliaryPluginEveryConstructorPrefix(t *testing.T) {
	for _, kind := range []astPluginKind{astDatalogPlugin, astPseudoBooleanPlugin} {
		success := false
		for ordinal := int64(0); ordinal < 128; ordinal++ {
			b := arena.NewBudget(1 << 28)
			r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64})
			astTestMust(t, e)
			q, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
			astTestMust(t, e)
			sy, e := usymbol.NewManager(usymbol.Config{Budget: b})
			astTestMust(t, e)
			m, e := NewASTManager(ASTManagerConfig{Symbols: sy, Numerals: q, Budget: b})
			astTestMust(t, e)
			fixed := b.Used() - m.Stats().AllocatedBytes - astTestSymbolGraph(sy) - r.Stats().TotalBytes
			b.FailAfter(ordinal)
			var call error
			if kind == astDatalogPlugin {
				_, call = m.RegisterDatalogPlugin()
			} else {
				_, call = m.RegisterPseudoBooleanPlugin()
			}
			astFloatingPointConstructorRawGraph(t, q, nil)
			if b.Used() != fixed+m.Stats().AllocatedBytes+astTestSymbolGraph(sy)+r.Stats().TotalBytes {
				t.Fatal("immediate actual cold ownership", kind, ordinal, b.Used())
			}
			for p := m.auxiliaryPlugins; p != nil; p = p.next {
				if p.self != p || p.owner != m || p.header == 0 || p.initialized > 17 {
					t.Fatal("actual acquired prefix", p)
				}
				if call != nil && p.bodyCause == nil {
					t.Fatal("source constructor cause missing", kind, ordinal)
				}
			}
			b.FailAfter(-1)
			astTestMust(t, m.Close())
			astTestMust(t, sy.Close())
			astTestMust(t, q.Close())
			astTestMust(t, r.Close())
			if b.Used() != 0 {
				t.Fatal("exact context end", kind, ordinal, b.Used())
			}
			if call == nil {
				success = true
				t.Log("literal constructor first-success", kind, ordinal)
				break
			}
		}
		if !success {
			t.Fatal("constructor horizon", kind)
		}
	}
}
