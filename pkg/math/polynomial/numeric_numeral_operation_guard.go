// Source: math/polynomial numerical calls; synchronous utility owner exclusion.
package polynomial

import (
	"github.com/glycerine/g3/pkg/util"
)

type polynomialNumeralOperationGuard struct {
	binding   util.NumericCallerBinding
	executing bool
}

func (g *polynomialNumeralOperationGuard) NumericExecuting() bool { return g != nil && g.executing }
func (g *polynomialNumeralOperationGuard) enter(z *util.IntegerManager) error {
	if g.executing {
		return ErrPolynomialPending
	}
	g.executing = true
	if e := z.BindCaller(&g.binding, g, 16); e != nil {
		g.executing = false
		return e
	}
	return nil
}
func (g *polynomialNumeralOperationGuard) leave() {
	if e := g.binding.Unbind(); e != nil {
		panic(e)
	}
	g.executing = false
}
func factorNumeralCallExecuting(z *util.IntegerManager) bool {
	for b := z.CallerBindings(); b != nil; b = b.Previous() {
		if b.Kind() == 16 && b.Guard().NumericExecuting() {
			return true
		}
	}
	return false
}
