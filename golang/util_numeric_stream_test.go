package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

func TestNumericStreamFloatProfilesAndSpecialBits(t *testing.T) {
	cases := []struct {
		v    float64
		mode NumericFloatMode
		p    int64
		want string
	}{{0, NumericDefaultFloat, 6, "0"}, {math.Copysign(0, -1), NumericFixedFloat, 2, "-0.00"}, {2.675, NumericFixedFloat, 2, "2.67"}, {2.675, NumericScientificFloat, 2, "2.67e+00"}, {2.5, NumericFixedFloat, 0, "2"}, {3.5, NumericFixedFloat, 0, "4"}, {.0001, NumericDefaultFloat, 6, "0.0001"}, {.00001, NumericDefaultFloat, 6, "1e-05"}, {1e6, NumericDefaultFloat, 6, "1e+06"}, {math.SmallestNonzeroFloat64, NumericDefaultFloat, 6, "4.94066e-324"}, {math.SmallestNonzeroFloat64, NumericScientificFloat, 2, "4.94e-324"}, {math.Float64frombits(0xfff8000000000001), NumericFixedFloat, 2, "-nan"}, {math.Inf(1), NumericDefaultFloat, 6, "inf"}, {math.Inf(-1), NumericScientificFloat, 2, "-inf"}}
	s, e := NewNumericTextStream(NumericTextStreamConfig{})
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	for _, c := range cases {
		s.ResetText()
		s.SetFloatMode(c.mode)
		s.SetPrecision(c.p)
		if e = s.AppendFloat64(c.v); e != nil {
			t.Fatal(e)
		}
		got, e := s.String()
		if e != nil || got != c.want {
			t.Fatalf("%g %q/%q %v", c.v, got, c.want, e)
		}
	}
}
func TestNumericStreamShowpointSignUpperAndPrecisionABI(t *testing.T) {
	s, _ := NewNumericTextStream(NumericTextStreamConfig{})
	defer s.Close()
	s.SetShowPoint(true)
	cases := []struct {
		mode NumericFloatMode
		p    int64
		v    float64
		want string
	}{{NumericDefaultFloat, 6, 1.25, "1.25000"}, {NumericDefaultFloat, 0, 1, "1."}, {NumericFixedFloat, 0, 2.5, "2."}, {NumericScientificFloat, 0, 2.5, "2.e+00"}, {NumericDefaultFloat, 6, .0001, "0.000100000"}, {NumericDefaultFloat, 6, 1e6, "1.00000e+06"}, {NumericFixedFloat, (1 << 32) + 2, 1.25, "1.25"}, {NumericScientificFloat, (1 << 32) + 2, 1.25, "1.25e+00"}, {NumericDefaultFloat, (1 << 32) + 2, 1.25, "1.2"}}
	for _, c := range cases {
		s.ResetText()
		s.SetFloatMode(c.mode)
		s.SetPrecision(c.p)
		s.AppendFloat64(c.v)
		got, _ := s.String()
		if got != c.want || s.State().Precision != c.p {
			t.Fatal(got, c.want, s.State())
		}
	}
	s.SetShowPoint(false)
	s.SetPrecision(-1)
	s.SetFloatMode(NumericFixedFloat)
	s.ResetText()
	s.AppendFloat64(1.25)
	got, _ := s.String()
	if got != "1.250000" {
		t.Fatal(got)
	}
	s.SetPrecision(1 << 31)
	s.ResetText()
	s.AppendFloat64(1.25)
	got, _ = s.String()
	if got != "1.250000" {
		t.Fatal("omitted fixed precision", got)
	}
	s.SetUppercase(true)
	s.SetShowPos(true)
	for _, c := range []struct {
		v    float64
		want string
	}{{math.Inf(1), "+INF"}, {math.Float64frombits(0xfff8000000000001), "-NAN"}, {math.NaN(), "+NAN"}, {math.Copysign(0, -1), "-0.000000"}} {
		s.ResetText()
		s.AppendFloat64(c.v)
		got, _ = s.String()
		if got != c.want {
			t.Fatal(got, c.want)
		}
	}
}
func TestNumericStreamPersistentWidthAlignmentAndIntegerDomains(t *testing.T) {
	s, _ := NewNumericTextStream(NumericTextStreamConfig{})
	defer s.Close()
	s.SetFloatMode(NumericFixedFloat)
	s.SetPrecision(2)
	s.SetShowPos(true)
	s.SetFill('_')
	s.SetWidth(9)
	s.SetAlignment(NumericInternal)
	s.AppendFloat64(1.25)
	got, _ := s.String()
	if got != "+____1.25" || s.State().Width != 0 || s.State().Precision != 2 {
		t.Fatal(got, s.State())
	}
	s.ResetText()
	s.SetShowPos(false)
	s.SetWidth(8)
	s.AppendCString(" ")
	s.AppendFloat64(1.25)
	got, _ = s.String()
	if got != "_______ 1.25" {
		t.Fatal(got)
	}
	for _, align := range []NumericAlignment{NumericLeft, NumericRight, NumericInternal} {
		s.ResetText()
		s.SetAlignment(align)
		s.SetWidth(5)
		s.AppendInt32(-12)
		got, _ = s.String()
		want := map[NumericAlignment]string{NumericLeft: "-12__", NumericRight: "__-12", NumericInternal: "-__12"}[align]
		if got != want {
			t.Fatal(got, want)
		}
	}
	s.SetWidth(0)
	s.SetShowBase(true)
	s.SetIntegerBase(16)
	s.SetUppercase(true)
	s.ResetText()
	s.AppendInt32(-1)
	s.AppendByte(' ')
	s.AppendInt64(-1)
	s.AppendByte(' ')
	s.AppendUint32(math.MaxUint32)
	got, _ = s.String()
	if got != "0XFFFFFFFF 0XFFFFFFFFFFFFFFFF 0XFFFFFFFF" {
		t.Fatal(got)
	}
	s.SetIntegerBase(8)
	s.SetUppercase(false)
	s.ResetText()
	s.AppendUint64(8)
	s.AppendByte(' ')
	s.AppendUint32(0)
	got, _ = s.String()
	if got != "010 0" {
		t.Fatal(got)
	}
	s.SetIntegerBase(10)
	s.SetShowPos(true)
	s.ResetText()
	s.AppendUint32(12)
	s.AppendByte(' ')
	s.AppendInt32(12)
	s.AppendByte(' ')
	s.AppendInt64(math.MinInt64)
	got, _ = s.String()
	if got != "12 +12 -9223372036854775808" {
		t.Fatal(got)
	}
	s.ResetText()
	s.SetWidth(-2)
	s.AppendCString("a\x00b")
	s.AppendByte(0)
	if s.Stats().Position != 2 || s.State().Width != 0 {
		t.Fatal(s.Stats())
	}
}
func TestNumericStreamFullByteOutputStringAndWidthConsumption(t *testing.T) {
	s, _ := NewNumericTextStream(NumericTextStreamConfig{})
	defer s.Close()
	s.AppendCString("a\x00ignored")
	s.AppendByte(0)
	s.AppendByte('b')
	got, e := s.String()
	if e != nil || got != "a\x00b" || s.Stats().Position != 3 {
		t.Fatalf("full stream bytes %q %v", got, e)
	}
	s.ResetText()
	s.AppendCString(strings.Repeat("x", 64))
	capBefore := s.Stats().OutputCapacity
	s.String()
	if s.Stats().OutputCapacity != capBefore {
		t.Fatal("str() must not append source CString terminator")
	}
}

func TestNumericStreamBudgetFailureStagesPrefixesAndPeak(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	s, _ := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
	base := b.Used()
	s.SetFloatMode(NumericFixedFloat)
	s.SetPrecision(2)
	s.SetWidth(8)
	b.FailAfter(0)
	e := s.AppendFloat64(1.25)
	if !errors.Is(e, arena.ErrInjectedFailure) || s.State().Width != 8 || s.Stats().Position != 0 || b.Used() != base {
		t.Fatal(e, s.State(), s.Stats())
	}
	b.FailAfter(-1)
	s.AppendFloat64(1.25)
	s.ResetText()
	s.AppendCString(strings.Repeat("x", 63))
	s.SetShowPos(true)
	s.SetWidth(0)
	b.FailAfter(0)
	e = s.AppendFloat64(1.25)
	if !errors.Is(e, arena.ErrInjectedFailure) || s.Stats().Position != 64 || s.State().Width != 0 {
		t.Fatal(e, s.Stats())
	}
	s.Close()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
	for fail := int64(0); fail < 4; fail++ {
		budget := arena.NewBudget(^uint64(0))
		budget.FailAfter(fail)
		owner, e := NewNumericTextStream(NumericTextStreamConfig{Budget: budget})
		if e == nil {
			owner.Close()
		}
		if budget.Used() != 0 {
			t.Fatal("ctor unwind")
		}
	}
	// Existing scratch remains charged while the entire larger scratch is reserved.
	b = arena.NewBudget(^uint64(0))
	s, _ = NewNumericTextStream(NumericTextStreamConfig{Budget: b})
	s.SetFloatMode(NumericScientificFloat)
	s.SetPrecision(2)
	s.AppendFloat64(1)
	old := s.Stats().ScratchBytes
	before := b.Used()
	b.FailAfter(0)
	s.SetPrecision(100)
	e = s.AppendFloat64(1)
	if !errors.Is(e, arena.ErrInjectedFailure) || s.Stats().ScratchBytes != old || b.Used() != before {
		t.Fatal(e)
	}
	b.FailAfter(-1)
	s.Close()
}
func TestNumericStreamSlowPrecisionWarmAllocationsAndGuards(t *testing.T) {
	s, _ := NewNumericTextStream(NumericTextStreamConfig{})
	defer s.Close()
	s.SetFloatMode(NumericFixedFloat)
	s.SetPrecision(1000)
	s.AppendFloat64(math.MaxFloat64)
	s.String()
	if n := testing.AllocsPerRun(30, func() {
		s.ResetText()
		if e := s.AppendFloat64(math.MaxFloat64); e != nil {
			panic(e)
		}
	}); n != 0 {
		t.Fatal(n)
	}
	if reflect.TypeOf(NumericStreamState{}).Size() == 0 {
		t.Fatal("state")
	}
	if e := arena.ValidateRecordType(reflect.TypeOf(NumericStreamState{})); e != nil {
		t.Fatal(e)
	}
	if e := s.SetIntegerBase(2); !errors.Is(e, ErrNumericStreamConfig) {
		t.Fatal(e)
	}
	if e := s.SetFloatMode(99); !errors.Is(e, ErrNumericStreamConfig) {
		t.Fatal(e)
	}
	if e := s.SetAlignment(99); !errors.Is(e, ErrNumericStreamConfig) {
		t.Fatal(e)
	}
	var nilOwner *NumericTextStream
	if e := nilOwner.AppendFloat64(0); !errors.Is(e, ErrNumericStreamOwner) {
		t.Fatal(e)
	}
	fake := &NumericTextStream{}
	fake.self = s
	if e := fake.AppendByte('x'); !errors.Is(e, ErrNumericStreamOwner) {
		t.Fatal(e)
	}
	text, _ := s.String()
	s.Close()
	if len(text) != 1310 || text[len(text)-1] != '0' {
		t.Fatal("output", len(text))
	}
	if e := s.SetPrecision(2); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}

// Independent exact binary rational rounding; no strconv float renderer oracle.
func numericFixedOracle(v float64, p int) string {
	negative := math.Signbit(v)
	r := new(big.Rat).SetFloat64(math.Abs(v))
	power := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(p)), nil)
	scaled := new(big.Int).Mul(r.Num(), power)
	q, rem := new(big.Int), new(big.Int)
	q.QuoRem(scaled, r.Denom(), rem)
	twice := new(big.Int).Lsh(rem, 1)
	cmp := twice.Cmp(r.Denom())
	if cmp > 0 || cmp == 0 && q.Bit(0) != 0 {
		q.Add(q, big.NewInt(1))
	}
	digits := q.String()
	if p > 0 {
		if len(digits) <= p {
			digits = strings.Repeat("0", p+1-len(digits)) + digits
		}
		digits = digits[:len(digits)-p] + "." + digits[len(digits)-p:]
	}
	if negative {
		return "-" + digits
	}
	return digits
}
func TestNumericStreamIndependentExactBinaryRounding(t *testing.T) {
	s, _ := NewNumericTextStream(NumericTextStreamConfig{})
	defer s.Close()
	s.SetFloatMode(NumericFixedFloat)
	state := uint64(123)
	for i := 0; i < 180; i++ {
		state = state*6364136223846793005 + 1
		bits := state
		if bits&0x7ff0000000000000 == 0x7ff0000000000000 {
			bits ^= 1 << 52
		}
		v := math.Float64frombits(bits)
		p := i % 7
		s.ResetText()
		s.SetPrecision(int64(p))
		if e := s.AppendFloat64(v); e != nil {
			t.Fatal(e)
		}
		got, _ := s.String()
		want := numericFixedOracle(v, p)
		if got != want {
			t.Fatalf("bits%x p%d %q/%q", bits, p, got, want)
		}
		s.SetFloatMode(NumericScientificFloat)
		s.ResetText()
		s.AppendFloat64(v)
		got, _ = s.String()
		want = numericScientificOracle(v, p)
		if got != want {
			t.Fatalf("science bits%x p%d %q/%q", bits, p, got, want)
		}
		s.SetFloatMode(NumericFixedFloat)
	}
	for _, v := range []float64{math.MaxFloat64, -math.MaxFloat64, math.SmallestNonzeroFloat64, -math.SmallestNonzeroFloat64, math.Nextafter(2.5, 0), 2.5, math.Nextafter(2.5, 3)} {
		for p := 0; p < 4; p++ {
			s.ResetText()
			s.SetPrecision(int64(p))
			s.AppendFloat64(v)
			got, _ := s.String()
			if got != numericFixedOracle(v, p) {
				t.Fatalf("endpoint %g %d %q", v, p, got)
			}
		}
	}
}
func TestNumericStreamExactOwnerScratchOverlapAndHugeWidthPrefix(t *testing.T) {
	unlimited := arena.NewBudget(^uint64(0))
	s, _ := NewNumericTextStream(NumericTextStreamConfig{Budget: unlimited})
	base := uint64(reflect.TypeOf(NumericTextStream{}).Size()) + uint64(reflect.TypeOf(StringBuffer64{}).Size())
	if unlimited.Used() != base {
		t.Fatal(unlimited.Used(), base)
	}
	s.Close()
	// Small scientific field uses10 scratchbytes. Raising precision8 requires16
	// while the old10 remain charged; final storage16 is less than peak26.
	for _, extra := range []uint64{25, 26} {
		b := arena.NewBudget(base + extra)
		s, _ := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
		s.SetFloatMode(NumericScientificFloat)
		s.SetPrecision(2)
		s.AppendFloat64(1)
		s.ResetText()
		s.SetPrecision(8)
		e := s.AppendFloat64(1)
		if extra == 25 {
			if !errors.Is(e, arena.ErrBudget) || b.Used() != base+10 || s.Stats().Position != 0 {
				t.Fatal(e, b.Used(), s.Stats())
			}
		} else if e != nil || b.Used() != base+16 {
			t.Fatal(e, b.Used())
		}
		s.Close()
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
	b := arena.NewBudget(base)
	s, _ = NewNumericTextStream(NumericTextStreamConfig{Budget: b})
	s.AppendByte('p')
	s.SetWidth(math.MaxInt64)
	e := s.AppendUint32(1)
	if !errors.Is(e, arena.ErrBudget) || s.Stats().Position != 64 || s.State().Width != 0 {
		t.Fatal(e, s.Stats(), s.State())
	}
	s.Close()
	if _, e := NewNumericTextStream(NumericTextStreamConfig{Profile: 99}); !errors.Is(e, ErrNumericStreamConfig) {
		t.Fatal(e)
	}
}

func numericScientificOracle(v float64, p int) string {
	negative := math.Signbit(v)
	r := new(big.Rat).SetFloat64(math.Abs(v))
	exp := 0
	if r.Sign() != 0 {
		exp = len(r.Num().String()) - len(r.Denom().String())
		power := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(absNumericTest(exp))), nil)
		left := new(big.Int).Set(r.Num())
		right := new(big.Int).Set(r.Denom())
		if exp >= 0 {
			right.Mul(right, power)
		} else {
			left.Mul(left, power)
		}
		if left.Cmp(right) < 0 {
			exp--
		}
	}
	shift := p - exp
	power := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(absNumericTest(shift))), nil)
	num := new(big.Int).Set(r.Num())
	den := new(big.Int).Set(r.Denom())
	if shift >= 0 {
		num.Mul(num, power)
	} else {
		den.Mul(den, power)
	}
	q, rem := new(big.Int), new(big.Int)
	q.QuoRem(num, den, rem)
	cmp := new(big.Int).Lsh(rem, 1).Cmp(den)
	if cmp > 0 || cmp == 0 && q.Bit(0) != 0 {
		q.Add(q, big.NewInt(1))
	}
	digits := q.String()
	if len(digits) > p+1 {
		q.Quo(q, big.NewInt(10))
		digits = q.String()
		exp++
	}
	if len(digits) < p+1 {
		digits = strings.Repeat("0", p+1-len(digits)) + digits
	}
	mant := digits[:1]
	if p > 0 {
		mant += "." + digits[1:]
	}
	sign := "+"
	if exp < 0 {
		sign = "-"
	}
	e := big.NewInt(int64(absNumericTest(exp))).String()
	if len(e) < 2 {
		e = "0" + e
	}
	if negative {
		mant = "-" + mant
	}
	return mant + "e" + sign + e
}
func absNumericTest(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
