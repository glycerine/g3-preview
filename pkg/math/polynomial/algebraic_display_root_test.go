package polynomial

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicRootDisplayOwnsActualPolynomialDisplay(t *testing.T) {
	for _, smt := range []bool{false, true} {
		t.Run(map[bool]string{false: "plain", true: "smt2"}[smt], func(t *testing.T) {
			c, up, _ := algebraicCompareFixtureProfile(t, true)
			q := unumeral.NewRational()
			serviceMust(t, c.config.Rationals.SetString(&q, "18446744073709551619"))
			var a Algebraic
			serviceMust(t, c.SetRational(&a, &q))
			vector, e := c.NewVector()
			serviceMust(t, e)
			serviceMust(t, c.ensureIsolation())
			serviceMust(t, c.roots.Resize(1))
			var ref unumeral.NumericCellRef
			var original unumeral.NumericCell
			z := c.config.Rationals.Integers()
			sink := &upolyDisplaySink{numeralHook: func(string) error {
				f := up.currentDisplay
				if ref == 0 && f != nil && f.constructed && f.numeral.Cell != 0 {
					ref = f.numeral.Cell
					original = z.Cell(ref)
					guard := original
					guard.Backing.Requested = uint64(^uint32(0)) + 1
					z.Store(ref, guard)
				}
				return nil
			}}
			var body error
			if smt {
				body = c.DisplayRootSMT2Into(sink, &a)
			} else {
				body = c.DisplayRootInto(sink, &a)
			}
			s := &c.rootDisplay
			if ref == 0 || !errors.Is(body, arena.ErrBounds) || !s.active || s.coefficients.slot == 0 || !s.negativeOwned || !up.displayReceiptPending(&s.display) {
				t.Fatal("actual polynomial display did not protect older root-display locals", body, ref)
			}
			frame, stop := s.display.frame, s.sourceStop
			if stop == nil || stop != frame.sourceCause || !errors.Is(body, stop) {
				t.Fatal("root display did not retain exact child source stop", body)
			}
			if e = vector.Close(); !errors.Is(e, ErrAlgebraicPending) || vector.closing || vector.closed {
				t.Fatal("pending root display allowed older vector header mutation", e)
			}
			if e = c.CleanupIsolation(); !errors.Is(e, ErrAlgebraicPending) || c.roots.state.size != 1 || c.roots.state.dropPending {
				t.Fatal("pending root display changed older isolation scratch", e)
			}
			// Qualified requested-size guard on the actual formatter NUM.
			z.Store(ref, original)
			serviceMust(t, c.CleanupRetained())
			if s.active || frame.active || frame.sourceCause != stop || s.sourceStop != stop || s.coefficients.slot != 0 || s.negativeOwned {
				t.Fatal("owning display cleanup lost source-stop history or locals")
			}
			serviceMust(t, vector.Close())
			serviceMust(t, c.Del(&a))
			serviceMust(t, c.config.Rationals.Del(&q))
		})
	}
}

type algebraicRootTextSink struct {
	bytes.Buffer
	calls, fail int
	marker      error
}

func (s *algebraicRootTextSink) put(v string) error {
	s.calls++
	if s.fail == s.calls {
		return s.marker
	}
	_, e := s.WriteString(v)
	return e
}
func (s *algebraicRootTextSink) AppendCString(v string) error     { return s.put(v) }
func (s *algebraicRootTextSink) AppendNumeralText(v string) error { return s.put(v) }
func (s *algebraicRootTextSink) AppendInt32(v int32) error {
	return s.put(strconv.FormatInt(int64(v), 10))
}

func TestAlgebraicRootDisplayLiteralAndCachedIndex(t *testing.T) {
	c, _, control := algebraicControlledFixture(t)
	var a Algebraic
	q := unumeral.NewRational()
	defer func() { serviceMust(t, c.Del(&a)); serviceMust(t, c.config.Rationals.Del(&q)) }()
	for _, tc := range []struct {
		n          int32
		plain, smt string
	}{
		{0, "(#, 1)", "(root-obj x 1)"},
		{2, "(3 # - 2, 1)", "(root-obj (+ (* 3 x) (- 2)) 1)"},
		{-2, "(3 # + 2, 1)", "(root-obj (+ (* 3 x) 2) 1)"},
	} {
		serviceMust(t, c.config.Rationals.SetFraction(&q, int64(tc.n), 3))
		serviceMust(t, c.SetRational(&a, &q))
		plain, smt := &algebraicRootTextSink{}, &algebraicRootTextSink{}
		serviceMust(t, c.DisplayRootInto(plain, &a))
		serviceMust(t, c.DisplayRootSMT2Into(smt, &a))
		if plain.String() != tc.plain || smt.String() != tc.smt {
			t.Fatalf("literal root %d: %q / %q", tc.n, plain.String(), smt.String())
		}
	}
	serviceMust(t, c.Del(&a))
	a = algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	plain := &algebraicRootTextSink{}
	serviceMust(t, c.DisplayRootInto(plain, &a))
	if plain.String() != "(#^2 - 2, 2)" {
		t.Fatal(plain.String())
	}
	l, e := c.location(&a)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	if r.flags>>3 != 2 {
		t.Fatal("root index not cached", r.flags)
	}
	// A warm display skips the Sturm root-ID engine and its checkpoints.
	before := control.visits
	smt := &algebraicRootTextSink{}
	serviceMust(t, c.DisplayRootSMT2Into(smt, &a))
	if smt.String() != "(root-obj (+ (^ x 2) (- 2)) 2)" {
		t.Fatal(smt.String())
	}
	if control.visits != before {
		t.Fatal("cached root index recomputed")
	}
}

func TestAlgebraicRootDisplayCachePublicationAtOutputBoundary(t *testing.T) {
	for _, afterPolynomial := range []bool{false, true} {
		c, _, _ := algebraicControlledFixture(t)
		a := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, -2, 0, -1, 0, true)
		marker := errors.New("display callback stop")
		sink := &upolyDisplaySink{literalHook: func(v string) error {
			if !afterPolynomial && v == "#" || afterPolynomial && v == ", " {
				return marker
			}
			return nil
		}}
		body := c.DisplayRootInto(sink, &a)
		if !errors.Is(body, marker) {
			t.Fatal(body)
		}
		l, e := c.location(&a)
		serviceMust(t, e)
		r, e := c.load(l)
		serviceMust(t, e)
		want := uint32(0)
		if afterPolynomial {
			want = 1
		}
		if r.flags>>3 != want {
			t.Fatal("root cache publication timing", r.flags>>3, want)
		}
		serviceMust(t, c.CleanupRetained())
		serviceMust(t, c.Del(&a))
	}
}

func TestAlgebraicRootDisplayEveryInsertionFailure(t *testing.T) {
	marker := errors.New("root stream stopped")
	for _, basic := range []bool{false, true} {
		for _, smt := range []bool{false, true} {
			success := false
			for fail := 1; fail < 64 && !success; fail++ {
				c, p, _ := algebraicControlledFixture(t)
				a, b := algebraicResultantInputs(t, c, p)
				q := unumeral.NewRational()
				if basic {
					serviceMust(t, c.config.Rationals.SetFraction(&q, 2, 3))
					serviceMust(t, c.SetRational(&a, &q))
				}
				sink := &algebraicRootTextSink{fail: fail, marker: marker}
				call := func(s *algebraicRootTextSink) error {
					if smt {
						return c.DisplayRootSMT2Into(s, &a)
					}
					return c.DisplayRootInto(s, &a)
				}
				body := call(sink)
				if body == nil {
					success = true
				} else if !errors.Is(body, marker) {
					t.Fatal(body)
				}
				serviceMust(t, c.CleanupRetained())
				retry := &algebraicRootTextSink{}
				serviceMust(t, call(retry))
				if !strings.HasPrefix(retry.String(), sink.String()) {
					t.Fatal("insertion failure corrupted prefix", sink.String(), retry.String())
				}
				if c.rootDisplay.active {
					t.Fatal("display locals survived")
				}
				serviceMust(t, c.Del(&b))
				serviceMust(t, c.Del(&a))
				serviceMust(t, c.config.Rationals.Del(&q))
			}
			if !success {
				t.Fatal("never reached complete display")
			}
		}
	}
}

func TestAlgebraicRootDisplayBigBasicCoefficientCopies(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetFraction(&q, -8589934593, 5))
	var a Algebraic
	serviceMust(t, c.SetRational(&a, &q))
	plain, smt := &algebraicRootTextSink{}, &algebraicRootTextSink{}
	serviceMust(t, c.DisplayRootInto(plain, &a))
	serviceMust(t, c.DisplayRootSMT2Into(smt, &a))
	if plain.String() != "(5 # + 8589934593, 1)" || smt.String() != "(root-obj (+ (* 5 x) 8589934593) 1)" {
		t.Fatal(plain.String(), smt.String())
	}
	if c.rootDisplay.active || c.rootDisplay.coefficients.slot != 0 {
		t.Fatal("coefficient copies retained")
	}
	serviceMust(t, c.Del(&a))
	serviceMust(t, c.config.Rationals.Del(&q))
}
