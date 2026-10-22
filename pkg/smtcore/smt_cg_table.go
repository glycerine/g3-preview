// Source: active implementation in src/smt/smt_cg_table.h and
// src/smt/smt_cg_table.cpp.
package smtcore

import "github.com/glycerine/g3/pkg/ast"

type SMTCongruenceInsertResult struct {
	Node     *SMTEnode
	Commuted bool
}

type SMTCongruenceTable struct {
	self         *SMTCongruenceTable
	manager      *ast.ASTManager
	tables       map[ast.ASTHandle][]*SMTEnode
	declarations []ast.ASTHandle
	resetIndex   int
	closed       bool
}

func NewSMTCongruenceTable(manager *ast.ASTManager) (*SMTCongruenceTable, error) {
	if manager == nil {
		return nil, ast.ErrASTDomain
	}
	table := &SMTCongruenceTable{manager: manager, tables: make(map[ast.ASTHandle][]*SMTEnode)}
	table.self = table
	return table, nil
}

func (t *SMTCongruenceTable) open() error {
	if t == nil || t.self != t || t.manager == nil || t.closed {
		return ast.ErrASTHandle
	}
	return nil
}

func (t *SMTCongruenceTable) validateNode(node *SMTEnode) error {
	if err := t.open(); err != nil {
		return err
	}
	if node == nil || node.manager != t.manager || len(node.arguments) == 0 {
		return ast.ErrASTDomain
	}
	return node.valid()
}

func (t *SMTCongruenceTable) ensureTable(node *SMTEnode) error {
	if _, found := t.tables[node.declaration]; found {
		return nil
	}
	if err := t.manager.IncRef(node.declaration); err != nil {
		return err
	}
	node.functionDeclID = uint32(len(t.declarations))
	t.declarations = append(t.declarations, node.declaration)
	t.tables[node.declaration] = nil
	return nil
}

func (t *SMTCongruenceTable) Insert(node *SMTEnode) (SMTCongruenceInsertResult, error) {
	if err := t.validateNode(node); err != nil {
		return SMTCongruenceInsertResult{}, err
	}
	if err := t.ensureTable(node); err != nil {
		return SMTCongruenceInsertResult{}, err
	}
	for _, candidate := range t.tables[node.declaration] {
		congruent, commuted, err := SMTEnodesCongruent(candidate, node)
		if err != nil {
			return SMTCongruenceInsertResult{}, err
		}
		if congruent {
			return SMTCongruenceInsertResult{Node: candidate, Commuted: commuted}, nil
		}
	}
	t.tables[node.declaration] = append(t.tables[node.declaration], node)
	return SMTCongruenceInsertResult{Node: node}, nil
}

func (t *SMTCongruenceTable) Find(node *SMTEnode) (*SMTEnode, bool, error) {
	if err := t.validateNode(node); err != nil {
		return nil, false, err
	}
	for _, candidate := range t.tables[node.declaration] {
		congruent, commuted, err := SMTEnodesCongruent(candidate, node)
		if err != nil {
			return nil, false, err
		}
		if congruent {
			return candidate, commuted, nil
		}
	}
	return nil, false, nil
}

func (t *SMTCongruenceTable) Contains(node *SMTEnode) (bool, error) {
	found, _, err := t.Find(node)
	return found != nil, err
}

func (t *SMTCongruenceTable) ContainsPointer(node *SMTEnode) (bool, error) {
	found, _, err := t.Find(node)
	return found == node, err
}

func (t *SMTCongruenceTable) Erase(node *SMTEnode) (bool, error) {
	if err := t.validateNode(node); err != nil {
		return false, err
	}
	bucket := t.tables[node.declaration]
	for i, candidate := range bucket {
		congruent, _, err := SMTEnodesCongruent(candidate, node)
		if err != nil {
			return false, err
		}
		if !congruent {
			continue
		}
		copy(bucket[i:], bucket[i+1:])
		bucket[len(bucket)-1] = nil
		t.tables[node.declaration] = bucket[:len(bucket)-1]
		return true, nil
	}
	return false, nil
}

func (t *SMTCongruenceTable) Reset() error {
	if err := t.open(); err != nil {
		return err
	}
	if t.resetIndex == 0 {
		t.resetIndex = len(t.declarations)
	}
	for t.resetIndex > 0 {
		i := t.resetIndex - 1
		handle := t.declarations[i]
		consumed, err := t.manager.DecRefConsumed(handle)
		if consumed {
			t.declarations[i] = ast.ASTHandle{}
			t.resetIndex--
		}
		if err != nil {
			return err
		}
	}
	t.declarations = nil
	t.tables = make(map[ast.ASTHandle][]*SMTEnode)
	return nil
}

func (t *SMTCongruenceTable) CheckInvariant() error {
	if err := t.open(); err != nil {
		return err
	}
	for _, bucket := range t.tables {
		for _, node := range bucket {
			contained, err := t.ContainsPointer(node)
			if err != nil {
				return err
			}
			if !contained {
				return ast.ErrASTDomain
			}
		}
	}
	return nil
}

// Display and DisplayCompact are intentionally empty in the active source
// implementation.
func (t *SMTCongruenceTable) Display() error        { return t.open() }
func (t *SMTCongruenceTable) DisplayCompact() error { return t.open() }

func (t *SMTCongruenceTable) Close() error {
	if t == nil || t.self != t {
		return ast.ErrASTHandle
	}
	if t.closed {
		return nil
	}
	if err := t.Reset(); err != nil {
		return err
	}
	t.manager = nil
	t.closed = true
	return nil
}
