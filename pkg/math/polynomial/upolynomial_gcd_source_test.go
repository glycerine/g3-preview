package polynomial

import (
	"errors"
	"math/big"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func upolyGCDExpect(t *testing.T, m *UPolynomialManager, p UPolynomial, want []int64) {
	t.Helper()
	got := upolyDivisionValues(t, m, p)
	if len(got) != len(want) {
		t.Fatal(got, want)
	}
	for i, v := range want {
		if got[i] != v {
			t.Fatal(got, want)
		}
	}
}
func TestUPolynomialContentNormalizationSourceBranches(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	a := upolySourceNew(t, m, []int32{-12, 18, -6})
	p := upolySourceNew(t, m, nil)
	var c unumeral.Int
	if e := m.PrimitiveAndContent(a, p, &c); e != nil {
		t.Fatal(e)
	}
	n, e := m.IntegerManager().Int64(&c)
	if e != nil || n != 6 {
		t.Fatal(n, e)
	}
	if e = m.IntegerManager().Del(&c); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, p, []int64{-2, 3, -1})
	if e = m.Normalize(a); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, a, []int64{-2, 3, -1})
	z := upolySourceNew(t, m, nil)
	if e = m.Reserve(z, 1); e != nil {
		t.Fatal(e)
	}
	if e = m.Normalize(z); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, z, []int64{-1}) // actual singleton-zero branch
	if e = m.Primitive(a, a); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, a, []int64{-2, 3, -1})
}
func TestUPolynomialRemainderSignedSourceBranches(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	a := upolySourceNew(t, m, []int32{1, 1, 1})
	b := upolySourceNew(t, m, []int32{1, -2})
	r := upolySourceNew(t, m, nil)
	var d uint32
	if e := m.Rem(a, b, r, &d); e != nil {
		t.Fatal(e)
	}
	if d != 2 {
		t.Fatal(d)
	}
	upolyGCDExpect(t, m, r, []int64{7})
	if e := m.SignedRem(a, b, r); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, r, []int64{-7})
	linear := upolySourceNew(t, m, []int32{1, 1})
	if e := m.Rem(linear, b, r, &d); e != nil {
		t.Fatal(e)
	}
	if d != 1 {
		t.Fatal(d)
	}
	upolyGCDExpect(t, m, r, []int64{-3})
	if e := m.SignedRem(linear, b, r); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, r, []int64{-3}) // odd d and negative LC: no negation
}
func TestUPolynomialExtendedGCDSourceZ7(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	if e := m.SetZPUint64(7); e != nil {
		t.Fatal(e)
	}
	a := upolySourceNew(t, m, []int32{2, 0, 0, 1})
	b := upolySourceNew(t, m, []int32{6, 0, 6})
	u := upolySourceNew(t, m, nil)
	v := upolySourceNew(t, m, nil)
	d := upolySourceNew(t, m, nil)
	if e := m.ExtendedGCD(a, b, u, v, d); e != nil {
		t.Fatal(e)
	}
	// Source comment 3*x+6, 3*x^2+6*x+4, 1 converted to its actual centered representatives.
	upolyGCDExpect(t, m, u, []int64{-1, 3})
	upolyGCDExpect(t, m, v, []int64{-3, -1, 3})
	upolyGCDExpect(t, m, d, []int64{1})
	if e := m.SetZ(); e != nil {
		t.Fatal(e)
	}
}
func TestUPolynomialGCDAndSquareFreeIndependent(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	// A=6*(x-1)^2*(x+2), B=9*(x-1)*(x+3). Primitive factor expected independently expanded.
	a := upolySourceNew(t, m, []int32{12, -18, 0, 6})
	b := upolySourceNew(t, m, []int32{-27, 18, 9})
	g := upolySourceNew(t, m, nil)
	if e := m.GCD(a, b, g); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, g, []int64{-3, 3})
	if e := m.EuclidGCD(a, b, g); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, g, []int64{-1, 1})
	if e := m.SubresultantGCD(a, b, g); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, g, []int64{-1, 1})
	if e := m.SquareFree(a, g); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, g, []int64{-2, 1, 1})
	yes, e := m.IsSquareFree(g)
	if e != nil || !yes {
		t.Fatal(yes, e)
	}
	yes, e = m.IsSquareFree(a)
	if e != nil || yes {
		t.Fatal(yes, e)
	}
}

func TestUPolynomialGCDRegisteredZAndZPInputs(t *testing.T) {
	// Source tst_zp/tst_zp2 are print-only; this executes their actual dense
	// GCD/subresultant calls. Independent checks require a common monic divisor
	// in fields and divisibility in Z, without inventing source ENSURE counts.
	for _, f := range []struct {
		a, b  []int32
		prime uint64
	}{
		{[]int32{0, 1, 2, 2, 1}, []int32{1, 1, 0, 1}, 0},
		{[]int32{0, 1, 2, 2, 1}, []int32{1, 1, 0, 1}, 3},
		{[]int32{8, 2, 8, 10, 10, 0, 1, 0, 1}, []int32{0, 5, 5, 0, 9, 5, 1}, 0},
		{[]int32{8, 2, 8, 10, 10, 0, 1, 0, 1}, []int32{0, 5, 5, 0, 9, 5, 1}, 13},
	} {
		m, _, _ := upolySourceOwner(t)
		if f.prime != 0 {
			if e := m.SetZPUint64(f.prime); e != nil {
				t.Fatal(e)
			}
		}
		a := upolySourceNew(t, m, f.a)
		b := upolySourceNew(t, m, f.b)
		g := upolySourceNew(t, m, nil)
		for _, sub := range []bool{false, true} {
			var e error
			if sub {
				e = m.SubresultantGCD(a, b, g)
			} else {
				e = m.GCD(a, b, g)
			}
			if e != nil {
				t.Fatal(f.prime, sub, e)
			}
			av := upolyDivisionValues(t, m, a)
			bv := upolyDivisionValues(t, m, b)
			gv := upolyDivisionValues(t, m, g)
			if len(gv) == 0 {
				t.Fatal("zero gcd")
			}
			if f.prime != 0 && gv[len(gv)-1] != 1 {
				t.Fatal("nonmonic", gv)
			}
			if !upolyIndependentDivides(av, gv, int64(f.prime)) || !upolyIndependentDivides(bv, gv, int64(f.prime)) {
				t.Fatal(f.prime, sub, av, bv, gv)
			}
		}
		if e := m.SetZ(); e != nil {
			t.Fatal(e)
		}
	}
}

// Finite independent long division over Q or a small field. This is a test
// oracle: no manager calls, pseudo-division counters, scratch swaps or CRA.
func upolyIndependentDivides(a, b []int64, p int64) bool {
	if len(b) == 0 {
		return false
	}
	r := append([]int64(nil), a...)
	for len(r) >= len(b) {
		top := r[len(r)-1]
		lead := b[len(b)-1]
		ratio := int64(0)
		if p == 0 {
			if lead == 0 || top%lead != 0 {
				return false
			}
			ratio = top / lead
		} else {
			inv := int64(0)
			for k := int64(1); k < p; k++ {
				if (lead*k%p+p)%p == 1 {
					inv = k
					break
				}
			}
			if inv == 0 {
				return false
			}
			ratio = (top * inv) % p
		}
		delta := len(r) - len(b)
		for i, v := range b {
			r[delta+i] -= ratio * v
			if p != 0 {
				r[delta+i] %= p
			}
		}
		for len(r) > 0 && r[len(r)-1] == 0 {
			r = r[:len(r)-1]
		}
	}
	return len(r) == 0
}
func TestUPolynomialExtendedGCDRegisteredZ13Input(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	if e := m.SetZPUint64(13); e != nil {
		t.Fatal(e)
	}
	a := upolySourceNew(t, m, []int32{0, 5, 5, 0, 9, 5, 1})
	b := upolySourceNew(t, m, []int32{8, 2, 8, 10, 10, 0, 1, 0, 1})
	u := upolySourceNew(t, m, nil)
	v := upolySourceNew(t, m, nil)
	d := upolySourceNew(t, m, nil)
	if e := m.ExtendedGCD(a, b, u, v, d); e != nil {
		t.Fatal(e)
	}
	av, bv, uv, vv, dv := upolyDivisionValues(t, m, a), upolyDivisionValues(t, m, b), upolyDivisionValues(t, m, u), upolyDivisionValues(t, m, v), upolyDivisionValues(t, m, d)
	sum := make([]int64, 32)
	for i, x := range av {
		for j, y := range uv {
			sum[i+j] += x * y
		}
	}
	for i, x := range bv {
		for j, y := range vv {
			sum[i+j] += x * y
		}
	}
	for i, x := range sum {
		want := int64(0)
		if i < len(dv) {
			want = dv[i]
		}
		if (x-want)%13 != 0 {
			t.Fatal(i, x, want)
		}
	}
	if len(dv) == 0 || dv[len(dv)-1] != 1 || !upolyIndependentDivides(av, dv, 13) || !upolyIndependentDivides(bv, dv, 13) {
		t.Fatal(dv)
	}
	if e := m.SetZ(); e != nil {
		t.Fatal(e)
	}
}

func TestUPolynomialGCDRegisteredFirstDerivativeExample(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	p := upolySourceNew(t, m, []int32{13})
	factor := upolySourceNew(t, m, nil)
	q := upolySourceNew(t, m, nil)
	g := upolySourceNew(t, m, nil)
	expected := []big.Int{*big.NewInt(1)}
	for _, f := range []struct {
		root  int32
		power int
	}{{3, 6}, {5, 5}, {11, 7}} {
		if e := m.SetInts(factor, []unumeral.Int{unumeral.NewInt(-f.root), unumeral.NewInt(1)}); e != nil {
			t.Fatal(e)
		}
		for k := 0; k < f.power; k++ {
			if e := m.Mul(p, factor, p); e != nil {
				t.Fatal(e)
			}
		}
		// Independent expected gcd retains each repeated root one fewer time.
		for k := 0; k < f.power-1; k++ {
			next := make([]big.Int, len(expected)+1)
			for i := range expected {
				var tmp big.Int
				tmp.Mul(&expected[i], big.NewInt(-int64(f.root)))
				next[i].Add(&next[i], &tmp)
				next[i+1].Add(&next[i+1], &expected[i])
			}
			expected = next
		}
	}
	if e := m.Derivative(p, q); e != nil {
		t.Fatal(e)
	}
	for _, sub := range []bool{false, true} {
		var e error
		if sub {
			e = m.SubresultantGCD(p, q, g)
		} else {
			e = m.GCD(p, q, g)
		}
		if e != nil {
			t.Fatal(sub, e)
		}
		n, e := m.Size(g)
		if e != nil || int(n) != len(expected) {
			t.Fatal(n, e)
		}
		for i := range expected {
			want := new(big.Int).Set(&expected[i])
			if !sub {
				want.Mul(want, big.NewInt(13))
			}
			var got unumeral.Int
			if e = m.ReadCoefficient(g, uint32(i), &got); e != nil {
				t.Fatal(e)
			}
			text, e := m.IntegerManager().String(&got)
			if e != nil {
				t.Fatal(e)
			}
			if e = m.IntegerManager().Del(&got); e != nil {
				t.Fatal(e)
			}
			if text != want.String() {
				t.Fatal(sub, i, text, want.String())
			}
		}
	}
}

func TestUPolynomialGCDChainImmediateFailureRoots(t *testing.T) {
	for op := 0; op < 12; op++ {
		for fail := int64(0); fail < 48; fail++ {
			m, q, b := upolySourceOwner(t)
			a := upolySourceNew(t, m, nil)
			other := upolySourceNew(t, m, nil)
			out := upolySourceNew(t, m, []int32{11, 12})
			v := upolySourceNew(t, m, nil)
			dout := upolySourceNew(t, m, nil)
			var large unumeral.Int
			if e := q.Integers().SetString(&large, "12345678901234567890123456789012345678901234567890"); e != nil {
				t.Fatal(e)
			}
			if op == 10 {
				if e := m.SetZPUint64(13); e != nil {
					t.Fatal(e)
				}
			}
			if e := m.SetInts(a, []unumeral.Int{large, unumeral.NewInt(2), unumeral.NewInt(1)}); e != nil {
				t.Fatal(e)
			}
			if e := m.SetInts(other, []unumeral.Int{large, unumeral.NewInt(3), unumeral.NewInt(1)}); e != nil {
				t.Fatal(e)
			}
			if e := q.Integers().Del(&large); e != nil {
				t.Fatal(e)
			}
			before := m.Stats()
			external := before.IntegerLiveCells - before.OwnedNumeralCells - before.IntegerCachedCells
			var content, lc, inv unumeral.Int
			two := unumeral.NewInt(2)
			counter := uint32(93)
			b.FailAfter(fail)
			var err error
			switch op {
			case 0:
				err = m.PrimitiveAndContent(a, out, &content)
			case 1:
				err = m.Normalize(a)
			case 2:
				err = m.DivScalar(a, &two)
			case 3:
				err = m.DivRem(a, other, out, v, &counter)
			case 4:
				err = m.Rem(a, other, out, &counter)
			case 5:
				_, err = m.ExactDiv(a, other, out)
			case 6:
				_, err = m.Divides(a, other)
			case 7:
				err = m.GCD(a, other, out)
			case 8:
				err = m.EuclidGCD(a, other, out)
			case 9:
				err = m.SubresultantGCD(a, other, out)
			case 10:
				err = m.ExtendedGCD(a, other, out, v, dout)
			case 11:
				err = m.MakeMonicWithLeadingInverse(a, &lc, &inv)
			}
			// Inspect actual numeric roots before formatting, caller Del or Close.
			// The only permitted additions outside the manager graph are the three
			// actual source output numerals, not an allowance for unnamed temporaries.
			outputs := uint32(0)
			for _, x := range []unumeral.Int{content, lc, inv} {
				if testFieldBySourceName(reflect.ValueOf(x), "cell").Uint() != 0 {
					outputs++
				}
			}
			now := m.Stats()
			if now.IntegerLiveCells != external+outputs+now.OwnedNumeralCells+now.IntegerCachedCells {
				t.Fatal("unnamed local root", op, fail, err, before, now, outputs)
			}
			upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, nil, upolyPhysicalExtraOwners{Integers: []*unumeral.Int{&content, &lc, &inv}, RationalManagers: []*unumeral.RationalManager{q}})
			b.FailAfter(-1)
			for _, x := range []*unumeral.Int{&inv, &lc, &content} {
				if e := q.Integers().Del(x); e != nil {
					t.Fatal(e)
				}
			}
			if e := m.SetZ(); e != nil {
				t.Fatal(e)
			}
		}
	}
}
func TestUPolynomialGCDRawZeroCountsAndSourceAliases(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	a := upolySourceNew(t, m, []int32{2, 1})
	b := upolySourceNew(t, m, []int32{1, 1})
	out := upolySourceNew(t, m, nil)
	captured := mustUPolyView(t, m, a)
	if e := m.Reset(a); e != nil {
		t.Fatal(e)
	}
	if e := m.Reserve(a, 20); e != nil {
		t.Fatal(e)
	}
	// Zero-count operations do not touch the freed original pointer.
	zero, e := m.View(a, 0, 0)
	if e != nil {
		t.Fatal(e)
	}
	if e = m.Reset(a); e != nil {
		t.Fatal(e)
	}
	if e = m.Reserve(a, 40); e != nil {
		t.Fatal(e)
	}
	if e = m.NormalizeView(zero); e != nil {
		t.Fatal(e)
	}
	equal, e := m.DividesViews(zero, captured)
	if e != nil || !equal {
		t.Fatal(equal, e)
	}
	// Real source debug rejects direct remainder input/output alias; release
	// same-base Set returns before mutation, then computes in that same buffer.
	if e = m.SetInts(a, []unumeral.Int{unumeral.NewInt(2), unumeral.NewInt(1)}); e != nil {
		t.Fatal(e)
	}
	var power uint32
	e = m.Rem(a, b, a, &power)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrUPolynomialAssertion) {
			t.Fatal(e)
		}
	} else {
		if e != nil {
			t.Fatal(e)
		}
		upolyGCDExpect(t, m, a, []int64{1})
	}
	// Public DivRem permits the same q/r owner; source swaps r first then q.
	if e = m.SetInts(a, []unumeral.Int{unumeral.NewInt(2), unumeral.NewInt(1)}); e != nil {
		t.Fatal(e)
	}
	if e = m.DivRem(a, b, out, out, &power); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, out, []int64{1})
	// A stale nonempty raw pointer fails only after source q reset and r setup.
	if e = m.RemViews(captured, mustUPolyView(t, m, b), out, &power); e == nil || errors.Is(e, arena.ErrBudget) {
		t.Fatal(e)
	}
}

func TestUPolynomialModularGCDCRAAndAllPrimeFallback(t *testing.T) {
	m, q, _, limit, _ := upolySourceOwnerServices(t)
	// First image represents x-100000 as x+17309; it is not a divisor of
	// either integer input. The next image's exact CRA reconstruction recovers
	// -100000. This freezes the actual centered reconstruction branch.
	a := upolySourceNew(t, m, []int32{-200000, -99998, 1})
	b := upolySourceNew(t, m, []int32{-300000, -99997, 1})
	out := upolySourceNew(t, m, nil)
	start := limit.Count()
	if e := m.GCD(a, b, out); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, out, []int64{-100000, 1})
	if delta := limit.Count() - start; delta != 10 {
		t.Fatal("two prime images checkpoint count", delta)
	}
	finite, e := m.Modular()
	if e != nil || finite {
		t.Fatal("mode restoration", finite, e)
	}
	// Permanent exact product of all 231 active source primes (39103..41549).
	// With primitive inputs 1+P*x and 2+P*x every modular image loses degree;
	// only the genuine Euclidean fallback contributes two checkpoints.
	var p unumeral.Int
	if e = q.Integers().SetString(&p, "72281563704163390468554344613865981697268096417532898185830731463956991202862156381723822442406430185902272343962106209037022259734054589589771509832616101969073563245939637295520171914563741101372722222388225724053114184535855293430488621916905340634375531252343157962097806024182556922014674849556094504323517239889043714631898242418094530203943384894412218371737626782087162982627243051339990230190141901901314943936056515164760334845853565665191897861856308388331482687299471868936528474006432293908790629327796904367282101244057605259325432547042168783461632004227401144367180753900872998214561525948245758691325277875513151944255585280059492764533971584562086452635471193993199666404816477690902095279935017263322371732225976354939437547459675038205557743700990878333665305741927690617573766495540385679031201157624530285725622749323004927254808222567172354936078488843650973852896345588008946477644700325911826018851139584535871728907061837776621736250767524650331370181989521743149010532598565109856032613723799663632947457509463583151138116994433092877081"); e != nil {
		t.Fatal(e)
	}
	if e = m.SetInts(a, []unumeral.Int{unumeral.NewInt(1), p}); e != nil {
		t.Fatal(e)
	}
	if e = m.SetInts(b, []unumeral.Int{unumeral.NewInt(2), p}); e != nil {
		t.Fatal(e)
	}
	if e = q.Integers().Del(&p); e != nil {
		t.Fatal(e)
	}
	start = limit.Count()
	if e = m.GCD(a, b, out); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, out, []int64{1})
	if delta := limit.Count() - start; delta != 2 {
		t.Fatal("all-primes-bad fallback checkpoints", delta)
	}
	finite, e = m.Modular()
	if e != nil || finite {
		t.Fatal(finite, e)
	}
}

func TestUPolynomialGCDAndDivisionWarmPhysicalPlateau(t *testing.T) {
	m, _, b := upolySourceOwner(t)
	a := upolySourceNew(t, m, []int32{-200000, -99998, 1})
	other := upolySourceNew(t, m, []int32{-300000, -99997, 1})
	g := upolySourceNew(t, m, nil)
	numerator := upolySourceNew(t, m, []int32{1, 1, 1})
	denominator := upolySourceNew(t, m, []int32{1, 1})
	q := upolySourceNew(t, m, nil)
	r := upolySourceNew(t, m, nil)
	var counter uint32
	var err error
	work := func() {
		if err = m.GCD(a, other, g); err == nil {
			err = m.DivRem(numerator, denominator, q, r, &counter)
		}
	}
	for i := 0; i < 20; i++ {
		work()
		if err != nil {
			t.Fatal(err)
		}
	}
	before := m.Stats()
	used := b.Used()
	allocations := testing.AllocsPerRun(100, work)
	if err != nil || (!upolyPhysicalSourceDebug && allocations != 0) {
		t.Fatal(err, allocations)
	}
	after := m.Stats()
	if after.TotalBytes != before.TotalBytes || b.Used() != used || after.LiveBackings != before.LiveBackings {
		t.Fatal(before, after, used, b.Used())
	}
	b.FailAfter(0)
	work()
	b.FailAfter(-1)
	// Source numeric fresh scoped roots may legitimately reserve despite warmed
	// Go storage; the strict physical plateau/allocation test above is separate.
	if err != nil && !errors.Is(err, arena.ErrBudget) && !numericStorageReservationRefusal(err) {
		t.Fatal(err)
	}
}

func TestUPolynomialExtendedGCDNonmonicAndOutputAliasQuirks(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	if e := m.SetZPUint64(7); e != nil {
		t.Fatal(e)
	}
	a := upolySourceNew(t, m, []int32{2, 2})
	b := upolySourceNew(t, m, []int32{2, 1})
	u := upolySourceNew(t, m, nil)
	v := upolySourceNew(t, m, nil)
	d := upolySourceNew(t, m, nil)
	e := m.ExtendedGCD(a, b, u, v, d)
	// Initial monic(A) does not rescale U. Actual release publication gives
	// A*(-1)+B*2 = 2, while D=1. Source debug stops at its real equality check
	// before the last monic/scaling actions, retaining U=1,V=-2,D=-1.
	if polynomialSourceDebug {
		if !errors.Is(e, ErrUPolynomialAssertion) {
			t.Fatal(e)
		}
		upolyGCDExpect(t, m, u, []int64{1})
		upolyGCDExpect(t, m, v, []int64{-2})
		upolyGCDExpect(t, m, d, []int64{-1})
	} else {
		if e != nil {
			t.Fatal(e)
		}
		upolyGCDExpect(t, m, u, []int64{-1})
		upolyGCDExpect(t, m, v, []int64{2})
		upolyGCDExpect(t, m, d, []int64{1})
	}
	if e = m.SetInts(a, []unumeral.Int{unumeral.NewInt(2), unumeral.NewInt(0), unumeral.NewInt(0), unumeral.NewInt(1)}); e != nil {
		t.Fatal(e)
	}
	if e = m.SetInts(b, []unumeral.Int{unumeral.NewInt(6), unumeral.NewInt(0), unumeral.NewInt(6)}); e != nil {
		t.Fatal(e)
	}
	// U and V may denote the same output vector: source has no output/output
	// assertion. V publication overwrites U, then final mul(U,inv),mul(V,inv)
	// scales that same vector twice. Frozen inv(2)=4 gives V_pre*[4]^2.
	if e = m.ExtendedGCD(a, b, u, u, d); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, u, []int64{2, 3, -2})
	upolyGCDExpect(t, m, d, []int64{1})
	if e = m.SetZ(); e != nil {
		t.Fatal(e)
	}
}

func TestUPolynomialPrimitiveRawAliasReserveAndMonicPrefixes(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	polynomialThrowAssertionFixture(t)
	p := upolySourceNew(t, m, []int32{2, 4})
	captured := mustUPolyView(t, m, p)
	if e := m.Shrink(p, 1); e != nil {
		t.Fatal(e)
	}
	var content unumeral.Int
	// gcd sees both original initialized coefficients before reserve. The
	// subsequent source reserve default-constructs the retained second slot
	// before the ascending input reads: exact same-base raw prefix is [1].
	if e := m.PrimitiveAndContentViews(captured, p, &content); e != nil {
		t.Fatal(e)
	}
	n, e := m.IntegerManager().Int64(&content)
	if e != nil || n != 2 {
		t.Fatal(n, e)
	}
	if e = m.IntegerManager().Del(&content); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, p, []int64{1})
	if e = m.SetInts(p, []unumeral.Int{unumeral.NewInt(4), unumeral.NewInt(2)}); e != nil {
		t.Fatal(e)
	}
	var lc, inv unumeral.Int
	// In Z inverse is unsupported after source top swap and inverse copy.
	if e = m.MakeMonicWithLeadingInverse(p, &lc, &inv); e == nil {
		t.Fatal("unsupported integer inverse")
	}
	upolyGCDExpect(t, m, p, []int64{4, 1})
	for _, x := range []*unumeral.Int{&lc, &inv} {
		n, e = m.IntegerManager().Int64(x)
		if e != nil || n != 2 {
			t.Fatal(n, e)
		}
		if e = m.IntegerManager().Del(x); e != nil {
			t.Fatal(e)
		}
	}
	if e = m.SetZPUint64(7); e != nil {
		t.Fatal(e)
	}
	if e = m.SetInts(p, []unumeral.Int{unumeral.NewInt(4), unumeral.NewInt(2)}); e != nil {
		t.Fatal(e)
	}
	if e = m.MakeMonicWithLeading(p, &lc); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, p, []int64{2, 1})
	n, e = m.IntegerManager().Int64(&lc)
	if e != nil || n != 2 {
		t.Fatal(n, e)
	}
	if e = m.IntegerManager().Del(&lc); e != nil {
		t.Fatal(e)
	}
	if e = m.MakeMonicView(mustUPolyView(t, m, p)); e != nil {
		t.Fatal(e)
	}
	if e = m.SetZ(); e != nil {
		t.Fatal(e)
	}
}

func TestUPolynomialGCDChainRawOverloadAndEmptyFieldRoutes(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	a := upolySourceNew(t, m, []int32{4, 6, 2})
	b := upolySourceNew(t, m, []int32{1, 1})
	q := upolySourceNew(t, m, nil)
	r := upolySourceNew(t, m, nil)
	av := mustUPolyView(t, m, a)
	two := unumeral.NewInt(2)
	if e := m.DivScalarView(av, &two); e != nil {
		t.Fatal(e)
	}
	if e := m.NormalizeView(av); e != nil {
		t.Fatal(e)
	}
	if e := m.PrimitiveView(av, q); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, q, []int64{2, 3, 1})
	bv := mustUPolyView(t, m, b)
	var count uint32
	if e := m.DivRemViews(av, bv, q, r, &count); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, q, []int64{2, 1})
	upolyGCDExpect(t, m, r, nil)
	if e := m.DivViews(av, bv, q); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, q, []int64{2, 1})
	if e := m.RemViews(av, bv, r, &count); e != nil {
		t.Fatal(e)
	}
	if e := m.SignedRemViews(av, bv, r); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, r, nil)
	ok, e := m.ExactDivViews(av, bv, q)
	if e != nil || !ok {
		t.Fatal(ok, e)
	}
	upolyGCDExpect(t, m, q, []int64{2, 1})
	ok, e = m.DividesViews(av, bv)
	if e != nil || !ok {
		t.Fatal(ok, e)
	}
	if e = m.GCDViews(av, bv, q); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, q, []int64{1, 1})
	if e = m.EuclidGCDViews(av, bv, q); e != nil {
		t.Fatal(e)
	}
	if e = m.SubresultantGCDViews(av, bv, q); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, q, []int64{1, 1})
	if e = m.SquareFreeView(av, q); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, q, []int64{2, 3, 1})
	ok, e = m.IsSquareFreeView(av)
	if e != nil || !ok {
		t.Fatal(ok, e)
	}
	if e = m.SetZPUint64(7); e != nil {
		t.Fatal(e)
	}
	d := upolySourceNew(t, m, nil)
	if e = m.ExtendedGCDViews(av, bv, q, r, d); e != nil {
		t.Fatal(e)
	}
	upolyGCDExpect(t, m, q, nil)
	upolyGCDExpect(t, m, r, []int64{1})
	upolyGCDExpect(t, m, d, []int64{1, 1})
	var lc, inv unumeral.Int
	if e = m.MakeMonicWithLeadingInverseView(av, &lc, &inv); e != nil {
		t.Fatal(e)
	}
	if e = m.MakeMonicWithLeadingView(av, &lc); e != nil {
		t.Fatal(e)
	}
	if e = m.MakeMonic(a); e != nil {
		t.Fatal(e)
	}
	if e = m.IntegerManager().Del(&inv); e != nil {
		t.Fatal(e)
	}
	if e = m.IntegerManager().Del(&lc); e != nil {
		t.Fatal(e)
	}
	empty := upolySourceNew(t, m, nil)
	other := upolySourceNew(t, m, []int32{-1, -2})
	for _, method := range []int{0, 1, 2} {
		switch method {
		case 0:
			e = m.GCD(empty, other, q)
		case 1:
			e = m.EuclidGCD(other, empty, q)
		case 2:
			e = m.SubresultantGCD(empty, other, q)
		}
		if e != nil {
			t.Fatal(e)
		}
		upolyGCDExpect(t, m, q, []int64{1, 2})
	} // sign flip, never monic in these early returns
	if e = m.SetZ(); e != nil {
		t.Fatal(e)
	}
}

func TestUPolynomialCancelledModularGCDZeroHostAllocation(t *testing.T) {
	m, _, _, limit, _ := upolySourceOwnerServices(t)
	a := upolySourceNew(t, m, []int32{-200000, -99998, 1})
	b := upolySourceNew(t, m, []int32{-300000, -99997, 1})
	g := upolySourceNew(t, m, nil)
	for i := 0; i < 10; i++ {
		if e := m.GCD(a, b, g); e != nil {
			t.Fatal(e)
		}
	}
	limit.Cancel()
	var err error
	allocations := testing.AllocsPerRun(20, func() { err = m.GCD(a, b, g) })
	limit.ResetCancel()
	if !errors.Is(err, ErrUPolynomialCanceled) || allocations != 0 {
		t.Fatal(err, allocations)
	}
	finite, e := m.Modular()
	if e != nil || finite {
		t.Fatal(finite, e)
	}
}

func TestUPolynomialModularGCDUnluckyDegreeAndInitialBadPrime(t *testing.T) {
	for _, f := range []struct {
		a, b   []int32
		want   []int64
		checks uint64
	}{
		// First prime makes A and B equal and gives an unlucky degree-two gcd.
		// Second image loses that extra factor and discards the previous image.
		{[]int32{-2, 1, 1}, []int32{-39105, 39104, 1}, []int64{-1, 1}, 7},
		// First prime kills both leading coefficients. The first surviving image
		// has i=1, so it must initialize via the even(bound=0) branch, then CRA.
		{[]int32{-2, 78205, 39103}, []int32{-3, 117308, 39103}, []int64{-1, 39103}, 10},
	} {
		m, _, _, limit, _ := upolySourceOwnerServices(t)
		a := upolySourceNew(t, m, f.a)
		b := upolySourceNew(t, m, f.b)
		g := upolySourceNew(t, m, nil)
		start := limit.Count()
		if e := m.GCD(a, b, g); e != nil {
			t.Fatal(e)
		}
		upolyGCDExpect(t, m, g, f.want)
		if count := limit.Count() - start; count != f.checks {
			t.Fatal(f, count)
		}
	}
}

func TestUPolynomialContentRawEarlyOneAndLateReadPrefix(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	a := upolySourceNew(t, m, []int32{2, 3})
	out := upolySourceNew(t, m, []int32{99})
	// C++ accepts pointer and unsigned count independently. The first two
	// numbers are coprime; array GCD therefore must not dereference index2.
	raw := mustUPolyView(t, m, a).WithCount(3)
	if e := m.NormalizeView(raw); e != nil {
		t.Fatal("early-one skipped tail", e)
	}
	content := unumeral.NewInt(7)
	e := m.PrimitiveAndContentViews(raw, out, &content)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("later actual source read", e)
	}
	n, e := m.IntegerManager().Int64(&content)
	if e != nil || n != 1 {
		t.Fatal("content published before failed set read", n, e)
	}
	upolyGCDExpect(t, m, out, []int64{2, 3, 0}) // reserve then first two copies, no final trim
	if e = m.IntegerManager().Del(&content); e != nil {
		t.Fatal(e)
	}
}

func TestUPolynomialModularGCDCRALateFailureRootSweep(t *testing.T) {
	// G=x^8-100000; A=G*(x+2), B=G*(x+3). First image is not
	// an integer divisor; second image's real CRT reconstructs G. Native
	// coefficient growth contributes no reservation or failure ordinals.
	m, _, budget, limit, _ := upolySourceOwnerServices(t)
	a := upolySourceNew(t, m, []int32{-200000, -100000, 0, 0, 0, 0, 0, 0, 2, 1})
	b := upolySourceNew(t, m, []int32{-300000, -100000, 0, 0, 0, 0, 0, 0, 3, 1})
	g := upolySourceNew(t, m, nil)
	const allowance = int64(1 << 20)
	budget.FailAfter(allowance)
	start := limit.Count()
	if e := m.GCD(a, b, g); e != nil {
		t.Fatal(e)
	}
	reservations := allowance - testFieldBySourceName(reflect.ValueOf(budget).Elem(), "remaining").Int()
	budget.FailAfter(-1)
	if delta := limit.Count() - start; delta != 10 {
		t.Fatal(delta)
	}
	upolyGCDExpect(t, m, g, []int64{-100000, 0, 0, 0, 0, 0, 0, 0, 1})
	if reservations <= 0 || reservations > 1000 {
		t.Fatal("bounded fixture reservation count", reservations)
	}
	lateFailures := 0
	for ordinal := int64(0); ordinal <= reservations; ordinal++ {
		owner, _, faults, steps, _ := upolySourceOwnerServices(t)
		aa := upolySourceNew(t, owner, []int32{-200000, -100000, 0, 0, 0, 0, 0, 0, 2, 1})
		bb := upolySourceNew(t, owner, []int32{-300000, -100000, 0, 0, 0, 0, 0, 0, 3, 1})
		out := upolySourceNew(t, owner, nil)
		before := owner.Stats()
		external := before.IntegerLiveCells - before.OwnedNumeralCells - before.IntegerCachedCells
		start = steps.Count()
		faults.FailAfter(ordinal)
		err := owner.GCD(aa, bb, out)
		// Exact immediate root audit, before output formatting/copies/Del/Close.
		after := owner.Stats()
		if after.IntegerLiveCells != external+after.OwnedNumeralCells+after.IntegerCachedCells {
			t.Fatal("late CRA unnamed numeral", ordinal, err, before, after)
		}
		upolyHenselAuditPhysical(t, []*UPolynomialManager{owner}, nil)
		finite, e := owner.Modular()
		if e != nil || finite {
			t.Fatal("failure mode restoration", ordinal, finite, e)
		}
		completedChecks := steps.Count() - start
		faults.FailAfter(-1)
		if ordinal < reservations {
			if !numericStorageReservationRefusal(err) {
				t.Fatal(ordinal, reservations, err)
			}
			if completedChecks > 5 {
				lateFailures++
			}
		} else {
			if err != nil || completedChecks != 10 {
				t.Fatal("successful final ordinal", ordinal, err, completedChecks)
			}
			upolyGCDExpect(t, owner, out, []int64{-100000, 0, 0, 0, 0, 0, 0, 0, 1})
		}
	}
	if lateFailures == 0 {
		t.Fatal("sweep did not reach real late reconstruction", reservations, lateFailures)
	}
	t.Log("actual reservation ordinals", reservations+1, "failures after first image", lateFailures)
}
