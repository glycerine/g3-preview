// Source: src/test/ext_numeral.cpp:24-397; src/util/ext_numeral.h:25-345.
package util

import (
	"encoding/json"
	"fmt"
	"os"
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
