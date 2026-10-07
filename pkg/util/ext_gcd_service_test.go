// Licensed under the MIT license.
// Source: src/util/ext_gcd.h and src/test/rational.cpp::tst7.
package util

import (
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestExtendedGCDWrapperSource999Cases(t *testing.T) {
	s, _ := serviceNew(t)
	two, err := s.NewValue()
	serviceMust(t, err)
	serviceMust(t, s.SetInt64(&two, 2))
	p, err := s.NewValue()
	serviceMust(t, err)
	power, err := s.Power(&two, 32)
	serviceMust(t, err)
	serviceMust(t, s.Copy(&p, &power))
	serviceMust(t, s.Del(&power))
	n, _ := s.NewValue()
	x, _ := s.NewValue()
	y, _ := s.NewValue()
	g, _ := s.NewValue()
	checks, assertions := 0, 0
	for i := int64(1); i < 1000; i++ {
		serviceMust(t, s.SetInt64(&n, i))
		serviceMust(t, s.ExtendedGCD(&n, &p, &g, &x, &y))
		parity, err := s.Mod(&n, &two)
		serviceMust(t, err)
		odd, err := s.IsOne(&parity)
		serviceMust(t, err)
		assertions++
		if odd {
			checks++
			product, err := s.Mul(&n, &x)
			serviceMust(t, err)
			residue, err := s.Mod(&product, &p)
			serviceMust(t, err)
			one, err := s.IsOne(&residue)
			serviceMust(t, err)
			if !one {
				t.Fatalf("source ENSURE case %d", i)
			}
			// Full-expression objects die after the condition is checked.
			serviceMust(t, s.Del(&residue))
			serviceMust(t, s.Del(&product))
			text, err := s.String(&x)
			serviceMust(t, err)
			coefficient, ok := new(big.Int).SetString(text, 10)
			if !ok || new(big.Int).Mod(new(big.Int).Mul(big.NewInt(i), coefficient), big.NewInt(4294967296)).Cmp(big.NewInt(1)) != 0 {
				t.Fatalf("source inverse case %d: x=%s", i, text)
			}
		}
		serviceMust(t, s.Del(&parity))
	}
	if checks != 500 || assertions != 999 {
		t.Fatal(checks, assertions)
	}
	for _, v := range []*OwnedRational{&p, &n, &x, &y, &g, &two} {
		serviceMust(t, s.Del(v))
	}
}

func TestExtendedGCDWrapperIndependentSignedBezout(t *testing.T) {
	s, _ := serviceNew(t)
	a, _ := s.NewValue()
	b, _ := s.NewValue()
	g, _ := s.NewValue()
	x, _ := s.NewValue()
	y, _ := s.NewValue()
	for i := int64(-12); i <= 12; i++ {
		for j := int64(-12); j <= 12; j++ {
			serviceMust(t, s.SetInt64(&a, i))
			serviceMust(t, s.SetInt64(&b, j))
			serviceMust(t, s.ExtendedGCD(&a, &b, &g, &x, &y))
			values := [3]*big.Int{}
			for k, v := range []*OwnedRational{&g, &x, &y} {
				text, err := s.String(v)
				serviceMust(t, err)
				var ok bool
				values[k], ok = new(big.Int).SetString(text, 10)
				if !ok {
					t.Fatal(text)
				}
			}
			lhs := new(big.Int).Add(new(big.Int).Mul(big.NewInt(i), values[1]), new(big.Int).Mul(big.NewInt(j), values[2]))
			want := new(big.Int).GCD(nil, nil, big.NewInt(i), big.NewInt(j))
			if lhs.Cmp(values[0]) != 0 || new(big.Int).Abs(values[0]).Cmp(want) != 0 {
				t.Fatalf("(%d,%d): g,x,y=%v", i, j, values)
			}
		}
	}
	for _, v := range []*OwnedRational{&a, &b, &g, &x, &y} {
		serviceMust(t, s.Del(v))
	}
}

func TestExtendedGCDWrapperImmediateFailureOwnership(t *testing.T) {
	failures, successes := 0, 0
	sawInitializedPrefix := false
	for prefix := int64(0); prefix < 600; prefix++ {
		s, budget := serviceNew(t)
		a := serviceValue(t, s, "123456789012345678901234567891")
		b := serviceValue(t, s, "99887766554433221101")
		g := serviceValue(t, s, "17")
		x := serviceValue(t, s, "19")
		y := serviceValue(t, s, "23")
		budget.FailAfter(prefix)
		err := s.ExtendedGCD(&a, &b, &g, &x, &y)
		if err != nil {
			failures++
			if !numericPhysicalReservationRefusal(err) {
				t.Fatalf("prefix %d: %v", prefix, err)
			}
			if g.Raw.IsSmall() && x.Raw.IsSmall() && y.Raw.IsSmall() &&
				g.Raw.Num.Val == 17 && g.Raw.Den.Val == 1 &&
				x.Raw.Num.Val == 0 && x.Raw.Den.Val == 1 &&
				y.Raw.Num.Val == 1 && y.Raw.Den.Val == 1 {
				if !sawInitializedPrefix {
					t.Logf("source initialized-output failure prefix: %d", prefix)
				}
				sawInitializedPrefix = true
			}
		} else {
			successes++
		}
		// Outputs may publish source prefixes. Count actual reachable owners
		// before String, caller Del, or Close can hide an intermediate leak.
		var roots []*Int
		for _, v := range []*OwnedRational{&a, &b, &g, &x, &y} {
			roots = append(roots, &v.Raw.Num, &v.Raw.Den)
		}
		// This independent graph includes the source manager's fixed argument,
		// result and constant cells, plus legitimately retained natural scratch.
		qopsAuditLiveOwnership(t, s.Manager, roots)
		budget.FailAfter(-1)
		serviceText(t, s, &a, "123456789012345678901234567891")
		serviceText(t, s, &b, "99887766554433221101")
		serviceMust(t, s.Close())
		if budget.Used() != 0 {
			t.Fatal(prefix, budget.Used())
		}
		if successes == 2 {
			break
		}
	}
	if failures == 0 || successes == 0 || !sawInitializedPrefix {
		t.Fatal(failures, successes, sawInitializedPrefix)
	}
}

func TestExtendedGCDWrapperSourceOutputAliases(t *testing.T) {
	s, _ := serviceNew(t)
	a := serviceValue(t, s, "12")
	b := serviceValue(t, s, "8")
	g := serviceValue(t, s, "0")
	x := serviceValue(t, s, "0")
	y := serviceValue(t, s, "0")
	serviceMust(t, s.ExtendedGCD(&a, &b, &g, &a, &b))
	serviceText(t, s, &g, "4")
	serviceText(t, s, &a, "1")
	serviceText(t, s, &b, "-1")
	serviceMust(t, s.SetInt64(&a, 12))
	serviceMust(t, s.SetInt64(&b, 8))
	serviceMust(t, s.ExtendedGCD(&a, &b, &g, &x, &x))
	serviceText(t, s, &g, "4")
	serviceText(t, s, &x, "1")
	serviceMust(t, s.ExtendedGCD(&a, &b, &x, &x, &y))
	serviceText(t, s, &x, "1") // final coefficient overwrites gcd publication.
	serviceText(t, s, &y, "-1")
	for _, v := range []*OwnedRational{&a, &b, &g, &x, &y} {
		serviceMust(t, s.Del(v))
	}
}

func TestExtendedGCDWrapperProvenanceGuards(t *testing.T) {
	s, budget := serviceNew(t)
	other, _ := serviceNew(t)
	a := serviceValue(t, s, "12")
	b := serviceValue(t, s, "8")
	g := serviceValue(t, s, "17")
	x := serviceValue(t, s, "19")
	y := serviceValue(t, s, "23")
	foreign := serviceValue(t, other, "12")
	for index := 0; index < 5; index++ {
		used := budget.Used()
		args := [5]*OwnedRational{&a, &b, &g, &x, &y}
		args[index] = &foreign
		if err := s.ExtendedGCD(args[0], args[1], args[2], args[3], args[4]); !errors.Is(err, ErrRationalServiceOwner) {
			t.Fatal(index, err)
		}
		if budget.Used() != used {
			t.Fatal("foreign operand changed budget", index)
		}
		serviceText(t, s, &g, "17")
		serviceText(t, s, &x, "19")
		serviceText(t, s, &y, "23")
	}
	for _, v := range []*OwnedRational{&a, &b, &g, &x, &y} {
		serviceMust(t, s.Del(v))
	}
	serviceMust(t, other.Del(&foreign))
}

// The actual coefficient free operator must retain its failed COPY prefix
// until the enclosing extended_gcd locals have run their source destructors.
func TestExtendedGCDWrapperCoefficientOuterBoundary(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	cfg := serviceConfig(b)
	cfg.Integers.Profile = SmallAllocatorProfile{PointerBits: 64, Debug: true}
	s, err := NewRationalService(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if e := s.Close(); e != nil {
			t.Error(e)
		}
		if b.Used() != 0 {
			t.Error(b.Used())
		}
	}()
	q := serviceValue(t, s, "18446744073709551617")
	current := serviceValue(t, s, "7")
	last := serviceValue(t, s, "18446744073709551619")
	outer := NewRational()
	if err = s.Manager.Copy(&outer, &last.Raw); err != nil {
		t.Fatal(err)
	}
	if err = s.beginLexical(); err != nil {
		t.Fatal(err)
	}
	b.FailAfter(0)
	err = s.extendedGCDCoefficientLocked(&q, &current, &last)
	st, _ := s.LexicalState()
	if !numericPhysicalReservationRefusal(err) || st.Aborts != 1 || st.Completed != 0 {
		t.Fatal(err, st)
	}
	serviceAuditRationalRoots(t, s, &q.Raw, &current.Raw, &last.Raw, &outer, &s.Lexical.Roots[0])
	b.FailAfter(-1)
	if e := s.Manager.Z.CleanupPendingAllocation(); e != nil {
		t.Fatal(e)
	}
	if e := s.destroyLexicalRational(&outer); e != nil {
		t.Fatal(e)
	}
	st, _ = s.LexicalState()
	if st.Aborts != 1 {
		t.Fatal("source outer dtor consumed failed inner copy", st)
	}
	s.endLexicalError(&err)
	st, _ = s.LexicalState()
	if st.Pending {
		t.Fatal(st)
	}
	serviceAuditRationalRoots(t, s, &q.Raw, &current.Raw, &last.Raw)
	for _, v := range []*OwnedRational{&q, &current, &last} {
		serviceMust(t, s.Del(v))
	}
}

// This bounded recipe models only ext_gcd.h's two initial COPY constructors.
// A genuine second-component allocation refusal must unwind completed a before
// owning-aborting partial b. The resulting real cell free-list order is observable.
func TestExtendedGCDWrapperCopyConstructorUnwindOrder(t *testing.T) {
	run := func(prefix int64, reference bool) (bool, [3]uint32) {
		budget := arena.NewBudget(1 << 24)
		cfg := serviceConfig(budget)
		cfg.Integers.Profile = SmallAllocatorProfile{PointerBits: 64, Debug: true}
		s, e := NewRationalService(cfg)
		if e != nil {
			t.Fatal(e)
		}
		defer func() {
			if e := s.Close(); e != nil {
				t.Error(e)
			}
			if budget.Used() != 0 {
				t.Error(budget.Used())
			}
		}()
		inA := serviceValue(t, s, "1208925819614629174706177")
		inB := serviceValue(t, s, "1208925819614629174706183/1208925819614629174706185")
		g, x, y := serviceValue(t, s, "17"), serviceValue(t, s, "19"), serviceValue(t, s, "23")
		budget.FailAfter(prefix)
		eligible := false
		if reference {
			a, b := NewRational(), NewRational()
			e = s.Manager.Copy(&a, &inA.Raw)
			if e == nil {
				e = s.Manager.Copy(&b, &inB.Raw)
				eligible = numericPhysicalReservationRefusal(e) && b.Num.Cell != 0 && !s.Manager.Z.IsFailureRef(b.Num.Cell) && b.Den.Cell == s.Manager.Z.FailedNullRef
				if eligible {
					serviceAuditRationalRoots(t, s, &inA.Raw, &inB.Raw, &g.Raw, &x.Raw, &y.Raw, &a, &b)
					// Source completed a dtor precedes separate owning abort of b.
					if e := s.destroyLexicalRational(&a); e != nil {
						t.Fatal(e)
					}
					if e := s.abortLexicalRational(&b); e != nil {
						t.Fatal(e)
					}
				} else {
					serviceMust(t, s.Manager.Del(&b))
					serviceMust(t, s.Manager.Del(&a))
				}
			} else {
				serviceMust(t, s.Manager.Del(&a))
			}
		} else {
			e = s.ExtendedGCD(&inA, &inB, &g, &x, &y)
			eligible = numericPhysicalReservationRefusal(e)
			if g.Raw.Num.Val != 17 || x.Raw.Num.Val != 19 || y.Raw.Num.Val != 23 {
				t.Fatal("unexpected later action", prefix, e)
			}
			serviceAuditRationalRoots(t, s, &inA.Raw, &inB.Raw, &g.Raw, &x.Raw, &y.Raw)
		}
		budget.FailAfter(-1)
		if e := s.Manager.Z.CleanupPendingAllocation(); e != nil {
			t.Fatal(e)
		}
		if e := s.CleanupLexical(); e != nil {
			t.Fatal(e)
		}
		probe := [3]Int{}
		slots := [3]uint32{}
		for i := range probe {
			if e := s.Manager.Z.SetString(&probe[i], "18446744073709551617"); e != nil {
				t.Fatal(e)
			}
			slots[i] = uint32(NumericSlot(probe[i].Cell))
		}
		serviceAuditRationalRoots(t, s, &inA.Raw, &inB.Raw, &g.Raw, &x.Raw, &y.Raw, &Rational{Num: probe[0], Den: NewInt(1)}, &Rational{Num: probe[1], Den: NewInt(1)}, &Rational{Num: probe[2], Den: NewInt(1)})
		for i := range probe {
			if e := s.Manager.Z.Del(&probe[i]); e != nil {
				t.Fatal(e)
			}
		}

		for _, v := range []*OwnedRational{&inA, &inB, &g, &x, &y} {
			serviceMust(t, s.Del(v))
		}
		return eligible, slots
	}
	found := 0
	for prefix := int64(0); prefix < 40; prefix++ {
		eligible, want := run(prefix, true)
		if !eligible {
			continue
		}
		found++
		ok, got := run(prefix, false)
		if !ok || got != want {
			t.Fatalf("prefix %d actual ctor-unwind slots%v, source slots%v", prefix, got, want)
		}
	}
	if found == 0 {
		t.Fatal("no actual partial-B allocation refusal reached")
	}
}

// A warmed all-inline Euclidean route makes no raw source allocation or Go
// allocation. Caller inputs remain independent and outputs satisfy Bezout.
func TestExtendedGCDWrapperWarmPhysicalBoundary(t *testing.T) {
	s, budget := serviceNew(t)
	a, b := serviceValue(t, s, "12"), serviceValue(t, s, "8")
	g, x, y := serviceValue(t, s, "0"), serviceValue(t, s, "0"), serviceValue(t, s, "0")
	serviceMust(t, s.ExtendedGCD(&a, &b, &g, &x, &y))
	before := numericWarmProfileSnapshot(t, s.Manager.Z, budget)
	budget.FailAfter(0)
	allocs := testing.AllocsPerRun(100, func() {
		if e := s.ExtendedGCD(&a, &b, &g, &x, &y); e != nil {
			panic(e)
		}
	})
	serviceAuditRationalRoots(t, s, &a.Raw, &b.Raw, &g.Raw, &x.Raw, &y.Raw)
	numericWarmProfileAudit(t, s.Manager.Z, budget, before, allocs, false)
	budget.FailAfter(-1)
	if g.Raw.Num.Val != 4 || x.Raw.Num.Val != 1 || y.Raw.Num.Val != -1 {
		t.Fatal(g, x, y)
	}
	for _, v := range []*OwnedRational{&a, &b, &g, &x, &y} {
		serviceMust(t, s.Del(v))
	}
}

// Six outer locals, one coefficient product, one completed inner work and an
// actual failed-return-Copy adapter token form the nine-record continuation
// envelope. The work refusal is explicitly malformed Go layout, not malloc.
func TestExtendedGCDWrapperNineRetainedDiagnosticRecords(t *testing.T) {
	if len((rationalLexicalScope{}).Roots) != 9 {
		t.Fatal("external source stack requires actual ninth charged record")
	}
	budget := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: budget, SizeTBits: 32})
	if e != nil {
		t.Fatal(e)
	}
	cfg := serviceConfig(budget)
	cfg.Integers.Memory = r
	cfg.Integers.Profile = SmallAllocatorProfile{PointerBits: 32, Debug: true}
	s, e := NewRationalService(cfg)
	if e != nil {
		t.Fatal(e)
	}
	z := s.Manager.Z
	completed := [8]Rational{}
	for i := range completed {
		completed[i] = NewRational()
		if e = s.Manager.SetString(&completed[i], "18446744073709551617"); e != nil {
			t.Fatal(e)
		}
	}
	partial := NewRational()
	if e = s.beginLexical(); e != nil {
		t.Fatal(e)
	}
	budget.FailAfter(0)
	e = z.Copy(&partial.Num, &completed[0].Num)
	if !numericPhysicalReservationRefusal(e) || partial.Num.Cell != z.FailedNullRef {
		t.Fatal(e)
	}
	serviceAuditRationalRoots(t, s, &completed[0], &completed[1], &completed[2], &completed[3], &completed[4], &completed[5], &completed[6], &completed[7], &partial)
	budget.FailAfter(-1)
	workRef := completed[7].Num.Cell
	workCell := z.Cell(workRef)
	bad := workCell
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	z.Store(workRef, bad)
	defer func() {
		if ValidNumericRef(z.Cells, workRef) {
			c := z.Cell(workRef)
			c.Backing.Requested = workCell.Backing.Requested
			z.Store(workRef, c)
		}
		if e := s.CleanupLexical(); e != nil {
			t.Error(e)
		}
		if e := s.Close(); e != nil {
			t.Error(e)
		}
		if e := r.Close(); e != nil {
			t.Error(e)
		}
		if budget.Used() != 0 {
			t.Error(budget.Used())
		}
	}()
	if next := s.abortLexicalRational(&partial); next != nil {
		t.Fatal(next)
	}
	for i := 7; i >= 0; i-- {
		if next := s.destroyLexicalRational(&completed[i]); !errors.Is(next, arena.ErrBounds) {
			t.Fatal(i, next)
		}
	}
	s.endLexicalError(&e)
	st, _ := s.LexicalState()
	if st.Roots != 9 || st.Completed != 8 || st.Aborts != 1 || !errors.Is(st.Cause, arena.ErrBounds) || !numericPhysicalReservationRefusal(st.OriginalCause) {
		t.Fatal(st)
	}
	var roots []*Rational
	for i := uint8(0); i < s.Lexical.Count; i++ {
		roots = append(roots, &s.Lexical.Roots[i])
	}
	serviceAuditRationalRoots(t, s, roots...)
	c := z.Cell(workRef)
	c.Backing.Requested = workCell.Backing.Requested
	z.Store(workRef, c)
	if e = s.CleanupLexical(); e != nil {
		t.Fatal(e)
	}
	if e = z.CleanupPendingAllocation(); e != nil {
		t.Fatal(e)
	}
	serviceAuditRationalRoots(t, s)
}
