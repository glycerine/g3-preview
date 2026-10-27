// Source: src/smt/theory_bv.cpp:1506-1552.
package smtcore

import "github.com/glycerine/g3/pkg/ast"

// Source region storage borrows the two enodes until the logical scope pops.
type smtBVBitEqualityJustification struct {
	SMTJustificationBase
	first, second          *SMTEnode
	theory                 SMTTheoryID
	consequent, antecedent SMTLiteral
}

func (*smtBVBitEqualityJustification) Name() string              { return "bv-bit-eq" }
func (j *smtBVBitEqualityJustification) FromTheory() SMTTheoryID { return j.theory }
func (j *smtBVBitEqualityJustification) MarkAntecedents(marker SMTAntecedentMarker) error {
	if marker == nil {
		return ast.ErrASTHandle
	}
	if err := marker.MarkEquality(j.first, j.second); err != nil {
		return err
	}
	if j.antecedent.Var() != SMTTrueBoolVar {
		return marker.MarkLiteral(j.antecedent)
	}
	return nil
}

func (b *smtConflictProofBuilder) bvBitEqualityProof(j *smtBVBitEqualityJustification) (ast.ASTHandle, error) {
	parents := make([]ast.ASTHandle, 0, 2)
	visited := true
	equality, err := b.equalityProof(j.first, j.second)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if equality.IsNull() {
		visited = false
	} else {
		parents = append(parents, equality)
	}
	if j.antecedent.Var() != SMTTrueBoolVar {
		antecedent, err := b.literalProof(j.antecedent)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		if antecedent.IsNull() {
			visited = false
		} else {
			parents = append(parents, antecedent)
		}
	}
	if !visited {
		return ast.ASTHandle{}, nil
	}
	fact, err := b.literalExpression(j.consequent)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	return b.state.manager.MkProofTheoryLemma(ast.FamilyID(j.theory), fact, parents, nil)
}
