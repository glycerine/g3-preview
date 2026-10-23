// Source: theory_array_base model default partition, collect_selects,
// propagate_selects and array_value_proc preparation.
package smtcore

import (
	"fmt"
	"github.com/glycerine/g3/pkg/ast"
)

type smtArrayModelElse struct {
	expression ast.ASTHandle
	fresh      *SMTExtraFreshValue
}

type smtArrayModelData struct {
	parents       []int
	defaults      []*SMTEnode
	elseValues    []smtArrayModelElse
	unspecified   bool
	selects       map[*SMTEnode][]*SMTEnode
	selectDomains []*SMTEnode
}

func (d *smtArrayModelData) find(variable int) int {
	root := variable
	for d.parents[root] >= 0 {
		root = d.parents[root]
	}
	for d.parents[variable] >= 0 {
		next := d.parents[variable]
		d.parents[variable] = root
		variable = next
	}
	return root
}

func (d *smtArrayModelData) merge(first, second int) {
	first, second = d.find(first), d.find(second)
	if first == second {
		return
	}
	if d.parents[first] > d.parents[second] {
		first, second = second, first
	}
	d.parents[first] += d.parents[second]
	d.parents[second] = first
	if d.defaults[first] == nil {
		d.defaults[first] = d.defaults[second]
	}
}

func (t *SMTArrayTheory) modelRepresentative(node *SMTEnode) (int, error) {
	variable := node.root.TheoryVar(t.FamilyID())
	if variable < 0 || int(variable) >= len(t.variables) {
		return 0, ast.ErrASTDomain
	}
	return int(variable), nil
}

func (t *SMTArrayTheory) prepareArrayModel(generator *SMTModelGenerator) error {
	d := &smtArrayModelData{
		parents: make([]int, len(t.variables)), defaults: make([]*SMTEnode, len(t.variables)),
		elseValues: make([]smtArrayModelElse, len(t.variables)), unspecified: true,
		selects: make(map[*SMTEnode][]*SMTEnode),
	}
	t.modelData = d
	for i := range d.parents {
		d.parents[i] = -1
	}
	for _, node := range t.variables {
		if !generator.context.isRelevant(node.owner) {
			continue
		}
		op, array, err := t.arrayOperation(node.owner)
		if err != nil {
			return err
		}
		if array && (op == ast.ASTArrayOpStore || op == ast.ASTArrayOpConst || op == ast.ASTArrayOpDefault) {
			d.unspecified = false
			break
		}
	}
	if !d.unspecified {
		for variable, node := range t.variables {
			if !generator.context.isRelevant(node.owner) {
				continue
			}
			representative, err := t.modelRepresentative(node)
			if err != nil {
				return err
			}
			d.merge(variable, representative)
			op, array, err := t.arrayOperation(node.owner)
			if err != nil {
				return err
			}
			if !array {
				continue
			}
			switch op {
			case ast.ASTArrayOpStore:
				base, err := t.modelRepresentative(node.arguments[0])
				if err != nil {
					return err
				}
				d.merge(variable, base)
			case ast.ASTArrayOpConst:
				root := d.find(variable)
				if d.defaults[root] == nil {
					d.defaults[root] = node.arguments[0]
				}
			case ast.ASTArrayOpDefault:
				base := node.arguments[0].TheoryVar(t.FamilyID())
				if base < 0 {
					return ast.ErrASTDomain
				}
				root := d.find(int(base))
				if d.defaults[root] == nil {
					d.defaults[root] = node
				}
			}
		}
	}
	for variable, node := range t.variables {
		representative, err := t.modelRepresentative(node)
		if err != nil {
			return err
		}
		if variable != representative || !generator.context.isRelevant(node.root.owner) {
			continue
		}
		root := node.root
		for _, parent := range root.parents {
			op, array, err := t.arrayOperation(parent.owner)
			if err != nil {
				return err
			}
			if !array || op != ast.ASTArrayOpSelect || !parent.IsCongruenceRoot() || !generator.context.isRelevant(parent.owner) || parent.arguments[0].root != root {
				continue
			}
			if _, err = d.addSelect(root, parent); err != nil {
				return err
			}
		}
	}
	// Source first traverses the original select domains, then processes
	// newly reached store-parent domains through the explicit worklist.
	var todo []smtArrayEnodePair
	domains := append([]*SMTEnode(nil), d.selectDomains...)
	for _, root := range domains {
		for _, read := range append([]*SMTEnode(nil), d.selects[root]...) {
			if err := t.propagateModelSelect(root, read, generator, &todo); err != nil {
				return err
			}
		}
	}
	for head := 0; head < len(todo); head++ {
		pair := todo[head]
		if err := t.propagateModelSelect(pair.first, pair.second, generator, &todo); err != nil {
			return err
		}
	}
	return nil
}

func (d *smtArrayModelData) addSelect(root, read *SMTEnode) (bool, error) {
	entries, exists := d.selects[root]
	if !exists {
		d.selectDomains = append(d.selectDomains, root)
	}
	for _, previous := range entries {
		if sameArrayModelIndices(previous, read) {
			if previous.root != read.root {
				return false, fmt.Errorf("array model repeated index has distinct value roots %v and %v: %w", previous.owner, read.owner, ast.ErrASTDomain)
			}
			return false, nil
		}
	}
	d.selects[root] = append(entries, read)
	return true, nil
}

func sameArrayModelIndices(first, second *SMTEnode) bool {
	if len(first.arguments) != len(second.arguments) {
		return false
	}
	for i := 1; i < len(first.arguments); i++ {
		if first.arguments[i].root != second.arguments[i].root {
			return false
		}
	}
	return true
}

func (t *SMTArrayTheory) propagateModelSelect(root, read *SMTEnode, generator *SMTModelGenerator, todo *[]smtArrayEnodePair) error {
	if !generator.context.isRelevant(root.owner) {
		return nil
	}
	for _, parent := range root.parents {
		op, array, err := t.arrayOperation(parent.owner)
		if err != nil {
			return err
		}
		if !array || op != ast.ASTArrayOpStore || !generator.context.isRelevant(parent.owner) || parent.arguments[0].root != root {
			continue
		}
		if len(parent.arguments) != len(read.arguments)+1 {
			return ast.ErrASTDomain
		}
		entries, exists := t.modelData.selects[parent.root]
		if !exists {
			t.modelData.selects[parent.root] = nil
			t.modelData.selectDomains = append(t.modelData.selectDomains, parent.root)
		}
		present := false
		for _, previous := range entries {
			if sameArrayModelIndices(previous, read) {
				present = true
				break
			}
		}
		// Native upward propagation keeps an existing store-side entry. It
		// does not replace it with the base-side read for the same indices.
		if present {
			continue
		}
		overwritten := true
		for i := 1; i < len(read.arguments); i++ {
			if read.arguments[i].root != parent.arguments[i].root {
				overwritten = false
				break
			}
		}
		if overwritten {
			continue
		}
		added, err := t.modelData.addSelect(parent.root, read)
		if err != nil {
			return err
		}
		if added {
			*todo = append(*todo, smtArrayEnodePair{parent.root, read})
		}
	}
	return nil
}

func (t *SMTArrayTheory) FinalizeModel(*SMTModelGenerator) error {
	t.modelData = nil
	return nil
}
