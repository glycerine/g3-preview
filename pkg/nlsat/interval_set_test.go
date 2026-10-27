package nlsat

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/sat"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func intervalFixture(t *testing.T) *IntervalSetManager {
	t.Helper()
	am, _ := assignmentFixture(t)
	q, err := am.RationalManager()
	assignmentMust(t, err)
	a, err := umemory.NewSmallAllocator(umemory.SmallAllocatorConfig{Runtime: q.Integers().RuntimeOwner(), Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: diagnostic.DebugBuild}})
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, a.Close()) })
	m, err := NewIntervalSetManager(am, a)
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, m.Close()) })
	return m
}

func intervalBuild(t *testing.T, m *IntervalSetManager, pool map[int32]*polynomial.Algebraic, specs []intervalSpec) *IntervalSet {
	t.Helper()
	var result *IntervalSet
	for _, spec := range specs {
		s := intervalMk(t, m, pool, spec)
		next, err := m.MkUnion(result, s)
		assignmentMust(t, err)
		assignmentMust(t, m.IncRef(next))
		assignmentMust(t, m.DecRef(result))
		assignmentMust(t, m.DecRef(s))
		result = next
	}
	return result
}
func intervalCompositeOracle(t *testing.T, m *IntervalSetManager, pool map[int32]*polynomial.Algebraic, as, bs []intervalSpec) {
	t.Helper()
	a := intervalBuild(t, m, pool, as)
	defer func() { assignmentMust(t, m.DecRef(a)) }()
	b := intervalBuild(t, m, pool, bs)
	defer func() { assignmentMust(t, m.DecRef(b)) }()
	u, err := m.MkUnion(a, b)
	assignmentMust(t, err)
	assignmentMust(t, m.IncRef(u))
	defer func() { assignmentMust(t, m.DecRef(u)) }()
	q, err := m.am.RationalManager()
	assignmentMust(t, err)
	point := unumeral.NewRational()
	defer func() { assignmentMust(t, q.Del(&point)) }()
	ab, ba, full := true, true, true
	for x := int32(-7); x <= 7; x++ {
		av, bv := false, false
		for _, s := range as {
			av = av || s.at(x)
		}
		for _, s := range bs {
			bv = bv || s.at(x)
		}
		ab = ab && (!av || bv)
		ba = ba && (!bv || av)
		full = full && (av || bv)
		assignmentMust(t, q.SetFraction(&point, int64(x), 2))
		count, why := intervalActualAt(t, m, u, &point)
		if count > 1 || (count == 1) != (av || bv) {
			t.Fatal("composite union", x, as, bs, count)
		}
		if count == 1 {
			justified := false
			for _, s := range as {
				justified = justified || s.at(x) && s.why == why
			}
			for _, s := range bs {
				justified = justified || s.at(x) && s.why == why
			}
			if !justified {
				t.Fatal("composite reason does not cover point", x, why)
			}
		}
	}
	actual, err := m.Subset(a, b)
	assignmentMust(t, err)
	if actual != ab {
		t.Fatal("composite subset", as, bs, actual, ab)
	}
	actual, err = m.Subset(b, a)
	assignmentMust(t, err)
	if actual != ba {
		t.Fatal("reverse composite subset", as, bs, actual, ba)
	}
	actual, err = m.SetEq(a, b)
	assignmentMust(t, err)
	if actual != (ab && ba) {
		t.Fatal("composite set equality")
	}
	actual, err = m.IsFull(u)
	assignmentMust(t, err)
	if actual != full {
		t.Fatal("composite full flag")
	}
}
func TestIntervalCompositeUnionsIndependentOracle(t *testing.T) {
	m := intervalFixture(t)
	pool := intervalPool(t, m)
	rng := rand.New(rand.NewPCG(94719, 179321))
	for trial := 0; trial < 240; trial++ {
		var models [2][]intervalSpec
		for side := range models {
			for n := rng.IntN(6); n > 0; n-- {
				lo, hi := int32(rng.IntN(5)-2), int32(rng.IntN(5)-2)
				if lo > hi {
					lo, hi = hi, lo
				}
				s := intervalSpec{lower: lo, upper: hi, lowerOpen: rng.IntN(2) != 0, upperOpen: rng.IntN(2) != 0, why: sat.MakeLiteral(sat.BoolVar(1+rng.IntN(5)), rng.IntN(2) != 0)}
				if lo == hi {
					s.lowerOpen = false
					s.upperOpen = false
				}
				if rng.IntN(6) == 0 {
					s.lowerInf = true
					s.lowerOpen = true
				}
				if rng.IntN(6) == 0 {
					s.upperInf = true
					s.upperOpen = true
				}
				models[side] = append(models[side], s)
			}
		}
		intervalCompositeOracle(t, m, pool, models[0], models[1])
	}
}

func intervalNext(state uint32) (uint32, uint32) {
	// Independent mathematical LCG oracle, reducing the unsigned state first.
	next := uint32((uint64(state)*214013 + 2531011) & 0xffffffff)
	return next, (next >> 16) & 0x7fff
}
func TestIntervalComplementEmptyAndReservoirRandomSequence(t *testing.T) {
	m := intervalFixture(t)
	pool := intervalPool(t, m)
	for _, seed := range []uint32{0, 1, 17, 0xffffffff} {
		for _, isInt := range []bool{false, true} {
			assignmentMust(t, m.SetSeed(seed))
			var value polynomial.Algebraic
			assignmentMust(t, m.PeekInComplement(nil, isInt, &value, true))
			state, sign := intervalNext(seed)
			state, denK := intervalNext(state)
			num := int64(-1)
			if sign%2 == 0 {
				num = 1
			}
			den := uint64(1)
			if !isInt {
				den = 1 << (denK % 4)
			}
			q, err := m.am.RationalManager()
			assignmentMust(t, err)
			want := unumeral.NewRational()
			assignmentMust(t, q.SetFraction(&want, num, den))
			cmp, err := m.am.CompareRational(&value, &want)
			assignmentMust(t, err)
			if cmp != 0 {
				t.Fatal("empty randomized witness", seed, isInt)
			}
			_, next := intervalNext(state)
			if actual := m.rand.Next(); actual != next {
				t.Fatal("random draw count", seed, isInt, actual, next)
			}
			assignmentMust(t, q.Del(&want))
			assignmentMust(t, m.am.Del(&value))
		}
	}
	s := intervalMk(t, m, pool, intervalSpec{lower: 0, upper: 1, why: sat.MakeLiteral(1, false)})
	defer func() { assignmentMust(t, m.DecRef(s)) }()
	for seed := uint32(0); seed < 24; seed++ {
		assignmentMust(t, m.SetSeed(seed))
		var w polynomial.Algebraic
		assignmentMust(t, m.PeekInComplement(s, false, &w, true))
		_, draw := intervalNext(seed)
		want := int32(-1)
		if draw%2 == 0 {
			want = 2
		}
		assignmentWantInteger(t, m.am, &w, want)
		assignmentMust(t, m.am.Del(&w))
	}
	assignmentMust(t, m.SetSeed(17))
	var zero polynomial.Algebraic
	assignmentMust(t, m.PeekInComplement(nil, false, &zero, false))
	assignmentWantInteger(t, m.am, &zero, 0)
	_, first := intervalNext(17)
	if m.rand.Next() != first {
		t.Fatal("deterministic empty witness consumed randomness")
	}
	assignmentMust(t, m.am.Del(&zero))
}

func TestIntervalComplementGapsRationalPriorityAndIrrationalFallback(t *testing.T) {
	m := intervalFixture(t)
	pool := intervalPool(t, m)
	// Source is_int affects the empty-set denominator only. A nonempty gap
	// can still produce a fractional witness, for later integer checking.
	s := intervalBuild(t, m, pool, []intervalSpec{{lowerInf: true, lowerOpen: true, upper: 0, why: sat.MakeLiteral(1, false)}, {lower: 1, upperInf: true, upperOpen: true, why: sat.MakeLiteral(2, false)}})
	defer func() { assignmentMust(t, m.DecRef(s)) }()
	var w polynomial.Algebraic
	assignmentMust(t, m.PeekInComplement(s, true, &w, false))
	q, err := m.am.RationalManager()
	assignmentMust(t, err)
	half := unumeral.NewRational()
	assignmentMust(t, q.SetFraction(&half, 1, 2))
	cmp, err := m.am.CompareRational(&w, &half)
	assignmentMust(t, err)
	if cmp != 0 {
		t.Fatal("source gap selection")
	}
	assignmentMust(t, q.Del(&half))
	assignmentMust(t, m.am.Del(&w))
	root := assignmentInteger(t, m.am, 2)
	assignmentMust(t, m.am.Root(root, 2, root))
	left, err := m.Mk(true, true, nil, true, false, root, sat.MakeLiteral(3, false))
	assignmentMust(t, err)
	assignmentMust(t, m.IncRef(left))
	defer func() { assignmentMust(t, m.DecRef(left)) }()
	right, err := m.Mk(true, false, root, true, true, nil, sat.MakeLiteral(4, false))
	assignmentMust(t, err)
	assignmentMust(t, m.IncRef(right))
	defer func() { assignmentMust(t, m.DecRef(right)) }()
	onlyRoot, err := m.MkUnion(left, right)
	assignmentMust(t, err)
	assignmentMust(t, m.IncRef(onlyRoot))
	defer func() { assignmentMust(t, m.DecRef(onlyRoot)) }()
	assignmentMust(t, m.PeekInComplement(onlyRoot, true, &w, false))
	cmp, err = m.am.Compare(&w, root)
	assignmentMust(t, err)
	if cmp != 0 {
		t.Fatal("isolated irrational witness")
	}
	assignmentMust(t, m.am.Del(&w))
	middle, err := m.Mk(true, false, root, true, false, pool[2], sat.MakeLiteral(5, false))
	assignmentMust(t, err)
	assignmentMust(t, m.IncRef(middle))
	defer func() { assignmentMust(t, m.DecRef(middle)) }()
	tail := intervalMk(t, m, pool, intervalSpec{lower: 2, lowerOpen: true, upperInf: true, upperOpen: true, why: sat.MakeLiteral(6, false)})
	defer func() { assignmentMust(t, m.DecRef(tail)) }()
	partial, err := m.MkUnion(left, middle)
	assignmentMust(t, err)
	assignmentMust(t, m.IncRef(partial))
	defer func() { assignmentMust(t, m.DecRef(partial)) }()
	both, err := m.MkUnion(partial, tail)
	assignmentMust(t, err)
	assignmentMust(t, m.IncRef(both))
	defer func() { assignmentMust(t, m.DecRef(both)) }()
	assignmentMust(t, m.PeekInComplement(both, false, &w, false))
	assignmentWantInteger(t, m.am, &w, 2)
	assignmentMust(t, m.am.Del(&w))
}

type intervalSpec struct {
	lower, upper                             int32
	lowerOpen, upperOpen, lowerInf, upperInf bool
	why                                      sat.Literal
}

func (s intervalSpec) at(x2 int32) bool {
	return (s.lowerInf || x2 > 2*s.lower || x2 == 2*s.lower && !s.lowerOpen) && (s.upperInf || x2 < 2*s.upper || x2 == 2*s.upper && !s.upperOpen)
}
func intervalPool(t *testing.T, m *IntervalSetManager) map[int32]*polynomial.Algebraic {
	t.Helper()
	pool := map[int32]*polynomial.Algebraic{}
	for n := int32(-3); n <= 3; n++ {
		pool[n] = assignmentInteger(t, m.am, n)
	}
	return pool
}
func intervalMk(t *testing.T, m *IntervalSetManager, pool map[int32]*polynomial.Algebraic, s intervalSpec) *IntervalSet {
	t.Helper()
	value, err := m.Mk(s.lowerOpen, s.lowerInf, pool[s.lower], s.upperOpen, s.upperInf, pool[s.upper], s.why)
	assignmentMust(t, err)
	assignmentMust(t, m.IncRef(value))
	return value
}
func intervalActualAt(t *testing.T, m *IntervalSetManager, s *IntervalSet, q *unumeral.Rational) (int, sat.Literal) {
	t.Helper()
	if s == nil {
		return 0, sat.DefaultLiteral()
	}
	count := 0
	why := sat.DefaultLiteral()
	for j := range s.intervals {
		i := &s.intervals[j]
		if !i.lowerInf {
			cmp, err := m.am.CompareRational(&i.lower, q)
			assignmentMust(t, err)
			if cmp > 0 || cmp == 0 && i.lowerOpen {
				continue
			}
		}
		if !i.upperInf {
			cmp, err := m.am.CompareRational(&i.upper, q)
			assignmentMust(t, err)
			if cmp < 0 || cmp == 0 && i.upperOpen {
				continue
			}
		}
		count++
		why = i.justification
	}
	return count, why
}
func intervalPairOracle(t *testing.T, m *IntervalSetManager, pool map[int32]*polynomial.Algebraic, a, b intervalSpec) {
	t.Helper()
	s1 := intervalMk(t, m, pool, a)
	defer func() { assignmentMust(t, m.DecRef(s1)) }()
	s2 := intervalMk(t, m, pool, b)
	defer func() { assignmentMust(t, m.DecRef(s2)) }()
	u, err := m.MkUnion(s1, s2)
	assignmentMust(t, err)
	assignmentMust(t, m.IncRef(u))
	defer func() { assignmentMust(t, m.DecRef(u)) }()
	q, err := m.am.RationalManager()
	assignmentMust(t, err)
	point := unumeral.NewRational()
	defer func() { assignmentMust(t, q.Del(&point)) }()
	ab, ba, full := true, true, true
	for x := int32(-7); x <= 7; x++ {
		av, bv := a.at(x), b.at(x)
		ab = ab && (!av || bv)
		ba = ba && (!bv || av)
		full = full && (av || bv)
		assignmentMust(t, q.SetFraction(&point, int64(x), 2))
		count, why := intervalActualAt(t, m, u, &point)
		if count > 1 || (count == 1) != (av || bv) {
			t.Fatalf("union point %d/2, a=%+v b=%+v count=%d", x, a, b, count)
		}
		if count == 1 && !(av && why == a.why || bv && why == b.why) {
			t.Fatal("unjustified union point", x, a, b, why)
		}
	}
	actual, err := m.Subset(s1, s2)
	assignmentMust(t, err)
	if actual != ab {
		t.Fatal("subset", a, b, actual, ab)
	}
	actual, err = m.Subset(s2, s1)
	assignmentMust(t, err)
	if actual != ba {
		t.Fatal("reverse subset", a, b, actual, ba)
	}
	actual, err = m.SetEq(s1, s2)
	assignmentMust(t, err)
	if actual != (ab && ba) {
		t.Fatal("geometric equality", a, b, actual)
	}
	actual, err = m.Eq(s1, s2)
	assignmentMust(t, err)
	if actual != (ab && ba && a.why == b.why) {
		t.Fatal("justified equality", a, b, actual)
	}
	actual, err = m.IsFull(u)
	assignmentMust(t, err)
	if actual != full {
		t.Fatal("full flag", a, b, actual, full)
	}
}
func TestIntervalUnionAndSubsetIndependentNumberLineOracle(t *testing.T) {
	m := intervalFixture(t)
	pool := intervalPool(t, m)
	var specs []intervalSpec
	for low := int32(-2); low <= 2; low++ {
		for high := low; high <= 2; high++ {
			for flags := 0; flags < 4; flags++ {
				if low == high && flags != 0 {
					continue
				}
				specs = append(specs, intervalSpec{lower: low, upper: high, lowerOpen: flags&1 != 0, upperOpen: flags&2 != 0})
			}
		}
		for _, open := range []bool{false, true} {
			specs = append(specs, intervalSpec{lowerInf: true, lowerOpen: true, upper: low, upperOpen: open}, intervalSpec{lower: low, lowerOpen: open, upperInf: true, upperOpen: true})
		}
	}
	specs = append(specs, intervalSpec{lowerInf: true, upperInf: true, lowerOpen: true, upperOpen: true})
	for i, a := range specs {
		for _, b := range specs {
			a.why = sat.MakeLiteral(1, false)
			b.why = sat.MakeLiteral(2, true)
			if i%2 == 0 {
				b.why = a.why
			}
			intervalPairOracle(t, m, pool, a, b)
		}
	}
	t.Log("independent interval pairs", len(specs)*len(specs))
}
func TestIntervalUnionReasonBoundariesCompressionAndAliasing(t *testing.T) {
	m := intervalFixture(t)
	pool := intervalPool(t, m)
	p, q := sat.MakeLiteral(1, false), sat.MakeLiteral(2, true)
	a := intervalMk(t, m, pool, intervalSpec{lower: 0, upper: 2, why: p})
	defer func() { assignmentMust(t, m.DecRef(a)) }()
	b := intervalMk(t, m, pool, intervalSpec{lower: 2, upper: 3, why: q})
	defer func() { assignmentMust(t, m.DecRef(b)) }()
	u, err := m.MkUnion(a, b)
	assignmentMust(t, err)
	assignmentMust(t, m.IncRef(u))
	defer func() { assignmentMust(t, m.DecRef(u)) }()
	var out bytes.Buffer
	assignmentMust(t, m.Display(&out, u))
	if out.String() != "{[0, p1, 2), [2, ~p2, 3]}" {
		t.Fatal("source closed contact priority", out.String())
	}
	var js []sat.Literal
	assignmentMust(t, m.GetJustifications(u, &js))
	if fmt.Sprint(js) != fmt.Sprint([]sat.Literal{p, q}) {
		t.Fatal("justification order", js)
	}
	assignmentMust(t, m.GetJustifications(u, &js))
	if len(js) != 2 {
		t.Fatal("visited flags not cleared", js)
	}
	copy, err := m.GetInterval(u, 0)
	assignmentMust(t, err)
	assignmentMust(t, m.IncRef(copy))
	defer func() { assignmentMust(t, m.DecRef(copy)) }()
	n, err := m.NumIntervals(copy)
	assignmentMust(t, err)
	if n != 1 {
		t.Fatal("interval extraction")
	}
	alias, err := m.MkUnion(nil, a)
	assignmentMust(t, err)
	if alias != a || a.refs != 1 {
		t.Fatal("union nil changed pointer/reference count")
	}
	alias, err = m.MkUnion(a, a)
	assignmentMust(t, err)
	if alias != a || a.refs != 1 {
		t.Fatal("union identity changed pointer/reference count")
	}
	other := intervalMk(t, m, pool, intervalSpec{lower: 2, upper: 3, why: p})
	defer func() { assignmentMust(t, m.DecRef(other)) }()
	merged, err := m.MkUnion(a, other)
	assignmentMust(t, err)
	assignmentMust(t, m.IncRef(merged))
	defer func() { assignmentMust(t, m.DecRef(merged)) }()
	if len(merged.intervals) != 1 {
		t.Fatal("same-reason adjacent pieces not compressed")
	}
	duplicate := intervalMk(t, m, pool, intervalSpec{lower: 0, upper: 2, why: q})
	defer func() { assignmentMust(t, m.DecRef(duplicate)) }()
	left, err := m.MkUnion(a, duplicate)
	assignmentMust(t, err)
	assignmentMust(t, m.IncRef(left))
	defer func() { assignmentMust(t, m.DecRef(left)) }()
	if left.intervals[0].justification != p {
		t.Fatal("equal intervals lost left reason priority")
	}
}
