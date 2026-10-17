package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntegerPowerTwoSourceAssertionBeforeCapacity(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	d, _ := bindMPNDebugActions(t, "")
	m, _ := integerManager(t)
	two, out := NewInt(2), NewInt(17)
	serviceMust(t, d.Close())
	err := m.Power(&two, 32, &out)
	if !errors.Is(err, arena.ErrClosed) || out != NewInt(17) {
		t.Fatal("first source assertion switch must precede capacity and output mutation", err, out)
	}
}

func TestIntegerPowerZeroSourceAssertionActions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	d, outLog := bindMPNDebugActions(t, "TC")
	m, _ := integerManager(t)
	zero, out := NewInt(0), NewInt(17)
	if err := m.Power(&zero, 0, &out); !isMPNVerificationThrow(err) || out != NewInt(17) {
		t.Fatal("Throw must preserve output", err, out)
	}
	if err := m.Power(&zero, 0, &out); err != nil || out != NewInt(0) {
		t.Fatal("Continue must reach source reset", err, out)
	}
	contents, err := os.ReadFile(outLog.Name())
	serviceMust(t, err)
	if count := strings.Count(string(contents), "File: src/util/mpz.cpp\nLine: 1483\np != 0\n"); count != 2 {
		t.Fatal("source assertion action location and count", count, string(contents))
	}
	serviceMust(t, d.EnableAssertions(false))
	out = NewInt(17)
	if err := m.Power(&zero, 0, &out); err != nil || out != NewInt(0) {
		t.Fatal("disabled source assertion", err, out)
	}
	serviceMust(t, d.Close())
	out = NewInt(17)
	if err := m.Power(&zero, 1, &out); !errors.Is(err, arena.ErrClosed) || out != NewInt(17) {
		t.Fatal("true source assertion must read bound owner", err, out)
	}
}
