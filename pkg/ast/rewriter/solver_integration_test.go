package rewriter_test

import (
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/rewriter"
	"github.com/glycerine/g3/pkg/sat"
	sattactic "github.com/glycerine/g3/pkg/sat/tactic"
)

// All test dependencies initialize before any test executes, including when
// -run selects an internal source test. The adapter always uses the real tactic.
func init() {
	rewriter.TestNewGoalTranslator = func(m *ast.ASTManager, s *sat.Solver) (rewriter.TestGoalTranslator, error) {
		return sattactic.NewGoalTranslator(m, s)
	}
}
