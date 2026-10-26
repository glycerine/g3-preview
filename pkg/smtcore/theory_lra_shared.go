// Source: src/smt/theory_lra.cpp:1012-1145,1200-1243.
package smtcore

import "github.com/glycerine/g3/pkg/ast"

type smtLRASharedValueKey struct {
	main, epsilon string
	integer       bool
}

func (t *SMTLRATheory) relevantShared(node *SMTEnode) (bool, error) {
	if t.state.relevancy != nil && !t.state.relevancy.IsRelevant(node.owner) {
		return false, nil
	}
	shared, consult, err := t.state.sharedTheoryNode(node)
	if err != nil || !consult || shared || len(t.underspecified) == 0 {
		return shared, err
	}
	root := node.Root()
	if len(root.parents) > 2*len(t.underspecified) {
		for _, term := range t.underspecified {
			n, err := t.state.manager.GetNumArgs(term)
			if err != nil {
				return false, err
			}
			for i := uint32(0); i < n; i++ {
				arg, err := t.state.manager.GetArg(term, i)
				if err != nil {
					return false, err
				}
				enode := t.state.context.enodes[arg]
				if enode == nil {
					return false, ast.ErrASTDomain
				}
				if enode.Root() == root {
					return true, nil
				}
			}
		}
	} else {
		for _, parent := range root.parents {
			underspecified, err := t.underspecifiedTerm(parent.owner)
			if err != nil || underspecified {
				return underspecified, err
			}
		}
	}
	return false, nil
}

func (t *SMTLRATheory) delayedAssumeEqualities() (bool, error) {
	for t.assumeHead < len(t.assumeCandidates) {
		pair := t.assumeCandidates[t.assumeHead]
		left, err := t.Enode(pair.left)
		if err != nil {
			return false, err
		}
		right, err := t.Enode(pair.right)
		if err != nil {
			return false, err
		}
		t.assumeHead++ // Source consumes a candidate before checking current values.
		for _, v := range []SMTTheoryVar{pair.left, pair.right} {
			index, ok := t.numeric.index(v)
			if !ok || !t.numeric.solver.VarIsRegistered(index) {
				return false, ast.ErrASTDomain
			}
		}
		a, err := t.numeric.value(pair.left)
		if err != nil {
			return false, err
		}
		b, err := t.numeric.value(pair.right)
		if err != nil {
			return false, err
		}
		if a.Cmp(&b) == 0 && left.Root() != right.Root() {
			if progress, err := t.state.assumeEquality(left, right); err != nil || progress {
				return progress, err
			}
		}
	}
	return false, nil
}

func (t *SMTLRATheory) assumeSharedEqualities() (progress bool, err error) {
	if _, err = t.State(); err != nil {
		return false, err
	}
	if err = t.numeric.ready(); err != nil {
		return false, err
	}
	var indices []uint32
	for v, node := range t.variables {
		shared, err := t.relevantShared(node)
		if err != nil {
			return false, err
		}
		if !shared {
			continue
		}
		index, ok := t.numeric.index(SMTTheoryVar(v))
		// The native direct directory access requires a mapped variable here.
		// Do not invent a new column or silently substitute zero.
		if !ok {
			return false, ast.ErrASTDomain
		}
		indices = append(indices, index)
	}
	if len(indices) == 0 {
		return false, nil
	}
	if err = t.numeric.solver.RandomUpdate(indices); err != nil {
		t.numeric.stopped = true
		return false, err
	}
	values := make(map[smtLRASharedValueKey]SMTTheoryVar)
	size := len(t.variables)
	start := int(t.state.random.Next())
	for i := 0; i < size; i++ {
		v := SMTTheoryVar((i + start) % size)
		node := t.variables[v]
		shared, err := t.relevantShared(node)
		if err != nil {
			return false, err
		}
		if !shared {
			continue
		}
		index, ok := t.numeric.index(v)
		if !ok || !t.numeric.solver.VarIsRegistered(index) {
			continue
		}
		value, err := t.numeric.value(v)
		if err != nil {
			return false, err
		}
		sort, err := t.fixedKey(v, &value.Main)
		if err != nil {
			return false, err
		}
		key := smtLRASharedValueKey{value.Main.RatString(), value.Epsilon.RatString(), sort.integer}
		other, found := values[key]
		if !found {
			values[key] = v
			continue
		}
		if node.Root() != t.variables[other].Root() {
			t.assumeCandidates = append(t.assumeCandidates, smtArithmeticEqualityCandidate{v, other})
		}
	}
	return t.delayedAssumeEqualities()
}
