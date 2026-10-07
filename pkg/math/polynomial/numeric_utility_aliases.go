// Licensed under the MIT license; see LICENSE.txt.
package polynomial

import (
	"github.com/glycerine/g3/pkg/util"
)

// Caller algorithms keep their own scopes; primitive ownership helpers live in util.
type polynomialNumericRoots struct {
	operation                      polynomialNumeralOperationGuard
	integerBinding, naturalBinding util.NumericCallerBinding
	util.NumericAllocationRoots
	hooks   util.NumericInvocationHooks
	caller  integerAllocationInvocationContext
	integer *util.IntegerManager
}

func polynomialNumericBefore(z *util.IntegerManager) (polynomialNumericRoots, error) {
	r, e := util.CaptureNumericAllocationRoots(z)
	return polynomialNumericRoots{NumericAllocationRoots: r}, e
}
func polynomialNegNumericPending(r *polynomialNumericRoots) bool {
	return util.NumericAllocationPending(&r.NumericAllocationRoots)
}
func polynomialNewtonJoin(a, b error) error { return util.JoinNumericErrors(a, b) }
func upolynomialSourceNumeralDel(z *util.IntegerManager, a *util.Int) (bool, error) {
	return util.SourceNumeralDel(z, a)
}

func (r *polynomialNumericRoots) After(z *util.IntegerManager, before polynomialNumericRoots) error {
	return r.NumericAllocationRoots.After(z, before.NumericAllocationRoots)
}
func (r *polynomialNumericRoots) NumericRoots(z *util.IntegerManager, local *util.NumericAllocationRoots) *util.NumericAllocationRoots {
	if r == nil || r.integer != z {
		return nil
	}
	current := r.caller.roots(z)
	if current != r {
		// The early constant SET starts with its embedded binding. A reached
		// allocation creates its stable call record during Prepare; the same
		// in-flight primitive context must then acquire into that exact record.
		b := r.caller.factorFrontendConstantFrame(z)
		if b == nil || &b.numeric != r || b.record == nil || current != &b.record.numeric {
			return nil
		}
	}
	if local != nil {
		return local
	}
	return &current.NumericAllocationRoots
}
func (c integerAllocationInvocationContext) utility(z *util.IntegerManager) util.NumericAllocationContext {
	if c == (integerAllocationInvocationContext{}) {
		return util.NumericAllocationContext{}
	}
	r := c.roots(z)
	if r == nil {
		return util.NumericAllocationContext{}.WithCaller(&invalidPolynomialNumericHooks)
	}
	r.caller = c
	r.integer = z
	r.hooks.Policy = r
	return util.NumericAllocationContext{}.WithCaller(&r.hooks)
}

var invalidPolynomialNumericHooks util.NumericInvocationHooks

// The higher source frame retains the primitive allocation receipt.
func (r *polynomialNumericRoots) after(z *util.IntegerManager, before polynomialNumericRoots) error {
	return r.After(z, before)
}
func (r *polynomialNumericRoots) cleanup(z *util.IntegerManager) error {
	return r.NumericAllocationRoots.Cleanup(z)
}
