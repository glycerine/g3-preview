// Source: theory_array_full mk_var, parent maps, constructor merges,
// default initialization and delayed select-map propagation.
package smtcore

import "github.com/glycerine/g3/pkg/ast"

func (t *SMTArrayTheory) initializeFullArrayVariable(node *SMTEnode, data *smtArrayVarData, op int32) error {
	switch op {
	case ast.ASTArrayOpConst:
		data.consts = append(data.consts, node)
	case ast.ASTArrayOpMap:
		data.maps = append(data.maps, node)
	case ast.ASTArrayOpAsArray:
		data.asArrays = append(data.asArrays, node)
	default:
		return nil
	}
	s, err := t.State()
	if err != nil {
		return err
	}
	defaultTerm, err := s.manager.MkFamilyApp(t.FamilyID(), ast.ASTArrayOpDefault, nil, []ast.ASTHandle{node.owner})
	if err != nil {
		return err
	}
	return s.Internalize(defaultTerm, false)
}

func (t *SMTArrayTheory) markArrayEqualityRelevant(literal SMTLiteral) error {
	if literal.Var() < SMTFirstBoolVar {
		return nil
	}
	s, err := t.State()
	if err != nil {
		return err
	}
	return s.markRelevant(s.boolExpr[literal.Var()])
}

func (t *SMTArrayTheory) addParentMap(variable SMTTheoryVar, mapped *SMTEnode) error {
	if variable < 0 || int(variable) >= len(t.arrayData) {
		return ast.ErrASTDomain
	}
	data := t.arrayData[t.arrayUnion.Find(uint32(variable))]
	data.parentMaps = append(data.parentMaps, mapped)
	return t.arrayTrail.PushUndo(func() { data.parentMaps = data.parentMaps[:len(data.parentMaps)-1] })
}

func (t *SMTArrayTheory) setMapInputsUpward(mapped *SMTEnode) error {
	for _, input := range mapped.arguments {
		variable := input.TheoryVar(t.FamilyID())
		if variable < 0 {
			return ast.ErrASTDomain
		}
		if err := t.setArrayUpward(uint32(variable)); err != nil {
			return err
		}
	}
	return nil
}

func (t *SMTArrayTheory) instantiateMapRead(observer, mapped *SMTEnode) (bool, error) {
	s, err := t.State()
	if err != nil {
		return false, err
	}
	indices := make([]*SMTEnode, len(observer.arguments)-1)
	for i := range indices {
		indices[i] = observer.arguments[i+1]
	}
	if !t.insertArrayReadFingerprint(mapped.owner, indices) {
		return false, nil
	}
	args := []ast.ASTHandle{mapped.owner}
	for _, index := range observer.arguments[1:] {
		args = append(args, index.owner)
	}
	read, err := s.manager.MkFamilyApp(t.FamilyID(), ast.ASTArrayOpSelect, nil, args)
	if err != nil {
		return false, err
	}
	before := t.statistics.mapAxiom
	if err := t.assertSelectAxiom(read, mapped.owner); err != nil {
		return false, err
	}
	return before != t.statistics.mapAxiom, nil
}

func (t *SMTArrayTheory) mergeFullArrayClasses(root uint32, second *smtArrayVarData) error {
	first := t.arrayData[root]
	for _, mapped := range second.maps {
		for _, store := range first.stores {
			if err := t.setStoreBaseUpward(store); err != nil {
				return err
			}
		}
		first.maps = append(first.maps, mapped)
		if err := t.arrayTrail.PushUndo(func() { first.maps = first.maps[:len(first.maps)-1] }); err != nil {
			return err
		}
		for _, read := range first.parentSelects {
			if _, err := t.instantiateMapRead(read, mapped); err != nil {
				return err
			}
		}
		if err := t.setMapInputsUpward(mapped); err != nil {
			return err
		}
	}
	for _, mapped := range second.parentMaps {
		if err := t.addParentMap(SMTTheoryVar(root), mapped); err != nil {
			return err
		}
	}
	for _, node := range second.consts {
		for _, store := range first.stores {
			if err := t.setStoreBaseUpward(store); err != nil {
				return err
			}
		}
		first.consts = append(first.consts, node)
		if err := t.arrayTrail.PushUndo(func() { first.consts = first.consts[:len(first.consts)-1] }); err != nil {
			return err
		}
		for _, read := range first.parentSelects {
			if err := t.assertSelectAxiom(read.owner, node.owner); err != nil {
				return err
			}
		}
	}
	for _, node := range second.asArrays {
		for _, store := range first.stores {
			if err := t.setStoreBaseUpward(store); err != nil {
				return err
			}
		}
		first.asArrays = append(first.asArrays, node)
		if err := t.arrayTrail.PushUndo(func() { first.asArrays = first.asArrays[:len(first.asArrays)-1] }); err != nil {
			return err
		}
		for _, read := range first.parentSelects {
			if err := t.assertSelectAxiom(read.owner, node.owner); err != nil {
				return err
			}
		}
	}
	return nil
}
