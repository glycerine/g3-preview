package numeral

import (
	"bytes"
	"errors"
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

func TestMPNSingleLimbDivisionTraceInsertionAndEndlRefusal(t *testing.T) {
	for _, refused := range []bool{false, true} {
		m, _ := integerManager(t)
		w := &mpfEndlWriter{}
		if refused {
			w.failAt, w.refusal = 1, errors.New("division endl refused")
		}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		tags.Enable("mpn_div1")
		if err := m.BindTrace(tags); err != nil {
			t.Fatal(err)
		}
		q, r := make([]uint32, 2), make([]uint32, 1)
		if _, err := m.Natural.Div([]uint32{0, 1}, []uint32{3}, q, r); err != nil || q[0] != 1431655765 || r[0] != 1 {
			t.Fatal("division result after trace flush", q, r, err)
		}
		if !tags.ShouldTrace("mpn_div1") {
			if len(w.events) != 0 {
				t.Fatal("erased trace wrote", w.events)
			}
			continue
		}
		joined := strings.Join(w.events, "|")
		for _, sequence := range []string{
			"write:j=|write:2|write: q_hat=|write:0|write: r_hat=|write:1073741824",
			"write: ms=|write:1073741824|write: new numer=|write:[|write:0|write:|",
			"write: borrow=|write:0|write:\n|flush",
		} {
			if !strings.Contains(joined, sequence) {
				t.Fatalf("missing source insertion sequence %q in %q", sequence, w.events)
			}
		}
		if !refused {
			if w.flushes != 4 {
				t.Fatal("two endl and two TRACE footer flushes", w.flushes, w.events)
			}
			continue
		}
		if w.flushes != 3 {
			t.Fatal("refused endl and reached TRACE footer flushes", w.flushes, w.events)
		}
		flushes := 0
		for _, event := range w.events {
			if event == "flush" {
				flushes++
			} else if flushes != 0 {
				t.Fatal("physical write after refused endl", w.events)
			}
		}
	}
}
