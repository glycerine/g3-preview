package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestMPNToStringSourceEntryAssertionActions(t *testing.T) {
	if !SourceExtendedGCDDebug {
		t.Skip("source Z3DEBUG entry assertion only")
	}
	d, out := bindMPNDebugActions(t, "CT")
	m := naturalManager(t)
	a := limbView{external: []uint32{123}, size: 1}
	dst := []byte{'o', 'l', 'd', 0}
	if err := m.sourceToCString(a, sourceDecimalDestination{external: dst}, 0); err != nil || string(dst) != "old\x00" {
		t.Fatal("Continue must reach zero-size snprintf", err, dst)
	}
	if err := m.sourceToCString(a, sourceDecimalDestination{}, 1); !isMPNVerificationThrow(err) {
		t.Fatal("nil destination Throw", err)
	}
	serviceMust(t, d.EnableAssertions(false))
	if err := m.sourceToCString(a, sourceDecimalDestination{external: dst}, 0); err != nil || string(dst) != "old\x00" {
		t.Fatal("disabled entry assertion", err, dst)
	}
	contents, err := os.ReadFile(out.Name())
	serviceMust(t, err)
	if got := strings.Count(string(contents), "File: src/util/mpn.cpp\nLine: 361\nbuf && lbuf > 0\n"); got != 2 {
		t.Fatal("source entry assertion count/order", got, string(contents))
	}
	serviceMust(t, d.Close())
	valid := []byte{'?', '?', '?', '?'}
	if err := m.sourceToCString(a, sourceDecimalDestination{external: valid}, uint64(len(valid))); !errors.Is(err, arena.ErrClosed) || string(valid) != "????" {
		t.Fatal("valid conversion must read source assertion switch", err, valid)
	}
}
