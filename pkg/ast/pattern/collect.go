// Source: src/ast/pattern/pattern_inference.cpp:121-267,485-499.
package pattern

import "github.com/glycerine/g3/pkg/ast"

type collectKey struct {
	node  ast.ASTHandle
	delta uint32
}

type collectedInfo struct {
	node ast.ASTHandle
	vars map[uint32]struct{}
	size uint32
}

type collectFrame struct {
	key  collectKey
	post bool
}

func (c *patternCandidates) releaseCollectRefs() error {
	for len(c.collectRefs) != 0 {
		h := c.collectRefs[0]
		consumed, err := c.manager.DecRefConsumed(h)
		if consumed {
			c.collectRefs[0] = ast.ASTHandle{}
			c.collectRefs = c.collectRefs[1:]
		}
		if err != nil {
			return err
		}
		if !consumed {
			return ast.ErrASTPending
		}
	}
	return nil
}

func (c *patternCandidates) saveCollectInfo(cache map[collectKey]*collectedInfo, key collectKey, info *collectedInfo) error {
	if info != nil {
		if err := c.manager.IncRef(info.node); err != nil {
			return err
		}
		c.collectRefs = append(c.collectRefs, info.node)
	}
	cache[key] = info
	return nil
}

func (c *patternCandidates) forbiddenApp(n ast.ASTHandle) (bool, error) {
	ground, err := c.manager.IsGround(n)
	if err != nil || ground {
		return ground, err
	}
	decl, err := c.manager.GetDecl(n)
	if err != nil {
		return false, err
	}
	if c.avoidSkolems {
		skolem, e := c.manager.IsSkolemDecl(decl)
		if e != nil || skolem {
			return skolem, e
		}
	}
	family, err := c.manager.GetFamilyID(decl)
	if err != nil {
		return false, err
	}
	if family == c.manager.BasicFamilyID() {
		kind, e := c.manager.GetDeclKind(decl)
		if e != nil {
			return false, e
		}
		if kind != ast.ASTOpTrue && kind != ast.ASTOpFalse {
			return true, nil
		}
	}
	_, forbidden := c.forbiddenFamilies[family]
	return forbidden, nil
}

func (c *patternCandidates) collectNode(cache map[collectKey]*collectedInfo, key collectKey, arith ast.FamilyID) error {
	m := c.manager
	n := key.node
	kind, err := m.GetKind(n)
	if err != nil {
		return err
	}
	switch kind {
	case ast.ASTVar:
		index, e := m.GetVarIndex(n)
		if e != nil {
			return e
		}
		if index < key.delta {
			return c.saveCollectInfo(cache, key, nil)
		}
		index -= key.delta
		vars := make(map[uint32]struct{})
		if index < c.numBindings {
			vars[index] = struct{}{}
		}
		mapped := n
		if key.delta != 0 {
			sort, e := m.GetSort(n)
			if e != nil {
				return e
			}
			mapped, e = m.MkVar(index, sort)
			if e != nil {
				return e
			}
		}
		return c.saveCollectInfo(cache, key, &collectedInfo{node: mapped, vars: vars, size: 1})
	case ast.ASTApp:
		forbidden, e := c.forbiddenApp(n)
		if e != nil {
			return e
		}
		if forbidden {
			return c.saveCollectInfo(cache, key, nil)
		}
		count, e := m.GetNumArgs(n)
		if e != nil {
			return e
		}
		if count == 0 {
			return c.saveCollectInfo(cache, key, &collectedInfo{node: n, vars: make(map[uint32]struct{}), size: 1})
		}
		args := make([]ast.ASTHandle, count)
		vars := make(map[uint32]struct{})
		size := uint32(1)
		changed := false
		for i := uint32(0); i < count; i++ {
			child, e := m.GetArg(n, i)
			if e != nil {
				return e
			}
			info, found := cache[collectKey{node: child, delta: key.delta}]
			if !found {
				return ast.ErrASTDomain
			}
			if info == nil {
				return c.saveCollectInfo(cache, key, nil)
			}
			args[i] = info.node
			changed = changed || child != info.node
			for v := range info.vars {
				vars[v] = struct{}{}
			}
			size += info.size
		}
		mapped := n
		if changed {
			decl, e := m.GetDecl(n)
			if e != nil {
				return e
			}
			mapped, e = m.MkApp(decl, args)
			if e != nil {
				return e
			}
		}
		info := &collectedInfo{node: mapped, vars: vars, size: size}
		if err := c.saveCollectInfo(cache, key, info); err != nil {
			return err
		}
		family, e := m.GetFamilyID(n)
		if e != nil {
			return e
		}
		if len(vars) == 0 {
			return nil
		}
		if family == arith {
			if c.nestedArithOnly {
				return nil
			}
			op, e := m.GetDeclKind(n)
			if e != nil {
				return e
			}
			if op != 9 && op != 10 && op != 11 && op != 12 && op != 13 {
				return nil
			}
		}
		free := make([]uint32, 0, len(vars))
		for v := range vars {
			free = append(free, v)
		}
		return c.addCandidate(mapped, free, size)
	default:
		return c.saveCollectInfo(cache, key, nil)
	}
}

// collect follows the source postorder traversal. The cache keys include the
// current nested binder depth; every non-null cached node gets an independent
// temporary reference that is released in discovery order after collection.
func (c *patternCandidates) collect(body ast.ASTHandle) (err error) {
	if c == nil || c.closed {
		return ast.ErrASTHandle
	}
	if len(c.candidates) != 0 || len(c.collectRefs) != 0 {
		return ast.ErrASTDomain
	}
	arith, err := c.manager.ArithmeticFamilyID()
	if err != nil {
		return err
	}
	defer func() {
		if e := c.releaseCollectRefs(); err == nil {
			err = e
		}
	}()
	cache := make(map[collectKey]*collectedInfo)
	pending := make(map[collectKey]struct{})
	stack := []collectFrame{{key: collectKey{node: body}}}
	for len(stack) != 0 {
		frame := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if frame.post {
			if err = c.collectNode(cache, frame.key, arith); err != nil {
				return err
			}
			delete(pending, frame.key)
			continue
		}
		if _, done := cache[frame.key]; done {
			continue
		}
		if _, queued := pending[frame.key]; queued {
			continue
		}
		pending[frame.key] = struct{}{}
		stack = append(stack, collectFrame{key: frame.key, post: true})
		kind, e := c.manager.GetKind(frame.key.node)
		if e != nil {
			return e
		}
		switch kind {
		case ast.ASTApp:
			count, e := c.manager.GetNumArgs(frame.key.node)
			if e != nil {
				return e
			}
			for i := count; i != 0; i-- {
				child, e := c.manager.GetArg(frame.key.node, i-1)
				if e != nil {
					return e
				}
				stack = append(stack, collectFrame{key: collectKey{node: child, delta: frame.key.delta}})
			}
		case ast.ASTQuantifier:
			count, e := c.manager.GetNumDeclarations(frame.key.node)
			if e != nil {
				return e
			}
			child, e := c.manager.GetQuantifierBody(frame.key.node)
			if e != nil {
				return e
			}
			stack = append(stack, collectFrame{key: collectKey{node: child, delta: frame.key.delta + count}})
		}
	}
	return nil
}
