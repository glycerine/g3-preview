package polynomial

import (
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// Header, body, endl and footer callbacks borrow the actual enclosing owners.
// Accepted numerical child ownership remains with its separate receipt.
type upolyFactorTraceBorrow struct {
	owner     *upolyFactorSquareFreeScope
	tags      *diagnostic.TraceTags
	executing bool
	previous  [3]*upolyFactorTraceBorrow
	bindings  [5]unumeral.NumericCallerBinding
	bound     uint8
}

func (q *upolyFactorTraceBorrow) NumericExecuting() bool { return q.executing }
func (q *upolyFactorTraceBorrow) callback() bool {
	return q.tags.OutputCallbackActive() || q.owner.manager.config.Integer.RuntimeOwner().FailureTransportActive()
}
func (q *upolyFactorTraceBorrow) managers() [3]*UPolynomialManager {
	return [3]*UPolynomialManager{q.owner.manager, q.owner.zp, q.owner.zpe}
}
func (q *upolyFactorTraceBorrow) bind(c *upolyFactorSquareFreeScope, tags *diagnostic.TraceTags) error {
	if q.executing {
		return ErrUPolynomialCleanupPending
	}
	*q = upolyFactorTraceBorrow{owner: c, tags: tags, executing: true}
	for i, m := range q.managers() {
		if m != nil {
			q.previous[i] = m.factorTrace
			m.factorTrace = q
		}
	}
	z := c.manager.config.Integer
	if e := z.BindCaller(&q.bindings[0], q, 0); e != nil {
		q.unbind()
		return e
	}
	q.bound++
	if e := z.Natural.BindCaller(&q.bindings[1], q, 0); e != nil {
		q.unbind()
		return e
	}
	q.bound++
	for _, m := range q.managers() {
		if m == nil {
			continue
		}
		if e := m.modular.BindCaller(&q.bindings[q.bound], q, 0); e != nil {
			q.unbind()
			return e
		}
		q.bound++
	}
	return nil
}
func (q *upolyFactorTraceBorrow) unbind() {
	for q.bound != 0 {
		q.bound--
		if e := q.bindings[q.bound].Unbind(); e != nil {
			panic(e)
		}
	}
	for i, m := range q.managers() {
		if m != nil {
			if m.factorTrace != q {
				panic(ErrUPolynomialOwner)
			}
			m.factorTrace = q.previous[i]
			q.previous[i] = nil
		}
	}
	q.executing = false
	q.tags = nil
}
