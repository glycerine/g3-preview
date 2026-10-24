package bv

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
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	uparams "github.com/glycerine/g3/pkg/util/params"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func sharingTacticFixture(t *testing.T, proofs bool) *MaxBVSharingTactic {
	t.Helper()
	mode := ast.ASTProofDisabled
	if proofs {
		mode = ast.ASTProofEnabled
	}
	m, _ := sharingFixtureProofMode(t, mode)
	c, err := uparams.NewParamsContext(m.Symbols, m.NumeralService(), uparams.ParamsConfig{Budget: m.Budget()})
	sharingMust(t, err)
	catalog, err := uparams.NewGlobalParamsCatalog(m.Symbols, uparams.GlobalParamsCatalogConfig{Budget: m.Budget()})
	sharingMust(t, err)
	g, err := uparams.NewGlobalParamsService(m.Symbols, c, uparams.GlobalParamsConfig{Budget: m.Budget(), Catalog: catalog})
	sharingMust(t, err)
	sharingMust(t, g.InitializeSource())
	previous, err := uparams.BindProcessGlobalParamsService(g)
	sharingMust(t, err)
	t.Cleanup(func() {
		_, err := uparams.BindProcessGlobalParamsService(previous)
		sharingMust(t, err)
		sharingMust(t, g.Close())
		sharingMust(t, catalog.Close())
		sharingMust(t, c.Close())
	})
	r, err := NewMaxBVSharingTactic(m, c, nil, nil)
	sharingMust(t, err)
	t.Cleanup(func() { sharingMust(t, r.Close()) })
	return r
}

func sharingTacticGoal(t *testing.T, r *MaxBVSharingTactic, op int32, proofs bool) (*tactic.Goal, []ast.ASTHandle) {
	t.Helper()
	m := r.manager
	sort, err := m.BitVectorSort(8)
	sharingMust(t, err)
	var vars []ast.ASTHandle
	for _, name := range []string{"a", "b", "c", "d", "e"} {
		sym, err := m.Symbols.Intern(name)
		sharingMust(t, err)
		h, err := m.MkConst(sym, sort)
		sharingMust(t, err)
		ref, err := m.NewRef(h)
		sharingMust(t, err)
		t.Cleanup(func() { sharingMust(t, m.DelRef(&ref)) })
		vars = append(vars, h)
	}
	family, err := m.RegisterBitVectorPlugin()
	sharingMust(t, err)
	goal, err := tactic.NewGoal(m, proofs, false, !proofs)
	sharingMust(t, err)
	t.Cleanup(func() { sharingMust(t, goal.Close()) })
	for _, args := range [][]ast.ASTHandle{vars[:2], {vars[1], vars[2], vars[0], vars[3], vars[4]}} {
		term, err := m.MkFamilyApp(family, op, nil, args)
		sharingMust(t, err)
		form, err := m.MkEq(term, vars[2])
		sharingMust(t, err)
		sharingMust(t, goal.Assert(form))
	}
	return goal, vars
}

func sharingTermText(t *testing.T, r *MaxBVSharingTactic, h ast.ASTHandle) string {
	t.Helper()
	m := r.manager
	f, err := m.SMT2PPExprTree(h, nil)
	sharingMust(t, err)
	out, err := utext.NewNumericTextStream(utext.NumericTextStreamConfig{Budget: m.Budget()})
	sharingMust(t, err)
	defer func() { sharingMust(t, out.Close()) }()
	p := r.context.EmptyValue()
	sharingMust(t, m.PrintFormat(out, f, r.context, &p, uparams.BorrowProcessGlobalParamsService()))
	s, err := out.String()
	sharingMust(t, err)
	return s
}

func TestMaxBVSharingCPPGoalShapes(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	for _, op := range []int32{4, 6, 31, 33} {
		for _, maxArgs := range []uint32{2, 128} {
			for _, proofs := range []bool{false, true} {
				t.Run(fmt.Sprint(op, "/", maxArgs, "/", proofs), func(t *testing.T) {
					r := sharingTacticFixture(t, proofs)
					p, err := r.context.NewValue()
					sharingMust(t, err)
					defer func() { sharingMust(t, r.context.Del(&p)) }()
					key := "max_args"
					sharingMust(t, r.context.SetUIntName(&p, &key, maxArgs))
					sharingMust(t, r.UpdateParams(&p))
					goal, _ := sharingTacticGoal(t, r, op, proofs)
					var source strings.Builder
					fmt.Fprintf(&source, "(set-option :produce-proofs %v)(set-logic QF_BV)", proofs)
					for _, name := range []string{"a", "b", "c", "d", "e"} {
						fmt.Fprintf(&source, "(declare-const %s (_ BitVec 8))", name)
					}
					for i := uint32(0); i < 2; i++ {
						h, err := goal.Form(i)
						sharingMust(t, err)
						fmt.Fprintf(&source, "(assert %s)", sharingTermText(t, r, h))
					}
					fmt.Fprintf(&source, "(apply (using-params max-bv-sharing :max_args %d))", maxArgs)
					var result tactic.TacticResult
					defer func() { sharingMust(t, result.Close()) }()
					sharingMust(t, r.Apply(goal, &result))
					var actual strings.Builder
					actual.WriteString("(goals (goal ")
					n, err := goal.Size()
					sharingMust(t, err)
					for i := uint32(0); i < n; i++ {
						h, err := goal.Form(i)
						sharingMust(t, err)
						actual.WriteString(sharingTermText(t, r, h) + " ")
						if proofs {
							pr, err := goal.Proof(i)
							sharingMust(t, err)
							fact, err := r.manager.GetProofFact(pr)
							sharingMust(t, err)
							if fact != h {
								t.Fatal("proof fact differs from rewritten formula")
							}
						}
					}
					depth, err := goal.Depth()
					sharingMust(t, err)
					fmt.Fprintf(&actual, ":precision precise :depth %d))", depth)
					cmd := exec.Command(reference, "-in")
					cmd.Stdin = strings.NewReader(source.String())
					native, err := cmd.CombinedOutput()
					if err != nil {
						t.Fatal(err, string(native))
					}
					canonical := func(s string) string {
						return strings.Join(strings.Fields(strings.NewReplacer("(", " ( ", ")", " ) ").Replace(s)), " ")
					}
					if canonical(actual.String()) != canonical(string(native)) {
						t.Fatalf("Go: %s\nC++: %s", actual.String(), native)
					}
				})
			}
		}
	}
}

func TestMaxBVSharingLifecycleAndTrace(t *testing.T) {
	r := sharingTacticFixture(t, false)
	goal, _ := sharingTacticGoal(t, r, 4, false)
	var trace bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&trace)
	tags.Enable("bv_sharing_detail")
	tags.Enable("max_bv_sharing")
	sharingMust(t, r.BindTrace(tags))
	var result tactic.TacticResult
	defer func() { sharingMust(t, result.Close()) }()
	sharingMust(t, r.Apply(goal, &result))
	if len(result.Goals) != 1 || result.Goals[0] != goal {
		t.Fatal("expected borrowed input")
	}
	if tags.ShouldTrace("max_bv_sharing") && (!strings.Contains(trace.String(), "reusing args: 0 2\n") || !strings.Contains(trace.String(), "operator() src/tactic/bv/max_bv_sharing_tactic.cpp:268")) {
		t.Fatal(trace.String())
	}
	for _, pairs := range r.current.configuration.reducer.pairs {
		if len(pairs) != 0 {
			t.Fatal("successful pass retained pair cache")
		}
	}
	p, err := r.context.NewValue()
	sharingMust(t, err)
	defer func() { sharingMust(t, r.context.Del(&p)) }()
	key := "max_args"
	sharingMust(t, r.context.SetUIntName(&p, &key, 2))
	sharingMust(t, r.UpdateParams(&p))
	for i := 0; i < 2; i++ {
		sharingMust(t, r.Cleanup())
		if r.current.configuration.reducer.maxArgs != 2 {
			t.Fatal("cleanup lost params snapshot")
		}
	}
	other, err := r.Translate(r.manager)
	sharingMust(t, err)
	translated := other.(*MaxBVSharingTactic)
	defer func() { sharingMust(t, translated.Close()) }()
	if translated.current == r.current || translated.current.configuration.reducer.maxArgs != 2 {
		t.Fatal("translation snapshot")
	}
	sharingMust(t, r.IncRef())
	sharingMust(t, r.DecRef())
	if err := r.Cleanup(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("closed owner", err)
	}
}

func TestMaxBVSharingAllocationFrontiers(t *testing.T) {
	for _, phase := range []string{"constructor", "apply", "cleanup"} {
		success := false
		for stop := int64(0); stop < 1024; stop++ {
			t.Run(fmt.Sprint(phase, "/", stop), func(t *testing.T) {
				r := sharingTacticFixture(t, true)
				m := r.manager
				if phase == "constructor" {
					m.Budget().FailAfter(stop)
					fresh, err := NewMaxBVSharingTactic(m, r.context, nil, nil)
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
				goal, _ := sharingTacticGoal(t, r, 4, true)
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
		}
		if !success {
			t.Fatal("no successful frontier", phase)
		}
	}
}
