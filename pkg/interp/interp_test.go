package interp

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/symbol"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func interpMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func interpParamsFixture(t *testing.T) (*params.ParamsContext, *params.GlobalParamsService) {
	t.Helper()
	b := arena.NewBudget(1 << 30)
	s, err := symbol.NewManager(symbol.Config{Budget: b})
	interpMust(t, err)
	q, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	interpMust(t, err)
	c, err := params.NewParamsContext(s, q, params.ParamsConfig{Budget: b})
	interpMust(t, err)
	catalog, err := params.NewGlobalParamsCatalog(s, params.GlobalParamsCatalogConfig{Budget: b})
	interpMust(t, err)
	g, err := params.NewGlobalParamsService(s, c, params.GlobalParamsConfig{Budget: b, Catalog: catalog})
	interpMust(t, err)
	interpMust(t, g.InitializeSource())
	t.Cleanup(func() {
		interpMust(t, g.Close())
		interpMust(t, catalog.Close())
		interpMust(t, c.Close())
		interpMust(t, q.Close())
		interpMust(t, s.Close())
		if b.Used() != 0 {
			t.Errorf("interpolation fixture retained %d bytes", b.Used())
		}
	})
	return c, g
}

func TestInterpParamsDefaultsModuleAndLocalPrecedence(t *testing.T) {
	c, g := interpParamsFixture(t)
	p, err := NewParams(c, g, nil)
	interpMust(t, err)
	if profile, e := p.Profile(); e != nil || profile {
		t.Fatal(profile, e)
	}
	if check, e := p.Check(); e != nil || check {
		t.Fatal(check, e)
	}
	interpMust(t, p.Close())

	manager, err := g.Manager()
	interpMust(t, err)
	module, err := c.Symbols().Intern("interp")
	interpMust(t, err)
	global, err := manager.GetParams(module)
	interpMust(t, err)
	profileKey, err := c.Symbols().Intern("profile")
	interpMust(t, err)
	checkKey, err := c.Symbols().Intern("check")
	interpMust(t, err)
	interpMust(t, c.SetBool(global, profileKey, true))
	interpMust(t, c.SetBool(global, checkKey, true))
	local, err := c.NewValue()
	interpMust(t, err)
	defer func() { interpMust(t, c.Del(&local)) }()
	interpMust(t, c.SetBool(&local, profileKey, false))
	p, err = NewParams(c, g, &local)
	interpMust(t, err)
	if profile, _ := p.Profile(); profile {
		t.Fatal("local profile did not override module")
	}
	if check, _ := p.Check(); !check {
		t.Fatal("module check fallback")
	}
	interpMust(t, p.Close())
	if _, err = p.Check(); !errors.Is(err, params.ErrParamsOwner) {
		t.Fatal("closed facade", err)
	}
}

func TestInterpolationProfilerNestedAccumulationAndGuards(t *testing.T) {
	p := NewProfiler()
	base := time.Unix(100, 0)
	p.origin = base
	times := []time.Time{base, base.Add(time.Second), base.Add(3 * time.Second), base.Add(5 * time.Second), base.Add(6 * time.Second), base.Add(8 * time.Second)}
	i := 0
	p.now = func() time.Time { v := times[i]; i++; return v }
	interpMust(t, p.TimerStart("outer"))
	interpMust(t, p.TimerStart("inner"))
	interpMust(t, p.TimerStop("inner"))
	interpMust(t, p.TimerStop("outer"))
	interpMust(t, p.TimerStart("outer"))
	interpMust(t, p.TimerStop("outer"))
	if err := p.TimerStop("outer"); err == nil || err.Error() != "imbalanced timer_start and timer_stop" {
		t.Fatal(err)
	}
	var out bytes.Buffer
	interpMust(t, p.Print(&out))
	text := out.String()
	for _, want := range []string{"TOTAL", "outer", "inner", "TOTALS:", "outer 7", "inner 2"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in %q", want, text)
		}
	}
	p.now = func() time.Time { return base.Add(11 * time.Second) }
	out.Reset()
	interpMust(t, p.ShowTime(&out))
	if out.String() != "11\n" {
		t.Fatalf("elapsed wall clock = %q", out.String())
	}
}

func TestInterpolationErrorMatchesDefaultExceptionContract(t *testing.T) {
	err := NewError("interpolation failed")
	var classified z3exception.Z3Exception
	if !errors.As(err, &classified) || classified.Message() != "interpolation failed" || classified.HasErrorCode() || classified.ErrorCode() != z3exception.ErrorOK {
		t.Fatal(classified)
	}
}
