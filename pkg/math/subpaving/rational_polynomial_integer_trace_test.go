// Source: src/math/subpaving/subpaving_t_def.h:668-679,762-763.
package subpaving

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestRationalPolynomialSourceIntegerTraceBeforeVariablePublication(t *testing.T) {
	c, m, _, noninteger, _ := rationalClauseFixture(t)
	integer, e := c.NewVariable(true)
	if e != nil {
		t.Fatal(e)
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_mk_sum")
	if e := c.BindTrace(tags); e != nil {
		t.Fatal(e)
	}
	constant, coefficient := numeral.NewRational(), numeral.NewRational()
	t.Cleanup(func() {
		if e := m.Del(&coefficient); e != nil {
			t.Error(e)
		}
		if e := m.Del(&constant); e != nil {
			t.Error(e)
		}
	})
	if e := m.SetInt32(&constant, 0); e != nil {
		t.Fatal(e)
	}
	if e := m.SetInt32(&coefficient, 2); e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		variable uint32
		fraction bool
		want     bool
	}{
		{noninteger, false, false},
		{integer, false, true},
		{integer, true, false},
	} {
		if tc.fraction {
			if e := m.SetFractionSigned(&coefficient, 1, 2); e != nil {
				t.Fatal(e)
			}
		}
		prior := output.Len()
		id, e := c.DefinePolynomial(&constant, []numeral.Rational{coefficient}, []uint32{tc.variable})
		if e != nil {
			t.Fatal(e)
		}
		if c.variables[id] != tc.want {
			t.Fatal("variable integer flag differs from source trace", id, c.variables[id], tc.want)
		}
		if tags.ShouldTrace("subpaving_mk_sum") {
			want := "new variable is integer: 0\n"
			if tc.want {
				want = "new variable is integer: 1\n"
			}
			if got := output.String()[prior:]; !strings.Contains(got, "subpaving_t_def.h:762") || !strings.Contains(got, want) || strings.Count(got, "subpaving_t_def.h:762") != 1 {
				t.Fatalf("source mk_sum trace: %q", got)
			}
		} else if output.Len() != prior {
			t.Fatal("erased source trace emitted")
		}
	}
}
