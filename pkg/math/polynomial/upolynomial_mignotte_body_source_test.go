// Source: src/math/polynomial/upolynomial_factorization.cpp:979-1017.
package polynomial

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	primegen "github.com/glycerine/g3/pkg/util/prime"
)

func mignotteDirectFixture(t *testing.T) (*unumeral.RationalManager, *UPolynomialFactorizationService) {
	return henselDirectFixtureWithCheckpoint(t, func() bool { return true })
}
func henselDirectFixtureWithCheckpoint(t *testing.T, checkpoint func() bool) (*unumeral.RationalManager, *UPolynomialFactorizationService) {
	return henselDirectFixtureWithConfig(t, checkpoint, false)
}
func henselDirectFixtureWithConfig(t *testing.T, checkpoint func() bool, debugTag bool) (*unumeral.RationalManager, *UPolynomialFactorizationService) {
	t.Helper()
	b := arena.NewBudget(1 << 26)
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: serviceConfig(b).Integers})
	serviceMust(t, e)
	u, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.Integers(), Checkpoint: checkpoint, Cooperation: func() error { return nil }})
	serviceMust(t, e)
	primes, e := primegen.NewGenerator(b)
	serviceMust(t, e)
	s, e := NewUPolynomialFactorizationService(u, UPolynomialFactorizationConfig{Primes: primes, FactorizationDebugTag: debugTag})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, s.Close())
		serviceMust(t, primes.Close())
		serviceMust(t, u.Close())
		serviceMust(t, q.CloseChecked())
		if b.Used() != 0 {
			t.Error("actual enclosing owner refund", b.Used())
		}
	})
	return q, s
}
func TestUPolynomialMignotteLiteralIndependentExponent(t *testing.T) {
	for _, tc := range []struct {
		a     []unumeral.Int
		prime int32
		want  uint32
	}{{[]unumeral.Int{unumeral.NewInt(4), unumeral.NewInt(0), unumeral.NewInt(3)}, 2, 8}, {[]unumeral.Int{unumeral.NewInt(-2), unumeral.NewInt(0), unumeral.NewInt(1)}, 2, 4}, {[]unumeral.Int{unumeral.NewInt(-2), unumeral.NewInt(0), unumeral.NewInt(1)}, 3, 2}, {[]unumeral.Int{unumeral.NewInt(1), unumeral.NewInt(0), unumeral.NewInt(0), unumeral.NewInt(0), unumeral.NewInt(1)}, 2, 4}} {
		q, s := mignotteDirectFixture(t)
		c, e := s.newSquareFreeScope()
		serviceMust(t, e)
		c.constructed = 1
		serviceMust(t, s.manager.set(&c.fpp, tc.a))
		prime := unumeral.NewInt(tc.prime)
		got, e := s.mignotteBound(s.manager.stateView(&c.fpp), &prime)
		serviceMust(t, e)
		if got != tc.want || c.mignotte.active || c.mignotte.sourceRemaining != 0 || (c.mignotte.norm).Cell != 0 || (c.mignotte.bound).Cell != 0 || (c.mignotte.tmp).Cell != 0 {
			t.Fatal("independent source norm/root/bound/squaring exponent and lexical DT", got, tc.want, c.mignotte)
		}
		mignotteScopePhysicalAudit(t, q, s, &prime)
		serviceMust(t, s.CleanupRetained())
	}
}
func TestUPolynomialMignotteBIGEveryActualSourcePrefix(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 2048 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			q, s := mignotteDirectFixture(t)
			var big unumeral.Int
			serviceMust(t, q.Integers().SetLimbs(&big, []uint32{1, 0, 1}))
			t.Cleanup(func() { serviceMust(t, q.Integers().Del(&big)) })
			c, e := s.newSquareFreeScope()
			serviceMust(t, e)
			c.constructed = 1
			serviceMust(t, s.manager.set(&c.fpp, []unumeral.Int{unumeral.NewInt(1), unumeral.NewInt(0), big}))
			prime := unumeral.NewInt(3)
			q.Budget().FailAfter(ordinal)
			got, body := s.mignotteBound(s.manager.stateView(&c.fpp), &prime)
			mignotteScopePhysicalAudit(t, q, s, &big, &prime)
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("actual source numerical refusal", ordinal, body)
			}
			// The graph is recorded before lifting the real source fault. Child RAWs
			// are unjoined until the separate owning phase after completed older NUMs.
			q.Budget().FailAfter(-1)
			r := c.mignotte.receipt
			serviceMust(t, s.cleanupMignotteInvocation(&r, false))
			serviceMust(t, s.cleanupMignotteInvocation(&r, true))
			mignotteScopePhysicalAudit(t, q, s, &big, &prime)
			serviceMust(t, s.CleanupRetained())
			mignotteScopePhysicalAudit(t, q, s, &big, &prime)
			if body == nil {
				if got != 64 {
					t.Fatal("independent BIG bound requires 3^32 below and 3^64 above", got)
				}
				success = true
				t.Log("first actual complete BIG prefix", ordinal)
			}
		})
	}
	if !success {
		t.Fatal("actual BIG numerical campaign never completed")
	}
}

func TestUPolynomialMignotteSourceDegreeAssertionBeforeNUM(t *testing.T) {
	q, s := mignotteDirectFixture(t)
	c, e := s.newSquareFreeScope()
	serviceMust(t, e)
	c.constructed = 1
	coefficients := []unumeral.Int{unumeral.NewInt(1), unumeral.NewInt(0), unumeral.NewInt(1)}
	if polynomialSourceDebug {
		coefficients = []unumeral.Int{unumeral.NewInt(1), unumeral.NewInt(1)}
	}
	serviceMust(t, s.manager.set(&c.fpp, coefficients))
	prime := unumeral.NewInt(2)
	before := q.Budget().Used()
	_, e = s.mignotteBound(s.manager.stateView(&c.fpp), &prime)
	if polynomialSourceDebug {
		if e != ErrUPolynomialAssertion || c.mignotte.invocation != 0 || c.mignotte.constructed != 0 || q.Budget().Used() != before {
			t.Fatal("source degree assertion must precede lexical NUM construction", e, c.mignotte)
		}
	} else {
		serviceMust(t, e)
	}
	mignotteScopePhysicalAudit(t, q, s, &prime)
	serviceMust(t, s.CleanupRetained())
}
