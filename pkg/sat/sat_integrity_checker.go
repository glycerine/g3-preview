// Source: src/sat/sat_integrity_checker.cpp; src/sat/sat_integrity_checker.h.
package sat

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"

	"github.com/glycerine/g3/pkg/arena"
)

// A source stack checker borrows the actual solver member owners. It owns no
// clauses and has no separately allocated header. GC holds the sole frozen count.
type satIntegrityChecker struct {
	lifecycle    *satSolverClauseLifecycle
	gc           *satSolverGC
	debugActions bool
}

func newSATIntegrityChecker(l *satSolverClauseLifecycle, g *satSolverGC) (satIntegrityChecker, error) {
	if e := l.open(); e != nil {
		return satIntegrityChecker{}, e
	}
	if g == nil || g.self != g || g.lifecycle() != l {
		return satIntegrityChecker{}, ErrSATOwner
	}
	if g.closed || g.closing {
		return satIntegrityChecker{}, arena.ErrClosed
	}
	return satIntegrityChecker{lifecycle: l, gc: g}, nil
}
func (c satIntegrityChecker) open() error {
	if c.lifecycle == nil {
		return ErrSATOwner
	}
	return c.lifecycle.open()
}
func (c satIntegrityChecker) literal(cl StoredClauseRef, i uint32) (Literal, error) {
	return c.lifecycle.propagation.clauses.Literal(cl, i)
}
func (c satIntegrityChecker) eliminated(v BoolVar) (bool, error) {
	return c.lifecycle.propagation.variables.wasEliminated(v)
}
func (c satIntegrityChecker) containsTernary(index uint32, a, b Literal) (bool, error) {
	v := c.lifecycle.propagation.variables
	list, e := v.watches.Get(index)
	if e != nil {
		return false, e
	}
	ref, e := list.ref()
	if e != nil {
		return false, e
	}
	w, e := newSATTernaryWatch(a, b)
	if e != nil {
		return false, e
	}
	return v.watches.store.Contains(ref, w)
}
func (c satIntegrityChecker) containsClause(index uint32, cl StoredClauseRef, offset satClauseOffset) (bool, error) {
	p := c.lifecycle.propagation
	list, e := p.variables.watches.Get(index)
	if e != nil {
		return false, e
	}
	ref, e := list.ref()
	if e != nil {
		return false, e
	}
	it, e := p.variables.watches.store.Iter(ref)
	if e != nil {
		return false, e
	}
	for {
		w, ok, e := it.Next()
		if e != nil {
			return false, e
		}
		if !ok {
			break
		}
		if w.Clause() {
			off, e := w.ClauseOffset()
			if e != nil {
				return false, e
			}
			if off == offset {
				if satStorageSourceDebug {
					blocked, e := w.BlockedLiteral()
					if e != nil {
						return false, e
					}
					yes, e := p.clauses.ContainsLiteral(cl, blocked)
					if e != nil {
						return false, e
					}
					if !yes {
						return false, ErrSATAssertion
					}
				}
				return true, nil
			}
		}
	}
	// Source UNREACHABLE has undefined release continuation on GNU; expose a
	// checked assertion error rather than inventing a missing-watch success.
	return false, ErrSATAssertion
}
func (c satIntegrityChecker) checkClause(cl StoredClauseRef) (bool, error) {
	if e := c.open(); e != nil {
		return false, e
	}
	p := c.lifecycle.propagation
	a := p.clauses
	v := p.variables
	if satStorageSourceDebug {
		removed, e := a.Removed(cl)
		if e != nil {
			return false, e
		}
		if removed {
			return false, ErrSATAssertion
		}
		for i := uint32(0); ; i++ {
			n, e := a.Size(cl)
			if e != nil {
				return false, e
			}
			if i >= n {
				break
			}
			lit, e := a.Literal(cl, i)
			if e != nil {
				return false, e
			}
			vars, e := v.numVarsChecked()
			if e != nil {
				return false, e
			}
			if uint32(lit.variable()) > vars {
				return false, ErrSATAssertion
			}
			gone, e := c.eliminated(lit.variable())
			if e != nil {
				return false, e
			}
			if gone {
				return false, ErrSATAssertion
			}
		}
		yes, e := a.CheckApprox(cl)
		if e != nil {
			return false, e
		}
		if !yes {
			return false, ErrSATAssertion
		}
	}
	frozen, e := a.Frozen(cl)
	if e != nil {
		return false, e
	}
	if frozen {
		return true, nil
	}
	n, e := a.Size(cl)
	if e != nil {
		return false, e
	}
	if n == 3 {
		// Each VERIFY expression independently reads its source fields, in order.
		for k := uint32(0); k < 3; k++ {
			x, e := a.Literal(cl, k)
			if e != nil {
				return false, e
			}
			first, second := uint32(1), uint32(2)
			if k == 1 {
				first, second = 0, 2
			}
			if k == 2 {
				first, second = 0, 1
			}
			y, e := a.Literal(cl, first)
			if e != nil {
				return false, e
			}
			z, e := a.Literal(cl, second)
			if e != nil {
				return false, e
			}
			yes, e := c.containsTernary(x.Negated().index(), y, z)
			if e != nil {
				return false, e
			}
			if !yes {
				return false, ErrSATAssertion
			}
		}
	} else {
		first, e := a.Literal(cl, 0)
		if e != nil {
			return false, e
		}
		value, e := v.valueLiteral(first)
		if e != nil {
			return false, e
		}
		watchedFalse := value == lbool.False
		if !watchedFalse {
			second, e := a.Literal(cl, 1)
			if e != nil {
				return false, e
			}
			value, e = v.valueLiteral(second)
			if e != nil {
				return false, e
			}
			watchedFalse = value == lbool.False
		}
		if watchedFalse {
			onStack := false
			for i := v.qhead; ; i++ {
				size, e := v.trail.Size(v.trailRef)
				if e != nil {
					return false, e
				}
				if i >= size {
					break
				}
				trail, e := v.trailRead(i)
				if e != nil {
					return false, e
				}
				x, e := a.Literal(cl, 0)
				if e != nil {
					return false, e
				}
				same := trail.variable() == x.variable()
				if !same {
					x, e = a.Literal(cl, 1)
					if e != nil {
						return false, e
					}
					same = trail.variable() == x.variable()
				}
				if same {
					onStack = true
					break
				}
			}
			if !onStack {
				status, e := p.status(cl)
				if e != nil {
					return false, e
				}
				if status != lbool.True {
					for i := uint32(2); ; i++ {
						size, e := a.Size(cl)
						if e != nil {
							return false, e
						}
						if i >= size {
							break
						}
						if satStorageSourceDebug {
							lit, e := a.Literal(cl, i)
							if e != nil {
								return false, e
							}
							val, e := v.valueLiteral(lit)
							if e != nil {
								return false, e
							}
							if val != lbool.False {
								return false, ErrSATAssertion
							}
						}
					}
				}
			}
		}
		for k := uint32(0); k < 2; k++ {
			x, e := a.Literal(cl, k)
			if e != nil {
				return false, e
			}
			offset, e := a.Offset(cl)
			if e != nil {
				return false, e
			}
			yes, e := c.containsClause(x.Negated().index(), cl, offset)
			if e != nil {
				return false, e
			}
			if !yes {
				return false, ErrSATAssertion
			}
		}
	}
	return true, nil
}
func (c satIntegrityChecker) checkClauseVector(cs []StoredClauseRef) (bool, error) {
	if e := c.open(); e != nil {
		return false, e
	}
	for i := 0; i < len(cs); i++ {
		if satStorageSourceDebug {
			yes, e := c.checkClause(cs[i])
			if e != nil {
				return false, e
			}
			if !yes {
				return false, ErrSATAssertion
			}
		}
	}
	return true, nil
}
func (c satIntegrityChecker) checkClauses() (bool, error) {
	return c.checkClauseVector(c.lifecycle.Original)
}
func (c satIntegrityChecker) checkLearnedClauses() (bool, error) {
	if e := c.open(); e != nil {
		return false, e
	}
	count := uint32(0)
	for _, cl := range c.lifecycle.Original {
		frozen, e := c.lifecycle.propagation.clauses.Frozen(cl)
		if e != nil {
			return false, e
		}
		if frozen {
			count++
		}
	}
	if satStorageSourceDebug {
		if c.gc == nil || c.gc.self != c.gc || c.gc.lifecycle() != c.lifecycle {
			return false, ErrSATOwner
		}
		if c.gc.closed || c.gc.closing {
			return false, arena.ErrClosed
		}
		if count != c.gc.numFrozen {
			return false, ErrSATAssertion
		}
	}
	return c.checkClauseVector(c.lifecycle.learned)
}
func (c satIntegrityChecker) checkAssignment() (bool, error) {
	if e := c.open(); e != nil {
		return false, e
	}
	return true, nil
}
func (c satIntegrityChecker) checkBoolVars() (bool, error) {
	if e := c.open(); e != nil {
		return false, e
	}
	v := c.lifecycle.propagation.variables
	sourceAssertions := false
	if satStorageSourceDebug {
		enabled, e := diagnostic.BorrowProcessDebugContext().AssertionsEnabled()
		if e != nil {
			return false, e
		}
		sourceAssertions = enabled
		if enabled {
			n, e := v.numVarsChecked()
			if e != nil {
				return false, e
			}
			checks := [...]struct {
				got       uint32
				want      uint32
				line      int32
				condition string
			}{
				{uint32(len(v.watches.lists)), n * 2, 133, "s.m_watches.size() == s.num_vars() * 2"},
				{uint32(len(v.assignment)), n * 2, 134, "s.m_assignment.size() == s.num_vars() * 2"},
				{uint32(len(v.literalMark)), n * 2, 135, "s.m_lit_mark.size() == s.num_vars() * 2"},
				{0, n, 136, "s.m_justification.size() == s.num_vars()"},
				{uint32(len(v.decision)), n, 137, "s.m_decision.size() == s.num_vars()"},
				{uint32(len(v.eliminated)), n, 138, "s.m_eliminated.size() == s.num_vars()"},
				{uint32(len(v.external)), n, 139, "s.m_external.size() == s.num_vars()"},
				{uint32(len(v.level)), n, 140, "s.m_level.size() == s.num_vars()"},
				{uint32(len(v.mark)), n, 141, "s.m_mark.size() == s.num_vars()"},
				{0, n, 142, "s.m_activity.size() == s.num_vars()"},
				{uint32(len(v.phase)), n, 143, "s.m_phase.size() == s.num_vars()"},
				{uint32(len(v.previousPhase)), n, 144, "s.m_prev_phase.size() == s.num_vars()"},
				{uint32(len(v.assignedSinceGC)), n, 145, "s.m_assigned_since_gc.size() == s.num_vars()"},
			}
			for i := range checks {
				if checks[i].line == 136 {
					checks[i].got, e = v.justification.size()
				} else if checks[i].line == 142 {
					e = v.activity.open()
					if e == nil {
						checks[i].got = uint32(len(v.activity.values))
					}
				}
				if e != nil {
					return false, e
				}
				if e = satIntegrityAssertion(checks[i].got == checks[i].want, c.debugActions, checks[i].line, checks[i].condition); e != nil {
					return false, e
				}
			}
		}
	}
	for variable := uint32(0); ; variable++ {
		n, e := v.numVarsChecked()
		if e != nil {
			return false, e
		}
		if variable >= n {
			break
		}
		gone, e := v.wasEliminated(BoolVar(variable))
		if e != nil {
			return false, e
		}
		if gone && sourceAssertions {
			for sign := uint32(0); sign < 2; sign++ {
				list, e := v.watches.Get(variable*2 + sign)
				if e != nil {
					return false, e
				}
				size, e := list.Size()
				if e != nil {
					return false, e
				}
				if size != 0 {
					return false, ErrSATAssertion
				}
			}
		}
	}
	return true, nil
}
func (c satIntegrityChecker) checkWatches() (bool, error) {
	if e := c.open(); e != nil {
		return false, e
	}
	if !satStorageSourceDebug {
		return true, nil
	}
	p := c.lifecycle.propagation
	v := p.variables
	for index := uint32(0); uint64(index) < uint64(len(v.watches.lists)); index++ {
		literal := Literal(index).Negated()
		list, e := v.watches.Get(index)
		if e != nil {
			return false, e
		}
		gone, e := v.wasEliminated(literal.variable())
		if e != nil {
			return false, e
		}
		if gone {
			size, e := list.Size()
			if e != nil {
				return false, e
			}
			if size != 0 {
				return false, ErrSATAssertion
			}
		}
		ref, e := list.ref()
		if e != nil {
			return false, e
		}
		it, e := v.watches.store.Iter(ref)
		if e != nil {
			return false, e
		}
		for {
			w, ok, e := it.Next()
			if e != nil {
				return false, e
			}
			if !ok {
				break
			}
			switch w.Kind() {
			case 0:
				other, e := w.Literal()
				if e != nil {
					return false, e
				}
				gone, e := v.wasEliminated(other.variable())
				if e != nil {
					return false, e
				}
				if gone {
					return false, ErrSATAssertion
				}
				counterpart, e := v.watches.Get(other.Negated().index())
				if e != nil {
					return false, e
				}
				ref, e := counterpart.ref()
				if e != nil {
					return false, e
				}
				learned, e := w.Learned()
				if e != nil {
					return false, e
				}
				expected, e := newSATBinaryWatch(literal, learned)
				if e != nil {
					return false, e
				}
				yes, e := v.watches.store.Contains(ref, expected)
				if e != nil {
					return false, e
				}
				if !yes {
					return false, ErrSATAssertion
				}
			case 1:
				one, e := w.Literal1()
				if e != nil {
					return false, e
				}
				gone, e := v.wasEliminated(one.variable())
				if e != nil {
					return false, e
				}
				if gone {
					return false, ErrSATAssertion
				}
				two, e := w.Literal2()
				if e != nil {
					return false, e
				}
				gone, e = v.wasEliminated(two.variable())
				if e != nil {
					return false, e
				}
				if gone {
					return false, ErrSATAssertion
				}
				if one.index() >= two.index() {
					return false, ErrSATAssertion
				}
			case 2:
				off, e := w.ClauseOffset()
				if e != nil {
					return false, e
				}
				cl, e := p.clauses.GetClause(off)
				if e != nil {
					return false, e
				}
				removed, e := p.clauses.Removed(cl)
				if e != nil {
					return false, e
				}
				if removed {
					return false, ErrSATAssertion
				}
			}
		}
	}
	return true, nil
}
func (c satIntegrityChecker) checkReinitStack() (bool, error) {
	if e := c.open(); e != nil {
		return false, e
	}
	for _, w := range c.lifecycle.reinit {
		if w.Binary() {
			continue
		}
		if satStorageSourceDebug {
			cl, e := w.ClauseRef()
			if e != nil {
				return false, e
			}
			yes, e := c.lifecycle.propagation.clauses.OnReinitStack(cl)
			if e != nil {
				return false, e
			}
			if !yes {
				return false, ErrSATAssertion
			}
		}
	}
	return true, nil
}
func (c satIntegrityChecker) checkDisjointClauses() (yes bool, err error) {
	if e := c.open(); e != nil {
		return false, e
	}
	// Fresh source uint_set every invocation; use its existing real storage and
	// algorithms. Its cold Go owner allocation is an explicitly charged adapter.
	ids, e := newSATSolverUintSet(c.lifecycle.budget)
	if e != nil {
		return false, e
	}
	defer func() {
		if e := ids.Close(); e != nil {
			err = errors.Join(err, e)
		}
	}()
	for _, cl := range c.lifecycle.Original {
		id, e := c.lifecycle.propagation.clauses.ID(cl)
		if e != nil {
			return false, e
		}
		if e = ids.Insert(id); e != nil {
			return false, e
		}
	}
	for _, cl := range c.lifecycle.learned {
		id, e := c.lifecycle.propagation.clauses.ID(cl)
		if e != nil {
			return false, e
		}
		if ids.Contains(id) {
			return false, nil
		}
	}
	return true, nil
}
func (c satIntegrityChecker) run() (bool, error) {
	if e := c.open(); e != nil {
		return false, e
	}
	if c.lifecycle.propagation.inconsistent {
		return true, nil
	}
	if satStorageSourceDebug {
		yes, e := c.checkClauses()
		if e != nil {
			return false, e
		}
		if !yes {
			return false, ErrSATAssertion
		}
		yes, e = c.checkLearnedClauses()
		if e != nil {
			return false, e
		}
		if !yes {
			return false, ErrSATAssertion
		}
		yes, e = c.checkWatches()
		if e != nil {
			return false, e
		}
		if !yes {
			return false, ErrSATAssertion
		}
		yes, e = c.checkBoolVars()
		if e != nil {
			return false, e
		}
		if !yes {
			return false, ErrSATAssertion
		}
		yes, e = c.checkReinitStack()
		if e != nil {
			return false, e
		}
		if !yes {
			return false, ErrSATAssertion
		}
		yes, e = c.checkDisjointClauses()
		if e != nil {
			return false, e
		}
		if !yes {
			return false, ErrSATAssertion
		}
	}
	return true, nil
}
