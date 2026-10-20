package bit_blaster

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func TestBitBlasterRewriterTraversalAndStepAccounting(t *testing.T) {
	m, r := rewriterFixture(t)
	left, err := m.BitVectorNumeralUint64(3, 4)
	paramsMust(t, err)
	right, err := m.BitVectorNumeralUint64(5, 4)
	paramsMust(t, err)
	sort, err := m.GetSort(left)
	paramsMust(t, err)
	root, err := m.MkBitVectorPolynomialAdd(sort, ast.SliceArguments([]ast.ASTHandle{left, right}))
	paramsMust(t, err)
	rootRef, err := m.NewRef(root)
	paramsMust(t, err)
	defer func() { paramsMust(t, m.DelRef(&rootRef)) }()
	out, err := m.NewRef(ast.ASTHandle{})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.DelRef(&out)) }()

	paramsMust(t, r.Rewrite(root, out))
	got, err := m.RefGet(out)
	paramsMust(t, err)
	if value := rewriterValue(t, r, got); value != 8 {
		t.Fatal("rewritten value", value)
	}
	steps, err := r.NumSteps()
	paramsMust(t, err)
	if steps == 0 {
		t.Fatal("missing traversal steps")
	}
	paramsMust(t, r.Cleanup())
}

func TestBitBlasterRewriterQuantifierExpansion(t *testing.T) {
	m, r := rewriterFixture(t)
	r.params.BlastQuant = true
	sort, err := m.BitVectorSort(3)
	paramsMust(t, err)
	variable, err := m.MkVar(0, sort)
	paramsMust(t, err)
	numeral, err := m.BitVectorNumeralUint64(5, 3)
	paramsMust(t, err)
	body, err := m.MkEq(variable, numeral)
	paramsMust(t, err)
	name, err := m.Symbols.Intern("word")
	paramsMust(t, err)
	quantifier, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: false, Sorts: []ast.ASTHandle{sort}, Names: []usymbol.Symbol{name}, Body: body, Weight: 7})
	paramsMust(t, err)
	quantifierRef, err := m.NewRef(quantifier)
	paramsMust(t, err)
	defer func() { paramsMust(t, m.DelRef(&quantifierRef)) }()
	out, err := m.NewRef(ast.ASTHandle{})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.DelRef(&out)) }()

	paramsMust(t, r.Rewrite(quantifier, out))
	got, err := m.RefGet(out)
	paramsMust(t, err)
	if count, err := m.GetNumDeclarations(got); err != nil || count != 3 {
		t.Fatal("expanded declarations", count, err)
	}
	if forall, err := m.IsForall(got); err != nil || forall {
		t.Fatal("quantifier kind", forall, err)
	}
	if weight, err := m.GetQuantifierWeight(got); err != nil || weight != 7 {
		t.Fatal("quantifier weight", weight, err)
	}
	for i := uint32(0); i < 3; i++ {
		declSort, err := m.GetDeclarationSort(got, i)
		paramsMust(t, err)
		isBV, err := m.IsBitVectorSort(declSort)
		paramsMust(t, err)
		if isBV {
			t.Fatal("bit declaration retained BV sort", i)
		}
		declName, err := m.GetDeclarationName(got, i)
		paramsMust(t, err)
		text, err := m.Symbols.Display(declName)
		paramsMust(t, err)
		if want := "word." + string(rune('0'+i)); text != want {
			t.Fatal("bit declaration name", text, want)
		}
	}
	if len(r.bindings) != 0 {
		t.Fatal("quantifier bindings retained", len(r.bindings))
	}
}

func TestBitBlasterRewriterBlastFullVariable(t *testing.T) {
	m, r := rewriterFixture(t)
	r.params.BlastFull = true
	sort, err := m.BitVectorSort(3)
	paramsMust(t, err)
	variable, err := m.MkVar(0, sort)
	paramsMust(t, err)
	variableRef, err := m.NewRef(variable)
	paramsMust(t, err)
	defer func() { paramsMust(t, m.DelRef(&variableRef)) }()
	out, err := m.NewRef(ast.ASTHandle{})
	paramsMust(t, err)
	defer func() { paramsMust(t, m.DelRef(&out)) }()
	paramsMust(t, r.Rewrite(variable, out))
	got, err := m.RefGet(out)
	paramsMust(t, err)
	if yes, err := m.IsBitVectorOperation(got, 56); err != nil || !yes {
		t.Fatal("full variable assembly", yes, err)
	}
	for i := uint32(0); i < 3; i++ {
		bit, err := m.GetArg(got, i)
		paramsMust(t, err)
		if yes, err := m.IsBitVectorOperation(bit, 55); err != nil || !yes {
			t.Fatal("full variable bit", i, yes, err)
		}
	}

	// A nonconstant uninterpreted application follows the same full-term path.
	name, err := m.Symbols.Intern("bb-full")
	paramsMust(t, err)
	decl, err := m.MkFuncDecl(name, []ast.ASTHandle{sort}, sort, ast.ASTFuncDeclInfo{})
	paramsMust(t, err)
	app, err := m.MkApp(decl, []ast.ASTHandle{variable})
	paramsMust(t, err)
	appRef, err := m.NewRef(app)
	paramsMust(t, err)
	defer func() { paramsMust(t, m.DelRef(&appRef)) }()
	paramsMust(t, r.Rewrite(app, out))
	got, err = m.RefGet(out)
	paramsMust(t, err)
	if yes, err := m.IsBitVectorOperation(got, 56); err != nil || !yes {
		t.Fatal("full application assembly", yes, err)
	}
}
