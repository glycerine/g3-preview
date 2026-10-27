package smtcore

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTPBModelInitializerRegistersFactoryWithoutValueBuilder(t *testing.T) {
	m, config := smtModelGeneratorFixture(t)
	family, err := m.RegisterPseudoBooleanPlugin()
	astTestMust(t, err)
	context, err := NewSMTCheckerContext(m)
	astTestMust(t, err)
	state, err := NewSMTInternalizerState(m, context, nil)
	astTestMust(t, err)
	t.Cleanup(func() { astTestMust(t, state.Close()) })
	pb, err := NewSMTPseudoBooleanTheory(family)
	astTestMust(t, err)
	astTestMust(t, state.RegisterTheory(pb))
	var theory SMTTheoryInternalizer = pb
	if _, builder := theory.(SMTModelValueTheory); builder {
		t.Fatal("source build_models=false theory acquired an unused value procedure")
	}
	aux, err := m.MkFreshPseudoBooleanAux()
	astTestMust(t, err)
	astTestMust(t, state.Internalize(aux, true))
	lit, err := state.Literal(aux)
	astTestMust(t, err)
	astTestMust(t, state.SetAssignment(lit.Var(), lbool.True))
	node, err := state.MkEnode(aux, false, false, true)
	astTestMust(t, err)
	astTestMust(t, context.SetRelevant(aux, true))
	g, err := NewSMTModelGenerator(m)
	astTestMust(t, err)
	t.Cleanup(func() {
		if !g.closed {
			astTestMust(t, g.Close())
		}
	})
	astTestMust(t, g.SetContext(context))
	astTestMust(t, g.InitializeModel(config))
	astTestMust(t, g.BuildStateValues(state))
	factory, ok := g.model.Factory(family).(*smtPBValueFactory)
	if !ok || factory.manager != m {
		t.Fatal("source init_model skipped PB factory registration")
	}
	value, err := g.Value(node)
	astTestMust(t, err)
	if !m.IsTrue(value) {
		t.Fatal("PB Boolean root bypassed its assigned value")
	}
	a, err := factory.SomeValue(ast.ASTHandle{})
	astTestMust(t, err)
	first, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&first)) }()
	second, err := m.NewRef(ast.ASTHandle{})
	astTestMust(t, err)
	defer func() { astTestMust(t, m.DelRef(&second)) }()
	ok, err = factory.SomeValues(ast.ASTHandle{}, first, second)
	astTestMust(t, err)
	b, err := m.RefGet(first)
	astTestMust(t, err)
	c, err := m.RefGet(second)
	astTestMust(t, err)
	if !m.IsTrue(a) || !m.IsTrue(b) || !m.IsFalse(c) || !ok {
		t.Fatal("PB factory source constants")
	}
	fresh, err := factory.FreshValue(ast.ASTHandle{})
	astTestMust(t, err)
	if !fresh.IsNull() {
		t.Fatal("source PB factory invented a fresh value")
	}
	astTestMust(t, factory.RegisterValue(ast.ASTHandle{}))
	astTestMust(t, g.Close())
	if factory.manager != nil {
		t.Fatal("model closure retained factory manager borrow")
	}
	// The erased source operations do not dereference the factory owner.
	fresh, err = factory.FreshValue(ast.ASTHandle{})
	astTestMust(t, err)
	if !fresh.IsNull() {
		t.Fatal("erased factory operation changed")
	}
	if _, err = factory.SomeValue(ast.ASTHandle{}); !errors.Is(err, ast.ErrASTHandle) {
		t.Fatal("closed factory manager access", err)
	}
}

func TestSMTPBModelInitializerGuardsAndIndependentFactories(t *testing.T) {
	m, config := smtModelGeneratorFixture(t)
	family, err := m.RegisterPseudoBooleanPlugin()
	astTestMust(t, err)
	context, _, state := smtInternalizerFixture(t, m, nil)
	pb, err := NewSMTPseudoBooleanTheory(family)
	astTestMust(t, err)
	astTestMust(t, state.RegisterTheory(pb))
	if err = pb.InitializeModel(nil); !errors.Is(err, ast.ErrASTDomain) {
		t.Fatal("nil model initializer", err)
	}
	var factories []*smtPBValueFactory
	for repeat := 0; repeat < 2; repeat++ {
		g, err := NewSMTModelGenerator(m)
		astTestMust(t, err)
		astTestMust(t, g.SetContext(context))
		if err = pb.InitializeModel(g); !errors.Is(err, ast.ErrASTDomain) {
			t.Fatal("missing proto-model accepted", err)
		}
		astTestMust(t, g.InitializeModel(config))
		astTestMust(t, pb.InitializeModel(g))
		factories = append(factories, g.model.Factory(family).(*smtPBValueFactory))
		if err = pb.InitializeModel(g); !errors.Is(err, ast.ErrASTDomain) {
			t.Fatal("duplicate factory registration", err)
		}
		astTestMust(t, g.Close())
	}
	if factories[0] == factories[1] || factories[0].manager != nil || factories[1].manager != nil {
		t.Fatal("models shared or retained a factory")
	}
}
