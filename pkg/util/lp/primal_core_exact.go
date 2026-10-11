// Source: src/util/lp/lp_primal_core_solver.h: stage-one mpq/impq constructor,
// init_inf_set and get_column_out_of_bounds_delta_sign.
package lp

import "github.com/glycerine/g3/pkg/util/numeral"

// PairPrimalCoreSolver is the retained exact primal core. Construction, bound
// classification, tableau cost updates, exact pricing and LU entry points are
// implemented; the complete Solve loop remains open. Inputs are borrowed.
type PairPrimalCoreSolver struct {
	*PairCoreSolverBase
	Beta                                         *IndexedRationalVector
	BreakpointQueue                              *PairPriorityQueue
	Breakpoints                                  []*PairBreakpoint
	EpsilonOfReducedCost, ConvertedHarrisEpsilon numeral.Rational
	CostsBackup                                  *IndexedRationalVector
	ColumnNormUpdateCounter, BlandModeThreshold  uint32
	SignOfEnteringDelta                          int32
	NonBasisList                                 []uint32
	LeftBasisTableau                             IntSet
	LeavingCandidates                            []uint32
	calls                                        exactEtaCalls
	work                                         *IndexedRationalVector
	pairWork                                     *IndexedPairVector
	stepEpsilon                                  *IndexedPairVector
	basisMultiplier                              *IndexedRationalVector
	breakpointWork                               *IndexedPairVector
	pendingBreakpoint                            *PairBreakpoint
	stopped                                      bool
}

func NewPairPrimalCoreSolver(in PairCoreInputs) (m *PairPrimalCoreSolver, err error) {
	base, err := NewPairCoreSolverBase(in)
	if base == nil {
		return nil, err
	}
	m = &PairPrimalCoreSolver{PairCoreSolverBase: base, EpsilonOfReducedCost: numeral.NewRational(), ConvertedHarrisEpsilon: numeral.NewRational()}
	defer func() {
		m.stopped = err != nil
		if err != nil {
			base.stopped = true
		}
	}()
	if err != nil {
		return m, err
	}
	m.BreakpointQueue = NewPairPriorityQueue(base.manager, 0)
	m.Beta = NewIndexedRationalVector(base.manager, in.A.RowCount())
	before, err := m.calls.begin(base.manager)
	if err != nil {
		return m, err
	}
	defer func() { err = m.calls.after(base.manager, before, err) }()
	m.work = NewIndexedRationalVector(base.manager, 3)
	temporary, one, den := &m.work.Data[0], &m.work.Data[1], &m.work.Data[2]
	if err = base.manager.SetInt32(one, 1); err != nil {
		return m, err
	}
	if err = base.manager.SetInt32(den, 10000000); err != nil {
		return m, err
	}
	if err = etaRationalQuotient(base.manager, temporary, &m.EpsilonOfReducedCost, one, den); err != nil {
		return m, err
	}
	// The source finishes epsilon member construction before default backup and
	// converted-Harris members, then assigns threshold/Harris/status in the body.
	m.CostsBackup = NewIndexedRationalVector(base.manager, 0)
	m.BlandModeThreshold = 1000
	// Precise traits take the zero branch regardless of floating tolerance.
	m.Status = StatusUnknown
	if err = m.work.Close(); err != nil {
		return m, err
	}
	m.work = nil
	return m, nil
}
func (m *PairPrimalCoreSolver) ready() error {
	if m.stopped || m.work != nil || m.pairWork != nil || m.stepEpsilon != nil || m.basisMultiplier != nil || m.breakpointWork != nil || m.pendingBreakpoint != nil || m.calls.pending() {
		return ErrExactVectorPending
	}
	return m.PairCoreSolverBase.ready()
}
func (m *PairPrimalCoreSolver) InitInfSet() error {
	if err := m.ready(); err != nil {
		return err
	}
	if err := m.InfSet.Clear(); err != nil {
		return err
	}
	for j := uint32(0); j < m.A.ColumnCount(); j++ {
		if uint64(j) >= uint64(len(*m.Heading)) {
			return ErrBounds
		}
		if (*m.Heading)[j] < 0 {
			continue
		}
		ok, err := m.ColumnIsFeasible(j)
		if err != nil {
			return err
		}
		if !ok {
			if err = m.InfSet.Insert(j); err != nil {
				return err
			}
		}
	}
	return nil
}
func (m *PairPrimalCoreSolver) GetColumnOutOfBoundsDeltaSign(j uint32) (int32, error) {
	if err := m.ready(); err != nil {
		return 0, err
	}
	if uint64(j) >= uint64(len(*m.ColumnTypes)) || uint64(j) >= uint64(len(*m.X)) {
		return 0, ErrBounds
	}
	t := (*m.ColumnTypes)[j]
	switch t {
	case ColumnFixed, ColumnBoxed, ColumnLowBound:
		if uint64(j) >= uint64(len(*m.LowBounds)) {
			return 0, ErrBounds
		}
		below, err := m.BelowBound(&(*m.X)[j], &(*m.LowBounds)[j])
		if err != nil {
			return 0, err
		}
		if below {
			return -1, nil
		}
		if t == ColumnLowBound {
			return 0, nil
		}
		fallthrough
	case ColumnUpperBound:
		if uint64(j) >= uint64(len(*m.UpperBounds)) {
			return 0, ErrBounds
		}
		above, err := m.AboveBound(&(*m.X)[j], &(*m.UpperBounds)[j])
		if err != nil {
			return 0, err
		}
		if above {
			return 1, nil
		}
		return 0, nil
	case ColumnFree:
		return 0, nil
	default:
		return 0, sourceAssertionFailure()
	}
}
func (m *PairPrimalCoreSolver) Close() error {
	if err := m.calls.cleanupRaw(m.manager); err != nil {
		return err
	}
	if m.pairWork != nil {
		if err := m.pairWork.Close(); err != nil {
			return err
		}
		m.pairWork = nil
	}
	if m.work != nil {
		if err := m.work.Close(); err != nil {
			return err
		}
		m.work = nil
	}
	if m.stepEpsilon != nil {
		if err := m.stepEpsilon.Close(); err != nil {
			return err
		}
		m.stepEpsilon = nil
	}
	if m.pendingBreakpoint != nil {
		if err := m.pendingBreakpoint.Delta.Del(m.manager); err != nil {
			return err
		}
		m.pendingBreakpoint = nil
	}
	if m.breakpointWork != nil {
		if err := m.breakpointWork.Close(); err != nil {
			return err
		}
		m.breakpointWork = nil
	}
	if m.basisMultiplier != nil {
		if err := m.basisMultiplier.Close(); err != nil {
			return err
		}
		m.basisMultiplier = nil
	}
	if err := m.manager.Del(&m.ConvertedHarrisEpsilon); err != nil {
		return err
	}
	if m.CostsBackup != nil {
		if err := m.CostsBackup.Close(); err != nil {
			return err
		}
	}
	if err := m.manager.Del(&m.EpsilonOfReducedCost); err != nil {
		return err
	}
	if m.Beta != nil {
		if err := m.Beta.Close(); err != nil {
			return err
		}
	}
	if m.BreakpointQueue != nil {
		if err := m.BreakpointQueue.Close(); err != nil {
			return err
		}
	}
	if err := m.destroyBreakpointRecords(); err != nil {
		return err
	}
	if err := m.calls.cleanupRoots(m.manager); err != nil {
		return err
	}
	return m.PairCoreSolverBase.Close()
}
