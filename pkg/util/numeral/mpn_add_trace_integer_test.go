package numeral

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestMPNAddLiveIntegerSourceTraceOrder(t *testing.T) {
	m, _ := integerManager(t)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("mpn")
	serviceMust(t, m.BindTrace(tags))
	var a, b, c Int
	serviceMust(t, m.SetLimbs(&a, []uint32{0, 1}))
	serviceMust(t, m.SetLimbs(&b, []uint32{1, 1}))
	serviceMust(t, m.Add(&a, &b, &c))
	if got := integerString(t, m, &c); got != "8589934593" {
		t.Fatal("live integer result", got)
	}
	if !SourceExtendedGCDDebug || !tags.ShouldTrace("mpn") {
		if output.Len() != 0 {
			t.Fatal("erased MPN trace wrote", output.String())
		}
		return
	}
	want := "-------- [mpn] trace src/util/mpn.cpp:411 ---------\n" +
		"[mpn] 4294967296 + 4294967297 == ------------------------------------------------\n" +
		"-------- [mpn] trace_nl src/util/mpn.cpp:427 ---------\n" +
		"8589934593\n------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("live MPZ/MPN source trace = %q, want %q", got, want)
	}
}

type mpnCloseDebugOnFlush struct {
	bytes.Buffer
	debug    *diagnostic.DebugContext
	closeAt  int
	flushes  int
	closeErr error
}

func (w *mpnCloseDebugOnFlush) Flush() error {
	w.flushes++
	at := w.closeAt
	if at == 0 {
		at = 1
	}
	if w.flushes == at {
		w.closeErr = w.debug.Close()
	}
	return nil
}

func TestMPNAddResultTracePrecedesMPZAssertion(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	d, _ := bindMPNDebugActions(t, "")
	m, _ := integerManager(t)
	w := &mpnCloseDebugOnFlush{debug: d, closeAt: 2}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("mpn")
	if !tags.ShouldTrace("mpn") {
		t.Skip("source MPN TRACE erased")
	}
	serviceMust(t, m.BindTrace(tags))
	var a, b Int
	serviceMust(t, m.SetLimbs(&a, []uint32{0, 1}))
	serviceMust(t, m.SetLimbs(&b, []uint32{1, 1}))
	c := NewInt(99)
	err := m.Add(&a, &b, &c)
	if !errors.Is(err, arena.ErrClosed) || w.closeErr != nil || w.flushes < 2 || c.Val != 99 || c.Cell != 0 ||
		m.Digit(m.Tmp[0], 0) != 1 || m.Digit(m.Tmp[0], 1) != 2 {
		t.Fatal("result TRACE must precede MPZ assertion and destination publication", err, w.closeErr, w.flushes, c)
	}
}

func TestMPNAddOperandTracePrecedesSourceAssertion(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	d, _ := bindMPNDebugActions(t, "")
	m, _ := integerManager(t)
	w := &mpnCloseDebugOnFlush{debug: d}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("mpn")
	if !tags.ShouldTrace("mpn") {
		t.Skip("source MPN TRACE erased")
	}
	serviceMust(t, m.BindTrace(tags))
	var a, b Int
	serviceMust(t, m.SetLimbs(&a, []uint32{0, 1}))
	serviceMust(t, m.SetLimbs(&b, []uint32{1, 1}))
	c := NewInt(99)
	err := m.Add(&a, &b, &c)
	if !errors.Is(err, arena.ErrClosed) || w.closeErr != nil || w.flushes == 0 || c.Val != 99 || c.Cell != 0 || m.Cell(m.Tmp[0]).span != 3 {
		t.Fatal("operand TRACE must precede first add assertion and output", err, w.closeErr, w.flushes, c, m.Cell(m.Tmp[0]))
	}
}
