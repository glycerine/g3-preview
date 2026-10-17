package numeral

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestMPNAddSourceAssertionsThroughIntegerConsumer(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	d, out := bindMPNDebugActions(t, "T")
	m, _ := integerManager(t)
	var a, b Int
	serviceMust(t, m.SetLimbs(&a, []uint32{0, 1}))
	serviceMust(t, m.SetLimbs(&b, []uint32{0, 1}))
	c := NewInt(99)
	m.ResizeCell(a.Cell, 0)
	m.ResizeCell(b.Cell, 0)
	err := m.Add(&a, &b, &c)
	if !isMPNVerificationThrow(err) || c.Val != 99 || c.Cell != 0 || m.Cell(m.Tmp[0]).span != 1 {
		t.Fatal("source mpn add length assertion before first scratch write", err, c, m.Cell(m.Tmp[0]))
	}
	contents, err := os.ReadFile(out.Name())
	serviceMust(t, err)
	if !strings.Contains(string(contents), "File: src/util/mpn.cpp\nLine: 68\nlngc_alloc == len+1 && len > 0\n") {
		t.Fatal("wrong source add diagnostic", string(contents))
	}
	m.ResizeCell(a.Cell, 2)
	m.ResizeCell(b.Cell, 2)
	serviceMust(t, d.Close())
	err = m.Add(&a, &b, &c)
	if !errors.Is(err, arena.ErrClosed) || c.Val != 99 || c.Cell != 0 || m.Cell(m.Tmp[0]).span != 3 {
		t.Fatal("valid mpn add must read source assertion switch after scratch preparation", err, c, m.Cell(m.Tmp[0]))
	}
}
