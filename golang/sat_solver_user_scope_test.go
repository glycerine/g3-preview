//go:build port_pending_sat

// Copyright (c) 2015 Microsoft Corporation. Licensed under the MIT license.
// Finite translation of src/test/sat_user_scope.cpp; exact oracle is Go-only.
package g3

import (
	"testing"

	"github.com/glycerine/g3/golang/utilhash"
)

// The source campaign never returns. One finite source outer cycle performs
// 42 executions of its one registered ENSURE site and creates 168 clauses.
// Test-owned cold arrays retain the original formulas; they are not solver
// storage and the exhaustive oracle is never used by production search.
func TestSATSolverSourceUserScopeFiniteCycle(t *testing.T) {
	ctx, _, _ := paramsTestOwner(t)
	p, err := ctx.NewValue()
	paramsMust(t, err)
	t.Cleanup(func() { paramsMust(t, ctx.Del(&p)) })
	runtime, err := newSATRuntime(satRuntimeConfig{Budget: ctx.budget, ParamsContext: ctx})
	paramsMust(t, err)
	t.Cleanup(func() { paramsMust(t, runtime.Close()) })
	limit, err := newUtilReslimit(ctx.budget)
	paramsMust(t, err)
	t.Cleanup(limit.close)
	config := satSolverConfig{ParamsContext: ctx, Params: p, RLimit: limit, Budget: ctx.budget, Runtime: runtime}
	solver, err := NewSATSolver(config)
	paramsMust(t, err)
	t.Cleanup(func() { paramsMust(t, solver.Close()) })
	for i := 0; i <= 6; i++ {
		v, err := solver.MkVar(false, true)
		paramsMust(t, err)
		if v != satBoolVar(i) {
			t.Fatalf("source initial variable %d: %d", i, v)
		}
	}
	var rng utilhash.Random
	var frames [][][3]satLiteral
	checks, clauses := 0, 0
	check := func() {
		t.Helper()
		freshLimit, err := newUtilReslimit(ctx.budget)
		paramsMust(t, err)
		defer freshLimit.close()
		freshConfig := config
		freshConfig.RLimit = freshLimit
		fresh, err := NewSATSolver(freshConfig)
		paramsMust(t, err)
		defer func() { paramsMust(t, fresh.Close()) }()
		for i := 0; i <= 6; i++ {
			_, err = fresh.MkVar(false, true)
			paramsMust(t, err)
		}
		for _, frame := range frames {
			for _, original := range frame {
				copyForSourceSort := original
				paramsMust(t, fresh.MkClause(copyForSourceSort[:]))
			}
		}
		incrementalResult, err := solver.Check(nil)
		paramsMust(t, err)
		freshResult, err := fresh.Check(nil)
		paramsMust(t, err)
		// Actual source sat_user_scope.cpp:75 ENSURE.
		if incrementalResult != freshResult {
			t.Fatalf("source coherence check %d: incremental=%v rebuilt=%v", checks, incrementalResult, freshResult)
		}
		want := satSolverScopeOracle(frames)
		if incrementalResult != want {
			t.Fatalf("independent original-CNF oracle check %d: got=%v want=%v", checks, incrementalResult, want)
		}
		if incrementalResult == lTrue {
			model, err := solver.AppendModel(nil)
			paramsMust(t, err)
			if !satSolverScopeModelSatisfies(frames, model) {
				t.Fatalf("source model reconstruction check %d: %v", checks, model)
			}
		}
		checks++
	}
	for i := 0; i < 7; i++ {
		for k := 0; k < 3; k++ {
			paramsMust(t, solver.UserPush())
			frames = append(frames, nil)
			for j := 0; j < 8; j++ {
				var clause [3]satLiteral
				for q := range clause {
					// Explicit deterministic Go order for source's unspecified
					// C++ constructor argument evaluation: variable then sign.
					v := rng.Bounded(6) + 1
					sign := rng.Bounded(2) == 0
					clause[q] = satMakeLiteral(satBoolVar(v), sign)
				}
				frames[len(frames)-1] = append(frames[len(frames)-1], clause)
				paramsMust(t, solver.MkClause(clause[:]))
				clauses++
			}
			check()
		}
		for k := 0; k < 2; k++ {
			paramsMust(t, solver.UserPop(1))
			frames = frames[:len(frames)-1]
			check()
		}
	}
	for i := 0; i < 7; i++ {
		paramsMust(t, solver.UserPop(1))
		frames = frames[:len(frames)-1]
		check()
	}
	if checks != 42 || clauses != 168 || len(frames) != 0 {
		t.Fatalf("finite source campaign counts checks=%d clauses=%d frames=%d", checks, clauses, len(frames))
	}
}

func satSolverScopeOracle(frames [][][3]satLiteral) lbool {
	for bits := uint32(0); bits < 64; bits++ {
		good := true
		for _, frame := range frames {
			for _, clause := range frame {
				clauseTrue := false
				for _, l := range clause {
					value := bits&(1<<(uint32(l.variable())-1)) != 0
					clauseTrue = clauseTrue || value != l.sign()
				}
				good = good && clauseTrue
			}
		}
		if good {
			return lTrue
		}
	}
	return lFalse
}

func satSolverScopeModelSatisfies(frames [][][3]satLiteral, model []lbool) bool {
	for _, frame := range frames {
		for _, clause := range frame {
			good := false
			for _, l := range clause {
				if uint64(l.variable()) >= uint64(len(model)) {
					return false
				}
				good = good || model[l.variable()] == lTrue && !l.sign() || model[l.variable()] == lFalse && l.sign()
			}
			if !good {
				return false
			}
		}
	}
	return true
}
