package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strconv"
	"testing"
)

func sturmProductInputs(t *testing.T, m *UPolynomialManager) (UPolynomial, UPolynomial, UPolynomialView, UPolynomialView) {
	t.Helper()
	a, e := m.NewPolynomial()
	serviceMust(t, e)
	b, e := m.NewPolynomial()
	serviceMust(t, e)
	t.Cleanup(func() {
		contextTLSFinish(t, m.config.Integer.RuntimeOwner(), func() error { return m.DelPolynomial(&a) })
	})
	t.Cleanup(func() {
		contextTLSFinish(t, m.config.Integer.RuntimeOwner(), func() error { return m.DelPolynomial(&b) })
	})
	z := m.config.Integer
	for _, row := range []struct {
		p    UPolynomial
		text []string
	}{{a, []string{"17", "18446744073709551617", "18446744073709551618"}}, {b, []string{"3", "18446744073709551619"}}} {
		values := make([]unumeral.Int, len(row.text))
		for i, text := range row.text {
			serviceMust(t, z.SetString(&values[i], text))
		}
		serviceMust(t, m.SetInts(row.p, values))
		for i := range values {
			serviceMust(t, z.Del(&values[i]))
		}
	}
	av, e := m.wholeView(a)
	serviceMust(t, e)
	bv, e := m.wholeView(b)
	serviceMust(t, e)
	return a, b, av, bv
}
func sturmProductExpect(t *testing.T, m *UPolynomialManager, v UPolynomialView) {
	t.Helper()
	if v.count != 3 {
		t.Fatal("product size", v.count)
	}
	for i, text := range []string{"55340232221128654851", "340282366920938463647842048168863727631", "680564733841876927111216655600631939084"} {
		got, e := m.read(v, uint32(i))
		serviceMust(t, e)
		var expected unumeral.Int
		serviceMust(t, m.config.Integer.SetString(&expected, text))
		equal, e := m.config.Integer.Equal(got, &expected)
		serviceMust(t, e)
		serviceMust(t, m.config.Integer.Del(&expected))
		if !equal {
			t.Fatal("independent derivative-times-polynomial coefficient", i, text)
		}
	}
}
func TestUPolynomialSequenceSturmProductReachedReservations(t *testing.T) {
	success, unjoined := false, false
	for after := int64(0); after < 256 && !success; after++ {
		t.Run(strconv.FormatInt(after, 10), func(t *testing.T) {
			c, m, _ := algebraicNegTLSFixture(t, nil)
			a, b, av, bv := sturmProductInputs(t, m)
			z := m.config.Integer
			s, e := m.NewSequence()
			serviceMust(t, e)
			q, e := s.beginSturm()
			serviceMust(t, e)
			serviceMust(t, q.constructPrime())
			serviceMust(t, q.pushCopy(av))
			serviceMust(t, m.derivativeStateWithSourceContext(av, &q.prime, q.context()))
			z.Budget().FailAfter(after)
			body := m.mulWithSourceContext(m.stateView(&q.prime), bv, &q.prime, &q.product, q.context())
			z.Budget().FailAfter(-1)
			tmp := unumeral.NumericCellRef(0)
			if body == nil {
				success = true
				sturmProductExpect(t, m, m.stateView(&q.prime))
			} else {
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal(body)
				}
				if z.MatchesAddMulInvocation(&q.product) && z.AddMulInvocation.Stage < 3 && unumeral.ValidNumericRef(z.Cells, z.AddMulInvocation.Tmp.Cell) && !z.Cell(z.AddMulInvocation.Tmp.Cell).Released {
					unjoined = true
					tmp = z.AddMulInvocation.Tmp.Cell
					if z.AddMulInvocation.SourceCause != nil {
						t.Fatal("body failure invented an END destructor")
					}
				}
			}
			_ = q.finish(body)
			if tmp != 0 && (!unumeral.ValidNumericRef(z.Cells, tmp) || z.Cell(tmp).Released) {
				t.Fatal("plain RAW TMP received a destructor without reached END")
			}
			source := q.sourceCursor
			serviceMust(t, s.CleanupSturm())
			if q.active || q.pending() || source != q.sourceCursor {
				t.Fatal("product owning acknowledgement changed source history")
			}
			serviceMust(t, s.Close())
			algebraicCellSourcePhysical(t, c, []UPolynomial{a, b}, nil, nil)
		})
	}
	if !success || !unjoined {
		t.Fatal("product reservation campaign incomplete", success, unjoined)
	}
}

func TestUPolynomialSequenceSturmProductEndStopProtectsPrime(t *testing.T) {
	c, m, runtime := algebraicNegTLSFixture(t, nil)
	a, b, av, bv := sturmProductInputs(t, m)
	z := m.config.Integer
	s, e := m.NewSequence()
	serviceMust(t, e)
	q, e := s.beginSturm()
	serviceMust(t, e)
	serviceMust(t, q.constructPrime())
	serviceMust(t, q.pushCopy(av))
	serviceMust(t, m.derivativeStateWithSourceContext(av, &q.prime, q.context()))
	// Real large multiplication warms the shared result scratch. Product reset
	// still executes literally on the initially empty manager working vector.
	var warm, result unumeral.Int
	digits := make([]uint32, 17)
	digits[16] = 1
	serviceMust(t, z.SetLimbs(&warm, digits))
	serviceMust(t, z.Mul(&warm, &warm, &result))
	serviceMust(t, z.Del(&result))
	serviceMust(t, z.Del(&warm))
	armed := false
	var lane umemory.Block
	m.config.Checkpoint = func() bool {
		if armed {
			return true
		}
		armed = true
		contextTLSNearMinimum(t, runtime)
		lane, e = runtime.Allocate(8)
		serviceMust(t, e)
		delta := int64(-1000000)
		if e = runtime.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
			t.Fatal(e)
		}
		return true
	}
	m.config.Cooperation = func() error { return nil }
	body := m.mulWithSourceContext(m.stateView(&q.prime), bv, &q.prime, &q.product, q.context())
	body = q.finish(body)
	if !armed || !errors.Is(body, arena.ErrBounds) || !z.MatchesAddMulInvocation(&q.product) || z.AddMulInvocation.Stage != 3 || z.AddMulInvocation.SourceCause == nil || q.childStop == nil || q.sourceStop != nil || !q.live[1] || q.sourceCursor != ([2]uint32{}) {
		t.Fatal("actual product END stop did not protect older prime", body, z.AddMulInvocation.Stage, q.sourceCursor)
	}
	tmp := z.AddMulInvocation.Tmp.Cell
	if !unumeral.ValidNumericRef(z.Cells, tmp) || !z.Cell(tmp).Released {
		t.Fatal("literal END did not consume physical TMP")
	}
	for _, state := range []*upolyVectorState{&q.prime, &s.state.coefficients} {
		for _, value := range m.values(state)[:state.size] {
			if value.Cell != 0 && z.Cell(value.Cell).Released {
				t.Fatal("older source coefficient destroyed after END stop")
			}
		}
	}
	if e := m.SourceFinalize(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("source manager crossed stopped product", e)
	}
	source := q.sourceCursor
	contextTLSFinish(t, runtime, s.CleanupSturm)
	if q.active || source != q.sourceCursor || z.MatchesAddMulInvocation(&q.product) {
		t.Fatal("product cleanup rewrote source history")
	}
	contextTLSFinish(t, runtime, s.Close)
	contextTLSFinish(t, runtime, func() error {
		if lane == (umemory.Block{}) {
			return nil
		}
		return runtime.Deallocate(&lane)
	})
	algebraicCellSourcePhysical(t, c, []UPolynomial{a, b}, nil, nil)
}

func TestUPolynomialSequenceSturmProductPublicReachedReservations(t *testing.T) {
	success, productDebt := false, false
	for after := int64(0); after < 1024 && !success; after++ {
		t.Run(strconv.FormatInt(after, 10), func(t *testing.T) {
			c, m, _ := algebraicNegTLSFixture(t, nil)
			a, b, av, bv := sturmProductInputs(t, m)
			z := m.config.Integer
			s, e := m.NewSequence()
			serviceMust(t, e)
			z.Budget().FailAfter(after)
			body := m.SturmTarskiSequence(av, bv, s)
			z.Budget().FailAfter(-1)
			q := &s.sturm
			if z.MatchesAddMulInvocation(&q.product) {
				productDebt = true
			}
			if body == nil {
				success = true
				row, e := s.RowView(1)
				serviceMust(t, e)
				sturmProductExpect(t, m, row)
			} else if !numericPhysicalReservationRefusal(body) {
				t.Fatal(body)
			}
			source := q.sourceCursor
			serviceMust(t, s.CleanupSturm())
			if q.active || q.pending() || source != q.sourceCursor || m.sturmPending != nil {
				t.Fatal("public product acknowledgement lost invocation ownership")
			}
			serviceMust(t, s.Close())
			algebraicCellSourcePhysical(t, c, []UPolynomial{a, b}, nil, nil)
		})
	}
	if !success || !productDebt {
		t.Fatal("public Tarski reservation campaign incomplete", success, productDebt)
	}
}
