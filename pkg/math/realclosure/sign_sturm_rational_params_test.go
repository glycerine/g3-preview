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
	if err := store.bindInitialPrecision(pc, &p, nil); err != nil {
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
	if err := store.bindInitialPrecision(pc, &p, nil); err != nil || !store.initialPrecisionBound || store.initialPrecision != 0 {
		t.Fatal("explicit zero precision was lost", err)
	}
	third, precise := measure()
	if precise {
		t.Fatal("explicit zero precision used the default")
	}
	module, err := pc.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	if err := pc.SetUIntName(&module, &name, 6); err != nil {
		t.Fatal(err)
	}
	if err := pc.Reset(&p); err != nil {
		t.Fatal(err)
	}
	if err := store.bindInitialPrecision(pc, &p, &module); err != nil || store.initialPrecision != 6 {
		t.Fatal("module fallback initial precision", err)
	}
	fourth, precise := measure()
	if precise {
		t.Fatal("module fallback unexpectedly used the default precision")
	}
	if err := pc.SetUIntName(&p, &name, 4); err != nil {
		t.Fatal(err)
	}
	if err := store.bindInitialPrecision(pc, &p, &module); err != nil || store.initialPrecision != 4 {
		t.Fatal("local precision did not override module fallback", err)
	}
	for _, value := range []*realValue{first, second, third, fourth} {
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
	if err := pc.Del(&module); err != nil {
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

func TestRealClosureInitialPrecisionAcquiresGlobalModuleRef(t *testing.T) {
	budget := arena.NewBudget(1 << 27)
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
	catalog, err := params.NewGlobalParamsCatalog(sy, params.GlobalParamsCatalogConfig{
		Budget:       budget,
		Descriptions: arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{Budget: budget}},
		Text:         arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	})
	if err != nil {
		t.Fatal(err)
	}
	globals, err := params.NewGlobalParamsService(sy, pc, params.GlobalParamsConfig{
		Budget: budget, Catalog: catalog, DescriptorPageSize: 2, DescriptorWordsPageSize: 16,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := globals.InitializeSource(); err != nil {
		t.Fatal(err)
	}
	local, err := pc.NewValue()
	if err != nil {
		t.Fatal(err)
	}
	moduleKey, err := sy.Intern("rcf")
	if err != nil {
		t.Fatal(err)
	}
	module, err := globals.Current.GetParams(moduleKey)
	if err != nil {
		t.Fatal(err)
	}
	name := "initial_precision"
	if err := pc.SetUIntName(module, &name, 6); err != nil {
		t.Fatal(err)
	}
	store := &realValueStore{numbers: service.Manager}
	if err := store.bindInitialPrecisionFromGlobals(pc, &local, globals); err != nil || store.initialPrecision != 6 || store.pendingPrecisionHelper != nil {
		t.Fatal("acquired module fallback", err, store.initialPrecision)
	}
	bounds, err := numeral.NewBinaryRationalManager(service.Manager.Z, numeral.BinaryRationalConfig{Diagnostics: service})
	if err != nil {
		t.Fatal(err)
	}
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
	if err != nil || precise {
		t.Fatal("global-module precision did not reach approximation", precise, err)
	}
	if err := bounds.Del(&width); err != nil {
		t.Fatal(err)
	}
	if err := pc.SetUIntName(&local, &name, 4); err != nil {
		t.Fatal(err)
	}
	if err := store.bindInitialPrecisionFromGlobals(pc, &local, globals); err != nil || store.initialPrecision != 4 || store.pendingPrecisionHelper != nil {
		t.Fatal("local override through acquired module", err, store.initialPrecision)
	}
	if err := store.cleanupInitialPrecisionHelper(); err != nil {
		t.Fatal(err)
	}
	if err := store.release(value); err != nil {
		t.Fatal(err)
	}
	if err := bounds.CloseChecked(); err != nil {
		t.Fatal(err)
	}
	if err := pc.Del(&local); err != nil {
		t.Fatal(err)
	}
	if err := globals.Close(); err != nil {
		t.Fatal(err)
	}
	if err := catalog.Close(); err != nil {
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
		t.Fatal("module helper retained bytes", used)
	}
}
