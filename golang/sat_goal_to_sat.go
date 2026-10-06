// Source: src/sat/tactic/goal2sat.cpp:41-423,488-500;
// src/ast/expr2var.cpp:25-54.
// Licensed under the MIT license. Source author: Leonardo de Moura (2011).
package g3

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

var ErrSATGoalUnsupported = errors.New("operator not supported, apply simplifier before invoking translator")
var ErrSATGoalFailed = errors.New("sat: translator failed; close before reuse")

type satGoalFrame struct {
	term       ASTHandle
	index      uint32
	root, sign bool
}

func satGoalFrameBytes() uint64 { return uint64(reflect.TypeOf(satGoalFrame{}).Size()) }

// This implements the source array-of-formulas overload, using the actual CDCL
// solver. Goal dependencies, tactic/model converters and public API factories
// remain separate callers. Caller assertions must stay alive through Compile.
type satGoalTranslator struct {
	self    *satGoalTranslator
	manager *ASTManager
	solver  *satSolver
	// Until the context owns an AST-manager limit, the caller explicitly shares
	// its SAT limit with this translation; no SAT propagation checkpoint is used.
	limit                                     *utilReslimit
	cooperation                               *utilCooperation
	cache, atoms, interfaces                  satGoalASTMap
	frames                                    []satGoalFrame
	results                                   []satLiteral
	interpreted                               []ASTHandle
	trueVar                                   satBoolVar
	boolSort                                  ASTHandle
	maxMemory                                 uint64
	ownerBytes                                uint64
	iteExtra, defaultExternal, failed, closed bool
}

func newSATGoalTranslator(m *ASTManager, s *satSolver) (tr *satGoalTranslator, err error) {
	if m == nil {
		return nil, ErrASTHandle
	}
	if err = s.open(); err != nil {
		return nil, err
	}
	boolSort, err := m.MkBoolSort()
	if err != nil {
		return nil, err
	}
	amount := uint64(reflect.TypeOf(satGoalTranslator{}).Size())
	if err = s.budget.Reserve(amount); err != nil {
		return nil, err
	}
	tr = &satGoalTranslator{manager: m, solver: s, limit: s.limit, trueVar: satNullBoolVar, boolSort: boolSort, ownerBytes: amount}
	tr.self = tr
	defer func() {
		if err != nil {
			err = errors.Join(err, tr.Close())
			tr = nil
		}
	}()
	// MkBoolSort is owned by the basic plugin and remains alive with the manager.
	for _, table := range []*satGoalASTMap{&tr.cache, &tr.atoms, &tr.interfaces} {
		if err = table.prepare(s.budget); err != nil {
			return tr, err
		}
	}
	if tr.cooperation, err = newUtilCooperation(s.budget); err != nil {
		return tr, err
	}
	name := "ite_extra"
	if tr.iteExtra, err = s.context.GetBoolName(&s.params, &name, true); err != nil {
		return tr, err
	}
	name = "max_memory"
	var mb uint32
	if mb, err = s.context.GetUIntName(&s.params, &name, ^uint32(0)); err != nil {
		return tr, err
	}
	tr.maxMemory = uint64(mb) * 1024 * 1024
	return tr, nil
}
func (tr *satGoalTranslator) open() error {
	if tr == nil || tr.self != tr {
		return ErrSATOwner
	}
	if tr.closed {
		return arena.ErrClosed
	}
	if tr.failed {
		return ErrSATGoalFailed
	}
	return tr.solver.open()
}
func (tr *satGoalTranslator) Close() error {
	if tr == nil || tr.self != tr {
		return ErrSATOwner
	}
	if tr.closed {
		return nil
	}
	for len(tr.interpreted) > 0 {
		i := len(tr.interpreted) - 1
		if e := tr.manager.DecRef(tr.interpreted[i]); e != nil {
			return e
		}
		tr.interpreted = tr.interpreted[:i]
	}
	for i := range tr.atoms.cells {
		entry := &tr.atoms.cells[i]
		if entry.used {
			if e := tr.manager.DecRef(entry.key); e != nil {
				return e
			}
			entry.used = false
		}
	}
	if tr.cooperation != nil {
		if e := tr.cooperation.Close(); e != nil {
			return e
		}
		tr.cooperation = nil
	}
	b := tr.solver.budget
	tr.cache.close(b)
	tr.atoms.close(b)
	tr.interfaces.close(b)
	b.Release(uint64(cap(tr.frames)) * uint64(reflect.TypeOf(satGoalFrame{}).Size()))
	b.Release(uint64(cap(tr.results)) * 4)
	b.Release(uint64(cap(tr.interpreted)) * uint64(reflect.TypeOf(ASTHandle{}).Size()))
	tr.interpreted = nil
	tr.frames = nil
	tr.results = nil
	b.Release(tr.ownerBytes)
	tr.ownerBytes = 0
	tr.closed = true
	return nil
}
func (tr *satGoalTranslator) HasInterpretedAtoms() (bool, error) {
	if e := tr.open(); e != nil {
		return false, e
	}
	return len(tr.interpreted) != 0, nil
}
func (tr *satGoalTranslator) retainInterpreted(h ASTHandle) error {
	stride := uint64(reflect.TypeOf(h).Size())
	if len(tr.interpreted) == cap(tr.interpreted) {
		n, e := satGoalCapacity(cap(tr.interpreted), stride)
		if e != nil {
			return e
		}
		if e = tr.solver.budget.Reserve(uint64(n) * stride); e != nil {
			return e
		}
		next := make([]ASTHandle, len(tr.interpreted), n)
		copy(next, tr.interpreted)
		tr.solver.budget.Release(uint64(cap(tr.interpreted)) * stride)
		tr.interpreted = next
	}
	if e := tr.manager.IncRef(h); e != nil {
		return e
	}
	tr.interpreted = append(tr.interpreted, h)
	return nil
}
func satGoalCapacity(old int, stride uint64) (int, error) {
	n := uint64(2)
	if old != 0 {
		n = (3*uint64(old) + 1) >> 1
	}
	if n <= uint64(old) || n > uint64(^uint32(0)) || n > uint64(^uint(0)>>1)/stride {
		return 0, arena.ErrBounds
	}
	return int(n), nil
}
func (tr *satGoalTranslator) pushFrame(fr satGoalFrame) error {
	if len(tr.frames) == cap(tr.frames) {
		stride := uint64(reflect.TypeOf(fr).Size())
		n, e := satGoalCapacity(cap(tr.frames), stride)
		if e != nil {
			return e
		}
		if e = tr.solver.budget.Reserve(uint64(n) * stride); e != nil {
			return e
		}
		next := make([]satGoalFrame, len(tr.frames), n)
		copy(next, tr.frames)
		tr.solver.budget.Release(uint64(cap(tr.frames)) * stride)
		tr.frames = next
	}
	tr.frames = append(tr.frames, fr)
	return nil
}
func (tr *satGoalTranslator) pushResult(l satLiteral) error {
	if len(tr.results) == cap(tr.results) {
		n, e := satGoalCapacity(cap(tr.results), 4)
		if e != nil {
			return e
		}
		if e = tr.solver.budget.Reserve(uint64(n) * 4); e != nil {
			return e
		}
		next := make([]satLiteral, len(tr.results), n)
		copy(next, tr.results)
		tr.solver.budget.Release(uint64(cap(tr.results)) * 4)
		tr.results = next
	}
	tr.results = append(tr.results, l)
	return nil
}
func (tr *satGoalTranslator) Variable(h ASTHandle) (satBoolVar, error) {
	if e := tr.open(); e != nil {
		return satNullBoolVar, e
	}
	hash, e := tr.manager.GetHash(h)
	if e != nil {
		return satNullBoolVar, e
	}
	if v, found := tr.atoms.find(h, hash); found {
		return satBoolVar(v), nil
	}
	return satNullBoolVar, nil
}
func (tr *satGoalTranslator) isBool(h ASTHandle) (bool, error) {
	sort, e := tr.manager.GetSort(h)
	return sort == tr.boolSort, e
}
func (tr *satGoalTranslator) isUninterpretedConstant(h ASTHandle) (bool, error) {
	k, e := tr.manager.GetKind(h)
	if e != nil || k != ASTApp {
		return false, e
	}
	n, e := tr.manager.GetNumArgs(h)
	if e != nil || n != 0 {
		return false, e
	}
	family, e := tr.manager.GetFamilyID(h)
	return family == -1, e
}
func (tr *satGoalTranslator) emit(root, sign bool, l satLiteral) error {
	if sign {
		l = l.negated()
	}
	if root {
		return tr.solver.MkClause([]satLiteral{l})
	}
	return tr.pushResult(l)
}
func (tr *satGoalTranslator) cached(h ASTHandle, root, sign bool) (bool, error) {
	hash, e := tr.manager.GetHash(h)
	if e != nil {
		return false, e
	}
	v, found := tr.cache.find(h, hash)
	if !found {
		return false, nil
	}
	return true, tr.emit(root, sign, satLiteral(v))
}
func (tr *satGoalTranslator) convertAtom(h ASTHandle, root, sign bool) error {
	isBool, e := tr.isBool(h)
	if e != nil {
		return e
	}
	if !isBool {
		return ErrASTDomain
	}
	hash, e := tr.manager.GetHash(h)
	if e != nil {
		return e
	}
	value, found := tr.atoms.find(h, hash)
	var l satLiteral
	if found {
		l = satMakeLiteral(satBoolVar(value), false)
	} else {
		// Compare canonical basic-plugin nodes, exactly as is_true/is_false.
		if h == tr.manager.external(tr.manager.trueApp) || h == tr.manager.external(tr.manager.falseApp) {
			if tr.trueVar == satNullBoolVar {
				tr.trueVar, e = tr.solver.MkVar(false, true)
				if e != nil {
					return e
				}
				if e = tr.solver.MkClause([]satLiteral{satMakeLiteral(tr.trueVar, false)}); e != nil {
					return e
				}
			}
			l = satMakeLiteral(tr.trueVar, h == tr.manager.external(tr.manager.falseApp))
		} else {
			uninterpreted, e := tr.isUninterpretedConstant(h)
			if e != nil {
				return e
			}
			_, inInterface := tr.interfaces.find(h, hash)
			v, e := tr.solver.MkVar(tr.defaultExternal || !uninterpreted || inInterface, true)
			if e != nil {
				return e
			}
			if e = tr.atoms.prepare(tr.solver.budget); e != nil {
				return e
			}
			if e = tr.manager.IncRef(h); e != nil {
				return e
			}
			tr.atoms.insertPrepared(h, hash, uint32(v))
			l = satMakeLiteral(v, false)
			if !uninterpreted {
				if e = tr.retainInterpreted(h); e != nil {
					return e
				}
			}
		}
	}
	return tr.emit(root, sign, l)
}
func (tr *satGoalTranslator) visit(h ASTHandle, root, sign bool) (bool, error) {
	kind, e := tr.manager.GetKind(h)
	if e != nil {
		return true, e
	}
	if kind != ASTApp {
		return true, tr.convertAtom(h, root, sign)
	}
	if found, e := tr.cached(h, root, sign); found || e != nil {
		return true, e
	}
	family, e := tr.manager.GetFamilyID(h)
	if e != nil {
		return true, e
	}
	if family != 0 {
		return true, tr.convertAtom(h, root, sign)
	}
	op, e := tr.manager.GetDeclKind(h)
	if e != nil {
		return true, e
	}
	switch op {
	case ASTOpNot, ASTOpOr, ASTOpAnd, ASTOpIff:
	case ASTOpIte, ASTOpEq:
		arg, e := tr.manager.GetArg(h, 1)
		if e != nil {
			return true, e
		}
		ok, e := tr.isBool(arg)
		if e != nil {
			return true, e
		}
		if !ok {
			return true, tr.convertAtom(h, root, sign)
		}
	case ASTOpXor, ASTOpImplies, ASTOpDistinct:
		// The source SMT2 pretty-printer message is a separate dependency. Carry
		// the offending AST and operation rather than inventing its printed text.
		return true, fmt.Errorf("%w (AST %d, operation %d)", ErrSATGoalUnsupported, h.Slot(), op)
	default:
		return true, tr.convertAtom(h, root, sign)
	}
	return false, tr.pushFrame(satGoalFrame{term: h, root: root, sign: sign})
}
func (tr *satGoalTranslator) Compile(formulas []ASTHandle) (err error) {
	if err = tr.open(); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tr.failed = true
		}
	}()
	tr.interfaces.clear()
	if err = tr.collectInterface(formulas); err != nil {
		return err
	}
	for _, h := range formulas {
		if err = tr.process(h); err != nil {
			return err
		}
	}
	return nil
}
func (tr *satGoalTranslator) process(h ASTHandle) error {
	done, e := tr.visit(h, true, false)
	if e != nil || done {
		return e
	}
	for len(tr.frames) != 0 {
		if e = tr.cooperation.Cooperate("goal2sat"); e != nil {
			return e
		}
		if !tr.limit.inc() {
			return fmt.Errorf("goal2sat: %s", tr.limit.cancelMessage())
		}
		size, e := tr.solver.runtime.memory.AllocationSize()
		if e != nil {
			return e
		}
		if size > tr.maxMemory {
			return fmt.Errorf("goal2sat: max. memory exceeded")
		}
		i := len(tr.frames) - 1
		fr := tr.frames[i]
		if fr.index == 0 {
			if found, e := tr.cached(fr.term, fr.root, fr.sign); found || e != nil {
				if e != nil {
					return e
				}
				tr.frames = tr.frames[:i]
				continue
			}
		}
		op, e := tr.manager.GetDeclKind(fr.term)
		if e != nil {
			return e
		}
		if op == ASTOpNot {
			tr.frames = tr.frames[:i]
			arg, e := tr.manager.GetArg(fr.term, 0)
			if e != nil {
				return e
			}
			if _, e = tr.visit(arg, fr.root, !fr.sign); e != nil {
				return e
			}
			continue
		}
		n, e := tr.manager.GetNumArgs(fr.term)
		if e != nil {
			return e
		}
		descended := false
		for tr.frames[i].index < n {
			arg, e := tr.manager.GetArg(fr.term, tr.frames[i].index)
			if e != nil {
				return e
			}
			tr.frames[i].index++
			done, e := tr.visit(arg, false, false)
			if e != nil {
				return e
			}
			if !done {
				descended = true
				break
			}
		}
		if descended {
			continue
		}
		if e = tr.convert(fr, op, int(n)); e != nil {
			return e
		}
		tr.frames = tr.frames[:i]
	}
	if len(tr.results) != 0 {
		return ErrASTDomain
	}
	return nil
}
func (tr *satGoalTranslator) convert(fr satGoalFrame, op int32, n int) error {
	sz := len(tr.results)
	if sz < n || fr.root && sz != n {
		return ErrASTDomain
	}
	base := sz - n
	var l satLiteral
	if !fr.root {
		v, e := tr.solver.MkVar(false, true)
		if e != nil {
			return e
		}
		l = satMakeLiteral(v, false)
		hash, e := tr.manager.GetHash(fr.term)
		if e != nil {
			return e
		}
		if e = tr.cache.prepare(tr.solver.budget); e != nil {
			return e
		}
		tr.cache.insertPrepared(fr.term, hash, uint32(l))
	}
	clause := tr.solver.MkClause
	switch op {
	case ASTOpOr, ASTOpAnd:
		if fr.root {
			units := op == ASTOpAnd && !fr.sign || op == ASTOpOr && fr.sign
			if units {
				for _, child := range tr.results {
					if fr.sign {
						child = child.negated()
					}
					if e := clause([]satLiteral{child}); e != nil {
						return e
					}
				}
			} else {
				if fr.sign {
					for i := range tr.results {
						tr.results[i] = tr.results[i].negated()
					}
				}
				if e := clause(tr.results); e != nil {
					return e
				}
			}
		} else {
			for _, child := range tr.results[base:] {
				pair := [2]satLiteral{child.negated(), l}
				if op == ASTOpAnd {
					pair = [2]satLiteral{l.negated(), child}
				}
				if e := clause(pair[:]); e != nil {
					return e
				}
			}
			extra := l.negated()
			if op == ASTOpAnd {
				extra = l
				for i := base; i < sz; i++ {
					tr.results[i] = tr.results[i].negated()
				}
			}
			if e := tr.pushResult(extra); e != nil {
				return e
			}
			if e := clause(tr.results[base:]); e != nil {
				return e
			}
		}
	case ASTOpIte:
		if n != 3 {
			return ErrASTDomain
		}
		c, t, e := tr.results[base], tr.results[base+1], tr.results[base+2]
		if fr.root {
			if fr.sign {
				t = t.negated()
				e = e.negated()
			}
			if err := clause([]satLiteral{c.negated(), t}); err != nil {
				return err
			}
			if err := clause([]satLiteral{c, e}); err != nil {
				return err
			}
		} else {
			triples := [6][3]satLiteral{{l.negated(), c.negated(), t}, {l.negated(), c, e}, {l, c.negated(), t.negated()}, {l, c, e.negated()}, {t.negated(), e.negated(), l}, {t, e, l.negated()}}
			count := 4
			if tr.iteExtra {
				count = 6
			}
			for i := 0; i < count; i++ {
				if err := clause(triples[i][:]); err != nil {
					return err
				}
			}
		}
	case ASTOpIff, ASTOpEq:
		if n != 2 {
			return ErrASTDomain
		}
		l1, l2 := tr.results[base+1], tr.results[base]
		if fr.root {
			pairs := [2][2]satLiteral{{l1, l2.negated()}, {l1.negated(), l2}}
			if fr.sign {
				pairs = [2][2]satLiteral{{l1, l2}, {l1.negated(), l2.negated()}}
			}
			for i := range pairs {
				if e := clause(pairs[i][:]); e != nil {
					return e
				}
			}
		} else {
			triples := [4][3]satLiteral{{l.negated(), l1, l2.negated()}, {l.negated(), l1.negated(), l2}, {l, l1, l2}, {l, l1.negated(), l2.negated()}}
			for i := range triples {
				if e := clause(triples[i][:]); e != nil {
					return e
				}
			}
		}
	default:
		return ErrASTDomain
	}
	tr.results = tr.results[:base]
	if !fr.root {
		return tr.emit(false, fr.sign, l)
	}
	return nil
}
