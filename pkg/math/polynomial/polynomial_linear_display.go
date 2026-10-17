// Source: src/math/polynomial/linear_eq_solver.h:140-150;
// src/util/mpq.cpp display; src/util/mpzzp.h display.
package polynomial

import (
	"io"

	"github.com/glycerine/g3/pkg/arena"
)

// io.Writer is a cold host-output adapter. Read and format each actual source
// field at its reached position; no matrix snapshot or numerical mirror.
func linearWriteText(out io.Writer, text string) error {
	if out == nil {
		return arena.ErrConfig
	}
	n, e := io.WriteString(out, text)
	if e != nil {
		return e
	}
	if n != len(text) {
		return io.ErrShortWrite
	}
	return nil
}
func (s *RationalLinearSolver) Display(out io.Writer) error {
	if s != nil && s.self == s && s.running {
		return ErrPolynomialPending
	}
	if e := s.open(); e != nil {
		return e
	}
	s.running = true
	defer func() { s.running = false }()
	return s.displaySource(out)
}
func (s *RationalLinearSolver) displaySource(out io.Writer) error {
	if e := s.open(); e != nil {
		return e
	}
	size := uint32(len(s.A))
	for i := uint32(0); i < size; i++ {
		row, e := s.row(i)
		if e != nil {
			return e
		}
		if polynomialSourceDebug {
			size := uint32(len(*row))
			if e := linearSourceAssert(size == s.n, 141, "A[i].size() == n"); e != nil {
				return e
			}
		}
		for j := uint32(0); j < s.n; j++ {
			a, e := s.load(row, j)
			if e != nil {
				return e
			}
			text, e := s.manager.String(&a)
			if e != nil {
				return e
			}
			if e = linearWriteText(out, text); e != nil {
				return e
			}
			if e = linearWriteText(out, " "); e != nil {
				return e
			}
		}
		b, e := s.load(&s.b, i)
		if e != nil {
			return e
		}
		text, e := s.manager.String(&b)
		if e != nil {
			return e
		}
		if e = linearWriteText(out, text); e != nil {
			return e
		}
		if e = linearWriteText(out, "\n"); e != nil {
			return e
		}
	}
	return nil
}
func (s *ModularLinearSolver) Display(out io.Writer) error {
	if s != nil && s.self == s && s.running {
		return ErrPolynomialPending
	}
	if e := s.open(); e != nil {
		return e
	}
	s.running = true
	defer func() { s.running = false }()
	return s.displaySource(out)
}
func (s *ModularLinearSolver) displaySource(out io.Writer) error {
	if e := s.open(); e != nil {
		return e
	}
	size := uint32(len(s.A))
	for i := uint32(0); i < size; i++ {
		row, e := s.row(i)
		if e != nil {
			return e
		}
		if polynomialSourceDebug {
			size := uint32(len(*row))
			if e := linearSourceAssert(size == s.n, 141, "A[i].size() == n"); e != nil {
				return e
			}
		}
		for j := uint32(0); j < s.n; j++ {
			a, e := s.load(row, j)
			if e != nil {
				return e
			}
			// mpzzp.h display calls the integer manager directly, without the
			// normalization assertion of the separate modular String API.
			text, e := s.manager.Integers().String(&a)
			if e != nil {
				return e
			}
			if e = linearWriteText(out, text); e != nil {
				return e
			}
			if e = linearWriteText(out, " "); e != nil {
				return e
			}
		}
		b, e := s.load(&s.b, i)
		if e != nil {
			return e
		}
		text, e := s.manager.Integers().String(&b)
		if e != nil {
			return e
		}
		if e = linearWriteText(out, text); e != nil {
			return e
		}
		if e = linearWriteText(out, "\n"); e != nil {
			return e
		}
	}
	return nil
}
