//go:build g3debug

// Copyright (c) 2006 Microsoft Corporation
// Source: src/util/mpz.cpp extended gcd Z3DEBUG diagnostic.
package mpnum

const sourceExtendedGCDDebug = true

// Keep the source action order and reread the actual inputs, including aliases.
// The caller owns the existing nexta/nextb locals and cleans them on every exit.
func (m *IntegerManager) extendedGCDDiagnostic(r1, r2, a, b, expected, nexta, nextb *Int) error {
	if e := m.Mul(a, r1, nexta); e != nil {
		return e
	}
	if e := m.Mul(b, r2, nextb); e != nil {
		return e
	}
	if e := m.Add(nexta, nextb, nexta); e != nil {
		return e
	}
	comparison, e := m.Compare(nexta, expected)
	if e != nil {
		return e
	}
	if comparison != 0 {
		return errExtendedGCDInvariant
	}
	return nil
}
