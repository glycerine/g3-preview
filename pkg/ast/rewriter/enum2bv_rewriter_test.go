package rewriter

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func enum2BVFixture(t *testing.T) (*ast.ASTManager, ast.ASTHandle, []ast.ASTHandle) {
	t.Helper()
	m := astTestManager(t)
	_, err := m.RegisterArithmeticPlugin()
	astTestMust(t, err)
	_, err = m.RegisterBitVectorPlugin()
	astTestMust(t, err)
	_, err = m.RegisterDatatypePlugin()
	astTestMust(t, err)
	sorts, err := m.MkDatatypes([]ast.DatatypeDefinition{{
		Name: astTestSymbol(t, m, "TrafficLight"),
		Constructors: []ast.DatatypeConstructorDefinition{
			{Name: astTestSymbol(t, m, "red"), Recognizer: astTestSymbol(t, m, "is-red")},
			{Name: astTestSymbol(t, m, "amber"), Recognizer: astTestSymbol(t, m, "is-amber")},
			{Name: astTestSymbol(t, m, "green"), Recognizer: astTestSymbol(t, m, "is-green")},
		},
	}})
	astTestMust(t, err)
	constructors, err := m.DatatypeConstructors(sorts[0])
	astTestMust(t, err)
	return m, sorts[0], constructors
}

func enum2BVRefs(t *testing.T, m *ast.ASTManager) (ast.ASTRef, ast.ASTRef) {
	t.Helper()
	out, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	proof, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, m.DelRef(&proof)); astTestMust(t, m.DelRef(&out)) })
	return out, proof
}

func TestEnum2BVConstantsRecognizersMappingsBoundsAndScopes(t *testing.T) {
	m, enumSort, constructors := enum2BVFixture(t)
	r, err := NewEnum2BVRewriter(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	out, proof := enum2BVRefs(t, m)
	truth, err := m.MkTrue()
	astTestMust(t, err)
	astTestMust(t, r.Rewrite(truth, out, proof))
	if unchanged, e := m.RefGet(out); e != nil || unchanged != truth {
		t.Fatal("ordinary expression changed", unchanged, e)
	}

	constant, err := m.MkConst(astTestSymbol(t, m, "light"), enumSort)
	astTestMust(t, err)
	green, err := m.MkApp(constructors[2], nil)
	astTestMust(t, err)
	equality, err := m.MkEq(constant, green)
	astTestMust(t, err)
	astTestMust(t, r.Push())
	astTestMust(t, r.Rewrite(equality, out, proof))
	rewritten, err := m.RefGet(out)
	astTestMust(t, err)
	if kind, e := m.GetDeclKind(rewritten); e != nil || kind != ast.ASTOpEq {
		t.Fatal("enum equality was not converted to bit-vector equality")
	}
	left, err := m.GetArg(rewritten, 0)
	astTestMust(t, err)
	leftSort, err := m.GetSort(left)
	astTestMust(t, err)
	if width, e := m.BitVectorSize(leftSort); e != nil || width != 2 {
		t.Fatal("converted width", width, e)
	}
	if len(r.EnumToBitVector()) != 1 || len(r.BitVectorToEnum()) != 1 || len(r.EnumDefinitions()) != 1 {
		t.Fatal("conversion maps", r.EnumToBitVector(), r.BitVectorToEnum(), r.EnumDefinitions())
	}
	var bounds []ast.ASTHandle
	astTestMust(t, r.FlushSideConstraints(&bounds))
	if len(bounds) != 1 {
		t.Fatal("three-value enum bound", len(bounds))
	}
	recognizer, err := m.DatatypeConstructorRecognizer(constructors[2])
	astTestMust(t, err)
	test, err := m.MkApp(recognizer, []ast.ASTHandle{constant})
	astTestMust(t, err)
	astTestMust(t, r.Rewrite(test, out, proof))
	if translated, e := r.NumTranslated(); e != nil || translated < 2 {
		t.Fatal("translation count", translated, e)
	}
	astTestMust(t, r.Pop(1))
	if len(r.EnumToBitVector()) != 0 {
		t.Fatal("scope pop retained enum mapping")
	}
	astTestMust(t, r.Cleanup())
	if translated, e := r.NumTranslated(); e != nil || translated != 0 {
		t.Fatal("cleanup translation count", translated, e)
	}
}

func TestEnum2BVQuantifierSortAndBound(t *testing.T) {
	m, enumSort, constructors := enum2BVFixture(t)
	r, err := NewEnum2BVRewriter(m)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, r.Close()) })
	out, proof := enum2BVRefs(t, m)
	variable, err := m.MkVar(0, enumSort)
	astTestMust(t, err)
	red, err := m.MkApp(constructors[0], nil)
	astTestMust(t, err)
	body, err := m.MkEq(variable, red)
	astTestMust(t, err)
	q, err := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{enumSort},
		Names: []usymbol.Symbol{astTestSymbol(t, m, "x")}, Body: body,
	})
	astTestMust(t, err)
	astTestMust(t, r.Rewrite(q, out, proof))
	result, err := m.RefGet(out)
	astTestMust(t, err)
	convertedSort, err := m.GetDeclarationSort(result, 0)
	astTestMust(t, err)
	if width, e := m.BitVectorSize(convertedSort); e != nil || width != 2 {
		t.Fatal("quantifier enum sort", width, e)
	}
	convertedBody, err := m.GetQuantifierBody(result)
	astTestMust(t, err)
	if kind, e := m.GetDeclKind(convertedBody); e != nil || kind != ast.ASTOpImplies {
		t.Fatal("universal enum bound", kind, e)
	}
}
