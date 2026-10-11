// Source: src/util/lp/mps_reader.h:680-739, exact lp_solver<mpq,mpq> population.
package lp

import "github.com/glycerine/g3/pkg/util/numeral"

// fillRationalSolver populates the common exact model. Concrete outer primal
// and dual creation/optimization are separate unfinished source consumers.
func (m *RationalMPSReader) fillRationalSolver(s *RationalLPSolver) (err error) {
	if err = m.ready(); err != nil {
		return err
	}
	defer func() { m.stopped = err != nil }()
	if s == nil || s.manager != m.manager {
		return ErrExactVectorOwner
	}
	for _, row := range m.rows.All() {
		cost := row.name == m.costName
		if !cost {
			var rel LPRelation
			switch row.typ {
			case mpsLess:
				rel = RelationLessOrEqual
			case mpsGreater:
				rel = RelationGreaterOrEqual
			case mpsEqual:
				rel = RelationEqual
			default:
				if err = m.message("Unexpected rt %d\n", row.typ); err != nil {
					return err
				}
				if err = m.fail(); err != nil {
					return err
				}
				return ErrMPSUnhandled
			}
			if err = s.AddConstraint(rel, row.rhs, row.index); err != nil {
				return err
			}
		}
		for name, value := range row.coefficients.All() {
			m.work = NewIndexedRationalVector(m.manager, 1)
			if err = m.copyValue(&m.work.Data[0], value); err != nil {
				return err
			}
			col, ok := m.columns.Get2(name)
			if !ok {
				if err = sourceAssertionFailure(); err != nil {
					return err
				}
				m.columns.Set(name, nil)
				return ErrBounds
			}
			if col == nil {
				return ErrBounds
			}
			if cost {
				err = s.SetCostForColumn(col.index, &m.work.Data[0])
			} else {
				err = s.SetRowColumnCoefficient(row.index, col.index, &m.work.Data[0])
			}
			if err != nil {
				return err
			}
			if err = m.finishWork(); err != nil {
				return err
			}
		}
	}
	for _, col := range m.columns.All() {
		if col == nil {
			return ErrBounds
		}
		b := col.bound
		if b == nil {
			zero := numeral.NewRational()
			if err = s.SetLowBound(col.index, &zero); err != nil {
				return err
			}
		} else if !b.free {
			if b.lowSet {
				if err = s.SetLowBound(col.index, b.low); err != nil {
					return err
				}
			}
			if b.upperSet {
				if err = s.SetUpperBound(col.index, b.upper); err != nil {
					return err
				}
			}
			if b.fixedSet {
				if err = s.SetFixedValue(col.index, b.fixed); err != nil {
					return err
				}
			}
		}
		if err = s.GiveSymbolicNameToColumn(col.name, col.index); err != nil {
			return err
		}
	}
	return nil
}
