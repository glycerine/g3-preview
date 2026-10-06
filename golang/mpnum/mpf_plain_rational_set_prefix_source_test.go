// Source: src/util/mpf.cpp:186-192,233-300; src/util/mpq.cpp:78-86,312-315.
// Licensed under the MIT license. Source authors: Christoph Wintersteiger; Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestArbitraryFloatPlainRationalSetEveryActualPrefix(t *testing.T) {
	plainRationalSetEveryPrefix(t, 0)
}
func TestArbitraryFloatPlainRationalSetBIGBelowOnePrefix(t *testing.T) {
	plainRationalSetEveryPrefix(t, 2)
}
func plainRationalSetEveryPrefix(t *testing.T, route uint8) {
	success := false
	for ordinal := int64(0); ordinal < 600; ordinal++ {
		b := arena.NewBudget(1 << 28)
		m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
		serviceMust(t, e)
		q := NewRational()
		if route != 0 {
			n, d := make([]uint32, 21), make([]uint32, 21)
			n[0] = 1
			n[20] = 2
			d[0] = 3
			d[20] = 1
			if route == 2 {
				n, d = d, n
			}
			if route == 3 {
				n[0] = 5
				n[20] = 4
			}
			serviceMust(t, m.integer.SetLimbs(&q.num, n))
			serviceMust(t, m.integer.SetLimbs(&q.den, d))
		} else {
			serviceMust(t, m.rational.SetFraction(&q, 3, 2))
		}
		var out ArbitraryFloat
		width := uint32(4000)
		if route != 0 {
			width = 80
		}
		b.FailAfter(ordinal)
		body := m.SetRational(&out, 8, width, ArbitraryFloatNearestEven, &q)
		s := &m.rationalSet
		r := &m.roundScope
		integerPhysicalAudit(t, m.integer, &q.num, &q.den, &m.plainRationalSet.exp, &out.significand, &s.sig.num, &s.sig.den, &s.t.num, &s.t.den, &s.sq.num, &s.sq.den, &s.exp, &s.p, &s.p2, &s.prev, &m.rational.nTmp, &m.rational.addTmp1, &m.rational.addTmp2, &m.rational.ltTmp1.num, &m.rational.ltTmp1.den, &m.rational.ltTmp2.num, &m.rational.ltTmp2.den, &r.sigma, &r.cap, &r.sticky)
		if b.Used() != m.memory.Stats().TotalBytes+m.OwnedStorageBytes() {
			t.Fatal("immediate named source graph", ordinal, b.Used(), m.memory.Stats().TotalBytes, m.OwnedStorageBytes())
		}
		b.FailAfter(-1)
		if body == nil {
			if route != 0 {
				if out.significand != NewInt(0) || out.exponent != []int64{0, 1, -1, 2}[route] {
					t.Fatal("independent BIG input rounds to 2", out)
				}
			} else {
				power, e := m.integer.PrevPowerOfTwo(&out.significand)
				serviceMust(t, e)
				if power != 3998 || out.exponent != 0 {
					t.Fatal("literal successful result", power, out.exponent)
				}
			}
			success = true
		}
		serviceMust(t, m.CleanupPlainRationalSet())
		serviceMust(t, m.CleanupRound())
		serviceMust(t, m.Del(&out))
		serviceMust(t, m.rational.Del(&q))
		if e = m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
			t.Fatal("unclassified source destructor", ordinal, e)
		}
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal("own leak", ordinal, b.Used())
		}
		if success {
			break
		}
	}
	if !success {
		t.Fatal("no actual complete source call")
	}
}
