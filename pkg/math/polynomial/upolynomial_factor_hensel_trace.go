package polynomial

import (
	"io"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// Source: factorization.cpp:906-908,921-923,927-929,935-939. All
// polynomial insertions use the source integer core, including modular vectors.
func (s *UPolynomialFactorizationService) traceHenselVectors(caller *upolyHenselCallerScope, line int) error {
	tags := s.manager.modular.BorrowTraceTags()
	tag := "polynomial::factorization::bughunt"
	if line == 935 {
		tag = "polynomial::factorization"
	}
	if !tags.ShouldTrace(tag) {
		return nil
	}
	if caller == nil || !caller.matches() {
		return ErrUPolynomialOwner
	}
	c := caller.owner
	if e := c.traceBorrow.bindManagers([3]*UPolynomialManager{s.manager, caller.prime, caller.lift}, tags); e != nil {
		return e
	}
	defer c.traceBorrow.unbind()
	var computation error
	_ = tags.EmitConditional(tag, "hensel_lift", "src/math/polynomial/upolynomial_factorization.cpp", line,
		func() bool { return true }, func(raw io.Writer) error {
			out := factorizationTraceWriter{raw}
			if line == 935 {
				_ = linearWriteText(out, "lifted to ")
				p, e := caller.lift.modular.BorrowedModulus()
				if e != nil {
					computation = e
					return e
				}
				prime, e := s.manager.modular.String(p)
				if e != nil {
					computation = e
					return e
				}
				_ = linearWriteText(out, prime)
				_ = diagnostic.WriteTraceEndl(out)
			}
			index, label := 0, "A = "
			if line == 921 {
				index, label = 2, "C = "
			} else if line == 927 {
				index, label = 1, "B = "
			}
			_ = linearWriteText(out, label)
			if e := s.manager.displayIntoWithReceipt(s.manager.stateView(&caller.outer[index]), factorizationTraceStream{out}, "x", false, &c.tracePolynomial); e != nil {
				computation = e
				return e
			}
			_ = diagnostic.WriteTraceEndl(out)
			if line == 935 {
				_ = linearWriteText(out, "B = ")
				if e := s.manager.displayIntoWithReceipt(s.manager.stateView(&caller.outer[1]), factorizationTraceStream{out}, "x", false, &c.tracePolynomial); e != nil {
					computation = e
					return e
				}
				_ = diagnostic.WriteTraceEndl(out)
			}
			return nil
		})
	return computation
}
