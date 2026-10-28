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
			for _, tag := range []string{"array", "array_axiom", "array_bug", "array_map_bug", "mk_var_bug"} {
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
			function, err := manager.MkFuncDecl(astTestSymbol(t, manager, "array-trace-function"), []ast.ASTHandle{indexSort, indexSort}, boolean, ast.ASTFuncDeclInfo{})
			astTestMust(t, err)
			asArray, err := manager.MkAsArray(function)
			astTestMust(t, err)
			astTestMust(t, state.Internalize(asArray, false))
			if !theory.unsupported {
				t.Fatal("as-array did not retain the unsupported-operation marker")
			}
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
				"[mk_var_bug] mk_select src/smt/theory_array_base.cpp:45",
				"[array] assert_store_axiom1_core src/smt/theory_array_base.cpp:106",
				"[mk_var_bug] assert_store_axiom1_core src/smt/theory_array_base.cpp:113",
				"[array] assert_store_axiom2_core src/smt/theory_array_base.cpp:131",
				"[array_bug] assert_store_axiom2_core src/smt/theory_array_base.cpp:162",
				"[array] assert_store_axiom2_core src/smt/theory_array_base.cpp:174",
				"[array] assert_store_axiom2_core src/smt/theory_array_base.cpp:184",
				"[array_map_bug] assert_store_axiom2_core src/smt/theory_array_base.cpp:185",
				"[array_axiom] assert_axiom src/smt/theory_array_base.cpp:70",
				"[array] found_unsupported_op src/smt/theory_array_base.cpp:36",
			} {
				if !strings.Contains(trace, want) {
					t.Fatalf("missing source array trace %q in %q", want, trace)
				}
			}
		})
	}
}

func TestSMTArrayExtensionalityAndSharedSourceTraces(t *testing.T) {
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	for _, tag := range []string{"array_ext", "ext", "array_axiom", "array_shared", "array_bug"} {
		tags.Enable(tag)
	}
	manager := smtArrayTraceManager(t, tags)
	family, err := manager.RegisterArrayPlugin()
	astTestMust(t, err)
	_, _, state := smtInternalizerFixture(t, manager, nil)
	theory, err := NewSMTArrayTheory(family)
	astTestMust(t, err)
	astTestMust(t, state.RegisterTheory(theory))
	boolean, err := manager.MkBoolSort()
	astTestMust(t, err)
	arraySort, err := manager.MkArraySort(ast.SliceArguments([]ast.ASTHandle{boolean}), boolean)
	astTestMust(t, err)

	a, err := manager.MkConst(astTestSymbol(t, manager, "array-ext-trace-a"), arraySort)
	astTestMust(t, err)
	b, err := manager.MkConst(astTestSymbol(t, manager, "array-ext-trace-b"), arraySort)
	astTestMust(t, err)
	index, err := manager.MkTrue()
	astTestMust(t, err)
	left, err := manager.MkFamilyApp(family, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{a, index})
	astTestMust(t, err)
	right, err := manager.MkFamilyApp(family, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{b, index})
	astTestMust(t, err)
	readEquality, err := theory.MkEqualityLiteral(left, right, true)
	astTestMust(t, err)
	for _, expression := range []ast.ASTHandle{left, right, state.boolExpr[readEquality.Var()]} {
		astTestMust(t, state.markRelevant(expression))
	}
	astTestMust(t, state.AssignLiteral(readEquality.Negated(), NewSMTBooleanAxiom()))
	_, err = state.PropagateCore()
	astTestMust(t, err)
	arrayEquality, err := theory.MkEqualityLiteral(a, b, true)
	astTestMust(t, err)
	astTestMust(t, state.AssignLiteral(arrayEquality.Negated(), NewSMTBooleanAxiom()))
	_, err = state.PropagateCore()
	astTestMust(t, err)

	c, err := manager.MkConst(astTestSymbol(t, manager, "array-ext-trace-c"), arraySort)
	astTestMust(t, err)
	d, err := manager.MkConst(astTestSymbol(t, manager, "array-ext-trace-d"), arraySort)
	astTestMust(t, err)
	witnessEquality, err := theory.MkEqualityLiteral(c, d, true)
	astTestMust(t, err)
	astTestMust(t, state.AssignLiteral(witnessEquality.Negated(), NewSMTBooleanAxiom()))
	astTestMust(t, state.PropagateTheoryAssignments())
	astTestMust(t, state.PropagateTheoryEvents())
	astTestMust(t, theory.Propagate())

	outerSort, err := manager.MkArraySort(ast.SliceArguments([]ast.ASTHandle{arraySort}), boolean)
	astTestMust(t, err)
	outer, err := manager.MkConst(astTestSymbol(t, manager, "array-shared-trace-outer"), outerSort)
	astTestMust(t, err)
	shared := make([]ast.ASTHandle, 2)
	for i, name := range []string{"array-shared-trace-e", "array-shared-trace-f"} {
		shared[i], err = manager.MkConst(astTestSymbol(t, manager, name), arraySort)
		astTestMust(t, err)
		innerRead, err := manager.MkFamilyApp(family, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{shared[i], index})
		astTestMust(t, err)
		astTestMust(t, state.Internalize(innerRead, false))
		outerRead, err := manager.MkFamilyApp(family, ast.ASTArrayOpSelect, nil, []ast.ASTHandle{outer, shared[i]})
		astTestMust(t, err)
		astTestMust(t, state.Internalize(outerRead, false))
		astTestMust(t, state.markRelevant(shared[i]))
	}
	progress, err := theory.makeInterfaceEqualities()
	astTestMust(t, err)
	if !progress {
		t.Fatal("shared array trace fixture created no interface equality")
	}
	if !tags.ShouldTrace("ext") {
		if output.Len() != 0 {
			t.Fatal("compiled-out array traces executed", output.String())
		}
		return
	}
	trace := output.String()
	for _, want := range []string{
		"[array_ext] already_diseq src/smt/theory_array_base.cpp:281",
		"[ext] assert_extensionality_core src/smt/theory_array_base.cpp:325",
		"[array_shared] collect_shared_vars src/smt/theory_array_base.cpp:422",
		"[array_shared] collect_shared_vars src/smt/theory_array_base.cpp:432",
		"[array_bug] mk_interface_eqs src/smt/theory_array_base.cpp:459",
	} {
		if !strings.Contains(trace, want) {
			t.Fatalf("missing source array trace %q in %q", want, trace)
		}
	}
}

func TestSMTArrayModelDefaultAndMergeSourceTraces(t *testing.T) {
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("array")
	manager := smtArrayTraceManager(t, tags)
	family, err := manager.RegisterArrayPlugin()
	astTestMust(t, err)
	context, _, state := smtInternalizerFixture(t, manager, nil)
	theory, err := NewSMTArrayTheory(family)
	astTestMust(t, err)
	astTestMust(t, state.RegisterTheory(theory))
	boolean, err := manager.MkBoolSort()
	astTestMust(t, err)
	arraySort, err := manager.MkArraySort(ast.SliceArguments([]ast.ASTHandle{boolean}), boolean)
	astTestMust(t, err)
	value, err := manager.MkFalse()
	astTestMust(t, err)
	constant, err := manager.MkConstArray(arraySort, value)
	astTestMust(t, err)
	index, err := manager.MkTrue()
	astTestMust(t, err)
	store, err := manager.MkFamilyApp(family, ast.ASTArrayOpStore, nil, []ast.ASTHandle{constant, index, value})
	astTestMust(t, err)
	astTestMust(t, state.Internalize(store, false))
	for _, expression := range []ast.ASTHandle{constant, store} {
		astTestMust(t, state.markRelevant(expression))
	}
	generator := &SMTModelGenerator{manager: manager, context: context}
	astTestMust(t, theory.prepareArrayModel(generator))
	astTestMust(t, theory.FinalizeModel(generator))
	if !tags.ShouldTrace("array") {
		if output.Len() != 0 {
			t.Fatal("compiled-out array model traces executed", output.String())
		}
		return
	}
	trace := output.String()
	for _, want := range []string{
		"[array] set_default src/smt/theory_array_base.cpp:521",
		"[array] mg_merge src/smt/theory_array_base.cpp:568",
		"[array] collect_defaults src/smt/theory_array_base.cpp:646",
	} {
		if !strings.Contains(trace, want) {
			t.Fatalf("missing source array model trace %q in %q", want, trace)
		}
	}
}
