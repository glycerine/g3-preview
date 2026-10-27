package portfolio_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/solver"
	"github.com/glycerine/g3/pkg/tactic/portfolio"
	"github.com/glycerine/g3/pkg/util/lbool"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

// This recorder retains the actual SMT backend; only the targeted source
// boundary reports a test failure/result instead of delegating that operation.
type enumRecordingBackend struct {
	solver.Solver
	pushError, popError error
	popNoop             bool
	updates             uint32
	consequence         func([]ast.ASTHandle, []ast.ASTHandle) (lbool.LBool, []ast.ASTHandle, error)
}

func (b *enumRecordingBackend) Push() error {
	if b.pushError != nil {
		return b.pushError
	}
	return b.Solver.Push()
}
func (b *enumRecordingBackend) Pop(n uint32) error {
	if b.popError != nil {
		return b.popError
	}
	if b.popNoop {
		return nil
	}
	return b.Solver.Pop(n)
}
func (b *enumRecordingBackend) UpdateParams(*uparams.ParamsRef) error { b.updates++; return nil }
func (b *enumRecordingBackend) GetConsequences(a, v []ast.ASTHandle) (lbool.LBool, []ast.ASTHandle, error) {
	if b.consequence != nil {
		return b.consequence(a, v)
	}
	return b.Solver.GetConsequences(a, v)
}
func enumRecorded(t *testing.T, f *enumFixture) (*portfolio.Enum2BVSolver, *enumRecordingBackend) {
	t.Helper()
	b := &enumRecordingBackend{Solver: f.backend}
	s, e := portfolio.NewEnum2BVSolver(f.manager, f.services.Context, nil, b, f.services)
	enumMust(t, e)
	t.Cleanup(func() { enumMust(t, s.Close()) })
	return s, b
}
func TestEnum2BVSolverScopeErrorPrefixes(t *testing.T) {
	f := enumSolverFixture(t)
	m := f.manager
	s, b := enumRecorded(t, f)
	stop := errors.New("source backend scope stopped")
	b.pushError = stop
	if err := s.Push(); !errors.Is(err, stop) {
		t.Fatal("push failure", err)
	}
	n, err := s.ScopeLevel()
	enumMust(t, err)
	if n != 1 {
		t.Fatal("source named scope was rolled back", n)
	}
	// The enum rewriter must also have pushed before the failed backend call.
	b.popNoop = true
	enumMust(t, s.Pop(1))
	b.popNoop = false
	b.pushError = nil
	_, h, x := enumGoal(t, f)
	enumMust(t, s.Assert(h))
	enumMust(t, s.Push())
	sort, err := m.GetSort(x)
	enumMust(t, err)
	name, err := m.Symbols.Intern("scoped-enum-y")
	enumMust(t, err)
	y, err := m.MkConst(name, sort)
	enumMust(t, err)
	enumPin(t, m, y)
	green, err := m.GetArg(h, 1)
	enumMust(t, err)
	eq, err := m.MkEq(y, green)
	enumMust(t, err)
	enumMust(t, s.Assert(eq))
	status, err := s.CheckSat(nil)
	enumMust(t, err)
	if status != lbool.True {
		t.Fatal(status)
	}
	b.popError = stop
	if err = s.Pop(1); !errors.Is(err, stop) {
		t.Fatal("pop failure", err)
	}
	// A stopped backend pop must leave enum mappings and definitions intact.
	md, err := s.AcquireModel()
	enumMust(t, err)
	if md == nil {
		t.Fatal("missing stopped-pop model")
	}
	d, err := m.GetDecl(y)
	enumMust(t, err)
	value, err := md.ConstantInterpretation(d)
	enumMust(t, err)
	if value != green {
		t.Fatal("rewriter popped before the stopped backend", value)
	}
	enumMust(t, md.DecRef())
	b.popError = nil
	enumMust(t, s.Pop(1))
}
func TestEnum2BVSolverConsequenceProjectionBranches(t *testing.T) {
	for _, which := range []string{"enum", "out-of-range", "reverse-equality", "foreign-BV", "non-numeral", "Boolean"} {
		for _, status := range []lbool.LBool{lbool.True, lbool.Undef} {
			t.Run(fmt.Sprint(which, "/", status), func(t *testing.T) {
				f := enumSolverFixture(t)
				m := f.manager
				s, b := enumRecorded(t, f)
				_, _, x := enumGoal(t, f)
				tag := enumTag(t, m, "consequence-named")
				extra := enumTag(t, m, "consequence-extra")
				boolean := enumTag(t, m, "consequence-boolean")
				truth, e := m.MkTrue()
				enumMust(t, e)
				enumMust(t, s.AssertTracked(truth, tag))
				before, e := b.Solver.NumAssertions()
				enumMust(t, e)
				var original ast.ASTHandle
				b.consequence = func(a, v []ast.ASTHandle) (lbool.LBool, []ast.ASTHandle, error) {
					if len(a) != 2 || a[0] != tag || a[1] != extra || len(v) != 2 || v[1] != boolean {
						t.Fatal("named assumptions or variable order", a, v)
					}
					sort, e := m.GetSort(v[0])
					enumMust(t, e)
					width, e := m.BitVectorSize(sort)
					enumMust(t, e)
					if width != 2 {
						t.Fatal("untranslated query variable")
					}
					n, e := b.Solver.NumAssertions()
					enumMust(t, e)
					if n != before+1 {
						t.Fatal("absent enum side bound not asserted before query", n)
					}
					index := uint32(1)
					if which == "out-of-range" {
						index = 3
					}
					number, e := m.BitVectorNumeralUnsigned(index, 2)
					enumMust(t, e)
					lhs, rhs := v[0], number
					switch which {
					case "reverse-equality":
						lhs, rhs = rhs, lhs
					case "foreign-BV", "non-numeral":
						name, e := m.Symbols.Intern("other-BV")
						enumMust(t, e)
						foreign, e := m.MkConst(name, sort)
						enumMust(t, e)
						if which == "foreign-BV" {
							lhs = foreign
						} else {
							rhs = foreign
						}
					}
					body := boolean
					if which != "Boolean" {
						body, e = m.MkEq(lhs, rhs)
						enumMust(t, e)
					}
					antecedent, e := m.MkAnd(extra, tag)
					enumMust(t, e)
					original, e = m.MkImplies(antecedent, body)
					enumMust(t, e)
					return status, []ast.ASTHandle{original}, nil
				}
				got, out, err := s.GetConsequences([]ast.ASTHandle{extra}, []ast.ASTHandle{x, boolean})
				enumMust(t, err)
				if got != status || len(out) != 1 {
					t.Fatal(got, out)
				}
				if which == "enum" {
					if out[0] == original {
						t.Fatal("enum consequence not projected")
					}
					head, e := m.GetArg(out[0], 1)
					enumMust(t, e)
					left, e := m.GetArg(head, 0)
					enumMust(t, e)
					if left != x {
						t.Fatal("projection uses fresh BV instead of original enum")
					}
					right, e := m.GetArg(head, 1)
					enumMust(t, e)
					decl, e := m.GetDecl(right)
					enumMust(t, e)
					name, e := m.GetASTName(decl)
					enumMust(t, e)
					text, e := m.Symbols.String(name)
					enumMust(t, e)
					if text != "amber" {
						t.Fatal("constructor index", text)
					}
				} else if out[0] != original {
					t.Fatal("nonmatching consequence changed")
				}
				refs, e := m.GetRefCount(out[0])
				enumMust(t, e)
				if refs == 0 {
					t.Fatal("borrowed consequence has no publishing owner")
				}
				count, e := s.NumAssumptions()
				enumMust(t, e)
				if count != 1 {
					t.Fatal("temporary query assumptions survived", count)
				}
			})
		}
	}
}
func TestEnum2BVSolverConstructorAndParameterPrefixes(t *testing.T) {
	f := enumSolverFixture(t)
	m, c := f.manager, f.services.Context
	p, err := c.NewValue()
	enumMust(t, err)
	t.Cleanup(func() { enumMust(t, c.Del(&p)) })
	key := "enum-wrapper-snapshot"
	enumMust(t, c.SetUIntName(&p, &key, 17))
	b := &enumRecordingBackend{Solver: f.backend}
	warm, err := portfolio.NewEnum2BVSolver(m, c, &p, b, f.services)
	enumMust(t, err)
	enumMust(t, warm.Close())
	baseline := m.Budget().Used()
	refs := c.Stats().References
	complete := false
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		m.Budget().FailAfter(ordinal)
		s, err := portfolio.NewEnum2BVSolver(m, c, &p, b, f.services)
		m.Budget().FailAfter(-1)
		if err != nil && !errors.Is(err, arena.ErrInjectedFailure) {
			t.Fatal("unexpected constructor prefix", ordinal, err)
		}
		if s != nil {
			enumMust(t, s.Close())
		}
		if m.Budget().Used() != baseline || c.Stats().References != refs {
			t.Fatal("constructor retained owner prefix", ordinal, m.Budget().Used(), baseline, c.Stats())
		}
		if err == nil {
			complete = true
			break
		}
	}
	if !complete {
		t.Fatal("constructor census did not complete")
	}
	s, err := portfolio.NewEnum2BVSolver(m, c, &p, b, f.services)
	enumMust(t, err)
	t.Cleanup(func() { enumMust(t, s.Close()) })
	if b.updates != 0 {
		t.Fatal("constructor called virtual backend UpdateParams")
	}
	enumMust(t, c.SetUIntName(&p, &key, 29))
	enumMust(t, s.UpdateParams(&p))
	actual, err := s.Params()
	enumMust(t, err)
	value, err := c.GetUIntName(actual, &key, 0)
	enumMust(t, err)
	if value != 29 || b.updates != 1 {
		t.Fatal("base/derived update forwarding", value, b.updates)
	}
}

func TestEnum2BVSolverBaseStatusIndependentOfBackend(t *testing.T) {
	f := enumSolverFixture(t)
	before, e := f.backend.CheckSat(nil)
	enumMust(t, e)
	if before != lbool.True {
		t.Fatal("backend status fixture", before)
	}
	s, _ := enumRecorded(t, f)
	status, err := s.Status()
	enumMust(t, err)
	if status != lbool.Undef {
		t.Fatal("source check_sat_result base did not start unknown", status)
	}
	enumMust(t, s.SetStatus(lbool.False))
	backend, err := f.backend.Status()
	enumMust(t, err)
	status, err = s.Status()
	enumMust(t, err)
	if status != lbool.False || backend != lbool.True {
		t.Fatal("source nonvirtual base status forwarded to backend", status, backend)
	}
	actual, err := s.CheckSat(nil)
	enumMust(t, err)
	status, err = s.Status()
	enumMust(t, err)
	if actual != lbool.True || status != lbool.False {
		t.Fatal("check_sat_core overwrote caller-managed source base status", actual, status)
	}
}
