package macros_test

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/macros"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func fixture(t *testing.T) *ast.ASTManager {
	t.Helper()
	b := arena.NewBudget(1 << 30)
	s, err := usymbol.NewManager(usymbol.Config{Budget: b})
	must(t, err)
	n, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	must(t, err)
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: n, Budget: b})
	must(t, err)
	t.Cleanup(func() {
		must(t, m.Close())
		must(t, n.Close())
		must(t, s.Close())
		if b.Used() != 0 {
			t.Fatalf("retained %d bytes", b.Used())
		}
	})
	return m
}

func declaration(t *testing.T, m *ast.ASTManager, name string, domain []ast.ASTHandle, result ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	s, err := m.Symbols.Intern(name)
	must(t, err)
	d, err := m.MkFuncDecl(s, domain, result, ast.ASTFuncDeclInfo{})
	must(t, err)
	return d
}

func TestMacroUtilHeadsSimpleMacrosAndForbiddenDeclarations(t *testing.T) {
	m := fixture(t)
	u, err := macros.NewUtil(m)
	must(t, err)
	b, err := m.MkBoolSort()
	must(t, err)
	v0, err := m.MkVar(0, b)
	must(t, err)
	v1, err := m.MkVar(1, b)
	must(t, err)
	f := declaration(t, m, "f", []ast.ASTHandle{b, b}, b)
	head, err := m.MkApp(f, []ast.ASTHandle{v1, v0})
	must(t, err)
	if yes, e := u.IsMacroHead(head, 2); e != nil || !yes {
		t.Fatal("permuted head", yes, e)
	}
	duplicate, err := m.MkApp(f, []ast.ASTHandle{v0, v0})
	must(t, err)
	if yes, e := u.IsMacroHead(duplicate, 2); e != nil || yes {
		t.Fatal("duplicate head", yes, e)
	}
	eq, err := m.MkEq(head, v0)
	must(t, err)
	gotHead, def, yes, err := u.IsSimpleMacro(eq, 2)
	must(t, err)
	if !yes || gotHead != head || def != v0 {
		t.Fatal("simple macro", yes, gotHead, def)
	}
	u.SetForbidden([]ast.ASTHandle{f})
	if _, _, matched, e := u.IsSimpleMacro(eq, 2); e != nil || matched {
		t.Fatal("forbidden macro", matched, e)
	}
	u.SetForbidden(nil)
	self, err := m.MkEq(head, duplicate)
	must(t, err)
	if _, _, matched, e := u.IsSimpleMacro(self, 2); e != nil || matched {
		t.Fatal("recursive definition", matched, e)
	}
}

func TestMacroUtilQuasiHeadNormalizationAndCandidates(t *testing.T) {
	m := fixture(t)
	u, err := macros.NewUtil(m)
	must(t, err)
	b, err := m.MkBoolSort()
	must(t, err)
	v0, err := m.MkVar(0, b)
	must(t, err)
	v1, err := m.MkVar(1, b)
	must(t, err)
	f := declaration(t, m, "f", []ast.ASTHandle{b, b, b}, b)
	g := declaration(t, m, "g", []ast.ASTHandle{b, b}, b)
	groundName, err := m.Symbols.Intern("ground")
	must(t, err)
	ground, err := m.MkConst(groundName, b)
	must(t, err)
	quasi, err := m.MkApp(f, []ast.ASTHandle{v0, ground, v1})
	must(t, err)
	if yes, e := u.IsQuasiMacroHead(quasi, 2); e != nil || !yes {
		t.Fatal("quasi head", yes, e)
	}

	head, err := m.MkApp(g, []ast.ASTHandle{v1, v0})
	must(t, err)
	definition, err := m.MkApp(f, []ast.ASTHandle{v1, v0, ground})
	must(t, err)
	out, err := m.NewRef(ast.ASTHandle{})
	must(t, err)
	must(t, u.NormalizeExpr(head, 2, definition, out))
	normalized, err := m.RefGet(out)
	must(t, err)
	want, err := m.MkApp(f, []ast.ASTHandle{v0, v1, ground})
	must(t, err)
	if normalized != want {
		t.Fatal("variable permutation was not normalized", normalized, want)
	}
	t.Cleanup(func() { must(t, m.DelRef(&out)) })

	eq, err := m.MkEq(head, definition)
	must(t, err)
	candidates, err := macros.NewCandidates(m)
	must(t, err)
	t.Cleanup(func() { must(t, candidates.Close()) })
	must(t, u.CollectMacroCandidates(eq, 2, candidates))
	if candidates.Len() != 2 {
		t.Fatalf("candidates = %d", candidates.Len())
	}
	row, err := candidates.At(0)
	must(t, err)
	if row.Declaration != g || row.Definition != want || !m.IsTrue(row.Condition) || !row.SatisfyAtom {
		t.Fatal("candidate row", row)
	}
}

func TestMacroUtilQuasiConversionClauseConditionsAndHints(t *testing.T) {
	m := fixture(t)
	u, err := macros.NewUtil(m)
	must(t, err)
	b, err := m.MkBoolSort()
	must(t, err)
	v0, err := m.MkVar(0, b)
	must(t, err)
	f := declaration(t, m, "candidate-f", []ast.ASTHandle{b, b}, b)
	duplicate, err := m.MkApp(f, []ast.ASTHandle{v0, v0})
	must(t, err)
	headRef, err := m.NewRef(ast.ASTHandle{})
	must(t, err)
	t.Cleanup(func() { must(t, m.DelRef(&headRef)) })
	conditionRef, err := m.NewRef(ast.ASTHandle{})
	must(t, err)
	t.Cleanup(func() { must(t, m.DelRef(&conditionRef)) })
	declarations, err := u.QuasiMacroHeadToMacroHead(duplicate, 1, headRef, conditionRef)
	must(t, err)
	if declarations != 2 {
		t.Fatal("converted declaration count", declarations)
	}
	converted, err := m.RefGet(headRef)
	must(t, err)
	if yes, e := u.IsMacroHead(converted, 2); e != nil || !yes {
		t.Fatal("converted head", yes, e)
	}
	v1, err := m.MkVar(1, b)
	must(t, err)
	wantCondition, err := m.MkEq(v1, v0)
	must(t, err)
	condition, err := m.RefGet(conditionRef)
	must(t, err)
	if condition != wantCondition {
		t.Fatal("quasi conversion condition", condition, wantCondition)
	}

	valueSort, err := m.MkUninterpretedSort(mustSymbol(t, m, "candidate-value"), nil)
	must(t, err)
	oneArg := declaration(t, m, "candidate-one", []ast.ASTHandle{b}, valueSort)
	oneHead, err := m.MkApp(oneArg, []ast.ASTHandle{v0})
	must(t, err)
	ground, err := m.MkConst(mustSymbol(t, m, "candidate-ground"), valueSort)
	must(t, err)
	equality, err := m.MkEq(oneHead, ground)
	must(t, err)
	guard, err := m.MkConst(mustSymbol(t, m, "candidate-guard"), b)
	must(t, err)
	clause, err := m.MkOr(equality, guard)
	must(t, err)
	q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b}, Names: []usymbol.Symbol{mustSymbol(t, m, "candidate-x")}, Body: clause})
	must(t, err)
	candidates, err := macros.NewCandidates(m)
	must(t, err)
	t.Cleanup(func() { must(t, candidates.Close()) })
	must(t, u.CollectQuantifierCandidates(q, candidates))
	if candidates.Len() != 1 {
		t.Fatalf("clause candidates = %d", candidates.Len())
	}
	row, err := candidates.At(0)
	must(t, err)
	wantGuard, err := m.MkNotSimplified(guard)
	must(t, err)
	if row.Declaration != oneArg || row.Definition != ground || row.Condition != wantGuard || row.Hint || !row.SatisfyAtom {
		t.Fatal("conditional clause candidate", row)
	}

	hintDecl := declaration(t, m, "candidate-hint", []ast.ASTHandle{b}, b)
	hintHead, err := m.MkApp(hintDecl, []ast.ASTHandle{v0})
	must(t, err)
	hintEquality, err := m.MkEq(hintHead, v0)
	must(t, err)
	must(t, u.CollectMacroCandidates(hintEquality, 2, candidates))
	if candidates.Len() != 1 {
		t.Fatalf("hint candidates = %d", candidates.Len())
	}
	row, err = candidates.At(0)
	must(t, err)
	if row.Declaration != hintDecl || row.Definition != v0 || !row.Condition.IsNull() || !row.Hint {
		t.Fatal("hint candidate", row)
	}
}

func quantifiedMacro(t *testing.T, m *ast.ASTManager, declaration ast.ASTHandle, definition ast.ASTHandle, sort ast.ASTHandle) ast.ASTHandle {
	t.Helper()
	v, err := m.MkVar(0, sort)
	must(t, err)
	head, err := m.MkApp(declaration, []ast.ASTHandle{v})
	must(t, err)
	body, err := m.MkEq(head, definition)
	must(t, err)
	name, err := m.Symbols.Intern("x")
	must(t, err)
	q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{sort}, Names: []usymbol.Symbol{name}, Body: body})
	must(t, err)
	return q
}

func TestMacroManagerInsertionCyclesScopesAndForbiddenTraversal(t *testing.T) {
	m := fixture(t)
	manager, err := macros.NewManager(m)
	must(t, err)
	t.Cleanup(func() { must(t, manager.Close()) })
	b, err := m.MkBoolSort()
	must(t, err)
	v0, err := m.MkVar(0, b)
	must(t, err)
	f := declaration(t, m, "manager-f", []ast.ASTHandle{b}, b)
	g := declaration(t, m, "manager-g", []ast.ASTHandle{b}, b)
	gv, err := m.MkApp(g, []ast.ASTHandle{v0})
	must(t, err)
	fv, err := m.MkApp(f, []ast.ASTHandle{v0})
	must(t, err)
	qf := quantifiedMacro(t, m, f, gv, b)
	qg := quantifiedMacro(t, m, g, fv, b)
	must(t, manager.PushScope())
	inserted, err := manager.Insert(f, qf, ast.ASTHandle{}, ast.ASTDependency{})
	must(t, err)
	if !inserted || manager.NumMacros() != 1 || manager.FirstMacroLastLevel() != 0 {
		t.Fatal("first insertion", inserted, manager.NumMacros(), manager.FirstMacroLastLevel())
	}
	inserted, err = manager.Insert(g, qg, ast.ASTHandle{}, ast.ASTDependency{})
	must(t, err)
	if inserted {
		t.Fatal("dependency cycle accepted")
	}
	if got, yes := manager.MacroQuantifier(f); !yes || got != qf {
		t.Fatal("macro lookup", yes, got)
	}
	out, err := m.NewRef(ast.ASTHandle{})
	must(t, err)
	t.Cleanup(func() { must(t, m.DelRef(&out)) })
	if got, e := manager.MacroInterpretation(0, out); e != nil || got != f {
		t.Fatal("interpretation", got, e)
	}
	must(t, manager.MarkForbidden([]ast.ASTHandle{gv}))
	if !manager.IsForbidden(g) || manager.IsForbidden(f) {
		t.Fatal("forbidden traversal")
	}
	h := declaration(t, m, "manager-pattern-h", []ast.ASTHandle{b}, b)
	hv, err := m.MkApp(h, []ast.ASTHandle{v0})
	must(t, err)
	pattern, err := m.MkPattern([]ast.ASTHandle{hv})
	must(t, err)
	name, err := m.Symbols.Intern("pattern-x")
	must(t, err)
	annotated, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b}, Names: []usymbol.Symbol{name}, Body: fv, Patterns: []ast.ASTHandle{pattern}})
	must(t, err)
	must(t, manager.MarkForbidden([]ast.ASTHandle{annotated}))
	if !manager.IsForbidden(h) {
		t.Fatal("quantifier pattern declaration was not marked forbidden")
	}
	must(t, manager.PopScope(1))
	if manager.HasMacros() || manager.IsForbidden(g) || manager.IsForbidden(h) {
		t.Fatal("scope restoration")
	}
}

func TestMacroManagerExpandFixedPointTheoryRewriteAndDependencies(t *testing.T) {
	m := fixture(t)
	manager, err := macros.NewManager(m)
	must(t, err)
	t.Cleanup(func() { must(t, manager.Close()) })
	b, err := m.MkBoolSort()
	must(t, err)
	v, err := m.MkVar(0, b)
	must(t, err)
	f := declaration(t, m, "expand-f", []ast.ASTHandle{b}, b)
	g := declaration(t, m, "expand-g", []ast.ASTHandle{b}, b)
	doubleNot, err := m.MkNot(v)
	must(t, err)
	doubleNot, err = m.MkNot(doubleNot)
	must(t, err)
	qg := quantifiedMacro(t, m, g, doubleNot, b)
	gv, err := m.MkApp(g, []ast.ASTHandle{v})
	must(t, err)
	qf := quantifiedMacro(t, m, f, gv, b)
	depGValue, err := m.MkConst(mustSymbol(t, m, "dep-g"), b)
	must(t, err)
	depFValue, err := m.MkConst(mustSymbol(t, m, "dep-f"), b)
	must(t, err)
	inputDepValue, err := m.MkConst(mustSymbol(t, m, "dep-input"), b)
	must(t, err)
	depG, err := m.DependencyLeaf(depGValue)
	must(t, err)
	depF, err := m.DependencyLeaf(depFValue)
	must(t, err)
	inputDep, err := m.DependencyLeaf(inputDepValue)
	must(t, err)
	inserted, err := manager.Insert(g, qg, ast.ASTHandle{}, depG)
	must(t, err)
	if !inserted {
		t.Fatal("g macro was not inserted")
	}
	inserted, err = manager.Insert(f, qf, ast.ASTHandle{}, depF)
	must(t, err)
	if !inserted {
		t.Fatal("f macro was not inserted")
	}
	a, err := m.MkConst(mustSymbol(t, m, "expand-a"), b)
	must(t, err)
	term, err := m.MkApp(f, []ast.ASTHandle{a})
	must(t, err)
	out, err := m.NewRef(ast.ASTHandle{})
	must(t, err)
	t.Cleanup(func() { must(t, m.DelRef(&out)) })
	outProof, err := m.NewRef(ast.ASTHandle{})
	must(t, err)
	t.Cleanup(func() { must(t, m.DelRef(&outProof)) })
	used, err := manager.ExpandMacros(term, ast.ASTHandle{}, inputDep, out, outProof)
	must(t, err)
	t.Cleanup(func() { must(t, m.DependencyDec(used)) })
	got, err := m.RefGet(out)
	must(t, err)
	if got != a {
		t.Fatal("fixed-point expansion and theory rewrite", got, a)
	}
	values, err := m.DependencyLinearize(used, nil)
	must(t, err)
	want := map[ast.ASTHandle]bool{inputDepValue: true, depFValue: true, depGValue: true}
	if len(values) != len(want) {
		t.Fatal("dependency count", len(values), values)
	}
	for _, value := range values {
		if !want[value] {
			t.Fatal("unexpected dependency", value)
		}
	}
}

func mustSymbol(t *testing.T, m *ast.ASTManager, text string) usymbol.Symbol {
	t.Helper()
	s, err := m.Symbols.Intern(text)
	must(t, err)
	return s
}

func arithmeticNumeral(t *testing.T, m *ast.ASTManager, text string) ast.ASTHandle {
	t.Helper()
	value, err := m.NumeralService().NewValue()
	must(t, err)
	defer func() { must(t, m.NumeralService().Del(&value)) }()
	must(t, m.NumeralService().SetString(&value, text))
	h, err := m.ArithmeticNumeral(&value, true)
	must(t, err)
	return h
}

func bitVectorNumeral(t *testing.T, m *ast.ASTManager, value int64, width uint32) ast.ASTHandle {
	t.Helper()
	n, err := m.NumeralService().NewValue()
	must(t, err)
	defer func() { must(t, m.NumeralService().Del(&n)) }()
	must(t, m.NumeralService().SetInt64(&n, value))
	h, err := m.BitVectorNumeral(&n, width)
	must(t, err)
	return h
}

func TestMacroUtilArithmeticCandidatesAndPseudoPredicates(t *testing.T) {
	m := fixture(t)
	_, err := m.RegisterArithmeticPlugin()
	must(t, err)
	u, err := macros.NewUtil(m)
	must(t, err)
	arithmetic, err := m.ArithmeticFamilyID()
	must(t, err)
	zero := arithmeticNumeral(t, m, "0")
	integerSort, err := m.GetSort(zero)
	must(t, err)
	v, err := m.MkVar(0, integerSort)
	must(t, err)
	f := declaration(t, m, "arith-candidate-f", []ast.ASTHandle{integerSort}, integerSort)
	head, err := m.MkApp(f, []ast.ASTHandle{v})
	must(t, err)
	two := arithmeticNumeral(t, m, "2")
	five := arithmeticNumeral(t, m, "5")
	sum, err := m.MkFamilyApp(arithmetic, 6, nil, []ast.ASTHandle{head, two})
	must(t, err)
	equality, err := m.MkEq(sum, five)
	must(t, err)
	candidates, err := macros.NewCandidates(m)
	must(t, err)
	t.Cleanup(func() { must(t, candidates.Close()) })
	must(t, u.CollectMacroCandidates(equality, 1, candidates))
	if candidates.Len() != 1 {
		t.Fatalf("arithmetic candidates = %d", candidates.Len())
	}
	row, err := candidates.At(0)
	must(t, err)
	if row.Declaration != f || row.Inequality || row.Hint || !row.SatisfyAtom {
		t.Fatal("arithmetic candidate flags", row)
	}
	if yes, e := u.IsZero(zero); e != nil || !yes {
		t.Fatal("arithmetic zero", yes, e)
	}
	if family, e := m.GetFamilyID(row.Definition); e != nil || family != arithmetic {
		t.Fatal("arithmetic definition family", family, e)
	}
	if op, e := m.GetDeclKind(row.Definition); e != nil || op != 7 {
		t.Fatal("arithmetic definition operation", op, e)
	}
	legacyHead, legacyDefinition, inverse, yes, err := u.IsArithmeticMacro(equality, 1)
	must(t, err)
	if !yes || inverse || legacyHead != head || legacyDefinition != row.Definition {
		t.Fatal("legacy arithmetic macro", yes, inverse, legacyHead, legacyDefinition)
	}
	varPlusGround, err := m.MkFamilyApp(arithmetic, 6, nil, []ast.ASTHandle{two, v})
	must(t, err)
	groundRef, err := m.NewRef(ast.ASTHandle{})
	must(t, err)
	t.Cleanup(func() { must(t, m.DelRef(&groundRef)) })
	inverse, variable, yes, err := u.IsVarPlusGround(varPlusGround, groundRef)
	must(t, err)
	groundPart, err := m.RefGet(groundRef)
	must(t, err)
	if !yes || inverse || variable != v || groundPart != two {
		t.Fatal("variable plus ground", yes, inverse, variable, groundPart)
	}

	_, err = m.RegisterBitVectorPlugin()
	must(t, err)
	b, err := m.MkBoolSort()
	must(t, err)
	bvValue := bitVectorNumeral(t, m, 3, 8)
	bvSort, err := m.GetSort(bvValue)
	must(t, err)
	bvHeadDecl := declaration(t, m, "pseudo-f", []ast.ASTHandle{b}, bvSort)
	bvVar, err := m.MkVar(0, b)
	must(t, err)
	bvHead, err := m.MkApp(bvHeadDecl, []ast.ASTHandle{bvVar})
	must(t, err)
	pseudoAtom, err := m.MkEq(bvHead, bvValue)
	must(t, err)
	pseudoBody, err := m.MkIff(pseudoAtom, bvVar)
	must(t, err)
	pseudo, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{b}, Names: []usymbol.Symbol{mustSymbol(t, m, "pseudo-x")}, Body: pseudoBody})
	must(t, err)
	gotHead, gotValue, gotDefinition, yes, err := u.IsPseudoPredicateMacro(pseudo)
	must(t, err)
	if !yes || gotHead != bvHead || gotValue != bvValue || gotDefinition != bvVar {
		t.Fatal("pseudo predicate", yes, gotHead, gotValue, gotDefinition)
	}
}

func TestMacroManagerExpandProofChain(t *testing.T) {
	m := fixture(t)
	must(t, m.ToggleProofMode(ast.ASTProofEnabled))
	manager, err := macros.NewManager(m)
	must(t, err)
	t.Cleanup(func() { must(t, manager.Close()) })
	b, err := m.MkBoolSort()
	must(t, err)
	v, err := m.MkVar(0, b)
	must(t, err)
	f := declaration(t, m, "proof-expand-f", []ast.ASTHandle{b}, b)
	notV, err := m.MkNot(v)
	must(t, err)
	q := quantifiedMacro(t, m, f, notV, b)
	qProof, err := m.MkProofAsserted(q)
	must(t, err)
	inserted, err := manager.Insert(f, q, qProof, ast.ASTDependency{})
	must(t, err)
	if !inserted {
		t.Fatal("proof macro was not inserted")
	}
	a, err := m.MkConst(mustSymbol(t, m, "proof-expand-a"), b)
	must(t, err)
	term, err := m.MkApp(f, []ast.ASTHandle{a})
	must(t, err)
	termProof, err := m.MkProofAsserted(term)
	must(t, err)
	out, err := m.NewRef(ast.ASTHandle{})
	must(t, err)
	t.Cleanup(func() { must(t, m.DelRef(&out)) })
	outProof, err := m.NewRef(ast.ASTHandle{})
	must(t, err)
	t.Cleanup(func() { must(t, m.DelRef(&outProof)) })
	used, err := manager.ExpandMacros(term, termProof, ast.ASTDependency{}, out, outProof)
	must(t, err)
	if !used.IsNull() {
		t.Fatal("unexpected dependency")
	}
	want, err := m.MkNot(a)
	must(t, err)
	got, err := m.RefGet(out)
	must(t, err)
	if got != want {
		t.Fatal("proof expansion result", got, want)
	}
	proof, err := m.RefGet(outProof)
	must(t, err)
	fact, err := m.GetProofFact(proof)
	must(t, err)
	if fact != want {
		t.Fatal("expanded proof fact", fact, want)
	}
}
