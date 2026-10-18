// Source: upolynomial.cpp:366-401, reached through get_root_id's Sturm loop.
package polynomial

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type upolyRootIDNormalizeScope struct {
	values                    [3]unumeral.Int // g, old_p_i, tmp, in source construction order
	copy                      unumeral.IntegerCopyAllocationReceipt
	normalize                 unumeral.IntegerNormalizeInvocationReceipt
	invocation                uint64
	constructed, ownRemaining uint8
	executing, owning, owned  bool
	bodyCause, sourceStop     error
}

func (q *upolyRootIDScope) normalizePending() bool {
	s := &q.normalization
	return s.constructed != 0 && !s.owned || q.owner.config.Integer.MatchesCopyAllocationInvocation(&s.copy) || q.owner.modular.MatchesPNormalizeInvocation(&s.normalize)
}

func (q *upolyRootIDScope) finishNormalization(until uint8, cause error) error {
	s, z := &q.normalization, q.owner.config.Integer
	if s.sourceStop == nil {
		for i := len(q.raw) - 1; i >= 0; i-- {
			if stop := z.RawInvocationSourceCause(&q.raw[i]); stop != nil {
				s.sourceStop = stop
				break
			}
		}
	}
	if s.sourceStop == nil {
		q.finishing = true
		defer func() { q.finishing = false }()
		for s.constructed > until {
			if e := unumeral.SourceIntegerDel(z, &s.values[s.constructed-1]); e != nil {
				s.sourceStop = e
				break
			}
			s.constructed--
		}
	}
	if s.sourceStop != nil && q.childStop == nil {
		q.childStop = s.sourceStop
	}
	return errors.Join(cause, s.sourceStop)
}

func (q *upolyRootIDScope) normalizeView(v UPolynomialView) (cause error) {
	m := q.owner
	if q.context().rootIDFrame(m.config.Integer) != q {
		return ErrUPolynomialOwner
	}
	// These early returns precede the source g declaration.
	if v.count <= 1 {
		return m.normalizeWithSourceContext(v, q.context())
	}
	s := &q.normalization
	if q.normalizePending() || s.executing || s.invocation == ^uint64(0) {
		return ErrUPolynomialCleanupPending
	}
	*s = upolyRootIDNormalizeScope{invocation: s.invocation + 1, constructed: 1, executing: true}
	defer func() {
		s.executing = false
		s.bodyCause = cause
		cause = q.finishNormalization(0, cause)
	}()
	g := &s.values[0]
	if e := m.gcdSpanWithSourceContext(v, g, q.context()); e != nil {
		return e
	}
	one, e := m.modular.IsOne(g)
	if e != nil || one {
		return e
	}
	for i := uint32(0); i < v.count; i++ {
		if e := q.normalizeCoefficient(v, i); e != nil {
			return e
		}
	}
	return nil
}

func (q *upolyRootIDScope) normalizeCoefficient(v UPolynomialView, i uint32) (cause error) {
	m, s := q.owner, &q.normalization
	g, old, tmp := &s.values[0], &s.values[1], &s.values[2]
	if polynomialSourceDebug {
		s.values[1] = unumeral.Int{}
		s.constructed = 2
		defer func() { cause = q.finishNormalization(1, cause) }()
		x, e := m.read(v, i)
		if e != nil {
			return e
		}
		if e = m.modular.CopyFromOwnerTracked(old, m.config.Integer, x, &s.copy, &s.normalize); e != nil {
			return e
		}
	}
	x, e := m.read(v, i)
	if e != nil {
		return e
	}
	if e = m.modular.WithAllocationContext(q.context().utility(m.config.Integer)).Div(x, g, x); e != nil {
		return e
	}
	if polynomialSourceDebug {
		s.values[2] = unumeral.Int{}
		s.constructed = 3
		if e = m.modular.WithAllocationContext(q.context().utility(m.config.Integer)).Mul(g, x, tmp); e != nil {
			return e
		}
		equal, e := m.modular.Equal(tmp, old)
		if e != nil {
			return e
		}
		if e = refinementSourceAssert(equal, 393, "tmp == old_p_i"); e != nil {
			return e
		}
	}
	return nil
}

func (q *upolyRootIDScope) cleanupNormalization() error {
	s, z := &q.normalization, q.owner.config.Integer
	if e := q.owner.modular.CleanupPNormalizeInvocation(&s.normalize, true); e != nil {
		return e
	}
	if e := z.CleanupCopyAllocationInvocation(&s.copy); e != nil {
		return e
	}
	if !s.owning {
		s.ownRemaining = s.constructed
		s.owning = true
	}
	for s.ownRemaining > 0 {
		if e := z.CleanupRawNumeral(&s.values[s.ownRemaining-1]); e != nil {
			return e
		}
		s.ownRemaining--
	}
	s.owned = true
	return nil
}
