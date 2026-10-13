package numeral

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestMPNMultiLimbDivisionTraceStopsAtSourceOutOfBoundsRead(t *testing.T) {
	m, _ := integerManager(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpn_div")
	compiled := tags.ShouldTrace("mpn_div")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	q, r := make([]uint32, 1), make([]uint32, 2)
	if _, err := m.Natural.Div([]uint32{5, 2}, []uint32{1, 1}, q, r); err != nil || q[0] != 2 || r[0] != 3 || r[1] != 0 {
		t.Fatal("Algorithm D result", q, r, err)
	}
	if !compiled {
		if output.Len() != 0 {
			t.Fatal("erased multi-limb TRACE emitted", output.String())
		}
		return
	}
	const want = "-------- [mpn_div] div_n src/util/mpn.cpp:350 ---------\n" +
		"q_hat=2 r_hat=2 ms=[1|0] new numer="
	if got := output.String(); got != want {
		t.Fatalf("source-defined TRACE prefix got %q, want %q", got, want)
	}
}
