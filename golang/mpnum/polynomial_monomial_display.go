// Source: src/math/polynomial/polynomial.cpp; src/math/polynomial/polynomial.h. Licensed under the MIT license.
package mpnum

import "github.com/glycerine/g3/golang/arena"

// MonomialTextSink is a borrowed cold output transport, never stored in a hot cell.
type MonomialTextSink interface {
	WriteLiteral(string) error
	WriteUnsigned(uint32) error
	WriteVariable(uint32) error
}

func (m *MonomialManager) DisplayInto(a Monomial, out MonomialTextSink, star bool) error {
	if e := m.check(); e != nil {
		return e
	}
	h, e := m.headerCore(a)
	if e != nil {
		return e
	}
	if out == nil {
		return arena.ErrConfig
	}
	if h.Size == 0 {
		return out.WriteLiteral("1")
	}
	for i := uint32(0); i < h.Size; i++ {
		if i > 0 {
			s := " "
			if star {
				s = "*"
			}
			if e = out.WriteLiteral(s); e != nil {
				return e
			}
		}
		p, e := m.powerCore(a, i)
		if e != nil {
			return e
		}
		if e = out.WriteVariable(p.Var); e != nil {
			return e
		}
		if p.Degree > 1 {
			if e = out.WriteLiteral("^"); e != nil {
				return e
			}
			if e = out.WriteUnsigned(p.Degree); e != nil {
				return e
			}
		}
	}
	return nil
}
func (m *MonomialManager) DisplaySMT2Into(a Monomial, out MonomialTextSink) error {
	if e := m.check(); e != nil {
		return e
	}
	h, e := m.headerCore(a)
	if e != nil {
		return e
	}
	if out == nil {
		return arena.ErrConfig
	}
	if h.Size == 0 {
		if e = out.WriteLiteral("1"); e != nil {
			return e
		}
	} else if h.Size == 1 {
		p, e := m.powerCore(a, 0)
		if e != nil {
			return e
		}
		if p.Degree == 1 {
			if e = out.WriteVariable(p.Var); e != nil {
				return e
			}
		} else {
			if e = m.displaySMT2Product(a, h.Size, out); e != nil {
				return e
			}
		}
	} else {
		if e = m.displaySMT2Product(a, h.Size, out); e != nil {
			return e
		}
	}
	return out.WriteLiteral(")")
}
func (m *MonomialManager) displaySMT2Product(a Monomial, n uint32, out MonomialTextSink) error {
	if e := out.WriteLiteral("(*"); e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		p, e := m.powerCore(a, i)
		if e != nil {
			return e
		}
		if modularSourceDebug && p.Degree == 0 {
			return ErrMonomialAssertion
		}
		for j := uint32(0); j < p.Degree; j++ {
			if e = out.WriteLiteral(" "); e != nil {
				return e
			}
			if e = out.WriteVariable(p.Var); e != nil {
				return e
			}
		}
	}
	return nil
}
