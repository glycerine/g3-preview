package numeral

import (
	"bytes"
	"io"
	"reflect"
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
		"q_hat=2 r_hat=2 ms=[1|0] new numer=["
	if got := output.String(); got != want {
		t.Fatalf("source-defined TRACE prefix got %q, want %q", got, want)
	}
}

func TestMPNMultiLimbDivisionTraceSourcePrefixAndRefusal(t *testing.T) {
	for _, refused := range []bool{false, true} {
		m, _ := integerManager(t)
		var w io.Writer
		good := &mpfEndlWriter{}
		bad := &mpfRoundRefusalWriter{refuse: " r_hat="}
		if refused {
			w = bad
		} else {
			w = good
		}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		tags.Enable("mpn_div")
		if err := m.BindTrace(tags); err != nil {
			t.Fatal(err)
		}
		q, r := make([]uint32, 1), make([]uint32, 2)
		if _, err := m.Natural.Div([]uint32{5, 2}, []uint32{1, 1}, q, r); err != nil || q[0] != 2 || r[0] != 3 || r[1] != 0 {
			t.Fatal("Algorithm D after trace stop", q, r, err)
		}
		if !tags.ShouldTrace("mpn_div") {
			if len(good.events) != 0 || len(bad.parts) != 0 {
				t.Fatal("erased trace wrote", good.events, bad.parts)
			}
			continue
		}
		if refused {
			if got := bad.parts[len(bad.parts)-2:]; !reflect.DeepEqual(got, []string{"q_hat=", "2"}) {
				t.Fatal("refusal after separate quotient insertions", bad.parts)
			}
			continue
		}
		if good.flushes != 0 {
			t.Fatal("undefined limb read reached TRACE footer", good.events)
		}
		want := []string{"write:q_hat=", "write:2", "write: r_hat=", "write:2", "write: ms=", "write:[", "write:1", "write:|", "write:0", "write:]", "write: new numer=", "write:["}
		if len(good.events) < len(want) || !reflect.DeepEqual(good.events[len(good.events)-len(want):], want) {
			t.Fatal("source operands and opening bracket before invalid limb", good.events)
		}
	}
}
