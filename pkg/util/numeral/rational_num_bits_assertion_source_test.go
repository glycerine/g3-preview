package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalNumBitsSourceAssertionActions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	t.Run("integer", func(t *testing.T) {
		d, log := bindMPNDebugActions(t, "T")
		s, _ := serviceNew(t)
		a, e := s.NewValue()
		serviceMust(t, e)
		serviceMust(t, s.SetFraction(&a, 1, 2))
		if _, e := s.NumBits(&a); !isMPNVerificationThrow(e) {
			t.Fatal("fractional input stops at first assertion", e)
		}
		contents, e := os.ReadFile(log.Name())
		serviceMust(t, e)
		if n := strings.Count(string(contents), "File: src/util/rational.h\nLine: 402\nis_int()\n"); n != 1 {
			t.Fatal("first source assertion site", n, string(contents))
		}
		serviceMust(t, d.EnableAssertions(false))
		if v, e := s.NumBits(&a); e != nil || v != 1 {
			t.Fatal("disabled assertion reaches source quotient loop", v, e)
		}
		serviceMust(t, s.Del(&a))
	})
	t.Run("nonnegative", func(t *testing.T) {
		d, log := bindMPNDebugActions(t, "TC")
		s, _ := serviceNew(t)
		a, e := s.NewValue()
		serviceMust(t, e)
		serviceMust(t, s.SetInt64(&a, -1))
		if _, e := s.NumBits(&a); !isMPNVerificationThrow(e) {
			t.Fatal("negative input stops at second assertion", e)
		}
		if v, e := s.NumBits(&a); e != nil || v != 1 {
			t.Fatal("Continue reaches source quotient loop", v, e)
		}
		contents, e := os.ReadFile(log.Name())
		serviceMust(t, e)
		if n := strings.Count(string(contents), "File: src/util/rational.h\nLine: 403\n!is_neg()\n"); n != 2 {
			t.Fatal("second source assertion site", n, string(contents))
		}
		serviceMust(t, s.SetInt64(&a, 7))
		serviceMust(t, d.Close())
		if _, e := s.NumBits(&a); !errors.Is(e, arena.ErrClosed) {
			t.Fatal("valid input reads closed assertion owner", e)
		}
		serviceMust(t, s.Del(&a))
	})
}
