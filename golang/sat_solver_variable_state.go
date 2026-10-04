// Licensed under the MIT license.
// Actual solver member owners and complete sat_solver.cpp:135-160 mk_var body.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
)

// This source-member grouping is not a complete solver constructor. It supplies
// actual columns/queue/trail directly to that owner. Elimination todo is the
// simplifier's real borrowed set. No callback supplies missing solver behavior.
type satSolverVariableState struct {
	self                                                                                     *satSolverVariableState
	budget                                                                                   *arena.Budget
	watches                                                                                  *satSolverWatchLists
	assignment                                                                               satSolverAssignmentColumn
	memory                                                                                   *memory.Runtime
	decision, mark, literalMark, eliminated, external, phase, previousPhase, assignedSinceGC satSolverAssignmentColumn
	justification                                                                            satSolverRuntimeJustificationColumn
	level                                                                                    satSolverRuntimeUintColumn
	activity                                                                                 *satSolverRuntimeActivity
	queue                                                                                    *satVarQueue
	trail                                                                                    *satSolverRuntimeTrail
	trailRef                                                                                 satSolverLiteralListRef
	elimTodo                                                                                 *satSolverUintSet
	ownerBytes                                                                               uint64
	mkVarCount, qhead, scopeLevel, activityInc                                               uint32
	modelIsCurrent, modelKnown, closed                                                       bool
}
type satSolverVariableStats struct {
	OwnerBytes, ColumnBytes, WatchBytes, ActivityBytes, QueueBytes, TrailBytes, TotalBytes uint64
	NumVars, MkVarCount                                                                    uint32
}

func newSATSolverVariableState(todo *satSolverUintSet, b *arena.Budget, c arena.VectorConfig, raw *memory.Runtime) (*satSolverVariableState, error) {
	if e := todo.open(); e != nil {
		return nil, e
	}
	if c.Descriptors.Budget != b {
		return nil, arena.ErrConfig
	}
	if raw == nil {
		return nil, ErrSATOwner
	}
	if _, e := raw.AllocationSize(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(satSolverVariableState{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	s := &satSolverVariableState{budget: b, elimTodo: todo, ownerBytes: n, activityInc: 128}
	s.self = s
	s.memory = raw
	s.assignment.bind(raw)
	s.justification.bind(raw, b)
	for _, col := range []*satSolverAssignmentColumn{&s.decision, &s.mark, &s.literalMark, &s.eliminated, &s.external, &s.phase, &s.previousPhase, &s.assignedSinceGC} {
		col.bind(raw)
	}
	s.level.bind(raw)
	var e error
	s.watches, e = newSATSolverWatchLists(b, c)
	if e == nil {
		s.activity, e = newSATSolverRuntimeActivity(b, raw)
	}
	if e == nil {
		s.queue, e = newSATSolverRuntimeVarQueue(s.activity, b)
	}
	if e == nil {
		s.trail, s.trailRef, e = newSATSolverRuntimeTrail(b, raw)
	}
	if e != nil {
		if cleanup := s.Close(); cleanup != nil {
			return s, cleanup
		}
		return nil, e
	}
	return s, nil
}
func (s *satSolverVariableState) open() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed {
		return arena.ErrClosed
	}
	return nil
}
func (s *satSolverVariableState) numVars() uint32 { n, _ := s.level.size(); return n }
func (s *satSolverVariableState) mkVar(ext, dvar bool) (satBoolVar, error) {
	if e := s.open(); e != nil {
		return 0, e
	}
	s.modelIsCurrent = false
	s.modelKnown = true
	s.mkVarCount++
	n, e := s.level.size()
	if e != nil {
		return 0, e
	}
	v := satBoolVar(n)
	if e := s.watches.PushEmpty(); e != nil {
		return v, e
	}
	if e := s.watches.PushEmpty(); e != nil {
		return v, e
	}
	if e := s.assignment.push(int8(lUndef)); e != nil {
		return v, e
	}
	if e := s.assignment.push(int8(lUndef)); e != nil {
		return v, e
	}
	if e := s.justification.push(satJustification{}); e != nil {
		return v, e
	}
	decision := int8(0)
	if dvar {
		decision = 1
	}
	if e := s.decision.push(decision); e != nil {
		return v, e
	}
	if e := s.eliminated.push(0); e != nil {
		return v, e
	}
	external := int8(0)
	if ext {
		external = 1
	}
	if e := s.external.push(external); e != nil {
		return v, e
	}
	if e := s.activity.Push(0); e != nil {
		return v, e
	}
	if e := s.level.push(^uint32(0)); e != nil {
		return v, e
	}
	if e := s.mark.push(0); e != nil {
		return v, e
	}
	if e := s.literalMark.push(0); e != nil {
		return v, e
	}
	if e := s.literalMark.push(0); e != nil {
		return v, e
	}
	if e := s.phase.push(2); e != nil {
		return v, e
	}
	if e := s.previousPhase.push(2); e != nil {
		return v, e
	}
	if e := s.assignedSinceGC.push(0); e != nil {
		return v, e
	}
	if e := s.queue.MkVar(v); e != nil {
		return v, e
	}
	if e := s.elimTodo.Insert(uint32(v)); e != nil {
		return v, e
	}
	if satStorageSourceDebug {
		eliminated, e := s.eliminated.read(uint32(v))
		if e != nil {
			return v, e
		}
		if eliminated != 0 {
			return v, ErrSATAssertion
		}
	}
	return v, nil
}

// This view is an ephemeral borrow for a clause operation with no variable
// creation during that call. It never copies source assignment/level storage.
func (s *satSolverVariableState) clauseState() satSolverClauseState {
	return satSolverClauseState{variables: s}
}
func (s *satSolverVariableState) valueLiteral(l satLiteral) (lbool, error) {
	if e := s.open(); e != nil {
		return lUndef, e
	}
	v, e := s.assignment.read(l.index())
	return lbool(v), e
}
func (s *satSolverVariableState) valueVar(v satBoolVar) (lbool, error) {
	return s.valueLiteral(satLiteral(uint32(v) << 1))
}
func (s *satSolverVariableState) levelVar(v satBoolVar) (uint32, error) {
	if e := s.open(); e != nil {
		return 0, e
	}
	return s.level.read(uint32(v))
}
func (s *satSolverVariableState) isExternal(v satBoolVar) (bool, error) {
	if e := s.open(); e != nil {
		return false, e
	}
	n, e := s.external.read(uint32(v))
	return n != 0, e
}
func (s *satSolverVariableState) wasEliminated(v satBoolVar) (bool, error) {
	if e := s.open(); e != nil {
		return false, e
	}
	n, e := s.eliminated.read(uint32(v))
	return n != 0, e
}
func (s *satSolverVariableState) trailPush(l satLiteral) error {
	if e := s.open(); e != nil {
		return e
	}
	return s.trail.Push(s.trailRef, l)
}
func (s *satSolverVariableState) trailRead(i uint32) (satLiteral, error) {
	if e := s.open(); e != nil {
		return 0, e
	}
	return s.trail.Read(s.trailRef, i)
}
func (s *satSolverVariableState) Stats() satSolverVariableStats {
	if s == nil || s.self != s || s.closed {
		return satSolverVariableStats{}
	}
	out := satSolverVariableStats{OwnerBytes: s.ownerBytes, NumVars: s.numVars(), MkVarCount: s.mkVarCount}
	out.ColumnBytes += s.justification.provenanceBytes
	if s.watches != nil {
		out.WatchBytes = s.watches.Stats().TotalBytes
	}
	if s.activity != nil {
		v := s.activity.Stats()
		out.ActivityBytes = v.OwnerBytes + v.BackingBytes + v.BindingBytes
	}
	if s.queue != nil {
		v := s.queue.Stats()
		out.QueueBytes = v.OwnerBytes + v.ValuesBytes + v.IndicesBytes
	}
	if s.trail != nil {
		out.TrailBytes = s.trail.ownerBytes
	}
	out.TotalBytes = out.OwnerBytes + out.ColumnBytes + out.WatchBytes + out.ActivityBytes + out.QueueBytes + out.TrailBytes
	return out
}

// Independent graph walk: registry liveness alone is not evidence of ownership.
// Every active watch descriptor is named exactly once in the source outer vector;
// the only trail descriptor is the actual trail member token.
func (s *satSolverVariableState) checkPhysicalOwners() error {
	if e := s.open(); e != nil {
		return e
	}
	for i := range s.watches.store.records {
		r := &s.watches.store.records[i]
		if !r.live {
			continue
		}
		count := 0
		for _, owner := range s.watches.lists {
			if owner.owner == s.watches.store.origin && owner.slot == uint32(i) && owner.generation == r.generation {
				count++
			}
		}
		if count != 1 {
			return ErrSATOwner
		}
	}
	for _, r := range s.watches.lists {
		if !s.watches.store.IsLive(r) {
			return arena.ErrInvalidSlot
		}
	}
	if !s.trail.IsLive(s.trailRef) {
		return arena.ErrInvalidSlot
	}
	return nil
}
func (s *satSolverVariableState) Close() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed {
		return nil
	}
	// Selected member destruction follows reverse declaration order. No clauses
	// or borrowed elimination todo are owned here, and no AST/SAT callbacks run.
	if s.trail != nil {
		if e := s.trail.Close(); e != nil {
			return e
		}
		s.trail = nil
	}
	if s.queue != nil {
		if e := s.queue.Close(); e != nil {
			return e
		}
		s.queue = nil
	}
	if e := s.assignedSinceGC.finalize(); e != nil {
		return e
	}
	if e := s.previousPhase.finalize(); e != nil {
		return e
	}
	if e := s.phase.finalize(); e != nil {
		return e
	}
	if s.activity != nil {
		if e := s.activity.Close(); e != nil {
			return e
		}
		s.activity = nil
	}
	if e := s.level.finalize(); e != nil {
		return e
	}
	if e := s.external.finalize(); e != nil {
		return e
	}
	if e := s.eliminated.finalize(); e != nil {
		return e
	}
	if e := s.literalMark.finalize(); e != nil {
		return e
	}
	if e := s.mark.finalize(); e != nil {
		return e
	}
	if e := s.decision.finalize(); e != nil {
		return e
	}
	if e := s.justification.finalize(); e != nil {
		return e
	}
	if e := s.assignment.finalize(); e != nil {
		return e
	}
	if s.watches != nil {
		if e := s.watches.Close(); e != nil {
			return e
		}
		s.watches = nil
	}
	s.budget.Release(s.ownerBytes)
	s.ownerBytes = 0
	s.closed = true
	return nil
}
