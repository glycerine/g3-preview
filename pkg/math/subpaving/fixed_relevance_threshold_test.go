// Source: src/math/subpaving/subpaving_t_def.h:1233-1309; config_mpfx.
package subpaving

import (
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestFixedRelevanceUsesSourceMaximumOfGapAndOne(t *testing.T) {
	m, err := numeral.NewFixedPointManager(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	v, err := m.New()
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewFixedContext(m, &control.Reslimit{Limit: 1000}, []bool{false}, 20, 2, 8192)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, e := range []error{c.Close(), v.Close(), m.Close()} {
			if e != nil {
				t.Error(e)
			}
		}
	})
	root, err := c.Root()
	if err != nil {
		t.Fatal(err)
	}
	for _, bound := range []struct {
		value int64
		lower bool
	}{{5, true}, {10, false}} {
		if err := m.SetInt64(v, bound.value); err != nil {
			t.Fatal(err)
		}
		if err := c.PropagateBound(root, 0, v, bound.lower, false); err != nil {
			t.Fatal(err)
		}
	}
	var output polynomialTraceFlushingBuffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subpaving_relevant_bound")
	tags.Enable("subpaving_relevant_bound_bug")
	if err := c.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		num, den    int64
		lower, want bool
		line        int
	}{
		{41, 8, true, false, 1259},  // 5.125 does not exceed 5 + 5/20.
		{11, 2, true, true, 1310},   // 5.5 does.
		{79, 8, false, false, 1304}, // 9.875 is not below 10 - 5/20.
		{19, 2, false, true, 1310},  // 9.5 is.
	} {
		if err := m.SetRatio(v, tc.num, uint64(tc.den)); err != nil {
			t.Fatal(err)
		}
		prior, flushes := output.Len(), output.flushes
		got, err := c.RelevantNewBound(root, 0, v, tc.lower, false)
		if err != nil || got != tc.want {
			t.Fatalf("relevance of %d/%d lower=%t: got %t, %v; want %t", tc.num, tc.den, tc.lower, got, err, tc.want)
		}
		if tags.ShouldTrace("subpaving_relevant_bound") {
			record := output.String()[prior:]
			wantFlushes := 2
			if tc.lower {
				wantFlushes++
				if !strings.Contains(record, "subpaving_t_def.h:1253") {
					t.Fatal("missing lower threshold debug TRACE", record)
				}
			}
			if !strings.Contains(record, "subpaving_t_def.h:1210") || !strings.Contains(record, "existing bounds:\n") ||
				!strings.Contains(record, "subpaving_t_def.h:"+strconv.Itoa(tc.line)) || output.flushes-flushes != wantFlushes {
				t.Fatalf("relevance TRACE %d/%d: %q, flushes=%d", tc.num, tc.den, record, output.flushes-flushes)
			}
		} else if output.Len() != prior || output.flushes != flushes {
			t.Fatal("erased relevance TRACE emitted", output.String())
		}
	}
}
