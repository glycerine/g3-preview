// Added profile/ownership tests for inf_int_rational.h:98-119,
// inf_rational.h:111-132, inf_eps_rational.h:107-128 and rational.h:101-125.
package numeral

import (
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func infProjectionWant(n *big.Int, a *Int) (int64, uint64) {
	u := projectionLowMagnitude(n)
	i := int64(u)
	if n.Sign() < 0 {
		i = -i
	}
	if a.IsSmall() {
		return int64(a.Val), uint64(int64(a.Val))
	}
	return i, u
}

func TestInfScalarProjectionAllGetterProfilesAndRawPredicates(t *testing.T) {
	s, _ := serviceNew(t)
	for _, text := range []string{"-9223372036854775808", "18446744073709551615", "18446744073709551623", "-18446744073709551623", "-7", "0", "2"} {
		n, ok := new(big.Int).SetString(text, 10)
		if !ok {
			t.Fatal(text)
		}
		for _, den := range []int32{0, 1, 3} {
			for _, epsilon := range []int32{-1, 0, 7} {
				a, e := s.InfIntNewValue()
				serviceMust(t, e)
				b, e := s.NewInfValue()
				serviceMust(t, e)
				c, e := s.InfEpsInfNewValue()
				serviceMust(t, e)
				for _, first := range []*Rational{&a.First, &b.First, &c.numeral.First} {
					setIntegerOracle(t, s.Manager.Z, &first.Num, n)
					first.Den = NewInt(den) // source raw component fixture; no normalization
				}
				a.second = epsilon
				b.Second = Rational{Num: NewInt(epsilon), Den: NewInt(0)}
				c.numeral.Second = Rational{Num: NewInt(epsilon), Den: NewInt(0)}
				c.infinity = Rational{Num: NewInt(0), Den: NewInt(0)}
				wantI, wantU := infProjectionWant(n, &a.First.Num)
				fitI := den == 1 && epsilon == 0 && n.IsInt64()
				fitU := den == 1 && epsilon == 0 && n.Sign() >= 0 && n.BitLen() <= 64
				for _, f := range []struct {
					i                          func() (int64, error)
					u                          func() (uint64, error)
					ip, up, integral, rational func() (bool, error)
				}{
					{func() (int64, error) { return s.InfIntInt64(&a) }, func() (uint64, error) { return s.InfIntUint64(&a) }, func() (bool, error) { return s.InfIntIsInt64(&a) }, func() (bool, error) { return s.InfIntIsUint64(&a) }, func() (bool, error) { return s.InfIntIsInt(&a) }, func() (bool, error) { return s.InfIntIsRational(&a) }},
					{func() (int64, error) { return s.InfInt64(&b) }, func() (uint64, error) { return s.InfUint64(&b) }, func() (bool, error) { return s.InfIsInt64(&b) }, func() (bool, error) { return s.InfIsUint64(&b) }, func() (bool, error) { return s.InfIsInt(&b) }, func() (bool, error) { return s.InfIsRational(&b) }},
					{func() (int64, error) { return s.InfEpsInfInt64(&c) }, func() (uint64, error) { return s.InfEpsInfUint64(&c) }, func() (bool, error) { return s.InfEpsInfIsInt64(&c) }, func() (bool, error) { return s.InfEpsInfIsUint64(&c) }, func() (bool, error) { return s.InfEpsInfIsInt(&c) }, func() (bool, error) { return s.InfEpsInfIsRational(&c) }},
				} {
					iv, ie := f.i()
					uv, ue := f.u()
					serviceAuditRationalRoots(t, s, &a.First, &b.First, &b.Second, &c.infinity, &c.numeral.First, &c.numeral.Second)
					projectionExpectError(t, ie, !fitI)
					projectionExpectError(t, ue, !fitU)
					if (ie == nil && iv != wantI) || (ue == nil && uv != wantU) {
						t.Fatal("release delegates FIRST, ignoring epsilon/infinity", text, den, epsilon, iv, uv, wantI, wantU)
					}
					for _, p := range []struct {
						call func() (bool, error)
						want bool
					}{{f.ip, fitI}, {f.up, fitU}, {f.integral, den == 1 && epsilon == 0}, {f.rational, epsilon == 0}} {
						if got, e := p.call(); e != nil || got != p.want {
							t.Fatal("actual source raw predicates", text, den, epsilon, got, p.want, e)
						}
					}
				}
				serviceMust(t, s.InfIntDel(&a))
				serviceMust(t, s.InfDel(&b))
				serviceMust(t, s.InfEpsInfDel(&c))
			}
		}
	}
}

func TestInfScalarProjectionUnreadFieldsAndNestedShortCircuit(t *testing.T) {
	s, _ := serviceNew(t)
	v, e := s.InfEpsInfNewValue()
	serviceMust(t, e)
	var dead Int
	setIntegerOracle(t, s.Manager.Z, &dead, new(big.Int).Lsh(big.NewInt(1), 100))
	stale := dead
	serviceMust(t, s.Manager.Z.Del(&dead))
	// A bounded ended-pointee fixture: scalar object fields remain readable,
	// but an actual pointee read must reject the stale token. Restore before Del.
	v.numeral.First.Num = stale
	v.infinity.Num = NewInt(1)
	if got, e := s.InfEpsInfIsInt64(&v); e != nil || got {
		t.Fatal("source infinity short-circuit must not query FIRST", got, e)
	}
	if got, e := s.InfEpsInfIsUint64(&v); e != nil || got {
		t.Fatal("source infinity short-circuit unsigned", got, e)
	}
	if got, e := s.InfEpsInfIsRational(&v); e != nil || got {
		t.Fatal("source infinity short-circuit rational", got, e)
	}
	v.infinity.Num = NewInt(0)
	if got, e := s.InfIsRational(&v.numeral); e != nil || !got {
		t.Fatal("is_rational reads only epsilon NUMval", got, e)
	}
	if got, e := s.Manager.IsInt(&v.numeral.First); e != nil || !got {
		t.Fatal("is_int reads only DEN structural fields", got, e)
	}
	v.numeral.First.Num = NewInt(7)
	v.numeral.Second.Num = stale
	if got, e := s.InfEpsInfInt64(&v); modularSourceDebug {
		if !errors.Is(e, ErrIntegerAssertion) {
			t.Fatal("DEBUG tests epsilon scalar value, not its dead pointee", e)
		}
	} else if e != nil || got != 7 {
		t.Fatal("release getter never reads epsilon pointee", got, e)
	}
	serviceAuditRationalRoots(t, s, &v.infinity, &v.numeral.First)
	v.numeral.Second.Num = NewInt(0)
	serviceMust(t, s.InfEpsInfDel(&v))
}

func TestInfScalarProjectionFacadeRawFieldsAndNarrowing(t *testing.T) {
	s, _ := serviceNew(t)
	for _, den := range []int32{-1, 0, 1, 3} {
		for _, num := range []int64{math.MinInt32, -1, 0, math.MaxInt32, math.MaxUint32, 1 << 32} {
			v, e := s.NewValue()
			serviceMust(t, e)
			serviceMust(t, s.Manager.Z.SetInt64(&v.Raw.Num, num))
			v.Raw.Den = NewInt(den)
			i, ie := s.Int64(&v)
			u, ue := s.Uint64(&v)
			i32, i32e := s.Int32(&v)
			u32, u32e := s.Unsigned(&v)
			serviceAuditRationalRoots(t, s, &v.Raw)
			projectionExpectError(t, ie, den != 1)
			projectionExpectError(t, ue, den != 1 || num < 0)
			projectionExpectError(t, i32e, den != 1 || num < math.MinInt32 || num > math.MaxInt32)
			projectionExpectError(t, u32e, den != 1 || num < 0 || uint64(num) > math.MaxUint32)
			wantU := uint64(num)
			if !v.Raw.Num.IsSmall() && num < 0 {
				wantU = uint64(-num)
			}
			if (ie == nil && i != num) || (ue == nil && u != wantU) || (i32e == nil && i32 != int32(num)) || (u32e == nil && u32 != uint32(wantU)) {
				t.Fatal(num, den, i, u, i32, u32)
			}
			for _, p := range []struct {
				call func(*OwnedRational) (bool, error)
				want bool
			}{{s.IsInt, den == 1}, {s.IsInt64, den == 1}, {s.IsUint64, den == 1 && num >= 0}, {s.IsUnsigned, den == 1 && num >= 0 && uint64(num) <= math.MaxUint32}, {s.IsInt32, den == 1 && num >= math.MinInt32 && num <= math.MaxInt32}} {
				if got, e := p.call(&v); e != nil || got != p.want {
					t.Fatal("facade predicate", num, den, got, p.want, e)
				}
			}
			f, fe := s.Float64(&v)
			if modularSourceDebug && den <= 0 {
				if !errors.Is(fe, ErrRationalFloatDomain) {
					t.Fatal(fe)
				}
			} else {
				numerator := float64(num)
				if num == 1<<32 { // selected source BIG radix UINT_MAX, not 2^32
					numerator = float64(math.MaxUint32)
				}
				want := numerator / float64(den)
				if fe != nil || (math.IsNaN(want) && !math.IsNaN(f)) || (!math.IsNaN(want) && f != want) {
					t.Fatal(f, want, fe)
				}
			}
			serviceMust(t, s.Del(&v))
		}
	}
}

func TestInfScalarProjectionOwnershipAndWarmGraph(t *testing.T) {
	s, b := serviceNew(t)
	v, e := s.InfIntFromInt(7)
	serviceMust(t, e)
	r, e := s.NewValue()
	serviceMust(t, e)
	n, e := s.NewInfValue()
	serviceMust(t, e)
	i, e := s.InfEpsInfNewValue()
	serviceMust(t, e)
	r.Raw.Num, n.First.Num, i.numeral.First.Num = NewInt(7), NewInt(7), NewInt(7)
	before := numericWarmProfileSnapshot(t, s.Manager.Z, b)
	allocs := testing.AllocsPerRun(100, func() {
		s.InfIntIsInt(&v)
		s.InfIntIsRational(&v)
		s.InfIntIsInt64(&v)
		s.InfIntIsUint64(&v)
		s.InfIntInt64(&v)
		s.InfIntUint64(&v)
		s.IsInt(&r)
		s.IsInt64(&r)
		s.IsUint64(&r)
		s.IsInt32(&r)
		s.IsUnsigned(&r)
		s.Int64(&r)
		s.Uint64(&r)
		s.Int32(&r)
		s.Unsigned(&r)
		s.Float64(&r)
		s.InfIsInt(&n)
		s.InfIsRational(&n)
		s.InfIsInt64(&n)
		s.InfIsUint64(&n)
		s.InfInt64(&n)
		s.InfUint64(&n)
		s.InfEpsInfIsInt(&i)
		s.InfEpsInfIsRational(&i)
		s.InfEpsInfIsInt64(&i)
		s.InfEpsInfIsUint64(&i)
		s.InfEpsInfInt64(&i)
		s.InfEpsInfUint64(&i)
	})
	serviceAuditRationalRoots(t, s, &v.First, &r.Raw, &n.First, &n.Second, &i.infinity, &i.numeral.First, &i.numeral.Second)
	numericWarmProfileAudit(t, s.Manager.Z, b, before, allocs, false)
	serviceMust(t, s.Del(&r))
	serviceMust(t, s.InfDel(&n))
	serviceMust(t, s.InfEpsInfDel(&i))
	foreign := v
	foreign.ownerID++ // nonowning bad-provenance fixture only
	if _, e := s.InfIntIsRational(&foreign); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatal(e)
	}
	serviceMust(t, s.InfIntDel(&v))
	if _, e := s.InfIntInt64(&v); !errors.Is(e, ErrRationalServiceOwner) {
		t.Fatal(e)
	}
	serviceMust(t, s.Close())
	if _, e := s.InfIntIsRational(&v); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

func TestInfScalarProjectionRegisteredDefaultGetter(t *testing.T) {
	// Existing src/test/inf_rational.cpp:105 ENSURE, already counted among80.
	// This extra route does not add registered source assertion credit.
	s, _ := serviceNew(t)
	v, e := s.NewInfValue()
	serviceMust(t, e)
	if got, e := s.InfInt64(&v); e != nil || got != 0 {
		t.Fatal(got, e)
	}
	serviceAuditRationalRoots(t, s, &v.First, &v.Second)
	serviceMust(t, s.InfDel(&v))
}

func TestInfScalarProjectionRawManagerEndangeredReads(t *testing.T) {
	m, _ := rationalManager(t, false)
	var dead Int
	setIntegerOracle(t, m.Z, &dead, new(big.Int).Lsh(big.NewInt(1), 100))
	stale := dead
	qopsMust(t, m.Z.Del(&dead))
	a := Rational{Num: NewInt(9), Den: stale}
	i, ie := m.Int64(&a)
	u, ue := m.Uint64(&a)
	if modularSourceDebug {
		if !errors.Is(ie, ErrIntegerAssertion) || !errors.Is(ue, ErrIntegerAssertion) {
			t.Fatal("DEBUG noninteger DEN structural check precedes NUM getter", ie, ue)
		}
	} else if ie != nil || ue != nil || i != 9 || u != 9 {
		t.Fatal("release numerator getter never accesses dead DEN", i, u, ie, ue)
	}
	a = Rational{Num: stale, Den: NewInt(3)}
	for _, call := range []func(*Rational) (bool, error){m.IsInt, m.IsInt64, m.IsUint64, m.IsInt32, m.IsUnsigned} {
		if got, e := call(&a); e != nil || got {
			t.Fatal("noninteger DEN short-circuit never accesses dead NUM", got, e)
		}
	}
	if _, e := m.Int64(&a); modularSourceDebug {
		if !errors.Is(e, ErrIntegerAssertion) {
			t.Fatal("DEBUG structural predicate fails before dead NUM getter", e)
		}
	} else if !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("release actual accessed NUM remains guarded", e)
	}
	if _, e := m.Float64(&a); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("Float64 actually accesses NUM", e)
	}
	a = Rational{Num: NewInt(0), Den: stale}
	if _, e := m.Float64(&a); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("zero NUM does not shortcut actual DEN conversion", e)
	}
	a = NewRational() // bounded nonowning dead-token fixture ends before destruction
	qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &dead})
}
