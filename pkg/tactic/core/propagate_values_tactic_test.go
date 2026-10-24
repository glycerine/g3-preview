package core

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestPropagateValuesQuantifierRootsAndSharedGroundAtoms(t *testing.T) {
	for _, shared := range []bool{false, true} {
		t.Run(fmt.Sprint(shared), func(t *testing.T) {
			m, c, g := simplifyFixture(t)
			runner, err := NewPropagateValuesTactic(m, c, g, nil)
			simplifyMust(t, err)
			t.Cleanup(func() { simplifyMust(t, runner.Close()) })
			a, _, _ := propagateAtoms(t, m)
			boolean, err := m.MkBoolSort()
			simplifyMust(t, err)
			variable, err := m.MkVar(0, boolean)
			simplifyMust(t, err)
			body := variable
			if shared {
				body, err = m.MkOr(a, variable)
				simplifyMust(t, err)
			}
			name, err := m.Symbols.Intern("x")
			simplifyMust(t, err)
			quantifier, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolean}, Names: []usymbol.Symbol{name}, Body: body, Weight: 1})
			simplifyMust(t, err)
			goal, err := tactic.NewGoal(m, false, true, false)
			simplifyMust(t, err)
			t.Cleanup(func() { simplifyMust(t, goal.Close()) })
			if shared {
				simplifyMust(t, goal.Assert(a))
			}
			simplifyMust(t, goal.Assert(quantifier))
			var result tactic.TacticResult
			t.Cleanup(func() { simplifyMust(t, result.Close()) })
			simplifyMust(t, runner.Apply(goal, &result))
			size, err := goal.Size()
			simplifyMust(t, err)
			if size != 1 {
				t.Fatal("unexpected quantified propagation result", size)
			}
			remaining, err := goal.Form(0)
			simplifyMust(t, err)
			want := quantifier
			if shared {
				want = a
			}
			if remaining != want {
				t.Fatal("quantifier root or shared ground substitution was mishandled", remaining, want)
			}
		})
	}
}

func propagateAtoms(t *testing.T, m *ast.ASTManager) (ast.ASTHandle, ast.ASTHandle, ast.ASTHandle) {
	t.Helper()
	sort, err := m.MkBoolSort()
	simplifyMust(t, err)
	var terms []ast.ASTHandle
	for _, name := range []string{"a", "b"} {
		symbol, err := m.Symbols.Intern(name)
		simplifyMust(t, err)
		term, err := m.MkConst(symbol, sort)
		simplifyMust(t, err)
		simplifyPin(t, m, term)
		terms = append(terms, term)
	}
	notB, err := m.MkNot(terms[1])
	simplifyMust(t, err)
	simplifyPin(t, m, notB)
	return terms[0], terms[1], notB
}

func TestPropagateValuesDefaultRoundsTranslationAndParameterSnapshot(t *testing.T) {
	m, c, g := simplifyFixture(t)
	settings, err := c.NewValue()
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, c.Del(&settings)) })
	key := "max_rounds"
	simplifyMust(t, c.SetUIntName(&settings, &key, 0))
	source, err := NewPropagateValuesTactic(m, c, g, &settings)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, source.Close()) })
	target, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: m.Symbols, Numerals: m.NumeralService(), Budget: m.Budget()})
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, target.Close()) })
	translated, err := source.Translate(target)
	simplifyMust(t, err)
	simplifyMust(t, translated.IncRef())
	t.Cleanup(func() { simplifyMust(t, translated.DecRef()) })
	simplifyMust(t, c.SetUIntName(&settings, &key, 4))
	simplifyMust(t, source.UpdateParams(&settings))
	simplifyMust(t, source.Close())
	a, b, notB := propagateAtoms(t, target)
	sort, err := target.MkBoolSort()
	simplifyMust(t, err)
	name, err := target.Symbols.Intern("c")
	simplifyMust(t, err)
	other, err := target.MkConst(name, sort)
	simplifyMust(t, err)
	simplifyPin(t, target, other)
	first, err := target.MkOr(a, b)
	simplifyMust(t, err)
	simplifyPin(t, target, first)
	notA, err := target.MkNot(a)
	simplifyMust(t, err)
	second, err := target.MkOr(notA, other)
	simplifyMust(t, err)
	simplifyPin(t, target, second)
	goal, err := tactic.NewGoal(target, false, true, false)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, goal.Close()) })
	simplifyMust(t, goal.Assert(first))
	simplifyMust(t, goal.Assert(second))
	simplifyMust(t, goal.Assert(notB))
	var result tactic.TacticResult
	t.Cleanup(func() { simplifyMust(t, result.Close()) })
	simplifyMust(t, translated.Apply(goal, &result))
	got, err := goal.Form(0)
	simplifyMust(t, err)
	if got != first {
		t.Fatal("source update changed the translated zero-round parameter snapshot")
	}
	simplifyMust(t, c.SetUIntName(&settings, &key, 2))
	simplifyMust(t, translated.UpdateParams(&settings))
	simplifyMust(t, translated.Cleanup())
	fresh, err := tactic.NewGoal(target, false, true, false)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, fresh.Close()) })
	simplifyMust(t, fresh.Assert(first))
	simplifyMust(t, fresh.Assert(second))
	simplifyMust(t, fresh.Assert(notB))
	simplifyMust(t, translated.Apply(fresh, &result))
	got, err = fresh.Form(1)
	simplifyMust(t, err)
	if got != second {
		t.Fatal("two rounds propagated beyond the native directional boundary")
	}
	empty, err := c.NewValue()
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, c.Del(&empty)) })
	simplifyMust(t, translated.UpdateParams(&empty))
	simplifyMust(t, translated.Cleanup())
	if translated.(*PropagateValuesTactic).maxRounds != 4 {
		t.Fatal("native constructor/update default is four rounds despite the descriptor text")
	}
	simplifyMust(t, translated.Apply(goal, &result))
	got, err = goal.Form(1)
	simplifyMust(t, err)
	if got != other {
		t.Fatal("default rounds did not propagate the backward result forward again")
	}
	if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
		command := exec.Command(reference, "-in")
		command.Stdin = strings.NewReader("(set-logic QF_UF)(declare-const a Bool)(declare-const b Bool)(declare-const c Bool)(assert (or a b))(assert (or (not a) c))(assert (not b))(apply propagate-values)")
		native, err := command.CombinedOutput()
		if err != nil {
			t.Fatal(err, string(native))
		}
		tokens := func(s string) string {
			return strings.Join(strings.Fields(strings.NewReplacer("(", " ( ", ")", " ) ").Replace(s)), " ")
		}
		want := "(goals (goal a c (not b) :precision precise :depth 1))"
		if tokens(string(native)) != tokens(want) {
			t.Fatal("native default-round result changed", string(native))
		}
	}
}

func TestPropagateValuesEndTraceAndStreamRefusal(t *testing.T) {
	m, c, g := simplifyFixture(t)
	settings, err := c.NewValue()
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, c.Del(&settings)) })
	key := "max_rounds"
	simplifyMust(t, c.SetUIntName(&settings, &key, 0))
	runner, err := NewPropagateValuesTactic(m, c, g, &settings)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, runner.Close()) })
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("propagate_values")
	tags.Enable("propagate_values_core")
	simplifyMust(t, runner.BindTrace(tags))
	a, _, _ := propagateAtoms(t, m)
	goal, err := tactic.NewGoal(m, false, true, false)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, goal.Close()) })
	simplifyMust(t, goal.Assert(a))
	var result tactic.TacticResult
	t.Cleanup(func() { simplifyMust(t, result.Close()) })
	simplifyMust(t, runner.Apply(goal, &result))
	if !tags.ShouldTrace("propagate_values") {
		if output.Len() != 0 {
			t.Fatal("erased propagation traces wrote output")
		}
	} else {
		record := func(tag string, line int, body string) string {
			return fmt.Sprintf("-------- [%s] operator() src/tactic/core/propagate_values_tactic.cpp:%d ---------\n%s------------------------------------------------\n", tag, line, body)
		}
		want := record("propagate_values", 211, "end\n(goal\n  a)\n") + record("propagate_values_core", 212, "(goal\n  |-\n  a\n  :precision precise :depth 1)\n")
		if output.String() != want {
			t.Fatalf("propagation end trace differs from source\nGo: %q\nwant: %q", output.String(), want)
		}
	}
	refused := diagnostic.NewTraceTagsWithWriter(simplifyRefusedWriter{})
	refused.EnableAll(true)
	simplifyMust(t, runner.BindTrace(refused))
	simplifyMust(t, runner.Apply(goal, &result))
	copy := *runner
	if err = copy.Close(); err == nil {
		t.Fatal("copied tactic accepted ownership of its original implementation")
	}
	size, err := goal.Size()
	simplifyMust(t, err)
	if size != 1 {
		t.Fatal("trace refusal altered the goal")
	}
}

func TestPropagateValuesDirectionalRoundsAndCPPApply(t *testing.T) {
	for _, rounds := range []uint32{0, 1, 2, 4} {
		t.Run(fmt.Sprint(rounds), func(t *testing.T) {
			m, c, g := simplifyFixture(t)
			settings, err := c.NewValue()
			simplifyMust(t, err)
			t.Cleanup(func() { simplifyMust(t, c.Del(&settings)) })
			key := "max_rounds"
			simplifyMust(t, c.SetUIntName(&settings, &key, rounds))
			runner, err := NewPropagateValuesTactic(m, c, g, &settings)
			simplifyMust(t, err)
			t.Cleanup(func() { simplifyMust(t, runner.Close()) })
			a, b, notB := propagateAtoms(t, m)
			clause, err := m.MkOr(a, b)
			simplifyMust(t, err)
			goal, err := tactic.NewGoal(m, false, true, false)
			simplifyMust(t, err)
			t.Cleanup(func() { simplifyMust(t, goal.Close()) })
			simplifyMust(t, goal.Assert(clause))
			simplifyMust(t, goal.Assert(notB))
			var result tactic.TacticResult
			t.Cleanup(func() { simplifyMust(t, result.Close()) })
			simplifyMust(t, runner.Apply(goal, &result))
			first, err := goal.Form(0)
			simplifyMust(t, err)
			want := clause
			if rounds >= 2 {
				want = a
			}
			if first != want {
				t.Fatal("directional round boundary differs from C++", rounds, first, want)
			}
			second, err := goal.Form(1)
			simplifyMust(t, err)
			if second != notB {
				t.Fatal("propagation dropped its defining assertion")
			}
			if result.ModelConverter != nil || result.ProofConverter != nil || result.Core != (ast.ASTDependency{}) {
				t.Fatal("propagation introduced an unnecessary converter or separate core")
			}
			if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
				var goOutput bytes.Buffer
				goOutput.WriteString("(goals (goal ")
				printer, err := ast.NewLegacySMTPrinter(m)
				simplifyMust(t, err)
				size, err := goal.Size()
				simplifyMust(t, err)
				for i := uint32(0); i < size; i++ {
					term, err := goal.Form(i)
					simplifyMust(t, err)
					simplifyMust(t, printer.DisplayExpr(&goOutput, term, nil))
					goOutput.WriteByte(' ')
				}
				depth, err := goal.Depth()
				simplifyMust(t, err)
				fmt.Fprintf(&goOutput, ":precision precise :depth %d))", depth)
				command := exec.Command(reference, "-in")
				command.Stdin = strings.NewReader(fmt.Sprintf("(set-logic QF_UF)(declare-const a Bool)(declare-const b Bool)(assert (or a b))(assert (not b))(apply (using-params propagate-values :max_rounds %d))", rounds))
				native, err := command.CombinedOutput()
				if err != nil {
					t.Fatal(err, string(native))
				}
				tokens := func(s string) string {
					return strings.Join(strings.Fields(strings.NewReplacer("(", " ( ", ")", " ) ").Replace(s)), " ")
				}
				if tokens(goOutput.String()) != tokens(string(native)) {
					t.Fatalf("propagate-values differs from C++\nGo: %s\nC++: %s", goOutput.String(), native)
				}
			}
		})
	}
}

func TestPropagateValuesCarriesProofAndAllUsedDependencies(t *testing.T) {
	m, c, g := simplifyFixture(t)
	simplifyMust(t, m.ToggleProofMode(ast.ASTProofEnabled))
	runner, err := NewPropagateValuesTactic(m, c, g, nil)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, runner.Close()) })
	a, b, notB := propagateAtoms(t, m)
	notA, err := m.MkNot(a)
	simplifyMust(t, err)
	imp, err := m.MkOr(notA, b)
	simplifyMust(t, err)
	goal, err := tactic.NewGoal(m, true, true, true)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, goal.Close()) })
	simplifyMust(t, goal.SetDepth(7))
	var tags []ast.ASTHandle
	for i, term := range []ast.ASTHandle{a, imp, notB} {
		sort, err := m.MkBoolSort()
		simplifyMust(t, err)
		name, err := m.Symbols.Intern(fmt.Sprintf("propagate-tag-%d", i))
		simplifyMust(t, err)
		tag, err := m.MkConst(name, sort)
		simplifyMust(t, err)
		simplifyPin(t, m, tag)
		tags = append(tags, tag)
		dep, err := m.DependencyLeaf(tag)
		simplifyMust(t, err)
		proof, err := m.MkProofAsserted(term)
		simplifyMust(t, err)
		simplifyMust(t, goal.AssertWithProof(term, proof, dep))
	}
	var result tactic.TacticResult
	t.Cleanup(func() { simplifyMust(t, result.Close()) })
	simplifyMust(t, runner.Apply(goal, &result))
	if !goal.IsDecidedUnsat() {
		t.Fatal("propagation did not derive the contradiction")
	}
	simplifyMust(t, runner.Cleanup())
	proof, err := goal.Proof(0)
	simplifyMust(t, err)
	fact, err := m.GetProofFact(proof)
	simplifyMust(t, err)
	if !m.IsFalse(fact) {
		t.Fatal("propagation did not retain a proof of false")
	}
	dep, err := goal.Dependency(0)
	simplifyMust(t, err)
	leaves, err := m.DependencyLinearize(dep, nil)
	simplifyMust(t, err)
	if len(leaves) != 3 {
		t.Fatal("propagation lost a transitive dependency", leaves)
	}
	for _, want := range tags {
		found := false
		for _, got := range leaves {
			found = found || got == want
		}
		if !found {
			t.Fatal("missing original assertion tag", want, leaves)
		}
	}
	depth, err := goal.Depth()
	simplifyMust(t, err)
	if depth != 8 {
		t.Fatal("propagation goal depth", depth)
	}
}

func TestPropagateValuesRewriteFailureCleanupAndRecovery(t *testing.T) {
	m, c, g := simplifyFixture(t)
	settings, err := c.NewValue()
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, c.Del(&settings)) })
	key := "max_steps"
	simplifyMust(t, c.SetUIntName(&settings, &key, 0))
	runner, err := NewPropagateValuesTactic(m, c, g, &settings)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, runner.Close()) })
	a, b, _ := propagateAtoms(t, m)
	notA, err := m.MkNot(a)
	simplifyMust(t, err)
	clause, err := m.MkOr(notA, b)
	simplifyMust(t, err)
	goal, err := tactic.NewGoal(m, false, true, false)
	simplifyMust(t, err)
	t.Cleanup(func() { simplifyMust(t, goal.Close()) })
	simplifyMust(t, goal.Assert(a))
	simplifyMust(t, goal.Assert(clause))
	var result tactic.TacticResult
	t.Cleanup(func() { simplifyMust(t, result.Close()) })
	err = runner.Apply(goal, &result)
	var exception tactic.TacticException
	if !errors.As(err, &exception) || exception.Message() != tactic.TacticMaxStepsMessage {
		t.Fatal("rewriter failure did not become a tactic exception", err)
	}
	simplifyMust(t, runner.Cleanup())
	simplifyMust(t, c.SetUIntName(&settings, &key, ^uint32(0)))
	simplifyMust(t, runner.UpdateParams(&settings))
	simplifyMust(t, runner.Apply(goal, &result))
	second, err := goal.Form(1)
	simplifyMust(t, err)
	if second != b {
		t.Fatal("propagation did not recover after replacing its failed rewriter", second)
	}
}

func TestPropagateValuesNumericEqualityBothOrientations(t *testing.T) {
	for _, leftValue := range []bool{false, true} {
		t.Run(fmt.Sprint(leftValue), func(t *testing.T) {
			m, c, g := simplifyFixture(t)
			runner, err := NewPropagateValuesTactic(m, c, g, nil)
			simplifyMust(t, err)
			t.Cleanup(func() { simplifyMust(t, runner.Close()) })
			five, err := m.ArithmeticIntegerNumeralText("5")
			simplifyMust(t, err)
			simplifyPin(t, m, five)
			zero, err := m.ArithmeticIntegerNumeralText("0")
			simplifyMust(t, err)
			simplifyPin(t, m, zero)
			sort, err := m.GetSort(five)
			simplifyMust(t, err)
			name, err := m.Symbols.Intern("x")
			simplifyMust(t, err)
			x, err := m.MkConst(name, sort)
			simplifyMust(t, err)
			simplifyPin(t, m, x)
			eqZero, err := m.MkEq(x, zero)
			simplifyMust(t, err)
			notZero, err := m.MkNot(eqZero)
			simplifyMust(t, err)
			var defining ast.ASTHandle
			if leftValue {
				defining, err = m.MkEq(five, x)
			} else {
				defining, err = m.MkEq(x, five)
			}
			simplifyMust(t, err)
			goal, err := tactic.NewGoal(m, false, true, false)
			simplifyMust(t, err)
			t.Cleanup(func() { simplifyMust(t, goal.Close()) })
			simplifyMust(t, goal.Assert(notZero))
			simplifyMust(t, goal.Assert(defining))
			var result tactic.TacticResult
			t.Cleanup(func() { simplifyMust(t, result.Close()) })
			simplifyMust(t, runner.Apply(goal, &result))
			size, err := goal.Size()
			simplifyMust(t, err)
			if size != 1 {
				t.Fatal("numeric value was not propagated", size)
			}
			remaining, err := goal.Form(0)
			simplifyMust(t, err)
			if remaining != defining {
				t.Fatal("value propagation changed its original definition's orientation")
			}
			if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
				definition := "(= x 5)"
				if leftValue {
					definition = "(= 5 x)"
				}
				command := exec.Command(reference, "-in")
				command.Stdin = strings.NewReader("(set-logic QF_LIA)(declare-const x Int)(assert (not (= x 0)))(assert " + definition + ")(apply propagate-values)")
				native, err := command.CombinedOutput()
				if err != nil {
					t.Fatal(err, string(native))
				}
				var output bytes.Buffer
				printer, err := ast.NewLegacySMTPrinter(m)
				simplifyMust(t, err)
				output.WriteString("(goals (goal ")
				simplifyMust(t, printer.DisplayExpr(&output, remaining, nil))
				output.WriteString(" :precision precise :depth 1))")
				tokens := func(s string) string {
					return strings.Join(strings.Fields(strings.NewReplacer("(", " ( ", ")", " ) ").Replace(s)), " ")
				}
				if tokens(output.String()) != tokens(string(native)) {
					t.Fatalf("numeric propagation differs from C++\nGo: %s\nC++: %s", output.String(), native)
				}
			}
		})
	}
}
