// Source: src/ast/format.cpp:119-145; src/ast/recurse_expr.h:28-60;
// src/ast/recurse_expr_def.h:25-107; src/util/obj_hashtable.h:65-125;
// src/util/hashtable.h:186-226,247-256,371-398,497-517.
// Source author: Leonardo de Moura (2008-01-11; 2008-01-20).
// Licensed under the MIT license; see LICENSE.txt.
package ast

import (
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

// Source vector<format*> and ptr_vector<expr> store only concrete scalar
// manager-local pointer representations. Capacity/size live in actual backing.
type astFormatRefVector struct{ words []uint32 }

func (v *astFormatRefVector) size() uint32 {
	if len(v.words) == 0 {
		return 0
	}
	return v.words[1]
}
func (v *astFormatRefVector) bytes() uint64 { return uint64(len(v.words)) * 4 }
func (v *astFormatRefVector) at(i uint32) (astNodeRef, error) {
	if len(v.words) == 0 || i >= v.words[1] {
		return astNodeRef{}, arena.ErrBounds
	}
	return astNodeRef{v.words[2+i*2], v.words[3+i*2]}, nil
}
func (v *astFormatRefVector) push(b *arena.Budget, r astNodeRef) error {
	size := v.size()
	var capacity uint32
	if len(v.words) != 0 {
		capacity = v.words[0]
	}
	if size == capacity {
		n := (uint64(capacity)*3 + 1) / 2
		if n < 2 {
			n = 2
		}
		if n > math.MaxUint32 || n*2+2 > uint64(^uint(0)>>1)/4 {
			return arena.ErrBounds
		}
		bytes := (n*2 + 2) * 4
		if e := b.Reserve(bytes); e != nil {
			return e
		}
		a := make([]uint32, int(n)*2+2)
		a[0] = uint32(n)
		a[1] = size
		for i := uint32(0); i < size*2; i++ {
			a[2+i] = v.words[2+i]
		}
		b.Release(v.bytes())
		v.words = a
	}
	v.words[2+size*2] = r.slot
	v.words[3+size*2] = r.generation
	v.words[1]++
	return nil
}
func (v *astFormatRefVector) pop() { v.words[1]-- }
func (v *astFormatRefVector) reset() {
	if len(v.words) != 0 {
		v.words[1] = 0
	}
}
func (v *astFormatRefVector) close(b *arena.Budget) { b.Release(v.bytes()); v.words = nil }

type astFormatMapEntry struct{ key, value astNodeRef }
type astFormatFlatFrame struct {
	next                             *astFormatFlatFrame
	owner                            *ASTManager
	cache, pending                   []astFormatMapEntry
	todo, results1, results2         astFormatRefVector
	capacity, size                   uint32
	family                           FamilyID
	header                           uint64
	invocation                       uint64
	bodyCause, sourceCause, ownCause error
	sourceDone, active               bool
}

func (f *astFormatFlatFrame) bytes() uint64 {
	return f.header + uint64(len(f.cache)+len(f.pending))*uint64(reflect.TypeOf(astFormatMapEntry{}).Size()) + f.todo.bytes() + f.results1.bytes() + f.results2.bytes()
}
func (f *astFormatFlatFrame) find(key astNodeRef) (astNodeRef, bool, error) {
	m := f.owner
	if e := m.validNode(key); e != nil {
		return astNodeRef{}, false, e
	}
	idx := m.nodeAt(key.slot).hash & (f.capacity - 1)
	// Literal suffix then prefix linear probe, object entry computes stored hash
	// through its actual key, never a duplicate cached-hash field.
	for step := uint32(0); step < f.capacity; step++ {
		i := idx + step
		if i >= f.capacity {
			i -= f.capacity
		}
		p := f.cache[i]
		if p.key.slot == 0 {
			return astNodeRef{}, false, nil
		}
		if e := m.validNode(p.key); e != nil {
			return astNodeRef{}, false, e
		}
		if m.nodeAt(p.key.slot).hash == m.nodeAt(key.slot).hash && p.key == key {
			return p.value, true, nil
		}
	}
	return astNodeRef{}, false, nil
}
func (f *astFormatFlatFrame) allocateCache(n uint32) ([]astFormatMapEntry, error) {
	width := uint64(reflect.TypeOf(astFormatMapEntry{}).Size())
	if uint64(n) > uint64(^uint(0)>>1)/width {
		return nil, arena.ErrBounds
	}
	if e := f.owner.BudgetField.Reserve(uint64(n) * width); e != nil {
		return nil, e
	}
	return make([]astFormatMapEntry, int(n)), nil
}
func (f *astFormatFlatFrame) insert(key, value astNodeRef) error {
	m := f.owner
	if f.size<<2 > f.capacity*3 {
		if f.capacity > math.MaxUint32/2 {
			return arena.ErrBounds
		}
		next, e := f.allocateCache(f.capacity << 1)
		if e != nil {
			return e
		}
		f.pending = next
		// Source NEW is captured before ascending move. A reached invalid-key guard
		// keeps that exact unjoined NEW named; source cache DT only releases OLD.
		for i := uint32(0); i < f.capacity; i++ {
			p := f.cache[i]
			if p.key.slot == 0 {
				continue
			}
			if e = m.validNode(p.key); e != nil {
				return e
			}
			start := m.nodeAt(p.key.slot).hash & (uint32(len(next)) - 1)
			found := false
			for step := uint32(0); step < uint32(len(next)); step++ {
				j := start + step
				if j >= uint32(len(next)) {
					j -= uint32(len(next))
				}
				if next[j].key.slot == 0 {
					next[j] = p
					found = true
					break
				}
			}
			if !found {
				return ErrASTDomain
			}
		}
		m.BudgetField.Release(uint64(len(f.cache)) * uint64(reflect.TypeOf(astFormatMapEntry{}).Size()))
		f.cache = next
		f.pending = nil
		f.capacity = uint32(len(next))
	}
	if e := m.validNode(key); e != nil {
		return e
	}
	hash := m.nodeAt(key.slot).hash
	start := hash & (f.capacity - 1)
	for step := uint32(0); step < f.capacity; step++ {
		i := start + step
		if i >= f.capacity {
			i -= f.capacity
		}
		p := &f.cache[i]
		if p.key.slot != 0 {
			if e := m.validNode(p.key); e != nil {
				return e
			}
			if m.nodeAt(p.key.slot).hash == hash && p.key == key {
				*p = astFormatMapEntry{key, value}
				return nil
			}
		} else {
			*p = astFormatMapEntry{key, value}
			f.size++
			return nil
		}
	}
	return ErrASTDomain
}
func (f *astFormatFlatFrame) sourceDestroy() {
	if f.sourceDone {
		return
	}
	b := f.owner.BudgetField
	// Actual reverse member DT: results2, results1, todo, cache. Pointer element
	// destruction is trivial; it grants no AST INC/DEC ownership.
	f.results2.close(b)
	f.results1.close(b)
	f.todo.close(b)
	b.Release(uint64(len(f.cache)) * uint64(reflect.TypeOf(astFormatMapEntry{}).Size()))
	f.cache = nil
	f.sourceDone = true
	f.active = false
}
func (m *ASTManager) formatFramesBytes() uint64 {
	var n uint64
	for f := m.formatFrames; f != nil; f = f.next {
		n += f.bytes()
	}
	return n
}
func (m *ASTManager) releaseFormatFrames() {
	for f := m.formatFrames; f != nil; {
		next := f.next
		f.sourceDestroy()
		m.BudgetField.Release(uint64(len(f.pending)) * uint64(reflect.TypeOf(astFormatMapEntry{}).Size()))
		f.pending = nil
		m.BudgetField.Release(f.header)
		f.header = 0
		f = next
	}
	m.formatFrames = nil
}
func (m *ASTManager) flatVisit(f *astFormatFlatFrame, n astNodeRef) (astNodeRef, error) {
	if m.nodeAt(n.slot).kind != ASTApp {
		return astNodeRef{}, ErrASTDomain
	} // selected source UNREACHABLE
	a := m.appAt(n.slot)
	id, kind, e := m.declIdentity(a.decl)
	if e != nil {
		return astNodeRef{}, e
	}
	if id == f.family && kind == ASTFormatLineBreak {
		name, e := m.Symbols.Intern(" ")
		if e != nil {
			return astNodeRef{}, e
		}
		p := astParameterRecord{Kind: ASTParameterSymbol, symbol: name}
		return m.formatApp(ASTFormatString, &p, nil)
	}
	if id == f.family && kind == ASTFormatLineBreakExt {
		info, e := m.funcInfo(m.declAt(a.decl.slot).info)
		if e != nil {
			return astNodeRef{}, e
		}
		p, e := m.infoParameter(info, 0)
		if e != nil {
			return astNodeRef{}, e
		}
		value, e := m.parameter(p)
		if e != nil {
			return astNodeRef{}, e
		}
		if value.Kind != ASTParameterSymbol {
			return astNodeRef{}, ErrASTDomain
		}
		// symbol bare_str access is performed before actual mk_string symbol ctor.
		name, e := m.Symbols.InternBorrowedSymbolCString(value.symbol)
		if e != nil {
			return astNodeRef{}, e
		}
		parameter := astParameterRecord{Kind: ASTParameterSymbol, symbol: name}
		return m.formatApp(ASTFormatString, &parameter, nil)
	}
	if id == f.family && kind == ASTFormatChoice {
		return m.readInternal(a.args, 0)
	}
	return m.mkAppArguments(a.decl, AppArguments{format: &f.results1, formatOwner: m, count: f.results1.size()})
}
func (m *ASTManager) formatFlat(root astNodeRef) (result astNodeRef, err error) {
	family, e := m.formatFamilyID()
	if e != nil {
		return astNodeRef{}, e
	}
	if m.formatFrameSerial == math.MaxUint64 {
		return astNodeRef{}, arena.ErrBounds
	}
	header := uint64(reflect.TypeOf(astFormatFlatFrame{}).Size())
	if e = m.BudgetField.Reserve(header); e != nil {
		return astNodeRef{}, e
	}
	m.formatFrameSerial++
	f := &astFormatFlatFrame{owner: m, family: family, header: header, invocation: m.formatFrameSerial, active: true, next: m.formatFrames}
	m.formatFrames = f
	defer func() {
		if err != nil && f.bodyCause == nil {
			f.bodyCause = err
		}
		f.sourceDestroy()
		if err == nil && len(f.pending) == 0 {
			m.formatFrames = f.next
			m.BudgetField.Release(f.header)
			f.header = 0
		}
	}()
	f.cache, e = f.allocateCache(64)
	if e != nil {
		return astNodeRef{}, e
	}
	f.capacity = 64
	if e = f.todo.push(m.BudgetField, root); e != nil {
		return astNodeRef{}, e
	}
	for f.todo.size() != 0 {
		n, e := f.todo.at(f.todo.size() - 1)
		if e != nil {
			return astNodeRef{}, e
		}
		_, found, e := f.find(n)
		if e != nil {
			return astNodeRef{}, e
		}
		if found {
			f.todo.pop()
			continue
		}
		if e = m.validNode(n); e != nil {
			return astNodeRef{}, e
		}
		visited := true
		switch m.nodeAt(n.slot).kind {
		case ASTApp:
			a := m.appAt(n.slot)
			for j := uint32(0); j < a.count; j++ {
				child, e := m.readInternal(a.args, j)
				if e != nil {
					return astNodeRef{}, e
				}
				_, found, e = f.find(child)
				if e != nil {
					return astNodeRef{}, e
				}
				if !found {
					if e = f.todo.push(m.BudgetField, child); e != nil {
						return astNodeRef{}, e
					}
					visited = false
				}
			}
		case ASTQuantifier:
			child := m.quantifierAt(n.slot).body
			_, found, e = f.find(child)
			if e != nil {
				return astNodeRef{}, e
			}
			if !found {
				if e = f.todo.push(m.BudgetField, child); e != nil {
					return astNodeRef{}, e
				}
				visited = false
			}
		}
		if !visited {
			continue
		}
		f.todo.pop()
		if m.nodeAt(n.slot).kind != ASTApp {
			return astNodeRef{}, ErrASTDomain
		}
		f.results1.reset()
		a := m.appAt(n.slot)
		for j := uint32(0); j < a.count; j++ {
			child, e := m.readInternal(a.args, j)
			if e != nil {
				return astNodeRef{}, e
			}
			value, found, e := f.find(child)
			if e != nil {
				return astNodeRef{}, e
			}
			if !found {
				return astNodeRef{}, ErrASTDomain
			}
			if e = f.results1.push(m.BudgetField, value); e != nil {
				return astNodeRef{}, e
			}
		}
		result, e = m.flatVisit(f, n)
		if e != nil {
			return astNodeRef{}, e
		}
		if e = f.insert(n, result); e != nil {
			return astNodeRef{}, e
		}
	}
	result, found, e := f.find(root)
	if e != nil {
		return astNodeRef{}, e
	}
	if !found {
		return astNodeRef{}, ErrASTDomain
	}
	return result, nil
}
func (m *ASTManager) FormatFlat(root ASTHandle) (ASTHandle, error) {
	f, e := m.lockFormat()
	if e != nil {
		return ASTHandle{}, e
	}
	defer f.unlockFormat()
	r, e := f.requireExpr(root)
	if e != nil {
		return ASTHandle{}, e
	}
	r, e = f.formatFlat(r)
	return f.external(r), e
}
func (m *ASTManager) FormatGroup(root ASTHandle) (ASTHandle, error) {
	f, e := m.lockFormat()
	if e != nil {
		return ASTHandle{}, e
	}
	defer f.unlockFormat()
	r, e := f.requireExpr(root)
	if e != nil {
		return ASTHandle{}, e
	}
	flat, e := f.formatFlat(r)
	if e != nil {
		return ASTHandle{}, e
	}
	args := [2]ASTHandle{f.external(flat), root}
	r, e = f.formatApp(ASTFormatChoice, nil, args[:])
	return f.external(r), e
}
