// Source: src/ast/proofs/proof_utils.cpp:374-420,468-584,683-727.
package proofs

import (
	"errors"

	"github.com/glycerine/g3/pkg/ast"
)

type hypothesisReducer struct {
	manager *ast.ASTManager
	sets    *hypothesisSets
	cache   map[ast.ASTHandle]ast.ASTHandle
	refs    []ast.ASTRef
}

func (r *hypothesisReducer) pin(h ast.ASTHandle) error {
	ref, err := r.manager.NewRef(h)
	if err != nil {
		return err
	}
	r.refs = append(r.refs, ref)
	return nil
}

func (r *hypothesisReducer) close() error {
	var result error
	for i := range r.refs {
		result = errors.Join(result, r.manager.DelRef(&r.refs[i]))
	}
	r.sets.reset()
	clear(r.cache)
	return result
}

func (r *hypothesisReducer) complement(e ast.ASTHandle) (ast.ASTHandle, ast.ASTRef, error) {
	yes, child, err := r.manager.IsNot(e)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTRef{}, err
	}
	if yes {
		ref, e := r.manager.NewRef(child)
		return child, ref, e
	}
	other, err := r.manager.MkNot(e)
	if err != nil {
		return ast.ASTHandle{}, ast.ASTRef{}, err
	}
	ref, err := r.manager.NewRef(other)
	return other, ref, err
}

func (r *hypothesisReducer) inHypotheses(e ast.ASTHandle, set map[ast.ASTHandle]struct{}) (bool, error) {
	if set == nil {
		return false, nil
	}
	other, ref, err := r.complement(e)
	if err != nil {
		return false, err
	}
	_, found := set[other]
	return found, r.manager.DelRef(&ref)
}

func (r *hypothesisReducer) literals(clause ast.ASTHandle) ([]ast.ASTHandle, error) {
	kind, err := r.manager.GetKind(clause)
	if err != nil {
		return nil, err
	}
	if kind == ast.ASTApp {
		family, e := r.manager.GetFamilyID(clause)
		if e != nil {
			return nil, e
		}
		op, e := r.manager.GetDeclKind(clause)
		if e != nil {
			return nil, e
		}
		if family == r.manager.BasicFamilyID() && op == ast.ASTOpOr {
			n, e := r.manager.GetNumArgs(clause)
			if e != nil {
				return nil, e
			}
			result := make([]ast.ASTHandle, 0, n)
			for i := uint32(0); i < n; i++ {
				arg, e := r.manager.GetArg(clause, i)
				if e != nil {
					return nil, e
				}
				result = append(result, arg)
			}
			return result, nil
		}
	}
	return []ast.ASTHandle{clause}, nil
}

func (r *hypothesisReducer) lemma(p ast.ASTHandle) (ast.ASTHandle, error) {
	n, err := r.manager.GetProofNumParents(p)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	if n != 1 {
		return ast.ASTHandle{}, ast.ErrASTDomain
	}
	parent, err := r.manager.GetProofParent(p, 0)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	child, err := r.elim(parent)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	hyps := r.sets.byProof[child]
	var remaining map[ast.ASTHandle]struct{}
	if hyps != nil {
		remaining = make(map[ast.ASTHandle]struct{}, len(hyps))
		for fact := range hyps {
			remaining[fact] = struct{}{}
		}
	}
	fact, err := r.manager.GetProofFact(p)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	whole, err := r.inHypotheses(fact, hyps)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	var literals []ast.ASTHandle
	if hyps != nil && len(hyps) == 1 && whole {
		literals = []ast.ASTHandle{fact}
	} else {
		literals, err = r.literals(fact)
		if err != nil {
			return ast.ASTHandle{}, err
		}
	}
	for i := 0; i < len(literals); {
		found, e := r.inHypotheses(literals[i], hyps)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		if !found {
			literals[i] = literals[len(literals)-1]
			literals = literals[:len(literals)-1]
			continue
		}
		complement, ref, e := r.complement(literals[i])
		if e != nil {
			return ast.ASTHandle{}, e
		}
		delete(remaining, complement)
		if e = r.manager.DelRef(&ref); e != nil {
			return ast.ASTHandle{}, e
		}
		i++
	}
	result := child
	if len(literals) != 0 {
		clause := literals[0]
		if len(literals) > 1 {
			clause, err = r.manager.MkOrArray(literals)
			if err != nil {
				return ast.ASTHandle{}, err
			}
			if err = r.pin(clause); err != nil {
				return ast.ASTHandle{}, err
			}
		}
		result, err = r.manager.MkProofLemma(child, clause)
		if err != nil {
			return ast.ASTHandle{}, err
		}
		if err = r.pin(result); err != nil {
			return ast.ASTHandle{}, err
		}
	}
	if len(remaining) == 0 {
		remaining = nil
	}
	r.sets.byProof[result] = remaining
	return result, nil
}

func (r *hypothesisReducer) elim(p ast.ASTHandle) (ast.ASTHandle, error) {
	if result, found := r.cache[p]; found {
		return result, nil
	}
	hypothesis, err := r.manager.IsProofKind(p, ast.ASTPrHypothesis)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	lemma, err := r.manager.IsProofKind(p, ast.ASTPrLemma)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	unit, err := r.manager.IsProofKind(p, ast.ASTPrUnitResolution)
	if err != nil {
		return ast.ASTHandle{}, err
	}
	result := p
	switch {
	case hypothesis:
		err = r.sets.add(result)
	case lemma:
		result, err = r.lemma(p)
	case unit:
		return ast.ASTHandle{}, ast.ErrASTPending
	default:
		var changed bool
		args := make([]ast.ASTHandle, 0)
		n, e := r.manager.GetProofNumParents(p)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		for i := uint32(0); i < n; i++ {
			parent, e := r.manager.GetProofParent(p, i)
			if e != nil {
				return ast.ASTHandle{}, e
			}
			child, e := r.elim(parent)
			if e != nil {
				return ast.ASTHandle{}, e
			}
			fact, e := r.manager.GetProofFact(child)
			if e != nil {
				return ast.ASTHandle{}, e
			}
			if r.manager.IsFalse(fact) {
				result = child
				goto done
			}
			changed = changed || child != parent
			args = append(args, child)
		}
		hasFact, e := r.manager.ProofHasFact(p)
		if e != nil {
			return ast.ASTHandle{}, e
		}
		if hasFact {
			fact, e := r.manager.GetProofFact(p)
			if e != nil {
				return ast.ASTHandle{}, e
			}
			args = append(args, fact)
		}
		if changed {
			decl, e := r.manager.GetDecl(p)
			if e != nil {
				return ast.ASTHandle{}, e
			}
			result, e = r.manager.MkApp(decl, args)
			if e != nil {
				return ast.ASTHandle{}, e
			}
			if e = r.pin(result); e != nil {
				return ast.ASTHandle{}, e
			}
		}
		err = r.sets.add(result)
	}
done:
	if err != nil {
		return ast.ASTHandle{}, err
	}
	r.cache[p] = result
	return result, nil
}

// ReduceHypothesesNoUnits ports the source hypothesis, lemma and ordinary
// proof branches. Unit-resolution rewriting remains an explicit pending path.
func ReduceHypothesesNoUnits(m *ast.ASTManager, proof ast.ASTRef) (err error) {
	if m == nil {
		return ast.ErrASTHandle
	}
	root, err := m.RefGet(proof)
	if err != nil {
		return err
	}
	r := &hypothesisReducer{manager: m, sets: newHypothesisSets(m), cache: make(map[ast.ASTHandle]ast.ASTHandle)}
	defer func() { err = errors.Join(err, r.close()) }()
	result, err := r.elim(root)
	if err != nil {
		return err
	}
	return m.SetRef(proof, result)
}
