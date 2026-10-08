// Source: src/test/ext_numeral.cpp:24-397; src/util/ext_numeral.h:25-345.
package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
)

type extUpstreamCase struct {
	Operation string            `json:"operation"`
	Line      int               `json:"line"`
	Args      []json.RawMessage `json:"arguments"`
}

func extArg(t *testing.T, v json.RawMessage) int64 {
	t.Helper()
	var n int64
	serviceMust(t, json.Unmarshal(v, &n))
	return n
}
func extBinary(m ExtNumeralManager, op string, ak, bk ExtNumeralKind, ck *ExtNumeralKind) error {
	switch op {
	case "add":
		return ExtAdd(m, ak, bk, ck)
	case "sub":
		return ExtSub(m, ak, bk, ck)
	case "mul":
		return ExtMul(m, ak, bk, ck)
	case "div":
		return ExtDiv(m, ak, bk, ck)
	}
	panic(op)
}
func extRelation(m ExtNumeralManager, op string, ak, bk ExtNumeralKind) (bool, error) {
	switch op {
	case "eq":
		return ExtEq(m, ak, bk)
	case "neq":
		return ExtNeq(m, ak, bk)
	case "lt":
		return ExtLt(m, ak, bk)
	case "gt":
		return ExtGt(m, ak, bk)
	case "le":
		return ExtLe(m, ak, bk)
	case "ge":
		return ExtGe(m, ak, bk)
	}
	panic(op)
}
func TestExtNumeralUpstreamArithmeticAndRelations(t *testing.T) {
	data, e := os.ReadFile("test_vectors/ext_numeral_upstream.json")
	serviceMust(t, e)
	var vector struct {
		Source string            `json:"source"`
		Cases  []extUpstreamCase `json:"cases"`
	}
	serviceMust(t, json.Unmarshal(data, &vector))
	if vector.Source != "src/test/ext_numeral.cpp" || len(vector.Cases) != 212 {
		t.Fatal("upstream inventory")
	}
	for _, row := range vector.Cases {
		t.Run(fmt.Sprintf("ext_numeral.cpp:%d/%s", row.Line, row.Operation), func(t *testing.T) {
			p, _ := mpqInfFixture(t, false)
			a, b, c := NewRational(), NewRational(), NewRational()
			serviceMust(t, p.m.SetInt64(&a, extArg(t, row.Args[0])))
			ak := ExtNumeralKind(extArg(t, row.Args[1]))
			bind := ExtRationalBinding{p.m, [3]*Rational{&a, &b, &c}}
			if len(row.Args) == 4 {
				ck := ak
				if row.Operation == "neg" {
					serviceMust(t, ExtNeg(&bind, &ck))
				} else {
					serviceMust(t, ExtInv(&bind, &ck))
				}
				if ck != ExtNumeralKind(extArg(t, row.Args[3])) {
					t.Fatal("unary kind")
				}
				if ck == ExtFinite && rationalString(t, p.m, &a) != fmt.Sprint(extArg(t, row.Args[2])) {
					t.Fatal("unary value")
				}
			} else {
				serviceMust(t, p.m.SetInt64(&b, extArg(t, row.Args[2])))
				bk := ExtNumeralKind(extArg(t, row.Args[3]))
				repetitions := 1
				if row.Operation == "add" || row.Operation == "mul" || row.Operation == "eq" || row.Operation == "neq" {
					repetitions = 2
				}
				for repetition := 0; repetition < repetitions; repetition++ {
					if repetition == 1 {
						bind.Operands[ExtA], bind.Operands[ExtB] = bind.Operands[ExtB], bind.Operands[ExtA]
						ak, bk = bk, ak
					}
					if len(row.Args) == 6 {
						ck := ExtFinite
						serviceMust(t, extBinary(&bind, row.Operation, ak, bk, &ck))
						if ck != ExtNumeralKind(extArg(t, row.Args[5])) {
							t.Fatal("binary kind")
						}
						if ck == ExtFinite && rationalString(t, p.m, &c) != fmt.Sprint(extArg(t, row.Args[4])) {
							t.Fatal("binary value")
						}
					} else {
						var want bool
						serviceMust(t, json.Unmarshal(row.Args[4], &want))
						got, e := extRelation(&bind, row.Operation, ak, bk)
						serviceMust(t, e)
						if got != want {
							t.Fatal("relation", got, want)
						}
					}
				}
			}
			serviceMust(t, p.m.Del(&a))
			serviceMust(t, p.m.Del(&b))
			serviceMust(t, p.m.Del(&c))
		})
	}
}

func TestExtNumeralInfinityReadErasureAndPower(t *testing.T) {
	for _, k := range []ExtNumeralKind{ExtMinusInfinity, ExtPlusInfinity} {
		zero, e := ExtIsZero(nil, k)
		serviceMust(t, e)
		if zero {
			t.Fatal("infinite zero")
		}
		pos, e := ExtIsPos(nil, k)
		serviceMust(t, e)
		neg, e := ExtIsNeg(nil, k)
		serviceMust(t, e)
		if pos != (k == ExtPlusInfinity) || neg != (k == ExtMinusInfinity) || !ExtIsInfinite(k) {
			t.Fatal("infinite signs")
		}
		for _, n := range []uint32{0, 1, 2, 3, 1 << 31, ^uint32(0)} {
			kind := k
			serviceMust(t, ExtPower(nil, &kind, n))
			want := k
			if k == ExtMinusInfinity && n%2 == 0 {
				want = ExtPlusInfinity
			}
			if kind != want {
				t.Fatal("power", k, n, kind)
			}
		}
		kind := k
		serviceMust(t, ExtNeg(nil, &kind))
		if kind == k {
			t.Fatal("infinite neg")
		}
	}
	for _, a := range []ExtNumeralKind{ExtMinusInfinity, ExtFinite, ExtPlusInfinity} {
		for _, b := range []ExtNumeralKind{ExtMinusInfinity, ExtFinite, ExtPlusInfinity} {
			if a == ExtFinite && b == ExtFinite {
				continue
			}
			eq, e := ExtEq(nil, a, b)
			serviceMust(t, e)
			lt, e := ExtLt(nil, a, b)
			serviceMust(t, e)
			if eq != (a == b) || lt != (a < b) {
				t.Fatal("kind comparison", a, b)
			}
		}
	}
	if !ExtIsInfinite(99) {
		t.Fatal("source invalid kind classification")
	}
	kind := ExtNumeralKind(99)
	serviceMust(t, ExtNeg(nil, &kind))
	serviceMust(t, ExtPower(nil, &kind, 2))
	if kind != 99 {
		t.Fatal("invalid switch wrote kind")
	}
	serviceMust(t, ExtDisplay(nil, nil, 99))
}

func TestExtNumeralUpstreamPredicatesResetAndDisplay(t *testing.T) {
	p, _ := mpqInfFixture(t, false)
	a, b := NewRational(), NewRational()
	binding := ExtRationalBinding{p.m, [3]*Rational{&a, &b, nil}}
	for _, n := range []int64{-5, 0, 10} {
		serviceMust(t, p.m.SetInt64(&a, n))
		for _, kind := range []ExtNumeralKind{ExtMinusInfinity, ExtFinite, ExtPlusInfinity} {
			zero, e := ExtIsZero(&binding, kind)
			serviceMust(t, e)
			pos, e := ExtIsPos(&binding, kind)
			serviceMust(t, e)
			neg, e := ExtIsNeg(&binding, kind)
			serviceMust(t, e)
			if zero != (kind == ExtFinite && n == 0) || pos != (kind == ExtPlusInfinity || kind == ExtFinite && n > 0) || neg != (kind == ExtMinusInfinity || kind == ExtFinite && n < 0) {
				t.Fatal("upstream predicates", n, kind)
			}
		}
	}
	kind := ExtMinusInfinity
	serviceMust(t, ExtReset(&binding, &kind))
	if kind != ExtFinite || rationalString(t, p.m, &a) != "0" {
		t.Fatal("reset")
	}
	serviceMust(t, p.m.SetInt64(&b, 17))
	serviceMust(t, ExtSet(&binding, &kind, ExtPlusInfinity))
	if kind != ExtPlusInfinity || rationalString(t, p.m, &a) != "17" {
		t.Fatal("set copies payload even infinity")
	}
	for _, n := range []uint32{0, 1, 2, 7} {
		serviceMust(t, p.m.SetInt64(&a, -2))
		kind = ExtFinite
		serviceMust(t, ExtPower(&binding, &kind, n))
		want := int64(1)
		for j := uint32(0); j < n; j++ {
			want *= -2
		}
		if kind != ExtFinite || rationalString(t, p.m, &a) != fmt.Sprint(want) {
			t.Fatal("finite power")
		}
	}
	for _, pretty := range []bool{false, true} {
		for _, kind := range []ExtNumeralKind{ExtMinusInfinity, ExtFinite, ExtPlusInfinity} {
			out, e := NewNumericTextStream(NumericTextStreamConfig{})
			serviceMust(t, e)
			serviceMust(t, p.m.SetInt64(&a, -10))
			if pretty {
				serviceMust(t, ExtDisplayPP(out, &binding, kind))
			} else {
				serviceMust(t, ExtDisplay(out, &binding, kind))
			}
			got, e := out.String()
			serviceMust(t, e)
			want := "-10"
			if kind == ExtMinusInfinity {
				want = "-oo"
				if pretty {
					want = "-&infin;"
				}
			}
			if kind == ExtPlusInfinity {
				want = "+oo"
				if pretty {
					want = "+&infin;"
				}
			}
			if got != want {
				t.Fatal("display", pretty, kind, got)
			}
			serviceMust(t, out.Close())
		}
	}
	serviceMust(t, p.m.Del(&a))
	serviceMust(t, p.m.Del(&b))
}

// A controlled manager stop checks branch publication independently of the
// numeric allocator; actual numerical allocation audits are a separate gate.
type extStoppedBinding struct {
	ExtRationalBinding
	stop  error
	calls int
}

func (b *extStoppedBinding) Reset(s ExtNumeralSlot) error  { b.calls++; return b.stop }
func (b *extStoppedBinding) Set(a, c ExtNumeralSlot) error { b.calls++; return b.stop }
func (b *extStoppedBinding) Add() error                    { b.calls++; return b.stop }
func (b *extStoppedBinding) Sub() error                    { b.calls++; return b.stop }
func (b *extStoppedBinding) Mul() error                    { b.calls++; return b.stop }
func (b *extStoppedBinding) Div() error                    { b.calls++; return b.stop }

func TestExtNumeralSourceKindPublicationOnManagerStop(t *testing.T) {
	p, _ := mpqInfFixture(t, false)
	a, b, c := NewRational(), NewRational(), NewRational()
	serviceMust(t, p.m.SetInt64(&a, 3))
	serviceMust(t, p.m.SetInt64(&b, -4))
	stop := fmt.Errorf("controlled numerical stop")
	binding := extStoppedBinding{ExtRationalBinding{p.m, [3]*Rational{&a, &b, &c}}, stop, 0}
	for _, op := range []string{"add", "sub", "mul", "div"} {
		for _, ak := range []ExtNumeralKind{ExtFinite, ExtPlusInfinity} {
			bk := ExtFinite
			kind := ExtNumeralKind(99)
			binding.calls = 0
			e := extBinary(&binding, op, ak, bk, &kind)
			if e != stop || binding.calls != 1 {
				t.Fatal(op, ak, e, binding.calls)
			}
			want := ExtNumeralKind(99)
			if op == "mul" || op == "div" {
				want = ExtFinite
				if ak == ExtPlusInfinity {
					want = ExtMinusInfinity
				}
			}
			if kind != want {
				t.Fatal("publication order", op, ak, kind, want)
			}
		}
	}
	kind := ExtPlusInfinity
	if e := ExtReset(&binding, &kind); e != stop || kind != ExtPlusInfinity {
		t.Fatal("reset publication")
	}
	if e := ExtSet(&binding, &kind, ExtFinite); e != stop || kind != ExtPlusInfinity {
		t.Fatal("set publication")
	}
	if e := ExtInv(&binding, &kind); e != stop || kind != ExtFinite {
		t.Fatal("inv publishes finite BEFORE reset")
	}
	serviceMust(t, p.m.Del(&a))
	serviceMust(t, p.m.Del(&b))
	serviceMust(t, p.m.Del(&c))
}

func extIndependentBinary(op string, a, b *big.Rat, ak, bk ExtNumeralKind) (*big.Rat, ExtNumeralKind, bool) {
	zero := new(big.Rat)
	if op == "sub" {
		b = new(big.Rat).Neg(b)
		if bk != ExtFinite {
			bk = ExtNumeralKind(2 - int(bk))
		}
		op = "add"
	}
	if op == "add" {
		if ak != ExtFinite && bk != ExtFinite && ak != bk {
			return nil, 0, false
		}
		if ak != ExtFinite {
			return zero, ak, true
		}
		if bk != ExtFinite {
			return zero, bk, true
		}
		return new(big.Rat).Add(a, b), ExtFinite, true
	}
	if op == "div" && (bk == ExtFinite && b.Sign() == 0 || ak != ExtFinite && bk != ExtFinite) {
		return nil, 0, false
	}
	if ak == ExtFinite && a.Sign() == 0 || op == "mul" && bk == ExtFinite && b.Sign() == 0 {
		return zero, ExtFinite, true
	}
	if op == "div" && bk != ExtFinite {
		return zero, ExtFinite, true
	}
	if ak != ExtFinite || bk != ExtFinite {
		sa, sb := a.Sign(), b.Sign()
		if ak != ExtFinite {
			sa = int(ak) - 1
		}
		if bk != ExtFinite {
			sb = int(bk) - 1
		}
		kind := ExtMinusInfinity
		if sa*sb > 0 {
			kind = ExtPlusInfinity
		}
		return zero, kind, true
	}
	if op == "mul" {
		return new(big.Rat).Mul(a, b), ExtFinite, true
	}
	return new(big.Rat).Quo(a, b), ExtFinite, true
}

func TestExtNumeralIndependentRationalArithmeticAndAliases(t *testing.T) {
	for _, sync := range []bool{false, true} {
		p, _ := mpqInfFixture(t, sync)
		for _, first := range []string{"-2/3", "0", "3/2", "123456789012345678901234567890123456789/23"} {
			for _, second := range []string{"-5/7", "0", "7/3", "-987654321098765432109876543210987654321/19"} {
				left, _ := new(big.Rat).SetString(first)
				right, _ := new(big.Rat).SetString(second)
				for ak := ExtMinusInfinity; ak <= ExtPlusInfinity; ak++ {
					for bk := ExtMinusInfinity; bk <= ExtPlusInfinity; bk++ {
						for _, op := range []string{"add", "sub", "mul", "div"} {
							want, wk, defined := extIndependentBinary(op, left, right, ak, bk)
							if !defined {
								continue
							}
							for _, alias := range []ExtNumeralSlot{ExtC, ExtA, ExtB} {
								a, b, c := NewRational(), NewRational(), NewRational()
								serviceMust(t, p.m.SetString(&a, first))
								serviceMust(t, p.m.SetString(&b, second))
								serviceMust(t, p.m.SetInt64(&c, 19))
								bind := ExtRationalBinding{p.m, [3]*Rational{&a, &b, &c}}
								if alias != ExtC {
									bind.Operands[ExtC] = bind.Operands[alias]
								}
								ck := ExtNumeralKind(99)
								serviceMust(t, extBinary(&bind, op, ak, bk, &ck))
								if ck != wk || rationalString(t, p.m, bind.Operands[ExtC]) != want.RatString() {
									t.Fatal(sync, first, second, ak, bk, op, alias, ck, wk)
								}
								serviceMust(t, p.m.Del(&a))
								serviceMust(t, p.m.Del(&b))
								serviceMust(t, p.m.Del(&c))
							}
						}
					}
				}
			}
		}
	}
}

func TestExtNumeralEveryColdNumericalAndDisplayReservation(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for _, op := range []string{"add", "sub", "mul", "div", "set", "inv", "power", "lt", "display"} {
			completed := false
			for ordinal := int64(0); ordinal < 150; ordinal++ {
				p, budget := mpqInfFixture(t, sync)
				x := mpqInfValue(t, p, "123456789012345678901234567890123456789/23", "987654321098765432109876543210987654321/19")
				output := NewMPQInf()
				bind := ExtRationalBinding{p.m, [3]*Rational{&x.First, &x.Second, &output.First}}
				ak := ExtFinite
				ck := ExtPlusInfinity
				var out *NumericTextStream
				if op == "display" {
					var e error
					out, e = NewNumericTextStream(NumericTextStreamConfig{Budget: budget})
					serviceMust(t, e)
					serviceMust(t, out.SetWidth(150))
				}
				budget.FailAfter(ordinal)
				var e error
				switch op {
				case "set":
					e = ExtSet(&bind, &ak, ExtPlusInfinity)
				case "inv":
					e = ExtInv(&bind, &ak)
				case "power":
					e = ExtPower(&bind, &ak, 3)
				case "lt":
					_, e = ExtLt(&bind, ExtFinite, ExtFinite)
				case "display":
					e = ExtDisplay(out, &bind, ExtFinite)
				default:
					e = extBinary(&bind, op, ExtFinite, ExtFinite, &ck)
				}
				qopsAuditLiveOwnership(t, p.m, []*Int{&x.First.Num, &x.First.Den, &x.Second.Num, &x.Second.Den, &output.First.Num, &output.First.Den, &output.Second.Num, &output.Second.Den})
				actual := p.header + p.m.HeaderBytes + integerActualOwnedBytes(p.m.Z)
				if out != nil {
					st := out.Stats()
					actual += st.OwnerBytes + st.ScratchBytes + st.OutputOwnerBytes + st.OutputOverflowBytes
				}
				if budget.Used() != actual {
					t.Fatal("actual immediate graph", sync, op, ordinal, budget.Used(), actual)
				}
				if e != nil {
					if !numericPhysicalReservationRefusal(e) {
						t.Fatal(sync, op, ordinal, e)
					}
					if (op == "add" || op == "sub") && ck != ExtPlusInfinity {
						t.Fatal("early kind publication")
					}
					if (op == "mul" || op == "div") && ck != ExtFinite {
						t.Fatal("late kind publication")
					}
					if op == "set" && ak != ExtFinite {
						t.Fatal("failed set kind publication")
					}
				} else {
					completed = true
				}
				if out != nil {
					text, err := out.String()
					serviceMust(t, err)
					literal := "123456789012345678901234567890123456789/23"
					if !strings.HasPrefix(strings.Repeat(" ", 150-strings.IndexByte(literal, '/'))+literal, text) {
						t.Fatal("display prefix", ordinal, text)
					}
				}
				budget.FailAfter(-1)
				serviceMust(t, p.CleanupValue(&output))
				serviceMust(t, p.CleanupValue(&x))
				if out != nil {
					serviceMust(t, out.Close())
				}
				serviceMust(t, p.Close())
				if budget.Used() != 0 {
					t.Fatal("cold refund", sync, op, ordinal, budget.Used())
				}
				if completed {
					break
				}
			}
			if !completed {
				t.Fatal("completion horizon", sync, op)
			}
		}
	}
}

func TestExtNumeralDebugDomainsReleaseErasureAndInvalidKinds(t *testing.T) {
	p, _ := mpqInfFixture(t, false)
	a, b, c := NewRational(), NewRational(), NewRational()
	bind := ExtRationalBinding{p.m, [3]*Rational{&a, &b, &c}}
	for _, tc := range []struct {
		op     string
		ak, bk ExtNumeralKind
	}{
		{"add", ExtMinusInfinity, ExtPlusInfinity}, {"add", ExtPlusInfinity, ExtMinusInfinity}, {"sub", ExtMinusInfinity, ExtMinusInfinity}, {"sub", ExtPlusInfinity, ExtPlusInfinity}, {"div", ExtPlusInfinity, ExtPlusInfinity},
	} {
		ck := ExtNumeralKind(99)
		e := extBinary(&bind, tc.op, tc.ak, tc.bk, &ck)
		if sourceDebug {
			if !errors.Is(e, ErrExtNumeralAssertion) || ck != 99 {
				t.Fatal("debug assertion", tc, e, ck)
			}
		} else {
			serviceMust(t, e)
			if ck != tc.ak {
				t.Fatal("release original branch", tc, ck)
			}
		}
	}
	kind := ExtFinite
	e := ExtInv(&bind, &kind)
	if sourceDebug {
		if !errors.Is(e, ErrExtNumeralAssertion) || a.Num.Val != 0 || a.Den.Val != 1 {
			t.Fatal("debug zero inverse", e)
		}
	} else {
		serviceMust(t, e)
		if a.Num.Val != 1 || a.Den.Val != 0 {
			t.Fatal("release inverse swap")
		}
	}
	// Invalid kind switches have deliberately different reached behavior.
	if _, e = ExtLt(nil, ExtFinite, 99); !errors.Is(e, ErrExtNumeralKind) {
		t.Fatal("reached invalid rhs")
	}
	if _, e = ExtLt(nil, 99, ExtFinite); !errors.Is(e, ErrExtNumeralKind) {
		t.Fatal("reached invalid lhs")
	}
	yes, e := ExtLt(nil, ExtPlusInfinity, 99)
	serviceMust(t, e)
	if yes {
		t.Fatal("erased invalid rhs")
	}
	ck := ExtPlusInfinity
	serviceMust(t, ExtSub(nil, ExtFinite, 99, &ck))
	if ck != ExtPlusInfinity {
		t.Fatal("invalid sub switch changed output")
	}
	serviceMust(t, p.m.Del(&a))
	serviceMust(t, p.m.Del(&b))
	serviceMust(t, p.m.Del(&c))
}

func TestExtNumeralIndependentComparisonOracle(t *testing.T) {
	for _, sync := range []bool{false, true} {
		p, _ := mpqInfFixture(t, sync)
		a, b := NewRational(), NewRational()
		bind := ExtRationalBinding{p.m, [3]*Rational{&a, &b, nil}}
		for _, first := range []string{"-123456789012345678901234567890/23", "0", "98765432109876543210987654321/19"} {
			for _, second := range []string{"-98765432109876543210987654321/17", "0", "123456789012345678901234567890/29"} {
				serviceMust(t, p.m.SetString(&a, first))
				serviceMust(t, p.m.SetString(&b, second))
				x, _ := new(big.Rat).SetString(first)
				y, _ := new(big.Rat).SetString(second)
				for ak := ExtMinusInfinity; ak <= ExtPlusInfinity; ak++ {
					for bk := ExtMinusInfinity; bk <= ExtPlusInfinity; bk++ {
						comparison := int(ak) - int(bk)
						if ak == bk {
							comparison = 0
							if ak == ExtFinite {
								comparison = x.Cmp(y)
							}
						}
						for _, op := range []string{"eq", "neq", "lt", "gt", "le", "ge"} {
							got, e := extRelation(&bind, op, ak, bk)
							serviceMust(t, e)
							want := false
							switch op {
							case "eq":
								want = comparison == 0
							case "neq":
								want = comparison != 0
							case "lt":
								want = comparison < 0
							case "gt":
								want = comparison > 0
							case "le":
								want = comparison <= 0
							case "ge":
								want = comparison >= 0
							}
							if got != want {
								t.Fatal(sync, first, second, ak, bk, op, got, want)
							}
						}
					}
				}
			}
		}
		serviceMust(t, p.m.Del(&a))
		serviceMust(t, p.m.Del(&b))
	}
}

type extNonFieldBinding struct{ ExtRationalBinding }

func (*extNonFieldBinding) Field() bool { return false }
func TestExtNumeralDebugFieldAssertionAndPoisonedErasure(t *testing.T) {
	p, _ := mpqInfFixture(t, false)
	a := NewRational()
	nonField := extNonFieldBinding{ExtRationalBinding{p.m, [3]*Rational{&a, nil, nil}}}
	kind := ExtPlusInfinity
	e := ExtInv(&nonField, &kind)
	if sourceDebug {
		if !errors.Is(e, ErrExtNumeralAssertion) || kind != ExtPlusInfinity {
			t.Fatal("field assertion order", e, kind)
		}
	} else {
		serviceMust(t, e)
		if kind != ExtFinite {
			t.Fatal("release field erasure")
		}
	}
	// A finite zero times infinity erases every read of the poisoned infinite
	// operand. Its kind is considered while its invalid denominator is ignored.
	a = NewRational()
	poison := Rational{}
	c := NewRational()
	binding := ExtRationalBinding{p.m, [3]*Rational{&a, &poison, &c}}
	ck := ExtPlusInfinity
	serviceMust(t, ExtMul(&binding, ExtFinite, ExtPlusInfinity, &ck))
	if ck != ExtFinite || c.Num.Val != 0 {
		t.Fatal("zero/infinity read erasure")
	}
	serviceMust(t, p.m.Del(&a))
	serviceMust(t, p.m.Del(&c))
}
