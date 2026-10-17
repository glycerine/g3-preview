package numeral

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestMPNMulLiveIntegerSourceTraceOrder(t *testing.T) {
	m, _ := integerManager(t)
	var a, b, c Int
	serviceMust(t, m.SetLimbs(&a, []uint32{1, 1}))
	serviceMust(t, m.SetLimbs(&b, []uint32{1, 1}))
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpn")
	serviceMust(t, m.BindTrace(tags))
	serviceMust(t, m.Mul(&a, &b, &c))
	if got := integerString(t, m, &c); got != "18446744082299486209" {
		t.Fatal("live integer multiplication result", got)
	}
	if !SourceExtendedGCDDebug || !tags.ShouldTrace("mpn") {
		if output.Len() != 0 {
			t.Fatal("erased MPN trace wrote", output.String())
		}
		return
	}
	want := "-------- [mpn] trace src/util/mpn.cpp:411 ---------\n" +
		"[mpn] 4294967297 * 4294967297 == ------------------------------------------------\n" +
		"-------- [mpn] trace_nl src/util/mpn.cpp:427 ---------\n" +
		"18446744082299486209\n------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("live multiplication trace = %q, want %q", got, want)
	}
}
