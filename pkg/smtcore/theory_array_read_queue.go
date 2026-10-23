// Source: theory_array_base::assert_store_axiom2 and assert_store_axiom2_core.
package smtcore

import "github.com/glycerine/g3/pkg/ast"

type smtArrayReadFingerprint struct {
	store   ast.ASTHandle
	indices []*SMTEnode
}

func (t *SMTArrayTheory) enqueueStoreReadAxiom(read, store ast.ASTHandle) error {
	_, err := t.queueStoreReadAxiom(read, store, false)
	return err
}

func (t *SMTArrayTheory) queueStoreReadAxiom(read, store ast.ASTHandle, upward bool) (bool, error) {
	s, err := t.State()
	if err != nil {
		return false, err
	}
	readNode, storeNode := s.context.enodes[read], s.context.enodes[store]
	if readNode == nil || storeNode == nil || len(readNode.arguments) < 2 || len(storeNode.arguments) != len(readNode.arguments)+1 {
		return false, ast.ErrASTDomain
	}
	allEqual := true
	indices := make([]*SMTEnode, len(readNode.arguments)-1)
	for i := range indices {
		indices[i] = readNode.arguments[i+1]
		allEqual = allEqual && indices[i].root == storeNode.arguments[i+1].root
	}
	if allEqual {
		return false, nil
	}
	if !t.insertArrayReadFingerprint(store, indices) {
		return false, nil
	}
	t.readTodo = append(t.readTodo, smtArrayAxiomPair{observer: read, array: store})
	if upward {
		t.statistics.expandedAxiom2++
	} else {
		t.statistics.axiom2++
	}
	return true, nil
}

// Source uses the same constructor/indices fingerprint shape for stores and
// maps. The concrete AST identity distinguishes those retained operations.
func (t *SMTArrayTheory) insertArrayReadFingerprint(constructor ast.ASTHandle, indices []*SMTEnode) bool {
	// fingerprint_set::insert checks the original tuple before replacing it
	// with current roots. An older fingerprint can still name a former root.
	for _, entry := range t.readFingerprints[constructor] {
		if smtFingerprintArgsEqual(entry.indices, indices) {
			return false
		}
	}
	roots := make([]*SMTEnode, len(indices))
	for i, node := range indices {
		roots[i] = node.root
	}
	for _, entry := range t.readFingerprints[constructor] {
		if smtFingerprintArgsEqual(entry.indices, roots) {
			return false
		}
	}
	entry := &smtArrayReadFingerprint{store: constructor, indices: roots}
	t.readFingerprints[constructor] = append(t.readFingerprints[constructor], entry)
	t.readFingerprintTrail = append(t.readFingerprintTrail, entry)
	return true
}

func (t *SMTArrayTheory) assertStoreReadAxiomCore(read, store ast.ASTHandle) error {
	s, err := t.State()
	if err != nil {
		return err
	}
	readNode, storeNode := s.context.enodes[read], s.context.enodes[store]
	var consequence SMTLiteral
	var consequenceExpr ast.ASTHandle
	initialized := false
	for i := 1; i < len(readNode.arguments); i++ {
		left, right := storeNode.arguments[i], readNode.arguments[i]
		if left.root == right.root {
			continue
		}
		if !initialized {
			// Source uses select(store, indices), even when the observer is
			// a read of an equal alias or of the store's base (upward rule).
			args := []ast.ASTHandle{store}
			for _, index := range readNode.arguments[1:] {
				args = append(args, index.owner)
			}
			selected, err := s.manager.MkFamilyApp(t.FamilyID(), ast.ASTArrayOpSelect, nil, args)
			if err != nil {
				return err
			}
			args[0] = storeNode.arguments[0].owner
			fallback, err := s.manager.MkFamilyApp(t.FamilyID(), ast.ASTArrayOpSelect, nil, args)
			if err != nil {
				return err
			}
			if selected == fallback {
				break
			}
			consequenceExpr, err = s.mkEqualityAtom(selected, fallback)
			if err != nil {
				return err
			}
			if err = s.Internalize(consequenceExpr, true); err != nil {
				return err
			}
			consequence, err = s.Literal(consequenceExpr)
			if err != nil {
				return err
			}
			initialized = true
		}
		antecedent, err := t.MkEqualityLiteral(left.owner, right.owner, true)
		if err != nil {
			return err
		}
		if antecedent.Var() >= SMTFirstBoolVar {
			atom := s.boolExpr[antecedent.Var()]
			if err = s.markRelevant(atom); err != nil {
				return err
			}
			if s.relevancy != nil {
				if err = s.relevancy.AddWatchTarget(atom, antecedent.Sign(), consequenceExpr); err != nil {
					return err
				}
			}
		} else if antecedent == SMTFalseLiteral {
			if err = s.markRelevant(consequenceExpr); err != nil {
				return err
			}
		}
		if err = s.addTheoryAxiom(t.FamilyID(), []SMTLiteral{antecedent, consequence}); err != nil {
			return err
		}
	}
	return nil
}
