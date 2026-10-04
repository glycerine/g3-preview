//go:build g3debug

// Licensed under the MIT license.
// Source: src/util/mpz.cpp selected Lehmer and ExtendedGCD SASSERT sites.
package mpnum

const sourceGCDDebug = true

func gcdSourceRequire(condition bool) error {
	if !condition {
		return ErrIntegerAssertion
	}
	return nil
}

// Actual source ge expression, rather than an arithmetic postcondition oracle.
func (m *IntegerManager) gcdSourceGreaterEqual(a, b *Int) error {
	cmp, e := m.Compare(a, b)
	if e != nil {
		return e
	}
	return gcdSourceRequire(cmp >= 0)
}
