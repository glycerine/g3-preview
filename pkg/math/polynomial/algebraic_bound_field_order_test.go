// Raw root views establish readable fields, never source construction/ownership.
// Source: algebraic_numbers.cpp:2987-3023; mpbq.h SET.
package polynomial

import (
	"errors"
	"fmt"
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func algebraicBoundRawPrefix(t *testing.T, c *AlgebraicRootCore, upper bool, parts int) Algebraic {
	t.Helper()
	a := algebraicQueryRawCell(t, c, 1)
	l, e := c.location(&a)
	serviceMust(t, e)
	// Literal predicate-cache data; this is not a constructor/ownership marker.
	serviceMust(t, c.config.Allocator.WriteUint32(l.raw, 16, 4))
	off := uint64(24)
	n := uint32(7)
	if upper {
		off = 40
		n = 9
	}
	if parts > 0 {
		for i, w := range []uint32{n, 0, 0} {
			serviceMust(t, c.config.Allocator.WriteUint32(l.raw, off+4*uint64(i), w))
		}
	}
	if parts > 1 {
		serviceMust(t, c.config.Allocator.WriteUint32(l.raw, off+12, 3))
	}
	return a
}
func TestAlgebraicBinaryBoundReadsSelectedNUMBeforeExponent(t *testing.T) {
	for _, upper := range []bool{false, true} {
		for parts := 0; parts <= 2; parts++ {
			t.Run(fmt.Sprintf("upper=%v/parts=%d", upper, parts), func(t *testing.T) {
				c, _, _ := algebraicCompareFixtureProfile(t, true)
				a := algebraicBoundRawPrefix(t, c, upper, parts)
				out := unumeral.NewBinaryIntPower(99, 4)
				var e error
				if upper {
					e = c.GetUpper(&a, &out)
				} else {
					e = c.GetLower(&a, &out)
				}
				if parts < 2 && !errors.Is(e, umemory.ErrRuntimeUninitialized) {
					t.Error("missing bound field guard", e)
				}
				if parts == 2 && e != nil {
					t.Error("selected bound read unrelated fields", e)
				}
				num, k := int32(99), uint32(4)
				if parts > 0 {
					num = 7
					if upper {
						num = 9
					}
				}
				if parts == 2 {
					k = 3
				}
				if out.Num.Val != num || out.KField != k {
					t.Error("NUM SET/exponent publication order", out, num, k)
				}
				serviceMust(t, c.CleanupRetained())
				serviceMust(t, c.config.Binary.Del(&out))
			})
		}
	}
}
func TestAlgebraicRawRationalBoundReadsOnlySelectedBound(t *testing.T) {
	for _, upper := range []bool{false, true} {
		t.Run(fmt.Sprint(upper), func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			a := algebraicBoundRawPrefix(t, c, upper, 2)
			q := c.config.Rationals
			out := unumeral.NewRational()
			serviceMust(t, q.SetFraction(&out, 99, 5))
			var e error
			if upper {
				e = c.GetUpperRational(&a, &out)
			} else {
				e = c.GetLowerRational(&a, &out)
			}
			if e != nil {
				t.Error("raw bound read unrelated fields", e)
			} else {
				n := int32(7)
				if upper {
					n = 9
				}
				if out.Num.Val != n || out.Den.Val != 8 {
					t.Error("raw bound conversion", out)
				}
			}
			serviceMust(t, c.CleanupRetained())
			serviceMust(t, q.Del(&out))
		})
	}
}

func TestAlgebraicPrecisionBasicBoundUsesBasicValueReference(t *testing.T) {
	for _, upper := range []bool{false, true} {
		t.Run(fmt.Sprint(upper), func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			a := basicValueRawPrefix(t, c, 1)
			q := c.config.Rationals
			out := unumeral.NewRational()
			serviceMust(t, q.SetFraction(&out, 99, 5))
			var e error
			if upper {
				e = c.GetUpperRationalAtPrecision(&a, &out, 3)
			} else {
				e = c.GetLowerRationalAtPrecision(&a, &out, 3)
			}
			if !errors.Is(e, umemory.ErrRuntimeUninitialized) || out.Num.Val != 7 || out.Den.Val != 5 {
				t.Error("precision basic branch preread DEN", e, out)
			}
			serviceMust(t, c.CleanupRetained())
			serviceMust(t, q.Del(&out))
		})
	}
}
func TestAlgebraicIntervalFromCellCopiesBoundsBeforePolynomial(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	a := algebraicQueryRawCell(t, c, 1)
	l, e := c.location(&a)
	serviceMust(t, e)
	for i, w := range []uint32{1, 0, 0, 0, 2, 0, 0, 0} {
		serviceMust(t, c.config.Allocator.WriteUint32(l.raw, 24+4*uint64(i), w))
	}
	lower, upper := unumeral.NewBinaryInt(99), unumeral.NewBinaryInt(101)
	var receipt unumeral.IntegerCopyAllocationReceipt
	_, body := c.copyIntervalAtPrecisionFromCell(l, &lower, &upper, 0, &receipt)
	if !errors.Is(body, umemory.ErrRuntimeUninitialized) || lower.Num.Val != 1 || upper.Num.Val != 2 || lower.KField != 0 || upper.KField != 0 {
		t.Error("get_interval read polynomial before bound SETs", body, lower, upper)
	}
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, c.config.Binary.Del(&lower))
	serviceMust(t, c.config.Binary.Del(&upper))
}
func TestAlgebraicPrecisionRootBoundDoesNotReadPackedFlags(t *testing.T) {
	for _, upper := range []bool{false, true} {
		t.Run(fmt.Sprint(upper), func(t *testing.T) {
			c, p, _ := algebraicCompareFixtureProfile(t, true)
			poly, e := p.NewPolynomial()
			serviceMust(t, e)
			serviceMust(t, p.SetInts(poly, []unumeral.Int{unumeral.NewInt(-2), unumeral.NewInt(0), unumeral.NewInt(1)}))
			a := algebraicQueryRawCell(t, c, 1)
			l, e := c.location(&a)
			serviceMust(t, e)
			for i, w := range []uint32{uint32(poly.origin), uint32(poly.origin >> 32), poly.slot, poly.generation} {
				serviceMust(t, c.config.Allocator.WriteUint32(l.raw, 4*uint64(i), w))
			}
			serviceMust(t, c.config.Allocator.WriteUint32(l.raw, 20, 3))
			for i, w := range []uint32{1, 0, 0, 0, 2, 0, 0, 0} {
				serviceMust(t, c.config.Allocator.WriteUint32(l.raw, 24+4*uint64(i), w))
			}
			q := c.config.Rationals
			out := unumeral.NewRational()
			if upper {
				e = c.GetUpperRationalAtPrecision(&a, &out, 0)
			} else {
				e = c.GetLowerRationalAtPrecision(&a, &out, 0)
			}
			if e != nil {
				t.Error("precision root read unrelated packed flags", e)
			} else {
				n, d := int32(1), int32(1)
				if upper {
					n = 3
					d = 2
				}
				if out.Num.Val != n || out.Den.Val != d {
					t.Error("literal one-step sqrt2 interval", out, n, d)
				}
			}
			if _, e := c.config.Allocator.ReadUint32(l.raw, 16); !errors.Is(e, umemory.ErrRuntimeUninitialized) {
				t.Error("fixture initialized packed flags", e)
			}
			serviceMust(t, c.CleanupRetained())
			serviceMust(t, q.Del(&out))
			serviceMust(t, p.DelPolynomial(&poly))
		})
	}
}
