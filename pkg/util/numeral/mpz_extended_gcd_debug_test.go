//go:build g3debug

package numeral

import (
	"math/big"
	"testing"
)

func TestExtendedGCDDebugSourceAssertion(t *testing.T) {
	owner, _ := rationalManager(t, true)
	m := owner.Z
	var r1, r2, a, b, g Int
	defer func() {
		for _, v := range []*Int{&r1, &r2, &a, &b, &g} {
			m.Del(v)
		}
	}()
	for x := int64(-12); x <= 12; x++ {
		for y := int64(-12); y <= 12; y++ {
			qopsMust(t, m.SetInt64(&r1, x))
			qopsMust(t, m.SetInt64(&r2, y))
			qopsMust(t, m.ExtendedGCD(&r1, &r2, &a, &b, &g))
			av, e := m.Int64(&a)
			qopsMust(t, e)
			bv, e := m.Int64(&b)
			qopsMust(t, e)
			gv, e := m.Int64(&g)
			qopsMust(t, e)
			if av*x+bv*y != gv {
				t.Fatal(x, y, av, bv, gv)
			}
		}
	}
	// The source rereads aliased input r1 after coefficient publication. Its
	// debug assertion fails before the final gcd assignment, unlike release.
	qopsMust(t, m.SetInt64(&r1, -26))
	qopsMust(t, m.SetInt64(&r2, 7))
	qopsMust(t, m.SetInt64(&g, 17))
	bindMPNDebugActions(t, "T")
	e := m.ExtendedGCD(&r1, &r2, &r1, &b, &g)
	qopsAuditLiveOwnership(t, owner, []*Int{&r1, &r2, &a, &b, &g})
	if !isMPNVerificationThrow(e) {
		t.Fatal(e)
	}
	if integerString(t, m, &r1) != "3" || integerString(t, m, &b) != "-11" || integerString(t, m, &g) != "17" {
		t.Fatal("source assertion publication prefix")
	}
}

func TestExtendedGCDDebugDiagnosticOrderAndFailures(t *testing.T) {
	factor := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 257), big.NewInt(19))
	x := new(big.Int).Mul(factor, big.NewInt(15))
	y := new(big.Int).Mul(factor, big.NewInt(25))
	want := new(big.Int).Mul(factor, big.NewInt(5))
	failures, successes := 0, 0
	for prefix := int64(0); prefix < 40; prefix++ {
		m, budget := rationalManager(t, true)
		var r1, r2, a, b, expected, nexta, nextb Int
		roots := []*Int{&r1, &r2, &a, &b, &expected, &nexta, &nextb}
		setIntegerOracle(t, m.Z, &r1, x)
		setIntegerOracle(t, m.Z, &r2, y)
		qopsMust(t, m.Z.SetInt64(&a, 2))
		qopsMust(t, m.Z.SetInt64(&b, -1))
		setIntegerOracle(t, m.Z, &expected, want)
		qopsMust(t, m.Z.SetInt64(&nexta, 31))
		qopsMust(t, m.Z.SetInt64(&nextb, 37))
		budget.FailAfter(prefix)
		e := m.Z.extendedGCDDiagnostic(&r1, &r2, &a, &b, &expected, &nexta, &nextb)
		qopsAuditLiveOwnership(t, m, roots)
		budget.FailAfter(-1)
		if e != nil {
			if !numericPhysicalReservationRefusal(e) {
				t.Fatal(prefix, e)
			}
			failures++
		} else {
			successes++
			if integerString(t, m.Z, &nexta) != want.String() || integerString(t, m.Z, &nextb) != new(big.Int).Neg(y).String() {
				t.Fatal("source mul/mul/add outputs")
			}
		}
		for _, v := range roots {
			qopsMust(t, m.Z.Del(v))
		}
		m.Close()
		if budget.Used() != 0 {
			t.Fatal(prefix, budget.Used())
		}
	}
	if failures == 0 || successes == 0 {
		t.Fatal(failures, successes)
	}
}
