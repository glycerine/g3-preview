package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestExtendedGCDOrderAssertionReadsAfterInitialPublication(t *testing.T) {
	if !sourceGCDDebug {
		t.Skip("source Z3DEBUG assertion only")
	}
	d, _ := bindMPNDebugActions(t, "")
	m, _ := integerManager(t)
	r1, r2 := NewInt(26), NewInt(7)
	a, b, g := NewInt(99), NewInt(99), NewInt(99)
	serviceMust(t, d.Close())
	err := m.ExtendedGCD(&r1, &r2, &a, &b, &g)
	if !errors.Is(err, arena.ErrClosed) || a.Val != 1 || b.Val != 0 || g.Val != 99 ||
		m.ExtendedRaw.Values[0].Val != 26 || m.ExtendedRaw.Values[1].Val != 7 {
		t.Fatal("source order assertion must follow coefficient initialization and positive input copies", err, a, b, g, m.ExtendedRaw.Values)
	}
}

func TestExtendedGCDOrderAssertionSourceAction(t *testing.T) {
	if !sourceGCDDebug {
		t.Skip("source Z3DEBUG assertion only")
	}
	d, out := bindMPNDebugActions(t, "T")
	m, _ := integerManager(t)
	one, two := NewInt(1), NewInt(2)
	if err := m.gcdSourceGreaterEqualAt(&one, &two, 937, "ge(tmp1, tmp2)"); !isMPNVerificationThrow(err) {
		t.Fatal("false source order assertion must use bound Throw action", err)
	}
	contents, err := os.ReadFile(out.Name())
	serviceMust(t, err)
	if !strings.Contains(string(contents), "File: src/util/mpz.cpp\nLine: 937\nge(tmp1, tmp2)\n") {
		t.Fatal("missing source assertion location", string(contents))
	}
	serviceMust(t, d.EnableAssertions(false))
	if err := m.gcdSourceGreaterEqualAt(&one, &two, 937, "ge(tmp1, tmp2)"); err != nil {
		t.Fatal("disabled source assertion must continue", err)
	}
}
