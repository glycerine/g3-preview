package numeral

import (
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntegerEnsureTmpPostAllocationSourceAssertionActions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	d, log := bindMPNDebugActions(t, "TC")
	m, _ := integerManager(t)
	const request = uint32(1431655773) // source unsigned32 growth wraps to 12 words
	old := m.Tmp[0]
	if err := m.EnsureTmp(0, request); !isMPNVerificationThrow(err) || m.Tmp[0] == 0 || m.Tmp[0] == old || m.Cell(m.Tmp[0]).Capacity != 12 {
		t.Fatal("Throw must retain the published replacement", err, old, m.Tmp[0])
	}
	first := m.Tmp[0]
	if err := m.EnsureTmp(0, request); err != nil || m.Tmp[0] == 0 || m.Tmp[0] == first || m.Cell(m.Tmp[0]).Capacity != 12 {
		t.Fatal("Continue must return after the published replacement", err, first, m.Tmp[0])
	}
	serviceMust(t, d.EnableAssertions(false))
	if err := m.EnsureTmp(0, request); err != nil || m.Tmp[0] == 0 || m.Cell(m.Tmp[0]).Capacity != 12 {
		t.Fatal("disabled source assertion must keep the undersized replacement", err, m.Tmp[0])
	}
	contents, err := os.ReadFile(log.Name())
	serviceMust(t, err)
	if n := strings.Count(string(contents), "File: src/util/mpz.h\nLine: 162\nm_tmp[IDX]->m_capacity >= capacity\n"); n != 2 {
		t.Fatal("post-allocation source assertion location and count", n, string(contents))
	}
	if err := m.prepareTmpSpan(0, request); err != arena.ErrBounds {
		t.Fatal("physical span guard still applies at actual access", err)
	}
}
