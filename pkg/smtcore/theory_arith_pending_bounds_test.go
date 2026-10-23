package smtcore

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTSharedOptimizationBoundsWaitForPropagationAndRewindScopes(t *testing.T) {
	for _, shareZ := range []bool{false, true} {
		t.Run(fmt.Sprint("share-z=", shareZ), func(t *testing.T) {
			m, cfg := smtModelGeneratorFixture(t)
			s, err := NewSMTSolver(m, nil, "QF_UFNRA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
			astTestMust(t, err)
			t.Cleanup(func() { astTestMust(t, s.Close()) })
			s.kernel.preprocess = false
			family, err := m.ArithmeticFamilyID()
			astTestMust(t, err)
			sort, err := m.MkFamilySort(family, 0, nil)
			astTestMust(t, err)
			var terms [3]ast.ASTHandle
			for i := range terms {
				terms[i], err = m.MkConst(astTestSymbol(t, m, []string{"pending-x", "pending-y", "pending-z"}[i]), sort)
				astTestMust(t, err)
			}
			x, y, z := terms[0], terms[1], terms[2]
			app := func(op int32, args ...ast.ASTHandle) ast.ASTHandle {
				h, err := m.MkFamilyApp(family, op, nil, args)
				astTestMust(t, err)
				return h
			}
			num := func(v int64) ast.ASTHandle {
				h, err := m.ArithmeticNumeralRat(big.NewRat(v, 1), false)
				astTestMust(t, err)
				return h
			}
			f, err := m.MkFuncDecl(astTestSymbol(t, m, "pending-f"), []ast.ASTHandle{sort}, sort, ast.ASTFuncDeclInfo{})
			astTestMust(t, err)
			fx, err := m.MkApp(f, []ast.ASTHandle{x})
			astTestMust(t, err)
			eq := func(a, b ast.ASTHandle) ast.ASTHandle { h, err := m.MkEq(a, b); astTestMust(t, err); return h }
			formulas := []ast.ASTHandle{eq(x, app(smtArithOpMul, num(2), y)), app(smtArithOpGE, y, num(1)), app(smtArithOpLE, y, num(2)), app(smtArithOpGE, z, num(-10)), app(smtArithOpLE, z, num(10)), eq(app(smtArithOpMul, x, z), num(1)), eq(fx, num(0))}
			if shareZ {
				fz, err := m.MkApp(f, []ast.ASTHandle{z})
				astTestMust(t, err)
				formulas = append(formulas, eq(fz, num(0)))
			}
			for _, formula := range formulas {
				astTestMust(t, s.Assert(formula))
			}
			theory := s.kernel.internalizer.theories[family].(*SMTLinearArithmeticTheory)
			config, err := s.kernel.smtParams.ArithmeticState()
			astTestMust(t, err)
			config.NLArith, config.NLArithGB, config.NLArithBranching = false, false, false
			astTestMust(t, theory.ConfigureArithmetic(config))
			status, err := s.CheckSat(nil)
			astTestMust(t, err)
			if status != lbool.Undef {
				t.Fatal("disabled nonlinear setup", status)
			}
			xColumn, err := theory.ensureVariable(x)
			astTestMust(t, err)
			replay, status, err := theory.buildSimplex()
			astTestMust(t, err)
			if status != lbool.True {
				t.Fatal(status)
			}
			if _, exists := replay.Upper(xColumn); exists {
				t.Fatal("test already has the projected bound")
			}
			before := len(theory.derivedBounds)
			_, err = theory.maxMinNonlinearVariables()
			astTestMust(t, err)
			if !shareZ {
				if theory.CanPropagate() || theory.pendingBoundHead != len(theory.pendingBounds) || len(theory.derivedBounds) == before {
					t.Fatal("later unshared optimum failed to propagate queued bounds")
				}
				value, exists := theory.solver.Upper(xColumn)
				if !exists || value.Main.Cmp(big.NewRat(4, 1)) != 0 {
					t.Fatal("shared optimum was lost before unshared propagation", value, exists)
				}
				if _, known := theory.solver.BoundDependencies(xColumn, false); !known {
					t.Fatal("immediate propagation lost shared optimum premises")
				}
				return
			}
			if !theory.CanPropagate() || len(theory.pendingBounds) == theory.pendingBoundHead {
				t.Fatal("shared optimum did not queue a bound")
			}
			if len(theory.derivedBounds) != before {
				t.Fatal("shared optimum propagated before the scheduler")
			}
			var bound smtArithmeticReplayBound
			found := false
			for _, candidate := range theory.pendingBounds {
				if candidate.column == xColumn && !candidate.lower && candidate.value.Main.Cmp(big.NewRat(4, 1)) == 0 {
					bound = candidate
					found = true
				}
			}
			if !found || bound.explanation == nil {
				t.Fatal("missing shared upper bound or explanation", theory.pendingBounds)
			}
			queued := len(theory.pendingBounds)
			if theory.queueOptimizationBound(replay, bound) || len(theory.pendingBounds) != queued {
				t.Fatal("queued duplicate bound made progress")
			}
			yColumn, err := theory.ensureVariable(y)
			astTestMust(t, err)
			zColumn, err := theory.ensureVariable(z)
			astTestMust(t, err)
			point, status, err := theory.buildSimplexWithValues(map[uint32]*big.Rat{xColumn: big.NewRat(3, 1), yColumn: big.NewRat(3, 2), zColumn: big.NewRat(1, 3)})
			astTestMust(t, err)
			if status != lbool.True {
				t.Fatal("valid temporary model point", status)
			}
			theory.solver = point
			// Propagation must rebuild logical bounds. The temporary upper
			// endpoint x=3 must not replace the queued logical bound x<=4.
			// A parent queue can be consumed in a child scope. Popping must restore
			// its old head, while removing the child's derived copies.
			astTestMust(t, theory.PushScope())
			astTestMust(t, theory.Propagate())
			if theory.pendingBoundHead != queued || theory.CanPropagate() {
				t.Fatal("queued bounds not consumed")
			}
			value, exists := theory.solver.Upper(xColumn)
			if !exists || value.Main.Cmp(big.NewRat(4, 1)) != 0 {
				t.Fatal("projected upper bound not installed", value, exists)
			}
			if _, known := theory.solver.BoundDependencies(xColumn, false); !known {
				t.Fatal("propagation lost optimum premises")
			}
			assignment := theory.solver.Value(xColumn)
			if assignment.Main.Cmp(big.NewRat(3, 1)) != 0 || assignment.Epsilon.Sign() != 0 {
				t.Fatal("logical replay discarded the feasible assignment", assignment)
			}
			astTestMust(t, theory.PopScopes(1))
			if theory.pendingBoundHead != 0 || !theory.CanPropagate() || len(theory.derivedBounds) != before {
				t.Fatal("parent pending head or derived bounds not restored")
			}
			assignment = theory.solver.Value(xColumn)
			if assignment.Main.Cmp(big.NewRat(3, 1)) != 0 || assignment.Epsilon.Sign() != 0 {
				t.Fatal("scope pop discarded the feasible assignment", assignment)
			}
			astTestMust(t, theory.Propagate())
			if theory.CanPropagate() {
				t.Fatal("parent bounds failed to replay")
			}
			// A new child-only queue disappears entirely on pop.
			astTestMust(t, theory.PushScope())
			if !theory.queueOptimizationBound(replay, bound) {
				t.Fatal("child queue not recorded")
			}
			astTestMust(t, theory.PopScopes(1))
			if len(theory.pendingBounds) != queued || theory.CanPropagate() {
				t.Fatal("child queue survived scope pop")
			}
		})
	}
}
