// Source: src/math/polynomial/polynomial.cpp:5193-5272,7320-7322;
// src/test/polynomial.cpp:140-163.
package polynomial

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialQuasiResultantLiteralBoundedChains(t *testing.T) {
	for _, row := range []struct {
		p, q   []int32
		pp, qp [][]MonomialPower
		want   int32
	}{
		{[]int32{1, 1}, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil}, [][]MonomialPower{{{0, 1}}, nil}, 2},
		{[]int32{2, 1}, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil}, [][]MonomialPower{{{0, 1}}, nil}, -1},
		{[]int32{1, 1}, []int32{1, 1}, [][]MonomialPower{{{0, 3}}, nil}, [][]MonomialPower{{{0, 2}}, nil}, 2},
		{[]int32{1, 1}, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil}, [][]MonomialPower{{{0, 2}}, nil}, 0},
	} {
		for _, swap := range []bool{false, true} {
			m := polynomialPSCFullFixture(t)
			c := m.core
			p := polynomialQueryTerms(t, c, row.p, row.pp)
			q := polynomialQueryTerms(t, c, row.q, row.qp)
			if swap {
				p, q = q, p
			}
			r, e := NewPolynomialRef(c)
			serviceMust(t, e)
			serviceMust(t, m.QuasiResultant(p, q, 0, r))
			v, e := r.Get()
			serviceMust(t, e)
			constant, e := c.IsConstant(v)
			serviceMust(t, e)
			if !constant {
				t.Fatal("nonconstant chain result")
			}
			if row.want == 0 {
				zero, e := c.IsZero(v)
				serviceMust(t, e)
				if !zero {
					t.Fatal("common-factor zero")
				}
			} else {
				a, e := c.Coefficient(v, 0)
				serviceMust(t, e)
				value, e := c.integer.Int32(&a)
				serviceMust(t, e)
				want := row.want
				if swap && row.pp[0][0].Degree == 1 && row.qp[0][0].Degree == 1 {
					want = -want
				}
				if value != want {
					t.Fatal("literal remainder constant", row.p, swap, value, want)
				}
			}
			serviceMust(t, r.Close())
		}
	}
}

func TestPolynomialQuasiResultantOriginalDisplayedFixture(t *testing.T) {
	m := polynomialDiscriminantFixture(t)
	c := m.core
	parser := polynomialDiscriminantExpression{t: t, c: c}
	for i := 0; i < 2; i++ {
		x, e := c.monomials.MkVar()
		serviceMust(t, e)
		parser.variables[i] = polynomialCaller(t, c, polynomialVarsVariable(t, c, x))
	}
	// Original x0=a, x1=b fixture polynomial.cpp:160-163 is display-only,
	// dormant after driver1825; the exact elimination below is independent.
	p := parser.evaluate("((a-b)^2)-2")
	q := parser.evaluate("(b^2)-3")
	expected := parser.evaluate("(a^4)-10*(a^2)+1")
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	serviceMust(t, m.QuasiResultant(p, q, 1, out))
	same, e := c.Equal(out.root.value, expected)
	serviceMust(t, e)
	if !same {
		t.Fatal("literal source fixture pseudo-elimination")
	}
	degree, e := c.Degree(out.root.value, 1)
	serviceMust(t, e)
	if degree != 0 {
		t.Fatal("eliminated variable survives", degree)
	}
	serviceMust(t, out.Close())
}
func quasiResultantNamedAudit(t *testing.T, m *MultivariatePolynomialManager, out *PolynomialRef, known []Polynomial) {
	t.Helper()
	s := &m.core.quasiResultant
	for i := range s.refs {
		known = append(known, s.refs[i].root.value, s.refs[i].root.incoming)
	}
	known = append(known, out.root.value, out.root.incoming)
	polynomialImpPhysicalAuditNamed(t, m, known, nil, out.header)
}
func TestPolynomialQuasiResultantEveryReservationAndOwnedRetry(t *testing.T) {
	success, retained := false, false
	for prefix := int64(0); prefix < 512; prefix++ {
		stop := false
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			c := m.core
			p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 3}}, nil})
			q := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
			out, e := NewPolynomialRef(c)
			serviceMust(t, e)
			serviceMust(t, out.Assign(c.unit))
			c.integer.Budget().FailAfter(prefix)
			returned := m.QuasiResultant(p, q, 0, out)
			quasiResultantNamedAudit(t, m, out, []Polynomial{p, q})
			s := &c.quasiResultant
			if returned != nil {
				if s.bodyCause == nil && s.sourceCause == nil {
					t.Fatal("body/source refusal missing history", returned)
				}
				retained = retained || c.matchesPseudoInvocation(&s.pseudo)
			}
			bodyHistory, sourceHistory := s.bodyCause, s.sourceCause
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, m.CleanupQuasiResultant())
			if s.bodyCause != bodyHistory || s.sourceCause != sourceHistory {
				t.Fatal("own acknowledgement erased immutable history")
			}
			if returned == nil {
				success = true
				stop = true
			} else {
				if !errors.Is(returned, arena.ErrInjectedFailure) && !numericPhysicalReservationRefusal(returned) {
					t.Fatal(returned)
				}
				text := returned.Error()
				history := errors.Join(s.bodyCause, s.sourceCause)
				serviceMust(t, m.QuasiResultant(p, q, 0, out))
				if history == nil || returned.Error() != text {
					t.Fatal("immutable body error changed")
				}
			}
			quasiResultantNamedAudit(t, m, out, []Polynomial{p, q})
			serviceMust(t, out.Close())
		})
		if stop {
			break
		}
	}
	if !success || !retained {
		t.Fatal("firstsuccess/actual retained Pseudo horizon", success, retained)
	}
}

func TestPolynomialQuasiResultantConcreteFrameAndUnreadGuards(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
	q := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	var first polynomialQuasiResultantInvocationReceipt
	serviceMust(t, c.quasiResultantWithReceipt(p, q, 0, out, &first))
	copied := *first.frame
	foreign := first
	foreign.frame = &copied
	if c.matchesQuasiResultantInvocation(&foreign) {
		t.Fatal("copied frame accepted")
	}
	before := out.root.value
	serviceMust(t, c.cleanupQuasiResultantInvocation(&foreign, true))
	if out.root.value != before {
		t.Fatal("foreign history changed caller result")
	}
	var second polynomialQuasiResultantInvocationReceipt
	serviceMust(t, c.quasiResultantWithReceipt(p, q, 0, out, &second))
	if first.invocation == second.invocation || c.matchesQuasiResultantInvocation(&first) {
		t.Fatal("stale invocation retargeted")
	}
	serviceMust(t, c.cleanupQuasiResultantInvocation(&first, true))
	if !c.matchesQuasiResultantInvocation(&second) {
		t.Fatal("stale cleanup changed current history")
	}
	quasiResultantNamedAudit(t, m, out, []Polynomial{p, q})
	serviceMust(t, out.Close())
}

// The three real default reference declarations are infallible and empty.
// This fixture populates the youngest reference through its actual INC engine,
// then reaches a real last DEC while finishing the captured lexical prefix.
func TestPolynomialQuasiResultantActualDestructorReentry(t *testing.T) {
	for _, owning := range []bool{false, true} {
		full := polynomialPSCFullFixture(t)
		c := full.core
		s := &c.quasiResultant
		*s = polynomialQuasiResultantScope{invocation: 1, active: true, constructed: 3}
		for i := range s.refs {
			s.refs[i].owner = c
			s.refs[i].self = &s.refs[i]
		}
		p := polynomialICCPUnownedTerms(t, c, []int32{73}, [][]MonomialPower{nil})
		serviceMust(t, s.refs[2].root.assign(c, p))
		calls := 0
		h, e := NewPolynomialDeletionHandler(full.originalBudget, func(_ *PolynomialDeletionHandler, _ *MultivariatePolynomialCore, v Polynomial) error {
			if v != p {
				return nil
			}
			calls++
			cursor, root, cause, state := s.cursor, s.refs[2].root.value, s.cleanupCause, c.memory.State()
			if e := c.finishQuasiResultantPhase(owning, owning); !errors.Is(e, ErrPolynomialPending) {
				t.Errorf("same destructor reentry: %v", e)
			}
			if s.cursor != cursor || s.refs[2].root.value != root || s.cleanupCause != cause || c.memory.State() != state {
				t.Error("same destructor reentry changed exact prefix")
			}
			return arena.ErrBounds
		})
		serviceMust(t, e)
		serviceMust(t, c.AddDeletionHandler(h))
		returned := c.finishQuasiResultantPhase(owning, owning)
		serviceMust(t, c.RemoveDeletionHandler(h))
		serviceMust(t, h.CloseChecked())
		if calls != 1 || !errors.Is(returned, arena.ErrBounds) || s.cursor != 0 {
			t.Fatal("actual youngest destructor stop", calls, returned, s.cursor)
		}
		body, source, own := s.bodyCause, s.sourceCause, s.cleanupCause
		serviceMust(t, full.CleanupQuasiResultant())
		if s.bodyCause != body || s.sourceCause != source || s.cleanupCause != own {
			t.Fatal("acknowledgment erased first cause")
		}
		polynomialImpPhysicalAuditNamedRaw(t, full, nil, nil, 0, nil)
		old := polynomialQuasiResultantInvocationReceipt{c, s, s.invocation}
		a := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
		b := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
		output, e := NewPolynomialRef(c)
		serviceMust(t, e)
		c.integer.Budget().FailAfter(0)
		refused := full.QuasiResultant(a, b, 0, output)
		if refused != nil && !numericPhysicalReservationRefusal(refused) {
			t.Fatal("actual algorithm refusal", refused)
		}
		if s.invocation != old.invocation+1 || len(c.quasiHistory) != 1 {
			t.Fatal("native archive did not publish old tuple before next invocation", refused)
		}
		saved, ok := c.quasiResultantDiagnostic(&old)
		if !ok || saved.bodyCause != body || saved.sourceCause != source || saved.cleanupCause != own {
			t.Fatal("native archive changed old tuple")
		}
		quasiResultantNamedAudit(t, full, output, []Polynomial{a, b})
		c.integer.Budget().FailAfter(-1)
		serviceMust(t, full.CleanupQuasiResultant())
		serviceMust(t, full.QuasiResultant(a, b, 0, output))
		history, ok := c.quasiResultantDiagnostic(&old)
		if !ok || history.sourceCause != source || history.bodyCause != body || history.cleanupCause != own || c.quasiResultantInvocationSourceCause(&old) != source {
			t.Fatal("actual later invocation erased earlier source history")
		}
		serviceMust(t, output.Close())
	}
}

func TestPolynomialQuasiResultantBIGEveryReservationAndOwnedRetry(t *testing.T) {
	success, retained := false, false
	for prefix := int64(0); prefix < 512; prefix++ {
		stop := false
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			c := m.core
			var big util.Int
			serviceMust(t, c.integer.SetLimbs(&big, []uint32{17, 1 << 30, 1 << 16}))
			term, e := c.monomials.MkMonomial([]MonomialPower{{0, 3}})
			serviceMust(t, e)
			raw, e := c.MkTerms([]util.Int{big, util.NewInt(1)}, []Monomial{term, c.monomials.unit})
			serviceMust(t, e)
			p := polynomialCaller(t, c, raw)
			serviceMust(t, c.integer.Del(&big))
			q := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
			out, e := NewPolynomialRef(c)
			serviceMust(t, e)
			serviceMust(t, out.Assign(c.unit))
			c.integer.Budget().FailAfter(prefix)
			returned := m.QuasiResultant(p, q, 0, out)
			quasiResultantNamedAudit(t, m, out, []Polynomial{p, q})
			s := &c.quasiResultant
			if returned != nil {
				if s.bodyCause == nil && s.sourceCause == nil {
					t.Fatal("body/source refusal missing history", returned)
				}
				retained = retained || c.matchesPseudoInvocation(&s.pseudo)
			}
			// mpz.go ensureTmp frees the old member before replacement. A
			// refused replacement leaves the source dangling-pointer domain
			// represented by an exact guarded zero scratch slot in Go.
			scratchHole := c.integer.Tmp[0] == 0 || c.integer.Tmp[1] == 0
			bodyHistory, sourceHistory := s.bodyCause, s.sourceCause
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, m.CleanupQuasiResultant())
			if s.bodyCause != bodyHistory || s.sourceCause != sourceHistory {
				t.Fatal("own acknowledgement erased immutable history")
			}
			if returned == nil {
				success = true
				stop = true
			} else {
				if !errors.Is(returned, arena.ErrInjectedFailure) && !numericPhysicalReservationRefusal(returned) {
					t.Fatal(returned)
				}
				text := returned.Error()
				history := errors.Join(s.bodyCause, s.sourceCause)
				retry := m.QuasiResultant(p, q, 0, out)
				if scratchHole {
					if !errors.Is(retry, arena.ErrInvalidSlot) {
						t.Fatal("actual unrepaired scratch-hole guard", retry)
					}
					quasiResultantNamedAudit(t, m, out, []Polynomial{p, q})
					serviceMust(t, m.CleanupQuasiResultant())
				} else {
					serviceMust(t, retry)
				}
				if history == nil || returned.Error() != text {
					t.Fatal("immutable body error changed")
				}
			}
			quasiResultantNamedAudit(t, m, out, []Polynomial{p, q})
			serviceMust(t, out.Close())
		})
		if stop {
			break
		}
	}
	if !success || !retained {
		t.Fatal("firstsuccess/actual retained Pseudo horizon", success, retained)
	}
}

func TestPolynomialQuasiResultantActualCallerPublicationCallback(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
	q := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	old := polynomialICCPUnownedTerms(t, c, []int32{73}, [][]MonomialPower{nil})
	serviceMust(t, out.Assign(old))
	calls := 0
	h, e := NewPolynomialDeletionHandler(full.originalBudget, func(_ *PolynomialDeletionHandler, _ *MultivariatePolynomialCore, v Polynomial) error {
		if v != old {
			return nil
		}
		calls++
		s := &c.quasiResultant
		cursor, incoming, value, cause, state := s.cursor, out.root.incoming, out.root.value, s.cleanupCause, c.memory.State()
		for _, op := range []func() error{full.CleanupQuasiResultant, full.CloseChecked, c.CloseChecked, out.Close, func() error { return full.QuasiResultant(p, q, 0, out) }} {
			if x := op(); !errors.Is(x, ErrPolynomialPending) {
				t.Errorf("actual output DEC callback reentry: %v", x)
			}
		}
		if s.cursor != cursor || out.root.incoming != incoming || out.root.value != value || s.cleanupCause != cause || c.memory.State() != state {
			t.Error("output callback reentry mutated captured state")
		}
		return arena.ErrBounds
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	returned := full.QuasiResultant(p, q, 0, out)
	serviceMust(t, c.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
	if calls != 1 || !errors.Is(returned, arena.ErrBounds) || out.root.incoming == (Polynomial{}) {
		t.Fatal("actual incoming INC/old DEC stop", returned, calls)
	}
	quasiResultantNamedAudit(t, full, out, []Polynomial{p, q})
	incoming := out.root.incoming
	serviceMust(t, full.CleanupQuasiResultant())
	if out.root.incoming != incoming {
		t.Fatal("parent own ACK consumed caller incoming")
	}
	quasiResultantNamedAudit(t, full, out, []Polynomial{p, q})
	serviceMust(t, out.Close())
}
