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
	polynomialChecked           map[*UPolynomialManager]bool
	flushes                     int
	headerChecked, flushChecked bool
	quadraticChecked            bool
}

type factorDegreeTraceFailure struct {
	t       *testing.T
	z       *unumeral.IntegerManager
	held    []unumeral.Int
	count   int
	limit   uint32
	trigger string
	armed   bool
	seen    strings.Builder
}

func (w *factorDegreeTraceFailure) Write(p []byte) (int, error) {
	trigger := w.trigger
	if trigger == "" {
		trigger = "zpe_fs"
	}
	if !w.armed && string(p) == trigger {
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
	if !w.quadraticChecked && string(p) == "A_lifted = " && s != nil && s.scope != nil && s.scope.hensel.related != nil && s.scope.traceBorrow.executing {
		w.quadraticChecked = true
		w.checkEnclosing()
	}
	if !w.headerChecked && s != nil && s.scope != nil && s.scope.traceBorrow.executing {
		w.headerChecked = true
		w.checkEnclosing()
	}
	if s != nil && s.scope != nil {
		r := &s.scope.tracePolynomial
		if r.owner.displayReceiptPending(r) && r.frame.executing {
			if w.polynomialChecked == nil {
				w.polynomialChecked = map[*UPolynomialManager]bool{}
			}
			if !w.polynomialChecked[r.owner] {
				w.polynomialChecked[r.owner] = true
				w.checkEnclosing()
			}
		}
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
		for _, site := range []string{":744 ", ":771 ", "A_lifted = x + 2\nB_lifted = x + 1\nC = x^2 + 3 x + 2\n", "g = (1 - A_lifted*U - B_lifted*V)/2\ng == x + 1\n", "g = (1 - A_lifted*U - B_lifted*V)/4\ng == 0\n", ":637 ", ":641 ", ":646 ", ":653 ", ":658 ", ":663 ", ":886 ", ":960 ", "polynomial::hansel_lift(x^2 + 3 x + 2, 1 * (x)^1 * (x + 1)^1) => 1 * (x + 2)^1 * (x + 1)^1\n", ":906 ", ":921 ", ":927 ", "A = x\n", "C = x^2 + x\n", "B = x + 1\n", ":1274 ", "factors = 1 * (x + 2)^1\nf_pp*lc(f_pp) = x + 1\nlc(f_pp) = 1\n", ":1239 ", ":1246 ", "result = true\n", ":1052 ", ":1100 ", "sage: f_pp = x^2 + 3 x + 2\n", "sage: Rp.<x_p> = GF(2)['x_p']\nsage: f_pp_zp = x_p^2 + x_p\n", ":1163 ", "out p = 2, and we'll work p^e for e = 4\n", ":1088 ", "sage: if (not (gcd(2, 1)) == 1): print 'Error, wrong gcd'\n", ":1141 ", ":1155 ", ":1192 ", "best degree set: [0, 1, 2] represented by 111\n", "degree_set = [0, 1, 2] represented by 111\n", "STARTING TRIAL DIVISION\nzpe_fs1 * ("} {
			if !strings.Contains(text, site) {
				t.Fatal("missing source trace record", site, text)
			}
		}
		if strings.Count(text, "best zp factorization (Z_2): ") != 2 || len(w.checked) != 3 || len(w.polynomialChecked) != 2 || w.flushes < 10 || !w.headerChecked || !w.flushChecked || !w.quadraticChecked {
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
	serviceMust(t, s.traceFactorLiftExponent(0))
	serviceMust(t, s.traceFactorPolynomial(nil, UPolynomialView{}, 1052))
	serviceMust(t, s.traceRecombinationDivision(1239, false))
	serviceMust(t, s.traceRecombinationDivision(1246, false))
	serviceMust(t, s.traceRecombinationFactors(nil))
	serviceMust(t, s.traceHenselVectors(nil, 906))
	serviceMust(t, s.traceHenselVectors(nil, 935))
	serviceMust(t, s.traceHenselFactors(nil, UPolynomialView{}, nil, nil, false))
	serviceMust(t, s.traceHenselFactors(nil, UPolynomialView{}, nil, nil, true))
	serviceMust(t, s.traceHenselPairStage(nil, 614))
	serviceMust(t, s.traceHenselPairStage(nil, 637))
	serviceMust(t, s.traceHenselPairEntry(nil, nil, nil, nil))
	serviceMust(t, s.traceHenselQuadratic(nil, UPolynomialView{}, nil, 744))
	serviceMust(t, s.traceHenselQuadratic(nil, UPolynomialView{}, nil, 771))
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

func TestUPolynomialFactorPrimitiveTraceChildRefusalStopsPrimitiveVector(t *testing.T) {
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
	w := factorDegreeTraceFailure{t: t, z: z, limit: 256, held: make([]unumeral.Int, 256), trigger: "sage: f_pp = "}
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
	if !tags.ShouldTrace("polynomial::factorization::bughunt") {
		serviceMust(t, body)
		return
	}
	c := s.scope
	if !errors.Is(body, arena.ErrSlotLimit) || !w.armed || c == nil || c.constructed != 1 || !c.tracePending() || !m.displayReceiptPending(&c.tracePolynomial) {
		t.Fatal("primitive display child/frontier", body, w.armed, c)
	}
	if c.fpp.size != 3 || fs.count != 0 || c.zp != nil {
		t.Fatal("older primitive vector lost or younger prime constructed")
	}
	extras := upolyPhysicalExtraOwners{Services: []*UPolynomialFactorizationService{s}, RationalManagers: []*unumeral.RationalManager{q}}
	for i := 0; i < w.count; i++ {
		extras.Integers = append(extras.Integers, &w.held[i])
	}
	upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, nil, extras)
	serviceMust(t, s.CleanupRetained())
	if s.scope != nil {
		t.Fatal("explicit trace cleanup retained source scope")
	}
}

func TestUPolynomialFactorRecombinationTraceChildRefusalStopsOlderLocals(t *testing.T) {
	for _, trigger := range []string{"f_pp*lc(f_pp) = ", "trial_factor_quo = ", "factors = "} {
		t.Run(trigger, func(t *testing.T) { factorRecombinationTraceChildRefusal(t, trigger) })
	}
}
func factorRecombinationTraceChildRefusal(t *testing.T, trigger string) {
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
	w := factorDegreeTraceFailure{t: t, z: z, limit: 256, held: make([]unumeral.Int, 256), trigger: trigger}
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
	var receipt upolyFactorSquareFreeInvocationReceipt
	_, body := s.factorSquareFreeWithReceipt(mustUPolyView(t, m, p), fs, 1, DefaultUPolynomialFactorParams(), &receipt)
	if !tags.ShouldTrace("polynomial::factorization::bughunt") {
		serviceMust(t, body)
		return
	}
	c := s.scope
	if !errors.Is(body, arena.ErrSlotLimit) || !w.armed || c == nil || c.constructed != 14 || !c.tracePending() {
		t.Fatal("recombination display child/frontier", body, w.armed, c)
	}
	if !errors.Is(s.factorSquareFreeInvocationSourceCause(&receipt), ErrUPolynomialCleanupPending) {
		t.Fatal("enclosing source caller lost exact display blocker", s.factorSquareFreeInvocationSourceCause(&receipt), s.matchesSquareFreeHistory(&receipt), c.traceSourceBlocker())
	}
	if c.zp == nil || !c.recombinationStep.active {
		t.Fatal("older recombination scope lost")
	}
	if trigger == "factors = " {
		if c.fpp.size != 2 || fs.count != 1 || !c.trace.pending() {
			t.Fatal("published factor or exact factor-display child lost")
		}
	} else if c.fpp.size != 3 || fs.count != 0 || !m.displayReceiptPending(&c.tracePolynomial) {
		t.Fatal("older recombination locals lost")
	}
	if trigger == "trial_factor_quo = " && c.quotient.size != 2 {
		t.Fatal("actual division quotient lost")
	}

	extras := upolyPhysicalExtraOwners{Services: []*UPolynomialFactorizationService{s}, RationalManagers: []*unumeral.RationalManager{q}}
	for i := 0; i < w.count; i++ {
		extras.Integers = append(extras.Integers, &w.held[i])
	}
	upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, []*UPolynomialFactors{fs}, extras)
	serviceMust(t, s.CleanupRetained())
	if s.scope != nil {
		t.Fatal("explicit trace cleanup retained source scope")
	}
}

func TestUPolynomialFactorHenselTraceChildRefusalStopsOuterVectors(t *testing.T) {
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
	w := factorDegreeTraceFailure{t: t, z: z, limit: 256, held: make([]unumeral.Int, 256), trigger: "lifted to "}
	t.Cleanup(func() {
		serviceMust(t, s.CleanupRetained())
		for i := 0; i < w.count; i++ {
			serviceMust(t, z.Del(&w.held[i]))
		}
	})
	tags := diagnostic.NewTraceTagsWithWriter(&w)
	tags.Enable("polynomial::factorization")
	serviceMust(t, m.modular.BindTrace(tags))
	t.Cleanup(func() { serviceMust(t, m.modular.BindTrace(nil)) })
	var receipt upolyFactorSquareFreeInvocationReceipt
	_, body := s.factorSquareFreeWithReceipt(mustUPolyView(t, m, p), fs, 1, DefaultUPolynomialFactorParams(), &receipt)
	if !tags.ShouldTrace("polynomial::factorization") {
		serviceMust(t, body)
		return
	}
	c := s.scope
	if !errors.Is(body, arena.ErrSlotLimit) || !w.armed || c == nil || c.constructed != 9 || !c.tracePending() {
		t.Fatal("recombination display child/frontier", body, w.armed, c)
	}
	if !errors.Is(s.factorSquareFreeInvocationSourceCause(&receipt), ErrUPolynomialCleanupPending) {
		t.Fatal("enclosing source caller lost exact display blocker", s.factorSquareFreeInvocationSourceCause(&receipt), s.matchesSquareFreeHistory(&receipt), c.traceSourceBlocker())
	}
	if c.zp == nil || !c.hensel.active || c.hensel.outerConstructed != 4 || c.fpp.size != 3 || fs.count != 0 || c.lifted.count != 0 || !m.displayReceiptPending(&c.tracePolynomial) {
		t.Fatal("older Hensel vectors lost or factor prematurely published")
	}
	extras := upolyPhysicalExtraOwners{Services: []*UPolynomialFactorizationService{s}, RationalManagers: []*unumeral.RationalManager{q}}
	for i := 0; i < w.count; i++ {
		extras.Integers = append(extras.Integers, &w.held[i])
	}
	upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, []*UPolynomialFactors{fs}, extras)
	serviceMust(t, s.CleanupRetained())
	if s.scope != nil {
		t.Fatal("explicit trace cleanup retained source scope")
	}
}

func TestUPolynomialFactorHenselBoundaryTraceExactChildren(t *testing.T) {
	for _, exit := range []bool{false, true} {
		name := "entry"
		if exit {
			name = "exit"
		}
		t.Run(name, func(t *testing.T) { factorHenselBoundaryTraceChildRefusal(t, exit) })
	}
}
func factorHenselBoundaryTraceChildRefusal(t *testing.T, exit bool) {
	trigger := "polynomial::hansel_lift("
	if exit {
		trigger = ") => "
	}
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
	w := factorDegreeTraceFailure{t: t, z: z, limit: 256, held: make([]unumeral.Int, 256), trigger: trigger}
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
	var receipt upolyFactorSquareFreeInvocationReceipt
	_, body := s.factorSquareFreeWithReceipt(mustUPolyView(t, m, p), fs, 1, DefaultUPolynomialFactorParams(), &receipt)
	if !tags.ShouldTrace("polynomial::factorization::bughunt") {
		serviceMust(t, body)
		return
	}
	c := s.scope
	if !errors.Is(body, arena.ErrSlotLimit) || !w.armed || c == nil || c.constructed != 9 || !c.tracePending() {
		t.Fatal("recombination display child/frontier", body, w.armed, c)
	}
	if !errors.Is(s.factorSquareFreeInvocationSourceCause(&receipt), ErrUPolynomialCleanupPending) {
		t.Fatal("enclosing source caller lost exact display blocker", s.factorSquareFreeInvocationSourceCause(&receipt), s.matchesSquareFreeHistory(&receipt), c.traceSourceBlocker())
	}
	if c.zp == nil || !c.hensel.active || c.fpp.size != 3 || fs.count != 0 {
		t.Fatal("older Hensel scope lost")
	}
	if exit {
		if c.hensel.outerConstructed != 4 || !c.hensel.finalLCInvConstructed || c.hensel.finalLCInvEnded || c.lifted.count != 2 || !c.trace.pending() {
			t.Fatal("exit trace lost older inverse/vectors or published factors")
		}
	} else if c.hensel.outerConstructed != 0 || c.hensel.finalLCInvConstructed || c.lifted.count != 0 || !m.displayReceiptPending(&c.tracePolynomial) {
		t.Fatal("entry trace constructed younger source locals")
	}
	extras := upolyPhysicalExtraOwners{Services: []*UPolynomialFactorizationService{s}, RationalManagers: []*unumeral.RationalManager{q}}
	for i := 0; i < w.count; i++ {
		extras.Integers = append(extras.Integers, &w.held[i])
	}
	upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, []*UPolynomialFactors{fs}, extras)
	serviceMust(t, s.CleanupRetained())
	if s.scope != nil {
		t.Fatal("explicit trace cleanup retained source scope")
	}
}

func TestUPolynomialFactorHenselQuadraticTraceChildBeforeIteration(t *testing.T) {
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
	w := factorDegreeTraceFailure{t: t, z: z, limit: 256, held: make([]unumeral.Int, 256), trigger: "A_lifted = "}
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
	var receipt upolyFactorSquareFreeInvocationReceipt
	_, body := s.factorSquareFreeWithReceipt(mustUPolyView(t, m, p), fs, 1, DefaultUPolynomialFactorParams(), &receipt)
	if !tags.ShouldTrace("polynomial::factorization::bughunt") {
		serviceMust(t, body)
		return
	}
	c := s.scope
	if !errors.Is(body, arena.ErrSlotLimit) || !w.armed || c == nil || c.constructed != 9 || !c.tracePending() {
		t.Fatal("recombination display child/frontier", body, w.armed, c)
	}
	if !errors.Is(s.factorSquareFreeInvocationSourceCause(&receipt), ErrUPolynomialCleanupPending) {
		t.Fatal("enclosing source caller lost exact display blocker", s.factorSquareFreeInvocationSourceCause(&receipt), s.matchesSquareFreeHistory(&receipt), c.traceSourceBlocker())
	}
	if c.hensel.quadraticConstructed != 5 || c.hensel.iterationConstructed != 0 || c.hensel.related == nil {
		t.Fatal("quadratic trace lost older locals or declared younger iteration")
	}
	if c.zp == nil || !c.hensel.active || c.hensel.outerConstructed != 4 || c.fpp.size != 3 || fs.count != 0 || c.lifted.count != 0 || !m.displayReceiptPending(&c.tracePolynomial) {
		t.Fatal("older Hensel vectors lost or factor prematurely published")
	}
	extras := upolyPhysicalExtraOwners{Services: []*UPolynomialFactorizationService{s}, RationalManagers: []*unumeral.RationalManager{q}}
	for i := 0; i < w.count; i++ {
		extras.Integers = append(extras.Integers, &w.held[i])
	}
	upolyHenselAuditPhysical(t, []*UPolynomialManager{m}, []*UPolynomialFactors{fs}, extras)
	serviceMust(t, s.CleanupRetained())
	if s.scope != nil {
		t.Fatal("explicit trace cleanup retained source scope")
	}
}
