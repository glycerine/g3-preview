// Source: src/sat/sat_mus.cpp:28–229; src/sat/sat_mus.h; src/sat/sat_solver.cpp:1257–1261,2072–2083.
// Licensed under the MIT license. Source author: Nikolaj Bjorner, 2014.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

var ErrSATMUSEmptyRecursion = errors.New("sat: empty MUS qx recursion has no finite source continuation")

// A dynamic lexical frame names actual source local value owners. Its reflected
// cold header is a Go ownership adapter, not an extra source vector allocation.
// Borrowed qx arguments never own the same backing twice. Failed source bodies
// retain interrupted destructors here, without restoring outputs or marks.
type satMUSFrame struct {
	appendActive       bool
	appendOld          uint32
	innerCoreCompleted bool
	next               *satMUSFrame
	sets               [3]satMUSLiteralSet
	vector             satSolverRuntimeUintColumn
	setsUsed           uint32
	vectorUsed         bool
	drain              uint32
	cleanupStopped     bool
	bytes              uint64
}

func (m *satSolverMUSMembers) frame(sets uint32, vector bool) (*satMUSFrame, error) {
	s := m.solver
	n := uint64(reflect.TypeOf(satMUSFrame{}).Size())
	if e := s.runtime.budget.Reserve(n); e != nil {
		return nil, e
	}
	f := &satMUSFrame{next: m.frames, setsUsed: sets, vectorUsed: vector, bytes: n}
	for i := uint32(0); i < sets; i++ {
		f.sets[i].bind(s.runtime.memory)
	}
	f.vector.bind(s.runtime.memory)
	m.frames = f
	return f, nil
}
func (m *satSolverMUSMembers) drop(f *satMUSFrame, cause error) error {
	if f.cleanupStopped {
		return cause
	}
	if m.frames != f {
		if cause != nil {
			return cause
		}
		return ErrSATOwner
	}
	// Source locals unwind in reverse declaration order, interrupted at the
	// actual reached release. A failed constructor's buffers are owning-aborted
	// only after the older completed locals; current constructors are empty.
	if f.vectorUsed {
		if e := f.vector.finalize(); e != nil {
			return errors.Join(cause, e)
		}
		f.vectorUsed = false
	}
	if f.innerCoreCompleted {
		if e := f.sets[0].close(); e != nil {
			return errors.Join(cause, e)
		}
		f.innerCoreCompleted = false
		f.setsUsed = 0
	}
	if f.appendActive {
		if e := satMUSResize(&m.mus, f.appendOld); e != nil {
			return errors.Join(cause, e)
		}
		f.appendActive = false
	}
	for f.drain < f.setsUsed {
		i := f.setsUsed - 1 - f.drain
		if e := f.sets[i].close(); e != nil {
			return errors.Join(cause, e)
		}
		f.drain++
	}
	if m.frames != f {
		return errors.Join(cause, ErrSATOwner)
	}
	m.frames = f.next
	m.solver.runtime.budget.Release(f.bytes)
	return cause
}
func (m *satSolverMUSMembers) cleanup() error {
	for m.frames != nil {
		m.frames.cleanupStopped = false
		if e := m.drop(m.frames, nil); e != nil {
			return e
		}
	}
	return nil
}
func satMUSAppend(dst, src *satSolverRuntimeUintColumn) error {
	for i := uint32(0); ; i++ {
		n, e := src.size()
		if e != nil {
			return e
		}
		if i >= n {
			return nil
		}
		v, e := src.read(i)
		if e != nil {
			return e
		}
		if e = dst.push(v); e != nil {
			return e
		}
	}
}
func satMUSContains(v *satSolverRuntimeUintColumn, l uint32) (bool, error) {
	for i := uint32(0); ; i++ {
		n, e := v.size()
		if e != nil {
			return false, e
		}
		if i >= n {
			return false, nil
		}
		a, e := v.read(i)
		if e != nil {
			return false, e
		}
		if a == l {
			return true, nil
		}
	}
}
func (m *satSolverMUSMembers) reset() error {
	if e := m.core.shrink(0); e != nil {
		return e
	}
	if e := m.mus.shrink(0); e != nil {
		return e
	}
	return m.model.shrink(0)
}
func (m *satSolverMUSMembers) setCore() error {
	if e := satMUSAppend(&m.mus, &m.core); e != nil {
		return e
	}
	if e := m.solver.conflict.core.shrink(0); e != nil {
		return e
	}
	return satMUSAppend(&m.solver.conflict.core, &m.mus)
}
func (m *satSolverMUSMembers) updateModel() error {
	n, e := m.model.size()
	if e != nil {
		return e
	}
	if n == 0 {
		return satMUSAppend(&m.model, &m.solver.model)
	}
	return nil
}
func (m *satSolverMUSMembers) getCore() error {
	if e := m.core.shrink(0); e != nil {
		return e
	}
	if e := m.mus.shrink(0); e != nil {
		return e
	}
	if e := satMUSAppend(&m.core, &m.solver.conflict.core); e != nil {
		return e
	}
	for i := uint32(0); ; i++ {
		n, e := m.core.size()
		if e != nil {
			return e
		}
		if i >= n {
			return nil
		}
		v, e := m.core.read(i)
		if e != nil {
			return e
		}
		yes, e := satMUSContains(&m.solver.conflict.userScopeLiterals, v)
		if e != nil {
			return e
		}
		if yes {
			v, e = m.core.read(i)
			if e != nil {
				return e
			}
			if e = m.mus.push(v); e != nil {
				return e
			}
			n, e = m.core.size()
			if e != nil {
				return e
			}
			v, e = m.core.read(n - 1)
			if e != nil {
				return e
			}
			if e = m.core.write(i, v); e != nil {
				return e
			}
			if e = m.core.shrink(n - 1); e != nil {
				return e
			}
			i--
		}
	}
}
func (m *satSolverMUSMembers) run() (lbool, error) {
	if m == nil || m.solver == nil || m != &m.solver.mus {
		return lUndef, ErrSATOwner
	}
	if e := m.solver.open(); e != nil {
		return lUndef, e
	}
	if m.frames != nil || m.isActive {
		return lUndef, arena.ErrClosed
	}
	cfg := &m.solver.config.state
	if cfg.CoreMinimizePartial {
		m.maxNumRestarts = m.solver.gc.restarts + 10
	} else {
		m.maxNumRestarts = ^uint32(0)
	}
	oldMin := cfg.CoreMinimize
	cfg.CoreMinimize = false
	oldActive := m.isActive
	m.isActive = true
	defer func() { m.isActive = oldActive; cfg.CoreMinimize = oldMin }()
	if e := m.reset(); e != nil {
		return lUndef, e
	}
	return m.mus1()
}
func (m *satSolverMUSMembers) scopedCheck(core *satSolverRuntimeUintColumn, lit satLiteral) (r lbool, e error) {
	cfg := &m.solver.config.state
	oldRestart := cfg.RestartMax
	cfg.RestartMax = m.maxNumRestarts
	defer func() { cfg.RestartMax = oldRestart }()
	n, e := m.mus.size()
	if e != nil {
		return lUndef, e
	}
	// Failed scoped_append construction has no source destructor. Completed
	// append restores size only after its actual nested Check returns/unwinds.
	if e = satMUSAppend(&m.mus, core); e != nil {
		return lUndef, e
	}
	defer func() {
		if cleanup := satMUSResize(&m.mus, n); cleanup != nil {
			e = errors.Join(e, cleanup)
		}
	}()
	if e = m.mus.push(uint32(lit.negated())); e != nil {
		return lUndef, e
	}
	count, e := m.mus.size()
	if e != nil {
		return lUndef, e
	}
	return m.solver.checkSpan(count, satConflictRuntimeSpan(&m.mus))
}
func (m *satSolverMUSMembers) mus1() (lbool, error) {
	partial := m.solver.config.state.CoreMinimizePartial
	if e := m.getCore(); e != nil {
		return lUndef, e
	}
	n, e := m.core.size()
	if e != nil {
		return lUndef, e
	}
	if !partial && n > 64 {
		return m.mus2()
	}
	for {
		n, e = m.core.size()
		if e != nil {
			return lUndef, e
		}
		if n == 0 {
			break
		}
		if !m.solver.limit.inc() {
			if e = m.setCore(); e != nil {
				return lUndef, e
			}
			return lUndef, nil
		}
		k, e := m.mus.size()
		if e != nil {
			return lUndef, e
		}
		if n+k <= 2 {
			break
		}
		v, e := m.core.read(n - 1)
		if e != nil {
			return lUndef, e
		}
		if e = m.core.shrink(n - 1); e != nil {
			return lUndef, e
		}
		lit := satLiteral(v)
		result, e := m.scopedCheck(&m.core, lit)
		if e != nil {
			return lUndef, e
		}
		switch result {
		case lUndef:
			if m.solver.limit.inc() {
				if e = m.mus.push(v); e != nil {
					return lUndef, e
				}
			} else {
				if e = m.core.push(v); e != nil {
					return lUndef, e
				}
				if e = m.setCore(); e != nil {
					return lUndef, e
				}
				return lUndef, nil
			}
		case lTrue:
			if satStorageSourceDebug {
				a, e := m.solver.model.read(uint32(lit.variable()))
				if e != nil {
					return lUndef, e
				}
				value := lbool(int32(a))
				if lit.sign() {
					value = value.negated()
				}
				if value != lFalse {
					return lUndef, ErrSATAssertion
				}
			}
			if e = m.mus.push(v); e != nil {
				return lUndef, e
			}
			if e = m.updateModel(); e != nil {
				return lUndef, e
			}
		case lFalse:
			yes, e := satMUSContains(&m.solver.conflict.core, uint32(lit.negated()))
			if e != nil {
				return lUndef, e
			}
			if !yes {
				if e = m.core.shrink(0); e != nil {
					return lUndef, e
				}
				for i := uint32(0); ; i++ {
					sz, e := m.solver.conflict.core.size()
					if e != nil {
						return lUndef, e
					}
					if i >= sz {
						break
					}
					v, e := m.solver.conflict.core.read(i)
					if e != nil {
						return lUndef, e
					}
					yes, e := satMUSContains(&m.mus, v)
					if e != nil {
						return lUndef, e
					}
					if !yes {
						if e = m.core.push(v); e != nil {
							return lUndef, e
						}
					}
				}
			}
		}
	}
	if e = m.setCore(); e != nil {
		return lUndef, e
	}
	return lTrue, nil
}
func (m *satSolverMUSMembers) setFromColumn(dst *satMUSLiteralSet, src *satSolverRuntimeUintColumn) error {
	for i := uint32(0); ; i++ {
		n, e := src.size()
		if e != nil {
			return e
		}
		if i >= n {
			return nil
		}
		v, e := src.read(i)
		if e != nil {
			return e
		}
		if e = dst.insert(satLiteral(v)); e != nil {
			return e
		}
	}
}
func (m *satSolverMUSMembers) toVector(set *satMUSLiteralSet, dst *satSolverRuntimeUintColumn) error {
	// Named return-value move is explicit: one source result vector, whose
	// lexical caller owns its destruction. Captured begin/end never rebind.
	n, e := set.size()
	if e != nil {
		return e
	}
	span := satConflictRuntimeSpan(&set.values)
	for i := uint32(0); i < n; i++ {
		l, e := span.literal(i)
		if e != nil {
			return e
		}
		if e = dst.push(uint32(l)); e != nil {
			return e
		}
	}
	return nil
}
func (m *satSolverMUSMembers) mus2() (r lbool, e error) {
	f, e := m.frame(2, true)
	if e != nil {
		return lUndef, e
	}
	defer func() { e = m.drop(f, e) }()
	if e = m.getCore(); e != nil {
		return lUndef, e
	}
	if e = m.setFromColumn(&f.sets[0], &m.core); e != nil {
		return lUndef, e
	}
	r, e = m.qx(&f.sets[0], &f.sets[1], false)
	if e != nil {
		return lUndef, e
	}
	if e = m.solver.conflict.core.shrink(0); e != nil {
		return lUndef, e
	}
	if e = m.toVector(&f.sets[0], &f.vector); e != nil {
		return lUndef, e
	}
	e = satMUSAppend(&m.solver.conflict.core, &f.vector)
	return r, e
}
func (m *satSolverMUSMembers) split(a, b *satMUSLiteralSet) (e error) {
	n, e := a.size()
	if e != nil {
		return e
	}
	half := n / 2
	f, e := m.frame(1, false)
	if e != nil {
		return e
	}
	defer func() { e = m.drop(f, e) }()
	span := satConflictRuntimeSpan(&a.values)
	for i := uint32(0); i < n; i++ {
		v, e := span.literal(i)
		if e != nil {
			return e
		}
		dst := b
		if i < half {
			dst = &f.sets[0]
		}
		if e = dst.insert(v); e != nil {
			return e
		}
	}
	return a.assign(&f.sets[0])
}
func (m *satSolverMUSMembers) unsplit(a, b *satMUSLiteralSet) (e error) {
	f, e := m.frame(2, false)
	if e != nil {
		return e
	}
	defer func() { e = m.drop(f, e) }()
	n, e := a.size()
	if e != nil {
		return e
	}
	span := satConflictRuntimeSpan(&a.values)
	for i := uint32(0); i < n; i++ {
		v, e := span.literal(i)
		if e != nil {
			return e
		}
		yes, e := b.contains(v)
		if e != nil {
			return e
		}
		idx := 0
		if yes {
			idx = 1
		}
		if e = f.sets[idx].insert(v); e != nil {
			return e
		}
	}
	if e = a.assign(&f.sets[0]); e != nil {
		return e
	}
	return b.assign(&f.sets[1])
}
func (m *satSolverMUSMembers) qx(a, b *satMUSLiteralSet, support bool) (r lbool, e error) {
	r = lTrue
	if support {
		r, e = m.qxCheck(a, b)
		if e != nil {
			return r, e
		}
		if r == lFalse {
			return lTrue, nil
		}
		if r != lTrue {
			return r, nil
		}
	}
	n, e := a.size()
	if e != nil {
		return lUndef, e
	}
	if n == 1 {
		return lTrue, nil
	}
	f, e := m.frame(1, false)
	if e != nil {
		return lUndef, e
	}
	defer func() { e = m.drop(f, e) }()
	a2 := &f.sets[0]
	if e = m.split(a, a2); e != nil {
		return lUndef, e
	}
	if e = b.union(a); e != nil {
		return lUndef, e
	}
	n, e = a.size()
	if e != nil {
		return lUndef, e
	}
	if n == 0 {
		second, e := a2.size()
		if e != nil {
			return lUndef, e
		}
		// The first split and union have completed. The next identical empty
		// recursive call has no finite source continuation; preserve that prefix.
		if second == 0 {
			return lUndef, ErrSATMUSEmptyRecursion
		}
	}
	r, e = m.qx(a2, b, n != 0)
	if e != nil {
		return r, e
	}
	if e = m.unsplit(b, a); e != nil {
		return lUndef, e
	}
	if r != lTrue {
		return r, nil
	}
	if e = b.union(a2); e != nil {
		return lUndef, e
	}
	n, e = a2.size()
	if e != nil {
		return lUndef, e
	}
	r, e = m.qx(a, b, n != 0)
	if e != nil {
		return r, e
	}
	if e = a.union(a2); e != nil {
		return lUndef, e
	}
	if e = m.unsplit(b, a2); e != nil {
		return lUndef, e
	}
	return r, nil
}
func (m *satSolverMUSMembers) qxCheck(a, b *satMUSLiteralSet) (r lbool, e error) {
	f, e := m.frame(0, true)
	if e != nil {
		return lUndef, e
	}
	defer func() { e = m.drop(f, e) }()
	if e = m.toVector(b, &f.vector); e != nil {
		return lUndef, e
	}
	old, e := m.mus.size()
	if e != nil {
		return lUndef, e
	}
	if e = satMUSAppend(&m.mus, &f.vector); e != nil {
		return lUndef, e
	}
	f.appendActive = true
	f.appendOld = old
	// Full-expression temporary is destroyed BEFORE the following Check body.
	if e = f.vector.finalize(); e != nil {
		f.cleanupStopped = true
		return lUndef, e
	}
	f.vectorUsed = false
	n, e := m.mus.size()
	if e != nil {
		return lUndef, e
	}
	r, e = m.solver.checkSpan(n, satConflictRuntimeSpan(&m.mus))
	if e != nil {
		return r, e
	}
	switch r {
	case lFalse:
		f.setsUsed = 1
		f.sets[0].bind(m.solver.runtime.memory)
		if e = m.setFromColumn(&f.sets[0], &m.solver.conflict.core); e != nil {
			return lUndef, e
		}
		f.innerCoreCompleted = true
		if e = b.intersect(&f.sets[0]); e != nil {
			return lUndef, e
		}
		if e = a.reset(); e != nil {
			return lUndef, e
		}
		return lFalse, nil
	case lTrue:
		if e = m.updateModel(); e != nil {
			return lUndef, e
		}
	}
	return r, nil
}

func (s *satSolver) setModel(src *satSolverRuntimeUintColumn) error {
	if e := s.model.shrink(0); e != nil {
		return e
	}
	if e := satMUSAppend(&s.model, src); e != nil {
		return e
	}
	n, e := s.model.size()
	if e != nil {
		return e
	}
	s.variables.modelIsCurrent = n != 0
	return nil
}

func (m *satSolverMUSMembers) verifyCore(core *satSolverRuntimeUintColumn) error {
	n, e := core.size()
	if e != nil {
		return e
	}
	_, e = m.solver.checkSpan(n, satConflictRuntimeSpan(core))
	return e
}
func (m *satSolverMUSMembers) isActiveSource() bool                  { return m.isActive }
func (m *satSolverMUSMembers) getModel() *satSolverRuntimeUintColumn { return &m.model }

// Source vector resize publishes the new size before ascending value placement.
// scoped_append uses this full operation, including growth if its borrower was
// mutated; retained old bytes do not substitute for T() initialization.
func satMUSResize(v *satSolverRuntimeUintColumn, n uint32) error {
	old, e := v.size()
	if e != nil {
		return e
	}
	if n <= old {
		return v.shrink(n)
	}
	for {
		capacity, e := v.capacity()
		if e != nil {
			return e
		}
		if capacity >= n {
			break
		}
		if e = v.raw.expandStride(4); e != nil {
			return e
		}
	}
	if e = satAssignmentWriteWord(v.raw.runtime, v.raw.block, 4, n); e != nil {
		return e
	}
	for i := old; i < n; i++ {
		if e = satAssignmentWriteWord(v.raw.runtime, v.raw.block, 8+4*uint64(i), 0); e != nil {
			return e
		}
	}
	return nil
}
