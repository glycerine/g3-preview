// Source: src/math/polynomial/linear_eq_solver.h:140-150;
// src/util/mpq.cpp display; src/util/mpzzp.h display.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2012.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"io"
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
	if e := s.open(); e != nil {
		return e
	}
	size, e := s.A.size(s.runtime)
	if e != nil {
		return e
	}
	for i := uint32(0); i < size; i++ {
		row, e := s.row(i)
		if e != nil {
			return e
		}
		if modularSourceDebug {
			size, e := row.size(s.runtime)
			if e != nil {
				return e
			}
			if size != s.n {
				return ErrPolynomialAssertion
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
	if e := s.open(); e != nil {
		return e
	}
	size, e := s.A.size(s.runtime)
	if e != nil {
		return e
	}
	for i := uint32(0); i < size; i++ {
		row, e := s.row(i)
		if e != nil {
			return e
		}
		if modularSourceDebug {
			size, e := row.size(s.runtime)
			if e != nil {
				return e
			}
			if size != s.n {
				return ErrPolynomialAssertion
			}
		}
		for j := uint32(0); j < s.n; j++ {
			a, e := s.load(row, j)
			if e != nil {
				return e
			}
			// mpzzp.h display calls the integer manager directly, without the
			// normalization assertion of the separate modular String API.
			text, e := s.manager.manager.String(&a)
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
		text, e := s.manager.manager.String(&b)
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
