package polynomial

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/z3exception"
	"testing"
)

func TestUPolynomialPrimeExtractionSourceExceptionAndPrefix(t *testing.T) {
	for _, tc := range []struct {
		name, value string
		accepted    bool
		want        uint32
	}{
		{"narrowing", "4294967296", false, 0},
		{"uint64-overflow", "18446744073709551617", false, 0},
		{"boundary", "4294967295", true, 4294967295},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, q, _ := upolySourceOwner(t)
			var p unumeral.Int
			serviceMust(t, q.Integers().SetString(&p, tc.value))
			serviceMust(t, m.SetZP(&p))
			serviceMust(t, q.Integers().Del(&p))
			value, e := m.factorPrime()
			if tc.accepted {
				serviceMust(t, e)
				if value != tc.want {
					t.Fatal("source unsigned narrowing boundary", value)
				}
				return
			}
			assertUPolynomialException(t, e, "The prime number attempted in factorization is too big!", ErrUPolynomialDomain)
			f := upolySourceNew(t, m, []int32{1, 0, 1})
			fs, e := m.NewFactors()
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, fs.Close()) })
			before := q.Budget().Used()
			yes, e := m.FactorZPSquareFreeBerlekamp(mustUPolyView(t, m, f), fs, false)
			assertUPolynomialException(t, e, "The prime number attempted in factorization is too big!", ErrUPolynomialDomain)
			n, e := fs.DistinctFactors()
			serviceMust(t, e)
			if yes || n != 0 || q.Budget().Used() != before {
				t.Fatal("prime exception passed source matrix/factor publication frontier", yes, n)
			}
		})
	}
}
func assertUPolynomialException(t *testing.T, e error, message string, sentinel error) {
	t.Helper()
	var source UPolynomialException
	var base z3exception.DefaultException
	if !errors.Is(e, sentinel) || !errors.As(e, &source) || !errors.As(e, &base) || source.Error() != message || base.Message() != message || source.HasErrorCode() {
		t.Fatal("source upolynomial/default exception or Go classification", e, source, base)
	}
}
func TestUPolynomialCheckpointSourceExceptionAndCooperationOrder(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	calls, cooperation := 0, 0
	m.config.Checkpoint = func() bool { calls++; return false }
	m.config.Cooperation = func() error { cooperation++; return nil }
	e := m.checkpoint()
	assertUPolynomialException(t, e, "canceled", ErrUPolynomialCanceled)
	var copied UPolynomialException
	if !errors.As(e, &copied) {
		t.Fatal("source cancellation class")
	}
	copied.MessageField = "caller changed its copy"
	assertUPolynomialException(t, e, "canceled", ErrUPolynomialCanceled)
	if calls != 1 || cooperation != 0 || m.callbackDepth != 0 {
		t.Fatal("cancellation order/borrow", calls, cooperation, m.callbackDepth)
	}
	marker := errors.New("cooperation failure")
	m.config.Checkpoint = func() bool { calls++; return true }
	m.config.Cooperation = func() error { cooperation++; return marker }
	e = m.checkpoint()
	if e != marker || calls != 2 || cooperation != 1 || m.callbackDepth != 0 {
		t.Fatal("cooperation error replaced or reordered", e, calls, cooperation)
	}
}
