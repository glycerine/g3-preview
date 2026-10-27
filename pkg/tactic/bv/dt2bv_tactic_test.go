package bv

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/model"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/cooperate"
	uparams "github.com/glycerine/g3/pkg/util/params"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func dt2bvFixture(t *testing.T, proofs bool) *DT2BVTactic {
	t.Helper()
	base := sharingTacticFixture(t, proofs)
	m := base.manager
	sharingMust(t, m.RegisterDeclarationPlugins())
	coop, err := cooperate.NewCooperation(m.Budget())
	sharingMust(t, err)
	t.Cleanup(func() { sharingMust(t, coop.Close()) })
	globals := uparams.BorrowProcessGlobalParamsService()
	services := model.ModelEvaluatorConfig{Context: base.context, Globals: globals, Memory: m.NumeralService().RuntimeOwner(), Cooperation: coop}
	r, err := NewDT2BVTactic(m, base.context, nil, services)
	sharingMust(t, err)
	t.Cleanup(func() { sharingMust(t, r.Close()) })
	return r
}
func dt2bvEnum(t *testing.T, r *DT2BVTactic) (ast.ASTHandle, []ast.ASTHandle) {
	t.Helper()
	m := r.manager
	name := func(s string) usymbol.Symbol { v, e := m.Symbols.Intern(s); sharingMust(t, e); return v }
	sorts, err := m.MkDatatypes([]ast.DatatypeDefinition{{Name: name("DTColor"), Constructors: []ast.DatatypeConstructorDefinition{{Name: name("red"), Recognizer: name("is-red")}, {Name: name("amber"), Recognizer: name("is-amber")}, {Name: name("green"), Recognizer: name("is-green")}}}})
	sharingMust(t, err)
	refs, err := m.NewRef(sorts[0])
	sharingMust(t, err)
	t.Cleanup(func() { sharingMust(t, m.DelRef(&refs)) })
	constructors, err := m.DatatypeConstructors(sorts[0])
	sharingMust(t, err)
	for _, decl := range constructors {
		ref, e := m.NewRef(decl)
		sharingMust(t, e)
		t.Cleanup(func() { sharingMust(t, m.DelRef(&ref)) })
	}
	return sorts[0], constructors
}
func dt2bvGoal(t *testing.T, r *DT2BVTactic, sort ast.ASTHandle, constructors []ast.ASTHandle, name string, proofs bool) (*tactic.Goal, ast.ASTHandle) {
	t.Helper()
	m := r.manager
	symbol, err := m.Symbols.Intern(name)
	sharingMust(t, err)
	x, err := m.MkConst(symbol, sort)
	sharingMust(t, err)
	green, err := m.MkApp(constructors[2], nil)
	sharingMust(t, err)
	eq, err := m.MkEq(x, green)
	sharingMust(t, err)
	g, err := tactic.NewGoal(m, proofs, true, true)
	sharingMust(t, err)
	t.Cleanup(func() { sharingMust(t, g.Close()) })
	pr, err := m.MkProofAsserted(eq)
	sharingMust(t, err)
	dep, err := m.DependencyLeaf(eq)
	sharingMust(t, err)
	sharingMust(t, g.AssertWithProof(eq, pr, dep))
	decl, err := m.GetDecl(x)
	sharingMust(t, err)
	return g, decl
}
func TestDT2BVModelReconstructionProofsAndDependencies(t *testing.T) {
	r := dt2bvFixture(t, true)
	m := r.manager
	sort, constructors := dt2bvEnum(t, r)
	g, decl := dt2bvGoal(t, r, sort, constructors, "dt-model-x", true)
	dependency, err := g.Dependency(0)
	sharingMust(t, err)
	var result tactic.TacticResult
	t.Cleanup(func() { sharingMust(t, result.Close()) })
	sharingMust(t, r.Apply(g, &result))
	if result.ModelConverter == nil || len(result.Goals) != 1 || result.Goals[0] != g {
		t.Fatal("missing model converter/goal")
	}
	dep, err := g.Dependency(0)
	sharingMust(t, err)
	if dep != dependency {
		t.Fatal("rewriter lost source dependency")
	}
	n, err := g.Size()
	sharingMust(t, err)
	if n != 2 {
		t.Fatal("three-value enum needs one side constraint", n)
	}
	for i := uint32(0); i < n; i++ {
		pr, e := g.Proof(i)
		sharingMust(t, e)
		fact, e := m.GetProofFact(pr)
		sharingMust(t, e)
		form, e := g.Form(i)
		sharingMust(t, e)
		if fact != form {
			t.Fatal("rewritten goal proof fact", i)
		}
	}
	form, err := g.Form(0)
	sharingMust(t, err)
	bv, err := m.GetArg(form, 0)
	sharingMust(t, err)
	bvDecl, err := m.GetDecl(bv)
	sharingMust(t, err)
	sharingMust(t, r.Close())
	md, err := model.NewModel(m)
	sharingMust(t, err)
	var out tactic.ModelRef
	sharingMust(t, out.Set(md.Core()))
	t.Cleanup(func() { sharingMust(t, out.Close()) })
	value, err := m.BitVectorNumeralUnsigned(1, 2)
	sharingMust(t, err)
	sharingMust(t, md.Core().RegisterConstant(bvDecl, value))
	sharingMust(t, result.ModelConverter.ApplyModel(&out, 0))
	reconstructed, err := out.Get()
	sharingMust(t, err)
	value, err = reconstructed.ConstantInterpretation(decl)
	sharingMust(t, err)
	valueDecl, err := m.GetDecl(value)
	sharingMust(t, err)
	if valueDecl != constructors[1] {
		t.Fatal("enum interpretation not reconstructed", valueDecl)
	}
	helper, err := reconstructed.ConstantInterpretation(bvDecl)
	sharingMust(t, err)
	if !helper.IsNull() {
		t.Fatal("fresh BV helper escaped model filter")
	}
}
func TestDT2BVClassificationPersistsUntilCleanupAndAppendsGoals(t *testing.T) {
	r := dt2bvFixture(t, false)
	m := r.manager
	sort, constructors := dt2bvEnum(t, r)
	first, decl := dt2bvGoal(t, r, sort, constructors, "dt-classify-x", false)
	boolean, err := m.MkBoolSort()
	sharingMust(t, err)
	symbol, err := m.Symbols.Intern("uses-enum")
	sharingMust(t, err)
	fn, err := m.MkFuncDecl(symbol, []ast.ASTHandle{sort}, boolean, ast.ASTFuncDeclInfo{})
	sharingMust(t, err)
	x, err := m.MkApp(decl, nil)
	sharingMust(t, err)
	predicate, err := m.MkApp(fn, []ast.ASTHandle{x})
	sharingMust(t, err)
	sharingMust(t, first.Assert(predicate))
	var result tactic.TacticResult
	t.Cleanup(func() { sharingMust(t, result.Close()) })
	sharingMust(t, result.BindManager(m))
	sharingMust(t, result.AddGoal(first))
	sharingMust(t, r.Apply(first, &result))
	if result.ModelConverter != nil || len(result.Goals) != 2 {
		t.Fatal("nonfinite sort translated or input goal buffer erased")
	}
	second, _ := dt2bvGoal(t, r, sort, constructors, "dt-classify-y", false)
	sharingMust(t, r.Apply(second, &result))
	if result.ModelConverter != nil {
		t.Fatal("source non-finite cache did not persist")
	}
	sharingMust(t, r.Cleanup())
	sharingMust(t, r.Apply(second, &result))
	if result.ModelConverter == nil {
		t.Fatal("cleanup did not reset classification")
	}
	depth, err := second.Depth()
	sharingMust(t, err)
	if depth != 2 {
		t.Fatal("source depth progression", depth)
	}
}
func TestDT2BVApplicationAllocationPrefixes(t *testing.T) {
	r := dt2bvFixture(t, false)
	sort, constructors := dt2bvEnum(t, r)
	complete := false
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		sharingMust(t, r.Cleanup())
		g, _ := dt2bvGoal(t, r, sort, constructors, "dt-prefix-x", false)
		var result tactic.TacticResult
		r.manager.Budget().FailAfter(ordinal)
		err := r.Apply(g, &result)
		r.manager.Budget().FailAfter(-1)
		if err != nil && !errors.Is(err, arena.ErrInjectedFailure) {
			t.Fatal("unexpected source prefix error", ordinal, err)
		}
		sharingMust(t, result.Close())
		sharingMust(t, g.Close())
		if r.pending != nil {
			t.Fatal("allocation refusal left unretired lexical owner", ordinal)
		}
		if err == nil {
			complete = true
			break
		}
	}
	if !complete {
		t.Fatal("application allocation census did not finish")
	}
}

func TestDT2BVConstructorParameterSnapshotAndRefusals(t *testing.T) {
	base := dt2bvFixture(t, false)
	c, m := base.context, base.manager
	p, err := c.NewValue()
	sharingMust(t, err)
	t.Cleanup(func() { sharingMust(t, c.Del(&p)) })
	key := "dt-constructor-snapshot"
	sharingMust(t, c.SetUIntName(&p, &key, 17))
	baseline := m.Budget().Used()
	refs := c.Stats().References
	complete := false
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		m.Budget().FailAfter(ordinal)
		r, err := NewDT2BVTactic(m, c, &p, base.services)
		m.Budget().FailAfter(-1)
		if err != nil && !errors.Is(err, arena.ErrInjectedFailure) {
			t.Fatal("unexpected constructor error", ordinal, err)
		}
		if r != nil {
			sharingMust(t, r.Close())
		}
		if m.Budget().Used() != baseline || c.Stats().References != refs {
			t.Fatal("constructor prefix retained owner", ordinal, m.Budget().Used(), baseline, c.Stats())
		}
		if err == nil {
			complete = true
			break
		}
	}
	if !complete {
		t.Fatal("constructor allocation census did not finish")
	}
	r, err := NewDT2BVTactic(m, c, &p, base.services)
	sharingMust(t, err)
	t.Cleanup(func() { sharingMust(t, r.Close()) })
	if r.parameters != p || c.Stats().References != refs+1 {
		t.Fatal("constructor lost shared parameter reference")
	}
	sharingMust(t, c.SetUIntName(&p, &key, 29))
	sharingMust(t, r.UpdateParams(&p))
	value, err := c.GetUIntName(&r.parameters, &key, 0)
	sharingMust(t, err)
	if value != 17 {
		t.Fatal("empty source UpdateParams replaced constructor snapshot", value)
	}
}
