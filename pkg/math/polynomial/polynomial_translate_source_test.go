// Source: src/math/polynomial/polynomial.cpp:3282-3291,6865-6902;
// src/math/polynomial/polynomial.h:991-1000.
package polynomial

import (
	"errors"
	"fmt"
	bigmath "math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestPolynomialTranslateIndependentMultivariableIdentity(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1, 3, 5}, [][]MonomialPower{{{0, 2}, {1, 1}}, {{0, 1}, {1, 1}}, nil})
	two := util.NewInt(2)
	r, e := m.Translate(p, 0, &two)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(r))
	t.Cleanup(func() { serviceMust(t, m.DecRef(r)) })
	// (x+2)^2*y +3*(x+2)*y+5 = x²y+7xy+10y+5.
	polynomialQueryExpect(t, m, r, map[[3]uint32]int64{{2, 1, 0}: 1, {1, 1, 0}: 7, {0, 1, 0}: 10, {0, 0, 0}: 5})
}
func TestPolynomialTranslateArraySourceAssignment(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
	xs, _ := variableHelperArray(t, m.memory, []uint32{0, 1})
	r, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, r.Close()) })
	serviceMust(t, m.TranslateMany(p, 2, xs, []util.Int{util.NewInt(2), util.NewInt(-3)}, r))
	got, e := r.Get()
	serviceMust(t, e)
	polynomialQueryExpect(t, m, got, map[[3]uint32]int64{{1, 1, 0}: 1, {1, 0, 0}: -3, {0, 1, 0}: 2, {0, 0, 0}: -5})
}
func TestPolynomialCoefficientsActualSOMVector(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{2, 3, 5}, [][]MonomialPower{{{0, 2}, {1, 1}}, {{0, 1}}, nil})
	cs, e := NewPolynomialSOMVector(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, cs.CloseChecked()) })
	serviceMust(t, m.Coefficients(p, 0, cs))
	for k, want := range []map[[3]uint32]int64{{{0, 0, 0}: 5}, {{0, 0, 0}: 3}, {{0, 1, 0}: 2}} {
		c, e := cs.At(uint32(k))
		serviceMust(t, e)
		r, e := c.Mk(false)
		serviceMust(t, e)
		serviceMust(t, m.IncRef(r))
		polynomialQueryExpect(t, m, r, want)
		serviceMust(t, m.DecRef(r))
	}
}

func TestPolynomialTranslateSourceShortCircuits(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1, 5}, [][]MonomialPower{{{0, 2}}, nil})
	m.config.Checkpoint = func() bool { t.Fatal("unreached checkpoint"); return false }
	m.config.Cooperation = func() error { t.Fatal("unreached cooperation"); return nil }
	r, e := m.Translate(p, 1, nil)
	serviceMust(t, e)
	if r != p {
		t.Fatal("degree-zero did not return actual input")
	}
	zero := util.NewInt(0)
	r, e = m.Translate(p, 0, &zero)
	serviceMust(t, e)
	if r != p || len(m.translateBuffers.children) != 0 {
		t.Fatal("zero value touched coefficients")
	}
	out, e := NewPolynomialRef(m)
	serviceMust(t, e)

	serviceMust(t, m.TranslateMany(p, 0, PolynomialVariablesView{}, nil, out))
	if out.root.value != p {
		t.Fatal("array assignment before count short circuit")
	}
	constant := polynomialQueryTerms(t, m, []int32{7}, [][]MonomialPower{nil})
	serviceMust(t, m.TranslateMany(constant, 2, PolynomialVariablesView{}, nil, out))
	if out.root.value != constant {
		t.Fatal("constant must assign before unread arrays")
	}
	serviceMust(t, out.Close())
	polynomialSOMPhysicalWithPolynomials(t, m, nil, nil, nil, []Polynomial{p, constant})
}

func TestPolynomialTranslateCheckpointPrefixes(t *testing.T) {
	for stop := 0; stop < 3; stop++ {
		t.Run(string(rune('0'+stop)), func(t *testing.T) {
			m := polynomialCellFixture(t)
			p := polynomialQueryTerms(t, m, []int32{1, 2}, [][]MonomialPower{{{0, 3}}, nil})
			visits, cooperations := 0, 0
			m.config.Checkpoint = func() bool { visits++; return visits <= stop }
			m.config.Cooperation = func() error { cooperations++; return nil }
			v := util.NewInt(2)
			r, e := m.Translate(p, 0, &v)
			if !errors.Is(e, ErrUPolynomialCanceled) || r != (Polynomial{}) {
				t.Fatal(r, e)
			}
			if visits != stop+1 || cooperations != stop {
				t.Fatal(visits, cooperations)
			}
			if m.translation.constructed {
				t.Fatal("monomial_ref constructed before coefficient loops finished")
			}
			polynomialSOMPhysicalWithPolynomials(t, m, nil, nil, nil, []Polynomial{p}, &v)
			serviceMust(t, m.CleanupTranslation())
			polynomialSOMPhysicalWithPolynomials(t, m, nil, nil, nil, []Polynomial{p}, &v)
		})
	}
}

func TestPolynomialTranslateEveryReservationPrefix(t *testing.T) {
	succeeded := false
	for fault := int64(0); fault < 256 && !succeeded; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			m := polynomialCellFixture(t)
			p := polynomialQueryTerms(t, m, []int32{1, 3, 5}, [][]MonomialPower{{{0, 2}, {1, 1}}, {{0, 1}, {1, 1}}, nil})
			shift := util.NewInt(2)
			m.integer.Budget().FailAfter(fault)
			out, e := m.Translate(p, 0, &shift)
			named := []Polynomial{p}
			if out != (Polynomial{}) {
				named = append(named, out)
			}
			polynomialSOMPhysicalWithPolynomials(t, m, nil, nil, nil, named, &shift)
			m.integer.Budget().FailAfter(-1)
			if e == nil {
				succeeded = true
				serviceMust(t, m.IncRef(out))
				polynomialQueryExpect(t, m, out, map[[3]uint32]int64{{2, 1, 0}: 1, {1, 1, 0}: 7, {0, 1, 0}: 10, {0, 0, 0}: 5})
				serviceMust(t, m.DecRef(out))
			} else {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual reservation category", fault, e)
				}
				if out != (Polynomial{}) {
					t.Fatal("unpublished result", out)
				}
				serviceMust(t, m.CleanupTranslation())
			}
			polynomialSOMPhysicalWithPolynomials(t, m, nil, nil, nil, []Polynomial{p}, &shift)
		})
	}
	if !succeeded {
		t.Fatal("finite reservation campaign did not reach mathematical success")
	}
}

func TestPolynomialTranslateManyEveryReservationPrefix(t *testing.T) {
	succeeded := false
	for fault := int64(0); fault < 256 && !succeeded; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			m := polynomialCellFixture(t)
			p := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
			xs, raw := variableHelperArray(t, m.memory, []uint32{0, 1})
			out, e := NewPolynomialRef(m)
			serviceMust(t, e)
			values := []util.Int{util.NewInt(2), util.NewInt(-3)}
			m.integer.Budget().FailAfter(fault)
			e = m.TranslateMany(p, 2, xs, values, out)
			polynomialSOMPhysicalWithOwners(t, m, nil, nil, []umemory.Block{*raw}, []Polynomial{p}, []*PolynomialRef{out}, &values[0], &values[1])
			m.integer.Budget().FailAfter(-1)
			if e == nil {
				succeeded = true
				polynomialQueryExpect(t, m, out.root.value, map[[3]uint32]int64{{1, 1, 0}: 1, {1, 0, 0}: -3, {0, 1, 0}: 2, {0, 0, 0}: -5})
			} else {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual reservation category", fault, e)
				}
				serviceMust(t, m.CleanupTranslation())
			}
			polynomialSOMPhysicalWithOwners(t, m, nil, nil, []umemory.Block{*raw}, []Polynomial{p}, []*PolynomialRef{out}, &values[0], &values[1])
			serviceMust(t, out.Close())
		})
	}
	if !succeeded {
		t.Fatal("finite array campaign did not reach mathematical success")
	}
}

func TestPolynomialTranslateCapturedVariablesAndWrapperProfiles(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1, 5}, [][]MonomialPower{{{0, 1}}, nil})
	out, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	raw, e := m.memory.Allocate(4)
	serviceMust(t, e)
	serviceMust(t, m.memory.WriteUint32(raw, 0, 0))
	xs := BorrowPolynomialVariables(m.memory, raw, 0)
	serviceMust(t, m.memory.Deallocate(&raw))
	e = m.TranslateMany(p, 1, xs, []util.Int{util.NewInt(2)}, out)
	if !errors.Is(e, umemory.ErrRuntimeOwner) {
		t.Fatal("captured backing read", e)
	}
	if out.root.value != p {
		t.Fatal("initial source assignment was not published before variable read")
	}
	vector, e := NewPolynomialVariableDegrees(m.memory)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, vector.CloseChecked()) })
	serviceMust(t, vector.SetDegree(0, 0))
	serviceMust(t, out.Assign(m.zero))
	e = m.TranslateVariables(p, vector, nil, out)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrPolynomialAssertion) || out.root.value != m.zero {
			t.Fatal("source wrapper assertion before delegation", e, out.root.value)
		}
	} else {
		if !errors.Is(e, arena.ErrBounds) || out.root.value != p {
			t.Fatal("release raw count/initial assignment prefix", e, out.root.value)
		}
	}
	serviceMust(t, m.TranslateVariables(p, vector, []util.Int{util.NewInt(2)}, out))
	polynomialQueryExpect(t, m, out.root.value, map[[3]uint32]int64{{1, 0, 0}: 1, {0, 0, 0}: 7})
}

func TestPolynomialTranslateWarmAbsentSourceStorage(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{3, 5}, [][]MonomialPower{{{0, 2}}, {{1, 3}}})
	before, state, used := m.memory.Stats(), m.memory.State(), m.integer.Budget().Used()
	count, e := m.memory.AllocationCount()
	serviceMust(t, e)
	m.integer.Budget().FailAfter(0)
	allocs := testing.AllocsPerRun(25, func() {
		r, e := m.Translate(p, 2, nil)
		if e != nil || r != p {
			panic("source absent path")
		}
	})
	polynomialSOMPhysicalWithPolynomials(t, m, nil, nil, nil, []Polynomial{p})
	after, e := m.memory.AllocationCount()
	serviceMust(t, e)
	if allocs != 0 || before != m.memory.Stats() || state != m.memory.State() || used != m.integer.Budget().Used() || count != after {
		t.Fatal("source allocation-free absent route", allocs)
	}
	m.integer.Budget().FailAfter(-1)
}

func TestPolynomialCoefficientsEveryReservationPrefix(t *testing.T) {
	succeeded := false
	for fault := int64(0); fault < 256 && !succeeded; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			m := polynomialCellFixture(t)
			p := polynomialQueryTerms(t, m, []int32{2, 3, 5}, [][]MonomialPower{{{0, 2}, {1, 1}}, {{0, 1}}, nil})
			cs, e := NewPolynomialSOMVector(m)
			serviceMust(t, e)
			m.integer.Budget().FailAfter(fault)
			e = m.Coefficients(p, 0, cs)
			polynomialSOMPhysicalWithPolynomials(t, m, []*PolynomialSOMVector{cs}, nil, nil, []Polynomial{p})
			m.integer.Budget().FailAfter(-1)
			if e == nil {
				succeeded = true
				for k, want := range []map[[3]uint32]int64{{{0, 0, 0}: 5}, {{0, 0, 0}: 3}, {{0, 1, 0}: 2}} {
					c, e := cs.At(uint32(k))
					serviceMust(t, e)
					r, e := c.Mk(false)
					serviceMust(t, e)
					serviceMust(t, m.IncRef(r))
					polynomialQueryExpect(t, m, r, want)
					serviceMust(t, m.DecRef(r))
				}
			} else {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual coefficient reservation category", fault, e)
				}
				serviceMust(t, m.CleanupTranslation())
			}
			polynomialSOMPhysicalWithPolynomials(t, m, []*PolynomialSOMVector{cs}, nil, nil, []Polynomial{p})
			serviceMust(t, cs.CloseChecked())
		})
	}
	if !succeeded {
		t.Fatal("finite coefficient campaign did not reach success")
	}
}

func TestPolynomialTranslateBigCoefficientEveryReservationPrefix(t *testing.T) {
	succeeded := false
	for fault := int64(0); fault < 512 && !succeeded; fault++ {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			m := polynomialCellFixture(t)
			limbs := make([]uint32, 33)
			limbs[0] = 17
			limbs[32] = 1
			var big util.Int
			serviceMust(t, m.integer.SetLimbs(&big, limbs))
			t.Cleanup(func() { serviceMust(t, m.integer.Del(&big)) })
			term, e := m.monomials.MkVariable(0, 1)
			serviceMust(t, e)
			p, e := m.MkTerms([]util.Int{big, util.NewInt(1)}, []Monomial{term, m.monomials.unit})
			serviceMust(t, e)
			p = polynomialCaller(t, m, p)
			two := util.NewInt(2)
			m.integer.Budget().FailAfter(fault)
			out, e := m.Translate(p, 0, &two)
			polys := []Polynomial{p}
			if out != (Polynomial{}) {
				polys = append(polys, out)
			}
			polynomialSOMPhysicalWithPolynomials(t, m, nil, nil, nil, polys, &big, &two)
			m.integer.Budget().FailAfter(-1)
			if e != nil {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual BIG reservation category", fault, e)
				}
				serviceMust(t, m.CleanupTranslation())
			} else {
				succeeded = true
				serviceMust(t, m.IncRef(out))
				n, e := m.readWord(out, 8)
				serviceMust(t, e)
				if n != 2 {
					t.Fatal(n)
				}
				for i := uint32(0); i < n; i++ {
					term, e := m.monomialCore(out, i)
					serviceMust(t, e)
					d, e := m.monomials.DegreeOf(term, 0)
					serviceMust(t, e)
					coefficient, e := m.coefficientCore(out, i)
					serviceMust(t, e)
					decoded := roundPublicationDecode(t, m.integer, &coefficient)
					expected := new(bigmath.Int).Lsh(bigmath.NewInt(1), 1024)
					expected.Add(expected, bigmath.NewInt(17))
					if d == 0 {
						expected.Lsh(expected, 1)
						expected.Add(expected, bigmath.NewInt(1))
					} else if d != 1 {
						t.Fatal(d)
					}
					if decoded.Cmp(expected) != 0 {
						t.Fatal("independent BIG shift identity", d)
					}
				}
				serviceMust(t, m.DecRef(out))
			}
			polynomialSOMPhysicalWithPolynomials(t, m, nil, nil, nil, []Polynomial{p}, &big, &two)
		})
	}
	if !succeeded {
		t.Fatal("BIG reservation campaign did not reach success")
	}
}

func polynomialTranslateEvaluate(t *testing.T, m *MultivariatePolynomialCore, p Polynomial, x, y int64) int64 {
	t.Helper()
	n, e := m.readWord(p, 8)
	serviceMust(t, e)
	sum := int64(0)
	for i := uint32(0); i < n; i++ {
		a, e := m.coefficientCore(p, i)
		serviceMust(t, e)
		v, e := m.integer.Int64(&a)
		serviceMust(t, e)
		term, e := m.monomialCore(p, i)
		serviceMust(t, e)
		for variable, point := range []int64{x, y} {
			d, e := m.monomials.DegreeOf(term, uint32(variable))
			serviceMust(t, e)
			for k := uint32(0); k < d; k++ {
				v *= point
			}
		}
		sum += v
	}
	return sum
}
func TestPolynomialTranslateIndependentFiniteGrid(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{2, -3, 5, 7}, [][]MonomialPower{{{0, 3}, {1, 2}}, {{0, 1}, {1, 1}}, {{1, 2}}, nil})
	xs, raw := variableHelperArray(t, m.memory, []uint32{0, 1})
	out, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	for dx := int64(-2); dx <= 2; dx++ {
		for dy := int64(-2); dy <= 2; dy++ {
			values := []util.Int{util.NewInt(int32(dx)), util.NewInt(int32(dy))}
			serviceMust(t, m.TranslateMany(p, 2, xs, values, out))
			for x := int64(-2); x <= 2; x++ {
				for y := int64(-2); y <= 2; y++ {
					xx, yy := x+dx, y+dy
					expected := 2*xx*xx*xx*yy*yy - 3*xx*yy + 5*yy*yy + 7
					if actual := polynomialTranslateEvaluate(t, m, out.root.value, x, y); actual != expected {
						t.Fatal(dx, dy, x, y, actual, expected)
					}
				}
			}
			polynomialSOMPhysicalWithOwners(t, m, nil, nil, []umemory.Block{*raw}, []Polynomial{p}, []*PolynomialRef{out}, &values[0], &values[1])
		}
	}
}

func TestPolynomialTranslateWrapperColdOwnerBeforeReads(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{1}, [][]MonomialPower{{{0, 1}}})
	out, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	vector, e := NewPolynomialVariableDegrees(m.memory)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, vector.CloseChecked()) })
	copied := *vector
	if e = m.TranslateVariables(p, &copied, nil, out); !errors.Is(e, ErrPolynomialOwner) {
		t.Fatal("copied vector cold identity", e)
	}
	serviceMust(t, vector.CloseChecked())
	if e = m.TranslateVariables(p, vector, nil, out); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed vector cold lifetime", e)
	}
	if out.root.value != (Polynomial{}) {
		t.Fatal("cold error published initial assignment")
	}
}
