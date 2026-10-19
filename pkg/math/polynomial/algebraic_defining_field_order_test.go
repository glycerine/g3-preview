// Source: algebraic_numbers.cpp:1905-1924. Raw source views establish readable
// fields only; they own no polynomial/numeral members and claim no constructor.
package polynomial

import (
	"errors"
	"fmt"
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestAlgebraicDefiningBasicFieldsFollowNumeralSets(t *testing.T) {
	for _, complete := range []bool{false, true} {
		t.Run(fmt.Sprint(complete), func(t *testing.T) {
			c, p, _ := algebraicCompareFixtureProfile(t, true)
			a := basicValueRawPrefix(t, c, 1)
			l, e := c.location(&a)
			serviceMust(t, e)
			if complete {
				for i, w := range []uint32{3, 0, 0} {
					serviceMust(t, c.config.Allocator.WriteUint32(l.raw, 12+uint64(i)*4, w))
				}
			}
			out, e := p.NewPolynomial()
			serviceMust(t, e)
			serviceMust(t, p.SetInts(out, []unumeral.Int{unumeral.NewInt(11), unumeral.NewInt(13), unumeral.NewInt(17)}))
			body := c.GetPolynomial(&a, out)
			if complete {
				if body != nil {
					t.Error("complete source fields were not copied", body)
				}
			} else if !errors.Is(body, umemory.ErrRuntimeUninitialized) {
				t.Error("missing denominator read guard", body)
			}
			d, e := p.descriptor(out)
			serviceMust(t, e)
			values := p.values(&d.state)
			if complete {
				if d.state.size != 2 || values[0].Val != -7 || values[1].Val != 3 {
					t.Error("completed defining polynomial", d.state.size, values)
				}
			} else if d.state.size != 3 || values[0].Val != 7 || values[1].Val != 13 || values[2].Val != 17 {
				t.Error("source DEN was read before NUM SET or final negation/resize ran early", d.state.size, values)
			}
			serviceMust(t, c.CleanupRetained())
			serviceMust(t, p.DelPolynomial(&out))
		})
	}
}
func TestAlgebraicDefiningRootReadsOnlyPolynomialArguments(t *testing.T) {
	for _, alias := range []bool{false, true} {
		t.Run(fmt.Sprint(alias), func(t *testing.T) {
			c, p, _ := algebraicCompareFixtureProfile(t, true)
			input, e := p.NewPolynomial()
			serviceMust(t, e)
			serviceMust(t, p.SetInts(input, []unumeral.Int{unumeral.NewInt(-2), unumeral.NewInt(0), unumeral.NewInt(1)}))
			a := algebraicQueryRawCell(t, c, 1)
			l, e := c.location(&a)
			serviceMust(t, e)
			for i, w := range []uint32{uint32(input.origin), uint32(input.origin >> 32), input.slot, input.generation} {
				serviceMust(t, c.config.Allocator.WriteUint32(l.raw, uint64(i)*4, w))
			}
			serviceMust(t, c.config.Allocator.WriteUint32(l.raw, 20, 3))
			out := input
			if !alias {
				out, e = p.NewPolynomial()
				serviceMust(t, e)
			}
			body := c.GetPolynomial(&a, out)
			if body != nil {
				t.Error("defining copy read flags/interval or construction metadata", body)
			} else {
				d, e := p.descriptor(out)
				serviceMust(t, e)
				v := p.values(&d.state)
				if d.state.size != 3 || v[0].Val != -2 || v[1].Val != 0 || v[2].Val != 1 {
					t.Error("defining coefficient copy", d.state.size, v)
				}
			}
			for _, offset := range []uint64{16, 24, 40} {
				if _, e := c.config.Allocator.ReadUint32(l.raw, offset); !errors.Is(e, umemory.ErrRuntimeUninitialized) {
					t.Error("unrelated source field initialized", offset, e)
				}
			}
			serviceMust(t, c.CleanupRetained())
			if !alias {
				serviceMust(t, p.DelPolynomial(&out))
			}
			serviceMust(t, p.DelPolynomial(&input))
		})
	}
}

func TestAlgebraicDefiningRootAliasUsesCalleeAssertionActions(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions erased")
	}
	c, p, _ := algebraicCompareFixtureProfile(t, true)
	input, e := p.NewPolynomial()
	serviceMust(t, e)
	serviceMust(t, p.SetInts(input, []unumeral.Int{unumeral.NewInt(-2), unumeral.NewInt(0), unumeral.NewInt(1)}))
	a := algebraicQueryRawCell(t, c, 1)
	l, e := c.location(&a)
	serviceMust(t, e)
	for i, w := range []uint32{uint32(input.origin), uint32(input.origin >> 32), input.slot, input.generation} {
		serviceMust(t, c.config.Allocator.WriteUint32(l.raw, uint64(i)*4, w))
	}
	serviceMust(t, c.config.Allocator.WriteUint32(l.raw, 20, 2))
	polynomialAssertionActionsFixture(t, "TC")
	if e := c.GetPolynomial(&a, input); !errors.Is(e, ErrUPolynomialAssertion) {
		t.Error("alias mismatch skipped source Throw", e)
	}
	if e := c.GetPolynomial(&a, input); e != nil {
		t.Error("alias source Continue did not return", e)
	}
	serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	if e := c.GetPolynomial(&a, input); e != nil {
		t.Error("disabled alias assertion still evaluated", e)
	}
	d, e := p.descriptor(input)
	serviceMust(t, e)
	if d.state.size != 3 || p.values(&d.state)[2].Val != 1 {
		t.Error("alias return changed output")
	}
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, p.DelPolynomial(&input))
}
