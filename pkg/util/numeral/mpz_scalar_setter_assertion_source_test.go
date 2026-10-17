package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntegerBigScalarSetterSourceAssertionActions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	for _, unsigned := range []bool{false, true} {
		name, line := "signed", "170"
		if unsigned {
			name, line = "unsigned", "220"
		}
		t.Run(name, func(t *testing.T) {
			d, log := bindMPNDebugActions(t, "TC")
			m, _ := integerManager(t)
			var a Int
			serviceMust(t, m.SetInt64(&a, -(1<<40)))
			ref := a.Cell
			original := m.Cell(ref)
			forged := original
			forged.Capacity = 5 // the physical six-word payload remains accessible
			m.Store(ref, forged)
			t.Cleanup(func() { m.Store(ref, original) })
			call := func() error {
				if unsigned {
					return m.SetUint64(&a, ^uint64(0))
				}
				return m.SetInt64(&a, 1<<40)
			}
			if err := call(); !isMPNVerificationThrow(err) || a.Cell != ref || a.Val != -1 || m.Digit(ref, 0) != 0 || m.Digit(ref, 1) != 256 {
				t.Fatal("Throw must preserve the old big value", err, a)
			}
			if err := call(); err != nil || a.Cell != ref || a.Val != 1 {
				t.Fatal("Continue must reach source sign and digit writes", err, a)
			}
			if unsigned {
				if m.Digit(ref, 0) != ^uint32(0) || m.Digit(ref, 1) != ^uint32(0) {
					t.Fatal("unsigned continued digits")
				}
			} else if m.Digit(ref, 0) != 0 || m.Digit(ref, 1) != 256 {
				t.Fatal("signed continued digits")
			}
			serviceMust(t, d.EnableAssertions(false))
			if err := call(); err != nil {
				t.Fatal("disabled source assertion must reach digit writes", err)
			}
			contents, err := os.ReadFile(log.Name())
			serviceMust(t, err)
			if n := strings.Count(string(contents), "File: src/util/mpz.cpp\nLine: "+line+"\ncapacity(c) >= m_init_cell_capacity\n"); n != 2 {
				t.Fatal("source setter location and action count", n, string(contents))
			}
		})
		t.Run(name+"-closed-existing-cell", func(t *testing.T) {
			d, _ := bindMPNDebugActions(t, "")
			m, _ := integerManager(t)
			var a Int
			serviceMust(t, m.SetInt64(&a, -(1<<40)))
			ref := a.Cell
			serviceMust(t, d.Close())
			var err error
			if unsigned {
				err = m.SetUint64(&a, 1<<40)
			} else {
				err = m.SetInt64(&a, 1<<40)
			}
			if !errors.Is(err, arena.ErrClosed) || a.Cell != ref || a.Val != -1 {
				t.Fatal("setter assertion must stop before sign and digit writes", err, a)
			}
			if cell := m.Cell(ref); cell.Size != 2 || cell.Capacity != 6 || m.Digit(ref, 0) != 0 || m.Digit(ref, 1) != 256 {
				t.Fatal("old big setter cell changed", cell)
			}
		})
	}
}
