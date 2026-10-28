package subpaving

// Source: src/math/subpaving/subpaving_t_def.h:668-679,762-763; config_mpff.

import (
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestFastPolynomialSourceIntegerTraceAndPublication(t *testing.T) {
	m, c, constant, _ := fastMonomialFixture(t, []bool{false, true})
	if err := m.SetInt32(constant, 2); err != nil {
		t.Fatal(err)
	}
	halfOwner, err := numeral.NewScopedFastFloat(m)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if e := halfOwner.Close(); e != nil {
			t.Error(e)
		}
	})
	half, err := halfOwner.Get()
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetFraction32(half, 1, 2); err != nil {
		t.Fatal(err)
	}
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_mk_sum")
	tags.Enable("subpaving_is_int")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		variable    uint32
		coefficient *numeral.FastFloat
		want        bool
	}{
		{0, constant, false},
		{1, constant, true},
		{1, half, false},
	} {
		prior, priorFlushes := output.Len(), output.flushes
		id, err := c.DefinePolynomial(constant, []FastPolynomialTerm{{Variable: tc.variable, Coefficient: tc.coefficient}})
		if err != nil || c.isInt[id] != tc.want {
			t.Fatal("MPFF polynomial integer publication", id, err)
		}
		got := output.String()[prior:]
		if !tags.ShouldTrace("subpaving_mk_sum") {
			if got != "" || output.flushes != priorFlushes {
				t.Fatalf("erased MPFF polynomial trace: %q", got)
			}
			continue
		}
		if strings.Count(got, "subpaving_t_def.h:762") != 1 || !strings.Contains(got, "new variable is integer: ") {
			t.Fatalf("MPFF mk_sum source trace: %q", got)
		}
		wantInner := 0
		if !tc.want {
			wantInner = 2
			value, err := m.String(tc.coefficient)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(got, "polynomial is not integer due to monomial at i: 0\n") || !strings.Contains(got, "x"+strconv.FormatUint(uint64(tc.variable), 10)+" "+value+"\n") {
				t.Fatalf("MPFF first noninteger term: %q", got)
			}
		}
		if strings.Count(got, "subpaving_t_def.h:672") != wantInner || output.flushes-priorFlushes != 1+2*wantInner {
			t.Fatalf("MPFF nested trace and flush count: %q, flushes=%d", got, output.flushes-priorFlushes)
		}
	}
}

func TestFastPolynomialTraceStreamRefusalKeepsDefinition(t *testing.T) {
	m, c, value, _ := fastMonomialFixture(t, []bool{false})
	if err := m.SetInt32(value, 2); err != nil {
		t.Fatal(err)
	}
	tags := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	tags.Enable("subpaving_mk_sum")
	tags.Enable("subpaving_is_int")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	id, err := c.DefinePolynomial(value, []FastPolynomialTerm{{Variable: 0, Coefficient: value}})
	if err != nil || c.isInt[id] {
		t.Fatal("borrowed MPFF trace stream changed numeric publication", id, err)
	}
}
