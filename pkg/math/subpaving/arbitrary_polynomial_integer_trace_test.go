package subpaving

// Source: src/math/subpaving/subpaving_t_def.h:668-679,762-763; config_mpf.

import (
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestArbitraryPolynomialSourceIntegerTraceAndPublication(t *testing.T) {
	f := newArbitraryClauseFixtureWithPolicy(t, 0, true)
	c := f.context
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_mk_sum")
	tags.Enable("subpaving_is_int")
	if e := c.BindTrace(tags); e != nil {
		t.Fatal(e)
	}
	var half numeral.ArbitraryFloat
	if e := f.numbers.SetFractionSigned(&half, 1, 2); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := f.numbers.Del(&half); e != nil {
			t.Error(e)
		}
	})
	for _, tc := range []struct {
		variable    uint32
		coefficient *numeral.ArbitraryFloat
		want        bool
	}{
		{f.x, &f.four, false},
		{f.y, &f.four, true},
		{f.y, &half, false},
	} {
		prior, priorFlushes := output.Len(), output.flushes
		id, e := c.DefinePolynomial(&f.five, []numeral.ArbitraryFloat{*tc.coefficient}, []uint32{tc.variable})
		if e != nil || c.variables[id] != tc.want {
			t.Fatal("source polynomial integer publication", id, e)
		}
		got := output.String()[prior:]
		if !tags.ShouldTrace("subpaving_mk_sum") {
			if got != "" || output.flushes != priorFlushes {
				t.Fatalf("erased MPF polynomial trace: %q", got)
			}
			continue
		}
		if strings.Count(got, "subpaving_t_def.h:762") != 1 || !strings.Contains(got, "new variable is integer: ") {
			t.Fatalf("MPF mk_sum source trace: %q", got)
		}
		wantInner := 0
		if !tc.want {
			wantInner = 2
			value, e := f.numbers.ToString(tc.coefficient)
			if e != nil {
				t.Fatal(e)
			}
			if !strings.Contains(got, "polynomial is not integer due to monomial at i: 0\n") || !strings.Contains(got, "x"+strconv.FormatUint(uint64(tc.variable), 10)+" "+value+"\n") {
				t.Fatalf("MPF first noninteger term: %q", got)
			}
		}
		if strings.Count(got, "subpaving_t_def.h:672") != wantInner || output.flushes-priorFlushes != 1+2*wantInner {
			t.Fatalf("MPF nested trace and flush count: %q, flushes=%d", got, output.flushes-priorFlushes)
		}
	}
}

func TestArbitraryPolynomialTraceStreamRefusalKeepsDefinition(t *testing.T) {
	f := newArbitraryClauseFixture(t)
	tags := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	tags.Enable("subpaving_mk_sum")
	tags.Enable("subpaving_is_int")
	if e := f.context.BindTrace(tags); e != nil {
		t.Fatal(e)
	}
	id, e := f.context.DefinePolynomial(&f.five, []numeral.ArbitraryFloat{f.four}, []uint32{f.x})
	if e != nil || f.context.variables[id] {
		t.Fatal("borrowed MPF trace stream changed numeric publication", id, e)
	}
}
