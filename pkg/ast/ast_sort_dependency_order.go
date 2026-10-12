// Source: src/util/top_sort.h:31-99; concrete top_sort<sort> use in
// src/ast/decl_collector.cpp:118-124.
package ast

// sortDependencyOrder records only sorts collected by decl_collector. A
// dependency outside this set is ignored by the source traversal.
type sortDependencyOrder struct {
	roots []ASTHandle
	deps  map[ASTHandle][]ASTHandle
}

// insertASTSort supplies the parameter-sort edges collected by
// decl_collector::collect_deps before top_sort<sort> traverses them. Datatype
// constructor edges belong to the separate datatype collector path.
func (o *sortDependencyOrder) insertASTSort(m *ASTManager, sort ASTHandle) error {
	if o == nil || m == nil {
		return ErrASTHandle
	}
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	r, e := m.requireKind(sort, ASTSort)
	if e != nil {
		return e
	}
	var deps []ASTHandle
	info := m.sortAt(r.slot).info
	if info != (ASTSortInfo{}) {
		v, e := m.sortInfo(info)
		if e != nil {
			return e
		}
		seen := make(map[ASTHandle]struct{})
		for i := v.count; i > 0; i-- {
			p, e := m.infoParameter(v, i-1)
			if e != nil {
				return e
			}
			pv, e := m.parameter(p)
			if e != nil {
				return e
			}
			if pv.Kind != ASTParameterAST || pv.ast == (astNodeRef{}) || m.nodeAt(pv.ast.slot).kind != ASTSort {
				continue
			}
			child := m.external(pv.ast)
			if child == sort {
				continue
			}
			if _, found := seen[child]; !found {
				seen[child] = struct{}{}
				deps = append(deps, child)
			}
		}
	}
	o.insert(sort, deps)
	return nil
}

func (o *sortDependencyOrder) insert(sort ASTHandle, deps []ASTHandle) {
	if o.deps == nil {
		o.deps = make(map[ASTHandle][]ASTHandle)
	}
	if _, present := o.deps[sort]; !present {
		o.roots = append(o.roots, sort)
	}
	o.deps[sort] = append([]ASTHandle(nil), deps...)
}

// order follows the source preorder and completion stacks. The source
// obj_map iteration order is not a semantic contract, so roots and each
// dependency set are traversed in the order supplied by the AST caller.
func (o *sortDependencyOrder) order() []ASTHandle {
	var result []ASTHandle
	if o == nil {
		return result
	}
	dfsNum := make(map[ASTHandle]uint32, len(o.deps))
	partitionID := make(map[ASTHandle]uint32, len(o.deps))
	var stackS, stackP []ASTHandle
	var nextPreorder uint32
	var traverse func(ASTHandle)
	traverse = func(f ASTHandle) {
		if pID, visited := dfsNum[f]; visited {
			if _, done := partitionID[f]; !done {
				for len(stackP) != 0 {
					back := stackP[len(stackP)-1]
					backID, completed := partitionID[back]
					if !completed || backID <= pID {
						break
					}
					stackP = stackP[:len(stackP)-1]
				}
			}
			return
		}
		children, present := o.deps[f]
		if !present {
			return
		}
		dfsNum[f] = nextPreorder
		nextPreorder++
		stackS = append(stackS, f)
		stackP = append(stackP, f)
		for _, child := range children {
			traverse(child)
		}
		if len(stackP) != 0 && f == stackP[len(stackP)-1] {
			pID := uint32(len(result))
			for {
				back := stackS[len(stackS)-1]
				stackS = stackS[:len(stackS)-1]
				result = append(result, back)
				partitionID[back] = pID
				if back == f {
					break
				}
			}
			stackP = stackP[:len(stackP)-1]
		}
	}
	for _, root := range o.roots {
		traverse(root)
	}
	return result
}
