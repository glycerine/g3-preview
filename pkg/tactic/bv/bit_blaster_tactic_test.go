package bv

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	bb "github.com/glycerine/g3/pkg/ast/rewriter/bit_blaster"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func bitTacticFixture(t *testing.T, proofs bool) *BitBlasterTactic {
	t.Helper()
	base := sharingTacticFixture(t, proofs)
	r, err := NewBitBlasterTactic(base.manager, base.context, uparams.BorrowProcessGlobalParamsService(), nil, nil)
	sharingMust(t, err)
	t.Cleanup(func() { sharingMust(t, r.Close()) })
	return r
}
func bitTacticGoal(t *testing.T, r *BitBlasterTactic, op int32, proofs, models bool) *tactic.Goal {
	t.Helper()
	m := r.manager
	sort, err := m.BitVectorSort(3)
	sharingMust(t, err)
	var args []ast.ASTHandle
	for _, s := range []string{"a", "b"} {
		name, err := m.Symbols.Intern(s)
		sharingMust(t, err)
		h, err := m.MkConst(name, sort)
		sharingMust(t, err)
		ref, err := m.NewRef(h)
		sharingMust(t, err)
		t.Cleanup(func() { sharingMust(t, m.DelRef(&ref)) })
		args = append(args, h)
	}
	family, err := m.RegisterBitVectorPlugin()
	sharingMust(t, err)
	form, err := m.MkFamilyApp(family, op, nil, args)
	sharingMust(t, err)
	if op != 22 && op != 23 {
		n, err := m.BitVectorNumeralUint64(5, 3)
		sharingMust(t, err)
		form, err = m.MkEq(form, n)
		sharingMust(t, err)
	}
	goal, err := tactic.NewGoal(m, proofs, models, !proofs)
	sharingMust(t, err)
	t.Cleanup(func() { sharingMust(t, goal.Close()) })
	sharingMust(t, goal.Assert(form))
	return goal
}

func TestBitBlasterTacticCPPGoalShapes(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	for _, op := range []int32{4, 6, 22, 23, 33, 45} {
		for _, proofs := range []bool{false, true} {
			t.Run(fmt.Sprint(op, "/", proofs), func(t *testing.T) {
				r := bitTacticFixture(t, proofs)
				goal := bitTacticGoal(t, r, op, proofs, false)
				var source bytes.Buffer
				fmt.Fprintf(&source, "(set-option :produce-proofs %v)(set-logic QF_BV)(declare-const a (_ BitVec 3))(declare-const b (_ BitVec 3))(assert ", proofs)
				form, err := goal.Form(0)
				sharingMust(t, err)
				sharingMust(t, r.writeExpr(&source, form))
				source.WriteString(")(apply bit-blast)")
				var result tactic.TacticResult
				defer func() { sharingMust(t, result.Close()) }()
				sharingMust(t, r.Apply(goal, &result))
				var actual bytes.Buffer
				actual.WriteString("(goals (goal ")
				n, err := goal.Size()
				sharingMust(t, err)
				for i := uint32(0); i < n; i++ {
					h, err := goal.Form(i)
					sharingMust(t, err)
					sharingMust(t, r.writeExpr(&actual, h))
					actual.WriteByte(' ')
					if proofs {
						pr, err := goal.Proof(i)
						sharingMust(t, err)
						fact, err := r.manager.GetProofFact(pr)
						sharingMust(t, err)
						if fact != h {
							t.Fatal("rewritten assertion proof fact")
						}
					}
				}
				depth, err := goal.Depth()
				sharingMust(t, err)
				fmt.Fprintf(&actual, ":precision precise :depth %d))", depth)
				cmd := exec.Command(reference, "-in")
				cmd.Stdin = bytes.NewReader(source.Bytes())
				native, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatal(err, string(native))
				}
				canonical := func(s string) string {
					names := make(map[string]string)
					s = regexp.MustCompile(`k![0-9]+`).ReplaceAllStringFunc(s, func(name string) string {
						if n, ok := names[name]; ok {
							return n
						}
						n := fmt.Sprint("bit", len(names))
						names[name] = n
						return n
					})
					return strings.Join(strings.Fields(strings.NewReplacer("(", " ( ", ")", " ) ").Replace(s)), " ")
				}
				if canonical(actual.String()) != canonical(string(native)) {
					t.Fatalf("Go: %s\nC++: %s", actual.String(), native)
				}
			})
		}
	}
}

func TestBitBlasterTacticLifecycleExternalRewriterAndTrace(t *testing.T) {
	r := bitTacticFixture(t, false)
	goal := bitTacticGoal(t, r, 4, false, true)
	var trace bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&trace)
	for _, tag := range []string{"before_bit_blaster", "bit_blaster", "after_bit_blaster"} {
		tags.Enable(tag)
	}
	sharingMust(t, r.BindTrace(tags))
	var result tactic.TacticResult
	defer func() { sharingMust(t, result.Close()) }()
	sharingMust(t, r.Apply(goal, &result))
	if len(result.Goals) != 1 || result.Goals[0] != goal || result.ModelConverter == nil {
		t.Fatal("goal or converter publication")
	}
	steps, err := r.NumSteps()
	sharingMust(t, err)
	if steps == 0 {
		t.Fatal("missing rewrite steps")
	}
	if tags.ShouldTrace("bit_blaster") && (!strings.Contains(trace.String(), "bit_blaster_rewriter.cpp:350") || !strings.Contains(trace.String(), "bit_blaster_tactic.cpp:85")) {
		t.Fatal(trace.String())
	}
	sharingMust(t, r.Cleanup())
	var display bytes.Buffer
	sharingMust(t, result.ModelConverter.Display(&display))
	if !strings.Contains(display.String(), "bit-blaster-model-converter") {
		t.Fatal(display.String())
	}
	p, err := r.context.NewValue()
	sharingMust(t, err)
	defer func() { sharingMust(t, r.context.Del(&p)) }()
	key := "blast_add"
	sharingMust(t, r.context.SetBoolName(&p, &key, false))
	sharingMust(t, r.UpdateParams(&p))
	sharingMust(t, r.Cleanup())
	other, err := r.Translate(r.manager)
	sharingMust(t, err)
	translated := other.(*BitBlasterTactic)
	defer func() { sharingMust(t, translated.Close()) }()
	policy, err := translated.policy()
	sharingMust(t, err)
	if policy.BlastAdd {
		t.Fatal("translation lost parameter snapshot")
	}
	external, err := bb.NewBitBlasterRewriter(r.manager, nil)
	sharingMust(t, err)
	defer func() { sharingMust(t, external.Close()) }()
	borrower, err := NewBitBlasterTacticWithRewriter(r.manager, external, r.context, r.globals, nil, nil)
	sharingMust(t, err)
	if borrower.current.base == external || borrower.current.active != external {
		t.Fatal("source base/external ownership")
	}
	sharingMust(t, external.Push())
	borrowedGoal := bitTacticGoal(t, borrower, 33, false, false)
	var borrowedResult tactic.TacticResult
	defer func() { sharingMust(t, borrowedResult.Close()) }()
	sharingMust(t, borrower.Apply(borrowedGoal, &borrowedResult))
	sharingMust(t, borrower.Cleanup())
	cache, err := external.Const2Bits()
	sharingMust(t, err)
	if len(cache) != 2 {
		t.Fatal("external cache did not survive tactic cleanup", len(cache))
	}
	externalCopy, err := borrower.Translate(r.manager)
	if ast.ASTSourceDebug {
		if !errors.Is(err, ast.ErrASTDomain) || externalCopy != nil {
			t.Fatal("source external-translation assertion", err)
		}
	} else {
		sharingMust(t, err)
		owned := externalCopy.(*BitBlasterTactic)
		if owned.external != nil {
			t.Fatal("release translation retained external rewriter")
		}
		sharingMust(t, owned.Close())
	}
	sharingMust(t, borrower.Close())
	if _, err := external.Manager(); err != nil {
		t.Fatal("borrowed rewriter was destroyed", err)
	}
	sharingMust(t, external.Pop(1))
	cache, err = external.Const2Bits()
	sharingMust(t, err)
	if len(cache) != 0 {
		t.Fatal("external scoped constants survived pop", len(cache))
	}
	copy := *r
	if err := copy.Cleanup(); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("copied owner", err)
	}
	sharingMust(t, r.IncRef())
	sharingMust(t, r.DecRef())
	if err := r.ResetStatistics(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed owner", err)
	}
}

func TestBitBlasterTacticAllocationFrontiers(t *testing.T) {
	for _, phase := range []string{"constructor", "apply", "cleanup"} {
		success := false
		for stop := int64(0); stop < 2048; stop++ {
			t.Run(fmt.Sprint(phase, "/", stop), func(t *testing.T) {
				r := bitTacticFixture(t, true)
				m := r.manager
				if phase == "constructor" {
					m.Budget().FailAfter(stop)
					fresh, err := NewBitBlasterTactic(m, r.context, r.globals, nil, nil)
					m.Budget().FailAfter(-1)
					if fresh != nil {
						sharingMust(t, fresh.Close())
					}
					if err == nil {
						success = true
					} else if !errors.Is(err, arena.ErrInjectedFailure) {
						t.Fatal(err)
					}
					return
				}
				goal := bitTacticGoal(t, r, 4, true, true)
				var result tactic.TacticResult
				defer func() { sharingMust(t, result.Close()) }()
				if phase == "cleanup" {
					sharingMust(t, r.Apply(goal, &result))
				}
				m.Budget().FailAfter(stop)
				var err error
				if phase == "cleanup" {
					err = r.Cleanup()
				} else {
					err = r.Apply(goal, &result)
				}
				m.Budget().FailAfter(-1)
				if err == nil {
					success = true
				} else if !errors.Is(err, arena.ErrInjectedFailure) {
					t.Fatal(err)
				}
			})
			if success {
				break
			}
			if t.Failed() {
				t.Fatal("failed allocation frontier", phase, stop)
			}
		}
		if !success {
			t.Fatal("no successful reservation frontier", phase)
		}
	}
}
