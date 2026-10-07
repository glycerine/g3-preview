package rewriter

import (
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/sat"
)

// The model package legitimately imports rewriter. Internal rewriter tests
// therefore cannot import model's SAT tactic consumer. This test-only seam
// lets the external test package supply the real translator while preserving
// the existing internal source-structure assertions and CDCL truth tables.
type TestGoalTranslator interface {
	Compile([]ast.ASTHandle) error
	Close() error
}

var TestNewGoalTranslator func(*ast.ASTManager, *sat.Solver) (TestGoalTranslator, error)
