// Source: src/math/subpaving/subpaving_t_def.h:1773-1801; config_mpq.
package subpaving

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestRationalRootInitializationTraceSourceOrderAndResume(t *testing.T) {
	c, m, value, x, _ := rationalClauseFixture(t)
	if err := m.SetInt32(value, 3); err != nil {
		t.Fatal(err)
	}
	a, err := c.AddInequality(x, value, true, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if a.refs != 1 {
		t.Fatal("unit ownership")
	}
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_init")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.InitializeRoot(); err != nil {
		t.Fatal(err)
	}
	if c.initStage != 3 || c.unitHead != 1 || c.root.lower[x] == nil {
		t.Fatal("root not initialized", c.initStage, c.unitHead)
	}
	if tags.ShouldTrace("subpaving_init") {
		got := output.String()
		for _, part := range []string{"subpaving_t_def.h:1797", "3 <= x0\n", "subpaving_t_def.h:1780", "asserting: 3 <= x0, axiom: 1\n", "subpaving_t_def.h:1785", "bounds after init\n3 <= x0 \n", "subpaving_t_def.h:1801", "root bounds after propagation\n3 <= x0 \n"} {
			if !strings.Contains(got, part) {
				t.Fatalf("missing source initialization trace %q in %q", part, got)
			}
		}
		if output.flushes != 4 || !(strings.Index(got, "subpaving_t_def.h:1797") < strings.Index(got, "subpaving_t_def.h:1780") && strings.Index(got, "subpaving_t_def.h:1780") < strings.Index(got, "subpaving_t_def.h:1785") && strings.Index(got, "subpaving_t_def.h:1785") < strings.Index(got, "subpaving_t_def.h:1801")) {
			t.Fatalf("source order or footer flushes: %q; flushes %d", got, output.flushes)
		}
	} else if output.Len() != 0 || output.flushes != 0 {
		t.Fatal("erased source trace emitted", output.String(), output.flushes)
	}
	before, flushes := output.String(), output.flushes
	if err := c.InitializeRoot(); err != nil || output.String() != before || output.flushes != flushes {
		t.Fatal("completed root initialization replayed trace", err, output.String(), output.flushes)
	}
}

func TestRationalRootInitializationTraceRefusalAndPrinterPrefix(t *testing.T) {
	c, m, value, x, _ := rationalClauseFixture(t)
	if err := m.SetInt32(value, 5); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AddInequality(x, value, true, false, false); err != nil {
		t.Fatal(err)
	}
	failed := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	failed.Enable("subpaving_init")
	if err := c.BindTrace(failed); err != nil {
		t.Fatal(err)
	}
	if err := c.InitializeRoot(); err != nil || c.initStage != 3 || c.root.lower[x] == nil {
		t.Fatal("trace stream refusal changed initialization", err, c.initStage)
	}

	d, dm, dv, dx, _ := rationalClauseFixture(t)
	if err := dm.SetInt32(dv, 7); err != nil {
		t.Fatal(err)
	}
	if _, err := d.AddInequality(dx, dv, true, false, true); err != nil {
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
	if err := d.SetVariableDisplay(func(w io.Writer, variable uint32) error {
		calls++
		_, _ = io.WriteString(w, "candidate")
		if calls == 2 {
			return printerErr
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	err := d.InitializeRoot()
	if tags.ShouldTrace("subpaving_init") {
		if !errors.Is(err, printerErr) || d.unitHead != 0 || d.root.lower[dx] != nil || d.initStage != 0 || !d.initConstraintsTraced || !strings.Contains(output.String(), "asserting: 7 <= candidate") {
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
		t.Fatal("native trace gate changed root initialization", err, d.unitHead)
	}
}
