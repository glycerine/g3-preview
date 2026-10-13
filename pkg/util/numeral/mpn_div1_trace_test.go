package numeral

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestMPNSingleLimbDivisionTraceAndDecimalCaller(t *testing.T) {
	m, _ := integerManager(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpn_div1")
	compiled := tags.ShouldTrace("mpn_div1")
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	q, r := make([]uint32, 2), make([]uint32, 1)
	if _, err := m.Natural.Div([]uint32{0, 1}, []uint32{3}, q, r); err != nil || q[0] != 1431655765 || r[0] != 1 {
		t.Fatal("single-limb division", q, r, err)
	}
	const header = "-------- [mpn_div1] div_1 src/util/mpn.cpp:296 ---------\n"
	const footer = "------------------------------------------------\n"
	if compiled {
		want := header + "j=2 q_hat=0 r_hat=1073741824 ms=1073741824 new numer=[0|1073741824|0] borrow=0\n" + footer +
			header + "j=1 q_hat=1431655765 r_hat=1073741824 ms=1073741824 new numer=[0|0|1073741824] borrow=0\n" + footer
		if got := output.String(); got != want {
			t.Fatalf("source division loop TRACE got %q, want %q", got, want)
		}
	} else if output.Len() != 0 {
		t.Fatal("erased division TRACE emitted", output.String())
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
			t.Fatal("erased decimal division TRACE emitted", output.String())
		}
		return
	}
	got := output.String()
	first := header + "j=2 q_hat=0 r_hat=268435456 ms=268435456 new numer=[0|268435456|0] borrow=0\n" + footer
	if !strings.HasPrefix(got, first) || strings.Count(got, header) != 11 || strings.Count(got, footer) != 11 {
		t.Fatalf("decimal division source loop got %q", got)
	}
}
