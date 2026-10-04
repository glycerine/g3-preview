// Licensed under the MIT license.
// Added finite source-action fixtures: no registered direct mpzzp test exists.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func modularManager(t *testing.T, p uint64) (*IntegerModularManager, *IntegerManager, *arena.Budget) {
	t.Helper()
	z, b := integerManager(t)
	m, e := NewIntegerModularManager(z)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := m.Close(); e != nil {
			t.Error(e)
		}
	})
	if p != 0 {
		if e = m.SetZPUint64(p); e != nil {
			t.Fatal(e)
		}
	}
	return m, z, b
}
func modularValue(t *testing.T, z *IntegerManager, v *Int, want int64) {
	t.Helper()
	got, e := z.Int64(v)
	if e != nil || got != want {
		t.Fatalf("value %d/%v want %d", got, e, want)
	}
}
func centered(v, p int64) int64 {
	v %= p
	u := p / 2
	l := -u
	if p%2 == 0 {
		l++
	}
	if v > u {
		v -= p
	} else if v < l {
		v += p
	}
	return v
}
func TestIntegerModularCenteredOracle(t *testing.T) {
	for _, p := range []uint64{2, 3, 4, 5, 6, 7, 13} {
		t.Run(string(rune('A'+p)), func(t *testing.T) {
			m, z, _ := modularManager(t, p)
			for i := int64(-30); i <= 30; i++ {
				var a Int
				if e := m.SetInt64(&a, i); e != nil {
					t.Fatal(e)
				}
				modularValue(t, z, &a, centered(i, int64(p)))
				ok, e := m.IsPNormalized(&a)
				if e != nil || !ok {
					t.Fatal(ok, e)
				}
				for j := int64(-8); j <= 8; j++ {
					var b, c Int
					m.SetInt64(&b, j)
					if e := m.Add(&a, &b, &c); e != nil {
						t.Fatal(e)
					}
					modularValue(t, z, &c, centered(i+j, int64(p)))
					if e := m.Sub(&a, &b, &c); e != nil {
						t.Fatal(e)
					}
					modularValue(t, z, &c, centered(i-j, int64(p)))
					if e := m.Mul(&a, &b, &c); e != nil {
						t.Fatal(e)
					}
					modularValue(t, z, &c, centered(i*j, int64(p)))
					m.Del(&b)
					m.Del(&c)
				}
				m.Del(&a)
			}
		})
	}
}
func TestIntegerModularModeAndUnknownPrime(t *testing.T) {
	z, b := integerManager(t)
	before := b.Used()
	m, e := NewIntegerModularManagerWithUint64(z, 7, false)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = m.Field(); !errors.Is(e, ErrModularPrimeUninitialized) {
		t.Fatal(e)
	}
	a, c := NewInt(2), NewInt(0)
	if e = m.Add(&a, &a, &c); e != nil {
		t.Fatal(e)
	}
	modularValue(t, z, &c, -3)
	e = m.Div(&a, &a, &c)
	if modularSourceDebug {
		if !errors.Is(e, ErrModularPrimeUninitialized) {
			t.Fatal(e)
		}
	} else {
		if e != nil {
			t.Fatal(e)
		}
		modularValue(t, z, &c, 1)
	}
	if e = m.SetZPUint64(7); e != nil {
		t.Fatal(e)
	}
	if f, e := m.Field(); e != nil || !f {
		t.Fatal(f, e)
	}
	if e = m.SetPSquare(); e != nil {
		t.Fatal(e)
	}
	if f, e := m.Field(); e != nil || f {
		t.Fatal(f, e)
	}
	if e = m.SetZ(); e != nil {
		t.Fatal(e)
	}
	if f, e := m.Field(); e != nil || f {
		t.Fatal(f, e)
	}
	if e = m.Close(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != before {
		t.Fatal("header refund", b.Used(), before)
	}
	if e = m.Close(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
func TestIntegerModularInverseAndPowerActions(t *testing.T) {
	m, z, _ := modularManager(t, 7)
	for _, fixture := range [][2]int32{{1, 1}, {2, -3}, {3, -2}, {-1, -1}, {-2, 3}, {-3, 2}} {
		a := NewInt(fixture[0])
		if e := m.Inv(&a); e != nil {
			t.Fatal(e)
		}
		modularValue(t, z, &a, int64(fixture[1]))
		m.Del(&a)
	}
	for _, k := range []uint32{0, 1, 2, 3, 9, 31} {
		a := NewInt(3)
		var c Int
		v := int64(1)
		for i := uint32(0); i < k; i++ {
			v = centered(v*3, 7)
		}
		if e := m.Power(&a, k, &c); e != nil {
			t.Fatal(e)
		}
		modularValue(t, z, &c, v)
		m.Del(&c)
	}
	a, c := NewInt(1), NewInt(99)
	if e := m.Power(&a, 0x80000000, &c); !errors.Is(e, ErrNonterminatingPower) {
		t.Fatal(e)
	}
	modularValue(t, z, &c, 1)
	// Source composite inverse swaps before its final gcd assertion.
	m.SetZPUint64(6)
	a = NewInt(2)
	e := m.Inv(&a)
	if modularSourceDebug {
		if !errors.Is(e, ErrModularAssertion) {
			t.Fatal(e)
		}
	} else if e != nil {
		t.Fatal(e)
	}
	modularValue(t, z, &a, 1)
}
func TestIntegerModularMutableConversionAndAliases(t *testing.T) {
	m, z, _ := modularManager(t, 7)
	a := NewInt(8)
	if v, e := m.Int64(&a); e != nil || v != 8 {
		t.Fatal(v, e)
	}
	if v, e := m.Int64Mutable(&a); e != nil || v != 1 {
		t.Fatal(v, e)
	}
	modularValue(t, z, &a, 1)
	b, c := NewInt(2), NewInt(3)
	if e := m.AddMul(&a, &b, &c, &b); e != nil {
		t.Fatal(e)
	}
	modularValue(t, z, &b, 0)
	b = NewInt(2)
	if e := m.SubMul(&a, &b, &c, &a); e != nil {
		t.Fatal(e)
	}
	modularValue(t, z, &a, 2)
	m.SetZ()
	a = NewInt(-20)
	b = NewInt(6)
	if e := m.Div(&a, &b, &a); e != nil {
		t.Fatal(e)
	}
	modularValue(t, z, &a, -4)
}
func TestIntegerModularHeaderRefusalAndWarm(t *testing.T) {
	z, b := integerManager(t)
	before := b.Used()
	b.FailAfter(0)
	if _, e := NewIntegerModularManager(z); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if b.Used() != before {
		t.Fatal("reservation failure mutated budget")
	}
	b.FailAfter(-1)
	m, e := NewIntegerModularManager(z)
	if e != nil {
		t.Fatal(e)
	}
	defer m.Close()
	m.SetZPUint64(7)
	a, c := NewInt(2), NewInt(3)
	m.Mul(&a, &c, &c)
	b.FailAfter(0)
	if n := testing.AllocsPerRun(100, func() {
		if e := m.Add(&a, &a, &c); e != nil {
			panic(e)
		}
	}); n != 0 {
		t.Fatal(n)
	}
	b.FailAfter(-1)
}

func TestIntegerModularCompleteScalarRoutes(t *testing.T) {
	m, z, _ := modularManager(t, 7)
	a, b, c := NewInt(-2), NewInt(3), NewInt(0)
	for _, pred := range []struct {
		f    func(*Int) (bool, error)
		want bool
	}{{m.IsZero, false}, {m.IsOne, false}, {m.IsPos, false}, {m.IsNeg, true}, {m.IsNonPos, true}, {m.IsNonNeg, false}, {m.IsMinusOne, false}, {m.IsSmall, true}} {
		got, e := pred.f(&a)
		if e != nil || got != pred.want {
			t.Fatal(got, e)
		}
	}
	for _, pred := range []struct {
		f    func(*Int, *Int) (bool, error)
		want bool
	}{{m.Equal, false}, {m.Less, true}, {m.LessEqual, true}, {m.Greater, false}, {m.GreaterEqual, false}} {
		got, e := pred.f(&a, &b)
		if e != nil || got != pred.want {
			t.Fatal(got, e)
		}
	}
	if e := m.Neg(&a); e != nil {
		t.Fatal(e)
	}
	modularValue(t, z, &a, 2)
	m.Dec(&a)
	m.Inc(&a)
	m.Abs(&a)
	modularValue(t, z, &a, 2)
	if e := m.Swap(&a, &b); e != nil {
		t.Fatal(e)
	}
	modularValue(t, z, &a, 3)
	if e := m.Copy(&c, &a); e != nil {
		t.Fatal(e)
	}
	if h, e := m.Hash(&c); e != nil {
		t.Fatal(e)
	} else if want, e := z.Hash(&c); e != nil || h != want {
		t.Fatal(h, want, e)
	}
	if e := m.SetUint32(&c, 8); e != nil {
		t.Fatal(e)
	}
	modularValue(t, z, &c, 1)
	if e := m.SetUint64(&c, 9); e != nil {
		t.Fatal(e)
	}
	modularValue(t, z, &c, 2)
	if e := m.SetString(&c, "-12"); e != nil {
		t.Fatal(e)
	}
	modularValue(t, z, &c, 2)
	var mod Int
	if e := m.Modulus(&mod); e != nil {
		t.Fatal(e)
	}
	modularValue(t, z, &mod, 7)
	raw, e := m.borrowedModulus()
	if e != nil || raw != &m.p {
		t.Fatal(e)
	}
	next := NewInt(5)
	if e = m.SetZPSwap(&next); e != nil {
		t.Fatal(e)
	}
	modularValue(t, z, &next, 7)
	if e = m.SetZP(&mod); e != nil {
		t.Fatal(e)
	}
	if ok, e := m.Divides(&a, &b); e != nil || !ok {
		t.Fatal(ok, e)
	}
	if _, e = m.IsPerfectSquare(&a, &c); !errors.Is(e, ErrModularUnsupported) {
		t.Fatal(e)
	}
	if _, e = m.Root(&a, 2, &c); modularSourceDebug && !errors.Is(e, ErrModularAssertion) {
		t.Fatal(e)
	}
	if e = m.SetZ(); e != nil {
		t.Fatal(e)
	}
	c = NewInt(16)
	if v, e := m.PowerOfTwoMultiple(&c); e != nil || v != 4 {
		t.Fatal(v, e)
	}
	if v, e := m.Log2(&c); e != nil || v != 4 {
		t.Fatal(v, e)
	}
	if v, e := m.MLog2(&c); e != nil || v != 0 {
		t.Fatal(v, e)
	}
	if e = m.Mul2k(&c, 2, &c); e != nil {
		t.Fatal(e)
	}
	if e = m.MachineDiv2k(&c, 2, &c); e != nil {
		t.Fatal(e)
	}
	if ok, e := m.RootInPlace(&c, 2); e != nil || !ok {
		t.Fatal(ok, e)
	}
	modularValue(t, z, &c, 4)
	if ok, e := m.IsPerfectSquare(&c, &b); e != nil || !ok {
		t.Fatal(ok, e)
	}
	modularValue(t, z, &b, 2)
	a, b = NewInt(12), NewInt(18)
	if e = m.GCD(&a, &b, &c); e != nil {
		t.Fatal(e)
	}
	modularValue(t, z, &c, 6)
	if e = m.GCDMany([]Int{NewInt(12), NewInt(18), NewInt(9)}, &c); e != nil {
		t.Fatal(e)
	}
	modularValue(t, z, &c, 3)
	var x, y, g Int
	if e = m.ExtendedGCD(&a, &b, &x, &y, &g); e != nil {
		t.Fatal(e)
	}
	modularValue(t, z, &x, -1)
	modularValue(t, z, &y, 1)
	modularValue(t, z, &g, 6)
	if e = m.Reset(&c); e != nil {
		t.Fatal(e)
	}
	modularValue(t, z, &c, 0)
	for _, v := range []*Int{&a, &b, &c, &mod, &next, &x, &y, &g} {
		m.Del(v)
	}
}
func TestIntegerModularConversionOverloadsAndDisplay(t *testing.T) {
	m, z, _ := modularManager(t, 7)
	a := NewInt(8)
	if v, e := m.IsUint64(&a); e != nil || !v {
		t.Fatal(v, e)
	}
	if v, e := m.Uint64(&a); e != nil || v != 8 {
		t.Fatal(v, e)
	}
	if v, e := m.IsInt64(&a); e != nil || !v {
		t.Fatal(v, e)
	}
	if v, e := m.Int64(&a); e != nil || v != 8 {
		t.Fatal(v, e)
	}
	var out [16]byte
	n, e := m.FormatInto(&a, out[:])
	if e != nil || string(out[:n]) != "8" {
		t.Fatal(n, e)
	}
	if _, e = m.String(&a); modularSourceDebug && !errors.Is(e, ErrModularAssertion) {
		t.Fatal(e)
	}
	if v, e := m.IsUint64Mutable(&a); e != nil || !v {
		t.Fatal(v, e)
	}
	modularValue(t, z, &a, 1)
	a = NewInt(9)
	if v, e := m.IsInt64Mutable(&a); e != nil || !v {
		t.Fatal(v, e)
	}
	modularValue(t, z, &a, 2)
	a = NewInt(10)
	if v, e := m.Uint64Mutable(&a); e != nil || v != 3 {
		t.Fatal(v, e)
	}
	a = NewInt(11)
	if v, e := m.Float64Mutable(&a); e != nil || v != -3 {
		t.Fatal(v, e)
	}
	if s, e := m.String(&a); e != nil || s != "-3" {
		t.Fatal(s, e)
	}
	if z2, e := m.Manager(); e != nil || z2 != z {
		t.Fatal(e)
	}
	if !m.Precise() {
		t.Fatal("not precise")
	}
	if v, e := m.Finite(); e != nil || !v {
		t.Fatal(v, e)
	}
	if v, e := m.Modular(); e != nil || !v {
		t.Fatal(v, e)
	}
}

// Reuse the independently enumerated numeric ownership auditor with no
// rational scratch roots: this record is only an audit context, no callbacks.
func modularAudit(t *testing.T, m *IntegerModularManager, public []*Int) {
	roots := append([]*Int{&m.p, &m.lower, &m.upper, &m.invTmp1, &m.invTmp2, &m.invTmp3, &m.divTmp}, public...)
	qopsAuditLiveOwnership(t, &RationalManager{z: m.manager}, roots)
}
func TestIntegerModularPredictableFailurePrefixes(t *testing.T) {
	sawFinalSquare := false
	for _, op := range []string{"power", "inv", "div", "setp"} {
		for prefix := int64(0); prefix < 28; prefix++ {
			t.Run(op+string(rune('A'+prefix)), func(t *testing.T) {
				z, budget := integerManager(t)
				m, e := NewIntegerModularManager(z)
				if e != nil {
					t.Fatal(e)
				}
				defer m.Close()
				var a, b, c Int
				z.SetString(&a, "340282366920938463463374607431768211457")
				z.SetString(&b, "18446744073709551629")
				if op != "power" {
					if e = m.SetZP(&b); e != nil {
						t.Fatal(e)
					}
				}
				if op == "inv" || op == "div" {
					if e = m.PNormalize(&a); e != nil {
						t.Fatal(e)
					}
				}
				budget.FailAfter(prefix)
				switch op {
				case "power":
					e = m.Power(&a, 1, &c)
				case "inv":
					e = m.Inv(&a)
				case "div":
					e = m.Div(&a, &a, &c)
				case "setp":
					e = m.SetZP(&a)
				}
				// Audit before formatting, deleting any caller result or closing an owner.
				modularAudit(t, m, []*Int{&a, &b, &c})
				if e != nil && !numericPhysicalReservationRefusal(e) && !errors.Is(e, arena.ErrBudget) {
					t.Fatal(e)
				}
				if op == "power" && e != nil {
					equal, x := z.Equal(&a, &c)
					if x != nil {
						t.Fatal(x)
					}
					if equal {
						sawFinalSquare = true
					}
				}
				budget.FailAfter(-1)
				z.Del(&a)
				z.Del(&b)
				z.Del(&c)
			})
		}
	}
	if !sawFinalSquare {
		t.Fatal("did not exercise source final unused square failure after output publication")
	}
}
func TestIntegerModularReleaseSetupPreconditions(t *testing.T) {
	m, z, _ := modularManager(t, 0)
	e := m.SetZPUint64(1)
	if modularSourceDebug {
		if !errors.Is(e, ErrModularAssertion) {
			t.Fatal(e)
		}
	} else {
		if e != nil {
			t.Fatal(e)
		}
		a := NewInt(99)
		if e = m.PNormalize(&a); e != nil {
			t.Fatal(e)
		}
		modularValue(t, z, &a, 0)
	}
	m.SetZ()
	p := NewInt(7)
	e = m.SetZPSwap(&p)
	if modularSourceDebug {
		if !errors.Is(e, ErrModularAssertion) {
			t.Fatal(e)
		}
	} else {
		if e != nil {
			t.Fatal(e)
		}
		modularValue(t, z, &m.p, 7)
		modularValue(t, z, &p, 1)
	}
	m.SetZ()
	e = m.SetPSquare()
	if modularSourceDebug {
		if !errors.Is(e, ErrModularAssertion) {
			t.Fatal(e)
		}
	} else {
		if e != nil {
			t.Fatal(e)
		}
		modularValue(t, z, &m.p, 49)
	}
}
func TestIntegerModularFiniteSourceQuirks(t *testing.T) {
	m, z, _ := modularManager(t, 0)
	a, c := NewInt(0), NewInt(99)
	if e := m.Power(&a, 0, &c); e != nil {
		t.Fatal(e)
	}
	modularValue(t, z, &c, 1) // source loop, unlike MPZ power precondition
	a = NewInt(4)
	if ok, e := m.IsPerfectSquare(&a, &a); e != nil || !ok {
		t.Fatal(ok, e)
	}
	modularValue(t, z, &a, 0) // delegated source output/input alias
	p := NewInt(-7)
	e := m.SetZP(&p)
	if modularSourceDebug {
		if !errors.Is(e, ErrModularAssertion) {
			t.Fatal(e)
		}
	} else {
		if e != nil {
			t.Fatal(e)
		}
		modularValue(t, z, &m.upper, -4)
		modularValue(t, z, &m.lower, 4)
		a = NewInt(-6)
		if e = m.PNormalize(&a); e != nil {
			t.Fatal(e)
		}
		modularValue(t, z, &a, -13)
	}
	if e = m.SetZPUint64(7); e != nil {
		t.Fatal(e)
	}
	a = NewInt(0)
	e = m.Inv(&a)
	if modularSourceDebug {
		if !errors.Is(e, ErrModularAssertion) {
			t.Fatal(e)
		}
	} else {
		if e != nil {
			t.Fatal(e)
		}
		modularValue(t, z, &a, 0)
	}
}
func TestIntegerModularExplicitInPlaceOverloads(t *testing.T) {
	m, z, _ := modularManager(t, 7)
	a := NewInt(2)
	if e := m.Mul2kInPlace(&a, 2); e != nil {
		t.Fatal(e)
	}
	modularValue(t, z, &a, 1)
	if e := m.MachineDiv2kInPlace(&a, 1); e != nil {
		t.Fatal(e)
	}
	modularValue(t, z, &a, 0)
}
func TestIntegerModularColdOwnerIdentity(t *testing.T) {
	var zero IntegerModularManager
	panicked := false
	func() {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		if _, e := zero.Field(); e == nil {
			t.Error("zero owner accepted")
		}
	}()
	if panicked {
		t.Error("zero owner Field panicked")
	}
	panicked = false
	func() {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		if e := zero.Close(); e == nil {
			t.Error("zero Close accepted")
		}
	}()
	if panicked {
		t.Error("zero Close panicked")
	}
	z, b := integerManager(t)
	m, e := NewIntegerModularManager(z)
	if e != nil {
		t.Fatal(e)
	}
	defer m.Close()
	copy := *m
	before := b.Used()
	e = copy.Close()
	if e == nil {
		t.Error("copied Close accepted")
		if b.Used() < before {
			if x := b.Reserve(before - b.Used()); x != nil {
				t.Fatal(x)
			}
		}
	}
	if b.Used() != before {
		t.Fatal("copy changed owner budget")
	}
	if e = copy.SetZ(); e == nil {
		t.Error("copied method accepted")
	}
	var nilOwner *IntegerModularManager
	if _, e = nilOwner.Field(); e == nil {
		t.Fatal("nil accepted")
	}
}
func TestIntegerModularConstructorOwnershipAndRefusal(t *testing.T) {
	for prefix := int64(0); prefix < 16; prefix++ {
		t.Run(string(rune('A'+prefix)), func(t *testing.T) {
			z, b := integerManager(t)
			var p Int
			if e := z.SetString(&p, "340282366920938463463374607431768211457"); e != nil {
				t.Fatal(e)
			}
			b.FailAfter(prefix)
			m, e := NewIntegerModularManagerWithModulus(z, &p, true)
			if e != nil {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal(e)
				}
				qopsAuditLiveOwnership(t, &RationalManager{z: z}, []*Int{&p})
			} else {
				modularAudit(t, m, []*Int{&p})
				if _, e = m.Field(); !errors.Is(e, ErrModularPrimeUninitialized) {
					t.Fatal(e)
				}
				b.FailAfter(-1)
				if e = z.SetInt32(&p, 3); e != nil {
					t.Fatal(e)
				}
				if m.p.IsSmall() {
					t.Fatal("modulus shares caller ownership")
				}
				if e = m.Close(); e != nil {
					t.Fatal(e)
				}
			}
			b.FailAfter(-1)
			z.Del(&p)
		})
	}
	z, b := integerManager(t)
	before := b.Used()
	b.FailAfter(0)
	if n := testing.AllocsPerRun(100, func() {
		if m, e := NewIntegerModularManager(z); m != nil || !numericPhysicalReservationRefusal(e) {
			panic("refusal")
		}
	}); n != 0 {
		t.Fatal("allocation before header reservation", n)
	}
	if b.Used() != before {
		t.Fatal("header refusal retained bytes")
	}
	b.FailAfter(-1)
}
