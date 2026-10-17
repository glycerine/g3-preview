//go:build g3debug

package euclid

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestEuclideanAddmulAssertionStopsBeforeOutputReset(t *testing.T) {
	m := euclidNumeralManager(t)
	s, err := newSolverSetup(m)
	euclidMust(t, err)
	ints, err := numeral.NewScopedIntVector(m.Integers())
	euclidMust(t, err)
	rats, err := numeral.NewScopedRationalVector(m)
	euclidMust(t, err)
	seedInt, seedRat := numeral.NewInt(7), numeral.NewRational()
	euclidMust(t, ints.Push(&seedInt))
	euclidMust(t, rats.Push(&seedRat))
	intIDs, ratIDs := []uint32{91}, []uint32{92}
	budget := arena.NewBudget(1 << 20)
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	euclidMust(t, err)
	d, err := diagnostic.NewDebugContext(region, budget)
	euclidMust(t, err)
	old, err := diagnostic.BindProcessDebugContext(d)
	euclidMust(t, err)
	t.Cleanup(func() {
		_, e := diagnostic.BindProcessDebugContext(old)
		euclidMust(t, e)
		euclidMust(t, d.Close())
		euclidMust(t, region.Close())
		if budget.Used() != 0 {
			t.Fatal("borrowed addmul debug owner", budget.Used())
		}
	})
	euclidMust(t, d.Close())
	k := numeral.NewInt(1)
	err = s.addmulInt([]numeral.Int{numeral.NewInt(2)}, []uint32{0}, &k, nil, nil, ints, &intIDs, false, false, nullEquationIndex, nullEquationIndex)
	if !errors.Is(err, arena.ErrClosed) || ints.Size() != 1 || !equalUint32Slices(intIDs, []uint32{91}) {
		t.Fatal("integer addmul reset before source assertion", err, ints.Size(), intIDs)
	}
	err = s.addmulRational([]numeral.Rational{numeral.NewRational()}, []uint32{0}, &k, nil, nil, rats, &ratIDs)
	if !errors.Is(err, arena.ErrClosed) || rats.Size() != 1 || !equalUint32Slices(ratIDs, []uint32{92}) {
		t.Fatal("rational addmul reset before source assertion", err, rats.Size(), ratIDs)
	}
	_, err = diagnostic.BindProcessDebugContext(old)
	euclidMust(t, err)
	euclidMust(t, ints.Close())
	euclidMust(t, rats.Close())
	euclidMust(t, s.Close())
}
