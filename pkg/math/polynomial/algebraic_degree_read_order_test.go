package polynomial

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

// A raw object prefix exercises member-read guards; this does not claim that
// its C++ placement constructor or any numeral/polynomial member completed.
func algebraicQueryRawCell(t *testing.T, c *AlgebraicRootCore, kind uint8) Algebraic {
	t.Helper()
	size := uint64(24)
	if kind == 1 {
		size = 56
	}
	raw, e := c.config.Allocator.Allocate(size)
	serviceMust(t, e)
	c.locations = append(c.locations, algebraicLocation{raw: raw, kind: kind, generation: 1, live: true})
	a := Algebraic{owner: c.origin, slot: uint32(len(c.locations)), generation: 1}
	t.Cleanup(func() {
		l, e := c.locationCleanup(&a)
		serviceMust(t, e)
		if l != nil {
			serviceMust(t, c.deleteLocation(&a, l))
		}
	})
	return a
}

func TestAlgebraicDegreeReadsOnlyPolynomialSize(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	a := algebraicQueryRawCell(t, c, 1)
	l, e := c.location(&a)
	serviceMust(t, e)
	if _, e := c.Degree(&a); !errors.Is(e, umemory.ErrRuntimeUninitialized) {
		t.Fatal("missing actual size field was not reached", e)
	}
	for _, size := range []uint32{7, 1, 0} {
		serviceMust(t, c.config.Allocator.WriteUint32(l.raw, 20, size))
		degree, e := c.Degree(&a)
		serviceMust(t, e)
		if degree != size-1 {
			t.Fatal("literal unsigned polynomial size minus one", size, degree)
		}
	}
	if degree, e := c.Degree(&a); e != nil || degree != math.MaxUint32 {
		t.Fatal("source unsigned zero-size wrap", degree, e)
	}
	for _, offset := range []uint64{0, 16, 24, 40} {
		if _, e := c.config.Allocator.ReadUint32(l.raw, offset); !errors.Is(e, umemory.ErrRuntimeUninitialized) {
			t.Fatal("fixture initialized an unrelated polynomial/flag/interval field", offset, e)
		}
	}
}

func TestAlgebraicDegreeRootTagSwitchPrecedesSize(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source tag assertion erased")
	}
	c, _, _ := algebraicCompareFixture(t)
	root := algebraicQueryRawCell(t, c, 1)
	basic := algebraicQueryRawCell(t, c, 0)
	var zero Algebraic
	polynomialAssertionActionsFixture(t, "T")
	serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	if _, e := c.Degree(&root); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("to_algebraic switch did not precede uninitialized size", e)
	}
	for i, a := range []*Algebraic{&zero, &basic} {
		degree, e := c.Degree(a)
		serviceMust(t, e)
		if degree != uint32(i) {
			t.Fatal("zero/basic branch reached an unrelated assertion or field", i, degree)
		}
	}
	for _, a := range []*Algebraic{&zero, &basic, &root} {
		if _, e := c.IsPos(a); !errors.Is(e, arena.ErrClosed) {
			t.Fatal("positive query skipped the reached accessor assertion switch", e)
		}
		if _, e := c.IsNeg(a); !errors.Is(e, arena.ErrClosed) {
			t.Fatal("negative query skipped the reached accessor assertion switch", e)
		}
	}
}

func TestAlgebraicSignReadsOnlyReachedScalarFields(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	basic := algebraicQueryRawCell(t, c, 0)
	root := algebraicQueryRawCell(t, c, 1)
	bl, e := c.location(&basic)
	serviceMust(t, e)
	rl, e := c.location(&root)
	serviceMust(t, e)
	for _, value := range []int32{-7, 0, 7} {
		serviceMust(t, c.config.Allocator.WriteUint32(bl.raw, 0, uint32(value)))
		positive, e := c.IsPos(&basic)
		serviceMust(t, e)
		negative, e := c.IsNeg(&basic)
		serviceMust(t, e)
		if positive != (value > 0) || negative != (value < 0) {
			t.Fatal("literal basic numerator sign", value, positive, negative)
		}
	}
	serviceMust(t, c.config.Allocator.WriteUint32(rl.raw, 24, 0))
	positive, e := c.IsPos(&root)
	serviceMust(t, e)
	if !positive {
		t.Fatal("open positive interval permits zero lower endpoint")
	}
	if _, e := c.IsNeg(&root); !errors.Is(e, umemory.ErrRuntimeUninitialized) {
		t.Fatal("negative query did not reach its own upper sign field", e)
	}
	for _, endpoints := range [][2]int32{{0, 1}, {-1, 0}, {-1, 1}, {0, 0}} {
		serviceMust(t, c.config.Allocator.WriteUint32(rl.raw, 24, uint32(endpoints[0])))
		serviceMust(t, c.config.Allocator.WriteUint32(rl.raw, 40, uint32(endpoints[1])))
		positive, e := c.IsPos(&root)
		serviceMust(t, e)
		negative, e := c.IsNeg(&root)
		serviceMust(t, e)
		if positive != (endpoints[0] >= 0) || negative != (endpoints[1] <= 0) {
			t.Fatal("literal open interval endpoint sign", endpoints, positive, negative)
		}
	}
	for _, offset := range []uint64{0, 16, 20, 28, 36, 44, 52} {
		if _, e := c.config.Allocator.ReadUint32(rl.raw, offset); !errors.Is(e, umemory.ErrRuntimeUninitialized) {
			t.Fatal("fixture initialized an unrelated root record field", offset, e)
		}
	}
}

func TestAlgebraicPredicatesReadCachedIrrationalFlagFirst(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	a := algebraicQueryRawCell(t, c, 1)
	l, e := c.location(&a)
	serviceMust(t, e)
	for _, query := range []func(*Algebraic) (bool, error){c.IsInt, c.IsRational} {
		if _, e := query(&a); !errors.Is(e, umemory.ErrRuntimeUninitialized) {
			t.Fatal("missing cached flag was not reached", e)
		}
	}
	serviceMust(t, c.config.Allocator.WriteUint32(l.raw, 16, 4))
	for _, query := range []func(*Algebraic) (bool, error){c.IsInt, c.IsRational} {
		v, e := query(&a)
		serviceMust(t, e)
		if v || c.predicate.active {
			t.Fatal("known irrational entered refinement", v)
		}
	}
	for _, offset := range []uint64{0, 20, 24, 40} {
		if _, e := c.config.Allocator.ReadUint32(l.raw, offset); !errors.Is(e, umemory.ErrRuntimeUninitialized) {
			t.Fatal("unrelated field initialized", offset, e)
		}
	}
	if polynomialSourceDebug {
		polynomialAssertionActionsFixture(t, "T")
		serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
		for _, query := range []func(*Algebraic) (bool, error){c.IsInt, c.IsRational} {
			if _, e := query(&a); !errors.Is(e, arena.ErrClosed) {
				t.Fatal("accessor assertion switch skipped before cached flag", e)
			}
		}
	}
}
