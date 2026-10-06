// Source: src/math/polynomial/polynomial.cpp:6558-6577;
// src/math/polynomial/polynomial.cpp:6307-6344.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import "errors"

// The returned adapter names the actual completed scoped constructor. If its
// source destructor stops, the caller retains precisely this adapter for own
// acknowledgment; a failed constructor returns no constructed scope.
func (m *MultivariatePolynomialManager) minimumDegreeVariable(p Polynomial) (x uint32, scope *ScopedPolynomialMaxDegrees, cause error) {
	if e := m.ready(); e != nil {
		return 0, nil, e
	}
	if modularSourceDebug {
		constant, e := m.core.IsConstant(p)
		if e != nil {
			return 0, nil, e
		}
		if constant {
			return 0, nil, ErrPolynomialAssertion
		}
	}
	scope, cause = NewScopedPolynomialMaxDegrees(&m.core.substitution.max, p)
	if cause != nil {
		return 0, nil, cause
	}
	defer func() {
		dt := scope.SourceFinalize()
		cause = errors.Join(cause, dt)
		if dt == nil {
			cause = errors.Join(cause, scope.CloseChecked())
		}
	}()
	n, e := scope.NumVariables()
	if e != nil {
		return 0, scope, e
	}
	xs, e := scope.Vars()
	if e != nil {
		return 0, scope, e
	}
	x, min := ^uint32(0), ^uint32(0)
	for i := uint32(0); i < n; i++ {
		xi, e := xs.At(i)
		if e != nil {
			return 0, scope, e
		}
		degree, e := scope.Degree(xi)
		if e != nil {
			return 0, scope, e
		}
		if degree < min {
			x, min = xi, degree
		}
	}
	return x, scope, nil
}
