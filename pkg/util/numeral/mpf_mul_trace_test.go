package numeral

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestArbitraryFloatMulBugTraceSourceBoundaries(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_mul_bug")
	compiled := tags.ShouldTrace("mpf_mul_bug")
	serviceMust(t, m.BindTrace(tags))

	var zero, one, product ArbitraryFloat
	serviceMust(t, m.MkZero(11, 53, false, &zero))
	serviceMust(t, m.MkOne(11, 53, false, &one))
	for _, tc := range []struct {
		name      string
		x, y, out *ArbitraryFloat
		parts     []string
	}{
		{"zero shortcut", &zero, &one, &product, []string{"X: +zero\n", "Y: 1.0 0\n", "result: +zero\n"}},
		{"general alias", &one, &one, &one, []string{"X: 1.0 0\n", "Y: 1.0 0\n", "result: 1.0 0\n"}},
	} {
		output.Reset()
		serviceMust(t, m.Mul(ArbitraryFloatNearestEven, tc.x, tc.y, tc.out))
		got := output.String()
		if !compiled {
			if got != "" {
				t.Fatalf("%s erased trace: %q", tc.name, got)
			}
			continue
		}
		if strings.Count(got, "-------- [mpf_mul_bug] mul src/util/mpf.cpp:590 ---------\n") != 1 ||
			strings.Count(got, "-------- [mpf_mul_bug] mul src/util/mpf.cpp:664 ---------\n") != 1 {
			t.Fatalf("%s source boundaries: %q", tc.name, got)
		}
		for _, part := range tc.parts {
			if !strings.Contains(got, part) {
				t.Fatalf("%s missing %q in %q", tc.name, part, got)
			}
		}
	}
	serviceMust(t, m.Del(&zero))
	serviceMust(t, m.Del(&one))
	serviceMust(t, m.Del(&product))
}
