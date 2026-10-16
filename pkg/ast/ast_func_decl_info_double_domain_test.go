package ast

import (
	"errors"
	"math"
	"strings"
	"testing"
)

// parameter::hash converts PARAM_DOUBLE to unsigned before a declaration can
// reach the extension-model metadata printer. Its non-finite/out-of-range
// conversion is undefined in C++, and Go rejects that registration domain.
func TestASTFuncDeclInfoDoubleRegistrationAndSourceTextDomain(t *testing.T) {
	m := astTestManager(t)
	bs, err := m.MkBoolSort()
	astTestMust(t, err)
	for _, tc := range []struct {
		name  string
		value float64
	}{
		{"positive-infinity", math.Inf(1)},
		{"negative-infinity", math.Inf(-1)},
		{"positive-nan", math.NaN()},
		{"negative-nan", math.Copysign(math.NaN(), -1)},
		{"negative-one", -1},
		{"unsigned-overflow", 4294967296},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, e := m.NewParameterDouble(tc.value)
			astTestMust(t, e)
			info, e := m.NewFuncDeclInfo(NullFamilyID, 19, []ASTParameter{p}, ASTFuncFlags{}, false)
			astTestMust(t, e)
			d, e := m.MkFuncDeclWithInfo(astTestSymbol(t, m, tc.name), nil, bs, info)
			if !d.IsNull() || !errors.Is(e, ErrASTDomain) {
				t.Fatal("undefined source double hash accepted in declaration", tc.value, d, e)
			}
			astTestMust(t, m.DelFuncDeclInfo(&info))
			astTestMust(t, m.DelParameter(&p))
		})
	}
	for _, tc := range []struct {
		name, want string
		value      float64
	}{
		{"negative-fraction", "-0.5", -0.5},
		{"negative-zero", "-0", math.Copysign(0, -1)},
		{"exponent-positive", "1e+06", 1_000_000},
		{"exponent-negative", "1e-05", 0.00001},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, e := m.NewParameterDouble(tc.value)
			astTestMust(t, e)
			info, e := m.NewFuncDeclInfo(NullFamilyID, 19, []ASTParameter{p}, ASTFuncFlags{}, false)
			astTestMust(t, e)
			d, e := m.MkFuncDeclWithInfo(astTestSymbol(t, m, tc.name), nil, bs, info)
			astTestMust(t, e)
			body, present, e := m.FuncDeclInfoText(d)
			astTestMust(t, e)
			if !present || !strings.Contains(body, ":parameters ("+tc.want+")") {
				t.Fatal("defined source double ostream metadata", present, body, tc.want)
			}
			astTestMust(t, m.DelFuncDeclInfo(&info))
			astTestMust(t, m.DelParameter(&p))
		})
	}
}
