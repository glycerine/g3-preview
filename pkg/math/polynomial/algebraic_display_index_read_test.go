package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestAlgebraicCachedDisplayIndexReadsOnlyPackedFlags(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	a := algebraicQueryRawCell(t, c, 1)
	l, e := c.location(&a)
	serviceMust(t, e)
	for _, index := range []uint32{1, 7, 0x1fffffff} {
		for _, low := range []uint32{0, 7} {
			flags := index<<3 | low
			serviceMust(t, c.config.Allocator.WriteUint32(l.raw, 16, flags))
			got, e := c.displayRootIndex(l, UPolynomialView{}, 2652)
			serviceMust(t, e)
			if got != index {
				t.Fatal("cached packed index", flags, got, index)
			}
			after, e := c.config.Allocator.ReadUint32(l.raw, 16)
			serviceMust(t, e)
			if after != flags {
				t.Fatal("cached query changed unrelated flag bits", flags, after)
			}
		}
	}
	for _, offset := range []uint64{0, 4, 8, 12, 20, 24, 28, 32, 36, 40, 44, 48, 52} {
		if _, e := c.config.Allocator.ReadUint32(l.raw, offset); !errors.Is(e, umemory.ErrRuntimeUninitialized) {
			t.Fatal("unrelated root member initialized", offset, e)
		}
	}
}

func TestAlgebraicDisplayIndexFlagReadPrecedesAssertionSwitch(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertion erased")
	}
	c, _, _ := algebraicCompareFixture(t)
	a := algebraicQueryRawCell(t, c, 1)
	l, e := c.location(&a)
	serviceMust(t, e)
	polynomialAssertionActionsFixture(t, "T")
	serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	if _, e := c.displayRootIndex(l, UPolynomialView{}, 2652); !errors.Is(e, umemory.ErrRuntimeUninitialized) {
		t.Fatal("assertion switch preceded missing source index word", e)
	}
	serviceMust(t, c.config.Allocator.WriteUint32(l.raw, 16, 2<<3))
	if _, e := c.displayRootIndex(l, UPolynomialView{}, 2652); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("cached index skipped reached source assertion switch", e)
	}
}
