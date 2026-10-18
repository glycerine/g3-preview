package polynomial

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type factorDegreeTraceProbe struct {
	t                           *testing.T
	service                     *UPolynomialFactorizationService
	seen                        strings.Builder
	checked                     map[*UPolynomialManager]bool
	flushes                     int
	headerChecked, flushChecked bool
}

type factorDegreeTraceFailure struct {
	t     *testing.T
	z     *unumeral.IntegerManager
	held  []unumeral.Int
	count int
	limit uint32
	armed bool
	seen  strings.Builder
}

func (w *factorDegreeTraceFailure) Write(p []byte) (int, error) {
	if !w.armed && string(p) == "zpe_fs" {
		w.armed = true
		// Fill the real descriptor limit using caller-owned numeral copies.
		// No quota, source scratch, or source construction count is rewritten.
		for {
			stats := w.z.Cells.Stats()
			if uint64(stats.Live)+uint64(stats.Retired) >= uint64(w.limit) {
				break
			}
			if e := w.z.Copy(&w.held[w.count], &w.z.Two64); e != nil {
				w.t.Fatal("caller descriptor padding", e, stats, w.count)
			}
			w.count++
		}
	}
	return w.seen.Write(p)
}

func TestUPolynomialFactorDegreeTraceChildRefusalStopsOlderLocals(t *testing.T) {
	q, s := preparationPhysicalFixtureWithSlots(t, 256)
	m, z := s.manager, q.Integers()
	var coefficients [3]unumeral.Int
	serviceMust(t, z.SetString(&coefficients[0], "1606938044258990275541962092344965554322887681987282944917506"))
	serviceMust(t, z.SetString(&coefficients[1], "-2535301200456458802993406410755"))
	coefficients[2] = unumeral.NewInt(1)
	p, e := m.NewPolynomial()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.DelPolynomial(&p)) })
	serviceMust(t, m.SetInts(p, coefficients[:]))
	for i := range coefficients {
		serviceMust(t, z.Del(&coefficients[i]))
	}
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, fs.Close()) })
	w := factorDegreeTraceFailure{t: t, z: z, limit: 256, held: make([]unumeral.Int, 256)}
	t.Cleanup(func() {
		serviceMust(t, s.CleanupRetained())
		for i := 0; i < w.count; i++ {
			serviceMust(t, z.Del(&w.held[i]))
		}
	})
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("polynomial::factorization::bughunt")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	_, body := s.FactorSquareFreeDefault(mustUPolyView(t, m, p), fs, DefaultUPolynomialFactorParams())
	z.Budget().FailAfter(-1)
	if !tags.ShouldTrace("polynomial::factorization::bughunt") {
		serviceMust(t, body)
		return
	}
	if !w.armed || !errors.Is(body, arena.ErrSlotLimit) || s.scope == nil || !s.scope.trace.pending() || s.scope.constructed != 14 {
		t.Fatal("actual lifted-factor display refusal/frontier", body, w.armed, s.scope != nil)
	}
	if fs.count != 0 || !strings.HasSuffix(w.seen.String(), " * (x") {
		t.Fatal("output/source publication prefix", fs.count, w.seen.String())
	}
	roots := make([]*unumeral.Int, w.count)
	for i := range roots {
		roots[i] = &w.held[i]
	}
	upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, nil, upolyPhysicalExtraOwners{
		Services: []*UPolynomialFactorizationService{s}, RationalManagers: []*unumeral.RationalManager{q}, Integers: roots,
	})
	serviceMust(t, s.CleanupRetained())
	if s.scope != nil {
		t.Fatal("trace owning continuation retained older locals")
	}
}

func (w *factorDegreeTraceProbe) checkEnclosing() {
	c := w.service.scope
	if c == nil || !c.traceBorrow.executing {
		return
	}
	before := c.constructed
	for _, m := range c.traceBorrow.managers() {
		if m != nil {
			for _, e := range []error{m.Close(), m.SourceFinalize(), m.CleanupDisplay()} {
				if !errors.Is(e, ErrUPolynomialCleanupPending) {
					w.t.Error("whole TRACE callback acquired manager", e)
				}
			}
		}
	}
	for _, fs := range []*UPolynomialFactors{c.best, c.lifted, c.primeTrial.current, c.primePreparation.factors} {
		if fs != nil && !fs.closed {
			for _, e := range []error{fs.Close(), fs.Clear(), fs.CleanupDisplay()} {
				if !errors.Is(e, ErrUPolynomialCleanupPending) {
					w.t.Error("whole TRACE callback acquired factor owner", e)
				}
			}
		}
	}
	if c.constructed != before {
		w.t.Error("TRACE callback advanced enclosing source lifetime")
	}
}

func (w *factorDegreeTraceProbe) Write(p []byte) (int, error) {
	s := w.service
	if !w.headerChecked && s != nil && s.scope != nil && s.scope.traceBorrow.executing {
		w.headerChecked = true
		w.checkEnclosing()
	}
	if s != nil && s.scope != nil && s.scope.trace.matches() && s.scope.trace.frame.executing {
		fs := s.scope.trace.owner
		if !w.checked[fs.manager] {
			w.checked[fs.manager] = true
			for _, e := range []error{fs.Close(), fs.Clear(), fs.CleanupDisplay(), fs.manager.SourceFinalize(), fs.manager.Close(), s.CleanupRetained(), s.Close()} {
				if !errors.Is(e, ErrUPolynomialCleanupPending) {
					w.t.Error("factor trace callback acquired active owner", e)
				}
			}
		}
	}
	return w.seen.Write(p)
}
func (w *factorDegreeTraceProbe) Flush() error {
	w.flushes++
	if !w.flushChecked && w.service.scope != nil && w.service.scope.traceBorrow.executing {
		w.flushChecked = true
		w.checkEnclosing()
	}
	return nil
}

func TestUPolynomialFactorDegreeTraceRetainedRecordsAndOwners(t *testing.T) {
	q, s := preparationPhysicalFixture(t)
	w := factorDegreeTraceProbe{t: t, service: s, checked: map[*UPolynomialManager]bool{}}
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("polynomial::factorization::bughunt")
	serviceMust(t, q.Integers().BindTrace(tags))
	serviceMust(t, s.manager.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, q.Integers().BindTrace(nil)) })
	t.Cleanup(func() { serviceMust(t, s.manager.modular.BindTrace(nil)) })
	m := s.manager
	p := upolySourceNew(t, m, []int32{2, 3, 1})
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, fs.Close()) })
	_, e = s.FactorSquareFreeDefault(mustUPolyView(t, m, p), fs, DefaultUPolynomialFactorParams())
	serviceMust(t, e)
	if tags.ShouldTrace("polynomial::factorization::bughunt") {
		text := w.seen.String()
		for _, site := range []string{":1088 ", "sage: if (not (gcd(2, 1)) == 1): print 'Error, wrong gcd'\n", ":1141 ", ":1155 ", ":1192 ", "best degree set: [0, 1, 2] represented by 111\n", "degree_set = [0, 1, 2] represented by 111\n", "STARTING TRIAL DIVISION\nzpe_fs1 * ("} {
			if !strings.Contains(text, site) {
				t.Fatal("missing source trace record", site, text)
			}
		}
		if strings.Count(text, "best zp factorization (Z_2): ") != 2 || len(w.checked) != 2 || w.flushes < 10 || !w.headerChecked || !w.flushChecked {
			t.Fatal("record order/borrowed managers/endl", text, len(w.checked), w.flushes)
		}
	} else if w.seen.Len() != 0 || w.flushes != 0 || len(w.checked) != 0 {
		t.Fatal("erased trace touched display")
	}
	if n, e := fs.DistinctFactors(); e != nil || n != 2 {
		t.Fatal("factorization after trace", n, e)
	}
}

type factorDegreeRefusedTrace struct {
	calls, flushes int
	cause          error
}

func (w *factorDegreeRefusedTrace) Write(p []byte) (int, error) { w.calls++; return 0, w.cause }
func (w *factorDegreeRefusedTrace) Flush() error                { w.flushes++; return nil }

func TestUPolynomialFactorDegreeTraceStreamFailureStillComputesDisplays(t *testing.T) {
	q, s := preparationPhysicalFixture(t)
	m := s.manager
	w := factorDegreeRefusedTrace{cause: errors.New("refused trace output")}
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("polynomial::factorization::bughunt")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	var prime, lift *UPolynomialManager
	checkpoint := m.config.Checkpoint
	m.config.Checkpoint = func() bool {
		if s.scope != nil {
			if s.scope.zp != nil {
				prime = s.scope.zp
			}
			if s.scope.zpe != nil {
				lift = s.scope.zpe
			}
		}
		return checkpoint()
	}
	p := upolySourceNew(t, m, []int32{2, 3, 1})
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, fs.Close()) })
	completed, e := s.FactorSquareFreeDefault(mustUPolyView(t, m, p), fs, DefaultUPolynomialFactorParams())
	serviceMust(t, e)
	if !completed {
		t.Fatal("ostream failure changed factorization result")
	}
	if tags.ShouldTrace("polynomial::factorization::bughunt") {
		if w.calls != 1 || w.flushes < 3 || prime == nil || lift == nil || prime.displayNonce < 4 || lift.displayNonce < 2 {
			t.Fatal("badbit skipped source display operands or retried output", w.calls, w.flushes)
		}
	} else if w.calls != 0 || w.flushes != 0 {
		t.Fatal("erased stream output")
	}
	out := upolySourceNew(t, m, nil)
	serviceMust(t, fs.Multiply(out))
	sxPolyCoefficients(t, m, out, []int32{2, 3, 1})
	serviceMust(t, q.CheckOwnerReady())
}

func TestUPolynomialFactorDegreeTraceDisabledSkipsUnreachedOperands(t *testing.T) {
	_, s := preparationPhysicalFixture(t)
	tags := diagnostic.NewTraceTagsWithWriter(nil)
	serviceMust(t, s.manager.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, s.manager.modular.BindTrace(nil)) })
	before := s.manager.config.Integer.Budget().Used()
	s.manager.config.Integer.Budget().FailAfter(0)
	t.Cleanup(func() { s.manager.config.Integer.Budget().FailAfter(-1) })
	// No enclosing source scope or factor operand exists at this erased site.
	serviceMust(t, s.traceFactorDegreeSet(nil, 1155, false))
	serviceMust(t, s.traceFactorPrimeGCD(nil))
	if s.scope != nil || s.manager.config.Integer.Budget().Used() != before {
		t.Fatal("disabled trace acquired operands or scope")
	}
}

func TestUPolynomialFactorPrimeGCDTraceRejectedPrimeOrder(t *testing.T) {
	_, s := preparationPhysicalFixture(t)
	m := s.manager
	var text strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&text)
	tags.Enable("polynomial::factorization::bughunt")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	// The first candidate divides the actual leading coefficient and is rejected.
	p := upolySourceNew(t, m, []int32{1, 1, 2})
	fs, e := m.NewFactors()
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, fs.Close()) })
	_, e = s.FactorSquareFreeDefault(mustUPolyView(t, m, p), fs, DefaultUPolynomialFactorParams())
	serviceMust(t, e)
	if !tags.ShouldTrace("polynomial::factorization::bughunt") {
		if text.Len() != 0 {
			t.Fatal("erased GCD trace wrote output")
		}
		return
	}
	a := strings.Index(text.String(), "sage: if (not (gcd(2, 2)) == 2): print 'Error, wrong gcd'\n")
	b := strings.Index(text.String(), "sage: if (not (gcd(3, 2)) == 1): print 'Error, wrong gcd'\n")
	if a < 0 || b <= a {
		t.Fatal("rejected and accepted prime trace order", text.String())
	}
}
