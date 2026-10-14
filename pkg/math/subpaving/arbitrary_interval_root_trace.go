// Source: src/math/interval/interval_def.h:1427,1459,1470;
// concrete config_mpf instantiation, src/util/trace.h.
package subpaving

import (
	"errors"
	"io"

	"github.com/glycerine/g3/pkg/math/interval"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// Source ostream insertion failures are sticky but do not stop later operand
// computations. Numeric formatting failures still unwind the TRACE body.
type arbitraryRootTraceStream struct {
	w         io.Writer
	streamErr error
}

func (s *arbitraryRootTraceStream) Write(p []byte) (int, error) {
	n, err := s.w.Write(p)
	if err != nil && s.streamErr == nil {
		s.streamErr = err
	}
	return n, err
}

func (s *arbitraryRootTraceStream) operand(err error) error {
	if err != nil && (s.streamErr == nil || !errors.Is(err, s.streamErr)) {
		return err
	}
	return nil
}

func arbitraryIntervalTraceTags(roots interval.ArbitraryRoot) *diagnostic.TraceTags {
	if roots.TraceTags != nil {
		return roots.TraceTags
	}
	return diagnostic.BorrowProcessTraceTags()
}

func arbitraryIntervalRootRecord(roots interval.ArbitraryRoot, tag, function string, line int, body func(*arbitraryRootTraceStream) error) error {
	tags := arbitraryIntervalTraceTags(roots)
	if !tags.ShouldTrace(tag) {
		return nil
	}
	var computationErr error
	_ = tags.EmitConditional(tag, function, "src/math/interval/interval_def.h", line,
		func() bool { return true }, func(w io.Writer) error {
			computationErr = body(&arbitraryRootTraceStream{w: w})
			return computationErr
		})
	return computationErr
}

func (v *ArbitraryInterval) traceNthRoot(src *ArbitraryInterval, roots interval.ArbitraryRoot) error {
	return arbitraryIntervalRootRecord(roots, "interval_nth_root", "nth_root", 1427,
		func(s *arbitraryRootTraceStream) error {
			if err := s.operand(src.Display(s)); err != nil {
				return err
			}
			_, _ = io.WriteString(s, " --> ")
			if err := s.operand(v.Display(s)); err != nil {
				return err
			}
			_, _ = io.WriteString(s, "\n")
			return nil
		})
}

func (v *ArbitraryInterval) traceXnEqYBounds(src *ArbitraryInterval, lo, hi *numeral.ArbitraryFloat, roots interval.ArbitraryRoot) error {
	return arbitraryIntervalRootRecord(roots, "interval_xn_eq_y", "xn_eq_y", 1459,
		func(s *arbitraryRootTraceStream) error {
			_, _ = io.WriteString(s, "x^n = ")
			if err := s.operand(src.Display(s)); err != nil {
				return err
			}
			_, _ = io.WriteString(s, "\nsqrt(y) in ")
			if err := s.operand(roots.Numbers.DisplayTo(s, lo)); err != nil {
				return err
			}
			_, _ = io.WriteString(s, " ")
			if err := s.operand(roots.Numbers.DisplayTo(s, hi)); err != nil {
				return err
			}
			_, _ = io.WriteString(s, "\n")
			return nil
		})
}

func (v *ArbitraryInterval) traceXnEqYResult(roots interval.ArbitraryRoot) error {
	return arbitraryIntervalRootRecord(roots, "interval_xn_eq_y", "xn_eq_y", 1470,
		func(s *arbitraryRootTraceStream) error {
			_, _ = io.WriteString(s, "interval for x: ")
			if err := s.operand(v.Display(s)); err != nil {
				return err
			}
			_, _ = io.WriteString(s, "\n")
			return nil
		})
}
