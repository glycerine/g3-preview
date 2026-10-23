// Source: theory_array var_data, private union-find, parent registration,
// merge_eh, set_prop_upward, relevant_eh and assert_delayed_axioms.
package smtcore

import (
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/backtrack"
)

type smtArrayVarData struct {
	stores, parentStores, parentSelects []*SMTEnode
	propUpward                          bool
}

type smtArrayUnionContext struct{ theory *SMTArrayTheory }

func (c *smtArrayUnionContext) GetTrailStack() *backtrack.TrailStack      { return c.theory.arrayTrail }
func (*smtArrayUnionContext) UnmergeEH(uint32, uint32)                    {}
func (*smtArrayUnionContext) AfterMergeEH(uint32, uint32, uint32, uint32) {}
func (c *smtArrayUnionContext) MergeEH(root, child, _, _ uint32) {
	if c.theory.arrayMergeError == nil {
		c.theory.arrayMergeError = c.theory.mergeArrayClasses(root, child)
	}
}

func (t *SMTArrayTheory) Setup() error {
	if t.schedulingReady {
		return nil
	}
	s, err := t.State()
	if err != nil {
		return err
	}
	if s.relevancy == nil || !s.relevancy.enabled {
		t.arrayConfig.Laziness = 0
		if t.parameterOwner != nil {
			t.parameterOwner.state.Laziness = 0
		}
	}
	t.schedulingReady = true
	return nil
}

func (t *SMTArrayTheory) CreateVariable(node *SMTEnode) (SMTTheoryVar, error) {
	variable, err := t.SMTTheoryBase.CreateVariable(node)
	if err != nil {
		return variable, err
	}
	if t.arrayUnion.MkVar() != uint32(variable) {
		return SMTNullTheoryVar, ast.ErrASTDomain
	}
	data := &smtArrayVarData{}
	t.arrayData = append(t.arrayData, data)
	op, array, err := t.arrayOperation(node.owner)
	if err != nil {
		return variable, err
	}
	if array && op == ast.ASTArrayOpStore {
		data.stores = append(data.stores, node)
		if err = t.assertStoreAxiom(node.owner); err != nil {
			return variable, err
		}
	}
	return variable, nil
}

func (t *SMTArrayTheory) addParentSelect(variable SMTTheoryVar, read *SMTEnode) error {
	if variable < 0 || int(variable) >= len(t.arrayData) {
		return ast.ErrASTDomain
	}
	data := t.arrayData[t.arrayUnion.Find(uint32(variable))]
	data.parentSelects = append(data.parentSelects, read)
	if err := t.arrayTrail.PushUndo(func() { data.parentSelects = data.parentSelects[:len(data.parentSelects)-1] }); err != nil {
		return err
	}
	if !t.activeParentSelects[read.owner] {
		t.activeParentSelects[read.owner] = true
		if err := t.arrayTrail.PushUndo(func() { delete(t.activeParentSelects, read.owner) }); err != nil {
			return err
		}
	}
	for _, store := range data.stores {
		if err := t.enqueueStoreReadAxiom(read.owner, store.owner); err != nil {
			return err
		}
	}
	// Full constructor lists are still represented by the existing typed
	// observer machinery. Store parents above use the private array classes.
	if !t.simple {
		array := read.arguments[0]
		for i := 0; i < len(t.variables); i++ {
			candidate := t.variables[i]
			if candidate == nil || candidate.root != array.root {
				continue
			}
			op, belongs, err := t.arrayOperation(candidate.owner)
			if err != nil {
				return err
			}
			if !belongs || op == ast.ASTArrayOpStore {
				continue
			}
			if err := t.assertSelectAxiom(read.owner, candidate.owner); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *SMTArrayTheory) addParentStore(variable SMTTheoryVar, store *SMTEnode) error {
	if variable < 0 || int(variable) >= len(t.arrayData) {
		return ast.ErrASTDomain
	}
	data := t.arrayData[t.arrayUnion.Find(uint32(variable))]
	data.parentStores = append(data.parentStores, store)
	return t.arrayTrail.PushUndo(func() { data.parentStores = data.parentStores[:len(data.parentStores)-1] })
}

func (t *SMTArrayTheory) setStoreBaseUpward(store *SMTEnode) error {
	variable := store.arguments[0].TheoryVar(t.FamilyID())
	if variable < 0 {
		return ast.ErrASTDomain
	}
	return t.setArrayUpward(uint32(variable))
}

func (t *SMTArrayTheory) setArrayUpward(variable uint32) error {
	if t.arrayConfig.Weak {
		return nil
	}
	data := t.arrayData[t.arrayUnion.Find(variable)]
	if data.propUpward {
		return nil
	}
	data.propUpward = true
	if err := t.arrayTrail.PushUndo(func() { data.propUpward = false }); err != nil {
		return err
	}
	for _, store := range data.stores {
		if err := t.setStoreBaseUpward(store); err != nil {
			return err
		}
	}
	return nil
}

func (t *SMTArrayTheory) addArrayStore(variable uint32, store *SMTEnode) error {
	data := t.arrayData[t.arrayUnion.Find(variable)]
	for _, previous := range data.stores {
		if err := t.setStoreBaseUpward(previous); err != nil {
			return err
		}
	}
	data.stores = append(data.stores, store)
	if err := t.arrayTrail.PushUndo(func() { data.stores = data.stores[:len(data.stores)-1] }); err != nil {
		return err
	}
	for _, read := range data.parentSelects {
		if err := t.enqueueStoreReadAxiom(read.owner, store.owner); err != nil {
			return err
		}
	}
	return t.setStoreBaseUpward(store)
}

func (t *SMTArrayTheory) mergeArrayClasses(root, child uint32) error {
	first, second := t.arrayData[root], t.arrayData[child]
	if !first.propUpward && second.propUpward {
		if err := t.setArrayUpward(root); err != nil {
			return err
		}
	}
	for _, store := range second.stores {
		if err := t.addArrayStore(root, store); err != nil {
			return err
		}
	}
	for _, store := range second.parentStores {
		if err := t.addParentStore(SMTTheoryVar(root), store); err != nil {
			return err
		}
	}
	for _, read := range second.parentSelects {
		if err := t.addParentSelect(SMTTheoryVar(root), read); err != nil {
			return err
		}
	}
	return nil
}

func (t *SMTArrayTheory) Relevant(expression ast.ASTHandle) error {
	op, array, err := t.arrayOperation(expression)
	if err != nil || !array {
		return err
	}
	s, err := t.State()
	if err != nil {
		return err
	}
	node := s.context.enodes[expression]
	if node == nil {
		return nil
	}
	if t.arrayConfig.Laziness != 0 && (op == ast.ASTArrayOpStore || op == ast.ASTArrayOpSelect) {
		variable := node.arguments[0].TheoryVar(t.FamilyID())
		if op == ast.ASTArrayOpStore {
			err = t.addParentStore(variable, node)
		} else {
			err = t.addParentSelect(variable, node)
		}
		if err != nil {
			return err
		}
	}
	if !t.simple && op == ast.ASTArrayOpSelect {
		return t.addParentSelect(node.arguments[0].TheoryVar(t.FamilyID()), node)
	}
	return nil
}

func (t *SMTArrayTheory) assertDelayedStoreAxioms() (bool, error) {
	if !t.arrayConfig.DelayExpAxiom {
		return false, nil
	}
	progress := false
	for _, data := range t.arrayData {
		if !data.propUpward {
			continue
		}
		for _, store := range data.parentStores {
			for _, read := range data.parentSelects {
				added, err := t.queueStoreReadAxiom(read.owner, store.owner, true)
				if err != nil {
					return false, err
				}
				progress = progress || added
			}
		}
	}
	return progress, nil
}
