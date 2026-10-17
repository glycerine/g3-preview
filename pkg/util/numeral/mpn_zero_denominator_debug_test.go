//go:build g3debug

package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestNaturalSourceAllZeroDenominatorDebugAction(t *testing.T) {
	d, out := bindMPNDebugActions(t, "TTTC")
	serviceMust(t, d.EnableAssertions(false))
	n := naturalManager(t)
	q, r := []uint32{99}, []uint32{88}
	_, err := n.Div([]uint32{7}, []uint32{0}, q, r)
	var thrown z3exception.DefaultException
	if !errors.As(err, &thrown) || q[0] != 99 || r[0] != 88 {
		t.Fatal("debug all-zero denominator guard", err, q, r)
	}
	contents, err := os.ReadFile(out.Name())
	serviceMust(t, err)
	if !strings.Contains(string(contents), "File: src/util/mpn.cpp\nLine: 174\nUNREACHABLE CODE WAS REACHED.") {
		t.Fatal("wrong source diagnostic", string(contents))
	}
	_, err = n.Div([]uint32{7}, nil, nil, nil)
	if !errors.As(err, &thrown) {
		t.Fatal("empty denominator debug source guard", err)
	}
	serviceMust(t, d.EnableAssertions(true))
	bad := []uint32{1, 0}
	a := []uint32{7, 1}
	q, r = []uint32{99}, []uint32{88, 88}
	_, err = n.Div(a, bad, q, r)
	if !errors.As(err, &thrown) || q[0] != 99 || r[0] != 88 || r[1] != 88 {
		t.Fatal("top-limb source assertion Throw", err, q, r)
	}
	contents, err = os.ReadFile(out.Name())
	serviceMust(t, err)
	if !strings.Contains(string(contents), "File: src/util/mpn.cpp\nLine: 179\ndenom[lden-1] != 0") {
		t.Fatal("wrong top-limb assertion", string(contents))
	}
	_, err = n.Div(a, bad, q, r)
	if !errors.Is(err, arena.ErrBounds) || q[0] != 99 || r[0] != 88 || r[1] != 88 {
		t.Fatal("Continue must stop at first undefined shift", err, q, r)
	}
	serviceMust(t, d.EnableAssertions(false))
	_, err = n.Div(a, bad, q, r)
	if !errors.Is(err, arena.ErrBounds) || q[0] != 99 || r[0] != 88 || r[1] != 88 {
		t.Fatal("disabled assertion must stop at first undefined shift", err, q, r)
	}
	serviceMust(t, d.Close())
	q, r = []uint32{99}, []uint32{88}
	_, err = n.Div([]uint32{7}, []uint32{3}, q, r)
	if !errors.Is(err, arena.ErrClosed) || q[0] != 99 || r[0] != 88 {
		t.Fatal("valid denominator must read source assertion switch before output", err, q, r)
	}
}
