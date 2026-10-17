package numeral

import (
	"errors"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestMPZ32BitGetterSourceThrowContinueAndDisabled(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	d, out := bindMPNDebugActions(t, "TCTC")
	m, _ := integerManager(t)
	signed := NewInt(math.MaxInt32)
	unsigned := NewInt(-1)
	if _, err := m.Int32(&signed); !isMPNVerificationThrow(err) {
		t.Fatal("get_int strict fit must use source Throw", err)
	}
	if v, err := m.Int32(&signed); err != nil || v != math.MaxInt32 {
		t.Fatal("get_int Continue must reach narrowing cast", v, err)
	}
	if _, err := m.Uint32(&unsigned); !isMPNVerificationThrow(err) {
		t.Fatal("get_uint negative input must use source Throw", err)
	}
	if v, err := m.Uint32(&unsigned); err != nil || v != math.MaxUint32 {
		t.Fatal("get_uint Continue must reach unsigned cast", v, err)
	}
	contents, err := os.ReadFile(out.Name())
	serviceMust(t, err)
	for _, location := range []string{
		"File: src/util/mpz.h\nLine: 742\nis_int(a)\n",
		"File: src/util/mpz.h\nLine: 738\nis_uint(a)\n",
	} {
		if got := strings.Count(string(contents), location); got != 2 {
			t.Fatal("source 32-bit getter assertion location", got, string(contents))
		}
	}
	serviceMust(t, d.EnableAssertions(false))
	if v, err := m.Int32(&signed); err != nil || v != math.MaxInt32 {
		t.Fatal("disabled signed assertion must reach cast", v, err)
	}
	if v, err := m.Uint32(&unsigned); err != nil || v != math.MaxUint32 {
		t.Fatal("disabled unsigned assertion must reach cast", v, err)
	}
	serviceMust(t, d.Close())
	valid := NewInt(7)
	if _, err := m.Int32(&valid); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("true signed assertion still reads closed source owner", err)
	}
	if _, err := m.Uint32(&valid); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("true unsigned assertion still reads closed source owner", err)
	}
}
