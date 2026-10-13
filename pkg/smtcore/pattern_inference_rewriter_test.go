package smtcore

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func TestPatternInferenceParamsBindRewriterAndRefresh(t *testing.T) {
	c, g, local, b := dynAckSourceFixture(t)
	pi, err := g.Symbols.Intern("pi")
	paramsMust(t, err)
	global, err := g.Current.GetParams(pi)
	paramsMust(t, err)
	key := "max_multi_patterns"
	paramsMust(t, c.SetUIntName(global, &key, 3))
	key = "arith"
	paramsMust(t, c.SetUIntName(global, &key, 257))
	key = "block_loop_patterns"
	paramsMust(t, c.SetBoolName(global, &key, false))
	key = "use_database"
	paramsMust(t, c.SetBoolName(global, &key, false))
	key = "arith_weight"
	paramsMust(t, c.SetUIntName(global, &key, 29))
	key = "non_nested_arith_weight"
	paramsMust(t, c.SetUIntName(global, &key, 37))
	key = "warnings"
	paramsMust(t, c.SetBoolName(global, &key, true))
	key = "max_multi_patterns"
	paramsMust(t, c.SetUIntName(local, &key, 2))

	owner, err := NewSMTPatternInferenceParams(c, local, g, b)
	paramsMust(t, err)
	t.Cleanup(func() { paramsMust(t, owner.Close()) })
	options, err := owner.InferenceOptions(nil)
	paramsMust(t, err)
	if options.MaxMultiPatterns != 2 || options.Arithmetic != 257 || options.BlockLoopPatterns ||
		options.UseDatabase || options.ArithmeticWeight != 29 || options.NonNestedArithmeticWeight != 37 ||
		!options.PullQuantifiers || options.NoPatternWeight != -1 || !options.AvoidSkolems || !options.Warnings {
		t.Fatal("local/global pi values did not reach typed rewriter options", options)
	}

	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: c.Symbols(), Numerals: c.Numerals(), Budget: b})
	paramsMust(t, err)
	t.Cleanup(func() { paramsMust(t, m.Close()) })
	var warningOutput bytes.Buffer
	warnings := diagnostic.NewWarningService(&warningOutput)
	r, err := owner.NewInferenceRewriter(m, warnings)
	paramsMust(t, err)
	t.Cleanup(func() { paramsMust(t, r.Close()) })
	boolSort, err := m.MkBoolSort()
	paramsMust(t, err)
	variable, err := m.MkVar(0, boolSort)
	paramsMust(t, err)
	paramsMust(t, m.IncRef(variable))
	t.Cleanup(func() { paramsMust(t, m.DecRef(variable)) })
	truth, err := m.MkFamilyApp(m.BasicFamilyID(), ast.ASTOpTrue, nil, nil)
	paramsMust(t, err)
	inner, err := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: false, Sorts: []ast.ASTHandle{boolSort}, Names: []symbol.Symbol{symbol.Numerical(990)}, Body: truth,
	})
	paramsMust(t, err)
	paramsMust(t, m.IncRef(inner))
	t.Cleanup(func() { paramsMust(t, m.DecRef(inner)) })
	body, err := m.MkFamilyApp(m.BasicFamilyID(), ast.ASTOpAnd, nil, []ast.ASTHandle{variable, inner})
	paramsMust(t, err)
	paramsMust(t, m.IncRef(body))
	t.Cleanup(func() { paramsMust(t, m.DecRef(body)) })
	q, err := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{boolSort}, Names: []symbol.Symbol{symbol.Numerical(991)}, Body: body,
	})
	paramsMust(t, err)
	paramsMust(t, m.IncRef(q))
	t.Cleanup(func() { paramsMust(t, m.DecRef(q)) })
	out, err := m.NewRef(ast.ASTHandle{})
	paramsMust(t, err)
	t.Cleanup(func() { paramsMust(t, m.DelRef(&out)) })
	if err := r.Rewrite(q, out); !errors.Is(err, ast.ErrASTPending) {
		t.Fatal("source default pull remains pending", err)
	}
	key = "pull_quantifiers"
	paramsMust(t, c.SetBoolName(local, &key, false))
	paramsMust(t, owner.Update(local))
	paramsMust(t, owner.UpdateInferenceRewriter(r, warnings))
	paramsMust(t, r.Resume(out))
	got, err := m.RefGet(out)
	paramsMust(t, err)
	if got != q {
		t.Fatal("refreshed pi option did not reach suspended traversal", got, q)
	}
	warningOutput.Reset()
	decl, err := m.MkFuncDecl(symbol.Numerical(992), []ast.ASTHandle{boolSort}, boolSort, ast.ASTFuncDeclInfo{})
	paramsMust(t, err)
	paramsMust(t, m.IncRef(decl))
	t.Cleanup(func() { paramsMust(t, m.DecRef(decl)) })
	app, err := m.MkApp(decl, []ast.ASTHandle{variable})
	paramsMust(t, err)
	paramsMust(t, m.IncRef(app))
	t.Cleanup(func() { paramsMust(t, m.DecRef(app)) })
	qid, err := m.Symbols.Intern("smt-warning")
	paramsMust(t, err)
	withNoPattern, err := m.MkQuantifier(ast.ASTQuantifierSpec{
		Forall: true, Sorts: []ast.ASTHandle{boolSort}, Names: []symbol.Symbol{symbol.Numerical(993)},
		Body: app, QID: qid, NoPatterns: []ast.ASTHandle{app},
	})
	paramsMust(t, err)
	paramsMust(t, m.IncRef(withNoPattern))
	t.Cleanup(func() { paramsMust(t, m.DecRef(withNoPattern)) })
	paramsMust(t, r.Rewrite(withNoPattern, out))
	const wantWarning = "WARNING: ignoring nopats annotation because Z3 couldn't find any other pattern (quantifier id: smt-warning)\n"
	if got := warningOutput.String(); got != wantWarning {
		t.Fatal("SMT parameter warning did not reach the borrowed stream", got)
	}
}
