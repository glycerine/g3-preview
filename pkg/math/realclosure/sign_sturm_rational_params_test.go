package realclosure

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestRealClosureRationalIntervalReadsInitialPrecision(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	sy, err := symbol.NewManager(symbol.Config{Budget: budget})
	if err != nil {
		t.Fatal(err)
	}
	service, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	pc, err := params.NewParamsContext(sy, service, params.ParamsConfig{Budget: budget})
	if err != nil {
		t.Fatal(err)
	}
	p, err := pc.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	bounds, err := numeral.NewBinaryRationalManager(service.Manager.Z, numeral.BinaryRationalConfig{Diagnostics: service})
	if err != nil {
		t.Fatal(err)
	}
	store := &realValueStore{numbers: service.Manager}
	measure := func() (*realValue, bool) {
		t.Helper()
		value, err := store.newRational(1, 3)
		if err != nil {
			t.Fatal(err)
		}
		interval, err := store.cachedRationalApproxInterval(value, bounds)
		if err != nil {
			t.Fatal(err)
		}
		width := numeral.NewBinaryRational()
		if err := bounds.Sub(&interval.upper, &interval.lower, &width); err != nil {
			t.Fatal(err)
		}
		precise, err := bounds.LessOneDiv2k(&width, 24)
		if err != nil {
			t.Fatal(err)
		}
		if err := bounds.Del(&width); err != nil {
			t.Fatal(err)
		}
		return value, precise
	}
	first, precise := measure()
	if !precise {
		t.Fatal("source default initial precision is 24")
	}
	name := "initial_precision"
	if err := pc.SetUIntName(&p, &name, 4); err != nil {
		t.Fatal(err)
	}
	if err := store.bindInitialPrecision(pc, &p); err != nil {
		t.Fatal(err)
	}
	second, precise := measure()
	if precise || store.initialPrecision != 4 {
		t.Fatal("configured initial precision did not widen the new approximation")
	}
	if cached, err := store.cachedRationalApproxInterval(first, bounds); err != nil || cached != first.rationalInterval {
		t.Fatal("parameter update replaced an earlier lazy interval", err)
	}
	if err := pc.SetUIntName(&p, &name, 0); err != nil {
		t.Fatal(err)
	}
	if err := store.bindInitialPrecision(pc, &p); err != nil || !store.initialPrecisionBound || store.initialPrecision != 0 {
		t.Fatal("explicit zero precision was lost", err)
	}
	third, precise := measure()
	if precise {
		t.Fatal("explicit zero precision used the default")
	}
	for _, value := range []*realValue{first, second, third} {
		if err := store.release(value); err != nil {
			t.Fatal(err)
		}
	}
	if err := bounds.CloseChecked(); err != nil {
		t.Fatal(err)
	}
	if err := pc.Del(&p); err != nil {
		t.Fatal(err)
	}
	if err := pc.Close(); err != nil {
		t.Fatal(err)
	}
	if err := service.Close(); err != nil {
		t.Fatal(err)
	}
	if err := sy.Close(); err != nil {
		t.Fatal(err)
	}
	if used := budget.Used(); used != 0 {
		t.Fatal("precision owners retained bytes", used)
	}
}
