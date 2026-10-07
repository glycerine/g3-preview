// Source: src/ast/bv_decl_plugin.cpp:26-178; src/ast/bv_decl_plugin.h:143-208.
// Source author: Leonardo de Moura (2008-01-09).
// Licensed under the MIT license; see LICENSE.txt.
package ast

import (
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

type astBitVectorPluginOwner struct {
	literalPrimary                         astBVLiteralFrame
	literalFrames                          *astBVLiteralFrame
	literalSerial                          uint64
	numeralPrimary                         astBVNumeralFrame
	numeralFrames                          *astBVNumeralFrame
	numeralSerial                          uint64
	next                                   *astBitVectorPluginOwner
	self                                   *astBitVectorPluginOwner
	owner                                  *ASTManager
	names                                  [10]util.Symbol
	fields                                 [5]astNodeRef
	fieldAcquired                          [5]bool
	sortAcquired                           []byte // charged cold leaf receipts, not a second pointer table
	sorts                                  astFormatRefVector
	family                                 FamilyID
	header                                 uint64
	power                                  util.OwnedRational
	powerActive, published                 bool
	bodyCause, destructorCause, powerCause error
}

func (m *ASTManager) bitVectorPluginBytes() uint64 {
	var n uint64
	if p := m.bitVectorPlugin; p != nil {
		n += p.header + p.sorts.bytes() + p.numeralBytes() + p.literalBytes() + uint64(len(p.sortAcquired))
	}
	for p := m.bitVectorPluginFailed; p != nil; p = p.next {
		n += p.header + p.sorts.bytes() + p.numeralBytes() + p.literalBytes() + uint64(len(p.sortAcquired))
	}
	return n
}
func (m *ASTManager) releaseBitVectorPluginHeaders() {
	if p := m.bitVectorPlugin; p != nil {
		if !p.powerActive {
			p.releaseLiteralFrames()
			p.releaseNumeralFrames()
			p.sorts.close(m.BudgetField)
			m.BudgetField.Release(uint64(len(p.sortAcquired)))
			p.sortAcquired = nil
			m.BudgetField.Release(p.header)
			p.header = 0
			m.bitVectorPlugin = nil
		}
	}
	if m.sourceClosing {
		return
	}
	for p := m.bitVectorPluginFailed; p != nil; {
		next := p.next
		p.releaseLiteralFrames()
		p.releaseNumeralFrames()
		p.sorts.close(m.BudgetField)
		m.BudgetField.Release(uint64(len(p.sortAcquired)))
		p.sortAcquired = nil
		m.BudgetField.Release(p.header)
		p.header = 0
		p = next
	}
	m.bitVectorPluginFailed = nil
}
func (m *ASTManager) cleanupBitVectorPower() error {
	p := m.bitVectorPlugin
	if p == nil || !p.powerActive {
		return nil
	}
	e := m.numerals.Del(&p.power)
	if e != nil {
		if p.powerCause == nil {
			p.powerCause = e
		}
		return e
	}
	p.powerActive = false
	return nil
}
func (m *ASTManager) finalizeBitVectorPlugin() error {
	p := m.bitVectorPlugin
	if p == nil {
		return nil
	}
	if e := p.destroyLiteralFrames(!m.sourceClosing); e != nil {
		return e
	}
	if e := p.destroyNumeralFrames(!m.sourceClosing); e != nil {
		return e
	}
	for j := range p.fields {
		if p.fields[j].slot == 0 {
			continue
		}
		if !m.sourceClosing && !p.fieldAcquired[j] {
			// No owned reference was acquired at this source INC leaf. The manager's
			// actual table pass retains the node; own abort must not fabricate a DEC.
			p.fields[j] = astNodeRef{}
			continue
		}
		consumed, e := m.decNode(p.fields[j])
		if consumed {
			p.fields[j] = astNodeRef{}
			p.fieldAcquired[j] = false
		}
		if e != nil {
			if p.destructorCause == nil {
				p.destructorCause = e
			}
			return e
		}
	}
	for j := uint32(0); j < p.sorts.size(); j++ {
		r, e := p.sorts.at(j)
		if e != nil {
			return e
		}
		if r.slot == 0 {
			continue
		}
		if !m.sourceClosing && p.sortAcquired[j] == 0 {
			continue
		}
		consumed, e := m.decNode(r)
		if consumed {
			p.sorts.words[2+j*2] = 0
			p.sorts.words[3+j*2] = 0
			p.sortAcquired[j] = 0
		}
		if e != nil {
			if p.destructorCause == nil {
				p.destructorCause = e
			}
			return e
		}
	}
	// All remaining concrete source declaration-cache vectors are default-empty
	// in this constructor profile; no allocation/destruction leaf is fabricated.
	return nil
}
func (p *astBitVectorPluginOwner) mkSort(bits uint32) (astNodeRef, error) {
	m := p.owner
	if bits == math.MaxUint32 {
		return astNodeRef{}, ErrASTDomain
	}
	for p.sorts.size() < bits+1 {
		if e := p.pushSortSlot(); e != nil {
			return astNodeRef{}, e
		}
	}
	r, e := p.sorts.at(bits)
	if e != nil || r.slot != 0 {
		return r, e
	}
	if p.powerActive {
		return astNodeRef{}, ErrASTPending
	}
	size := VeryBigSortSize()
	if !SortSizeIsVeryBigBase2(bits) {
		power, e := m.numerals.PowerOfTwo(bits)
		if e != nil {
			return astNodeRef{}, e
		}
		// Actual successful returned numerical acquisition, not cache/count inference.
		p.power = power
		p.powerActive = true
		size, e = SortSizeFromRational(m.numerals, &p.power)
		if e != nil {
			return astNodeRef{}, e
		}
		if e = m.numerals.Del(&p.power); e != nil {
			if p.powerCause == nil {
				p.powerCause = e
			}
			return astNodeRef{}, e
		}
		p.powerActive = false
	}
	name, e := m.Symbols.Intern("bv")
	if e != nil {
		return astNodeRef{}, e
	}
	param := astParameterRecord{Kind: ASTParameterInt, integer: int32(bits)}
	if e = m.beginFormatInfo(true, p.family, 0, &param); e != nil {
		return astNodeRef{}, e
	}
	m.formatLexical.info.size = size
	r, e = m.mkSort(name, ASTSortInfo{m.origin, ^uint32(0) - 1, m.formatLexical.epoch})
	m.endFormatInfo()
	if e != nil {
		return astNodeRef{}, e
	}
	p.sorts.words[2+bits*2] = r.slot
	p.sorts.words[3+bits*2] = r.generation
	if e = m.incNode(r); e != nil {
		return r, e
	}
	p.sortAcquired[bits] = 1
	return r, nil
}
func (p *astBitVectorPluginOwner) setManager() error {
	m := p.owner
	for bits := uint32(1); bits <= 64; bits++ {
		if _, e := p.mkSort(bits); e != nil {
			return e
		}
	}
	one, e := p.sorts.at(1)
	if e != nil {
		return e
	}
	p.fields[0], e = m.makePluginDeclaration(p.family, "bit0", 2, nil, one, ASTFuncFlags{})
	if e != nil {
		return e
	}
	p.fields[1], e = m.makePluginDeclaration(p.family, "bit1", 1, nil, one, ASTFuncFlags{})
	if e != nil {
		return e
	}
	if e = m.incNode(p.fields[0]); e != nil {
		return e
	}
	p.fieldAcquired[0] = true
	if e = m.incNode(p.fields[1]); e != nil {
		return e
	}
	p.fieldAcquired[1] = true
	b := m.basic.boolSort
	domain := [3]astNodeRef{b, b, b}
	p.fields[2], e = m.makePluginDeclaration(p.family, "carry", 59, domain[:], b, ASTFuncFlags{})
	if e != nil {
		return e
	}
	if e = m.incNode(p.fields[2]); e != nil {
		return e
	}
	p.fieldAcquired[2] = true
	p.fields[3], e = m.makePluginDeclaration(p.family, "xor3", 60, domain[:], b, ASTFuncFlags{})
	if e != nil {
		return e
	}
	if e = m.incNode(p.fields[3]); e != nil {
		return e
	}
	p.fieldAcquired[3] = true
	name, e := m.Symbols.Intern("arith")
	if e != nil {
		return e
	}
	arith, e := m.families.MkID(name)
	if e != nil {
		return e
	}
	if m.pluginAt(arith).kind == astArithmeticPlugin && m.pluginAt(arith).present {
		p.fields[4] = m.arithmeticPlugin.fields[1]
	}
	// The invalid missing-arith profile remains the existing explicit Go source
	// assertion/domain guard; supported generated registration installs arith first.
	if p.fields[4].slot == 0 {
		return ErrASTDomain
	}
	if e = m.incNode(p.fields[4]); e != nil {
		return e
	}
	p.fieldAcquired[4] = true
	return nil
}
func (m *ASTManager) RegisterBitVectorPlugin() (FamilyID, error) {
	if e := m.lock(); e != nil {
		return NullFamilyID, e
	}
	defer m.mu.Unlock()
	m.formatExecuting = true
	defer func() { m.formatExecuting = false }()
	name, e := m.Symbols.Intern("bv")
	if e != nil {
		return NullFamilyID, e
	}
	id, e := m.families.GetID(name)
	if e != nil {
		return NullFamilyID, e
	}
	if id != NullFamilyID && m.pluginAt(id).present {
		return id, nil
	}
	bytes := uint64(reflect.TypeOf(astBitVectorPluginOwner{}).Size())
	if e = m.BudgetField.Reserve(bytes); e != nil {
		return NullFamilyID, e
	}
	p := &astBitVectorPluginOwner{owner: m, header: bytes}
	p.self = p
	p.next = m.bitVectorPluginFailed
	m.bitVectorPluginFailed = p
	for j, s := range [...]string{"bv", "concat", "sign_extend", "zero_extend", "extract", "rotate_left", "rotate_right", "repeat", "bit2bool", "mkbv"} {
		p.names[j], e = m.Symbols.Intern(s)
		if e != nil {
			p.bodyCause = e
			return NullFamilyID, e
		}
	}
	id, e = m.families.MkID(name)
	if e != nil {
		p.bodyCause = e
		return NullFamilyID, e
	}
	if e = m.setPlugin(id, astBitVectorPlugin); e != nil {
		p.bodyCause = e
		return NullFamilyID, e
	}
	m.bitVectorPluginFailed = p.next
	p.next = nil
	m.bitVectorPlugin = p
	p.family = id
	p.published = true
	if e = p.setManager(); e != nil {
		p.bodyCause = e
		return NullFamilyID, e
	}
	return id, nil
}
func (m *ASTManager) BitVectorSort(bits uint32) (ASTHandle, error) {
	if e := m.lock(); e != nil {
		return ASTHandle{}, e
	}
	defer m.mu.Unlock()
	m.formatExecuting = true
	defer func() { m.formatExecuting = false }()
	if bits == 0 || m.bitVectorPlugin == nil {
		return ASTHandle{}, ErrASTDomain
	}
	r, e := m.bitVectorPlugin.mkSort(bits)
	return m.external(r), e
}

// Concrete cache resize shares the existing pointer-free word codec. Coupled
// cold receipt backing is prepared at the real Go growth boundary before NEW
// publication; it records only each successful INC action, never inferred RC.
func (p *astBitVectorPluginOwner) pushSortSlot() error {
	m := p.owner
	size := p.sorts.size()
	var cap uint32
	if len(p.sorts.words) != 0 {
		cap = p.sorts.words[0]
	}
	if size == cap {
		next := (uint64(cap)*3 + 1) / 2
		if next < 2 {
			next = 2
		}
		if next > math.MaxUint32 || next*2+2 > uint64(^uint(0)>>1)/4 {
			return arena.ErrBounds
		}
		wordsBytes := (next*2 + 2) * 4
		if e := m.BudgetField.Reserve(wordsBytes + next); e != nil {
			return e
		}
		words := make([]uint32, int(next)*2+2)
		receipts := make([]byte, int(next))
		words[0] = uint32(next)
		words[1] = size
		for i := uint32(0); i < size*2; i++ {
			words[2+i] = p.sorts.words[2+i]
		}
		for i := uint32(0); i < size; i++ {
			receipts[i] = p.sortAcquired[i]
		}
		m.BudgetField.Release(p.sorts.bytes() + uint64(len(p.sortAcquired)))
		p.sorts.words = words
		p.sortAcquired = receipts
	}
	p.sorts.words[2+size*2] = 0
	p.sorts.words[3+size*2] = 0
	p.sortAcquired[size] = 0
	p.sorts.words[1]++
	return nil
}
