package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestUPolynomialFactorsLifecycleCopySwapCountersAndMultiply(t *testing.T) {
	m, _, budget := upolySourceOwner(t)
	fs, e := m.NewFactors()
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := fs.Close(); e != nil {
			t.Error(e)
		}
	}()
	if e = m.Close(); !errors.Is(e, ErrUPolynomialLive) {
		t.Fatal("real factorowner prevents coreClose", e)
	}
	p := upolySourceNew(t, m, []int32{1, 1})
	v := mustUPolyView(t, m, p)
	// Both factor owners and multiplicities grow through native slices.
	budget.FailAfter(0)
	e = fs.PushCopy(v, 3)
	budget.FailAfter(-1)
	if e != nil {
		t.Fatal(e)
	}
	if n, e := fs.TotalFactors(); e != nil || n != 3 {
		t.Fatal(n, e)
	}
	if n, e := fs.TotalDegree(); e != nil || n != 3 {
		t.Fatal(n, e)
	}
	if e = fs.SetMultiplicity(0, 2); e != nil {
		t.Fatal(e)
	}
	out := upolySourceNew(t, m, nil)
	if e = fs.Multiply(out); e != nil {
		t.Fatal(e)
	}
	expected := upolySourceNew(t, m, []int32{1, 2, 1})
	if equal, e := m.Equal(out, expected); e != nil || !equal {
		t.Fatal(equal, e)
	}
	if e = fs.Clear(); e != nil {
		t.Fatal(e)
	}
	if n, e := fs.DistinctFactors(); e != nil || n != 0 {
		t.Fatal(n, e)
	}
	if n, e := fs.TotalDegree(); e != nil || n != 0 {
		t.Fatal(n, e)
	}
	if e = fs.PushMove(p, 2); e != nil {
		t.Fatal(e)
	}
	if size, e := m.Size(p); e != nil || size != 0 {
		t.Fatal(size, e)
	}
	// Source computes degree(p) AFTER swapping it with newly empty factor.
	// Mathematical degree2 is deliberately distinguished from actual source0.
	if n, e := fs.TotalDegree(); e != nil || n != 0 {
		t.Fatal("source swapped-empty-input counter defect", n, e)
	}
	if n, e := fs.TotalFactors(); e != nil || n != 2 {
		t.Fatal(n, e)
	}
	if e = fs.Multiply(out); e != nil {
		t.Fatal(e)
	}
	if equal, e := m.Equal(out, expected); e != nil || !equal {
		t.Fatal(equal, e)
	}
}
func TestUPolynomialFactorsLexicalPowerFailureAndImmediateRoots(t *testing.T) {
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		m, _, b := upolySourceOwner(t)
		fs, e := m.NewFactors()
		if e != nil {
			t.Fatal(e)
		}
		p := upolySourceNew(t, m, nil)
		var coefficient util.Int
		if e = m.IntegerManager().SetString(&coefficient, "12345678901234567890"); e != nil {
			t.Fatal(e)
		}
		if e = m.SetInts(p, []util.Int{coefficient, util.NewInt(1)}); e != nil {
			t.Fatal(e)
		}
		if e = m.IntegerManager().Del(&coefficient); e != nil {
			t.Fatal(e)
		}
		out := upolySourceNew(t, m, nil)
		if e = fs.PushCopy(mustUPolyView(t, m, p), 3); e != nil {
			t.Fatal(e)
		}
		b.FailAfter(ordinal)
		err := fs.Multiply(out)
		upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, []*UPolynomialFactors{fs})
		if err != nil && !upolyHenselAllocationRefusal(m.IntegerManager(), err) {
			t.Fatal(err)
		}
		b.FailAfter(-1)
		if err == nil {
			expected := upolySourceNew(t, m, nil)
			inputs := make([]util.Int, 4)
			for i, text := range []string{"1881676372353657772490265749424677022198701224860897069000", "457247362597165102505715599625057156300", "37037036703703703670", "1"} {
				if e = m.IntegerManager().SetString(&inputs[i], text); e != nil {
					t.Fatal(e)
				}
			}
			if e = m.SetInts(expected, inputs); e != nil {
				t.Fatal(e)
			}
			for i := range inputs {
				if e = m.IntegerManager().Del(&inputs[i]); e != nil {
					t.Fatal(e)
				}
			}
			equal, e := m.Equal(out, expected)
			if e != nil || !equal {
				t.Fatal(equal, e)
			}
		}
		if e = fs.Close(); e != nil {
			t.Fatal(e)
		}
		if err == nil {
			break
		}
	}
}

func TestUPolynomialFactorsZeroMultiplicityAndUnsignedCounterQuirks(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	fs, e := m.NewFactors()
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := fs.Close(); e != nil {
			t.Error(e)
		}
	}()
	p := upolySourceNew(t, m, []int32{1, 1})
	out := upolySourceNew(t, m, nil)
	if e = fs.PushMove(p, 2); e != nil {
		t.Fatal(e)
	}
	if e = fs.SetMultiplicity(0, 3); e != nil {
		t.Fatal(e)
	}
	// Starting from the source swap-after-empty-degree defect, setter subtracts
	// actual factor degree*old2 then adds new3: unsigned source counter becomes1.
	if n, e := fs.TotalDegree(); e != nil || n != 1 {
		t.Fatal(n, e)
	}
	if e = fs.SetMultiplicity(0, 0); e != nil {
		t.Fatal(e)
	}
	if n, e := fs.TotalDegree(); e != nil || n != ^uint32(0)-1 {
		t.Fatal("literal unsigned counter prefix", n, e)
	}
	if n, e := fs.TotalFactors(); e != nil || n != 0 {
		t.Fatal(n, e)
	}
	if e = fs.Multiply(out); e != nil {
		t.Fatal(e)
	}
	// Source multiply uses degree>1 else direct-mul, so multiplicity0 still
	// multiplies once; the mathematical empty product would be constant1.
	expected := upolySourceNew(t, m, []int32{1, 1})
	if ok, e := m.Equal(out, expected); e != nil || !ok {
		t.Fatal("source zero multiplicity route", ok, e)
	}
}
func TestUPolynomialFactorsLexicalMultiplyDoesNotAllocateDescriptors(t *testing.T) {
	m, _, _ := upolySourceOwner(t)
	fs, e := m.NewFactors()
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if e := fs.Close(); e != nil {
			t.Error(e)
		}
	}()
	p := upolySourceNew(t, m, []int32{1, 1})
	out := upolySourceNew(t, m, nil)
	if e = fs.PushCopy(mustUPolyView(t, m, p), 3); e != nil {
		t.Fatal(e)
	}
	// Fill actual one-page descriptor capacity. Source lexical power is a
	// vector, not another public owning polynomial or descriptor allocation.
	for i := 0; i < 30; i++ {
		upolySourceNew(t, m, nil)
	}
	before := m.Stats().PageBytes
	if e = fs.Multiply(out); e != nil {
		t.Fatal(e)
	}
	if after := m.Stats().PageBytes; after != before {
		t.Fatal("source lexical temporary fabricated a descriptor", before, after)
	}
}

func TestUPolynomialFactorsColdGuardAndConstructorFailure(t *testing.T) {
	var nilOwner *UPolynomialFactors
	if e := nilOwner.PushMove(UPolynomial{}, 1); !errors.Is(e, ErrUPolynomialOwner) {
		t.Fatal(e)
	}
	if e := nilOwner.SwapFactor(0, UPolynomial{}); !errors.Is(e, ErrUPolynomialOwner) {
		t.Fatal(e)
	}
	var empty UPolynomialFactors
	for _, op := range []func() error{
		empty.Close,
		func() error { return empty.PushMove(UPolynomial{}, 1) },
		func() error { return empty.SwapFactor(0, UPolynomial{}) },
	} {
		if e := op(); !errors.Is(e, ErrUPolynomialOwner) {
			t.Fatal("zero cold owner", e)
		}
	}
	m, _, budget := upolySourceOwner(t)
	before := budget.Used()
	budget.FailAfter(0)
	if f, e := m.NewFactors(); f != nil || !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(f, e)
	}
	if budget.Used() != before {
		t.Fatal("failed cold ctor charge", before, budget.Used())
	}
	budget.FailAfter(-1)
	f, e := m.NewFactors()
	if e != nil {
		t.Fatal(e)
	}
	copy := *f
	owned := f.Stats()
	used := budget.Used()
	if e = copy.Close(); !errors.Is(e, ErrUPolynomialOwner) {
		t.Fatal("copied owning record", e)
	}
	if budget.Used() != used || f.Stats() != owned {
		t.Fatal("guard leaves true owner intact", used, budget.Used(), owned, f.Stats())
	}
	if e = f.Close(); e != nil {
		t.Fatal(e)
	}
	if e = f.Close(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	if budget.Used() != before {
		t.Fatal("real owner refunded once", before, budget.Used())
	}
}
