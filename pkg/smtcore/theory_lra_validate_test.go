package smtcore

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestSMTLRAValidatorsFreshDefaultContextsAndPolarity(t *testing.T) {
	f, ls, eqs := smtLemmaFixture(t)
	validWarm, errWarm := f.a.validateAssignment([]SMTLiteral{SMTTrueLiteral}, SMTTrueLiteral)
	astTestMust(t, errWarm)
	if !validWarm {
		t.Fatal("cache warm-up implication failed")
	}
	for _, tc := range []struct {
		core           []SMTLiteral
		lit            SMTLiteral
		conflict, want bool
	}{
		{nil, SMTFalseLiteral, true, false},
		{[]SMTLiteral{ls[0]}, SMTFalseLiteral, true, false},
		{[]SMTLiteral{ls[0], ls[0].Negated()}, SMTFalseLiteral, true, true},
		{[]SMTLiteral{ls[0]}, ls[0], false, true},
		{nil, ls[0], false, false},
		{[]SMTLiteral{ls[0].Negated()}, ls[0].Negated(), false, true},
		{nil, SMTFalseLiteral, false, false},
	} {
		var valid bool
		var err error
		before := append([]SMTLiteral(nil), tc.core...)
		parametersBefore := f.m.Stats().LiveParameters
		if tc.conflict {
			valid, err = f.a.validateConflict(tc.core)
		} else {
			valid, err = f.a.validateAssignment(tc.core, tc.lit)
		}
		astTestMust(t, err)
		if f.m.Stats().LiveParameters != parametersBefore {
			t.Fatal("fresh validation retained parameters", parametersBefore, f.m.Stats().LiveParameters)
		}
		lraValidationCPP(t, f.s, tc.core, tc.lit, tc.want, "")
		if valid != tc.want || f.a.validation != nil {
			t.Fatal(tc, valid, f.a.validation)
		}
		if len(tc.core) != len(before) {
			t.Fatal("temporary premise changed caller")
		}
		for i := range before {
			if before[i] != tc.core[i] {
				t.Fatal("premise overwritten")
			}
		}
	}
	eq := f.pin(f.m.MkEq(eqs[0].First.owner, eqs[0].Second.owner))
	astTestMust(t, f.s.Internalize(eq, true))
	lit, err := f.s.Literal(eq)
	astTestMust(t, err)
	for _, core := range [][]SMTLiteral{nil, {lit}} {
		valid, err := f.a.validateEquality(core, eqs[0].First, eqs[0].Second)
		astTestMust(t, err)
		lraValidationCPP(t, f.s, core, lit, len(core) != 0, "")
		if valid != (len(core) != 0) {
			t.Fatal("fresh equality validator inherited parent equalities", valid, core)
		}
	}
}

func TestSMTLRAValidatorBorrowsFParamsCopiesContextAndAdmitsUnknown(t *testing.T) {
	m, parent := smtSolverFixture(t)
	family, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	a, err := NewSMTLRATheory(family)
	astTestMust(t, err)
	astTestMust(t, parent.kernel.RegisterTheory(a))
	name := "random_seed"
	astTestMust(t, parent.kernel.paramContext.SetUIntName(&parent.kernel.assertedParams, &name, 73))
	c := &smtLRAValidationContext{}
	astTestMust(t, c.initialize(a))
	if c.kernel.smtParams != parent.kernel.smtParams || !c.kernel.smtParamsBorrowed || c.full != nil || c.params != nil || c.kernel.setupLogic != "" || c.kernel.internalizer == parent.kernel.internalizer {
		t.Fatal("fresh context identity/parameters")
	}
	copied, err := c.kernel.paramContext.GetUIntName(&c.kernel.assertedParams, &name, 0)
	astTestMust(t, err)
	if copied != 73 {
		t.Fatal("current params not copied", copied)
	}
	astTestMust(t, parent.kernel.paramContext.SetUIntName(&parent.kernel.assertedParams, &name, 91))
	copied, err = c.kernel.paramContext.GetUIntName(&c.kernel.assertedParams, &name, 0)
	astTestMust(t, err)
	if copied != 73 {
		t.Fatal("context param alias", copied)
	}
	astTestMust(t, c.close())
	_, err = parent.kernel.smtParams.State()
	astTestMust(t, err)
	// The source expression is l_true != nctx.check(): UNKNOWN validates.
	parent.kernel.smtParams.state.MaxConflicts = 0
	p := smtRelevancyBool(t, m, "validator-limit-p")
	q := smtRelevancyBool(t, m, "validator-limit-q")
	np, err := m.MkNot(p)
	astTestMust(t, err)
	astTestPin(t, m, np)
	nq, err := m.MkNot(q)
	astTestMust(t, err)
	astTestPin(t, m, nq)
	var clauses []ast.ASTHandle
	for _, args := range [][2]ast.ASTHandle{{p, q}, {np, q}, {p, nq}, {np, nq}} {
		h, err := m.MkOr(args[0], args[1])
		astTestMust(t, err)
		astTestPin(t, m, h)
		clauses = append(clauses, h)
	}
	formula, err := m.MkAndArgs(clauses)
	astTestMust(t, err)
	astTestPin(t, m, formula)
	astTestMust(t, parent.kernel.internalizer.Internalize(formula, true))
	lit, err := parent.kernel.internalizer.Literal(formula)
	astTestMust(t, err)
	c = &smtLRAValidationContext{}
	astTestMust(t, c.initialize(a))
	child := c
	t.Cleanup(func() { astTestMust(t, child.close()) })
	astTestMust(t, c.assert(formula))
	status, err := c.kernel.Check()
	astTestMust(t, err)
	if status != lbool.Undef || c.kernel.lastFailure != SMTFailureConflictLimit {
		t.Fatal("fixture failed to reach conflict-limit UNKNOWN", status, c.kernel.lastFailure)
	}
	astTestMust(t, c.close())
	lraValidationCPP(t, parent.kernel.internalizer, []SMTLiteral{lit}, SMTFalseLiteral, true, "(with smt :max_conflicts 0)")
	valid, err := a.validateConflict([]SMTLiteral{lit})
	astTestMust(t, err)
	if !valid {
		t.Fatal("UNKNOWN was treated as invalid")
	}

}

func TestSMTLRAValidatorDumpBadbitAndFailureTrace(t *testing.T) {
	t.Chdir(t.TempDir())
	old := smtLemmaID
	defer func() { smtLemmaID = old }()
	smtLemmaID = 0
	f, ls, _ := smtLemmaFixture(t)
	f.a.config.DumpLemmas = true
	astTestMust(t, os.Mkdir("lemma_0.smt2", 0700))
	valid, err := f.a.validateAssignment([]SMTLiteral{ls[0]}, ls[0])
	astTestMust(t, err)
	if !valid || smtLemmaID != 1 {
		t.Fatal("file badbit stopped validation", valid, smtLemmaID)
	}
	var trace bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&trace)
	tags.Enable("arith")
	f.a.trace = tags
	valid, err = f.a.validateAssignment(nil, ls[0])
	astTestMust(t, err)
	if valid {
		t.Fatal("unsupported implication validated")
	}
	if tags.ShouldTrace("arith") {
		text := trace.String()
		if !strings.Contains(text, "validate_assign src/smt/theory_lra.cpp:2306") || strings.Count(text, "(assert (not lemma-p))\n") != 1 {
			t.Fatal("temporary negation not popped before source trace", text)
		}
	} else if trace.Len() != 0 {
		t.Fatal("native CTRACE executed")
	}
	text, err := os.ReadFile("lemma_1.smt2")
	astTestMust(t, err)
	if strings.Count(string(text), "(assert (not lemma-p))\n") != 1 {
		t.Fatal(string(text))
	}
	astTestMust(t, os.Mkdir("lemma_2.smt2", 0700))
	_, err = f.a.validateAssignment([]SMTLiteral{SMTNullLiteral}, ls[0])
	if err == nil || smtLemmaID != 2 {
		t.Fatal("operand error hidden by dump badbit", err, smtLemmaID)
	}
	if _, err = f.a.validateEquality(nil, nil, nil); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("nil equality pair", err)
	}

}

func TestSMTLRAValidationPhysicalConstructorCleanup(t *testing.T) {
	failures, successes := 0, 0
	for _, budget := range []int64{-1, 0, 1, 2, 4, 8, 16, 32, 64, 128} {
		t.Run(fmt.Sprint(budget), func(t *testing.T) {
			f, ls, _ := smtLemmaFixture(t)
			f.m.Budget().FailAfter(budget)
			valid, err := f.a.validateAssignment([]SMTLiteral{ls[0]}, ls[0])
			f.m.Budget().FailAfter(-1)
			if err != nil {
				var failure umemory.Failure
				if budget < 0 || (!errors.Is(err, arena.ErrInjectedFailure) && !(errors.As(err, &failure) && failure.Kind == umemory.FailureMemout)) {
					t.Fatal("unexpected constructor failure", budget, err)
				}
				failures++
			} else {
				successes++
				if !valid {
					t.Fatal("valid tautological implication rejected")
				}
			}
			astTestMust(t, f.a.closeValidation())
			// Every default service/kernel constructed before refusal is owned by the
			// validation context; manager/parameter teardown checks the full budget.
		})
	}
	if failures == 0 || successes == 0 {
		t.Fatal("physical campaign missed refusal/success", failures, successes)
	}
}

func lraValidationCPP(t *testing.T, state *SMTInternalizerState, core []SMTLiteral, conclusion SMTLiteral, want bool, tactic string) {
	t.Helper()
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		return
	}
	var dump bytes.Buffer
	astTestMust(t, state.DisplayLemmaAsSMTProblem(&dump, core, nil, conclusion, usymbol.Symbol{}))
	if tactic == "" {
		tactic = "smt"
	}
	script := dump.String() + "(check-sat-using " + tactic + ")\n"
	cmd := exec.Command(reference, "-in")
	cmd.Stdin = strings.NewReader(script)
	output, err := cmd.CombinedOutput()
	astTestMust(t, err)
	result := strings.TrimSpace(string(output))
	if result != "sat" && result != "unsat" && result != "unknown" {
		t.Fatal("C++ status", result, script)
	}
	if (result != "sat") != want {
		t.Fatal("C++ validator parity", result, want, script)
	}
	if strings.Contains(tactic, "max_conflicts") && result != "unknown" {
		t.Fatal("C++ source limit did not reach UNKNOWN", result, script)
	}
}

func TestSMTLRAValidatorAssertionGatesAndSourceLocations(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: b}})
	astTestMust(t, err)
	d, err := diagnostic.NewDebugContext(region, b)
	astTestMust(t, err)
	inputFile, err := os.CreateTemp(t.TempDir(), "assert-input-*")
	astTestMust(t, err)
	_, err = inputFile.WriteString("TTT")
	astTestMust(t, err)
	_, err = inputFile.Seek(0, 0)
	astTestMust(t, err)
	report, err := os.CreateTemp(t.TempDir(), "assert-report-*")
	astTestMust(t, err)
	streams, err := diagnostic.NewMemoryStandardStreams(diagnostic.MemoryStandardStreamsConfig{Budget: b, Stdout: report, Stderr: report})
	astTestMust(t, err)
	input, err := diagnostic.NewDebuggerCharInput(streams, inputFile)
	astTestMust(t, err)
	actions, err := diagnostic.NewDebugActionContext(diagnostic.DebugActionConfig{Budget: b, Streams: streams, Input: input})
	astTestMust(t, err)
	astTestMust(t, d.BindAssertionStreams(streams))
	astTestMust(t, d.BindAssertionActions(actions))
	previous, err := diagnostic.BindProcessDebugContext(d)
	astTestMust(t, err)
	defer func() { _, err := diagnostic.BindProcessDebugContext(previous); astTestMust(t, err) }()
	t.Cleanup(func() {
		astTestMust(t, actions.Close())
		astTestMust(t, input.Close())
		astTestMust(t, streams.Close())
		astTestMust(t, inputFile.Close())
		astTestMust(t, report.Close())
		astTestMust(t, d.Close())
		astTestMust(t, region.Close())
		if b.Used() != 0 {
			t.Fatal("assertion owner leak", b.Used())
		}
	})
	f, ls, eqs := smtLemmaFixture(t)
	astTestMust(t, d.EnableAssertions(false))
	astTestMust(t, f.a.checkValidation(smtLRAValidateAssignment, []SMTLiteral{SMTNullLiteral}, SMTNullLiteral, nil, nil))
	astTestMust(t, d.EnableAssertions(true))
	for _, tc := range []struct {
		kind      smtLRAValidationKind
		line      int
		condition string
	}{
		{smtLRAValidateConflict, 2229, "validate_conflict()"},
		{smtLRAValidateAssignment, 1435, "validate_assign(lit)"},
		{smtLRAValidateEquality, 2129, "validate_eq(x, y)"},
	} {
		err := f.a.checkValidation(tc.kind, nil, ls[0], eqs[0].First, eqs[0].Second)
		if diagnostic.DebugBuild {
			var ex z3exception.DefaultException
			if !errors.As(err, &ex) || ex.Message() != "assertion violation" {
				t.Fatal(tc, err)
			}
		} else {
			astTestMust(t, err)
		}
	}
	_, err = report.Seek(0, 0)
	astTestMust(t, err)
	text, err := os.ReadFile(report.Name())
	astTestMust(t, err)
	if diagnostic.DebugBuild {
		for _, value := range []string{"src/smt/theory_lra.cpp", "2229", "1435", "2129", "validate_conflict()", "validate_assign(lit)", "validate_eq(x, y)"} {
			if !strings.Contains(string(text), value) {
				t.Fatal(value, string(text))
			}
		}
	} else if len(text) != 0 {
		t.Fatal("native assertions produced diagnostics", string(text))
	}
}

func TestSMTLRAValidationConfiguredConstructorCleanup(t *testing.T) {
	failures, successes := 0, 0
	for _, budget := range []int64{-1, 0, 1, 4, 8, 16, 32, 64, 128, 256} {
		t.Run(fmt.Sprint(budget), func(t *testing.T) {
			m, parent := smtSolverFixture(t)
			family, err := m.RegisterArithmeticPlugin()
			astTestMust(t, err)
			a, err := NewSMTLRATheory(family)
			astTestMust(t, err)
			astTestMust(t, parent.kernel.RegisterTheory(a))
			p := smtRelevancyBool(t, m, "validator-physical-parent-p")
			astTestMust(t, parent.kernel.internalizer.Internalize(p, true))
			lit, err := parent.kernel.internalizer.Literal(p)
			astTestMust(t, err)
			m.Budget().FailAfter(budget)
			valid, err := a.validateAssignment([]SMTLiteral{lit}, lit)
			m.Budget().FailAfter(-1)
			if err != nil {
				var failure umemory.Failure
				if budget < 0 || (!errors.Is(err, arena.ErrInjectedFailure) && !(errors.As(err, &failure) && failure.Kind == umemory.FailureMemout)) {
					t.Fatal("unexpected configured failure", budget, err)
				}
				failures++
			} else {
				successes++
				if !valid {
					t.Fatal("valid implication failed")
				}
			}
			astTestMust(t, a.closeValidation())
			_, err = parent.kernel.smtParams.State()
			astTestMust(t, err)
		})
	}
	if failures == 0 || successes == 0 {
		t.Fatal("configured physical campaign", failures, successes)
	}
}
