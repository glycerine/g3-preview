package polynomial

import (
	"fmt"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicManagerPrivateUnsynchronizedNumeralOwners(t *testing.T) {
	cfg, _, budget := algebraicManagerFixture(t)
	var owners [2]*unumeral.RationalManager
	var managers [2]*AlgebraicManager
	for i := range owners {
		q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: serviceConfig(budget).Integers})
		serviceMust(t, e)
		owners[i] = q
		t.Cleanup(func() { serviceMust(t, q.CloseChecked()) })
		cfg.Numerals = q
		m, e := NewAlgebraicManager(cfg)
		serviceMust(t, e)
		managers[i] = m
		t.Cleanup(func() { serviceMust(t, m.Close()) })
		actual, e := m.RationalManager()
		serviceMust(t, e)
		if actual != q || q.Synchronized || q == cfg.Rationals.Manager || m.binary.Integers() != q.Integers() || m.binary.Diagnostics != cfg.Rationals || m.polynomial.core.config.Rationals != q || m.univariate.config.Integer != q.Integers() {
			t.Fatal("private numeral owner was replaced by global diagnostics owner")
		}
	}
	source, target := managers[0], managers[1]
	var a, b, square Algebraic
	t.Cleanup(func() {
		serviceMust(t, target.CleanupRetained())
		serviceMust(t, target.CleanupValue(&square))
		serviceMust(t, target.CleanupValue(&b))
		if a.slot != 0 {
			serviceMust(t, source.CleanupValue(&a))
		}
	})
	two := unumeral.NewInt(2)
	serviceMust(t, source.SetInteger(&a, &two))
	serviceMust(t, source.Root(&a, 2, &a))
	serviceMust(t, target.CopyFromCore(&b, source.AlgebraicRootCore, &a))
	serviceMust(t, source.Del(&a))
	serviceMust(t, source.Close())
	serviceMust(t, owners[0].CloseChecked())
	serviceMust(t, target.Power(&b, 2, &square))
	want := unumeral.NewRational()
	serviceMust(t, owners[1].SetInt32(&want, 2))
	cmp, e := target.CompareRational(&square, &want)
	serviceMust(t, e)
	serviceMust(t, owners[1].Del(&want))
	if cmp != 0 {
		t.Fatal("private-owner copied square root did not square to two", cmp)
	}
	serviceMust(t, cfg.Rationals.Manager.CheckOwnerReady())
}

func TestAlgebraicManagerPrivateNumeralConstructorReservations(t *testing.T) {
	success, failures := false, 0
	for prefix := int64(0); prefix < 512 && !success; prefix++ {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			cfg, _, budget := algebraicManagerFixture(t)
			q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: serviceConfig(budget).Integers})
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, q.CloseChecked()) })
			cfg.Numerals = q
			budget.FailAfter(prefix)
			m, body := NewAlgebraicManager(cfg)
			budget.FailAfter(-1)
			if body != nil {
				failures++
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("unexpected private constructor refusal", body)
				}
			} else {
				success = true
			}
			if m != nil {
				serviceMust(t, m.Close())
			}
			serviceMust(t, q.CheckOwnerReady())
			serviceMust(t, cfg.Rationals.Manager.CheckOwnerReady())
		})
	}
	if !success || failures == 0 {
		t.Fatal("private constructor reservation coverage", failures, success)
	}
}
