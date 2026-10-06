// Licensed under the MIT license; see LICENSE.txt.
// Source: src/ast/ast_ll_pp.cpp:21-320; src/ast/for_each_ast.h:27-125;
// src/ast/for_each_ast.cpp:37-48; src/util/vector.h:43-125,303-355.
// Source author: Leonardo de Moura (2006-10-18; 2006-10-19).
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/mpnum"
	"math"
)

// Actual lexical headers live inline in the charged manager. The pointer stack
// has only scalar slot/generation words in the existing global vector provider;
// it neither INCs nodes nor owns a second shadow slice of the traversal.
type astLowLevelFrame struct {
	stack, pending                       arena.SmallBytesRef
	capacity, pendingCapacity            uint32
	marks                                astMarkState
	invocation                           uint64
	active, sourceDone                   bool
	bodyCause, destructorCause, ownCause error
	numeral                              mpnum.OwnedRational
	numeralAcquired                      bool
	numeralInsertion                     mpnum.NumeralInsertionReceipt
}
type astLowLevelPrinter struct {
	manager            *ASTManager
	out                *NumericTextStream
	root               astNodeRef
	onlyExprs, compact bool
	arithFamily        FamilyID
	frame              *astLowLevelFrame
}

func (m *ASTManager) llStackSize(f *astLowLevelFrame) (uint32, error) {
	if f.stack == (arena.SmallBytesRef{}) {
		return 0, nil
	}
	return m.vectorWord(f.stack, 4)
}
func (m *ASTManager) llStackCapacity(f *astLowLevelFrame) (uint32, error) {
	if f.stack == (arena.SmallBytesRef{}) {
		return 0, nil
	}
	return m.vectorWord(f.stack, 0)
}
func (m *ASTManager) llStackPush(f *astLowLevelFrame, r astNodeRef) error {
	size, e := m.llStackSize(f)
	if e != nil {
		return e
	}
	capacity, e := m.llStackCapacity(f)
	if e != nil {
		return e
	}
	if size == capacity {
		n := uint32(2)
		if f.stack != (arena.SmallBytesRef{}) {
			n = (3*capacity + 1) >> 1
			if n <= capacity {
				return arena.ErrBounds
			}
		}
		bytes := uint64(n)*8 + 8
		if bytes > math.MaxUint32 {
			return arena.ErrBounds
		}
		block, e := m.vectorAllocator.Allocate(uint32(bytes))
		if e != nil {
			return e
		}
		f.pending = block
		f.pendingCapacity = n
		if e = m.vectorPutWord(block, 0, n); e == nil {
			e = m.vectorPutWord(block, 4, size)
		}
		for i := uint32(0); e == nil && i < size; i++ {
			var x astNodeRef
			x, e = m.llStackAt(f, i)
			if e == nil {
				e = m.vectorPutWord(block, 8+8*i, x.slot)
			}
			if e == nil {
				e = m.vectorPutWord(block, 12+8*i, x.generation)
			}
		}
		if e != nil {
			return e
		}
		if f.stack != (arena.SmallBytesRef{}) {
			if e = m.vectorAllocator.Deallocate(f.capacity*8+8, f.stack); e != nil {
				return e
			}
		}
		f.stack = block
		f.capacity = n
		f.pending = arena.SmallBytesRef{}
		f.pendingCapacity = 0
	}
	if e = m.vectorPutWord(f.stack, 8+8*size, r.slot); e != nil {
		return e
	}
	if e = m.vectorPutWord(f.stack, 12+8*size, r.generation); e != nil {
		return e
	}
	return m.vectorPutWord(f.stack, 4, size+1)
}
func (m *ASTManager) llStackAt(f *astLowLevelFrame, i uint32) (astNodeRef, error) {
	size, e := m.llStackSize(f)
	if e != nil {
		return astNodeRef{}, e
	}
	if i >= size {
		return astNodeRef{}, arena.ErrBounds
	}
	slot, e := m.vectorWord(f.stack, 8+8*i)
	if e != nil {
		return astNodeRef{}, e
	}
	generation, e := m.vectorWord(f.stack, 12+8*i)
	return astNodeRef{slot, generation}, e
}
func (m *ASTManager) llStackPop(f *astLowLevelFrame) error {
	size, e := m.llStackSize(f)
	if e != nil {
		return e
	}
	if size == 0 {
		return ErrASTDomain
	}
	return m.vectorPutWord(f.stack, 4, size-1)
}
func (m *ASTManager) llFrameDestroy(owning bool) error {
	f := &m.lowLevel
	if !f.active {
		return nil
	}
	if !owning && f.destructorCause != nil {
		return f.destructorCause
	}
	stop := func(e error) error {
		if e != nil {
			if owning {
				if f.ownCause == nil {
					f.ownCause = e
				}
			} else if f.destructorCause == nil {
				f.destructorCause = e
			}
		}
		return e
	}
	if owning && f.numeralInsertion.Active() {
		if e := m.numerals.CompleteNumeralInsertion(&f.numeralInsertion, false); e != nil {
			return stop(e)
		}
	}
	if f.numeralAcquired {
		if e := m.numerals.Del(&f.numeral); e != nil {
			return stop(e)
		}
		f.numeralAcquired = false
	}
	if f.capacity != 0 {
		if e := m.vectorAllocator.Deallocate(f.capacity*8+8, f.stack); e != nil {
			return stop(e)
		}
		f.stack = arena.SmallBytesRef{}
		f.capacity = 0
	}
	if owning && f.pendingCapacity != 0 {
		if e := m.vectorAllocator.Deallocate(f.pendingCapacity*8+8, f.pending); e != nil {
			return stop(e)
		}
		f.pending = arena.SmallBytesRef{}
		f.pendingCapacity = 0
	}
	// A retained unpublished allocation is own-only and protects this frame's
	// metadata; source destruction never claims it by scanning provider roots.
	f.marks.decls.close(m)
	f.marks.exprs.close(m)
	f.sourceDone = true
	if owning && f.numeralInsertion.Active() {
		if e := m.numerals.CompleteNumeralInsertion(&f.numeralInsertion, true); e != nil {
			return stop(e)
		}
	}
	if f.pendingCapacity == 0 && !f.numeralInsertion.Active() {
		f.active = false
	}
	return nil
}
func (m *ASTManager) lowLevelPrint(out *NumericTextStream, r astNodeRef, onlyExprs, compact bool) (result error) {
	f := &m.lowLevel
	if f.active {
		return ErrASTPending
	}
	if f.invocation == math.MaxUint64 {
		return arena.ErrBounds
	}
	*f = astLowLevelFrame{invocation: f.invocation + 1, active: true}
	m.traceExecuting = true
	defer func() { m.traceExecuting = false }()
	symbol, e := m.symbols.Intern("arith")
	if e != nil {
		f.active = false
		return e
	}
	fid, e := m.families.MkID(symbol)
	if e != nil {
		f.active = false
		return e
	}
	p := astLowLevelPrinter{manager: m, out: out, root: r, onlyExprs: onlyExprs, compact: compact, arithFamily: fid, frame: f}
	defer func() {
		if result != nil && f.bodyCause == nil {
			f.bodyCause = result
		}
		result = errors.Join(result, m.llFrameDestroy(false))
	}()
	if e := m.validNode(r); e != nil {
		return e
	}
	if m.nodeAt(r.slot).kind == ASTSort {
		return p.sort(r)
	}
	if e := m.llStackPush(f, r); e != nil {
		return e
	}
	for {
		size, e := m.llStackSize(f)
		if e != nil {
			return e
		}
		if size == 0 {
			break
		}
		curr, e := m.llStackAt(f, size-1)
		if e != nil {
			return e
		}
		if e = m.validNode(curr); e != nil {
			return e
		}
		marked, e := f.marks.marked(m, m.external(curr))
		if e != nil {
			return e
		}
		if marked {
			if e = m.llStackPop(f); e != nil {
				return e
			}
			continue
		}
		complete := true
		switch m.nodeAt(curr.slot).kind {
		case ASTSort, ASTFuncDecl:
			info, e := p.info(curr)
			if e != nil {
				return e
			}
			if info != nil {
				complete, e = p.parameters(info)
				if e != nil {
					return e
				}
			}
			if !complete {
				continue
			}
			if m.nodeAt(curr.slot).kind == ASTFuncDecl {
				d := m.declAt(curr.slot)
				complete, e = p.arguments(d.domain, d.arity)
				if e != nil {
					return e
				}
				if !complete {
					continue
				}
				yes, e := f.marks.marked(m, m.external(d.rangeSort))
				if e != nil {
					return e
				}
				if !yes {
					if e = m.llStackPush(f, d.rangeSort); e != nil {
						return e
					}
					continue
				}
			}
		case ASTApp:
			a := m.appAt(curr.slot)
			yes, e := f.marks.marked(m, m.external(a.decl))
			if e != nil {
				return e
			}
			if !yes {
				if e = m.llStackPush(f, a.decl); e != nil {
					return e
				}
				continue
			}
			complete, e = p.arguments(a.args, a.count)
			if e != nil {
				return e
			}
			if !complete {
				continue
			}
		case ASTQuantifier:
			q := m.quantifierAt(curr.slot)
			complete, e = p.arguments(q.patterns, q.patternCount)
			if e != nil {
				return e
			}
			if !complete {
				continue
			}
			complete, e = p.arguments(q.noPatterns, q.noPatternCount)
			if e != nil {
				return e
			}
			if !complete {
				continue
			}
			yes, e := f.marks.marked(m, m.external(q.body))
			if e != nil {
				return e
			}
			if !yes {
				if e = m.llStackPush(f, q.body); e != nil {
					return e
				}
				continue
			}
		case ASTVar:
		default:
			return ErrASTDomain
		}
		if e = p.process(curr); e != nil {
			return e
		}
		if e = f.marks.mark(m, m.external(curr), true); e != nil {
			return e
		}
		if e = m.llStackPop(f); e != nil {
			return e
		}
	}
	return nil
}
func (p *astLowLevelPrinter) arguments(raw arena.SmallBytesRef, n uint32) (bool, error) {
	all := true
	for i := uint32(0); i < n; i++ {
		r, e := p.manager.readInternal(raw, i)
		if e != nil {
			return false, e
		}
		if e = p.manager.validNode(r); e != nil {
			return false, e
		}
		yes, e := p.frame.marks.marked(p.manager, p.manager.external(r))
		if e != nil {
			return false, e
		}
		if !yes {
			if e = p.manager.llStackPush(p.frame, r); e != nil {
				return false, e
			}
			all = false
		}
	}
	return all, nil
}
func (p *astLowLevelPrinter) info(r astNodeRef) (*astInfoRecord, error) {
	m := p.manager
	switch m.nodeAt(r.slot).kind {
	case ASTSort:
		t := m.sortAt(r.slot).info
		if t == (ASTSortInfo{}) {
			return nil, nil
		}
		return m.sortInfo(t)
	case ASTFuncDecl:
		t := m.declAt(r.slot).info
		if t == (ASTFuncDeclInfo{}) {
			return nil, nil
		}
		return m.funcInfo(t)
	}
	return nil, ErrASTDomain
}
func (p *astLowLevelPrinter) parameters(info *astInfoRecord) (bool, error) {
	all := true
	for i := uint32(0); i < info.count; i++ {
		token, e := p.manager.infoParameter(info, i)
		if e != nil {
			return false, e
		}
		v, e := p.manager.parameter(token)
		if e != nil {
			return false, e
		}
		if v.kind == ASTParameterAST {
			r := v.ast
			if e = p.manager.validNode(r); e != nil {
				return false, e
			}
			yes, e := p.frame.marks.marked(p.manager, p.manager.external(r))
			if e != nil {
				return false, e
			}
			if !yes {
				if e = p.manager.llStackPush(p.frame, r); e != nil {
					return false, e
				}
				all = false
			}
		}
	}
	return all, nil
}
func (p *astLowLevelPrinter) symbol(s Symbol) error { return p.out.appendASTSymbol(p.manager, s) }
func (p *astLowLevelPrinter) ref(r astNodeRef) error {
	if e := p.out.AppendCString("#"); e != nil {
		return e
	}
	if e := p.manager.validNode(r); e != nil {
		return e
	}
	return p.out.AppendUint32(p.manager.nodeAt(r.slot).id)
}
func (p *astLowLevelPrinter) header(r astNodeRef) error {
	if r != p.root {
		if e := p.ref(r); e != nil {
			return e
		}
		return p.out.AppendCString(" := ")
	}
	return nil
}
func (p *astLowLevelPrinter) name(r astNodeRef) error {
	d := p.manager.declAt(r.slot)
	flags, e := p.manager.funcFlags(r)
	if e != nil {
		return e
	}
	if flags.Skolem && d.name.IsNumerical() {
		if e = p.out.AppendCString("z3.sk."); e != nil {
			return e
		}
		return p.out.AppendUint32(d.name.value)
	}
	return p.symbol(d.name)
}
func (p *astLowLevelPrinter) sort(r astNodeRef) error {
	if e := p.symbol(p.manager.sortAt(r.slot).name); e != nil {
		return e
	}
	return p.displayParams(r)
}
func (p *astLowLevelPrinter) displayParams(r astNodeRef) error {
	info, e := p.info(r)
	if e != nil {
		return e
	}
	if info == nil || info.count == 0 || info.private {
		return nil
	}
	if e = p.out.AppendCString("["); e != nil {
		return e
	}
	for i := uint32(0); i < info.count; i++ {
		t, e := p.manager.infoParameter(info, i)
		if e != nil {
			return e
		}
		v, e := p.manager.parameter(t)
		if e != nil {
			return e
		}
		if v.kind == ASTParameterAST {
			e = p.child(v.ast)
		} else {
			e = p.parameter(v)
		}
		if e != nil {
			return e
		}
		if i < info.count-1 {
			if e = p.out.AppendCString(":"); e != nil {
				return e
			}
		} else {
			if e = p.out.AppendCString(""); e != nil {
				return e
			}
		}
	}
	return p.out.AppendCString("]")
}
func (p *astLowLevelPrinter) parameter(v *astParameterRecord) error {
	switch v.kind {
	case ASTParameterInt:
		return p.out.AppendInt32(v.integer)
	case ASTParameterAST:
		return p.ref(v.ast)
	case ASTParameterSymbol:
		return p.symbol(v.symbol)
	case ASTParameterRational:
		return p.manager.numerals.InsertInto(p.out, &v.rational)
	case ASTParameterDouble:
		return p.out.AppendFloat64(v.double)
	case ASTParameterExternal:
		if e := p.out.AppendCString("@"); e != nil {
			return e
		}
		return p.out.AppendUint32(v.external)
	}
	return ErrASTDomain
}
func (p *astLowLevelPrinter) isNumeral(r astNodeRef) (bool, error) {
	if p.manager.nodeAt(r.slot).kind != ASTApp {
		return false, nil
	}
	d := p.manager.declAt(p.manager.appAt(r.slot).decl.slot)
	if d.info == (ASTFuncDeclInfo{}) {
		return false, nil
	}
	info, e := p.manager.funcInfo(d.info)
	return e == nil && info.family == p.arithFamily && info.kind == 0, e
}
func (p *astLowLevelPrinter) numeral(r astNodeRef) (yes bool, result error) {
	f := p.frame
	m := p.manager
	if f.numeralAcquired {
		return false, ErrASTPending
	}
	v, e := m.numerals.NewValue()
	if e != nil {
		return false, e
	}
	f.numeral = v
	f.numeralAcquired = true
	defer func() {
		var younger *mpnum.NumeralSourceFailure
		if errors.As(result, &younger) {
			if f.destructorCause == nil {
				f.destructorCause = younger
			}
			return // younger source DT protects this still-owned older NUM
		}
		cleanup := m.numerals.Del(&f.numeral)
		if cleanup == nil {
			f.numeralAcquired = false
		} else if f.destructorCause == nil {
			f.destructorCause = cleanup
		}
		result = errors.Join(result, cleanup)
	}()
	yes, e = p.isNumeral(r)
	if e != nil || !yes {
		return yes, e
	}
	info, e := m.funcInfo(m.declAt(m.appAt(r.slot).decl.slot).info)
	if e != nil {
		return true, e
	}
	token, e := m.infoParameter(info, 0)
	if e != nil {
		return true, e
	}
	rat, e := m.parameter(token)
	if e != nil {
		return true, e
	}
	if rat.kind != ASTParameterRational {
		return true, ErrASTDomain
	}
	if e = m.numerals.CopyFrom(&f.numeral, m.numerals, &rat.rational); e != nil {
		return true, e
	}
	token, e = m.infoParameter(info, 1)
	if e != nil {
		return true, e
	}
	isInt, e := m.parameter(token)
	if e != nil {
		return true, e
	}
	if isInt.kind != ASTParameterInt {
		return true, ErrASTDomain
	}
	if e = m.numerals.InsertIntoTracked(p.out, &f.numeral, &f.numeralInsertion); e != nil {
		return true, e
	}
	if e = p.out.AppendCString("::"); e != nil {
		return true, e
	}
	if isInt.integer != 0 {
		return true, p.out.AppendCString("Int")
	}
	return true, p.out.AppendCString("Real")
}
func (p *astLowLevelPrinter) child(r astNodeRef) error {
	if e := p.manager.validNode(r); e != nil {
		return e
	}
	switch p.manager.nodeAt(r.slot).kind {
	case ASTSort:
		return p.sort(r)
	case ASTApp:
		yes, e := p.numeral(r)
		if e != nil {
			return e
		}
		if yes {
			return nil
		}
		a := p.manager.appAt(r.slot)
		if a.count == 0 {
			if e = p.name(a.decl); e != nil {
				return e
			}
			return p.displayParams(a.decl)
		}
	}
	return p.ref(r)
}
func (p *astLowLevelPrinter) children(raw arena.SmallBytesRef, n uint32) error {
	for i := uint32(0); i < n; i++ {
		if i > 0 {
			if e := p.out.AppendCString(" "); e != nil {
				return e
			}
		}
		r, e := p.manager.readInternal(raw, i)
		if e != nil {
			return e
		}
		if e = p.child(r); e != nil {
			return e
		}
	}
	return nil
}
func (p *astLowLevelPrinter) process(r astNodeRef) error {
	m := p.manager
	o := p.out
	switch m.nodeAt(r.slot).kind {
	case ASTSort:
		return nil
	case ASTFuncDecl:
		if p.onlyExprs {
			return nil
		}
		d := m.declAt(r.slot)
		info, e := p.info(r)
		if e != nil {
			return e
		}
		if info != nil && info.family != NullFamilyID {
			return nil
		}
		if e = o.AppendCString("decl "); e != nil {
			return e
		}
		if e = p.name(r); e != nil {
			return e
		}
		if e = o.AppendCString(" :: "); e != nil {
			return e
		}
		if d.arity == 0 {
			if e = p.child(d.rangeSort); e != nil {
				return e
			}
		} else {
			if e = o.AppendCString("(-> "); e != nil {
				return e
			}
			if e = p.children(d.domain, d.arity); e != nil {
				return e
			}
			if e = o.AppendCString(" "); e != nil {
				return e
			}
			if e = p.child(d.rangeSort); e != nil {
				return e
			}
			if e = o.AppendCString(")"); e != nil {
				return e
			}
			if e = p.displayParams(r); e != nil {
				return e
			}
			flags, e := m.funcFlags(r)
			if e != nil {
				return e
			}
			if flags.LeftAssociative && flags.RightAssociative {
				if e = o.AppendCString(" :assoc"); e != nil {
					return e
				}
			}
			if flags.Commutative {
				if e = o.AppendCString(" :comm"); e != nil {
					return e
				}
			}
			if flags.Injective {
				if e = o.AppendCString(" :inj"); e != nil {
					return e
				}
			}
		}
		return o.AppendCString("\n")
	case ASTVar:
		if e := p.header(r); e != nil {
			return e
		}
		if e := o.AppendCString("(:var "); e != nil {
			return e
		}
		v := m.varAt(r.slot)
		if e := o.AppendUint32(v.index); e != nil {
			return e
		}
		if e := o.AppendCString(" "); e != nil {
			return e
		}
		if e := p.sort(v.sort); e != nil {
			return e
		}
		return o.AppendCString(")\n")
	case ASTApp:
		numeric, e := p.isNumeral(r)
		if e != nil {
			return e
		}
		if numeric {
			if !p.compact {
				if e = p.header(r); e != nil {
					return e
				}
			}
			if r == p.root || !p.compact {
				if _, e = p.numeral(r); e != nil {
					return e
				}
				return o.AppendCString("\n")
			}
			return nil
		}
		proof, e := m.proofIs(m.external(r))
		if e != nil {
			return e
		}
		a := m.appAt(r.slot)
		if proof {
			if e = p.header(r); e != nil {
				return e
			}
			if e = o.AppendCString("["); e != nil {
				return e
			}
			if e = p.symbol(m.declAt(a.decl.slot).name); e != nil {
				return e
			}
			info, e := p.info(a.decl)
			if e != nil {
				return e
			}
			if info != nil {
				for i := uint32(0); i < info.count; i++ {
					if e = o.AppendCString(" "); e != nil {
						return e
					}
					token, e := m.infoParameter(info, i)
					if e != nil {
						return e
					}
					v, e := m.parameter(token)
					if e != nil {
						return e
					}
					if e = p.parameter(v); e != nil {
						return e
					}
				}
			}
			has, e := m.proofHasFact(m.external(r))
			if e != nil {
				return e
			}
			count := a.count
			if has {
				count--
			}
			for i := uint32(0); i < count; i++ {
				if e = o.AppendCString(" "); e != nil {
					return e
				}
				child, e := m.readInternal(a.args, i)
				if e != nil {
					return e
				}
				if e = p.child(child); e != nil {
					return e
				}
			}
			if e = o.AppendCString("]: "); e != nil {
				return e
			}
			if has {
				fact, e := m.proofFact(m.external(r))
				if e != nil {
					return e
				}
				if e = p.child(astNodeRef{fact.slot, fact.generation}); e != nil {
					return e
				}
			} else {
				if e = o.AppendCString("*"); e != nil {
					return e
				}
			}
			return o.AppendCString("\n")
		}
		if p.compact && a.count == 0 {
			if r == p.root {
				if e = p.child(r); e != nil {
					return e
				}
				return o.AppendCString("\n")
			}
			return nil
		}
		if e = p.header(r); e != nil {
			return e
		}
		if a.count > 0 {
			if e = o.AppendCString("("); e != nil {
				return e
			}
		}
		if e = p.name(a.decl); e != nil {
			return e
		}
		if e = p.displayParams(a.decl); e != nil {
			return e
		}
		if a.count > 0 {
			if e = o.AppendCString(" "); e != nil {
				return e
			}
			if e = p.children(a.args, a.count); e != nil {
				return e
			}
			if e = o.AppendCString(")"); e != nil {
				return e
			}
		}
		return o.AppendCString("\n")
	case ASTQuantifier:
		if e := p.header(r); e != nil {
			return e
		}
		if e := o.AppendCString("("); e != nil {
			return e
		}
		q := m.quantifierAt(r.slot)
		word := "exists"
		if q.forall {
			word = "forall"
		}
		if e := o.AppendCString(word); e != nil {
			return e
		}
		if e := o.AppendCString(" "); e != nil {
			return e
		}
		if e := o.AppendCString("(vars "); e != nil {
			return e
		}
		for i := uint32(0); i < q.count; i++ {
			if i > 0 {
				if e := o.AppendCString(" "); e != nil {
					return e
				}
			}
			if e := o.AppendCString("("); e != nil {
				return e
			}
			name, e := m.readSymbol(q.names, i)
			if e != nil {
				return e
			}
			if e = p.symbol(name); e != nil {
				return e
			}
			if e = o.AppendCString(" "); e != nil {
				return e
			}
			sort, e := m.readInternal(q.sorts, i)
			if e != nil {
				return e
			}
			if e = p.sort(sort); e != nil {
				return e
			}
			if e = o.AppendCString(")"); e != nil {
				return e
			}
		}
		if e := o.AppendCString(") "); e != nil {
			return e
		}
		if q.patternCount > 0 {
			if e := o.AppendCString("(:pat "); e != nil {
				return e
			}
			if e := p.children(q.patterns, q.patternCount); e != nil {
				return e
			}
			if e := o.AppendCString(") "); e != nil {
				return e
			}
		}
		if q.noPatternCount > 0 {
			if e := o.AppendCString("(:nopat "); e != nil {
				return e
			}
			if e := p.children(q.noPatterns, q.noPatternCount); e != nil {
				return e
			}
			if e := o.AppendCString(") "); e != nil {
				return e
			}
		}
		if e := p.child(q.body); e != nil {
			return e
		}
		return o.AppendCString(")\n")
	}
	return ErrASTDomain
}

func (out *NumericTextStream) appendASTSymbol(m *ASTManager, s Symbol) error {
	if astSourceDebug && s.IsMarked() {
		return ErrASTDomain
	}
	if s.tag == 0 {
		if s.value == 0 {
			return out.AppendCString("null")
		}
		if s.owner != m.symbols.owner {
			return ErrSymbolOwner
		}
		return out.AppendCStringView(ParamsStringView{region: m.symbols.region, regionRef: arena.BytesRegionRef{Slot: arena.Slot(s.value), Generation: s.generation}, offset: m.symbols.pointerWidth, mode: 3, present: true})
	}
	if e := out.AppendCString("k!"); e != nil {
		return e
	}
	return out.AppendUint32(s.value)
}

// DisplayLowLevel is the concrete source ast_ll_pp call, borrowing the existing
// formatter; it neither changes its flags nor owns the stream.
func (m *ASTManager) DisplayLowLevel(out *NumericTextStream, h ASTHandle, onlyExprs, compact bool) error {
	if e := m.lock(); e != nil {
		return e
	}
	defer m.mu.Unlock()
	r, e := m.internal(h)
	if e != nil {
		return e
	}
	return m.lowLevelPrint(out, r, onlyExprs, compact)
}
