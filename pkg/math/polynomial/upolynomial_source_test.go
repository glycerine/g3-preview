package polynomial

import (
	"errors"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/cooperate"
)

// Real source services; cooperation is a coordinator-owned prerequisite.
// No successful default callback substitutes for reslimit/cooperation.
func upolySourceOwnerServices(t *testing.T) (*UPolynomialManager, *util.RationalManager, *arena.Budget, *control.Reslimit, *cooperate.Cooperation) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	q, e := util.NewRationalManager(util.RationalConfig{Integers: util.IntegerConfig{Cells: arena.Config{PageSize: 8, Budget: b}, Words: arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{PageSize: 8, Budget: b}}}})
	if e != nil {
		t.Fatal(e)
	}
	limit, e := control.NewReslimit(b)
	if e != nil {
		t.Fatal(e)
	}
	cooperation, e := cooperate.NewCooperation(b)
	if e != nil {
		t.Fatal(e)
	}
	m, e := NewUPolynomialManager(UPolynomialConfig{Integer: q.Integers(), Checkpoint: limit.Inc, Cooperation: func() error { return cooperation.Cooperate("upolynomial") }})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := m.Close(); e != nil && !errors.Is(e, arena.ErrClosed) {
			t.Error(e)
		}
		if e := cooperation.Close(); e != nil {
			t.Error(e)
		}
		limit.Close() // actual source owner close is void
		q.Close()     // existing manager close is void; whole budget checked below
		if b.Used() != 0 {
			t.Error("owned graph bytes", b.Used())
		}
	})
	return m, q, b, limit, cooperation
}
func upolySourceOwner(t *testing.T) (*UPolynomialManager, *util.RationalManager, *arena.Budget) {
	m, q, b, _, _ := upolySourceOwnerServices(t)
	return m, q, b
}
func upolySourceNew(t *testing.T, m *UPolynomialManager, values []int32) UPolynomial {
	t.Helper()
	p, e := m.NewPolynomial()
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := m.DelPolynomial(&p); e != nil {
			t.Error(e)
		}
	})
	// Input literals are externally borrowed for the call; actual manager copies.
	nums := make([]util.Int, len(values))
	for i, v := range values {
		nums[i] = util.NewInt(v)
	}
	if e = m.SetInts(p, nums); e != nil {
		t.Fatal(e)
	}
	return p
}
func TestUPolynomialSourceNinePretranslationENSURE(t *testing.T) {
	m, q, _ := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{1})
	factor := upolySourceNew(t, m, nil)
	// Source constructs (x-1)*(x-2)*(x-3)*(x-4) with multivariate
	// polynomial_ref then converts. Actual dense factor construction is the
	// explicit input adapter; no multivariate conversion coverage is claimed.
	for i := int32(1); i <= 4; i++ {
		if e := m.SetInts(factor, []util.Int{util.NewInt(-i), util.NewInt(1)}); e != nil {
			t.Fatal(e)
		}
		if e := m.Mul(p, factor, p); e != nil {
			t.Fatal(e)
		}
	}
	view, e := m.View(p, 0, 5)
	if e != nil {
		t.Fatal(e)
	}
	for _, fixture := range []struct {
		n    int64
		zero bool
	}{{1, true}, {2, true}, {3, true}, {4, true}, {-1, false}, {5, false}, {-2, false}} {
		point := util.NewRational()
		if e = q.SetInt64(&point, fixture.n); e != nil {
			t.Fatal(e)
		}
		sign, e := m.EvalSignRational(view, &point)
		if e != nil || (sign == 0) != fixture.zero {
			t.Fatal(fixture, sign, e)
		}
		if e := q.Del(&point); e != nil {
			t.Fatal(e)
		}
	}
	// Source scoped_mpq c=1/3, r1=1-c, r2=3-c; actual output actions.
	c, r1, r2 := util.NewRational(), util.NewRational(), util.NewRational()
	if e = q.SetFraction(&c, 1, 3); e != nil {
		t.Fatal(e)
	}
	if e := q.SetInt64(&r1, 1); e != nil {
		t.Fatal(e)
	}
	if e = q.Sub(&r1, &c, &r1); e != nil {
		t.Fatal(e)
	}
	if e := q.SetInt64(&r2, 3); e != nil {
		t.Fatal(e)
	}
	if e = q.Sub(&r2, &c, &r2); e != nil {
		t.Fatal(e)
	}
	for _, r := range []*util.Rational{&r1, &r2} {
		sign, e := m.EvalSignRational(view, r)
		if e != nil || sign == 0 {
			t.Fatal(sign, e)
		}
	}
	if e := q.Del(&r2); e != nil {
		t.Fatal(e)
	}
	if e := q.Del(&r1); e != nil {
		t.Fatal(e)
	}
	if e := q.Del(&c); e != nil {
		t.Fatal(e)
	}
}
func TestUPolynomialSourceBasicCoefficientRoutes(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{1})
	factor := upolySourceNew(t, m, nil)
	for _, n := range []int32{-1, 1, 2, 3, -3} {
		if e := m.SetInts(factor, []util.Int{util.NewInt(n), util.NewInt(1)}); e != nil {
			t.Fatal(e)
		}
		if e := m.Mul(p, factor, p); e != nil {
			t.Fatal(e)
		}
	}
	// Frozen independent expansion of source tst1's five factors.
	for i, want := range []int32{18, 9, -20, -10, 2, 1} {
		var n util.Int
		if e := m.ReadCoefficient(p, uint32(i), &n); e != nil {
			t.Fatal(e)
		}
		got, e := m.IntegerManager().Int64(&n)
		if e != nil || got != int64(want) {
			t.Fatal(i, got, e)
		}
		if e := m.IntegerManager().Inc(&n); e != nil {
			t.Fatal(e)
		}
		if e = m.SetCoefficient(p, uint32(i), &n); e != nil {
			t.Fatal(e)
		}
		if e := m.IntegerManager().Del(&n); e != nil {
			t.Fatal(e)
		}
	}
	if e := m.Trim(p); e != nil {
		t.Fatal(e)
	}
	if e := m.Mul(p, p, p); e != nil {
		t.Fatal(e)
	}
	if e := m.Power(p, 3, p); e != nil {
		t.Fatal(e)
	}
	if degree, e := m.Degree(p); e != nil || degree != 30 {
		t.Fatal(degree, e)
	}
}

func TestUPolynomialStorageOwnershipAndPhysicalViews(t *testing.T) {
	m, _, b := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{3, 4})
	q := upolySourceNew(t, m, []int32{9})
	v, e := m.View(p, 0, 2)
	if e != nil {
		t.Fatal(e)
	}
	if e = m.Swap(p, q); e != nil {
		t.Fatal(e)
	}
	var n util.Int
	if e = m.ReadView(v, 0, &n); e != nil {
		t.Fatal(e)
	}
	if got, e := m.IntegerManager().Int64(&n); e != nil || got != 3 {
		t.Fatal(got, e)
	}
	if e := m.IntegerManager().Del(&n); e != nil {
		t.Fatal(e)
	}
	before := b.Used()
	b.FailAfter(0)
	if e = m.SetInts(q, []util.Int{util.NewInt(1), util.NewInt(2), util.NewInt(3)}); e == nil {
		t.Fatal("growth must refuse")
	}
	b.FailAfter(-1)
	if b.Used() != before {
		t.Fatal("failed growth leaked graph")
	}
	if e = m.ReadView(v, 1, &n); e != nil {
		t.Fatal(e)
	}
	if e := m.IntegerManager().Del(&n); e != nil {
		t.Fatal(e)
	}
	if e = m.SetInts(q, []util.Int{util.NewInt(1), util.NewInt(2), util.NewInt(3)}); e != nil {
		t.Fatal(e)
	}
	if e = m.ReadView(v, 0, &n); e == nil {
		t.Fatal("replaced physical backing remained readable")
	}
	if e = m.Close(); e == nil {
		t.Fatal("live scalar owners must block manager close")
	}
}

func TestUPolynomialFiniteArithmeticAndDirectDerivative(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	a := upolySourceNew(t, m, []int32{2, -3, 1})
	b := upolySourceNew(t, m, []int32{-1, 2})
	out := upolySourceNew(t, m, nil)
	checks := []struct {
		run  func() error
		want []int64
	}{
		{func() error { return m.Add(a, b, out) }, []int64{1, -1, 1}},
		{func() error { return m.Sub(a, b, out) }, []int64{3, -5, 1}},
		{func() error { return m.Mul(a, b, out) }, []int64{-2, 7, -7, 2}},
		{func() error { return m.Neg(a, out) }, []int64{-2, 3, -1}},
		{func() error { return m.Derivative(a, out) }, []int64{-3, 2}},
		{func() error { return m.Derivative(a, a) }, []int64{-3, 2}},
	}
	for i, c := range checks {
		if e := c.run(); e != nil {
			t.Fatal(i, e)
		}
		p := out
		if i == len(checks)-1 {
			p = a
		}
		size, e := m.Size(p)
		if e != nil || size != uint32(len(c.want)) {
			t.Fatal(i, size, e)
		}
		for j, want := range c.want {
			var n util.Int
			if e = m.ReadCoefficient(p, uint32(j), &n); e != nil {
				t.Fatal(e)
			}
			got, e := m.IntegerManager().Int64(&n)
			if e != nil || got != want {
				t.Fatal(i, j, got, e)
			}
			if e = m.IntegerManager().Del(&n); e != nil {
				t.Fatal(e)
			}
		}
	}
}

func TestUPolynomialViewOverloadsAndModularShape(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{4, 3})
	q := upolySourceNew(t, m, []int32{2, 1})
	out := upolySourceNew(t, m, nil)
	pv, e := m.View(p, 0, 2)
	if e != nil {
		t.Fatal(e)
	}
	qv, e := m.View(q, 0, 2)
	if e != nil {
		t.Fatal(e)
	}
	if e = m.AddViews(pv, qv, out); e != nil {
		t.Fatal(e)
	}
	if e = m.SubViews(pv, qv, out); e != nil {
		t.Fatal(e)
	}
	if e = m.NegView(pv, out); e != nil {
		t.Fatal(e)
	}
	if e = m.MulViews(pv, qv, out); e != nil {
		t.Fatal(e)
	}
	if e = m.DerivativeView(pv, out); e != nil {
		t.Fatal(e)
	}
	if e = m.PowerView(pv, 2, out); e != nil {
		t.Fatal(e)
	}
	eq, e := m.EqualViews(pv, pv)
	if e != nil || !eq {
		t.Fatal(eq, e)
	}
	if e = m.SetZPUint64(5); e != nil {
		t.Fatal(e)
	}
	finite, e := m.Modular()
	if e != nil || !finite {
		t.Fatal(finite, e)
	}
	field, e := m.Field()
	if e != nil || !field {
		t.Fatal(field, e)
	}
	if e = m.SetInts(out, []util.Int{util.NewInt(4), util.NewInt(5)}); e != nil {
		t.Fatal(e)
	}
	if size, e := m.Size(out); e != nil || size != 1 {
		t.Fatal(size, e)
	}
	var n util.Int
	if e = m.ReadCoefficient(out, 0, &n); e != nil {
		t.Fatal(e)
	}
	if got, e := m.IntegerManager().Int64(&n); e != nil || got != -1 {
		t.Fatal(got, e)
	}
	if e = m.IntegerManager().Del(&n); e != nil {
		t.Fatal(e)
	}
	if e = m.SetZ(); e != nil {
		t.Fatal(e)
	}
	stats := m.Stats()
	if stats.LivePolynomials != 3 || stats.LiveBackings == 0 || stats.OwnerBytes == 0 || stats.TotalBytes != stats.OwnerBytes+stats.StoreOwnerBytes+stats.DirectoryBytes+stats.PageBytes+stats.BackingBytes+stats.ModularBytes {
		t.Fatal(stats)
	}
}

func TestUPolynomialRawViewActualReadTiming(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{2, 3})
	out := upolySourceNew(t, m, nil)
	empty, e := m.View(p, 0, 0)
	if e != nil {
		t.Fatal(e)
	}
	stale, e := m.View(p, 0, 2)
	if e != nil {
		t.Fatal(e)
	}
	if e = m.SetInts(p, []util.Int{util.NewInt(1), util.NewInt(2), util.NewInt(3)}); e != nil {
		t.Fatal(e)
	}
	if sign, e := m.EvalSignInt(empty, nil); e != nil || sign != 0 {
		t.Fatal("zero count cannot read stale input", sign, e)
	}
	for _, run := range []func() (int, error){func() (int, error) { return m.EvalSignRational(empty, nil) }, func() (int, error) { return m.EvalSignBinary(empty, nil) }, func() (int, error) { return m.EvalSignZero(empty) }, func() (int, error) { return m.EvalSignPlusInfinity(empty) }, func() (int, error) { return m.EvalSignMinusInfinity(empty) }} {
		if sign, e := run(); e != nil || sign != 0 {
			t.Fatal("source zero count bypasses pointer read", sign, e)
		}
	}

	if e = m.SetView(out, stale); e == nil {
		t.Fatal("stale first coefficient must fail")
	}
	if size, e := m.Size(out); e != nil || size != 2 {
		t.Fatal("reserve/size prefix precedes stale input read", size, e)
	}
	a := upolySourceNew(t, m, []int32{1, 1})
	b := upolySourceNew(t, m, []int32{1, 2})
	scratchTarget := upolySourceNew(t, m, []int32{7, 8})
	other := upolySourceNew(t, m, []int32{9})
	captured, e := m.View(scratchTarget, 0, 2)
	if e != nil {
		t.Fatal(e)
	}
	if e = m.Mul(a, b, scratchTarget); e != nil {
		t.Fatal(e)
	}
	av, e := m.View(a, 0, 2)
	if e != nil {
		t.Fatal(e)
	}
	if e = m.MulViews(captured, av, other); e == nil {
		t.Fatal("captured scratch relocation must not keep old Go allocation readable")
	}
	var n util.Int
	if e = m.ReadCoefficient(other, 0, &n); e != nil {
		t.Fatal(e)
	}
	got, e := m.IntegerManager().Int64(&n)
	if e != nil || got != 9 {
		t.Fatal("output was published after failed scratch read", got, e)
	}
	if e = m.IntegerManager().Del(&n); e != nil {
		t.Fatal(e)
	}
}

func TestUPolynomialPredicatesTrimSourceStorage(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{5, 2})
	zero := util.NewInt(0)
	if e := m.SetCoefficient(p, 1, &zero); e != nil {
		t.Fatal(e)
	}
	if degree, e := m.Degree(p); e != nil || degree != 1 {
		t.Fatal("degree must not trim", degree, e)
	}
	if constant, e := m.IsConst(p); e != nil || !constant {
		t.Fatal("is_const trims", constant, e)
	}
	if size, e := m.Size(p); e != nil || size != 1 {
		t.Fatal(size, e)
	}
	if e := m.SetCoefficient(p, 0, &zero); e != nil {
		t.Fatal(e)
	}
	if iszero, e := m.IsZero(p); e != nil || !iszero {
		t.Fatal("is_zero trims", iszero, e)
	}
}

func TestUPolynomialIndependentScaledEvaluation(t *testing.T) {
	m, q, _ := upolySourceOwner(t)
	for _, coeff := range [][]int32{{}, {-7}, {3, -5, 0, 2}, {0, -3, 0, 1}, {-2, 0, 4}, {5, -1, 2, -3, 1}} {
		p := upolySourceNew(t, m, coeff)
		size, e := m.Size(p)
		if e != nil {
			t.Fatal(e)
		}
		view, e := m.View(p, 0, size)
		if e != nil {
			t.Fatal(e)
		}
		for n := int64(-4); n <= 4; n++ {
			for d := int64(1); d <= 4; d++ {
				point := util.NewRational()
				if e = q.SetFraction(&point, n, uint64(d)); e != nil {
					t.Fatal(e)
				}
				// Independent exact rational powers, not the integer-scaled Horner code.
				rat := new(big.Rat).SetFrac(big.NewInt(n), big.NewInt(d))
				sum := new(big.Rat)
				power := new(big.Rat).SetInt64(1)
				for _, c := range coeff {
					term := new(big.Rat).Mul(new(big.Rat).SetInt64(int64(c)), power)
					sum.Add(sum, term)
					power.Mul(power, rat)
				}
				sign, e := m.EvalSignRational(view, &point)
				if e != nil || sign != sum.Sign() {
					t.Fatal(coeff, n, d, sign, e, sum)
				}
				if e = q.Del(&point); e != nil {
					t.Fatal(e)
				}
				if d == 1 {
					integer := util.NewInt(int32(n))
					sign, e = m.EvalSignInt(view, &integer)
					if e != nil || sign != sum.Sign() {
						t.Fatal(coeff, n, sign, e)
					}
				}
			}
		}
		for k := uint32(0); k <= 4; k++ {
			for n := int32(-4); n <= 4; n++ {
				point := util.NewBinaryIntPower(n, k)
				rat := new(big.Rat).SetFrac(big.NewInt(int64(n)), new(big.Int).Lsh(big.NewInt(1), uint(k)))
				sum := new(big.Rat)
				power := new(big.Rat).SetInt64(1)
				for _, c := range coeff {
					sum.Add(sum, new(big.Rat).Mul(new(big.Rat).SetInt64(int64(c)), power))
					power.Mul(power, rat)
				}
				sign, e := m.EvalSignBinary(view, &point)
				if e != nil || sign != sum.Sign() {
					t.Fatal(coeff, n, k, sign, e, sum)
				}
			}
		}
	}
}

func TestUPolynomialRealCheckpointCancellationPrefixes(t *testing.T) {
	m, _, _, limit, cooperation := upolySourceOwnerServices(t)
	a := upolySourceNew(t, m, []int32{0, 1, 0, 2})
	b := upolySourceNew(t, m, []int32{1, 2})
	out := upolySourceNew(t, m, []int32{99})
	zero := upolySourceNew(t, m, nil)
	if e := cooperation.BeginSection(); e != nil {
		t.Fatal(e)
	}
	limit.Cancel()
	if e := m.Mul(a, b, out); !errors.Is(e, ErrUPolynomialCanceled) {
		t.Fatal(e)
	}
	// Source limit refusal precedes cooperation: actual noOMP owner remainsunset.
	if cooperation.OwnerThread != -1 {
		t.Fatal("cooperation ran after refusal", cooperation.OwnerThread)
	}
	if limit.Count() != 1 {
		t.Fatal(limit.Count())
	}
	if e := m.Mul(zero, b, out); e != nil {
		t.Fatal(e)
	}
	if limit.Count() != 1 {
		t.Fatal("zero source branch must omit checkpoint")
	}
	limit.ResetCancel()
	if e := m.Mul(a, b, out); e != nil {
		t.Fatal(e)
	}
	// Includes the two zero outer rows; longer input remains outer.
	if limit.Count() != 5 {
		t.Fatal("zero rows skipped checkpoint", limit.Count())
	}
	if e := cooperation.EndSection(); e != nil {
		t.Fatal(e)
	}
}

func TestUPolynomialImmediateNumericFailureAccounting(t *testing.T) {
	for fail := int64(0); fail < 24; fail++ {
		m, q, b := upolySourceOwner(t)
		p := upolySourceNew(t, m, nil)
		var z util.Int
		if e := q.Integers().SetString(&z, "1234567890123456789012345678901234567890"); e != nil {
			t.Fatal(e)
		}
		if e := m.SetInts(p, []util.Int{z, util.NewInt(0), util.NewInt(7)}); e != nil {
			t.Fatal(e)
		}
		if e := q.Integers().Del(&z); e != nil {
			t.Fatal(e)
		}
		point := util.NewRational()
		if e := q.SetFraction(&point, 9876543210123456, 12345678901234567); e != nil {
			t.Fatal(e)
		}
		v, e := m.View(p, 0, 3)
		if e != nil {
			t.Fatal(e)
		}
		base := m.Stats()
		external := base.IntegerLiveCells - base.OwnedNumeralCells - base.IntegerCachedCells - upolyIntegerRawOwnedCells(t, m.IntegerManager())
		b.FailAfter(fail)
		_, _ = m.EvalSignRational(v, &point)
		// Audit BEFORE any numeric formatting, deletion or owner close. These
		// observations count real storage independently from the arithmetic result.
		now := m.Stats()
		if now.IntegerLiveCells != external+now.OwnedNumeralCells+now.IntegerCachedCells+upolyIntegerRawOwnedCells(t, m.IntegerManager()) {
			t.Fatal("untracked numeric temporary", fail, base, now)
		}
		upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, nil, upolyPhysicalExtraOwners{Rationals: []*util.Rational{&point}, RationalManagers: []*util.RationalManager{q}})
		b.FailAfter(-1)
		if e = q.Del(&point); e != nil {
			t.Fatal(e)
		}
	}
}

func TestUPolynomialColdFailureAndWarmZeroAllocation(t *testing.T) {
	m, q, b, limit, cooperation := upolySourceOwnerServices(t)
	cfg := UPolynomialConfig{Integer: q.Integers(), Checkpoint: limit.Inc, Cooperation: func() error { return cooperation.Cooperate("upolynomial") }}
	baseline := b.Used()
	b.FailAfter(0)
	allocations := testing.AllocsPerRun(50, func() {
		owner, e := NewUPolynomialManager(cfg)
		if owner != nil || e == nil {
			panic("constructor budget preflight")
		}
	})
	b.FailAfter(-1)
	if allocations != 0 || b.Used() != baseline {
		t.Fatal(allocations, b.Used(), baseline)
	}
	copyOwner := *m
	if e := copyOwner.Close(); !errors.Is(e, ErrUPolynomialOwner) {
		t.Fatal(e)
	}
	var zeroOwner UPolynomialManager
	if e := zeroOwner.Close(); !errors.Is(e, ErrUPolynomialOwner) {
		t.Fatal(e)
	}
	p := upolySourceNew(t, m, []int32{1, 2})
	r := upolySourceNew(t, m, []int32{3, 4})
	out := upolySourceNew(t, m, nil)
	for i := 0; i < 4; i++ {
		if e := m.Add(p, r, out); e != nil {
			t.Fatal(e)
		}
		if e := m.Neg(out, out); e != nil {
			t.Fatal(e)
		}
	}
	before := m.Stats()
	used := b.Used()
	b.FailAfter(0)
	allocations = testing.AllocsPerRun(100, func() {
		if e := m.Add(p, r, out); e != nil {
			panic(e)
		}
		if e := m.Neg(out, out); e != nil {
			panic(e)
		}
	})
	b.FailAfter(-1)
	after := m.Stats()
	if allocations != 0 || used != b.Used() || before.TotalBytes != after.TotalBytes {
		t.Fatal("warm storage plateau", allocations, used, b.Used(), before, after)
	}
}

func TestUPolynomialSourceCopyAndRationalPrefixes(t *testing.T) {
	m, q, b := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{1, 2, 3})
	var large util.Int
	if e := q.Integers().SetString(&large, strings.Repeat("9", 1000)); e != nil {
		t.Fatal(e)
	}
	// Saturate the actual eight-cell pages; parsing may have left cached
	// blocks, so budget refusal must not assume a fresh limb allocation.
	var fillers [16]util.Int
	filled := 0
	for st := m.Stats(); st.IntegerLiveCells+st.IntegerRetiredCells < st.IntegerCellSlots || (st.IntegerLiveCells+st.IntegerRetiredCells)%8 != 0; st = m.Stats() {
		if filled == len(fillers) {
			t.Fatal("cell saturation bound", st)
		}
		if e := q.Integers().Copy(&fillers[filled], &large); e != nil {
			t.Fatal(e)
		}
		filled++
	}
	base := m.Stats()
	b.FailAfter(0)
	if e := m.SetInts(p, []util.Int{util.NewInt(7), large, util.NewInt(9)}); e == nil {
		t.Fatal("big coefficient must refuse fresh backing")
	}
	now := m.Stats()
	if now.IntegerLiveCells != base.IntegerLiveCells || now.IntegerWordsRefs != base.IntegerWordsRefs {
		t.Fatal("copy failure retained unnamed numerical root", base, now)
	}
	b.FailAfter(-1)
	var n util.Int
	if e := m.ReadCoefficient(p, 0, &n); e != nil {
		t.Fatal(e)
	}
	if got, e := q.Integers().Int64(&n); e != nil || got != 7 {
		t.Fatal("completed first copy lost", got, e)
	}
	if e := q.Integers().Del(&n); e != nil {
		t.Fatal(e)
	}
	// The second Copy published its sign and an owned failed-allocation
	// token. Reading that source-undefined pointer must remain guarded.
	if e := m.ReadCoefficient(p, 1, &n); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("failed second copy read guard", e)
	}
	if got, e := q.Integers().Int64(&n); e != nil || got != 7 {
		t.Fatal("rejected read changed caller destination", got, e)
	}
	if e := q.Integers().Del(&n); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < filled; i++ {
		if e := q.Integers().Del(&fillers[i]); e != nil {
			t.Fatal(e)
		}
	}
	if e := q.Integers().Del(&large); e != nil {
		t.Fatal(e)
	}
	one, fraction := util.NewRational(), util.NewRational()
	if e := q.SetInt64(&one, 1); e != nil {
		t.Fatal(e)
	}
	if e := q.SetFraction(&fraction, 2, 3); e != nil {
		t.Fatal(e)
	}
	target := upolySourceNew(t, m, nil)
	e := m.SetRationals(target, []util.Rational{one, fraction})
	if polynomialSourceDebug {
		if !errors.Is(e, ErrUPolynomialAssertion) {
			t.Fatal(e)
		}
	} else if e != nil {
		t.Fatal(e)
	}
	if size, e := m.Size(target); e != nil || size != 2 {
		t.Fatal("reserve prefix before integrality assertion", size, e)
	}
	if e := q.Del(&fraction); e != nil {
		t.Fatal(e)
	}
	if e := q.Del(&one); e != nil {
		t.Fatal(e)
	}
	v, e := m.View(p, 0, 0)
	if e != nil {
		t.Fatal(e)
	}
	e = m.SetView(p, v)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrUPolynomialAssertion) {
			t.Fatal(e)
		}
	} else if e != nil {
		t.Fatal(e)
	}
	if size, e := m.Size(p); e != nil || size != 3 {
		t.Fatal("source same-base zero-count copy must not reset", size, e)
	}
}

func TestUPolynomialConcreteScopedVectorActions(t *testing.T) {
	m, _, b := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{3, 4})
	captured, e := m.View(p, 0, 2)
	if e != nil {
		t.Fatal(e)
	}
	if e = m.Shrink(p, 1); e != nil {
		t.Fatal(e)
	}
	// Raw captured initialized slot remains source-readable after shrink.
	var n util.Int
	if e = m.ReadView(captured, 1, &n); e != nil {
		t.Fatal(e)
	}
	got, e := m.IntegerManager().Int64(&n)
	if e != nil || got != 4 {
		t.Fatal(got, e)
	}
	if e = m.IntegerManager().Del(&n); e != nil {
		t.Fatal(e)
	}
	if e = m.Resize(p, 2); e != nil {
		t.Fatal(e)
	}
	if e = m.ReadCoefficient(p, 1, &n); e != nil {
		t.Fatal(e)
	}
	got, e = m.IntegerManager().Int64(&n)
	if e != nil || got != 0 {
		t.Fatal("resize placement construction", got, e)
	}
	if e = m.IntegerManager().Del(&n); e != nil {
		t.Fatal(e)
	}
	b.FailAfter(0)
	e = m.PushCoefficient(p, &util.Int{})
	b.FailAfter(-1)
	if e == nil {
		t.Fatal("source growth-before-copy refusal", e)
	}
	if size, e := m.Size(p); e != nil || size != 2 {
		t.Fatal(size, e)
	}
	value := util.NewInt(8)
	if e = m.PushCoefficient(p, &value); e != nil {
		t.Fatal(e)
	}
	if e = m.PopCoefficient(p); e != nil {
		t.Fatal(e)
	}
	if size, e := m.Size(p); e != nil || size != 2 {
		t.Fatal(size, e)
	}
}

func TestUPolynomialSourceLifecycleAndNumericDestructionOrder(t *testing.T) {
	m, q, b := upolySourceOwner(t)
	base := m.Stats()
	// Real shared owner header includes all21 concrete source scratch states.
	if base.OwnerBytes != uint64(reflect.TypeOf(UPolynomialManager{}).Size()) || base.LiveBackings != 0 || base.BackingBytes != 0 {
		t.Fatal(base)
	}
	p, e := m.NewPolynomial()
	if e != nil {
		t.Fatal(e)
	}
	var n util.Int
	if e = q.Integers().SetString(&n, "123456789012345678901234567890"); e != nil {
		t.Fatal(e)
	}
	if e = m.SetInts(p, []util.Int{n, util.NewInt(1)}); e != nil {
		t.Fatal(e)
	}
	out, e := m.NewPolynomial()
	if e != nil {
		t.Fatal(e)
	}
	if e = m.SetInts(out, []util.Int{n}); e != nil {
		t.Fatal(e)
	}
	if e = m.Power(p, 3, out); e != nil {
		t.Fatal(e)
	}
	// Populate basic with an actual old owning output; leave source pw's old
	// output owned as well, so Close has real numerals to release in phase order.
	if e = m.Add(p, out, out); e != nil {
		t.Fatal(e)
	}
	if e = m.DelPolynomial(&out); e != nil {
		t.Fatal(e)
	}
	if e = m.DelPolynomial(&p); e != nil {
		t.Fatal(e)
	}
	if e = q.Integers().Del(&n); e != nil {
		t.Fatal(e)
	}
	before := m.Stats()
	if before.OwnedNumeralCells == 0 {
		t.Fatal("no real retained source scratch root")
	}
	external := before.IntegerLiveCells - before.OwnedNumeralCells - before.IntegerCachedCells
	record := reflect.ValueOf(m).Elem()
	pwBacking := testFieldBySourceName(testFieldBySourceName(record, "pw"), "backing").Uint()
	if pwBacking == 0 {
		t.Fatal("source power scratch lost old output")
	}
	lastSlot := uint32(testFieldBySourceName(testFieldBySourceName(testFieldBySourceName(testFieldBySourceName(record, "store").Elem(), "backings").Index(int(pwBacking-1)), "values").Index(0), "cell").Uint())
	if lastSlot == 0 {
		t.Fatal("power scratch must own actual big numeral")
	}
	if e = m.Close(); e != nil {
		t.Fatal(e)
	}
	// Inspect actual borrowed graph while it remains alive, before root cleanup.
	integer := reflect.ValueOf(q.Integers()).Elem()
	cells := testFieldBySourceName(integer, "cells").Elem()
	live := uint32(testFieldBySourceName(cells, "live").Uint())
	if live != external+before.IntegerCachedCells {
		t.Fatal("source scratch cleanup lost roots", live, before)
	}
	if e = q.Integers().SetInt64(&n, 1<<40); e != nil {
		t.Fatal(e)
	}
	reused := uint32(testFieldBySourceName(reflect.ValueOf(n), "cell").Uint())
	if reused != lastSlot {
		t.Fatal("actual destructor did not release pw after basic", reused, lastSlot)
	}
	if e = q.Integers().Del(&n); e != nil {
		t.Fatal(e)
	}
	// Remove this already-closed manager from helper's close expectation by
	// accepting ErrClosed in that helper; all borrowed owners still release.
	if b.Used() == 0 {
		t.Fatal("borrowed source services must remain live")
	}
}

func TestUPolynomialAllSelectedOutputFailurePrefixes(t *testing.T) {
	for op := 0; op < 7; op++ {
		for fail := int64(0); fail < 16; fail++ {
			m, q, b := upolySourceOwner(t)
			p := upolySourceNew(t, m, nil)
			other := upolySourceNew(t, m, nil)
			out := upolySourceNew(t, m, []int32{11, 12})
			var n util.Int
			if e := q.Integers().SetString(&n, "12345678901234567890123456789012345678901234567890"); e != nil {
				t.Fatal(e)
			}
			if e := m.SetInts(p, []util.Int{n, util.NewInt(3), util.NewInt(1)}); e != nil {
				t.Fatal(e)
			}
			if e := m.SetInts(other, []util.Int{n, util.NewInt(-2)}); e != nil {
				t.Fatal(e)
			}
			if e := q.Integers().Del(&n); e != nil {
				t.Fatal(e)
			}
			base := m.Stats()
			external := base.IntegerLiveCells - base.OwnedNumeralCells - base.IntegerCachedCells - upolyIntegerRawOwnedCells(t, m.IntegerManager())
			b.FailAfter(fail)
			switch op {
			case 0:
				_ = m.SetView(out, mustUPolyView(t, m, p))
			case 1:
				_ = m.Neg(p, out)
			case 2:
				_ = m.Add(p, other, out)
			case 3:
				_ = m.Sub(p, other, out)
			case 4:
				_ = m.Mul(p, other, out)
			case 5:
				_ = m.Derivative(p, out)
			case 6:
				_ = m.Power(p, 3, out)
			}
			now := m.Stats()
			if now.IntegerLiveCells != external+now.OwnedNumeralCells+now.IntegerCachedCells+upolyIntegerRawOwnedCells(t, m.IntegerManager()) {
				t.Fatal("unowned temporary after output prefix", op, fail, base, now)
			}
			upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, nil)
			b.FailAfter(-1)
		}
	}
}
func mustUPolyView(t *testing.T, m *UPolynomialManager, p UPolynomial) UPolynomialView {
	t.Helper()
	size, e := m.Size(p)
	if e != nil {
		t.Fatal(e)
	}
	v, e := m.View(p, 0, size)
	if e != nil {
		t.Fatal(e)
	}
	return v
}

func TestUPolynomialIndependentConvolutionAllAliases(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	a := upolySourceNew(t, m, nil)
	b := upolySourceNew(t, m, nil)
	out := upolySourceNew(t, m, nil)
	for _, modulus := range []int64{0, 5, 7} {
		if modulus == 0 {
			if e := m.SetZ(); e != nil {
				t.Fatal(e)
			}
		} else if e := m.SetZPUint64(uint64(modulus)); e != nil {
			t.Fatal(e)
		}
		for _, aa := range [][]int32{{}, {2}, {-2, 1}, {1, 0, -2, 1}} {
			for _, bb := range [][]int32{{}, {-1}, {2, -1, 1}} {
				for alias := 0; alias < 3; alias++ {
					numsA := make([]util.Int, len(aa))
					numsB := make([]util.Int, len(bb))
					for i, v := range aa {
						numsA[i] = util.NewInt(v)
					}
					for i, v := range bb {
						numsB[i] = util.NewInt(v)
					}
					if e := m.SetInts(a, numsA); e != nil {
						t.Fatal(e)
					}
					if e := m.SetInts(b, numsB); e != nil {
						t.Fatal(e)
					}
					dst := out
					if alias == 1 {
						dst = a
					}
					if alias == 2 {
						dst = b
					}
					var want []int64
					if len(aa) > 0 && len(bb) > 0 {
						want = make([]int64, len(aa)+len(bb)-1)
						for i, x := range aa {
							for j, y := range bb {
								want[i+j] += int64(x) * int64(y)
							}
						}
					}
					if modulus != 0 {
						for i := range want {
							want[i] %= modulus
							if want[i] < -(modulus / 2) {
								want[i] += modulus
							}
							if want[i] > modulus/2 {
								want[i] -= modulus
							}
						}
					}
					for len(want) > 0 && want[len(want)-1] == 0 {
						want = want[:len(want)-1]
					}
					if e := m.Mul(a, b, dst); e != nil {
						t.Fatal(e)
					}
					size, e := m.Size(dst)
					if e != nil || size != uint32(len(want)) {
						t.Fatal(modulus, aa, bb, alias, size, e, want)
					}
					for i, w := range want {
						var n util.Int
						if e = m.ReadCoefficient(dst, uint32(i), &n); e != nil {
							t.Fatal(e)
						}
						got, e := m.IntegerManager().Int64(&n)
						if e != nil || got != w {
							t.Fatal(modulus, aa, bb, alias, i, got, w, e)
						}
						if e = m.IntegerManager().Del(&n); e != nil {
							t.Fatal(e)
						}
					}
				}
			}
		}
	}
}

func TestUPolynomialScalarModeAndDirectCheckpoint(t *testing.T) {
	m, _, _, limit, _ := upolySourceOwnerServices(t)
	p := upolySourceNew(t, m, []int32{2, 1})
	zero := util.NewInt(0)
	one := util.NewInt(1)
	minusTwo := util.NewInt(-2)
	if e := m.MulScalar(p, &one); e != nil {
		t.Fatal(e)
	}
	if e := m.MulScalar(p, &minusTwo); e != nil {
		t.Fatal(e)
	}
	if e := m.MulScalar(p, &zero); e != nil {
		t.Fatal(e)
	}
	if size, e := m.Size(p); e != nil || size != 0 {
		t.Fatal(size, e)
	}
	before := limit.Count()
	if e := m.Checkpoint(); e != nil {
		t.Fatal(e)
	}
	if limit.Count() != before+1 {
		t.Fatal("direct checkpoint source action")
	}
	if e := m.SetZPUint64(7); e != nil {
		t.Fatal(e)
	}
	var modulus util.Int
	if e := m.Modulus(&modulus); e != nil {
		t.Fatal(e)
	}
	if got, e := m.IntegerManager().Int64(&modulus); e != nil || got != 7 {
		t.Fatal(got, e)
	}
	if e := m.IntegerManager().Del(&modulus); e != nil {
		t.Fatal(e)
	}
}

func TestUPolynomialRawMutableScalarOverloads(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{3, 4})
	v := mustUPolyView(t, m, p)
	if e := m.NegViewInPlace(v); e != nil {
		t.Fatal(e)
	}
	zero := util.NewInt(0)
	e := m.MulScalarView(v, &zero)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrUPolynomialAssertion) {
			t.Fatal(e)
		}
	} else if e != nil {
		t.Fatal(e)
	}
	if size, e := m.Size(p); e != nil || size != 2 {
		t.Fatal("raw scalar operation must not reset/trim vector", size, e)
	}
}
