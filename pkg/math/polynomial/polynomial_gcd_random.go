// Source: src/math/polynomial/polynomial.cpp:4053-4068.
package polynomial

import (
	"github.com/glycerine/g3/pkg/clibrand"
	"github.com/glycerine/g3/pkg/util"
)

// The source uses the one process C rand stream. Never reseed it here or bind
// an independent manager-local generator. vals remains a live borrowed vector;
// only its size is captured before the source rejection loop.
func (m *MultivariatePolynomialManager) peekFresh(vals PolynomialNewtonInputs, p uint32, out *util.Int) error {
	if e := m.ready(); e != nil {
		return e
	}
	if polynomialSourceDebug {
		n, e := vals.Size()
		if e != nil {
			return e
		}
		if n >= p {
			return ErrPolynomialAssertion
		}
	}
	size, e := vals.Size()
	if e != nil {
		return e
	}
	for {
		draw := clibrand.Rand()
		// The source uint32 modulo by zero is undefined. The checked Go
		// boundary is reached after the actual draw, before any numeral SET.
		if p == 0 {
			return util.ErrDivideByZero
		}
		if e = m.core.modular.SetUint32(out, draw%p); e != nil {
			return e
		}
		k := uint32(0)
		for ; k < size; k++ {
			a, e := vals.Get(k)
			if e != nil {
				return e
			}
			equal, e := m.core.modular.Equal(&a, out)
			if e != nil {
				return e
			}
			if equal {
				break
			}
		}
		if k == size {
			return nil
		}
	}
}
