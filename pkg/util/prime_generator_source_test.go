// Tests FIRST: complete registered source prime campaign, no substitute generator.
package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	primegen "github.com/glycerine/g3/pkg/util/prime"
)

func TestPrimeGeneratorRegisteredTenThousandWithIntegerRoot(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	q, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{Cells: arena.Config{PageSize: 8, Budget: budget}, Words: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 8, Budget: budget}}}})
	if e != nil {
		t.Fatal(e)
	}
	gen, e := primegen.NewGenerator(budget)
	if e != nil {
		t.Fatal(e)
	}
	z := q.Integers()
	var sqrt Int
	assertions := uint64(0)
	var last uint64
	for idx := uint32(0); idx < 10000; idx++ {
		p, e := gen.At(idx)
		if e != nil {
			t.Fatal(idx, e)
		}
		last = p
		if p == 2 {
			continue
		}
		if e = z.SetUint64(&sqrt, p); e != nil {
			t.Fatal(e)
		}
		if _, e = z.RootInPlace(&sqrt, 2); e != nil {
			t.Fatal(e)
		}
		k, e := z.Uint64(&sqrt)
		if e != nil {
			t.Fatal(e)
		}
		for divisor := uint64(2); divisor <= k; divisor++ {
			assertions++
			if p%divisor == 0 {
				t.Fatal("source ENSURE", idx, p, divisor)
			}
		}
	}
	if last != 104729 || assertions != 2073824 {
		t.Fatal("independent frozen endpoints/count", last, assertions)
	}
	if e = z.Del(&sqrt); e != nil {
		t.Fatal(e)
	}
	if e = gen.Close(); e != nil {
		t.Fatal(e)
	}
	q.Close()
	if budget.Used() != 0 {
		t.Fatal("owned graph", budget.Used())
	}
	t.Log("source2073824ENSURE executions; printed stream/flush remainsOPEN")
}
