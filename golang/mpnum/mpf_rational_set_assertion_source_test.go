// Source: src/util/mpf.cpp:279; src/util/mpq.cpp:rat_lt.
// Licensed under the MIT license. Source authors: Christoph Wintersteiger; Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

// This is a represented helper-domain assertion prefix, not a reachable
// normalized public SET body failure. The deliberately below-one input makes
// the first conjunct false, so the upper comparison must remain unread.
func TestArbitraryFloatExponentRationalSetAssertionHelperPrefix(t *testing.T) {
	b := arena.NewBudget(1 << 25)
	m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	q := NewRational()
	q.num = NewInt(1)
	d := make([]uint32, 21)
	d[0], d[20] = 3, 1
	serviceMust(t, m.integer.SetLimbs(&q.den, d))
	if e = m.assertRationalSetNormalized(&q, integerAllocationInvocationContext{}); e != ErrIntegerAssertion {
		t.Fatal(e)
	}
	cmp, e := m.integer.Compare(&m.rational.ltTmp1.num, &q.den)
	serviceMust(t, e)
	if cmp != 0 || m.rational.ltTmp2.num != q.num {
		t.Fatal("source le(1,sig) prefix changed or upper conjunct read")
	}
	integerPhysicalAudit(t, m.integer, &q.num, &q.den, &m.rational.ltTmp1.num, &m.rational.ltTmp1.den, &m.rational.ltTmp2.num, &m.rational.ltTmp2.den)
	if b.Used() != m.memory.Stats().TotalBytes+m.OwnedStorageBytes() {
		t.Fatal("helper immediate cold graph")
	}
	serviceMust(t, m.rational.Del(&q))
	serviceMust(t, m.SourceFinalize())
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
