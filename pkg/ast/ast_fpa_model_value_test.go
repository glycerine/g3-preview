package ast

import "testing"

func fpaModelTestManager(t *testing.T) *ASTManager {
	t.Helper()
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	_, err = m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	return m
}

func TestASTFPAModelPackedAndComponentsExhaustive(t *testing.T) {
	m := fpaModelTestManager(t)
	// Every encoding in a seven-bit format, including every NaN payload and
	// both signs. Canonical identity checks the actual numeral registration;
	// IEEE re-encoding independently checks the finite components and bias.
	for bits := uint64(0); bits < 128; bits++ {
		packed, err := m.BitVectorNumeralUint64(bits, 7)
		astTestMust(t, err)
		value, err := m.MkFPAModelValue(3, 4, []ASTHandle{packed})
		astTestMust(t, err)
		held, err := m.NewRef(value)
		astTestMust(t, err)
		fields := make([]ASTHandle, 3)
		for i, n := range []uint64{bits >> 6, (bits >> 3) & 7, bits & 7} {
			fields[i], err = m.BitVectorNumeralUint64(n, []uint32{1, 3, 3}[i])
			astTestMust(t, err)
		}
		other, err := m.MkFPAModelValue(3, 4, fields)
		astTestMust(t, err)
		if value != other {
			t.Fatalf("packed/components differ for %07b", bits)
		}
		kind, err := m.GetDeclKind(value)
		astTestMust(t, err)
		if (bits>>3)&7 == 7 && bits&7 != 0 {
			if kind != FPAOpNaN {
				t.Fatalf("NaN payload not canonical: %07b", bits)
			}
		} else {
			encoded, found, _, err := m.RewriteFPAToIEEEBVNumeral(value, false)
			astTestMust(t, err)
			if !found || encoded != packed {
				t.Fatalf("IEEE roundtrip %07b: found=%v", bits, found)
			}
		}
		astTestMust(t, m.DelRef(&held))
	}
}

func TestASTFPAModelRoundingModeCodes(t *testing.T) {
	m := fpaModelTestManager(t)
	family, err := m.RegisterFloatingPointPlugin()
	astTestMust(t, err)
	kinds := []int32{FPAOpRMNearestEven, FPAOpRMNearestAway, FPAOpRMTowardPositive, FPAOpRMTowardNegative, FPAOpRMTowardZero, FPAOpRMTowardZero, FPAOpRMTowardZero, FPAOpRMTowardZero}
	for code, kind := range kinds {
		bits, err := m.BitVectorNumeralUint64(uint64(code), 3)
		astTestMust(t, err)
		value, err := m.MkFPARoundingModeModelValue([]ASTHandle{bits})
		astTestMust(t, err)
		held, err := m.NewRef(value)
		astTestMust(t, err)
		expected, err := m.MkFamilyApp(family, kind, nil, nil)
		astTestMust(t, err)
		if value != expected {
			t.Fatalf("RM code %d did not select %d", code, kind)
		}
		astTestMust(t, m.DelRef(&held))
	}
}

func TestASTFPAModelWideBiasedExponent(t *testing.T) {
	m := fpaModelTestManager(t)
	// A large exponent exercises the owned multi-limb bias and subtraction.
	// Ebits 64 is outside the signed top-exponent helper domain (2^63),
	// so this boundary uses the largest representable top exponent.
	sign, err := m.BitVectorNumeralUint64(0, 1)
	astTestMust(t, err)
	exponent, err := m.BitVectorNumeralUint64(uint64(1)<<62, 63)
	astTestMust(t, err)
	sig, err := m.BitVectorNumeralUint64(1, 3)
	astTestMust(t, err)
	value, err := m.MkFPAModelValue(63, 4, []ASTHandle{sign, exponent, sig})
	astTestMust(t, err)
	astTestPin(t, m, value)
	kind, err := m.GetDeclKind(value)
	astTestMust(t, err)
	if kind != FPAOpNumeral {
		t.Fatal("wide exponent did not produce numeral", kind)
	}
}
