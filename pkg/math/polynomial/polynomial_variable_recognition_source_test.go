// Source: src/math/polynomial/polynomial.cpp:6970-6980;
// src/math/polynomial/polynomial.h:319-329;
// src/nlsat/nlsat_solver.cpp:2452-2485. Licensed under the MIT license.
// Source author: Leonardo de Moura.
package polynomial

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialVariableRecognitionLiteralWritePrefixes(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	mon, e := c.monomials.MkVariable(2, 1)
	serviceMust(t, e)
	x := uint32(77)
	ok, e := full.IsVariableMonomial(mon, &x)
	serviceMust(t, e)
	if !ok || x != 2 {
		t.Fatal("static single power source predicate", ok, x)
	}
	serviceMust(t, c.monomials.IncRef(mon))
	serviceMust(t, c.monomials.DecRef(mon))
	square, e := c.monomials.MkVariable(2, 2)
	serviceMust(t, e)
	x = 77
	ok, e = full.IsVariableMonomial(square, &x)
	serviceMust(t, e)
	if ok || x != 77 {
		t.Fatal("degree failure must skip variable publication", ok, x)
	}
	serviceMust(t, c.monomials.IncRef(square))
	serviceMust(t, c.monomials.DecRef(square))
	p := polynomialQueryTerms(t, c, []int32{2}, [][]MonomialPower{{{2, 1}}})
	x = 77
	ok, e = full.IsVariable(p, &x)
	serviceMust(t, e)
	if ok || x != 2 {
		t.Fatal("polynomial variable write precedes later coefficient rejection", ok, x)
	}
	p = polynomialQueryTerms(t, c, []int32{2, 5}, [][]MonomialPower{{{2, 1}}, nil})
	x = 77
	number := util.NewInt(9)
	ok, e = full.IsVariableNumeral(p, &x, &number)
	serviceMust(t, e)
	if ok || x != 77 || number != util.NewInt(9) {
		t.Fatal("numeral predicate coefficient rejection skips var and NUM", ok, x, number)
	}
	p = polynomialQueryTerms(t, c, []int32{1, 5}, [][]MonomialPower{{{2, 1}}, nil})
	x = 77
	ok, e = full.IsVariableNumeral(p, &x, &number)
	serviceMust(t, e)
	if !ok || x != 2 || number != util.NewInt(5) {
		t.Fatal("literal variable plus numeral", ok, x, number)
	}
	c.integer.Budget().FailAfter(0)
	ok, e = full.IsVariableNumeral(c.zero, nil, nil)
	serviceMust(t, e)
	if ok {
		t.Fatal("size zero source short circuit")
	}
	c.integer.Budget().FailAfter(-1)
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, []*util.Int{&number}, 0)
	serviceMust(t, c.integer.Del(&number))
}

func TestPolynomialVariableRecognitionBIGEveryActualPrefix(t *testing.T) {
	for _, field := range []bool{false, true} {
		success, retained := false, false
		for prefix := int64(0); prefix < 128; prefix++ {
			stop := false
			t.Run(fmt.Sprintf("field%v/%d", field, prefix), func(t *testing.T) {
				full := polynomialExactDivDirectPageOneFixture(t)
				for i := 0; i < 3; i++ {
					_, e := full.core.monomials.MkVar()
					serviceMust(t, e)
				}
				c := full.core
				digits := make([]uint32, 500)
				digits[0] = 17
				digits[499] = 1 << 16
				var big util.Int
				serviceMust(t, c.integer.SetLimbs(&big, digits))
				variable, e := c.monomials.MkVariable(2, 1)
				serviceMust(t, e)
				raw, e := c.MkTerms([]util.Int{util.NewInt(1), big}, []Monomial{variable, c.monomials.unit})
				serviceMust(t, e)
				p := polynomialCaller(t, c, raw)

				if field {
					serviceMust(t, c.modular.SetZPUint64(5))
				}
				out := util.NewInt(9)
				v := uint32(77)
				var receipt polynomialVariableNumeralInvocationReceipt
				c.integer.Budget().FailAfter(prefix)
				yes, returned := c.isVariableNumeralWithReceipt(p, &v, &out, &receipt)
				s := &c.variableNumeral
				roots, e := c.integer.SourceAllocator().PendingRoots()
				serviceMust(t, e)
				pending := c.integer.PendingCell
				if pending.Object != (util.ObjectRef{}) || roots != ([3]util.Block{}) {
					cellOwned, rawOwned := pending.Object == (util.ObjectRef{}), roots == ([3]util.Block{})
					if c.integer.MatchesCopyAllocationInvocation(&s.copy) {
						cellOwned = cellOwned || s.copy.Frame.Roots.Cell == pending
						rawOwned = rawOwned || s.copy.Frame.Roots.Allocator == roots
					}
					if c.modular.MatchesPNormalizeInvocation(&s.normalize) {
						cellOwned = cellOwned || s.normalize.Frame.Roots.Cell == pending
						rawOwned = rawOwned || s.normalize.Frame.Roots.Allocator == roots
					}
					if !cellOwned || !rawOwned {
						t.Fatal("actual pending SET/PN leaf lacks exact acquired child", prefix, pending, roots)
					}
				}
				polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, []*util.Int{&out, &big}, 0)
				if v != 2 || receipt.invocation == 0 {
					t.Fatal("actual variable publication and reached SET guard", v, receipt)
				}
				if returned != nil {
					if s.bodyCause == nil && s.sourceCause == nil {
						t.Fatal("actual returned cause omitted")
					}
					retained = retained || c.variableNumeralChildrenPending()
				}
				body, source, own := s.bodyCause, s.sourceCause, s.cleanupCause
				serviceMust(t, c.cleanupVariableNumeralInvocation(&receipt, false))
				serviceMust(t, c.cleanupVariableNumeralInvocation(&receipt, true))
				c.integer.Budget().FailAfter(-1)
				if s.bodyCause != body || s.sourceCause != source || s.cleanupCause != own {
					t.Fatal("exact own ACK erased immutable history")
				}
				if returned == nil {
					if !yes {
						t.Fatal("successful matched source shape rejected")
					}
					success = true
					t.Log("actual firstsuccess", field, prefix)
					stop = true
					if field && out != util.NewInt(-2) {
						t.Fatal("literal source SET mode normalization", out)
					}
				} else if !numericPhysicalReservationRefusal(returned) && !errors.Is(returned, arena.ErrInjectedFailure) {
					t.Fatal(returned)
				}
				polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, []*util.Int{&out, &big}, 0)
				serviceMust(t, c.integer.Del(&out))
				serviceMust(t, c.integer.Del(&big))
			})
			if stop {
				break
			}
		}
		if !success || !retained {
			t.Fatal("actual firstsuccess/owned pending SET or PN", field, success, retained)
		}
	}
}
