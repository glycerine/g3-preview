package ast

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	udiag "github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestASTExprSubstitutionOptionalMapMemoryWrappers(t *testing.T) {
	symbols, numerals, _ := astTestOwners(t)
	budget := arena.NewBudget(1 << 28)
	var output bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	defer func() {
		astTestMust(t, m.Close())
		if budget.Used() != 0 {
			t.Fatal("substitution owner retained", budget.Used())
		}
	}()
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	tags.Enable("memory")
	plain, err := m.NewExprSubstitutionPolicy(false, false)
	astTestMust(t, err)
	if state := m.exprSubstitutions[plain.record]; state.proof != nil || state.dep != nil {
		t.Fatal("source disabled optional maps")
	}
	astTestMust(t, m.DelExprSubstitution(&plain))
	if output.Len() != 0 {
		t.Fatal("inline source substitution has no optional map wrapper", output.String())
	}
	s, err := m.NewExprSubstitutionPolicy(true, true)
	astTestMust(t, err)
	state := m.exprSubstitutions[s.record]
	if state.proof == nil || state.dep == nil || state.proof == state.dep {
		t.Fatal("distinct source optional map owners")
	}
	proof, dep := state.proof, state.dep
	astTestMust(t, m.ExprSubstitutionCleanup(s)) // Keeps both map owners alive.
	astTestMust(t, m.DelExprSubstitution(&s))
	got := output.String()
	if !udiag.DebugBuild {
		if got != "" {
			t.Fatal("source Z3DEBUG wrappers must erase", got)
		}
		return
	}
	if strings.Count(got, "[memory] allocate ") != 2 || strings.Count(got, "[memory] deallocate ") != 2 {
		t.Fatal("source optional map lifetime records", got)
	}
	size := strconv.FormatUint(uint64(reflect.TypeOf(astExprSubstTable{}).Size()), 10)
	parts := []string{
		"alloc " + fmt.Sprintf("%p", proof) + " src/ast/expr_substitution.cpp:29 expr2proof " + size + "\n",
		"alloc " + fmt.Sprintf("%p", dep) + " src/ast/expr_substitution.cpp:31 expr2expr_dependency " + size + "\n",
		"dealloc " + fmt.Sprintf("%p", dep) + " src/util/util.h:224\n",
		"dealloc " + fmt.Sprintf("%p", proof) + " src/util/util.h:224\n",
	}
	previous := -1
	for _, part := range parts {
		at := strings.Index(got, part)
		if at <= previous || strings.Count(got, part) != 1 {
			t.Fatalf("source optional map order for %q in %q", part, got)
		}
		previous = at
	}
}

func TestASTExprSubstitutionSecondOptionalMapRefusalUnwindsFirst(t *testing.T) {
	symbols, numerals, _ := astTestOwners(t)
	budget := arena.NewBudget(1 << 28)
	var output bytes.Buffer
	tags := udiag.NewTraceTagsWithWriter(&output)
	m, err := NewASTManager(ASTManagerConfig{Symbols: symbols, Numerals: numerals, Budget: budget, TraceTags: tags})
	astTestMust(t, err)
	defer func() { astTestMust(t, m.Close()) }()
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	// Warm the source record allocator, then probe the actual reservation
	// prefixes because checked builds may charge additional nested storage.
	warm, err := m.NewExprSubstitutionPolicy(false, false)
	astTestMust(t, err)
	astTestMust(t, m.DelExprSubstitution(&warm))
	tags.Enable("memory")
	seenFailure, found := false, false
	for stop := int64(0); stop < 12; stop++ {
		output.Reset()
		used := budget.Used()
		budget.FailAfter(stop)
		s, e := m.NewExprSubstitutionPolicy(true, true)
		budget.FailAfter(-1)
		if e == nil {
			astTestMust(t, m.DelExprSubstitution(&s))
		} else {
			seenFailure = true
		}
		if len(m.exprSubstitutions) != 0 || budget.Used() != used {
			t.Fatal("constructor prefix retained owner storage", stop, e, len(m.exprSubstitutions), budget.Used(), used)
		}
		got := output.String()
		if !udiag.DebugBuild {
			if got != "" {
				t.Fatal("erased memory wrapper after refusal", got)
			}
			continue
		}
		if e != nil && strings.Count(got, "[memory] allocate ") == 1 &&
			strings.Count(got, "[memory] deallocate ") == 1 &&
			strings.Contains(got, "src/ast/expr_substitution.cpp:29 expr2proof ") &&
			!strings.Contains(got, "src/ast/expr_substitution.cpp:31 expr2expr_dependency ") &&
			strings.Contains(got, "src/util/util.h:224\n") {
			found = true
			break
		}
	}
	if !seenFailure || udiag.DebugBuild && !found {
		t.Fatal("missing completed proof-map unwind at a dependency-map refusal")
	}
}
