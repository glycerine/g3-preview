// Source: src/util/mpf.cpp:590,640,664; formatter exceptions in MUL TRACE.
package numeral

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestArbitraryFloatMulTraceFormatterRefusalsStopAtSourceSite(t *testing.T) {
	probe := diagnostic.NewTraceTagsWithWriter(&bytes.Buffer{})
	probe.Enable("mpf_mul_bug")
	if !probe.ShouldTrace("mpf_mul_bug") {
		return
	}
	for _, tc := range []struct {
		name, tag, site string
		general         bool
	}{
		{"entry", "mpf_mul_bug", "590", false},
		{"inner", "mpf_dbg", "640", true},
		{"result", "mpf_mul_bug", "664", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			found := false
			for ordinal := int64(0); ordinal < 256 && !found; ordinal++ {
				t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
					m, runtime, budget := mpfNumeralHelpersFixture(t)
					var output bytes.Buffer
					tags := diagnostic.NewTraceTagsWithWriter(&output)
					tags.Enable(tc.tag)
					serviceMust(t, m.BindTrace(tags))
					var x, y, out ArbitraryFloat
					if tc.general {
						serviceMust(t, m.MkOne(11, 53, false, &x))
					} else {
						serviceMust(t, m.MkZero(11, 53, false, &x))
					}
					serviceMust(t, m.MkOne(11, 53, false, &y))
					beforeX, beforeY, beforeOut := x, y, out
					budget.FailAfter(ordinal)
					body := m.Mul(ArbitraryFloatNearestEven, &x, &y, &out)
					budget.FailAfter(-1)
					if body != nil && !numericPhysicalReservationRefusal(body) {
						t.Fatalf("unexpected MUL refusal: %v", body)
					}
					header := []byte("mul src/util/mpf.cpp:" + tc.site + " ---------")
					next := ""
					if tc.name == "inner" {
						next = "641"
					}
					if body != nil && bytes.Contains(output.Bytes(), header) && (next == "" || !bytes.Contains(output.Bytes(), []byte("mul src/util/mpf.cpp:"+next+" ---------"))) {
						found = true
						if x != beforeX || y != beforeY {
							t.Fatal("formatter refusal changed borrowed inputs")
						}
						if tc.name == "entry" && (out != beforeOut || m.mul.active) {
							t.Fatal("entry formatter refusal reached multiplication")
						}
						if tc.name == "result" && (out == beforeOut || m.mul.active) {
							t.Fatal("result formatter refusal lost completed output")
						}
						if tc.general && !m.mul.active {
							t.Fatal("inner formatter refusal lost its pending multiplication scope")
						}
						integerPhysicalAudit(t, m.integer, &x.significand, &y.significand, &out.significand, &m.mul.a.significand, &m.mul.b.significand, &m.mul.sticky, &m.RoundScope.sigma, &m.RoundScope.cap, &m.RoundScope.sticky, &m.rational.NTmp, &m.rational.AddTmp1, &m.rational.AddTmp2, &m.rational.AddMulTmp.Num, &m.rational.AddMulTmp.Den, &m.rational.LtTmp1.Num, &m.rational.LtTmp1.Den, &m.rational.LtTmp2.Num, &m.rational.LtTmp2.Den)
						if budget.Used() != m.OwnedStorageBytes()+runtime.Stats().TotalBytes {
							t.Fatal("formatter refusal retained unowned backing", budget.Used(), m.OwnedStorageBytes(), runtime.Stats())
						}
					}
					serviceMust(t, m.CleanupMul())
					serviceMust(t, m.CleanupValue(&out))
					serviceMust(t, m.Del(&x))
					serviceMust(t, m.Del(&y))
					if e := m.Close(); e != nil {
						z := m.integer
						if !errors.Is(e, arena.ErrInvalidSlot) || !z.sourceGuard || z.SourceFinalizeStage < 2 || z.SourceFinalizeStage > 5 {
							t.Fatalf("unexpected manager source stop: %v", e)
						}
						index := int(z.SourceFinalizeStage-2) / 2
						ref := z.Tmp[index]
						if z.SourceFinalizeStage%2 == 1 {
							ref = z.Arg[index]
						}
						if ref != 0 {
							t.Fatalf("unproved source scratch hole: %v", ref)
						}
						serviceMust(t, m.Close())
					}
				})
			}
			if !found {
				t.Fatalf("MUL trace formatter refusal did not reach source line %s", tc.site)
			}
		})
	}
}

type refusingMulTraceWriter struct{}

func (refusingMulTraceWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestArbitraryFloatMulTraceWriterFailbitDoesNotStopArithmetic(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	tags := diagnostic.NewTraceTagsWithWriter(refusingMulTraceWriter{})
	tags.Enable("mpf_mul_bug")
	tags.Enable("mpf_dbg")
	serviceMust(t, m.BindTrace(tags))
	var x, y, out ArbitraryFloat
	serviceMust(t, m.MkOne(11, 53, false, &x))
	serviceMust(t, m.MkOne(11, 53, false, &y))
	serviceMust(t, m.Mul(ArbitraryFloatNearestEven, &x, &y, &out))
	one, e := m.IsOne(&out)
	serviceMust(t, e)
	if !one {
		t.Fatal("trace writer failbit changed multiplication")
	}
	serviceMust(t, m.CleanupMul())
	serviceMust(t, m.Del(&x))
	serviceMust(t, m.Del(&y))
	serviceMust(t, m.Del(&out))
}
