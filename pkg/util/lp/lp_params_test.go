package lp

// Source: src/util/lp/lp_params.hpp; retained getters in
// src/shell/lp_frontend.cpp and src/smt/theory_lra.cpp.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	uparams "github.com/glycerine/g3/pkg/util/params"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func lpParamsMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func lpParamsName(s string) *string { return &s }
func lpParamsSourceFixture(t *testing.T) (*uparams.ParamsContext, *uparams.GlobalParamsService, *uparams.ParamsRef, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(^uint64(0))
	sy, e := usymbol.NewManager(usymbol.Config{Budget: b, PageSize: 64, DescriptorPageSize: 2})
	lpParamsMust(t, e)
	q, e := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
	lpParamsMust(t, e)
	c, e := uparams.NewParamsContext(sy, q, uparams.ParamsConfig{Budget: b, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
	lpParamsMust(t, e)
	catalog, e := uparams.NewGlobalParamsCatalog(sy, uparams.GlobalParamsCatalogConfig{Budget: b, Descriptions: arena.BytesRegionConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Text: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
	lpParamsMust(t, e)
	g, e := uparams.NewGlobalParamsService(sy, c, uparams.GlobalParamsConfig{Budget: b, Catalog: catalog, DescriptorPageSize: 2, DescriptorWordsPageSize: 16})
	lpParamsMust(t, e)
	lpParamsMust(t, g.InitializeSource())
	local, e := c.NewValue()
	lpParamsMust(t, e)
	t.Cleanup(func() {
		b.FailAfter(-1)
		lpParamsMust(t, c.Del(&local))
		lpParamsMust(t, g.Close())
		lpParamsMust(t, catalog.Close())
		lpParamsMust(t, c.Close())
		lpParamsMust(t, q.Close())
		lpParamsMust(t, sy.Close())
		if b.Used() != 0 {
			t.Error("parameter owners retained", b.Used())
		}
	})
	return c, g, &local, b
}
func lpParamsModule(t *testing.T, g *uparams.GlobalParamsService) *uparams.ParamsRef {
	t.Helper()
	key, e := g.Symbols.Intern("lp")
	lpParamsMust(t, e)
	v, e := g.Current.GetParams(key)
	lpParamsMust(t, e)
	return v
}
func TestLPParamsFiveGettersFallbackAndLiveLocal(t *testing.T) {
	for _, key := range []string{"rep_freq", "min", "print_stats", "simplex_strategy", "bprop_on_pivoted_rows"} {
		for mode := 0; mode < 5; mode++ {
			t.Run(key+string(rune('0'+mode)), func(t *testing.T) {
				c, g, local, b := lpParamsSourceFixture(t)
				module := lpParamsModule(t, g)
				isUInt := key == "rep_freq" || key == "simplex_strategy"
				def := key == "bprop_on_pivoted_rows"
				// Module values exist before the owning constructor copy is taken.
				if mode > 0 {
					if isUInt {
						lpParamsMust(t, c.SetUIntName(module, &key, 77))
					} else {
						lpParamsMust(t, c.SetBoolName(module, &key, !def))
					}
				}
				if mode == 4 {
					if isUInt {
						lpParamsMust(t, c.SetBoolName(module, &key, true))
					} else {
						lpParamsMust(t, c.SetUIntName(module, &key, 99))
					}
				}
				p, e := NewLPParams(c, local, g, b)
				lpParamsMust(t, e)
				defer func() { lpParamsMust(t, p.Close()) }()
				// Mutate the borrowed local AFTER construction, including wrong kinds.
				if mode == 2 {
					if isUInt {
						lpParamsMust(t, c.SetUIntName(local, &key, 13))
					} else {
						lpParamsMust(t, c.SetBoolName(local, &key, def))
					}
				}
				if mode == 3 {
					if isUInt {
						lpParamsMust(t, c.SetBoolName(local, &key, true))
					} else {
						lpParamsMust(t, c.SetUIntName(local, &key, 99))
					}
				}
				if isUInt {
					var got uint32
					if key == "rep_freq" {
						got, e = p.RepFreq()
					} else {
						got, e = p.SimplexStrategy()
					}
					lpParamsMust(t, e)
					want := uint32(0)
					if mode == 1 || mode == 3 {
						want = 77
					}
					if mode == 2 {
						want = 13
					}
					if got != want {
						t.Fatal(got, want)
					}
				} else {
					var got bool
					switch key {
					case "min":
						got, e = p.Min()
					case "print_stats":
						got, e = p.PrintStats()
					default:
						got, e = p.BpropOnPivotedRows()
					}
					lpParamsMust(t, e)
					want := def
					if mode == 1 || mode == 3 {
						want = !def
					}
					if got != want {
						t.Fatal(got, want)
					}
				}
				if p.owner.Serial != 1 || p.owner.Helpers != p.frame || p.frame.Local != local {
					t.Fatal("getter replaced captured reference")
				}
			})
		}
	}
}
func TestLPParamsModuleCopySharingDetachmentAndCurrentReplacement(t *testing.T) {
	c, g, local, b := lpParamsSourceFixture(t)
	module := lpParamsModule(t, g)
	key := "rep_freq"
	lpParamsMust(t, c.SetUIntName(module, &key, 41))
	p, e := NewLPParams(c, local, g, b)
	lpParamsMust(t, e)
	defer func() { lpParamsMust(t, p.Close()) }()
	// Setters detach a shared ParamsRef; captured g retains its old backing.
	lpParamsMust(t, c.SetUIntName(module, &key, 99))
	got, e := p.RepFreq()
	lpParamsMust(t, e)
	if got != 41 {
		t.Fatal("module setter should detach", got)
	}
	// A genuinely shared reset modifies the acquired backing.
	alias, e := c.NewValue()
	lpParamsMust(t, e)
	lpParamsMust(t, c.Assign(&alias, &p.frame.Value))
	lpParamsMust(t, c.ResetKey(&alias, lpParamsIntern(t, g, key)))
	got, e = p.RepFreq()
	lpParamsMust(t, e)
	if got != 0 {
		t.Fatal("shared reset", got)
	}
	lpParamsMust(t, c.Del(&alias))
	lpParamsMust(t, g.InitializeSource())
	fresh := lpParamsModule(t, g)
	lpParamsMust(t, c.SetUIntName(fresh, &key, 123))
	got, e = p.RepFreq()
	lpParamsMust(t, e)
	if got != 0 {
		t.Fatal("getter re-read registry", got)
	}
	// Local replacement is observed, since it is a borrowed reference object.
	lpParamsMust(t, c.SetUIntName(local, &key, ^uint32(0)))
	got, e = p.RepFreq()
	lpParamsMust(t, e)
	if got != ^uint32(0) {
		t.Fatal("unsigned value narrowed", got)
	}
	lpParamsMust(t, c.Reset(local))
	got, e = p.RepFreq()
	lpParamsMust(t, e)
	if got != 0 {
		t.Fatal(got)
	}
}
func lpParamsIntern(t *testing.T, g *uparams.GlobalParamsService, key string) usymbol.Symbol {
	t.Helper()
	s, e := g.Symbols.Intern(key)
	lpParamsMust(t, e)
	return s
}
func TestLPParamsDefaultEmptyOwnerGuardsAndBorrowedLifetime(t *testing.T) {
	c, g, local, b := lpParamsSourceFixture(t)
	p, e := NewLPParams(c, nil, g, b)
	lpParamsMust(t, e)
	if got, e := p.BpropOnPivotedRows(); e != nil || !got {
		t.Fatal(got, e)
	}
	copied := *p
	if e = copied.Close(); !errors.Is(e, uparams.ErrParamsOwner) {
		t.Fatal("shallow copy", e)
	}
	p.owner.Running = true
	if _, e = p.Min(); !errors.Is(e, uparams.ErrParamsPrecondition) {
		t.Fatal(e)
	}
	if e = p.Close(); !errors.Is(e, uparams.ErrParamsPrecondition) {
		t.Fatal(e)
	}
	p.owner.Running = false
	lpParamsMust(t, p.Close())
	lpParamsMust(t, p.Close())
	if _, e = p.Min(); !errors.Is(e, uparams.ErrParamsOwner) {
		t.Fatal(e)
	}
	var nilOwner *LPParams
	if _, e = nilOwner.RepFreq(); !errors.Is(e, uparams.ErrParamsOwner) {
		t.Fatal(e)
	}
	if e = nilOwner.Close(); !errors.Is(e, uparams.ErrParamsOwner) {
		t.Fatal(e)
	}
	if _, e = NewLPParams(c, local, g, arena.NewBudget(1000)); !errors.Is(e, uparams.ErrParamsOwner) {
		t.Fatal(e)
	}
	// Source constructor borrows p without inspecting it; the getter reaches p.
	invalid := uparams.ParamsRef{}
	p, e = NewLPParams(c, &invalid, g, b)
	lpParamsMust(t, e)
	if _, e = p.PrintStats(); !errors.Is(e, uparams.ErrParamsOwner) {
		t.Fatal("borrowed lifetime", e)
	}
	lpParamsMust(t, p.Close())
}
func TestLPParamsConstructorReservationFailuresAndClose(t *testing.T) {
	succeeded := false
	for ordinal := int64(0); ordinal < 20; ordinal++ {
		c, g, local, b := lpParamsSourceFixture(t)
		// Prime the module symbol; only the actual owner and retained frame allocate.
		lpParamsIntern(t, g, "lp")
		before := b.Used()
		b.FailAfter(ordinal)
		p, cause := NewLPParams(c, local, g, b)
		if cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
			t.Fatal(ordinal, cause)
		}
		if p != nil {
			if cause != nil {
				if _, e := p.SimplexStrategy(); !errors.Is(e, uparams.ErrParamsPrecondition) {
					t.Fatal(e)
				}
			}
			b.FailAfter(-1)
			lpParamsMust(t, p.Close())
		}
		b.FailAfter(-1)
		if b.Used() != before {
			t.Fatal("constructor owning cleanup", ordinal, b.Used(), before)
		}
		if cause == nil {
			succeeded = true
			break
		}
	}
	if !succeeded {
		t.Fatal("no successful constructor")
	}
}

func TestLPParamsMissingGlobalAndAcquiredModuleLifetime(t *testing.T) {
	c, g, local, b := lpParamsSourceFixture(t)
	lpParamsMust(t, g.FinalizeSource())
	before := b.Used()
	p, e := NewLPParams(c, local, g, b)
	if p == nil || e == nil || p.ready || p.owner.Helpers != nil {
		t.Fatal("reached module constructor failure", p, e)
	}
	if !errors.Is(e, uparams.ErrGlobalParamsUninitialized) && !errors.Is(e, uparams.ErrGlobalParamsAssertion) {
		t.Fatal(e)
	}
	lpParamsMust(t, p.Close())
	if b.Used() < before {
		t.Fatal("borrowed global ownership refunded")
	}
	lpParamsMust(t, g.InitializeSource())
	module := lpParamsModule(t, g)
	key := "rep_freq"
	lpParamsMust(t, c.SetUIntName(module, &key, 71))
	p, e = NewLPParams(c, local, g, b)
	lpParamsMust(t, e)
	// Destroying the current registry releases its copy, not our acquired copy.
	lpParamsMust(t, g.FinalizeSource())
	got, e := p.RepFreq()
	lpParamsMust(t, e)
	if got != 71 {
		t.Fatal("captured module owner", got)
	}
	lpParamsMust(t, p.Close())
}

func TestLPParamsRetainedShellAndLRAConsumerValues(t *testing.T) {
	c, g, local, b := lpParamsSourceFixture(t)
	h := newLPPairHarness(t)
	for world := uint32(0); world < 30; world++ {
		lpParamsMust(t, c.SetUIntName(local, lpParamsName("simplex_strategy"), []uint32{0, 1, 2, 3, 99, ^uint32(0)}[world%6]))
		lpParamsMust(t, c.SetBoolName(local, lpParamsName("bprop_on_pivoted_rows"), world%2 == 0))
		lpParamsMust(t, c.SetUIntName(local, lpParamsName("rep_freq"), ^uint32(0)-world))
		lpParamsMust(t, c.SetBoolName(local, lpParamsName("min"), world%3 == 0))
		lpParamsMust(t, c.SetBoolName(local, lpParamsName("print_stats"), world%5 == 0))
		p, e := NewLPParams(c, local, g, b)
		lpParamsMust(t, e)
		s, e := NewLARSolver(h.m)
		lpParamsMust(t, e)
		strategy, e := p.SimplexStrategy()
		lpParamsMust(t, e)
		s.Settings.SimplexStrategy = SimplexStrategy(strategy)
		pivoted, e := p.BpropOnPivotedRows()
		lpParamsMust(t, e)
		s.SetPropagateBoundsOnPivotedRowsMode(pivoted)
		if uint32(s.Settings.SimplexStrategy) != []uint32{0, 1, 2, 3, 99, ^uint32(0)}[world%6] || (s.Core.RPrimal.PivotedRows != nil) != (world%2 == 0) || (strategy < 2) != (world%6 < 2) {
			t.Fatal("LRA init/use_tableau", world)
		}
		lpParamsMust(t, s.Close())
		m := lpFloatPrimal(t)
		column := m.GetOrCreateColumnInfo(4)
		column.Cost = 7
		minimize, e := p.Min()
		lpParamsMust(t, e)
		if minimize {
			lpParamsMust(t, m.FlipCosts())
		}
		freq, e := p.RepFreq()
		lpParamsMust(t, e)
		m.Settings.ReportFrequency = int32(freq)
		stats, e := p.PrintStats()
		lpParamsMust(t, e)
		m.Settings.PrintStatistics = stats
		m.Settings.SimplexStrategy = StrategyLU
		wantCost := float64(7)
		if world%3 == 0 {
			wantCost = -7
		}
		if column.Cost != wantCost || m.Settings.ReportFrequency != -1-int32(world) || m.Settings.PrintStatistics != (world%5 == 0) || !m.Settings.UseLU() {
			t.Fatal("shell settings and cost", world)
		}
		minimize, e = p.Min()
		lpParamsMust(t, e)
		if minimize {
			lpParamsMust(t, m.FlipCosts())
		}
		if column.Cost != 7 {
			t.Fatal("shell optimal-cost restoration")
		}
		m.Close()
		lpParamsMust(t, p.Close())
	}
}
