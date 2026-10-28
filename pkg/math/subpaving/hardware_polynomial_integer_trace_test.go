package subpaving

// Source: src/math/subpaving/subpaving_t_def.h:668-679,762-763; config_hwf.

import (
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func hardwarePolynomialTraceFixture(t *testing.T) (*HardwareContext, *numeral.HardwareFloatNumeralManager, numeral.HardwareFloat, numeral.HardwareFloat) {
	t.Helper()
	m, err := numeral.NewHardwareFloatNumeralManager(numeral.NewHardwareFloatManager(), 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewHardwareContext(m, &control.Reslimit{Limit: 1000}, 0, 2, 8192)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	})
	for _, integer := range []bool{false, true} {
		if _, err := c.NewVariable(integer); err != nil {
			t.Fatal(err)
		}
	}
	var two, half numeral.HardwareFloat
	if err := m.SetInt32(&two, 2); err != nil {
		t.Fatal(err)
	}
	if err := m.SetFloat64(&half, 0.5); err != nil {
		t.Fatal(err)
	}
	return c, m, two, half
}

func TestHardwarePolynomialSourceIntegerTraceAndPublication(t *testing.T) {
	c, m, two, half := hardwarePolynomialTraceFixture(t)
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_mk_sum")
	tags.Enable("subpaving_is_int")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		variable    uint32
		coefficient *numeral.HardwareFloat
		want        bool
	}{
		{0, &two, false},
		{1, &two, true},
		{1, &half, false},
	} {
		prior, priorFlushes := output.Len(), output.flushes
		id, err := c.DefinePolynomial(&two, []HardwarePolynomialTerm{{Variable: tc.variable, Coefficient: tc.coefficient}})
		if err != nil || c.isInt[id] != tc.want {
			t.Fatal("HWF polynomial integer publication", id, err)
		}
		got := output.String()[prior:]
		if !tags.ShouldTrace("subpaving_mk_sum") {
			if got != "" || output.flushes != priorFlushes {
				t.Fatalf("erased HWF polynomial trace: %q", got)
			}
			continue
		}
		if strings.Count(got, "subpaving_t_def.h:762") != 1 || !strings.Contains(got, "new variable is integer: ") {
			t.Fatalf("HWF mk_sum source trace: %q", got)
		}
		wantInner := 0
		if !tc.want {
			wantInner = 2
			value, err := m.ToString(tc.coefficient)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(got, "polynomial is not integer due to monomial at i: 0\n") || !strings.Contains(got, "x"+strconv.FormatUint(uint64(tc.variable), 10)+" "+value+"\n") {
				t.Fatalf("HWF first noninteger term: %q", got)
			}
		}
		if strings.Count(got, "subpaving_t_def.h:672") != wantInner || output.flushes-priorFlushes != 1+2*wantInner {
			t.Fatalf("HWF nested trace and flush count: %q, flushes=%d", got, output.flushes-priorFlushes)
		}
	}
}

func TestHardwarePolynomialTraceStreamRefusalKeepsDefinition(t *testing.T) {
	c, _, two, _ := hardwarePolynomialTraceFixture(t)
	tags := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	tags.Enable("subpaving_mk_sum")
	tags.Enable("subpaving_is_int")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	id, err := c.DefinePolynomial(&two, []HardwarePolynomialTerm{{Variable: 0, Coefficient: &two}})
	if err != nil || c.isInt[id] {
		t.Fatal("borrowed HWF trace stream changed numeric publication", id, err)
	}
}
