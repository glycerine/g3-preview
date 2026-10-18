package polynomial

import (
	"errors"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicOwnedBoundsDistinctGlobalOwner(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	service, _ := serviceNew(t)
	root := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	defer func() { serviceMust(t, c.Del(&root)) }()
	out, e := service.NewValue()
	serviceMust(t, e)
	defer func() { serviceMust(t, service.Del(&out)) }()
	for operation := uint8(1); operation <= 4; operation++ {
		serviceMust(t, service.SetInt64(&out, 99))
		switch operation {
		case 1:
			e = c.GetLowerOwnedRational(&root, service, &out)
		case 2:
			e = c.GetUpperOwnedRational(&root, service, &out)
		case 3:
			e = c.GetLowerOwnedRationalAtPrecision(&root, service, &out, 20)
		case 4:
			e = c.GetUpperOwnedRationalAtPrecision(&root, service, &out, 20)
		}
		serviceMust(t, e)
		text, e := service.String(&out)
		serviceMust(t, e)
		q := unumeral.NewRational()
		serviceMust(t, c.config.Rationals.SetString(&q, text))
		comparison, e := c.CompareRational(&root, &q)
		serviceMust(t, e)
		serviceMust(t, c.config.Rationals.Del(&q))
		if (operation == 1 || operation == 3) && comparison <= 0 || (operation == 2 || operation == 4) && comparison >= 0 {
			t.Fatal("global result is not the requested strict bound", operation, text, comparison)
		}
		if c.toOwnedRational.active || c.bounds.remaining != 0 {
			t.Fatal("successful bound wrapper retained local owners")
		}
	}
	var zero Algebraic
	serviceMust(t, c.GetLowerOwnedRationalAtPrecision(&zero, service, &out, 5))
	text, e := service.String(&out)
	serviceMust(t, e)
	if text != "0" {
		t.Fatal("precision basic dispatch", text)
	}
}

func TestAlgebraicOwnedBoundGlobalReservationPrefixes(t *testing.T) {
	for _, upper := range []bool{false, true} {
		success, failures := false, 0
		for fault := int64(0); fault < 256 && !success; fault++ {
			func() {
				c, _, _ := algebraicCompareFixture(t)
				service, budget := serviceNew(t)
				q := unumeral.NewRational()
				serviceMust(t, c.config.Rationals.SetString(&q, "18446744073709551617/18446744073709551619"))
				var input Algebraic
				serviceMust(t, c.SetRational(&input, &q))
				out, e := service.NewValue()
				serviceMust(t, e)
				budget.FailAfter(fault)
				if upper {
					e = c.GetUpperOwnedRationalAtPrecision(&input, service, &out, 2)
				} else {
					e = c.GetLowerOwnedRationalAtPrecision(&input, service, &out, 2)
				}
				budget.FailAfter(-1)
				if e != nil {
					failures++
					if !numericStorageReservationRefusal(e) {
						t.Fatal("unexpected global bound failure", fault, e)
					}
				}
				serviceMust(t, c.CleanupToOwnedRational())
				if e == nil {
					success = true
					text, e := service.String(&out)
					serviceMust(t, e)
					if text != "18446744073709551617/18446744073709551619" {
						t.Fatal("global basic bound", text)
					}
				}
				serviceMust(t, service.Del(&out))
				serviceMust(t, c.Del(&input))
				serviceMust(t, c.config.Rationals.Del(&q))
			}()
		}
		if !success || failures == 0 {
			t.Fatal("global bound campaign did not cover refusal and success", upper, failures)
		}
	}
}

func TestAlgebraicOwnedBoundRefusalPreservesOlderRawBound(t *testing.T) {
	c, _ := algebraicCellSourceFixtureProfile(t, true)
	service, _ := serviceNew(t)
	root := algebraicTinyBinaryBoundsRoot(t, c)
	q := c.config.Rationals
	raw := unumeral.NewRational()
	out, e := service.NewValue()
	serviceMust(t, e)
	pad := algebraicCellSourceTLSBoundary(t, c)
	body := c.GetUpperRational(&root, &raw)
	runtime := q.Integers().RuntimeOwner()
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	if body == nil || !c.bounds.conversion.pending(c) {
		t.Fatal("did not reach actual older raw bound stop", body)
	}
	invocation, remaining := c.bounds.invocation, c.bounds.remaining
	e = c.GetUpperOwnedRational(&root, service, &out)
	if !errors.Is(e, ErrAlgebraicPending) || c.toOwnedRational.boundInvocation != 0 {
		t.Fatal("refused wrapper adopted older raw call", e)
	}
	serviceMust(t, c.CleanupToOwnedRational())
	if c.bounds.invocation != invocation || c.bounds.remaining != remaining || !c.bounds.conversion.pending(c) {
		t.Fatal("wrapper cleanup consumed independent raw bound")
	}
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, q.Del(&raw))
	serviceMust(t, service.Del(&out))
	serviceMust(t, c.Del(&root))
}

func TestAlgebraicOwnedBoundCleansAcceptedRawStop(t *testing.T) {
	c, _ := algebraicCellSourceFixtureProfile(t, true)
	service, _ := serviceNew(t)
	root := algebraicTinyBinaryBoundsRoot(t, c)
	out, e := service.NewValue()
	serviceMust(t, e)
	serviceMust(t, service.SetInt64(&out, 77))
	pad := algebraicCellSourceTLSBoundary(t, c)
	body := c.GetUpperOwnedRational(&root, service, &out)
	if body == nil || c.toOwnedRational.boundInvocation == 0 || c.toOwnedRational.boundInvocation != c.bounds.invocation || !c.bounds.conversion.pending(c) {
		t.Fatal("wrapper did not retain accepted raw power stop", body)
	}
	runtime := c.config.Rationals.Integers().RuntimeOwner()
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	serviceMust(t, c.CleanupToOwnedRational())
	if c.toOwnedRational.active || c.bounds.conversion.pending(c) || c.bounds.remaining != 0 {
		t.Fatal("owning wrapper cleanup stranded its accepted raw child")
	}
	text, e := service.String(&out)
	serviceMust(t, e)
	if text != "77" {
		t.Fatal("failed raw child changed caller global value", text)
	}
	serviceMust(t, c.GetUpperOwnedRational(&root, service, &out))
	serviceMust(t, service.Del(&out))
	serviceMust(t, c.Del(&root))
}

func TestAlgebraicOwnedBoundRejectsReplacementRawCall(t *testing.T) {
	c, _ := algebraicCellSourceFixtureProfile(t, true)
	service, budget := serviceNew(t)
	root := algebraicTinyBinaryBoundsRoot(t, c)
	out, e := service.NewValue()
	serviceMust(t, e)
	budget.FailAfter(0)
	body := c.GetUpperOwnedRational(&root, service, &out)
	budget.FailAfter(-1)
	if body == nil || !c.toOwnedRational.active || c.toOwnedRational.boundInvocation == 0 {
		t.Fatal("did not stop global result construction after raw bound", body)
	}
	old := c.toOwnedRational.boundInvocation
	raw := unumeral.NewRational()
	body = c.GetUpperRational(&root, &raw)
	if !errors.Is(body, ErrAlgebraicPending) || c.bounds.invocation != old || c.bounds.conversion.pending(c) {
		t.Fatal("stopped owning wrapper permitted raw replacement", body)
	}
	serviceMust(t, c.CleanupToOwnedRational())
	serviceMust(t, c.GetUpperRational(&root, &raw))
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, c.config.Rationals.Del(&raw))
	serviceMust(t, service.Del(&out))
	serviceMust(t, c.Del(&root))
}
