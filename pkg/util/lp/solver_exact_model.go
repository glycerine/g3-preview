// Source: src/util/lp/lp_solver.h:45-140 and lp_solver.hpp:25-91.
// Concrete common model of the retained lp_solver<mpq,mpq> instantiation.
package lp

import (
	"strconv"

	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/insdict"
)

type RationalConstraint struct {
	RightSide *numeral.Rational
	Relation  LPRelation
}

// RationalLPSolver owns the exact input model shared by the source primal and
// dual solvers. Core preparation/optimization is not implemented by this layer.
// Dictionaries use insertion order in place of source unordered traversal.
// Do not copy owning values or the solver. Column owners and numerical call
// receipts close before model entries and their borrowed numeral manager.
type RationalLPSolver struct {
	AValues               *insdict.Dict[uint32, *insdict.Dict[uint32, *numeral.Rational]]
	Constraints           *insdict.Dict[uint32, *RationalConstraint]
	Columns               *insdict.Dict[uint32, *RationalColumnInfo]
	NamesToColumns        map[string]uint32
	CoreColumnsToExternal map[uint32]uint32
	Status                Status
	TotalIterations       uint32
	Settings              *Settings
	manager               *numeral.RationalManager
	calls                 exactEtaCalls
	work                  *IndexedRationalVector
	values                []*numeral.Rational
	stopped, closed       bool
}

func NewRationalLPSolver(m *numeral.RationalManager) (*RationalLPSolver, error) {
	if m == nil {
		return nil, ErrExactVectorOwner
	}
	settings, err := NewSettings()
	if err != nil {
		return nil, err
	}
	return &RationalLPSolver{manager: m, Settings: settings, AValues: insdict.NewDict[uint32, *insdict.Dict[uint32, *numeral.Rational]](), Constraints: insdict.NewDict[uint32, *RationalConstraint](), Columns: insdict.NewDict[uint32, *RationalColumnInfo](), NamesToColumns: make(map[string]uint32), CoreColumnsToExternal: make(map[uint32]uint32)}, nil
}
func (m *RationalLPSolver) ready() error {
	if m.closed || m.stopped || m.work != nil || m.calls.pending() {
		return ErrExactVectorPending
	}
	return nil
}
func (m *RationalLPSolver) newValue() *numeral.Rational {
	v := numeral.NewRational()
	m.values = append(m.values, &v)
	return &v
}
func (m *RationalLPSolver) copyValue(out, value *numeral.Rational) error {
	if out == nil || value == nil {
		return ErrBounds
	}
	b, err := m.calls.begin(m.manager)
	if err != nil {
		return err
	}
	err = m.manager.Copy(out, value)
	return m.calls.after(m.manager, b, err)
}
func (m *RationalLPSolver) getOrCreateColumn(column uint32) (*RationalColumnInfo, error) {
	if c, ok := m.Columns.Get2(column); ok {
		if c == nil {
			return nil, ErrBounds
		}
		return c, nil
	}
	c := NewRationalColumnInfo(m.manager, InvalidColumnIndex)
	m.Columns.Set(column, c)
	return c, nil
}
func (m *RationalLPSolver) GetOrCreateColumnInfo(column uint32) (*RationalColumnInfo, error) {
	if err := m.ready(); err != nil {
		return nil, err
	}
	return m.getOrCreateColumn(column)
}
func (m *RationalLPSolver) SetRowColumnCoefficient(row, column uint32, value *numeral.Rational) (err error) {
	if err = m.ready(); err != nil {
		return err
	}
	defer func() { m.stopped = err != nil }()
	r, ok := m.AValues.Get2(row)
	if !ok {
		r = insdict.NewDict[uint32, *numeral.Rational]()
		m.AValues.Set(row, r)
	}
	if r == nil {
		return ErrBounds
	}
	dst, ok := r.Get2(column)
	if !ok {
		dst = m.newValue()
		r.Set(column, dst)
	}
	return m.copyValue(dst, value)
}
func (m *RationalLPSolver) AddConstraint(relation LPRelation, rhs *numeral.Rational, row uint32) (err error) {
	if err = m.ready(); err != nil {
		return err
	}
	defer func() { m.stopped = err != nil }()
	if rhs == nil {
		return ErrBounds
	}
	m.work = NewIndexedRationalVector(m.manager, 3)
	w := m.work
	if err = w.copySlot(&w.Data[0], rhs); err != nil {
		return err
	}
	c, exists := m.Constraints.Get2(row)
	if exists {
		if err = sourceAssertionFailure(); err != nil {
			return err
		}
	}
	// lp_constraint(T,relation) has its own by-value parameter and owning member.
	if err = w.copySlot(&w.Data[1], &w.Data[0]); err != nil {
		return err
	}
	if err = w.copySlot(&w.Data[2], &w.Data[1]); err != nil {
		return err
	}
	if err = m.manager.Del(&w.Data[1]); err != nil {
		return err
	}
	w.Data[1] = numeral.NewRational()
	if !exists {
		c = &RationalConstraint{RightSide: m.newValue()}
		m.Constraints.Set(row, c)
	}
	if c == nil {
		return ErrBounds
	}
	if err = m.copyValue(c.RightSide, &w.Data[2]); err != nil {
		return err
	}
	c.Relation = relation
	if err = w.Close(); err != nil {
		return err
	}
	m.work = nil
	return nil
}

// These four source entry points take the numeral by value. Column mutation
// owns its own receipts; the parent parameter remains alive on child failure.
func (m *RationalLPSolver) setColumnValue(column uint32, value *numeral.Rational, which uint8) (err error) {
	if err = m.ready(); err != nil {
		return err
	}
	defer func() { m.stopped = err != nil }()
	if value == nil {
		return ErrBounds
	}
	m.work = NewIndexedRationalVector(m.manager, 1)
	if err = m.work.copySlot(&m.work.Data[0], value); err != nil {
		return err
	}
	c, e := m.getOrCreateColumn(column)
	if e != nil {
		return e
	}
	v := &m.work.Data[0]
	switch which {
	case 0:
		err = c.SetCost(v)
	case 1:
		err = c.SetLowBound(v)
	case 2:
		err = c.SetUpperBound(v)
	case 3:
		err = c.SetFixedValue(v)
	}
	if err != nil {
		return err
	}
	if err = m.work.Close(); err != nil {
		return err
	}
	m.work = nil
	return nil
}
func (m *RationalLPSolver) SetCostForColumn(column uint32, v *numeral.Rational) error {
	return m.setColumnValue(column, v, 0)
}
func (m *RationalLPSolver) SetLowBound(column uint32, v *numeral.Rational) error {
	return m.setColumnValue(column, v, 1)
}
func (m *RationalLPSolver) SetUpperBound(column uint32, v *numeral.Rational) error {
	return m.setColumnValue(column, v, 2)
}
func (m *RationalLPSolver) SetFixedValue(column uint32, v *numeral.Rational) error {
	return m.setColumnValue(column, v, 3)
}
func (m *RationalLPSolver) GiveSymbolicNameToColumn(name string, column uint32) error {
	if err := m.ready(); err != nil {
		return err
	}
	c, err := m.getOrCreateColumn(column)
	if err != nil {
		return err
	}
	c.Name = name
	m.NamesToColumns[name] = column
	return nil
}
func (m *RationalLPSolver) ColumnName(core uint32) string {
	ext, ok := m.CoreColumnsToExternal[core]
	if !ok {
		return "x" + strconv.FormatUint(uint64(core), 10)
	}
	c, ok := m.Columns.Get2(ext)
	if !ok {
		return "x" + strconv.FormatUint(uint64(ext), 10)
	}
	if c == nil {
		return ""
	} // undefined source null-column domain
	return c.Name
}
func (m *RationalLPSolver) ColumnIndexByName(name string) int32 {
	j, ok := m.NamesToColumns[name]
	if !ok {
		return -1
	}
	return int32(j)
}
func (m *RationalLPSolver) ProblemIsEmpty() bool {
	for _, row := range m.AValues.All() {
		if row != nil && row.Len() != 0 {
			return false
		}
	}
	return true
}
func (m *RationalLPSolver) Close() error {
	if m.closed {
		return nil
	}
	if err := m.calls.cleanupRaw(m.manager); err != nil {
		return err
	}
	// Interrupted column setters may borrow the parent's by-value parameter.
	for _, c := range m.Columns.All() {
		if c != nil {
			if err := c.Close(); err != nil {
				return err
			}
		}
	}
	if m.work != nil {
		if err := m.work.Close(); err != nil {
			return err
		}
		m.work = nil
	}
	for len(m.values) > 0 {
		last := len(m.values) - 1
		if err := m.manager.Del(m.values[last]); err != nil {
			return err
		}
		m.values = m.values[:last]
	}
	if err := m.calls.cleanupRoots(m.manager); err != nil {
		return err
	}
	m.closed = true
	return nil
}
