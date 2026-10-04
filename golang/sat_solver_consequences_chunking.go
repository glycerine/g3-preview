// Source: src/sat/sat_solver.cpp:3157-3249.
// Licensed under the MIT license. Source author: Nikolaj Bjorner.
package g3

import "github.com/glycerine/g3/golang/arena"

func satConsequencePruneUnfixed(lambda, model *satSolverRuntimeUintColumn) error {
	for i := uint32(0); ; i++ {
		n, e := lambda.size()
		if e != nil {
			return e
		}
		if i >= n {
			return nil
		}
		l, e := lambda.read(i)
		if e != nil {
			return e
		}
		v, e := model.read(uint32(satLiteral(l).variable()))
		if e != nil {
			return e
		}
		if (lbool(int32(v)) == lFalse) != satLiteral(l).sign() {
			n, e = lambda.size()
			if e != nil {
				return e
			}
			if n == 0 {
				return arena.ErrBounds
			}
			back, e := lambda.read(n - 1)
			if e != nil {
				return e
			}
			if e = lambda.write(i, back); e != nil {
				return e
			}
			n, e = lambda.size()
			if e != nil {
				return e
			}
			if e = lambda.shrink(n - 1); e != nil {
				return e
			}
			i--
		}
	}
}
func satConsequenceBackRemove(lits *satSolverRuntimeUintColumn, l uint32) error {
	n, e := lits.size()
	if e != nil {
		return e
	}
	for i := n; i > 0; {
		i--
		v, e := lits.read(i)
		if e != nil {
			return e
		}
		if v == l {
			n, e = lits.size()
			if e != nil {
				return e
			}
			if n == 0 {
				return arena.ErrBounds
			}
			back, e := lits.read(n - 1)
			if e != nil {
				return e
			}
			if e = lits.write(i, back); e != nil {
				return e
			}
			n, e = lits.size()
			if e != nil {
				return e
			}
			return lits.shrink(n - 1)
		}
	}
	// Source UNREACHABLE is outside the represented defined domain.
	return ErrSATAssertion
}

// Caller arrays are borrowed adapters; real columns execute source copy_core.
func (f *satConsequenceFrame) chunkCopy(s *satSolver, i uint8, input satConsequenceInput) error {
	if input.column != nil {
		return f.vector(s, i, input.column)
	}
	if e := f.vector(s, i, nil); e != nil {
		return e
	}
	for j := uint32(0); ; j++ {
		n, e := input.size()
		if e != nil {
			return e
		}
		if j >= n {
			return nil
		}
		v, e := input.read(j)
		if e != nil {
			return e
		}
		if e = f.vectors[i].push(v); e != nil {
			return e
		}
	}
}
func (s *satSolver) bruteForceConsequences(asms, gamma satConsequenceInput, out *satSolverLiteralRows) error {
	if e := s.consequenceEntry(out); e != nil {
		return e
	}
	previous := s.consequenceRunning
	s.consequenceRunning = true
	defer func() { s.consequenceRunning = previous }()
	for i := uint32(0); ; i++ {
		n, e := gamma.size()
		if e != nil {
			return e
		}
		if i >= n {
			return nil
		}
		l, e := gamma.read(i)
		if e != nil {
			return e
		}
		if e = s.bruteForceConsequenceOne(asms, l^1, out); e != nil {
			return e
		}
	}
}
func (s *satSolver) bruteForceConsequenceOne(asms satConsequenceInput, nlit uint32, out *satSolverLiteralRows) (err error) {
	f, e := s.consequenceFrame()
	if e != nil {
		return e
	}
	defer func() { err = s.consequenceDrop(f, err) }()
	if err = f.chunkCopy(s, 0, asms); err != nil {
		return err
	}
	if err = f.vectors[0].push(nlit); err != nil {
		return err
	}
	n, e := f.vectors[0].size()
	if e != nil {
		return e
	}
	r, e := s.checkSpan(n, satConflictRuntimeSpan(&f.vectors[0]))
	if e != nil {
		return e
	}
	if r == lFalse {
		return out.appendColumn(&s.conflict.core)
	}
	return nil
}
func (s *satSolver) coreChunking(model *satSolverRuntimeUintColumn, vars, asms satConsequenceInput, out *satSolverLiteralRows, K uint32) (result lbool, err error) {
	if err = s.consequenceEntry(out); err != nil {
		return lUndef, err
	}
	s.consequenceRunning = true
	defer func() { s.consequenceRunning = false }()
	f, e := s.consequenceFrame()
	if e != nil {
		return lUndef, e
	}
	defer func() { err = s.consequenceDrop(f, err) }()
	if err = f.vector(s, 0, nil); err != nil {
		return lUndef, err
	}
	lambda := &f.vectors[0]
	for i := uint32(0); ; i++ {
		n, e := vars.size()
		if e != nil {
			return lUndef, e
		}
		if i >= n {
			break
		}
		v, e := vars.read(i)
		if e != nil {
			return lUndef, e
		}
		m, e := model.read(v)
		if e != nil {
			return lUndef, e
		}
		if err = lambda.push(satMakeLiteral(satBoolVar(v), lbool(int32(m)) == lFalse).index()); err != nil {
			return lUndef, err
		}
	}
	for {
		n, e := lambda.size()
		if e != nil {
			return lUndef, e
		}
		if n == 0 {
			return lTrue, nil
		}
		if err = s.coreChunk(lambda, asms, out, K); err != nil {
			return lUndef, err
		}
	}
}
func (s *satSolver) coreChunk(lambda *satSolverRuntimeUintColumn, asms satConsequenceInput, out *satSolverLiteralRows, K uint32) (err error) {
	n, e := lambda.size()
	if e != nil {
		return e
	}
	k := K
	if n < k {
		k = n
	}
	f, e := s.consequenceFrame()
	if e != nil {
		return e
	}
	defer func() { err = s.consequenceDrop(f, err) }()
	if err = f.vector(s, 0, nil); err != nil {
		return err
	}
	if err = f.vector(s, 1, nil); err != nil {
		return err
	}
	gamma, omega := &f.vectors[0], &f.vectors[1]
	for i := uint32(0); i < k; i++ {
		n, e = lambda.size()
		if e != nil {
			return e
		}
		l, e := lambda.read(n - i - 1)
		if e != nil {
			return e
		}
		if err = gamma.push(l); err != nil {
			return err
		}
		if err = omega.push(l ^ 1); err != nil {
			return err
		}
	}
	for {
		done, e := s.coreChunkCheck(lambda, gamma, omega, asms, out)
		if e != nil {
			return e
		}
		if done {
			return nil
		}
	}
}
func (s *satSolver) coreChunkCheck(lambda, gamma, omega *satSolverRuntimeUintColumn, asms satConsequenceInput, out *satSolverLiteralRows) (done bool, err error) {
	f, e := s.consequenceFrame()
	if e != nil {
		return false, e
	}
	defer func() { err = s.consequenceDrop(f, err) }()
	if err = f.chunkCopy(s, 0, asms); err != nil {
		return false, err
	}
	if err = satMUSAppend(&f.vectors[0], omega); err != nil {
		return false, err
	}
	n, e := f.vectors[0].size()
	if e != nil {
		return false, e
	}
	r, e := s.checkSpan(n, satConflictRuntimeSpan(&f.vectors[0]))
	if e != nil {
		return false, e
	}
	if r == lTrue {
		return true, satConsequencePruneUnfixed(lambda, &s.model)
	}
	// Source reads the existing core after l_undef too; no status repair.
	if err = f.vector(s, 1, nil); err != nil {
		return false, err
	}
	occurs := &f.vectors[1]
	for i := uint32(0); ; i++ {
		n, e = omega.size()
		if e != nil {
			return false, e
		}
		if i >= n {
			break
		}
		l, e := omega.read(i)
		if e != nil {
			return false, e
		}
		yes, e := satMUSContains(&s.conflict.core, l)
		if e != nil {
			return false, e
		}
		if yes {
			if err = occurs.push(l); err != nil {
				return false, err
			}
		}
	}
	n, e = occurs.size()
	if e != nil {
		return false, e
	}
	if n == 1 {
		l, e := occurs.read(n - 1)
		if e != nil {
			return false, e
		}
		nlit := l ^ 1
		if err = out.appendColumn(&s.conflict.core); err != nil {
			return false, err
		}
		if err = satConsequenceBackRemove(lambda, nlit); err != nil {
			return false, err
		}
		if err = satConsequenceBackRemove(gamma, nlit); err != nil {
			return false, err
		}
		one := [1]satLiteral{satLiteral(nlit)}
		if err = s.MkClause(one[:]); err != nil {
			return false, err
		}
	}
	for i := uint32(0); ; i++ {
		n, e = omega.size()
		if e != nil {
			return false, e
		}
		if i >= n {
			break
		}
		l, e := omega.read(i)
		if e != nil {
			return false, e
		}
		yes, e := satMUSContains(occurs, l)
		if e != nil {
			return false, e
		}
		if yes {
			n, e = omega.size()
			if e != nil {
				return false, e
			}
			back, e := omega.read(n - 1)
			if e != nil {
				return false, e
			}
			if err = omega.write(i, back); err != nil {
				return false, err
			}
			n, e = omega.size()
			if e != nil {
				return false, e
			}
			if err = omega.shrink(n - 1); err != nil {
				return false, err
			}
			i--
		}
	}
	n, e = omega.size()
	if e != nil {
		return false, e
	}
	if n == 0 {
		count, e := occurs.size()
		if e != nil {
			return false, e
		}
		if count > 1 {
			if err = s.bruteForceConsequences(asms, satConsequenceColumn(gamma), out); err != nil {
				return false, err
			}
			for i := uint32(0); ; i++ {
				n, e = gamma.size()
				if e != nil {
					return false, e
				}
				if i >= n {
					break
				}
				l, e := gamma.read(i)
				if e != nil {
					return false, e
				}
				if err = satConsequenceBackRemove(lambda, l); err != nil {
					return false, err
				}
			}
			return true, nil
		}
	}
	return false, nil
}
