package polynomial

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

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
