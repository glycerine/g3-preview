package numeral

import (
	"bytes"
	"errors"
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

func TestMPNNormalizationTraceEndlFlushAndRefusal(t *testing.T) {
	for _, refused := range []bool{false, true} {
		m, _ := integerManager(t)
		w := &mpfEndlWriter{}
		if refused {
			w.failAt, w.refusal = 1, errors.New("normalization endl refused")
		}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		tags.Enable("mpn_norm")
		if err := m.BindTrace(tags); err != nil {
			t.Fatal(err)
		}
		q, r := make([]uint32, 2), make([]uint32, 1)
		if _, err := m.Natural.Div([]uint32{0, 1}, []uint32{3}, q, r); err != nil || q[0] != 1431655765 || r[0] != 1 {
			t.Fatal("division after normalization trace flush", q, r, err)
		}
		if !tags.ShouldTrace("mpn_norm") {
			if len(w.events) != 0 {
				t.Fatal("erased trace wrote", w.events)
			}
			continue
		}
		joined := strings.Join(w.events, "|")
		if !strings.Contains(joined, "write:Normalized: n_numer=|write:[|write:0") ||
			!strings.Contains(joined, "write: n_denom=|write:[|write:3221225472|write:]|write:\n|flush") {
			t.Fatal("normalization source insertion and endl order", w.events)
		}
		if w.flushes != 2 {
			t.Fatal("normalization endl and TRACE footer flushes", w.flushes, w.events)
		}
		if refused {
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
}
