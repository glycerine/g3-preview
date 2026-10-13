package numeral

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestMPNNormalizationTraceDivisionAndDecimalConversion(t *testing.T) {
	m, _ := integerManager(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpn_norm")
	compiled := tags.ShouldTrace("mpn_norm")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	q, r := make([]uint32, 2), make([]uint32, 1)
	if _, err := m.Natural.Div([]uint32{0, 1}, []uint32{3}, q, r); err != nil || q[0] != 1431655765 || r[0] != 1 {
		t.Fatal("normalized division", q, r, err)
	}
	const header = "-------- [mpn_norm] div_normalize src/util/mpn.cpp:257 ---------\n"
	const footer = "------------------------------------------------\n"
	if compiled {
		want := header + "Normalized: n_numer=[0|1073741824|0] n_denom=[3221225472]\n" + footer
		if got := output.String(); got != want {
			t.Fatalf("division normalization TRACE got %q, want %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased normalization TRACE emitted", output.String())
	}
	output.Reset()
	var chars [40]byte
	if err := m.Natural.sourceToCString(limbView{external: []uint32{0, 1}, size: 2},
		sourceDecimalDestination{external: chars[:]}, uint64(len(chars))); err != nil {
		t.Fatal(err)
	}
	if got := string(chars[:10]); got != "4294967296" {
		t.Fatal("decimal conversion", got)
	}
	if !compiled {
		if output.Len() != 0 {
			t.Fatal("erased decimal normalization TRACE emitted", output.String())
		}
		return
	}
	first := header + "Normalized: n_numer=[0|268435456|0] n_denom=[2684354560]\n" + footer
	got := output.String()
	if !strings.HasPrefix(got, first) || strings.Count(got, header) != 10 || strings.Count(got, footer) != 10 {
		t.Fatalf("decimal normalization source loop got %q", got)
	}
}
