// Source: src/math/polynomial/polynomial.cpp:1613-1683,6577-6590;
// src/util/vector.h; src/util/obj_ref.h.
package polynomial

import (
	"errors"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialFactorsConstructorColdRefusal(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, nil)
	used := m.core.integer.Budget().Used()
	stats := m.core.memory.Stats()
	m.core.integer.Budget().FailAfter(0)
	f, e := NewPolynomialFactors(m)
	if f != nil || !numericPhysicalReservationRefusal(e) {
		t.Fatal("real cold class-header refusal", f, e)
	}
	if used != m.core.integer.Budget().Used() || stats != m.core.memory.Stats() {
		t.Fatal("class ctor emitted fake source allocation")
	}
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
	m.core.integer.Budget().FailAfter(-1)
}
func TestPolynomialFactorsMultiplyEveryReservation(t *testing.T) {
	for _, empty := range []bool{false, true} {
		t.Run(strconv.FormatBool(empty), func(t *testing.T) {
			success := -1
			for fail := int64(0); fail < 200; fail++ {
				completed := false
				t.Run(strconv.FormatInt(fail, 10), func(t *testing.T) {
					m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{}}})
					f, e := NewPolynomialFactors(m)
					serviceMust(t, e)
					if !empty {
						serviceMust(t, f.Push(p, 3))
						serviceMust(t, f.Push(p, 2))
					}
					c := util.Int{}
					serviceMust(t, m.core.integer.Copy(&c, &m.core.integer.Two64))
					defer func() { serviceMust(t, m.core.integer.Del(&c)) }()
					serviceMust(t, f.SetConstant(&c))
					r, e := NewPolynomialRef(m.core)
					serviceMust(t, e)
					serviceMust(t, r.Assign(p))
					m.core.integer.Budget().FailAfter(fail)
					e = f.Multiply(r)
					polynomialFactorsAudit(t, f, []Polynomial{p}, []*util.Int{&c}, r)
					if e != nil && !numericPhysicalReservationRefusal(e) {
						t.Fatal("actual Multiply reservation", e)
					}
					m.core.integer.Budget().FailAfter(-1)
					serviceMust(t, f.Cleanup())
					// A stopped operation's owning abort must leave persistent source buffers
					// usable by an independent subsequent source invocation.
					later, x := m.core.Mul(p, p)
					serviceMust(t, x)
					serviceMust(t, m.core.CleanupValue(&later))
					// Operation-local cleanup alone must retain caller reference publication.
					polynomialFactorsAudit(t, f, []Polynomial{p}, []*util.Int{&c}, r)
					if e == nil {
						result, x := r.Get()
						serviceMust(t, x)
						v := util.NewInt(0)
						a := util.Int{}
						serviceMust(t, m.UnivariateEvaluate(result, 0, &v, &a))
						eq, x := m.core.integer.Equal(&a, &c)
						serviceMust(t, x)
						if !eq {
							t.Fatal("constant-times-product at zero", a, c)
						}
						serviceMust(t, m.core.integer.Del(&a))
						completed = true
					}
					serviceMust(t, r.Close())
					serviceMust(t, f.CloseChecked())
				})
				if completed {
					success = int(fail)
					break
				}
			}
			if success < 0 {
				t.Fatal("no success through all actual Multiply prefixes")
			}
			t.Log("actual Multiply failure prefixes", success)
		})
	}
}
func TestPolynomialFactorsSourceFinalizeAndClosedGuards(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{3, [3]uint32{1, 0, 0}}})
	f, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	copied := *f
	if _, e = copied.Total(); !errors.Is(e, ErrPolynomialOwner) {
		t.Fatal("copied owning class", e)
	}
	var nilOwner *PolynomialFactors
	if _, e = nilOwner.Manager(); !errors.Is(e, ErrPolynomialOwner) {
		t.Fatal("nil owning class", e)
	}
	serviceMust(t, f.Push(p, 2))
	serviceMust(t, f.SourceFinalize())
	polynomialFactorsAudit(t, f, []Polynomial{p}, nil)
	if f.sourceStage != 4 {
		t.Fatal("actual completed body and reverse vector dtors", f.sourceStage)
	}
	if _, e = f.Total(); e == nil {
		t.Fatal("source finalized ordinary guard")
	}
	serviceMust(t, f.SourceFinalize())
	serviceMust(t, f.CloseChecked())
	serviceMust(t, f.CloseChecked())
	if _, e = f.Distinct(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed owner", e)
	}
}
func TestPolynomialFactorsAbortOwnsReachedCalleesOnly(t *testing.T) {
	found := false
	completed := false
	for fail := int64(0); fail < 100 && !completed; fail++ {
		func() {
			m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{}}})
			f, e := NewPolynomialFactors(m)
			serviceMust(t, e)
			serviceMust(t, f.Push(p, 3))
			r, e := NewPolynomialRef(m.core)
			serviceMust(t, e)
			m.core.integer.Budget().FailAfter(fail)
			e = f.Multiply(r)
			polynomialFactorsAudit(t, f, []Polynomial{p}, nil, r)
			actualOwned := m.core.generatedActive || m.core.generatedCalleePending || m.core.pending.raw != (util.ObjectRef{}) || m.core.pending.value != (Polynomial{})
			m.core.integer.Budget().FailAfter(-1)
			serviceMust(t, f.Cleanup())
			if e != nil && actualOwned {
				found = true
				if m.core.generatedActive || m.core.generatedCalleePending || m.core.pending.raw != (util.ObjectRef{}) || m.core.pending.value != (Polynomial{}) {
					t.Fatal("operation abort left its reached source callee", fail)
				}
			}
			completed = e == nil
			serviceMust(t, r.Close())
			serviceMust(t, f.CloseChecked())
		}()
	}
	if !found {
		t.Fatal("no actual source callee prefix reached")
	}
}
func TestPolynomialFactorsDisplayEveryReservation(t *testing.T) {
	success := -1
	for fail := int64(0); fail < 80; fail++ {
		complete := false
		t.Run(strconv.FormatInt(fail, 10), func(t *testing.T) {
			m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{}}})
			f, e := NewPolynomialFactors(m)
			serviceMust(t, e)
			serviceMust(t, f.Push(p, 2))
			c := util.Int{}
			digits := make([]uint32, 130)
			digits[129] = 1
			serviceMust(t, m.core.integer.SetLimbs(&c, digits))
			defer func() { serviceMust(t, m.core.integer.Del(&c)) }()
			serviceMust(t, f.SetConstant(&c))
			sink := &polynomialDisplayTestSink{}
			m.core.integer.Budget().FailAfter(fail)
			e = f.DisplayInto(sink)
			polynomialFactorsAudit(t, f, []Polynomial{p}, []*util.Int{&c})
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual factors display prefix", e)
			}
			m.core.integer.Budget().FailAfter(-1)
			serviceMust(t, f.Cleanup())
			if m.core.displayPending() {
				t.Fatal("factors cleanup omitted its actual display invocation", fail)
			}
			serviceMust(t, f.CloseChecked())
			complete = e == nil
		})
		if complete {
			success = int(fail)
			break
		}
	}
	if success < 0 {
		t.Fatal("no real display success")
	}
	t.Log("actual display prefixes", success)
}

func TestPolynomialFactorsConstantFactoryOwnsSingleton(t *testing.T) {
	q, budget := rationalManager(t, false)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: budget, Memory: q.Integers().Memory, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	control := &polynomialEvalControl{limit: 100000}
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: control.checkpoint, Cooperation: control.cooperate}})
	serviceMust(t, e)
	f, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	a := util.NewInt(17)
	serviceMust(t, f.SetConstant(&a))
	r, e := NewPolynomialRef(m.core)
	serviceMust(t, e)
	budget.FailAfter(0)
	cause := f.Multiply(r)
	if !numericPhysicalReservationRefusal(cause) || !m.core.singletonReference {
		t.Fatal("actual factory refusal after singleton INC", cause, m.core.singletonReference)
	}
	polynomialFactorsAudit(t, f, nil, nil, r)
	budget.FailAfter(-1)
	serviceMust(t, f.Cleanup())
	if m.core.singletonReference {
		t.Fatal("matched singleton INC lacks operation cleanup authority")
	}
	polynomialFactorsAudit(t, f, nil, nil, r)
	serviceMust(t, r.Close())
	serviceMust(t, f.CloseChecked())
	serviceMust(t, m.CloseChecked())
}

func TestPolynomialFactorsSingletonForeignReplacement(t *testing.T) {
	q, budget := rationalManager(t, false)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: budget, Memory: q.Integers().Memory, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	control := &polynomialEvalControl{limit: 100000}
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: control.checkpoint, Cooperation: control.cooperate}})
	serviceMust(t, e)
	f, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	a := util.NewInt(17)
	serviceMust(t, f.SetConstant(&a))
	r, e := NewPolynomialRef(m.core)
	serviceMust(t, e)
	budget.FailAfter(0)
	cause := f.Multiply(r)
	if !numericPhysicalReservationRefusal(cause) {
		t.Fatal(cause)
	}
	polynomialFactorsAudit(t, f, nil, nil, r)
	budget.FailAfter(-1)
	// Caller explicitly drains the old concrete construction, then starts a
	// different generation. The retained factors receipt must not adopt it.
	serviceMust(t, m.core.CleanupConstruction())
	b := util.NewInt(23)
	budget.FailAfter(0)
	_, foreignCause := m.core.MkConstantTake(&b)
	if !numericPhysicalReservationRefusal(foreignCause) || !m.core.singletonReference {
		t.Fatal(foreignCause)
	}
	generation := m.core.singletonGeneration
	pending := m.core.pending.generation
	polynomialFactorsAudit(t, f, nil, []*util.Int{&b}, r)
	budget.FailAfter(-1)
	serviceMust(t, f.Cleanup())
	if !m.core.singletonReference || m.core.singletonGeneration != generation || m.core.pending.generation != pending {
		t.Fatal("own abort consumed foreign replacement")
	}
	polynomialFactorsAudit(t, f, nil, []*util.Int{&b}, r)
	serviceMust(t, m.core.CleanupConstruction())
	serviceMust(t, r.Close())
	serviceMust(t, f.CloseChecked())
	serviceMust(t, m.CloseChecked())
}

func TestPolynomialFactorsConstantHelpersEveryReservation(t *testing.T) {
	for op := 0; op < 3; op++ {
		t.Run(strconv.Itoa(op), func(t *testing.T) {
			success := -1
			for fail := int64(0); fail < 100; fail++ {
				completed := false
				t.Run(strconv.FormatInt(fail, 10), func(t *testing.T) {
					m, p, _ := polynomialEvalFixture(t, nil)
					f, e := NewPolynomialFactors(m)
					serviceMust(t, e)
					c := util.Int{}
					serviceMust(t, m.core.integer.Copy(&c, &m.core.integer.Two64))
					defer func() { serviceMust(t, m.core.integer.Del(&c)) }()
					serviceMust(t, f.SetConstant(&c))
					m.core.integer.Budget().FailAfter(fail)
					switch op {
					case 0:
						e = m.AccumulateFactorConstant(f, &c)
					case 1:
						e = m.FlipFactorSign(f)
					case 2:
						e = f.SetConstant(&c)
					}
					polynomialFactorsAudit(t, f, []Polynomial{p}, []*util.Int{&c})
					if e != nil && !numericPhysicalReservationRefusal(e) {
						t.Fatal(e)
					}
					m.core.integer.Budget().FailAfter(-1)
					if e == nil {
						expected := util.Int{}
						switch op {
						case 0:
							serviceMust(t, m.core.integer.SetLimbs(&expected, []uint32{0, 0, 0, 0, 1}))
						case 1:
							serviceMust(t, m.core.integer.Copy(&expected, &c))
							serviceMust(t, m.core.integer.Neg(&expected))
						case 2:
							serviceMust(t, m.core.integer.Copy(&expected, &c))
						}
						eq, x := m.core.integer.Equal(&f.constant, &expected)
						serviceMust(t, x)
						if !eq {
							t.Fatal("independent frozen coefficient expected")
						}
						serviceMust(t, m.core.integer.Del(&expected))
						completed = true
					}
					m.core.integer.Budget().FailAfter(-1)
					serviceMust(t, f.Cleanup())
					polynomialFactorsAudit(t, f, []Polynomial{p}, []*util.Int{&c})
					serviceMust(t, f.CloseChecked())
				})
				if completed {
					success = int(fail)
					break
				}
			}
			if success < 0 {
				t.Fatal("no bounded successful helper")
			}
			t.Log("actual helper prefixes", success)
		})
	}
}

func TestPolynomialFactorsNativePushGrowthAndReferenceOwnership(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{7, [3]uint32{1, 0, 0}}})
	f, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	defer func() { m.core.integer.Budget().FailAfter(-1); serviceMust(t, f.CloseChecked()) }()
	before, e := m.core.readWord(p, 0)
	serviceMust(t, e)
	for _, count := range []uint32{4097, 129} {
		stats, used := m.core.memory.Stats(), m.core.integer.Budget().Used()
		m.core.integer.Budget().FailAfter(0)
		for i := uint32(0); i < count; i++ {
			serviceMust(t, f.Push(p, 3))
		}
		serviceMust(t, f.SetDegree(1, 7))
		n, e := f.Distinct()
		serviceMust(t, e)
		total, e := f.Total()
		serviceMust(t, e)
		refs, e := m.core.readWord(p, 0)
		serviceMust(t, e)
		if n != count || total != 3*count+4 || refs != before+count {
			t.Fatal("native factor growth lost entries or references", n, total, refs)
		}
		for i := uint32(0); i < count; i++ {
			factor, e := f.factor(i)
			serviceMust(t, e)
			degree, e := f.Degree(i)
			serviceMust(t, e)
			want := uint32(3)
			if i == 1 {
				want = 7
			}
			if factor != p || degree != want {
				t.Fatal("factor/multiplicity association changed", i, degree)
			}
		}
		if m.core.memory.Stats() != stats || m.core.integer.Budget().Used() != used {
			t.Fatal("native factor storage charged Runtime")
		}
		polynomialFactorsAudit(t, f, []Polynomial{p}, nil)
		m.core.integer.Budget().FailAfter(-1)
		serviceMust(t, f.Reset())
		refs, e = m.core.readWord(p, 0)
		serviceMust(t, e)
		if refs != before || len(f.factors) != 0 || len(f.degrees) != 0 || f.total != 0 {
			t.Fatal("reset failed to release factor references", refs, before)
		}
	}
}
