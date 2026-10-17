package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestMPZScalarGetterSourceSwitchReadAndDisabledConditions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	t.Run("closed-owner", func(t *testing.T) {
		d, _ := bindMPNDebugActions(t, "")
		m, _ := integerManager(t)
		var big Int
		serviceMust(t, m.SetLimbs(&big, []uint32{1, 1}))
		small := NewInt(7)
		serviceMust(t, d.Close())
		if v, e := m.Uint64(&small); e != nil || v != 7 {
			t.Fatal("small unsigned getter bypasses source assertion", v, e)
		}
		if v, e := m.Int64(&small); e != nil || v != 7 {
			t.Fatal("small signed getter bypasses source assertion", v, e)
		}
		if _, e := m.Uint64(&big); !errors.Is(e, arena.ErrClosed) {
			t.Fatal("big unsigned getter reads source switch", e)
		}
		if _, e := m.Int64(&big); !errors.Is(e, arena.ErrClosed) {
			t.Fatal("big signed getter reads source switch", e)
		}
		if _, e := m.IsInt64(&big); !errors.Is(e, arena.ErrClosed) {
			t.Fatal("absolute-conversion helper reads its own source switch", e)
		}
	})
	t.Run("disabled", func(t *testing.T) {
		d, _ := bindMPNDebugActions(t, "")
		m, _ := integerManager(t)
		var big Int
		serviceMust(t, m.SetLimbs(&big, []uint32{5, 0, 1}))
		serviceMust(t, d.EnableAssertions(false))
		if v, e := m.Int64(&big); e != nil || v != 5 {
			t.Fatal("disabled signed fit assertion leaves source low-word cast", v, e)
		}
		cell := m.Cell(big.Cell)
		original := cell
		cell.Size = 0
		m.Store(big.Cell, cell)
		if v, e := m.Uint64(&big); e != nil || v != 5 {
			t.Fatal("disabled unsigned size assertion reads retained physical words", v, e)
		}
		m.Store(big.Cell, original)
	})
}

func TestMPZScalarGetterSourceThrowAndContinue(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	for _, tc := range []struct {
		name, commands, location string
		zeroSize                 bool
		signed                   bool
	}{
		{"uint-zero-size", "TC", "File: src/util/mpz.cpp\nLine: 1314\na.m_ptr->m_size > 0\n", true, false},
		{"uint-too-wide", "TC", "File: src/util/mpz.h\nLine: 232\nis_abs_uint64(a)\n", false, false},
		{"int-too-wide", "TCC", "File: src/util/mpz.cpp\nLine: 1339\nis_int64(a)\n", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, out := bindMPNDebugActions(t, tc.commands)
			m, _ := integerManager(t)
			var a Int
			serviceMust(t, m.SetLimbs(&a, []uint32{5, 0, 1}))
			original := m.Cell(a.Cell)
			if tc.zeroSize {
				cell := original
				cell.Size = 0
				m.Store(a.Cell, cell)
			}
			call := func() (uint64, error) {
				if tc.signed {
					v, e := m.Int64(&a)
					return uint64(v), e
				}
				return m.Uint64(&a)
			}
			if _, err := call(); !isMPNVerificationThrow(err) {
				t.Fatal("false source assertion must use Throw", err)
			}
			if v, err := call(); err != nil || v != 5 {
				t.Fatal("source Continue must reach low-word conversion", v, err)
			}
			contents, err := os.ReadFile(out.Name())
			serviceMust(t, err)
			if got := strings.Count(string(contents), tc.location); got != 2 {
				t.Fatal("source assertion location/action count", got, string(contents))
			}
			m.Store(a.Cell, original)
		})
	}
}
