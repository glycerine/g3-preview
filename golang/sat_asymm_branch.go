// Source: src/sat/sat_asymm_branch.cpp; src/sat/sat_asymm_branch.h; src/sat/sat_asymm_branch_params.pyg. Licensed under the MIT license.
// Source author: Leonardo de Moura, 2011.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/utiltime"
	"reflect"
)

type satAsymmBranch struct {
	self                                                                                  *satAsymmBranch
	lifecycle                                                                             *satSolverClauseLifecycle
	ownerBytes                                                                            uint64
	enabled                                                                               bool
	rounds, limit, elimLiterals                                                           uint32
	savedPhase                                                                            satSolverAssignmentColumn
	report                                                                                utiltime.Stopwatch
	debugPropagate, debugMissed                                                           bool
	phaseRestoreStopped, savedConstructed                                                 bool
	reportActive, closing, closed, debugInvariant, running, interrupted                   bool
	cursor, out, end, drainPrefix, drainSuffix                                            uint32
	detachedClause, discardedClause                                                       satStoredClauseRef
	detachActive                                                                          bool
	bodyCause, cleanupCause                                                               error
	scratch                                                                               []satStoredClauseRef
	scratchBytes                                                                          uint64
	sorting, sortSrcScratch                                                               bool
	sortLeft, sortMid, sortRight, sortEnd, sortOut, sortN, sortDrainStage, sortDrainIndex uint32
}

func newSATAsymmBranch(l *satSolverClauseLifecycle, p *ParamsRef) (*satAsymmBranch, error) {
	if e := l.open(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(satAsymmBranch{}).Size())
	if e := l.budget.Reserve(n); e != nil {
		return nil, e
	}
	a := &satAsymmBranch{lifecycle: l, ownerBytes: n}
	a.self = a
	a.savedPhase.bind(l.propagation.memory)
	l.propagation.asymmCounter = 0
	if e := a.Update(p); e != nil {
		l.budget.Release(n)
		return nil, e
	}
	a.resetStatistics()
	return a, nil
}
func (a *satAsymmBranch) open() error {
	if a == nil || a.self != a {
		return ErrSATOwner
	}
	if a.closed || a.closing || a.reportActive || a.savedConstructed || a.interrupted || a.sorting || a.detachActive || a.phaseRestoreStopped || a.savedPhase.block != (memory.Block{}) || a.savedPhase.pending != (memory.Block{}) {
		return arena.ErrClosed
	}
	return a.lifecycle.open()
}
func (a *satAsymmBranch) Update(p *ParamsRef) (err error) {
	if e := a.open(); e != nil {
		return e
	}
	cfg := a.lifecycle.propagation.config
	g, e := cfg.globals.GetModuleName("sat")
	if e != nil {
		return e
	}
	defer func() {
		if e := cfg.context.Del(&g); err == nil {
			err = e
		}
	}()
	key := "asymm_branch"
	b, e := cfg.context.GetBoolFallbackName(p, &key, &g, true)
	if e != nil {
		return e
	}
	a.enabled = b
	key = "asymm_branch.rounds"
	a.rounds, e = cfg.context.GetUIntFallbackName(p, &key, &g, 32)
	if e != nil {
		return e
	}
	key = "asymm_branch.limit"
	a.limit, e = cfg.context.GetUIntFallbackName(p, &key, &g, 100000000)
	if e != nil {
		return e
	}
	if a.limit > uint32(1<<31-1) {
		a.limit = 1<<31 - 1
	}
	return nil
}
func (a *satAsymmBranch) resetStatistics() { a.elimLiterals = 0 }
func (a *satAsymmBranch) collectStatistics(s *Statistics, k StatisticsKey) error {
	if e := a.open(); e != nil {
		return e
	}
	return s.UpdateUint(k, a.elimLiterals)
}
func collectAsymmBranchParamDescrs(d *ParamDescrs) error {
	for _, x := range [3]struct {
		name             string
		kind             ParamKind
		description, def string
	}{{"asymm_branch", ParamBool, "asymmetric branching", "true"}, {"asymm_branch.rounds", ParamUInt, "maximum number of rounds of asymmetric branching", "32"}, {"asymm_branch.limit", ParamUInt, "approx. maximum number of literals visited during asymmetric branching", "100000000"}} {
		if e := satConfigInsertDescriptor(d, x.name, x.kind, x.description, x.def); e != nil {
			return e
		}
	}
	return nil
}
func (a *satAsymmBranch) dec(n uint32) {
	p := a.lifecycle.propagation
	p.asymmCounter = int32(uint32(p.asymmCounter) - n)
}
func (a *satAsymmBranch) negateCounter() error {
	p := a.lifecycle.propagation
	if p.asymmCounter == -1<<31 {
		return arena.ErrBounds
	}
	p.asymmCounter = -p.asymmCounter
	return nil
}
func (a *satAsymmBranch) invariant() error {
	if satStorageSourceDebug && a.debugInvariant {
		return a.lifecycle.checkInvariant()
	}
	return nil
}
func (a *satAsymmBranch) trailInvariant() error {
	if satStorageSourceDebug {
		p := a.lifecycle.propagation
		n, e := p.variables.trail.Size(p.variables.trailRef)
		if e != nil {
			return e
		}
		if p.variables.qhead != n {
			return ErrSATAssertion
		}
	}
	return nil
}
func (a *satAsymmBranch) run(force bool) (err error) {
	if e := a.open(); e != nil {
		return e
	}
	p := a.lifecycle.propagation
	if !a.enabled {
		return nil
	}
	if _, e := a.lifecycle.propagate(false, a.debugPropagate, a.debugMissed); e != nil {
		return e
	}
	if p.inconsistent {
		return nil
	}
	if !force && p.asymmCounter > 0 {
		return nil
	}
	if e := a.invariant(); e != nil {
		return e
	}
	if e := a.report.Start(); e != nil {
		return e
	}
	a.reportActive = true
	a.bodyCause = nil
	a.cleanupCause = nil
	defer func() { err = a.finishRun(err) }()
	a.savedPhase = satSolverAssignmentColumn{}
	a.savedPhase.bind(p.memory)
	if e := a.copyPhase(&a.savedPhase, &p.variables.phase); e != nil {
		return e
	}
	a.savedConstructed = true
	p.asymmCounter = 0
	limit := -int32(a.limit)
	if e := a.sortOriginal(); e != nil {
		return e
	}
	a.dec(uint32(len(a.lifecycle.original)))
	if e := a.trailInvariant(); e != nil {
		return e
	}
	if e := a.beginCompaction(); e != nil {
		return e
	}
	defer func() { a.running = false }()
	for i := uint32(0); i < a.end; i++ {
		if p.inconsistent {
			a.appendRemainder(i)
			break
		}
		if e := a.trailInvariant(); e != nil {
			return e
		}
		if p.asymmCounter < limit || p.inconsistent {
			a.lifecycle.original[a.out] = a.lifecycle.original[i]
			a.out++
			a.cursor = i + 1
			continue
		}
		e := p.checkpoint()
		if e == nil {
			r := a.lifecycle.original[i]
			var n uint32
			n, e = p.clauses.Size(r)
			if e == nil {
				a.dec(n)
				var keep bool
				keep, e = a.process(r)
				if e == nil {
					if keep {
						a.lifecycle.original[a.out] = r
						a.out++
					}
					a.cursor = i + 1
				}
			}
		}
		if e != nil {
			var solver SATSolverException
			if errors.As(e, &solver) && !a.detachActive {
				a.appendRemainder(i)
				a.finishCompaction()
				if x := a.negateCounter(); x != nil {
					return errors.Join(e, x)
				}
			}
			return e
		}
	}
	a.finishCompaction()
	if e := a.negateCounter(); e != nil {
		return e
	}
	// vector::operator= destroys old storage before source copy_core publication.
	a.phaseRestoreStopped = true
	if e := p.variables.phase.release(&p.variables.phase.block); e != nil {
		return e
	}
	if e := a.copyPhase(&p.variables.phase, &a.savedPhase); e != nil {
		return e
	}
	a.phaseRestoreStopped = false
	return a.invariant()
}
func (a *satAsymmBranch) beginCompaction() error {
	l := a.lifecycle
	if l.asymmContinuation != nil {
		return ErrSATOwner
	}
	a.cursor = 0
	a.out = 0
	a.end = uint32(len(l.original))
	a.drainPrefix = 0
	a.drainSuffix = 0
	a.interrupted = true
	a.running = true
	l.asymmContinuation = a
	return nil
}
func (a *satAsymmBranch) appendRemainder(i uint32) {
	for ; i < a.end; i++ {
		a.lifecycle.original[a.out] = a.lifecycle.original[i]
		a.out++
		a.cursor = i + 1
	}
}
func (a *satAsymmBranch) finishCompaction() {
	a.lifecycle.original = a.lifecycle.original[:a.out]
	a.interrupted = false
	a.lifecycle.asymmContinuation = nil
}
func (a *satAsymmBranch) copyPhase(dst, src *satSolverAssignmentColumn) error {
	if src.block == (memory.Block{}) {
		dst.block = memory.Block{}
		return nil
	}
	n, e := src.size()
	if e != nil {
		return e
	}
	c, e := src.capacity()
	if e != nil {
		return e
	}
	h, e := src.runtime.Allocate(uint64(c) + 8)
	if e != nil {
		dst.pending = h
		return e
	}
	dst.pending = h
	if e = satAssignmentWriteWord(src.runtime, h, 0, c); e != nil {
		return e
	}
	if e = satAssignmentWriteWord(src.runtime, h, 4, n); e != nil {
		return e
	}
	dst.block = h
	dst.pending = memory.Block{}
	for i := uint32(0); i < n; i++ {
		v, e := src.read(i)
		if e != nil {
			return e
		}
		if e = src.runtime.Write(h, 8+uint64(i), byte(v)); e != nil {
			return e
		}
	}
	return nil
}

// Completed locals unwind in reverse construction order. A failed copy ctor
// has no source vector destructor: only the older report unwinds here; its
// actual partial allocation remains for the separate owning-abort Close path.
func (a *satAsymmBranch) finishRun(cause error) error {
	a.bodyCause = cause
	if a.savedConstructed {
		if e := a.savedPhase.finalize(); e != nil {
			a.cleanupCause = e
			if cause == nil {
				return e
			}
			return errors.Join(cause, e)
		}
		a.savedConstructed = false
	}
	if a.reportActive {
		if e := a.report.Stop(); e != nil {
			if cause == nil {
				return e
			}
			return errors.Join(cause, e)
		}
		a.reportActive = false
	}
	return cause
}
func (a *satAsymmBranch) Close() error {
	if a == nil || a.self != a {
		return ErrSATOwner
	}
	if a.closed {
		return nil
	}
	a.closing = true
	if e := a.finishRun(nil); e != nil {
		return e
	}
	if e := a.cleanupInterrupted(); e != nil {
		return e
	}
	if a.discardedClause != (satStoredClauseRef{}) {
		if e := a.lifecycle.propagation.clauses.Delete(a.discardedClause); e != nil {
			return e
		}
		a.discardedClause = satStoredClauseRef{}
	}

	// Only owning cleanup destroys a failed constructor's partial allocation,
	// after the older source report has completed its exception unwind.
	if e := a.savedPhase.finalize(); e != nil {
		return e
	}

	a.lifecycle.budget.Release(a.ownerBytes)
	a.ownerBytes = 0
	a.closed = true
	return nil
}
