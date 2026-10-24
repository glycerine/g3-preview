package ackermannization_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ackermannization"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type lazyCheckFixture struct {
	m        *ast.ASTManager
	services model.ModelEvaluatorConfig
	info     *ackermannization.Info
	checker  *ackermannization.LazyModelChecker
	md       *model.Model
	x, y     ast.ASTHandle
	a, b     ast.ASTHandle
	fx, fy   ast.ASTHandle
}

func newLazyCheckFixture(t *testing.T, width uint32, nested bool) *lazyCheckFixture {
	t.Helper()
	c := ackContext(t, fmt.Sprintf(`(set-logic QF_UFBV)
(declare-const x (_ BitVec %[1]d))(declare-const y (_ BitVec %[1]d))
(declare-const a (_ BitVec %[1]d))(declare-const b (_ BitVec %[1]d))
(declare-fun f ((_ BitVec %[1]d)) (_ BitVec %[1]d))`, width))
	m, err := c.Manager()
	ackMust(t, err)
	f := &lazyCheckFixture{m: m}
	app := func(name string, args ...ast.ASTHandle) ast.ASTHandle {
		h, err := c.MakeApp(name, args, nil, ast.ASTHandle{})
		ackMust(t, err)
		ref, err := m.NewRef(h)
		ackMust(t, err)
		t.Cleanup(func() { ackMust(t, m.DelRef(&ref)) })
		return h
	}
	f.x, f.y, f.a, f.b = app("x"), app("y"), app("a"), app("b")
	x, y := f.x, f.y
	if nested {
		// Values must be computed by the empty-model evaluator before keys
		// for f are compared, rather than comparing argument syntax.
		x = app("bvadd", f.x, f.x)
		y = app("bvadd", f.y, f.y)
	}
	f.fx, f.fy = app("f", x), app("f", y)
	f.info, err = ackermannization.NewInfo(m)
	ackMust(t, err)
	ackMust(t, f.info.IncRef())
	t.Cleanup(func() { ackMust(t, f.info.DecRef()) })
	ackMust(t, f.info.SetAbstraction(f.fx, f.a))
	ackMust(t, f.info.SetAbstraction(f.fy, f.b))
	ackMust(t, f.info.Seal())
	f.md, err = model.NewModel(m)
	ackMust(t, err)
	var held tactic.ModelRef
	ackMust(t, held.Set(f.md.Core()))
	t.Cleanup(func() { ackMust(t, held.Close()) })
	f.services = ackServices(t, c)
	f.checker, err = ackermannization.NewLazyModelChecker(m, f.info, f.services)
	if f.checker != nil {
		t.Cleanup(func() { ackMust(t, f.checker.Close()) })
	}
	ackMust(t, err)
	return f
}
func (f *lazyCheckFixture) set(t *testing.T, constant ast.ASTHandle, value, width uint32) {
	t.Helper()
	d, err := f.m.GetDecl(constant)
	ackMust(t, err)
	v, err := f.m.BitVectorNumeralUnsigned(value, width)
	ackMust(t, err)
	ackMust(t, f.md.Core().RegisterConstant(d, v))
}

func TestLazyModelCheckCongruenceCPP(t *testing.T) {
	for _, width := range []uint32{1, 4, 65, 257} {
		for _, nested := range []bool{false, true} {
			for bits := uint32(0); bits < 16; bits++ {
				t.Run(fmt.Sprintf("%d/nested-%v/%d", width, nested, bits), func(t *testing.T) {
					f := newLazyCheckFixture(t, width, nested)
					x, y, a, b := bits&1, (bits>>1)&1, (bits>>2)&1, (bits>>3)&1
					for i, h := range []ast.ASTHandle{f.x, f.y, f.a, f.b} {
						f.set(t, h, []uint32{x, y, a, b}[i], width)
					}
					want := x != y || a == b
					if nested && width == 1 {
						want = a == b // Doubling wraps both inputs to zero.
					}
					ok, err := f.checker.Check(f.md.Core())
					ackMust(t, err)
					if ok != want {
						t.Fatal("congruence result", ok, "want", want)
					}
					if !ok {
						conflicts, err := f.checker.Conflicts()
						ackMust(t, err)
						if len(conflicts) != 1 || conflicts[0].First != f.fy || conflicts[0].Second != f.fx {
							t.Fatal("conflict pair or constant order", conflicts)
						}
					}
					if os.Getenv("G3_CPP_REFERENCE") != "" {
						termX, termY := "x", "y"
						if nested {
							termX, termY = "(bvadd x x)", "(bvadd y y)"
						}
						script := fmt.Sprintf(`(set-logic QF_UFBV)(declare-const x (_ BitVec %[1]d))(declare-const y (_ BitVec %[1]d))(declare-fun f ((_ BitVec %[1]d)) (_ BitVec %[1]d))(assert (= x (_ bv%[2]d %[1]d)))(assert (= y (_ bv%[3]d %[1]d)))(assert (= (f %[6]s) (_ bv%[4]d %[1]d)))(assert (= (f %[7]s) (_ bv%[5]d %[1]d)))(check-sat)`, width, x, y, a, b, termX, termY)
						cpp := strings.TrimSpace(ackCPP(t, script))
						if cpp != map[bool]string{true: "sat", false: "unsat"}[ok] {
							t.Fatal("C++ congruence result", cpp, ok)
						}
					}
				})
			}
		}
	}
}

func TestLazyModelCheckRebuildCompletionAndTrace(t *testing.T) {
	f := newLazyCheckFixture(t, 4, true)
	// Missing input constants use get_some_value without mutating the abstract
	// model. Equal argument values with unequal abstraction values conflict.
	f.set(t, f.a, 1, 4)
	f.set(t, f.b, 2, 4)
	var trace bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&trace)
	tags.Enable("model_constructor")
	ackMust(t, f.checker.BindTrace(tags))
	ok, err := f.checker.Check(f.md.Core())
	ackMust(t, err)
	if ok {
		t.Fatal("missing input completion did not collide")
	}
	n, err := f.md.Core().NumConstants()
	ackMust(t, err)
	if n != 2 {
		t.Fatal("checker completed the input model", n)
	}
	if tags.ShouldTrace("model_constructor") {
		for _, s := range []string{"mk_value(", "arg val 0(", "eval(", "already mapped by(", "lackr_model_constructor.cpp:308"} {
			if !strings.Contains(trace.String(), s) {
				t.Fatal("missing model-check trace", s, trace.String())
			}
		}
	}
	// Every check destroys the old caches and creates new ones.
	f.set(t, f.y, 1, 4)
	ok, err = f.checker.Check(f.md.Core())
	ackMust(t, err)
	if !ok {
		t.Fatal("old argument values survived the next check")
	}
	f.set(t, f.y, 0, 4)
	ok, err = f.checker.Check(f.md.Core())
	ackMust(t, err)
	if ok {
		t.Fatal("new conflict not found")
	}
	conflicts, err := f.checker.Conflicts()
	ackMust(t, err)
	if len(conflicts) != 1 {
		t.Fatal("old conflicts survived rebuild", conflicts)
	}
	conflicts[0] = ackermannization.CongruenceConflict{}
	again, err := f.checker.Conflicts()
	ackMust(t, err)
	if again[0].First.IsNull() {
		t.Fatal("caller mutated owned conflict storage")
	}
}

func TestLazyModelCheckAllocationFrontiers(t *testing.T) {
	for _, phase := range []string{"constructor", "check", "recheck"} {
		t.Run(phase, func(t *testing.T) {
			success := false
			for stop := int64(0); stop < 512 && !success; stop++ {
				t.Run(fmt.Sprint(stop), func(t *testing.T) {
					f := newLazyCheckFixture(t, 4, true)
					f.set(t, f.a, 1, 4)
					f.set(t, f.b, 2, 4)
					if phase == "recheck" {
						_, err := f.checker.Check(f.md.Core())
						ackMust(t, err)
					}
					f.m.Budget().FailAfter(stop)
					var err error
					if phase == "constructor" {
						var fresh *ackermannization.LazyModelChecker
						fresh, err = ackermannization.NewLazyModelChecker(f.m, f.info, f.services)
						f.m.Budget().FailAfter(-1)
						if fresh != nil {
							ackMust(t, fresh.Close())
						}
					} else {
						_, err = f.checker.Check(f.md.Core())
					}
					f.m.Budget().FailAfter(-1)
					if err == nil {
						success = true
					} else if !errors.Is(err, arena.ErrInjectedFailure) {
						t.Fatal(err)
					}
					// Check retries must retire all partial evaluator/map
					// state and produce the original semantic answer.
					ok, err := f.checker.Check(f.md.Core())
					ackMust(t, err)
					if ok {
						t.Fatal("recovery lost the congruence conflict")
					}
				})
				if t.Failed() {
					t.Fatal("reservation frontier", phase, stop)
				}
			}
			if !success {
				t.Fatal("no successful reservation frontier", phase)
			}
		})
	}
}
