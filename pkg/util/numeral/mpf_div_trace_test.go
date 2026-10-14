package numeral

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestArbitraryFloatDivDebugTraceSourceOrderAndShortcut(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpf_dbg")
	compiled := tags.ShouldTrace("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var one, zero, out ArbitraryFloat
	serviceMust(t, m.MkOne(11, 53, false, &one))
	serviceMust(t, m.MkZero(11, 53, false, &zero))
	serviceMust(t, m.Div(ArbitraryFloatNearestEven, &one, &one, &out))
	if compiled {
		got := output.String()
		last := -1
		for _, site := range []struct {
			line, label string
		}{{"670", "X = "}, {"671", "Y = "}, {"722", "A = "}, {"723", "B = "}, {"727", "A' = "}, {"728", "B' = "}, {"734", "QUOTIENT = "}, {"742", "QUOTIENT' = "}} {
			entry := "-------- [mpf_dbg] div src/util/mpf.cpp:" + site.line + " ---------\n" + site.label
			at := strings.Index(got, entry)
			if at <= last {
				t.Fatalf("division debug source order at %s: %q", site.line, got)
			}
			last = at
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased division debug trace: %q", output.String())
	}
	output.Reset()
	serviceMust(t, m.Div(ArbitraryFloatNearestEven, &zero, &one, &out))
	if compiled {
		got := output.String()
		if strings.Count(got, "-------- [mpf_dbg] div src/util/mpf.cpp:") != 2 || strings.Contains(got, "QUOTIENT = ") {
			t.Fatalf("zero shortcut reached quotient debug trace: %q", got)
		}
	} else if output.Len() != 0 {
		t.Fatalf("erased shortcut debug trace: %q", output.String())
	}
	serviceMust(t, m.Del(&one))
	serviceMust(t, m.Del(&zero))
	serviceMust(t, m.Del(&out))
}

func TestArbitraryFloatDivDebugTraceEndlFlushes(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	w := &mpfEndlWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var one, zero, out ArbitraryFloat
	serviceMust(t, m.MkOne(11, 53, false, &one))
	serviceMust(t, m.MkZero(11, 53, false, &zero))
	defer func() {
		serviceMust(t, m.Del(&one))
		serviceMust(t, m.Del(&zero))
		serviceMust(t, m.Del(&out))
	}()
	for _, tc := range []struct {
		name  string
		x     *ArbitraryFloat
		sites []string
	}{
		{"general", &one, []string{"670", "671", "722", "723", "727", "728", "734", "742"}},
		{"zero shortcut", &zero, []string{"670", "671"}},
	} {
		w.events = nil
		w.flushes = 0
		serviceMust(t, m.Div(ArbitraryFloatNearestEven, tc.x, &one, &out))
		if !tags.ShouldTrace("mpf_dbg") {
			if len(w.events) != 0 || w.flushes != 0 {
				t.Fatalf("erased %s trace: %q", tc.name, w.events)
			}
			continue
		}
		var written strings.Builder
		for _, event := range w.events {
			if strings.HasPrefix(event, "write:") {
				written.WriteString(strings.TrimPrefix(event, "write:"))
			}
		}
		all := written.String()
		for _, line := range tc.sites {
			if !strings.Contains(all, "div src/util/mpf.cpp:"+line+" ---------") {
				t.Fatalf("%s missing source site %s: %q", tc.name, line, w.events)
			}
		}
		newlines, footers := 0, 0
		for i, event := range w.events {
			if event == "write:------------------------------------------------\n" {
				footers++
				if i+1 == len(w.events) || w.events[i+1] != "flush" {
					t.Fatalf("%s TRACE footer did not flush: %q", tc.name, w.events)
				}
			}
			if event != "write:\n" {
				continue
			}
			newlines++
			if i+1 == len(w.events) || w.events[i+1] != "flush" {
				t.Fatalf("%s source std::endl did not flush: %q", tc.name, w.events)
			}
		}
		if newlines < len(tc.sites) || footers < len(tc.sites) || w.flushes != newlines+footers {
			t.Fatalf("%s completed %d newlines, %d footers and %d flushes: %q", tc.name, newlines, footers, w.flushes, w.events)
		}
	}
}
