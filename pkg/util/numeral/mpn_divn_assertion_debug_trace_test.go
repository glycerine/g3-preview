//go:build g3debug && g3trace

package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type mpnCloseDebugOnTrace struct {
	debug    *diagnostic.DebugContext
	writes   int
	closeErr error
}

func (w *mpnCloseDebugOnTrace) Write(p []byte) (int, error) {
	w.writes++
	if w.writes == 1 {
		w.closeErr = w.debug.Close()
	}
	return len(p), nil
}

func TestMPNDivNReadsDebugSwitchAfterNormalizationTrace(t *testing.T) {
	d, _ := bindMPNDebugActions(t, "")
	m := naturalManager(t)
	w := &mpnCloseDebugOnTrace{debug: d}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("mpn_norm")
	if !tags.ShouldTrace("mpn_norm") {
		t.Skip("source normalization TRACE erased")
	}
	m.trace = tags
	q, r := []uint32{99}, []uint32{88, 88}
	_, err := m.Div([]uint32{0, 2}, []uint32{3, 1}, q, r)
	if !errors.Is(err, arena.ErrClosed) || w.closeErr != nil || w.writes == 0 || q[0] != 99 || r[0] != 88 || r[1] != 88 || m.Division[2] != nil {
		t.Fatal("div_n assertion must read closed owner before ms resize or output", err, w.closeErr, w.writes, q, r, naturalDivisionCapacities(m))
	}
}

func TestMPNDivNReadsEstimateAssertionOnNextIteration(t *testing.T) {
	d, _ := bindMPNDebugActions(t, "")
	m := naturalManager(t)
	w := &mpnCloseDebugOnTrace{debug: d}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("mpn_div")
	if !tags.ShouldTrace("mpn_div") {
		t.Skip("source division TRACE erased")
	}
	m.trace = tags
	q, r := []uint32{99, 99}, []uint32{88, 88}
	_, err := m.Div([]uint32{5, 2, 1}, []uint32{1, 1}, q, r)
	if !errors.Is(err, arena.ErrClosed) || w.closeErr != nil || w.writes == 0 || q[0] != 99 || q[1] == 99 || r[0] != 88 || r[1] != 88 {
		t.Fatal("second quotient estimate assertion must stop before its multiplication/output", err, w.closeErr, w.writes, q, r)
	}
}

func TestMPNToStringReadsNormalizationAssertionAfterEntryTrace(t *testing.T) {
	d, _ := bindMPNDebugActions(t, "")
	m := naturalManager(t)
	w := &mpnCloseDebugOnTrace{debug: d}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("mpn_to_string")
	if !tags.ShouldTrace("mpn_to_string") {
		t.Skip("source decimal-entry TRACE erased")
	}
	m.trace = tags
	dst := []byte("unchanged\x00")
	err := m.sourceToCString(limbView{external: []uint32{0, 1}, size: 2}, sourceDecimalDestination{external: dst}, uint64(len(dst)))
	if !errors.Is(err, arena.ErrClosed) || w.closeErr != nil || w.writes == 0 || string(dst) != "unchanged\x00" || m.Formatting.Active {
		t.Fatal("normalization assertion must read closed owner before decimal writes", err, w.closeErr, w.writes, dst, m.Formatting.Active)
	}
}
