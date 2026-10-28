package ast

// Source: src/ast/bv_decl_plugin.cpp:753-765.

import "testing"

func TestASTBitVectorSignedNormalizationDistinctReturnAndCopy(t *testing.T) {
	m := bvOffsetRuntimeTestManager(t)
	_, e := m.RegisterArithmeticPlugin()
	astTestMust(t, e)
	_, e = m.RegisterBitVectorPlugin()
	astTestMust(t, e)
	for _, tc := range []struct{ input, want string }{
		{"18446744073709551615", "18446744073709551615"},
		{"18446744073709551616", "-18446744073709551616"},
		{"36893488147419103231", "-1"},
		{"-36893488147419103233", "-1"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			v, e := m.numerals.NewValue()
			astTestMust(t, e)
			defer func() { astTestMust(t, m.numerals.Del(&v)) }()
			astTestMust(t, m.numerals.SetString(&v, tc.input))
			var call ASTBitVectorNumeralInvocation
			returned, e := m.NormalizeSignedBitVectorValueWithInvocation(&v, 65, &call)
			astTestMust(t, e)
			defer func() { astTestMust(t, m.numerals.Del(&returned)) }()
			original, e := m.numerals.String(&v)
			astTestMust(t, e)
			signed, e := m.numerals.String(&returned)
			astTestMust(t, e)
			if original != tc.input || signed != tc.want {
				t.Fatal(original, signed, tc)
			}
			astTestMust(t, m.BitVectorNumeralInvocationSourceReady(&call))
			astTestMust(t, m.CleanupBitVectorNumeralInvocation(&call))
			// The return remains owned and readable after exact child retirement.
			signed, e = m.numerals.String(&returned)
			astTestMust(t, e)
			if signed != tc.want {
				t.Fatal(signed)
			}
			astTestMust(t, m.numerals.Copy(&v, &returned))
			original, e = m.numerals.String(&v)
			astTestMust(t, e)
			if original != tc.want {
				t.Fatal("physical copy", original)
			}
		})
	}
}
