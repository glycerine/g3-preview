// Source: src/math/euclid/euclidean_solver.cpp:44-65,506-565.
package euclid

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/heap"
	"github.com/glycerine/g3/pkg/util/numeral"
)

const nullEquationIndex = ^uint32(0)

// position finds a variable in the source's sorted equation ID slice.
func position(xs []uint32, x uint32) uint32 {
	if len(xs) == 0 {
		return nullEquationIndex
	}
	low, high := 0, len(xs)-1
	for low <= high {
		mid := low + (high-low)/2
		switch {
		case x > xs[mid]:
			low = mid + 1
		case x < xs[mid]:
			high = mid - 1
		default:
			return uint32(mid)
		}
	}
	return nullEquationIndex
}

// solverSetup owns the currently reached Euclidean variable, equation,
// substitution, normalization and unit-pivot state. It borrows its rational
// manager; non-unit decomposition and the public solver remain pending.
type solverSetup struct {
	self               *solverSetup
	manager            *numeral.RationalManager
	decomposeBuffer    *numeral.ScopedIntVector
	asBuffer           *numeral.ScopedIntVector
	bsBuffer           *numeral.ScopedRationalVector
	tmpAs              *numeral.ScopedIntVector
	tmpBs              *numeral.ScopedRationalVector
	tmpXs              []uint32
	queue              *heap.Int
	parameter          []bool
	solved             []uint32
	occurrences        [][]uint32
	nextJustification  uint32
	inconsistent       uint32
	nextX              uint32
	nextEq             uint32
	nextA              numeral.Int
	nextPosA           bool
	nextADone          bool
	equations          []*equation
	solution           []*equation
	equationCloseIndex int
	solutionCloseIndex int
	header             uint64
	closed             bool
	stop               error
}

func newSolverSetup(m *numeral.RationalManager) (*solverSetup, error) {
	return newSolverSetupWithDebug(m, nil)
}

func newSolverSetupWithDebug(m *numeral.RationalManager, debug *diagnostic.DebugContext) (*solverSetup, error) {
	if m == nil || m.BudgetField == nil || m.Integers() == nil {
		return nil, arena.ErrInvalidSlot
	}
	header := uint64(reflect.TypeOf(solverSetup{}).Size())
	if e := m.BudgetField.Reserve(header); e != nil {
		return nil, e
	}
	decompose, e := numeral.NewScopedIntVector(m.Integers())
	if e != nil {
		m.BudgetField.Release(header)
		return nil, e
	}
	as, e := numeral.NewScopedIntVector(m.Integers())
	if e != nil {
		_ = decompose.Close()
		m.BudgetField.Release(header)
		return nil, e
	}
	bs, e := numeral.NewScopedRationalVector(m)
	if e != nil {
		_ = as.Close()
		_ = decompose.Close()
		m.BudgetField.Release(header)
		return nil, e
	}
	tmpAs, e := numeral.NewScopedIntVector(m.Integers())
	if e != nil {
		_ = bs.Close()
		_ = as.Close()
		_ = decompose.Close()
		m.BudgetField.Release(header)
		return nil, e
	}
	tmpBs, e := numeral.NewScopedRationalVector(m)
	if e != nil {
		_ = tmpAs.Close()
		_ = bs.Close()
		_ = as.Close()
		_ = decompose.Close()
		m.BudgetField.Release(header)
		return nil, e
	}
	s := &solverSetup{manager: m, decomposeBuffer: decompose, asBuffer: as, bsBuffer: bs, tmpAs: tmpAs, tmpBs: tmpBs,
		inconsistent: nullEquationIndex, nextX: nullEquationIndex, nextEq: nullEquationIndex, header: header}
	s.self = s
	s.queue, e = heap.New(16, heap.Config{Budget: m.BudgetField, Debug: debug, Less: func(a, b int32) bool {
		return s.solved[a] < s.solved[b]
	}})
	if e != nil {
		_ = tmpBs.Close()
		_ = tmpAs.Close()
		_ = bs.Close()
		_ = as.Close()
		_ = decompose.Close()
		m.BudgetField.Release(header)
		return nil, e
	}
	return s, nil
}

func (s *solverSetup) open() error {
	if s == nil || s.self != s || s.manager == nil {
		return arena.ErrInvalidSlot
	}
	if s.closed {
		return arena.ErrClosed
	}
	if s.stop != nil {
		return s.stop
	}
	return nil
}

// mkVar preserves the source publication order: three metadata vectors,
// then a copied zero numeral in the ID-indexed coefficient buffer.
func (s *solverSetup) mkVar(parameter bool) (uint32, error) {
	if e := s.open(); e != nil {
		return 0, e
	}
	if len(s.solved) >= 1<<31-1 {
		return 0, arena.ErrBounds
	}
	x := uint32(len(s.solved))
	s.parameter = append(s.parameter, parameter)
	s.solved = append(s.solved, nullEquationIndex)
	s.occurrences = append(s.occurrences, nil)
	zero := numeral.NewInt(0)
	if e := s.asBuffer.Push(&zero); e != nil {
		s.stop = e
		return 0, e
	}
	if e := s.queue.Reserve(int32(x + 1)); e != nil {
		s.stop = e
		return 0, e
	}
	return x, nil
}

// scheduleVarSubst inserts only solved IDs, preserving source input order.
// Equal-priority behavior remains the heap's concrete source ordering.
func (s *solverSetup) scheduleVarSubst(xs []uint32) error {
	if e := s.open(); e != nil {
		return e
	}
	for _, x := range xs {
		if uint64(x) >= uint64(len(s.solved)) {
			return arena.ErrBounds
		}
		if s.solved[x] != nullEquationIndex {
			if e := s.queue.Insert(int32(x)); e != nil {
				s.stop = e
				return e
			}
		}
	}
	return nil
}

// mkJustification appends the zero buffer slot before advancing the ID.
func (s *solverSetup) mkJustification() (uint32, error) {
	if e := s.open(); e != nil {
		return 0, e
	}
	if s.nextJustification == ^uint32(0) {
		return 0, arena.ErrBounds
	}
	j := s.nextJustification
	zero := numeral.NewRational()
	if e := s.bsBuffer.Push(&zero); e != nil {
		s.stop = e
		return 0, e
	}
	s.nextJustification++
	return j, nil
}

// addEquation covers mk_eq and the pre-normalization publication to the
// source equation list. The solver owner retains the returned equation.
func (s *solverSetup) addEquation(as []numeral.Int, xs []uint32, c *numeral.Int,
	bs []numeral.Rational, js []uint32, sortJustifications bool) (*equation, error) {
	if e := s.open(); e != nil {
		return nil, e
	}
	eq, e := makeEquation(s.manager, as, xs, c, bs, js, s.asBuffer, s.bsBuffer, sortJustifications)
	if e != nil {
		s.stop = e
		return nil, e
	}
	s.equations = append(s.equations, eq)
	return eq, nil
}

// Close destroys the selected coefficient, equation lists and then member
// buffers in source destructor and reverse declaration order.
// The borrowed rational manager remains with its caller.
func (s *solverSetup) Close() error {
	if s == nil || s.self != s {
		return arena.ErrInvalidSlot
	}
	if s.closed {
		return nil
	}
	if !s.nextADone {
		if e := s.manager.Integers().Del(&s.nextA); e != nil {
			return e
		}
		s.nextADone = true
	}
	for s.equationCloseIndex < len(s.equations) {
		if s.equations[s.equationCloseIndex] == nil {
			s.equationCloseIndex++
			continue
		}
		if e := s.equations[s.equationCloseIndex].Close(); e != nil {
			return e
		}
		s.equations[s.equationCloseIndex] = nil
		s.equationCloseIndex++
	}
	for s.solutionCloseIndex < len(s.solution) {
		if e := s.solution[s.solutionCloseIndex].Close(); e != nil {
			return e
		}
		s.solution[s.solutionCloseIndex] = nil
		s.solutionCloseIndex++
	}
	if e := s.queue.Close(); e != nil {
		return e
	}
	if e := s.tmpBs.Close(); e != nil {
		return e
	}
	if e := s.tmpAs.Close(); e != nil {
		return e
	}
	if e := s.bsBuffer.Close(); e != nil {
		return e
	}
	if e := s.asBuffer.Close(); e != nil {
		return e
	}
	if e := s.decomposeBuffer.Close(); e != nil {
		return e
	}
	s.manager.BudgetField.Release(s.header)
	s.header = 0
	s.parameter, s.solved, s.occurrences, s.equations, s.solution = nil, nil, nil, nil, nil
	s.closed = true
	return nil
}
