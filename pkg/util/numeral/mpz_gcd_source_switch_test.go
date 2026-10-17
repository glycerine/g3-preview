package numeral

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestLehmerGCDFirstSourceAssertionReadsAfterInputCopies(t *testing.T) {
	if !sourceGCDDebug {
		t.Skip("source Z3DEBUG assertion only")
	}
	d, _ := bindMPNDebugActions(t, "")
	m, _ := integerManager(t)
	var a, b Int
	serviceMust(t, m.SetLimbs(&a, []uint32{3, 2, 1}))
	serviceMust(t, m.SetLimbs(&b, []uint32{1, 1}))
	c := NewInt(99)
	serviceMust(t, d.Close())
	err := m.GCD(&a, &b, &c)
	if !errors.Is(err, arena.ErrClosed) || c.Val != 99 || c.Cell != 0 {
		t.Fatal("first Lehmer assertion must stop before result publication", err, c)
	}
	if m.GcdRaw.Values[0].Cell == 0 || m.GcdRaw.Values[1].Cell == 0 ||
		m.GcdRaw.Values[0].Val <= 0 || m.GcdRaw.Values[1].Val <= 0 {
		t.Fatal("first assertion must follow the two positive source copies", m.GcdRaw.Values[0], m.GcdRaw.Values[1])
	}
}

func TestLehmerGCDStructuralAssertionSourceActions(t *testing.T) {
	if !sourceGCDDebug {
		t.Skip("source Z3DEBUG assertion only")
	}
	cases := []struct {
		line       int32
		expression string
	}{
		{756, "!is_small(a1)"},
		{757, "!is_small(b1)"},
		{760, "b_sz <= a_sz"},
		{769, "a_hat + A <= static_cast<int64_t>(UINT_MAX) + 1"},
		{770, "a_hat + B < static_cast<int64_t>(UINT_MAX) + 1"},
		{771, "b_hat + C < static_cast<int64_t>(UINT_MAX) + 1"},
		{772, "b_hat + D <= static_cast<int64_t>(UINT_MAX) + 1"},
	}
	d, out := bindMPNDebugActions(t, strings.Repeat("T", len(cases)))
	for _, tc := range cases {
		if err := gcdSourceRequireAt(false, tc.line, tc.expression); !isMPNVerificationThrow(err) {
			t.Fatalf("source assertion %d action: %v", tc.line, err)
		}
	}
	contents, err := os.ReadFile(out.Name())
	serviceMust(t, err)
	for _, tc := range cases {
		want := "File: src/util/mpz.cpp\nLine: " + strconv.FormatInt(int64(tc.line), 10) + "\n" + tc.expression + "\n"
		if !strings.Contains(string(contents), want) {
			t.Fatalf("missing source assertion %d: %q", tc.line, contents)
		}
	}
	serviceMust(t, d.EnableAssertions(false))
	if err := gcdSourceRequireAt(false, 756, cases[0].expression); err != nil {
		t.Fatal("disabled assertion must continue", err)
	}
	serviceMust(t, d.Close())
	if err := gcdSourceRequireAt(true, 756, cases[0].expression); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("true assertion still reads its closed owner", err)
	}
}
