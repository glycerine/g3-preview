package ast

import (
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func pbTestRational(t *testing.T, m *ASTManager, text string) unumeral.OwnedRational {
	t.Helper()
	v, e := m.numerals.NewValue()
	astTestMust(t, e)
	astTestMust(t, m.numerals.SetString(&v, text))
	return v
}

func pbTestParameters(t *testing.T, m *ASTManager, app ASTHandle) (int32, []int64) {
	t.Helper()
	a, e := m.requireKind(app, ASTApp)
	astTestMust(t, e)
	d := m.appAt(a.slot).decl
	_, kind, e := m.declIdentity(d)
	astTestMust(t, e)
	info, e := m.funcInfo(m.declAt(d.slot).info)
	astTestMust(t, e)
	values := make([]int64, info.count)
	for i := range values {
		p, e := m.infoParameter(info, uint32(i))
		astTestMust(t, e)
		r, e := m.parameter(p)
		astTestMust(t, e)
		switch r.Kind {
		case ASTParameterInt:
			values[i] = int64(r.integer)
		case ASTParameterRational:
			values[i], e = m.numerals.Int64(&r.Rational)
			astTestMust(t, e)
		default:
			t.Fatal("unexpected PB parameter", r.Kind)
		}
	}
	return kind, values
}

func TestASTPseudoBooleanOriginalNormalizationFloorCeilAndEquality(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterPseudoBooleanPlugin()
	astTestMust(t, e)
	truth, e := m.MkTrue()
	astTestMust(t, e)
	falsehood, e := m.MkFalse()
	astTestMust(t, e)
	coefficients := []unumeral.OwnedRational{pbTestRational(t, m, "1/2"), pbTestRational(t, m, "3/4")}
	defer func() {
		for i := range coefficients {
			astTestMust(t, m.numerals.Del(&coefficients[i]))
		}
	}()
	bound := pbTestRational(t, m, "5/6")
	defer func() { astTestMust(t, m.numerals.Del(&bound)) }()
	args := []ASTHandle{truth, falsehood}
	le, e := m.MkPseudoBooleanLE(coefficients, args, &bound)
	astTestMust(t, e)
	astTestPin(t, m, le)
	ge, e := m.MkPseudoBooleanGE(coefficients, args, &bound)
	astTestMust(t, e)
	astTestPin(t, m, ge)
	if kind, values := pbTestParameters(t, m, le); kind != ASTPBOpLE || len(values) != 3 || values[0] != 3 || values[1] != 2 || values[2] != 3 {
		t.Fatal("PB <= normalization", kind, values)
	}
	if kind, values := pbTestParameters(t, m, ge); kind != ASTPBOpGE || values[0] != 4 || values[1] != 2 || values[2] != 3 {
		t.Fatal("PB >= normalization", kind, values)
	}
	eq, e := m.MkPseudoBooleanEQ(coefficients, args, &bound)
	astTestMust(t, e)
	if eq != falsehood {
		t.Fatal("noninteger normalized PB equality", eq, falsehood)
	}
}

func TestASTPseudoBooleanOriginalCardinalityAndStrictLess(t *testing.T) {
	m := astTestManager(t)
	_, e := m.RegisterPseudoBooleanPlugin()
	astTestMust(t, e)
	a, e := m.MkTrue()
	astTestMust(t, e)
	b, e := m.MkFalse()
	astTestMust(t, e)
	cardinality, e := m.MkPseudoBooleanAtMost([]ASTHandle{a, b}, 1)
	astTestMust(t, e)
	astTestPin(t, m, cardinality)
	if kind, values := pbTestParameters(t, m, cardinality); kind != ASTPBOpAtMost || len(values) != 1 || values[0] != 1 {
		t.Fatal("PB cardinality", kind, values)
	}
	coefficients := []unumeral.OwnedRational{pbTestRational(t, m, "1/2"), pbTestRational(t, m, "3/4")}
	defer func() {
		for i := range coefficients {
			astTestMust(t, m.numerals.Del(&coefficients[i]))
		}
	}()
	bound := pbTestRational(t, m, "5/6")
	defer func() { astTestMust(t, m.numerals.Del(&bound)) }()
	strict, e := m.MkPseudoBooleanLT(coefficients, []ASTHandle{a, b}, &bound)
	astTestMust(t, e)
	astTestPin(t, m, strict)
	if kind, values := pbTestParameters(t, m, strict); kind != ASTPBOpGE || values[0] != 3 || values[1] != 2 || values[2] != 3 {
		t.Fatal("PB strict-less complement", kind, values)
	}
	for i, original := range []ASTHandle{a, b} {
		child, e := m.GetArg(strict, uint32(i))
		astTestMust(t, e)
		kind, e := m.GetDeclKind(child)
		astTestMust(t, e)
		arg, e := m.GetArg(child, 0)
		astTestMust(t, e)
		if kind != ASTOpNot || arg != original {
			t.Fatal("PB strict-less complemented argument", i, kind, arg, original)
		}
	}
}
