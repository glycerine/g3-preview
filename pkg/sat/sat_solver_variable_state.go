// Licensed under the MIT license.
// Actual solver member owners and complete sat_solver.cpp:135-160 mk_var body.
package sat

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// This source-member grouping is not a complete solver constructor. It supplies
// actual columns/queue/trail directly to that owner. Elimination todo is the
// simplifier's real borrowed set. No callback supplies missing solver behavior.
type satSolverVariableState struct {
	self                                                                                     *satSolverVariableState
	budget                                                                                   *arena.Budget
	watches                                                                                  *satSolverWatchLists
	assignment                                                                               RuntimeVectorColumn
	memory                                                                                   *util.Runtime
	decision, mark, literalMark, eliminated, external, phase, previousPhase, assignedSinceGC RuntimeVectorColumn
	justification                                                                            satSolverRuntimeJustificationColumn
	level                                                                                    RuntimeUintColumn
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

// Cold preparation provides actual source member borrows while later members
// are still unconstructed. Full solver ordinary operations remain guarded by
// its constructor completion; the standalone adapter below retains its order.
func newSATColdVariableState(todo *satSolverUintSet, b *arena.Budget, c arena.VectorConfig, raw *util.Runtime) (*satSolverVariableState, error) {
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
	s.assignment.Bind(raw)
	s.justification.bind(raw, b)
	for _, col := range []*RuntimeVectorColumn{&s.decision, &s.mark, &s.literalMark, &s.eliminated, &s.external, &s.phase, &s.previousPhase, &s.assignedSinceGC} {
		col.Bind(raw)
	}
	s.level.Bind(raw)
	return s, nil
}

func newSATSolverVariableState(todo *satSolverUintSet, b *arena.Budget, c arena.VectorConfig, raw *util.Runtime) (*satSolverVariableState, error) {
	s, e := newSATColdVariableState(todo, b, c, raw)
	if e != nil {
		return nil, e
	}
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
func (s *satSolverVariableState) constructionBorrow() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.closed {
		return arena.ErrClosed
	}
	return nil
}
func (s *satSolverVariableState) open() error {
	if e := s.constructionBorrow(); e != nil {
		return e
	}
	if s.watches == nil || s.activity == nil || s.queue == nil || s.trail == nil {
		return arena.ErrClosed
	}
	return nil
}

func (s *satSolverVariableState) NumVars() uint32 { n, _ := s.level.Size(); return n }
func (s *satSolverVariableState) mkVar(ext, dvar bool) (BoolVar, error) {
	if e := s.open(); e != nil {
		return 0, e
	}
	s.modelIsCurrent = false
	s.modelKnown = true
	s.mkVarCount++
	n, e := s.level.Size()
	if e != nil {
		return 0, e
	}
	v := BoolVar(n)
	if e := s.watches.PushEmpty(); e != nil {
		return v, e
	}
	if e := s.watches.PushEmpty(); e != nil {
		return v, e
	}
	if e := s.assignment.Push(int8(util.LUndef)); e != nil {
		return v, e
	}
	if e := s.assignment.Push(int8(util.LUndef)); e != nil {
		return v, e
	}
	if e := s.justification.push(satJustification{}); e != nil {
		return v, e
	}
	decision := int8(0)
	if dvar {
		decision = 1
	}
	if e := s.decision.Push(decision); e != nil {
		return v, e
	}
	if e := s.eliminated.Push(0); e != nil {
		return v, e
	}
	external := int8(0)
	if ext {
		external = 1
	}
	if e := s.external.Push(external); e != nil {
		return v, e
	}
	if e := s.activity.Push(0); e != nil {
		return v, e
	}
	if e := s.level.Push(^uint32(0)); e != nil {
		return v, e
	}
	if e := s.mark.Push(0); e != nil {
		return v, e
	}
	if e := s.literalMark.Push(0); e != nil {
		return v, e
	}
	if e := s.literalMark.Push(0); e != nil {
		return v, e
	}
	if e := s.phase.Push(2); e != nil {
		return v, e
	}
	if e := s.previousPhase.Push(2); e != nil {
		return v, e
	}
	if e := s.assignedSinceGC.Push(0); e != nil {
		return v, e
	}
	if e := s.queue.MkVar(v); e != nil {
		return v, e
	}
	if e := s.elimTodo.Insert(uint32(v)); e != nil {
		return v, e
	}
	if satStorageSourceDebug {
		eliminated, e := s.eliminated.Read(uint32(v))
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
func (s *satSolverVariableState) valueLiteral(l Literal) (util.LBool, error) {
	if e := s.open(); e != nil {
		return util.LUndef, e
	}
	v, e := s.assignment.Read(l.index())
	return util.LBool(v), e
}
func (s *satSolverVariableState) valueVar(v BoolVar) (util.LBool, error) {
	return s.valueLiteral(Literal(uint32(v) << 1))
}
func (s *satSolverVariableState) levelVar(v BoolVar) (uint32, error) {
	if e := s.open(); e != nil {
		return 0, e
	}
	return s.level.Read(uint32(v))
}
func (s *satSolverVariableState) IsExternal(v BoolVar) (bool, error) {
	if e := s.open(); e != nil {
		return false, e
	}
	n, e := s.external.Read(uint32(v))
	return n != 0, e
}
func (s *satSolverVariableState) wasEliminated(v BoolVar) (bool, error) {
	if e := s.open(); e != nil {
		return false, e
	}
	n, e := s.eliminated.Read(uint32(v))
	return n != 0, e
}
func (s *satSolverVariableState) trailPush(l Literal) error {
	if e := s.open(); e != nil {
		return e
	}
	return s.trail.Push(s.trailRef, l)
}
func (s *satSolverVariableState) trailRead(i uint32) (Literal, error) {
	if e := s.open(); e != nil {
		return 0, e
	}
	return s.trail.Read(s.trailRef, i)
}
func (s *satSolverVariableState) Stats() satSolverVariableStats {
	if s == nil || s.self != s || s.closed {
		return satSolverVariableStats{}
	}
	out := satSolverVariableStats{OwnerBytes: s.ownerBytes, NumVars: s.NumVars(), MkVarCount: s.mkVarCount}
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
	if e := s.finalizeTrail(); e != nil {
		return e
	}
	if e := s.finalizeQueue(); e != nil {
		return e
	}
	if e := s.finalizeColumnsWatches(); e != nil {
		return e
	}
	s.budget.Release(s.ownerBytes)
	s.ownerBytes = 0
	s.closed = true
	return nil
}

// These concrete stages expose the actual reverse member actions for the full
// facade. Standalone Close keeps its previous order. Retry is an explicit Go
// owning acknowledgment, not a C++ destructor exception continuation.
func (s *satSolverVariableState) finalizeTrail() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.trail != nil {
		if e := s.trail.Close(); e != nil {
			return e
		}
		s.trail = nil
	}
	return nil
}
func (s *satSolverVariableState) finalizeQueue() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if s.queue != nil {
		if e := s.queue.Close(); e != nil {
			return e
		}
		s.queue = nil
	}
	return nil
}
func (s *satSolverVariableState) finalizeColumnsWatches() error {
	if s == nil || s.self != s {
		return ErrSATOwner
	}
	if e := s.assignedSinceGC.Finalize(); e != nil {
		return e
	}
	if e := s.previousPhase.Finalize(); e != nil {
		return e
	}
	if e := s.phase.Finalize(); e != nil {
		return e
	}
	if s.activity != nil {
		if e := s.activity.Close(); e != nil {
			return e
		}
		s.activity = nil
	}
	if e := s.level.Finalize(); e != nil {
		return e
	}
	if e := s.external.Finalize(); e != nil {
		return e
	}
	if e := s.eliminated.Finalize(); e != nil {
		return e
	}
	if e := s.literalMark.Finalize(); e != nil {
		return e
	}
	if e := s.mark.Finalize(); e != nil {
		return e
	}
	if e := s.decision.Finalize(); e != nil {
		return e
	}
	if e := s.justification.finalize(); e != nil {
		return e
	}
	if e := s.assignment.Finalize(); e != nil {
		return e
	}
	if s.watches != nil {
		if e := s.watches.Close(); e != nil {
			return e
		}
		s.watches = nil
	}
	return nil
}
