// Source: src/math/polynomial/upolynomial.cpp:1324-1329,432-513.
// Licensed under the MIT license.
package polynomial

import (
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestUPolynomialRemoveOneHalfRootActualDivisionCaller(t *testing.T) {
	c, p := algebraicFixture(t)
	input := algebraicPolynomial(t, p, -1, 2)
	out, e := p.NewPolynomial()
	serviceMust(t, e)
	defer func() { serviceMust(t, p.DelPolynomial(&out)) }()
	v, e := p.wholeView(input)
	serviceMust(t, e)
	reached := false
	oldCheckpoint := p.config.Checkpoint
	defer func() { p.config.Checkpoint = oldCheckpoint }()
	p.config.Checkpoint = func() bool {
		reached = true
		before := p.modular.Header
		used := c.config.Rationals.Integers().Budget().Used()
		physical := c.config.Rationals.Integers().RuntimeOwner().Stats()
		q := &p.removeOneHalfRoot
		beforeFrame := *q
		r := p.removeOneHalfRootReceipt()
		for _, guard := range []func() error{p.SourceFinalize, p.Close, p.config.Integer.SourceFinalize, p.config.Integer.CloseChecked, p.config.Integer.Naturals().SourceFinalize, p.config.Integer.Naturals().CloseChecked, func() error { return p.cleanupRemoveOneHalfRootWithReceipt(&r, false) }, func() error { return p.cleanupRemoveOneHalfRootWithReceipt(&r, true) }, func() error { return p.DelPolynomial(&out) }} {
			if e := guard(); e == nil || !reflect.DeepEqual(beforeFrame, *q) || p.modular.Header != before || c.config.Rationals.Integers().Budget().Used() != used || c.config.Rationals.Integers().RuntimeOwner().Stats() != physical {
				t.Fatal("executing source/own guard mutated actual caller", e)
			}
		}
		e := p.modular.Close()
		if e != ErrPolynomialPending || p.modular.Header != before || c.config.Rationals.Integers().Budget().Used() != used || c.config.Rationals.Integers().RuntimeOwner().Stats() != physical {
			t.Fatal("actual RemoveOneHalfRoot division allowed modular owner retirement", e)
		}
		return oldCheckpoint()
	}
	serviceMust(t, p.RemoveOneHalfRoot(v, out))
	if !reached {
		t.Fatal("literal division checkpoint not reached")
	}
	ov, e := p.wholeView(out)
	serviceMust(t, e)
	if ov.count != 1 {
		t.Fatal("quotient degree", ov.count)
	}
	value, e := p.read(ov, 0)
	serviceMust(t, e)
	if *value != util.NewInt(2) {
		t.Fatal("literal integer pseudo-division quotient", *value)
	}
}
