package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestLehmerGCDFirstSourceAssertionReadsAfterInputCopies(t *testing.T) {
	if !sourceGCDDebug {
		t.Skip("source Z3DEBUG assertion only")
	}
	d, _ := bindMPNDebugActions(t, "")
	m, _ := integerManager(t)
	var a, b Int
	serviceMust(t, m.SetLimbs(&a, []uint32{3, 2, 1}))
	serviceMust(t, m.SetLimbs(&b, []uint32{1, 1}))
	c := NewInt(99)
	serviceMust(t, d.Close())
	err := m.GCD(&a, &b, &c)
	if !errors.Is(err, arena.ErrClosed) || c.Val != 99 || c.Cell != 0 {
		t.Fatal("first Lehmer assertion must stop before result publication", err, c)
	}
	if m.GcdRaw.Values[0].Cell == 0 || m.GcdRaw.Values[1].Cell == 0 ||
		m.GcdRaw.Values[0].Val <= 0 || m.GcdRaw.Values[1].Val <= 0 {
		t.Fatal("first assertion must follow the two positive source copies", m.GcdRaw.Values[0], m.GcdRaw.Values[1])
	}
}
