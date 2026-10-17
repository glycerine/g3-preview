package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntegerRootEntrySourceAssertionActions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertion only")
	}
	d, log := bindMPNDebugActions(t, "TC")
	m, _ := integerManager(t)
	a := NewInt(-4)
	if exact, err := m.RootInPlace(&a, 2); exact || !isMPNVerificationThrow(err) || a != NewInt(-4) || m.RootRaw.active {
		t.Fatal("Throw must stop before four RAW declarations", exact, err, a)
	}
	if exact, err := m.RootInPlace(&a, 2); exact || err != nil || a != NewInt(-3) {
		t.Fatal("Continue must execute the source bisection", exact, err, a)
	}
	contents, err := os.ReadFile(log.Name())
	serviceMust(t, err)
	if n := strings.Count(string(contents), "File: src/util/mpz.cpp\nLine: 1943\nn % 2 != 0 || is_nonneg(a)\n"); n != 2 {
		t.Fatal("source location and action count", n, string(contents))
	}
	serviceMust(t, d.EnableAssertions(false))
	a = NewInt(-4)
	if exact, err := m.RootInPlace(&a, 2); exact || err != nil || a != NewInt(-3) {
		t.Fatal("disabled assertion must execute the source bisection", exact, err, a)
	}
	serviceMust(t, d.EnableAssertions(true))
	serviceMust(t, d.Close())
	a = NewInt(0)
	if exact, err := m.RootInPlace(&a, 2); exact || !errors.Is(err, arena.ErrClosed) || a != NewInt(0) || m.RootRaw.active {
		t.Fatal("closed owner must stop before zero fast path", exact, err, a)
	}
}

func TestIntegerRootNegativeEvenReleaseContinues(t *testing.T) {
	if modularSourceDebug {
		t.Skip("release assertion erasure only")
	}
	m, _ := integerManager(t)
	a := NewInt(-4)
	if exact, err := m.RootInPlace(&a, 2); exact || err != nil || a != NewInt(-3) {
		t.Fatal("release profile follows the source bisection", exact, err, a)
	}
}
