package smtcore

import (
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestSMTGrobnerDependenciesOwnLiteralPolarityAndConsumerLifetime(t *testing.T) {
	m, cfg := smtModelGeneratorFixture(t)
	s, err := NewSMTSolver(m, nil, "QF_LRA", SMTSolverConfig{Params: cfg.Context, ModelEvaluator: cfg})
	astTestMust(t, err)
	// Exercise the retained legacy tableau algorithm explicitly.
	smtDisableAutoConfig(t, s)
	t.Cleanup(func() { astTestMust(t, s.Close()) })
	s.kernel.preprocess = false
	family, err := m.ArithmeticFamilyID()
	astTestMust(t, err)
	sort, err := m.MkFamilySort(family, 0, nil)
	astTestMust(t, err)
	x, err := m.MkConst(astTestSymbol(t, m, "gb-dependency-lifetime"), sort)
	astTestMust(t, err)
	one, err := m.ArithmeticRealNumeralText("1")
	astTestMust(t, err)
	bound, err := m.MkFamilyApp(family, smtArithOpGE, nil, []ast.ASTHandle{x, one})
	astTestMust(t, err)
	astTestMust(t, s.Assert(bound))
	status, err := s.CheckSat(nil)
	astTestMust(t, err)
	if status != lbool.True {
		t.Fatal(status)
	}
	theory := s.kernel.internalizer.theories[family].(*SMTLinearArithmeticTheory)
	literal, err := theory.state.Literal(bound)
	astTestMust(t, err)
	dependencies := smtGrobnerDependencies{theory: theory}
	root, err := dependencies.add(ast.ASTDependency{}, literal)
	astTestMust(t, err)
	root, err = dependencies.add(root, literal.Negated())
	astTestMust(t, err)
	// An equation may outlive the temporary builder's ownership.
	astTestMust(t, m.DependencyInc(root))
	astTestMust(t, dependencies.close())
	literals, err := dependencies.literals(root)
	astTestMust(t, err)
	if len(literals) != 2 {
		t.Fatal(literals)
	}
	seen := map[SMTLiteral]bool{}
	for _, got := range literals {
		seen[got] = true
	}
	if !seen[literal] || !seen[literal.Negated()] {
		t.Fatal("dependency lost literal polarity", literals)
	}
	astTestMust(t, m.DependencyDec(root))
}
