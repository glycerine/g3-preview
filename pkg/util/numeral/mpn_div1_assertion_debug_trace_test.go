//go:build g3debug && g3trace

package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestMPNDiv1ReadsDebugSwitchAfterNormalizationTrace(t *testing.T) {
	t.Run("numeric division", func(t *testing.T) {
		d, _ := bindMPNDebugActions(t, "")
		m := naturalManager(t)
		w := &mpnCloseDebugOnTrace{debug: d}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		tags.Enable("mpn_norm")
		m.trace = tags
		q, r := []uint32{99, 99}, []uint32{88}
		_, err := m.Div([]uint32{0, 1}, []uint32{3}, q, r)
		if !errors.Is(err, arena.ErrClosed) || w.closeErr != nil || w.writes == 0 || q[0] != 99 || q[1] != 99 || r[0] != 88 {
			t.Fatal("div_1 switch read before first quotient write", err, w.closeErr, w.writes, q, r)
		}
	})
	t.Run("decimal conversion", func(t *testing.T) {
		d, _ := bindMPNDebugActions(t, "")
		m := naturalManager(t)
		w := &mpnCloseDebugOnTrace{debug: d}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		tags.Enable("mpn_norm")
		m.trace = tags
		var chars [32]byte
		for i := range chars {
			chars[i] = '?'
		}
		err := m.sourceToCString(limbView{external: []uint32{0, 1}, size: 2}, sourceDecimalDestination{external: chars[:]}, uint64(len(chars)))
		if !errors.Is(err, arena.ErrClosed) || w.closeErr != nil || w.writes == 0 || chars[0] != '?' || m.Formatting.Active || m.Formatting.PendingField {
			t.Fatal("decimal div_1 switch read before character write and lexical cleanup", err, w.closeErr, w.writes, chars[:2], m.Formatting)
		}
	})
}

func TestMPNDiv1OversizedEstimateSourceActions(t *testing.T) {
	d, _ := bindMPNDebugActions(t, "TCTC")
	m := naturalManager(t)
	call := func(q []uint32) error {
		return m.divideViewOne(limbView{external: []uint32{0, 1}, size: 2}, 1, limbView{external: q, size: 1})
	}
	q := []uint32{99}
	if err := call(q); !isMPNVerificationThrow(err) || q[0] != 99 {
		t.Fatal("UNREACHABLE Throw before q_hat assertion or output", err, q)
	}
	if err := call(q); !isMPNVerificationThrow(err) || q[0] != 99 {
		t.Fatal("UNREACHABLE Continue then SASSERT Throw", err, q)
	}
	serviceMust(t, d.EnableAssertions(false))
	if err := call(q); err != nil || q[0] != 0 {
		t.Fatal("UNREACHABLE Continue with erased SASSERT reaches source cast", err, q)
	}
}
