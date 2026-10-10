// Source: src/test/prime_generator.cpp:22-35.
package numeral

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	primegen "github.com/glycerine/g3/pkg/util/prime"
)

func TestDefaultPrimeRegisteredIntegerRootAssertions(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	q, err := NewRationalManager(RationalConfig{Integers: IntegerConfig{Cells: arena.Config{PageSize: 8, Budget: b}, Words: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 8, Budget: b}}}})
	serviceMust(t, err)
	service, err := primegen.NewDefaultService(b)
	serviceMust(t, err)
	iterator, err := primegen.NewServiceIterator(service, nil)
	serviceMust(t, err)
	z := q.Integers()
	var sqrt Int
	assertions := uint64(0)
	var last uint64
	for i := 0; i < 10000; i++ {
		p, err := iterator.Next()
		serviceMust(t, err)
		last = p
		if p == 2 {
			continue
		}
		serviceMust(t, z.SetUint64(&sqrt, p))
		_, err = z.RootInPlace(&sqrt, 2)
		serviceMust(t, err)
		k, err := z.Uint64(&sqrt)
		serviceMust(t, err)
		for d := uint64(2); d <= k; d++ {
			assertions++
			if p%d == 0 {
				t.Fatal("source ENSURE", i, p, d)
			}
		}
	}
	if last != 104729 || assertions != 2073824 {
		t.Fatal(last, assertions)
	}
	serviceMust(t, z.Del(&sqrt))
	serviceMust(t, service.Close())
	serviceMust(t, q.CloseChecked())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
