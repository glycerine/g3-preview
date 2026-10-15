// Source: src/math/subpaving/subpaving_t_def.h:668-679,762-763; config_mpfx.
package subpaving

import (
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func fixedPolynomialTraceFixture(t *testing.T) (*FixedContext, *numeral.FixedPointManager, *numeral.FixedPoint, *numeral.FixedPoint) {
	t.Helper()
	m, err := numeral.NewFixedPointManager(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	two, err := m.New()
	if err != nil {
		t.Fatal(err)
	}
	half, err := m.New()
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt64(two, 2); err != nil {
		t.Fatal(err)
	}
	if err := m.SetRatio(half, 1, 2); err != nil {
		t.Fatal(err)
	}
	c, err := NewFixedContext(m, &control.Reslimit{Limit: 1000}, []bool{false, true}, 0, 2, 8192)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, e := range []error{c.Close(), half.Close(), two.Close(), m.Close()} {
			if e != nil {
				t.Error(e)
			}
		}
	})
	return c, m, two, half
}

func TestFixedPolynomialSourceIntegerTraceAndPublication(t *testing.T) {
	c, m, two, half := fixedPolynomialTraceFixture(t)
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_mk_sum")
	tags.Enable("subpaving_is_int")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		variable    uint32
		coefficient *numeral.FixedPoint
		want        bool
	}{
		{0, two, false},
		{1, two, true},
		{1, half, false},
	} {
		prior, priorFlushes := output.Len(), output.flushes
		id, err := c.DefinePolynomial(two, []FixedPolynomialTerm{{Variable: tc.variable, Coefficient: tc.coefficient}})
		if err != nil || c.isInt[id] != tc.want {
			t.Fatal("MPFX polynomial integer publication", id, err)
		}
		got := output.String()[prior:]
		if !tags.ShouldTrace("subpaving_mk_sum") {
			if got != "" || output.flushes != priorFlushes {
				t.Fatalf("erased MPFX polynomial trace: %q", got)
			}
			continue
		}
		if strings.Count(got, "subpaving_t_def.h:762") != 1 || !strings.Contains(got, "new variable is integer: ") {
			t.Fatalf("MPFX mk_sum source trace: %q", got)
		}
		wantInner := 0
		if !tc.want {
			wantInner = 2
			value, err := m.String(tc.coefficient)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(got, "polynomial is not integer due to monomial at i: 0\n") || !strings.Contains(got, "x"+strconv.FormatUint(uint64(tc.variable), 10)+" "+value+"\n") {
				t.Fatalf("MPFX first noninteger term: %q", got)
			}
		}
		if strings.Count(got, "subpaving_t_def.h:672") != wantInner || output.flushes-priorFlushes != 1+2*wantInner {
			t.Fatalf("MPFX nested trace and flush count: %q, flushes=%d", got, output.flushes-priorFlushes)
		}
	}
}

func TestFixedPolynomialTraceStreamRefusalKeepsDefinition(t *testing.T) {
	c, _, two, _ := fixedPolynomialTraceFixture(t)
	tags := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	tags.Enable("subpaving_mk_sum")
	tags.Enable("subpaving_is_int")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	id, err := c.DefinePolynomial(two, []FixedPolynomialTerm{{Variable: 0, Coefficient: two}})
	if err != nil || c.isInt[id] {
		t.Fatal("borrowed MPFX trace stream changed numeric publication", id, err)
	}
}

func TestFixedPolynomialTraceUsesBorrowedVariablePrinter(t *testing.T) {
	c, _, two, _ := fixedPolynomialTraceFixture(t)
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_is_int")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := c.SetVariableDisplay(func(w io.Writer, variable uint32) error {
		_, err := io.WriteString(w, "v"+strconv.FormatUint(uint64(variable), 10))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.DefinePolynomial(two, []FixedPolynomialTerm{{Variable: 0, Coefficient: two}}); err != nil {
		t.Fatal(err)
	}
	if tags.ShouldTrace("subpaving_is_int") && !strings.Contains(output.String(), "v0 2\n") {
		t.Fatal("borrowed source variable display", output.String())
	}
}
