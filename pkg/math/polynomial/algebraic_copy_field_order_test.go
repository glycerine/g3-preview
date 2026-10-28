package polynomial

// Source: algebraic_numbers.cpp:467-474. Raw source prefixes exercise reached
// reads only; they do not establish construction or ownership of source cells.

import (
	"errors"
	"fmt"
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicCopySourceFieldsFollowCompletedCallees(t *testing.T) {
	for _, foreign := range []bool{false, true} {
		for _, bounds := range []bool{false, true} {
			for dk := 0; dk < 3; dk++ {
				t.Run(fmt.Sprintf("foreign=%v/bounds=%v/dst=%d", foreign, bounds, dk), func(t *testing.T) {
					target, _, _ := algebraicCompareFixtureProfile(t, true)
					source := target
					if foreign {
						source, _, _ = algebraicCompareFixtureProfile(t, true)
					}
					p := source.config.Polynomials
					poly, e := p.NewPolynomial()
					serviceMust(t, e)
					serviceMust(t, p.SetInts(poly, []unumeral.Int{unumeral.NewInt(-2), unumeral.NewInt(0), unumeral.NewInt(1)}))
					src := algebraicQueryRawCell(t, source, 1)
					l, e := source.location(&src)
					serviceMust(t, e)
					words := []uint32{uint32(poly.origin), uint32(poly.origin >> 32), poly.slot, poly.generation}
					for i, w := range words {
						serviceMust(t, source.config.Allocator.WriteUint32(l.raw, uint64(i)*4, w))
					}
					serviceMust(t, source.config.Allocator.WriteUint32(l.raw, 20, 3))
					if !bounds {
						serviceMust(t, source.config.Allocator.WriteUint32(l.raw, 16, 0))
					}
					if bounds {
						for i, w := range []uint32{1, 0, 0, 0, 2, 0, 0, 0} {
							serviceMust(t, source.config.Allocator.WriteUint32(l.raw, 24+uint64(i)*4, w))
						}
					}
					dst := algebraicForeignValue(t, target, dk)
					var expectedFlags uint32
					if dk == 2 {
						dl, e := target.location(&dst)
						serviceMust(t, e)
						r, e := target.load(dl)
						serviceMust(t, e)
						expectedFlags = r.flags
					}
					body := target.CopyFromCore(&dst, source, &src)
					if !errors.Is(body, umemory.ErrRuntimeUninitialized) {
						t.Error("copy did not reach missing source member", body)
					}
					serviceMust(t, target.CleanupRetained())
					dl, e := target.location(&dst)
					serviceMust(t, e)
					if dl == nil {
						t.Fatal("copy failed before source's published destination")
					}
					out, e := target.load(dl)
					serviceMust(t, e)
					if out.poly.slot == 0 || out.size != 3 || dl.coefficientInitialized != 3 {
						t.Error("source flags/interval were read before coefficient copy", out.size, dl.coefficientInitialized)
					}
					if bounds && (out.interval.BorrowLower().BorrowNumerator().Val != 1 || out.interval.BorrowUpper().BorrowNumerator().Val != 2) {
						t.Error("source flags were read before bound SET")
					}
					if out.flags != expectedFlags {
						t.Error("copy assigned flags before completing interval", out.flags, expectedFlags)
					}
					missing := uint64(24)
					if bounds {
						missing = 16
					}
					if _, e := source.config.Allocator.ReadUint32(l.raw, missing); !errors.Is(e, umemory.ErrRuntimeUninitialized) {
						t.Fatal("missing source field was initialized", missing, e)
					}
					serviceMust(t, target.CleanupValue(&dst))
					serviceMust(t, p.DelPolynomial(&poly))
				})
			}
		}
	}
}
