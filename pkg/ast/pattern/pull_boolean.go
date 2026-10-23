// Source: src/ast/normal_forms/pull_quant.cpp no-proof consumer.
package pattern

import (
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/normal_forms"
)

func (c *patternCandidates) pullNoProof(root ast.ASTHandle) (ast.ASTRef, error) {
	return normal_forms.PullQuantifierNoProof(c.manager, root, c.traceTags)
}
