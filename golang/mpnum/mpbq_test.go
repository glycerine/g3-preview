// Licensed under the MIT license.
// Source: src/test/mpbq.cpp.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func binaryManager(t *testing.T) (*BinaryRationalManager, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 26)
	s, e := NewRationalService(serviceConfig(b))
	serviceMust(t, e)
	z, e := NewIntegerManager(serviceConfig(b).Integers)
	serviceMust(t, e)
	m, e := NewBinaryRationalManager(z, BinaryRationalConfig{Diagnostics: s})
	serviceMust(t, e)
	t.Cleanup(func() {
		m.Close()
		z.Close()
		serviceMust(t, s.Close())
		if b.Used() != 0 {
			t.Errorf("retained %d", b.Used())
		}
	})
	return m, b
}
func binaryText(t *testing.T, m *BinaryRationalManager, a *BinaryRational, want string) {
	t.Helper()
	got, e := m.String(a)
	if e != nil || got != want {
		t.Fatalf("got %q %v want %q", got, e, want)
	}
}

// Source has zero ENSURE assertions: these six expectations are independent
// regressions for its actual print events. Each expression temporary is closed
// at its full-expression boundary, and named a/b at the source brace boundary.
func TestBinarySourceTst1(t *testing.T) {
	m, _ := binaryManager(t)
	a, e := NewScopedBinary(m)
	serviceMust(t, e)
	b, e := NewScopedBinary(m)
	serviceMust(t, e)
	defer a.Close()
	defer b.Close()
	serviceMust(t, a.SetInt32(2147483647))
	x, e := a.AddInt(1)
	serviceMust(t, e)
	serviceMust(t, a.Assign(x))
	serviceMust(t, x.Close())
	x, e = a.Mul(a)
	serviceMust(t, e)
	serviceMust(t, a.Assign(x))
	serviceMust(t, x.Close())
	x, e = a.MulInt(3)
	serviceMust(t, e)
	y, e := x.SubInt(1)
	serviceMust(t, e)
	serviceMust(t, a.Assign(y))
	serviceMust(t, y.Close())
	serviceMust(t, x.Close())
	for _, n := range []int32{5, 7} {
		x, e = a.Mul(a)
		serviceMust(t, e)
		y, e = x.SubInt(n)
		serviceMust(t, e)
		serviceMust(t, a.Assign(y))
		serviceMust(t, y.Close())
		serviceMust(t, x.Close())
	}
	av, e := a.Get()
	serviceMust(t, e)
	bv, e := b.Get()
	serviceMust(t, e)
	serviceMust(t, m.Div2k(av, 67))
	binaryText(t, m, av, "36637340735244577447646647796940134554460176108377786095865669439633741578249/2^67")
	wants := []string{"1066286953144141515008134528672675792897940150822331957049483591681/2^32", "496528555240547989964547813251929261208484019535748268035/2", "-496528555240547989964547813251929261208484019535748268035/2", "-1066286953144141515008134528672675792897940150822331957049483591681/2^32"}
	for i, w := range wants {
		serviceMust(t, b.Assign(a))
		if i >= 2 {
			serviceMust(t, m.Neg(bv))
		}
		serviceMust(t, m.Approx(bv, 32, i%2 == 0))
		binaryText(t, m, bv, w)
	}
}
func TestBinarySourceTst2(t *testing.T) {
	m, _ := binaryManager(t)
	a, e := NewScopedBinary(m)
	serviceMust(t, e)
	b, e := NewScopedBinary(m)
	serviceMust(t, e)
	defer a.Close()
	defer b.Close()
	serviceMust(t, a.SetInt32(5))
	serviceMust(t, b.SetInt32(3))
	av, e := a.Get()
	serviceMust(t, e)
	bv, e := b.Get()
	serviceMust(t, e)
	serviceMust(t, m.ApproxDiv(av, bv, av, 128, false))
	binaryText(t, m, av, "283568639100782052886145506193140176213/2^127")
}
