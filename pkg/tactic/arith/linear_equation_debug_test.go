package arith

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLinearEquationCoreSourceSortedDebugGuard(t *testing.T) {
	debug := equationSourceDebugFixture(t)
	m, l := equationManager(t)
	as := []numeral.Int{numeral.NewInt(2), numeral.NewInt(3)}
	defer func() {
		for i := range as {
			equationMust(t, m.Integers().Del(&as[i]))
		}
	}()
	eq, err := l.mkCore(as, []uint32{4, 2})
	if diagnostic.DebugBuild {
		if !errors.Is(err, diagnostic.ErrDebugAssertion) || eq != nil {
			t.Fatalf("source sorted assertion: eq=%v err=%v", eq, err)
		}
		equationMust(t, debug.EnableAssertions(false))
		eq, err = l.mkCore(as, []uint32{4, 2})
		equationMust(t, err)
		equationMust(t, eq.Close())
		return
	}
	equationMust(t, err)
	equationMust(t, eq.Close())
}

func equationSourceDebugFixture(t *testing.T) *diagnostic.DebugContext {
	t.Helper()
	budget := arena.NewBudget(1 << 20)
	region, err := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	equationMust(t, err)
	debug, err := diagnostic.NewDebugContext(region, budget)
	equationMust(t, err)
	previous, err := diagnostic.BindProcessDebugContext(debug)
	equationMust(t, err)
	t.Cleanup(func() {
		_, e := diagnostic.BindProcessDebugContext(previous)
		equationMust(t, e)
		equationMust(t, debug.Close())
		equationMust(t, region.Close())
		if budget.Used() != 0 {
			t.Fatalf("debug owner retained %d bytes", budget.Used())
		}
	})
	return debug
}

func TestLinearEquationNormalizedSourceDuplicateDebugGuard(t *testing.T) {
	debug := equationSourceDebugFixture(t)
	m, l := equationManager(t)
	as := []numeral.Int{numeral.NewInt(2), numeral.NewInt(3)}
	defer func() {
		for i := range as {
			equationMust(t, m.Integers().Del(&as[i]))
		}
	}()
	if !diagnostic.DebugBuild {
		equationMust(t, l.assertUniqueNormalizedEquationVars([]uint32{2, 2}))
		if len(l.mark) != 0 {
			t.Fatal("erased DEBUG_CODE changed mark storage")
		}
		return
	}
	eq, err := l.MkInt(as, []uint32{3, 1}, true)
	equationMust(t, err)
	equationMust(t, eq.Close())
	if l.mark[1] || l.mark[3] {
		t.Fatal("successful DEBUG_CODE retained variable marks")
	}
	eq, err = l.MkInt(as, []uint32{2, 2}, true)
	if !errors.Is(err, diagnostic.ErrDebugAssertion) || eq != nil {
		t.Fatalf("source duplicate assertion: eq=%v err=%v", eq, err)
	}
	if len(l.mark) <= 2 || !l.mark[2] {
		t.Fatal("source partial mark was not retained after assertion")
	}
	equationMust(t, debug.EnableAssertions(false))
	equationMust(t, l.assertUniqueNormalizedEquationVars([]uint32{2, 2}))
	if !l.mark[2] {
		t.Fatal("disabled DEBUG_CODE cleared the source partial mark")
	}
}

func TestLinearEquationCombineZeroMultiplierSourceAssertionProfile(t *testing.T) {
	debug := equationSourceDebugFixture(t)
	m, l := equationManager(t)
	as := []numeral.Int{numeral.NewInt(2), numeral.NewInt(3)}
	defer func() {
		for i := range as {
			equationMust(t, m.Integers().Del(&as[i]))
		}
	}()
	eq, err := l.MkInt(as, []uint32{0, 1}, true)
	equationMust(t, err)
	defer func() { equationMust(t, eq.Close()) }()
	zero, one := numeral.NewInt(0), numeral.NewInt(1)
	defer func() {
		equationMust(t, m.Integers().Del(&one))
		equationMust(t, m.Integers().Del(&zero))
	}()
	if diagnostic.DebugBuild {
		for _, pair := range [][2]*numeral.Int{{&zero, &one}, {&one, &zero}} {
			result, e := l.Combine(pair[0], eq, pair[1], eq)
			if !errors.Is(e, diagnostic.ErrDebugAssertion) || result != nil {
				t.Fatalf("zero multiplier assertion: result=%v err=%v", result, e)
			}
		}
		equationMust(t, debug.EnableAssertions(false))
	}
	for _, pair := range [][2]*numeral.Int{{&zero, &one}, {&one, &zero}} {
		result, e := l.Combine(pair[0], eq, pair[1], eq)
		equationMust(t, e)
		var output bytes.Buffer
		equationMust(t, l.Display(&output, result))
		if output.String() != "2*x0 + 3*x1 = 0" {
			t.Fatalf("release zero multiplier result %q", output.String())
		}
		equationMust(t, result.Close())
	}
	result, err := l.Combine(&zero, eq, &zero, eq)
	equationMust(t, err)
	if result != nil {
		t.Fatal("two zero multipliers should cancel every shared term")
	}
}
