// Source: src/util/vector.h:45–201,389–435; src/sat/sat_solver.cpp:3614–3623.
package sat

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// SAT supplies literal interpretation; util owns the unsigned nested vectors.
// A defined type preserves the concrete owner address and reservation sequence.
type satSolverLiteralRows util.RuntimeUnsignedRows

func newSATSolverLiteralRows(r *util.Runtime, b *arena.Budget) (*satSolverLiteralRows, error) {
	o, e := util.NewRuntimeUnsignedRows(r, b)
	return (*satSolverLiteralRows)(o), e
}
func (o *satSolverLiteralRows) storage() *util.RuntimeUnsignedRows {
	return (*util.RuntimeUnsignedRows)(o)
}
func (o *satSolverLiteralRows) rowCount() (uint32, error) { return o.storage().RowCount() }
func (o *satSolverLiteralRows) appendColumn(c *RuntimeUintColumn) error {
	return o.storage().AppendColumn(c)
}
func (o *satSolverLiteralRows) unsignedRow(i uint32) (*RuntimeUintColumn, error) {
	return o.storage().Row(i)
}
func (o *satSolverLiteralRows) SourceFinalize() error { return o.storage().SourceFinalize() }
func (o *satSolverLiteralRows) CloseChecked() error   { return o.storage().CloseChecked() }
func (o *satSolverLiteralRows) CleanupStopped() error { return o.storage().CleanupStopped() }
func (o *satSolverLiteralRows) appendLiterals(input []Literal) (err error) {
	c, e := o.storage().BeginRow()
	if e != nil {
		return e
	}
	defer func() { err = o.storage().EndRow(err) }()
	for _, v := range input {
		if err = c.Push(v.index()); err != nil {
			return err
		}
	}
	return nil
}
func (o *satSolverLiteralRows) appendUnsigned(input []uint32) (err error) {
	c, e := o.storage().BeginRow()
	if e != nil {
		return e
	}
	defer func() { err = o.storage().EndRow(err) }()
	for _, v := range input {
		if err = c.Push(v); err != nil {
			return err
		}
	}
	return nil
}

type satSolverLiteralRowSpan struct {
	span  satSolverLiteralSpan
	count uint32
}

func (v satSolverLiteralRowSpan) read(i uint32) (Literal, error) {
	if satStorageSourceDebug && i >= v.count {
		return 0, ErrSATAssertion
	}
	return v.span.literal(i)
}
func (o *satSolverLiteralRows) captureRow(i uint32) (satSolverLiteralRowSpan, error) {
	row, e := o.unsignedRow(i)
	if e != nil {
		return satSolverLiteralRowSpan{}, e
	}
	n, e := row.Size()
	if e != nil {
		return satSolverLiteralRowSpan{}, e
	}
	return satSolverLiteralRowSpan{satConflictRuntimeSpan(row), n}, nil
}
