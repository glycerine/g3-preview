package smtcore

// Source: src/smt/theory_array_base.cpp:70,106,131,162,174,184,201,205.

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type smtArrayClosedTraceWriter struct{}

func (smtArrayClosedTraceWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func smtArrayTraceManager(t *testing.T, tags *diagnostic.TraceTags) *ast.ASTManager {
	t.Helper()
	symbols, numerals, budget := astTestOwners(t)
	manager, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, manager.Close()) })
	return manager
}

func TestSMTArrayStoreReadSourceTracesAndRefusal(t *testing.T) {
	for _, closed := range []bool{false, true} {
		t.Run(map[bool]string{false: "output", true: "closed"}[closed], func(t *testing.T) {
			var output bytes.Buffer
			var writer io.Writer = &output
			if closed {
				writer = smtArrayClosedTraceWriter{}
			}
			tags := diagnostic.NewTraceTagsWithWriter(writer)
			for _, tag := range []string{"array", "array_axiom", "array_bug"} {
				tags.Enable(tag)
			}
			manager := smtArrayTraceManager(t, tags)
			family, err := manager.RegisterArrayPlugin()
			astTestMust(t, err)
			_, _, state := smtInternalizerFixture(t, manager, nil)
			theory, err := NewSMTArrayTheory(family)
			astTestMust(t, err)
			astTestMust(t, state.RegisterTheory(theory))
			indexSort, indices := smtArrayQueueIndices(t, manager, "array-trace-index-", 3)
			boolean, err := manager.MkBoolSort()
			astTestMust(t, err)
			arraySort, err := manager.MkArraySort(ast.SliceArguments([]ast.ASTHandle{indexSort, indexSort}), boolean)
			astTestMust(t, err)
			base, err := manager.MkConst(astTestSymbol(t, manager, "array-trace-base"), arraySort)
			astTestMust(t, err)
			value, err := manager.MkTrue()
			astTestMust(t, err)
			store, err := manager.MkFamilyApp(family, ast.ASTArrayOpStore, nil, []ast.ASTHandle{base, indices[0], indices[1], value})
			astTestMust(t, err)
			astTestMust(t, state.Internalize(store, false))
			read, err := manager.MkFamilyApp(family, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{store, indices[0], indices[2]})
			astTestMust(t, err)
			astTestMust(t, state.Internalize(read, true))
			astTestMust(t, state.markRelevant(read))
			astTestMust(t, state.propagateTheoryRelevancy())
			if added, err := theory.queueStoreReadAxiom(read, store, false); err != nil || added {
				t.Fatal("duplicate store/read fingerprint", added, err)
			}
			astTestMust(t, theory.Propagate())
			if closed {
				return
			}
			if !tags.ShouldTrace("array") {
				if output.Len() != 0 {
					t.Fatal("compiled-out array trace executed", output.String())
				}
				return
			}
			trace := output.String()
			for _, want := range []string{
				"[array] assert_store_axiom2 src/smt/theory_array_base.cpp:201",
				"[array] assert_store_axiom2 src/smt/theory_array_base.cpp:205",
				"[array] assert_store_axiom1_core src/smt/theory_array_base.cpp:106",
				"[array] assert_store_axiom2_core src/smt/theory_array_base.cpp:131",
				"[array_bug] assert_store_axiom2_core src/smt/theory_array_base.cpp:162",
				"[array] assert_store_axiom2_core src/smt/theory_array_base.cpp:174",
				"[array] assert_store_axiom2_core src/smt/theory_array_base.cpp:184",
				"[array_axiom] assert_axiom src/smt/theory_array_base.cpp:70",
			} {
				if !strings.Contains(trace, want) {
					t.Fatalf("missing source array trace %q in %q", want, trace)
				}
			}
		})
	}
}
