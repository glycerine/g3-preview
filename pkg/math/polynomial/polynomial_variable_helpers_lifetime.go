// Source: src/math/polynomial/polynomial.cpp:6179-6232,6307-6344; src/util/vector.h:101-214,530-546.
// Licensed under the MIT license.
package polynomial

import (
	"errors"

	"github.com/glycerine/g3/pkg/util"
)

func (d *PolynomialMaxDegrees) SourceFinalize() error {
	if e := d.identity(); e != nil {
		return e
	}
	if d.sourceCause != nil {
		return d.sourceCause
	}
	if d.finalized {
		return nil
	}
	d.finalizing = true
	if d.sourceStage == 0 {
		if e := d.variables.SourceFinalize(); e != nil {
			d.sourceCause = e
			return e
		}
		d.sourceStage = 1
	}
	if d.sourceStage == 1 {
		if e := d.maxDegree.SourceFinalize(); e != nil {
			d.sourceCause = e
			return e
		}
		d.sourceStage = 2
	}
	d.finalized = true
	return nil
}
func (d *PolynomialMaxDegrees) CloseChecked() error {
	if d == nil || d.self != d {
		return ErrPolynomialOwner
	}
	if d.closed {
		return nil
	}
	d.closing = true
	if e := d.variables.CloseChecked(); e != nil {
		return e
	}
	if e := d.maxDegree.CloseChecked(); e != nil {
		return e
	}
	d.runtime.Budget().Release(d.header)
	d.header = 0
	d.closed = true
	return nil
}
func (d *PolynomialVariablePositions) SourceFinalize() error {
	if e := d.identity(); e != nil {
		return e
	}
	if d.sourceCause != nil {
		return d.sourceCause
	}
	if d.finalized {
		return nil
	}
	d.finalizing = true
	if e := d.positions.SourceFinalize(); e != nil {
		d.sourceCause = e
		return e
	}
	d.finalized = true
	return nil
}
func (d *PolynomialVariablePositions) CloseChecked() error {
	if d == nil || d.self != d {
		return ErrPolynomialOwner
	}
	if d.closed {
		return nil
	}
	d.closing = true
	if e := d.positions.CloseChecked(); e != nil {
		return e
	}
	d.runtime.Budget().Release(d.header)
	d.header = 0
	d.closed = true
	return nil
}
func variableCopyConstruct(to, from *PolynomialVariableDegrees) error {
	if e := from.open(); e != nil {
		return e
	}
	if from.raw != (util.Block{}) {
		if e := to.copyCore(from); e != nil {
			to.bodyCause = e
			return e
		}
	}
	if polynomialSourceDebug {
		n, e := to.size()
		if e != nil {
			return e
		}
		other, e := from.size()
		if e != nil {
			return e
		}
		if n != other {
			return ErrPolynomialAssertion
		}
	}
	to.constructed = true
	return nil
}
func variableMoveConstruct(to, from *PolynomialVariableDegrees) {
	to.raw, from.raw = from.raw, to.raw
	to.rawDestroyed, from.rawDestroyed = from.rawDestroyed, false
	to.constructed = true
}
func variableOwns(d *PolynomialVariableDegrees) bool {
	return d.pending != (util.Block{}) || (d.raw != (util.Block{}) && !d.rawDestroyed) || d.sourceCause != nil
}
func NewPolynomialMaxDegreesCopy(source *PolynomialMaxDegrees) (*PolynomialMaxDegrees, error) {
	if e := source.open(); e != nil {
		return nil, e
	}
	d, e := newPolynomialMaxDegrees(source.core, source.runtime, false)
	if e != nil {
		return nil, e
	}
	e = variableCopyConstruct(&d.maxDegree, &source.maxDegree)
	if e == nil {
		e = variableCopyConstruct(&d.variables, &source.variables)
	}
	if e != nil {
		d.bodyCause = e
		cleanup := d.SourceFinalize()
		e = errors.Join(e, cleanup)
		if !variableOwns(&d.maxDegree) && !variableOwns(&d.variables) && cleanup == nil {
			d.runtime.Budget().Release(d.header)
			d.header = 0
			d.closed = true
			return nil, e
		}
		return d, e
	}
	d.constructed = true
	return d, nil
}
func NewPolynomialMaxDegreesMove(source *PolynomialMaxDegrees) (*PolynomialMaxDegrees, error) {
	if e := source.open(); e != nil {
		return nil, e
	}
	d, e := newPolynomialMaxDegrees(source.core, source.runtime, false)
	if e != nil {
		return nil, e
	}
	variableMoveConstruct(&d.maxDegree, &source.maxDegree)
	variableMoveConstruct(&d.variables, &source.variables)
	d.constructed = true
	return d, nil
}
func (d *PolynomialMaxDegrees) CopyFrom(source *PolynomialMaxDegrees) error {
	if e := d.open(); e != nil {
		return e
	}
	if e := source.open(); e != nil {
		return e
	}
	if e := d.maxDegree.CopyFrom(&source.maxDegree); e != nil {
		d.bodyCause = e
		return e
	}
	if e := d.variables.CopyFrom(&source.variables); e != nil {
		d.bodyCause = e
		return e
	}
	return nil
}

// Both actual svector members select const& assignment for rvalue arguments.
func (d *PolynomialMaxDegrees) MoveFrom(source *PolynomialMaxDegrees) error {
	return d.CopyFrom(source)
}
func NewPolynomialVariablePositionsCopy(source *PolynomialVariablePositions) (*PolynomialVariablePositions, error) {
	if e := source.open(); e != nil {
		return nil, e
	}
	d, e := newPolynomialVariablePositions(source.runtime, false)
	if e != nil {
		return nil, e
	}
	e = variableCopyConstruct(&d.positions, &source.positions)
	if e != nil {
		d.bodyCause = e
		if !variableOwns(&d.positions) {
			d.runtime.Budget().Release(d.header)
			d.header = 0
			d.closed = true
			return nil, e
		}
		return d, e
	}
	d.constructed = true
	return d, nil
}
func NewPolynomialVariablePositionsMove(source *PolynomialVariablePositions) (*PolynomialVariablePositions, error) {
	if e := source.open(); e != nil {
		return nil, e
	}
	d, e := newPolynomialVariablePositions(source.runtime, false)
	if e != nil {
		return nil, e
	}
	variableMoveConstruct(&d.positions, &source.positions)
	d.constructed = true
	return d, nil
}
func (d *PolynomialVariablePositions) CopyFrom(source *PolynomialVariablePositions) error {
	if e := d.open(); e != nil {
		return e
	}
	if e := source.open(); e != nil {
		return e
	}
	return d.positions.CopyFrom(&source.positions)
}
func (d *PolynomialVariablePositions) MoveFrom(source *PolynomialVariablePositions) error {
	return d.CopyFrom(source)
}
