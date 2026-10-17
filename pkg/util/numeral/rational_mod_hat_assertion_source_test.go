package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalModHatSourceAssertionActions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	d, log := bindMPNDebugActions(t, "TC")
	s, _ := serviceNew(t)
	a, e := s.NewValue()
	serviceMust(t, e)
	b, e := s.NewValue()
	serviceMust(t, e)
	serviceMust(t, s.SetInt64(&a, 5))
	serviceMust(t, s.SetInt64(&b, -3))
	if _, e := s.ModHat(&a, &b); !isMPNVerificationThrow(e) {
		t.Fatal("nonpositive divisor stops before mod", e)
	}
	if r, e := s.ModHat(&a, &b); e != nil {
		t.Fatal("Continue reaches source mod and second assertion", e)
	} else {
		serviceText(t, s, &r, "5")
		serviceMust(t, s.Del(&r))
	}
	contents, e := os.ReadFile(log.Name())
	serviceMust(t, e)
	if n := strings.Count(string(contents), "File: src/util/rational.h\nLine: 209\nb.is_pos()\n"); n != 2 {
		t.Fatal("first source assertion site", n, string(contents))
	}
	serviceMust(t, d.EnableAssertions(false))
	if r, e := s.ModHat(&a, &b); e != nil {
		t.Fatal("disabled assertion reaches source mod", e)
	} else {
		serviceText(t, s, &r, "5")
		serviceMust(t, s.Del(&r))
	}
	serviceMust(t, s.SetInt64(&b, 3))
	serviceMust(t, d.Close())
	if _, e := s.ModHat(&a, &b); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("valid divisor reads closed assertion owner", e)
	}
	serviceMust(t, s.Del(&a))
	serviceMust(t, s.Del(&b))
}
