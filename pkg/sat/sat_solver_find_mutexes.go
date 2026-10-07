// Source: src/sat/sat_solver.cpp:2818-2838,3119-3150; src/util/max_cliques.h.
package sat

import (
	"reflect"

	"github.com/glycerine/g3/pkg/util"
)

type satMutexFrame struct {
	next            *satMutexFrame
	ownerBytes      uint64
	graph           *satMaxCliques
	seen            satMutexPairSet
	seenComplete    bool
	seenUsed        bool
	results         *satSolverLiteralRows
	ps, row         RuntimeUintColumn
	psUsed, rowUsed bool
	cursor          uint32
}

func (s *Solver) mutexFrame() (*satMutexFrame, error) {
	bytes := uint64(reflect.TypeOf(satMutexFrame{}).Size())
	if e := s.budget.Reserve(bytes); e != nil {
		return nil, e
	}
	f := &satMutexFrame{next: s.mutexFrames, ownerBytes: bytes}
	s.mutexFrames = f
	return f, nil
}
func (s *Solver) mutexDrop(f *satMutexFrame, cause error) error {
	if s.mutexFrames != f {
		if cause != nil {
			return cause
		}
		return ErrSATOwner
	}
	for f.cursor < 6 {
		var e error
		switch f.cursor {
		case 0:
			if f.rowUsed {
				e = f.row.Finalize()
			}
		case 1:
			if f.psUsed {
				e = f.ps.Finalize()
			}
		case 2:
			if f.results != nil {
				e = f.results.CloseChecked()
			}
		case 3:
			if f.seenComplete {
				e = f.seen.close()
			}
		case 5:
			if f.seenUsed && !f.seenComplete {
				e = f.seen.close()
			}
		case 4:
			if f.graph != nil {
				e = f.graph.Close()
			}
		}
		if e != nil {
			if cause != nil {
				return cause
			}
			return e
		}
		f.cursor++
	}
	s.mutexFrames = f.next
	s.budget.Release(f.ownerBytes)
	f.ownerBytes = 0
	return cause
}
func (s *Solver) cleanupMutexes() error {
	for s.mutexFrames != nil {
		if e := s.mutexDrop(s.mutexFrames, nil); e != nil {
			return e
		}
	}
	return nil
}
func (s *Solver) collectMutexBins(learned bool) error {
	for idx := 0; idx < len(s.Variables.watches.lists); idx++ {
		l := Literal(uint32(idx)).Negated()
		it, e := s.Variables.watches.store.Iter(s.Variables.watches.lists[idx])
		if e != nil {
			return e
		}
		for {
			w, yes, e := it.Next()
			if e != nil {
				return e
			}
			if !yes {
				break
			}
			if !w.Binary() {
				continue
			}
			if !learned {
				yes, e := w.Learned()
				if e != nil {
					return e
				}
				if yes {
					continue
				}
			}
			l2, e := w.Literal()
			if e != nil {
				return e
			}
			if l.index() > l2.index() {
				continue
			}
			if e = s.userBinClauses.Push(newSATSolverBinPair(l, l2)); e != nil {
				return e
			}
		}
	}
	return nil
}
func (s *Solver) findMutexes(lits []Literal, out *satSolverLiteralRows) (result util.LBool, err error) {
	if err = s.consequenceEntry(out); err != nil {
		return util.LUndef, err
	}
	s.mutexRunning = true
	defer func() { s.mutexRunning = false }()
	f, e := s.mutexFrame()
	if e != nil {
		return util.LUndef, e
	}
	defer func() { err = s.mutexDrop(f, err) }()
	f.graph, err = newSATMaxCliques(s.runtime.memory, s.budget)
	if err != nil {
		return util.LUndef, err
	}
	if err = s.userBinClauses.Reset(); err != nil {
		return util.LUndef, err
	}
	// Existing concrete empty/member graph reset. Actual owning rows destroy in
	// ascending order; retained outer cold backing is not a source malloc ticket.
	for i := range s.binaryClauseGraph {
		if err = s.binaryClauseGraph[i].Finalize(); err != nil {
			return util.LUndef, err
		}
	}
	s.binaryClauseGraph = s.binaryClauseGraph[:0]
	if err = s.collectMutexBins(true); err != nil {
		return util.LUndef, err
	}
	if err = s.collectMutexBins(false); err != nil {
		return util.LUndef, err
	}
	f.seenUsed = true
	if err = f.seen.init(s.runtime.memory); err != nil {
		return util.LUndef, err
	}
	f.seenComplete = true
	for i := uint32(0); ; i++ {
		n, e := s.userBinClauses.Size()
		if e != nil {
			return util.LUndef, e
		}
		if i >= n {
			break
		}
		bc, e := s.userBinClauses.Get(i)
		if e != nil {
			return util.LUndef, e
		}
		a, b := bc.Literal1().index(), bc.Literal2().index()
		yes, e := f.seen.contains(a, b)
		if e != nil {
			return util.LUndef, e
		}
		if !yes {
			if err = f.seen.insert(a, b); err != nil {
				return util.LUndef, err
			}
			if err = f.graph.addEdge(a, b); err != nil {
				return util.LUndef, err
			}
		}
	}
	f.results, err = newSATSolverLiteralRows(s.runtime.memory, s.budget)
	if err != nil {
		return util.LUndef, err
	}
	f.psUsed = true
	f.ps.Bind(s.runtime.memory)
	for _, l := range lits {
		if err = f.ps.Push(l.index()); err != nil {
			return util.LUndef, err
		}
	}
	if err = f.graph.cliques(&f.ps, f.results); err != nil {
		return util.LUndef, err
	}
	for i := uint32(0); ; i++ {
		n, e := f.results.rowCount()
		if e != nil {
			return util.LUndef, e
		}
		if i >= n {
			break
		}
		f.row = RuntimeUintColumn{}
		f.row.Bind(s.runtime.memory)
		f.rowUsed = true
		src, e := f.results.unsignedRow(i)
		if e != nil {
			return util.LUndef, e
		}
		for j := uint32(0); ; j++ {
			n, e := src.Size()
			if e != nil {
				return util.LUndef, e
			}
			if j >= n {
				break
			}
			v, e := src.Read(j)
			if e != nil {
				return util.LUndef, e
			}
			if err = f.row.Push(v); err != nil {
				return util.LUndef, err
			}
		}
		if err = out.appendColumn(&f.row); err != nil {
			return util.LUndef, err
		}
		if err = f.row.Finalize(); err != nil {
			return util.LUndef, err
		}
		f.rowUsed = false
	}
	return util.LTrue, nil
}
