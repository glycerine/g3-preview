package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntegerSignCellSourceAssertionActions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	d, log := bindMPNDebugActions(t, "TC")
	m, _ := integerManager(t)
	arg := m.Arg[0]
	m.writeDigit(arg, 0, 99)
	m.ResizeCell(arg, 2)
	a := NewInt(7)
	if _, _, err := m.signCell(&a, 0); !isMPNVerificationThrow(err) || m.Digit(arg, 0) != 99 {
		t.Fatal("Throw must precede argument digit mutation", err)
	}
	sign, cell, err := m.signCell(&a, 0)
	if err != nil || sign != 1 || cell != arg || m.Digit(arg, 0) != 7 || m.Cell(arg).Size != 2 {
		t.Fatal("Continue must reach original argument digit write", sign, cell, err)
	}
	serviceMust(t, d.EnableAssertions(false))
	a = NewInt(-8)
	sign, cell, err = m.signCell(&a, 0)
	if err != nil || sign != -1 || cell != arg || m.Digit(arg, 0) != 8 {
		t.Fatal("disabled source assertion", sign, cell, err)
	}
	contents, err := os.ReadFile(log.Name())
	serviceMust(t, err)
	if n := strings.Count(string(contents), "File: src/util/mpz.h\nLine: 253\ncell->m_size == 1\n"); n != 2 {
		t.Fatal("source argument assertion location and count", n, string(contents))
	}
	m.ResizeCell(arg, 1)
	serviceMust(t, d.Close())
	if _, _, err := m.signCell(&a, 0); !errors.Is(err, arena.ErrClosed) || m.Digit(arg, 0) != 8 {
		t.Fatal("true source assertion must read closed owner before mutation", err)
	}
}

func TestIntegerSignCellSourceAssertionCallerTransport(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	_, log := bindMPNDebugActions(t, "TTTT")
	m, _ := integerManager(t)
	var big Int
	serviceMust(t, m.SetLimbs(&big, []uint32{0, 1}))
	m.ResizeCell(m.Arg[0], 2)
	small, out, rem := NewInt(7), NewInt(99), NewInt(88)
	if _, err := m.Compare(&small, &big); !isMPNVerificationThrow(err) {
		t.Fatal("compare must transport source argument assertion", err)
	}
	if err := m.Add(&small, &big, &out); !isMPNVerificationThrow(err) || out != NewInt(99) {
		t.Fatal("add must stop before output mutation", err, out)
	}
	if err := m.Mul(&small, &big, &out); !isMPNVerificationThrow(err) || out != NewInt(99) {
		t.Fatal("multiply must stop before output mutation", err, out)
	}
	if err := m.MachineDivRem(&small, &big, &out, &rem); !isMPNVerificationThrow(err) || out != NewInt(99) || rem != NewInt(88) {
		t.Fatal("division must stop before output mutation", err, out, rem)
	}
	contents, err := os.ReadFile(log.Name())
	serviceMust(t, err)
	if n := strings.Count(string(contents), "File: src/util/mpz.h\nLine: 253\ncell->m_size == 1\n"); n != 4 {
		t.Fatal("caller source assertion count", n, string(contents))
	}
}
