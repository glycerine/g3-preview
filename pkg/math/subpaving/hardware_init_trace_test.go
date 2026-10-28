package subpaving

// Source: src/math/subpaving/subpaving_t_def.h:1773-1801; config_hwf.

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestHardwareRootInitializationTraceSourceOrderAndResume(t *testing.T) {
	m, c, v, root := hardwareSearchTraceFixture(t, []bool{false})
	if err := m.SetInt32(v, 3); err != nil {
		t.Fatal(err)
	}
	a, err := c.NewInequality(0, v, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.AddUnitClause(a, true); err != nil {
		t.Fatal(err)
	}
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_init")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.InitializeRoot(); err != nil || c.initStage != 3 || c.unitHead != 1 || root.lower[0] == nil {
		t.Fatal("root not initialized", err, c.initStage, c.unitHead)
	}
	if tags.ShouldTrace("subpaving_init") {
		got := output.String()
		for _, part := range []string{
			"subpaving_t_def.h:1797", "3 <= x0\n", "subpaving_t_def.h:1780", "asserting: 3 <= x0, axiom: 1\n",
			"subpaving_t_def.h:1785", "bounds after init\n3 <= x0 ",
			"subpaving_t_def.h:1801", "root bounds after propagation\n3 <= x0 ",
		} {
			if !strings.Contains(got, part) {
				t.Fatalf("missing source initialization TRACE %q in %q", part, got)
			}
		}
		if output.flushes != 4 || !(strings.Index(got, "subpaving_t_def.h:1797") < strings.Index(got, "subpaving_t_def.h:1780") &&
			strings.Index(got, "subpaving_t_def.h:1780") < strings.Index(got, "subpaving_t_def.h:1785") &&
			strings.Index(got, "subpaving_t_def.h:1785") < strings.Index(got, "subpaving_t_def.h:1801")) {
			t.Fatalf("source order or flushes: %q; %d", got, output.flushes)
		}
	} else if output.Len() != 0 || output.flushes != 0 {
		t.Fatal("erased initialization TRACE emitted")
	}
	before, flushes := output.String(), output.flushes
	if err := c.InitializeRoot(); err != nil || output.String() != before || output.flushes != flushes {
		t.Fatal("completed root initialization replayed TRACE", err)
	}
}

func TestHardwareRootInitializationTraceRefusalAndPrinterPrefix(t *testing.T) {
	m, c, v, _ := hardwareSearchTraceFixture(t, []bool{false})
	if err := m.SetInt32(v, 5); err != nil {
		t.Fatal(err)
	}
	a, err := c.NewInequality(0, v, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.AddUnitClause(a, true); err != nil {
		t.Fatal(err)
	}
	failed := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	failed.Enable("subpaving_init")
	if err := c.BindTrace(failed); err != nil {
		t.Fatal(err)
	}
	if err := c.InitializeRoot(); err != nil || c.initStage != 3 || c.root.lower[0] == nil {
		t.Fatal("borrowed writer refusal changed initialization", err)
	}

	dm, d, dv, _ := hardwareSearchTraceFixture(t, []bool{false})
	if err := dm.SetInt32(dv, 7); err != nil {
		t.Fatal(err)
	}
	atom, err := d.NewInequality(0, dv, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.AddUnitClause(atom, true); err != nil {
		t.Fatal(err)
	}
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_init")
	if err := d.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	printerErr := errors.New("borrowed variable printer stopped")
	calls := 0
	if err := d.SetVariableDisplay(func(w io.Writer, _ uint32) error {
		calls++
		_, _ = io.WriteString(w, "candidate")
		if calls == 2 {
			return printerErr
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	err = d.InitializeRoot()
	if tags.ShouldTrace("subpaving_init") {
		if !errors.Is(err, printerErr) || d.unitHead != 0 || d.root.lower[0] != nil || d.initStage != 0 ||
			!d.initConstraintsTraced || !strings.Contains(output.String(), "asserting: 7 <= candidate") {
			t.Fatal("printer stopped before first unit publication", err, d.unitHead, d.initStage, output.String())
		}
		if err := d.SetVariableDisplay(nil); err != nil {
			t.Fatal(err)
		}
		before := output.String()
		if err := d.InitializeRoot(); err != nil || d.unitHead != 1 || strings.Count(output.String(), "subpaving_t_def.h:1797") != 1 || output.String() == before {
			t.Fatal("resume repeated constraints or skipped unit", err, output.String())
		}
	} else if err != nil || d.unitHead != 1 {
		t.Fatal("native trace gate changed initialization", err, d.unitHead)
	}
}
