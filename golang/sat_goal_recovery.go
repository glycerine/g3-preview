// Source: src/sat/tactic/atom2bool_var.cpp:25-36;
// src/sat/tactic/goal2sat.cpp:685-779; src/util/ref_vector.h:67-71,127-131;
// src/util/vector.h:422-434; src/util/buffer.h:25-48.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

// Concrete expr_ref_vector used by the inverse map and SAT recovery. Each
// populated slot owns a reference, including both polarities of each atom.
type satGoalLiteralExpressions struct{ values []ASTHandle }

func (v *satGoalLiteralExpressions) resize(m *ASTManager, n uint32) error {
	stride := uint64(reflect.TypeOf(ASTHandle{}).Size())
	for uint64(n) > uint64(cap(v.values)) {
		next, e := satGoalCapacity(cap(v.values), stride)
		if e != nil {
			return e
		}
		if e = m.budget.Reserve(uint64(next) * stride); e != nil {
			return e
		}
		data := make([]ASTHandle, len(v.values), next)
		copy(data, v.values)
		m.budget.Release(uint64(cap(v.values)) * stride)
		v.values = data
	}
	if uint64(n) < uint64(len(v.values)) {
		for i := int(n); i < len(v.values); i++ {
			if !v.values[i].IsNull() {
				if e := m.DecRef(v.values[i]); e != nil {
					return e
				}
				v.values[i] = ASTHandle{}
			}
		}
	}
	v.values = v.values[:n]
	return nil
}
func (v *satGoalLiteralExpressions) set(m *ASTManager, i uint32, h ASTHandle) error {
	if uint64(i) >= uint64(len(v.values)) {
		return arena.ErrBounds
	}
	if !h.IsNull() {
		if e := m.IncRef(h); e != nil {
			return e
		}
	}
	old := v.values[i]
	if !old.IsNull() {
		if e := m.DecRef(old); e != nil {
			if !h.IsNull() {
				return errors.Join(e, m.DecRef(h))
			}
			return e
		}
	}
	v.values[i] = h
	return nil
}
func (v *satGoalLiteralExpressions) close(m *ASTManager) error {
	if e := v.resize(m, 0); e != nil {
		return e
	}
	m.budget.Release(uint64(cap(v.values)) * uint64(reflect.TypeOf(ASTHandle{}).Size()))
	v.values = nil
	return nil
}
func (tr *satGoalTranslator) inverseAtoms(out *satGoalLiteralExpressions) error {
	if e := tr.open(); e != nil {
		return e
	}
	for _, entry := range tr.atoms.cells {
		if !entry.used {
			continue
		}
		l := satMakeLiteral(satBoolVar(entry.value), false)
		if e := out.set(tr.manager, l.index(), entry.key); e != nil {
			return e
		}
		neg, e := tr.manager.MkNot(entry.key)
		if e != nil {
			return e
		}
		if e = out.set(tr.manager, l.negated().index(), neg); e != nil {
			return e
		}
	}
	return nil
}

// This is the original sat2goal formula-recovery portion. Model-converter
// construction/application is a separate outstanding dependency; this helper
// neither reports solver status nor supplies a model converter to a tactic.
func recoverSATGoalFormulas(tr *satGoalTranslator, g *tacticGoal, learned bool, maxMemory uint64) (err error) {
	if err = tr.open(); err != nil {
		return err
	}
	if err = g.open(); err != nil {
		return err
	}
	if g.manager != tr.manager {
		return ErrASTHandle
	}
	s, m := tr.solver, tr.manager
	inconsistent, e := s.sourceInconsistent()
	if e != nil {
		return e
	}
	if inconsistent {
		f, e := m.MkFalse()
		if e != nil {
			return e
		}
		return g.Assert(f)
	}
	checkpoint := func() error {
		canceled, e := m.Canceled()
		if e != nil {
			return e
		}
		if canceled {
			return fmt.Errorf("sat2goal: %s", m.limit.cancelMessage())
		}
		bytes, e := s.runtime.memory.AllocationSize()
		if e != nil {
			return e
		}
		if bytes > maxMemory {
			return fmt.Errorf("sat2goal: max. memory exceeded")
		}
		return nil
	}
	var expressions satGoalLiteralExpressions
	defer func() { err = errors.Join(err, expressions.close(m)) }()
	n, e := s.sourceNumVars()
	if e != nil {
		return e
	}
	if n > ^uint32(0)/2 {
		return arena.ErrBounds
	}
	if e = expressions.resize(m, n*2); e != nil {
		return e
	}
	if e = tr.inverseAtoms(&expressions); e != nil {
		return e
	}
	for v := uint32(0); v < n; v++ {
		if e = checkpoint(); e != nil {
			return e
		}
		if expressions.values[2*v].IsNull() {
			if !expressions.values[2*v+1].IsNull() {
				return ErrSATAssertion
			}
			aux, e := m.MkFreshConstNullable(nil, tr.boolSort)
			if e != nil {
				return e
			}
			if e = expressions.set(m, 2*v, aux); e != nil {
				return e
			}
			neg, e := m.MkNot(aux)
			if e != nil {
				return e
			}
			if e = expressions.set(m, 2*v+1, neg); e != nil {
				return e
			}
		} else if expressions.values[2*v+1].IsNull() {
			return ErrSATAssertion
		}
	}
	for v := uint32(0); v < n; v++ {
		if e = checkpoint(); e != nil {
			return e
		}
		value, e := s.sourceValueVar(satBoolVar(v))
		if e != nil {
			return e
		}
		switch value {
		case lTrue:
			e = g.Assert(expressions.values[2*v])
		case lFalse:
			e = g.Assert(expressions.values[2*v+1])
		}
		if e != nil {
			return e
		}
	}
	bins, e := newSATSolverBinPairVector(s.runtime.memory, s.budget)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, bins.Close()) }()
	if e = s.collectBinClauses(bins, learned); e != nil {
		return e
	}
	count, e := bins.Size()
	if e != nil {
		return e
	}
	for i := uint32(0); i < count; i++ {
		if e = checkpoint(); e != nil {
			return e
		}
		pair, e := bins.Get(i)
		if e != nil {
			return e
		}
		f, e := m.MkOr(expressions.values[pair.first], expressions.values[pair.second])
		if e != nil {
			return e
		}
		if e = g.Assert(f); e != nil {
			return e
		}
	}
	assertClauses := func(clauses []satStoredClauseRef) (err error) {
		// Original ptr_buffer<expr> has sixteen inline borrowed slots and doubles
		// spill capacity. The owning literal vector stays live throughout assertion.
		var inline [16]ASTHandle
		lits := inline[:0]
		stride := uint64(reflect.TypeOf(ASTHandle{}).Size())
		defer func() {
			if cap(lits) > len(inline) {
				m.budget.Release(uint64(cap(lits)) * stride)
			}
		}()
		for _, clause := range clauses {
			if e := checkpoint(); e != nil {
				return e
			}
			lits = lits[:0]
			size, e := s.clauses.Size(clause)
			if e != nil {
				return e
			}
			for i := uint32(0); i < size; i++ {
				literal, e := s.clauses.Literal(clause, i)
				if e != nil {
					return e
				}
				if len(lits) == cap(lits) {
					capacity := uint64(cap(lits)) * 2
					if capacity > uint64(^uint(0)>>1)/stride {
						return arena.ErrBounds
					}
					if e = m.budget.Reserve(capacity * stride); e != nil {
						return e
					}
					next := make([]ASTHandle, len(lits), int(capacity))
					copy(next, lits)
					if cap(lits) > len(inline) {
						m.budget.Release(uint64(cap(lits)) * stride)
					}
					lits = next
				}
				lits = append(lits, expressions.values[literal.index()])
			}
			f, e := m.MkOrArgs(lits)
			if e != nil {
				return e
			}
			if e = g.Assert(f); e != nil {
				return e
			}
		}
		return nil
	}
	if e = assertClauses(s.lifecycle.original); e != nil {
		return e
	}
	if learned {
		return assertClauses(s.lifecycle.learned)
	}
	return nil
}
