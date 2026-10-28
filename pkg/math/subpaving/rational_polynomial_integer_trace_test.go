package subpaving

// Source: src/math/subpaving/subpaving_t_def.h:668-679,762-763.

import (
	"bytes"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

type polynomialTraceRefusingWriter struct{}

func (polynomialTraceRefusingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

type polynomialTraceFlushingBuffer struct {
	bytes.Buffer
	flushes int
}

func (w *polynomialTraceFlushingBuffer) Flush() error {
	w.flushes++
	return nil
}

func TestRationalPolynomialSourceIntegerTraceBeforeVariablePublication(t *testing.T) {
	c, m, _, noninteger, _ := rationalClauseFixture(t)
	integer, e := c.NewVariable(true)
	if e != nil {
		t.Fatal(e)
	}
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_mk_sum")
	tags.Enable("subpaving_is_int")
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
		priorFlushes := output.flushes
		id, e := c.DefinePolynomial(&constant, []numeral.Rational{coefficient}, []uint32{tc.variable})
		if e != nil {
			t.Fatal(e)
		}
		if c.variables[id] != tc.want {
			t.Fatal("variable integer flag differs from source trace", id, c.variables[id], tc.want)
		}
		if tags.ShouldTrace("subpaving_mk_sum") {
			want := "new variable is integer: "
			if tc.want {
				want += "1\n"
			} else {
				want += "-------- [subpaving_is_int]"
			}
			if got := output.String()[prior:]; !strings.Contains(got, "subpaving_t_def.h:762") || !strings.Contains(got, want) || strings.Count(got, "subpaving_t_def.h:762") != 1 {
				t.Fatalf("source mk_sum trace: %q", got)
			}
			got := output.String()[prior:]
			wantInner := 0
			if !tc.want {
				wantInner = 2 // once inside TRACE, then again for mk_var
				coefficientText := "2"
				if tc.fraction {
					coefficientText = "1/2"
				}
				if !strings.Contains(got, "polynomial is not integer due to monomial at i: 0\n") || !strings.Contains(got, "x"+strconv.FormatUint(uint64(tc.variable), 10)+" "+coefficientText+"\n") {
					t.Fatalf("source is_int term display: %q", got)
				}
			}
			if strings.Count(got, "subpaving_t_def.h:672") != wantInner {
				t.Fatalf("source predicate trace count %d, want %d: %q", strings.Count(got, "subpaving_t_def.h:672"), wantInner, got)
			}
			if wantFlushes := 1 + 2*wantInner; output.flushes-priorFlushes != wantFlushes {
				t.Fatalf("source explicit and footer flushes: got %d, want %d", output.flushes-priorFlushes, wantFlushes)
			}
		} else if output.Len() != prior {
			t.Fatal("erased source trace emitted")
		}
	}
}

func TestRationalPolynomialTraceStreamRefusalKeepsDefinition(t *testing.T) {
	c, m, _, x, _ := rationalClauseFixture(t)
	tags := diagnostic.NewTraceTagsWithWriter(polynomialTraceRefusingWriter{})
	tags.Enable("subpaving_is_int")
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
	if e := m.SetInt32(&coefficient, 2); e != nil {
		t.Fatal(e)
	}
	id, e := c.DefinePolynomial(&constant, []numeral.Rational{coefficient}, []uint32{x})
	if e != nil || c.variables[id] {
		t.Fatal("trace stream refusal changed source integer classification", id, e)
	}
}
