// Source: src/math/polynomial caller reentry guards.
package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func numericIntegerModularManager_upolynomialCallerExecuting(m *unumeral.IntegerModularManager) bool {
	if m == nil || !m.ValidOwner() || m.Integers() == nil || !m.Integers().ValidOwner() {
		return false
	}
	z := m.Integers()
	if p := oneHalfRootCaller(z); p != nil && p.config.Integer == z && p.modular == m && p.oneHalfRootExecuting() {
		return true
	}
	if c := algebraicNegCaller(z); c != nil && c.config.Rationals != nil && c.config.Rationals.Integers() == z && c.negation.phase >= 2 && c.negationExecuting() {
		p := c.config.Polynomials
		if p != nil && p.self == p && p.config.Integer == z && p.modular == m && p.algebraicNegCaller == c {
			return true
		}
	}
	for q := upolyDisplayCaller(z); q != nil; q = q.previousInteger {
		if q.actual() && q.owner.config.Integer == z && q.owner.modular == m && (q.executing || q.sourceExecuting || q.ownExecuting) {
			return true
		}
	}
	if p := knuthCaller(z); p != nil && p.config.Integer == z && p.modular == m && p.knuthExecuting() {
		return true
	}
	if p := rootUpperBoundCaller(z); p != nil && p.config.Integer == z && p.modular == m && p.rootUpperBoundExecuting() {
		return true
	}
	if p := removeOneHalfRootCaller(z); p != nil && p.config.Integer == z && p.modular == m && p.removeOneHalfRootExecuting() {
		return true
	}
	if p := integerEvaluationCaller(z); p != nil && p.config.Integer == z && p.modular == m && p.integerEvaluationExecuting() {
		return true
	}
	if p := rationalEvaluationCaller(z); p != nil && p.config.Integer == z && p.modular == m && p.rationalEvaluationExecuting() {
		return true
	}
	if p := binaryEvaluationCaller(z); p != nil && p.config.Integer == z && p.modular == m && p.binaryEvaluationExecuting() {
		return true
	}
	return false
}
