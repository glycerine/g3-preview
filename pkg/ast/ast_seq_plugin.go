// Source: src/ast/seq_decl_plugin.cpp:343-357,500-610;
// src/ast/seq_decl_plugin.h:126-175; src/util/ref_vector.h:28-217.
// Source authors: Nikolaj Bjorner (2012-02-08); Leonardo de Moura.
// Licensed under the MIT license; see LICENSE.txt.
package ast

import (
	"errors"
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// IDs encode only the source pointer slots. The separately charged concrete
// cold owners hold actual members and stopped leaf receipts, without boxing.
type astSequenceSignature struct {
	next                                                       *astSequenceSignature
	owner                                                      *astSequencePluginOwner
	name                                                       util.Symbol
	parameters, id, cursor                                     uint32
	domain                                                     ASTRefVector
	rangeSort                                                  astNodeRef
	push                                                       ASTRefVectorPushReceipt
	header                                                     uint64
	domainReady, rangeAcquired, completed, ended, contextEnded bool
	bodyCause, sourceCause, ownCause                           error
}
type astSequencePluginOwner struct {
	next                    *astSequencePluginOwner
	self                    *astSequencePluginOwner
	owner                   *ASTManager
	names                   [2]util.Symbol
	family                  FamilyID
	signatures              astPluginVector
	owned                   *astSequenceSignature
	char, stringSort, regex astNodeRef
	acquired                [3]bool // exact successful String, Char, RE INC leaves
	serial                  uint32
	header                  uint64
	published, initialized  bool
	bodyCause, sourceCause  error
}

func (p *astSequencePluginOwner) signature(i uint32) (*astSequenceSignature, error) {
	if p == nil || p.self != p || len(p.signatures.words) == 0 || i >= p.signatures.words[1] {
		return nil, ErrASTHandle
	}
	id := p.signatures.words[2+i]
	if id == 0 {
		return nil, ErrASTHandle
	}
	for s := p.owned; s != nil; s = s.next {
		if s.id == id && s.owner == p {
			return s, nil
		}
	}
	return nil, ErrASTHandle
}
func (p *astSequencePluginOwner) resize(n uint32) error {
	v := &p.signatures
	for uint32(v.sizeSequence()) < n {
		var size, capacity uint32
		if len(v.words) != 0 {
			size, capacity = v.words[1], v.words[0]
		}
		if size == capacity {
			next := (uint64(capacity)*3 + 1) / 2
			if next < 2 {
				next = 2
			}
			if next > math.MaxUint32 || next+2 > uint64(^uint(0)>>1)/4 {
				return arena.ErrBounds
			}
			if e := p.owner.BudgetField.Reserve((next + 2) * 4); e != nil {
				return e
			}
			a := make([]uint32, int(next)+2)
			a[0], a[1] = uint32(next), size
			for i := uint32(0); i < size; i++ {
				a[2+i] = v.words[2+i]
			}
			p.owner.BudgetField.Release(v.bytes())
			v.words = a
		}
		v.words[2+size] = 0
		v.words[1]++
	}
	return nil
}
func (v *astPluginVector) sizeSequence() uint32 {
	if len(v.words) == 0 {
		return 0
	}
	return v.words[1]
}
func (p *astSequencePluginOwner) newSignature(name string, n uint32, dom []astNodeRef, rng astNodeRef) (_s *astSequenceSignature, _body error) {
	m := p.owner
	if p.serial == math.MaxUint32 {
		return nil, ErrASTPending
	}
	bytes := uint64(reflect.TypeOf(astSequenceSignature{}).Size())
	if e := m.BudgetField.Reserve(bytes); e != nil {
		return nil, e
	}
	p.serial++
	s := &astSequenceSignature{owner: p, id: p.serial, header: bytes, next: p.owned}
	p.owned = s
	defer func() {
		if _body != nil {
			s.bodyCause = _body
			_body = errors.Join(_body, p.destroySignature(s, false))
		}
	}()
	var e error
	s.name, e = m.Symbols.Intern(name)
	if e != nil {
		return s, e
	}
	s.parameters = n
	s.domain, e = m.newRefVector()
	if e != nil {
		return s, e
	}
	s.domainReady = true
	s.rangeSort = rng
	if e = m.incNode(rng); e != nil {
		return s, e
	}
	s.rangeAcquired = true
	for _, r := range dom {
		if e = m.pushVectorTracked(s.domain, r, &s.push); e != nil {
			return s, e
		}
		// The actual vector slot now owns this successful INC. No unjoined
		// invocation remains, so the call-local receipt storage may be reused.
		if s.push.published {
			s.push = ASTRefVectorPushReceipt{}
		}
	}
	s.completed = true
	return s, nil
}
func (p *astSequencePluginOwner) destroySignature(s *astSequenceSignature, own bool) error {
	if s.owner != p || s.id == 0 || s.id > p.serial {
		return ErrASTHandle
	}
	if s.contextEnded {
		return nil
	}
	if s.ended {
		return nil
	}
	if !own && s.sourceCause != nil {
		return s.sourceCause
	}
	stop := func(e error) error {
		if e != nil {
			if own {
				if s.ownCause == nil {
					s.ownCause = e
				}
			} else if s.sourceCause == nil {
				s.sourceCause = e
			}
		}
		return e
	}
	m := p.owner
	if s.rangeAcquired {
		consumed, e := m.decNode(s.rangeSort)
		if consumed {
			s.rangeAcquired = false
		}
		if e != nil {
			return stop(e)
		}
	}
	if s.domainReady {
		st, e := m.vectorState(s.domain)
		if e != nil {
			return stop(e)
		}
		for s.cursor < st.size {
			r, e := m.vectorValue(st, s.cursor)
			if e != nil {
				return stop(e)
			}
			consumed, e := m.decNode(r)
			if consumed {
				s.cursor++
			}
			if e != nil {
				return stop(e)
			}
		}
		if st.capacity != 0 {
			if e = m.vectorAllocator.Deallocate(st.capacity*8+astRefVectorHeaderBytes, st.payload); e != nil {
				return stop(e)
			}
		}
		if e = m.allocator.Deallocate(astRefVectorRecordBytes, s.domain.record); e != nil {
			return stop(e)
		}
		s.domainReady = false
	}
	if own && s.push.acquired && !s.push.published && !s.push.acknowledged {
		r := &s.push
		if r.self != r || r.owner != m || r.vector != s.domain || r.invocation == 0 || r.invocation > m.refVectorPushSerial {
			return stop(ErrASTHandle)
		}
		if !r.consumed {
			consumed, e := m.decNode(r.node)
			if consumed {
				r.consumed = true
			}
			if e != nil {
				return stop(e)
			}
		}
		r.acknowledged = true
	}
	// Incomplete constructor unjoined INC is never a source destructor action.
	if !s.push.acquired || s.push.published || s.push.acknowledged {
		s.ended = true
	}
	return nil
}
func (m *ASTManager) sequencePluginBytes() uint64 {
	var n uint64
	add := func(p *astSequencePluginOwner) {
		if p == nil {
			return
		}
		n += p.header + p.signatures.bytes()
		for s := p.owned; s != nil; s = s.next {
			n += s.header
		}
	}
	add(m.sequencePlugin)
	for p := m.sequencePluginFailed; p != nil; p = p.next {
		add(p)
	}
	return n
}
func (m *ASTManager) endSequenceContexts() {
	// Called only after this exact manager's source allocator member has ended.
	// It grants no source replay/DEC against ended nodes or vector records.
	for _, p := range []*astSequencePluginOwner{m.sequencePlugin, m.sequencePluginFailed} {
		for ; p != nil; p = p.next {
			for s := p.owned; s != nil; s = s.next {
				s.contextEnded = true
			}
		}
	}
}
func (m *ASTManager) releaseSequencePluginHeaders() {
	// Source plugin destruction ends only completed signatures. Retain the real
	// Go owner/cause/tuple storage for unjoined constructor acquisitions until
	// actual enclosing owner completion, separately from the source registry.
	if m.sourceClosing {
		return
	}
	for p := m.sequencePlugin; p != nil; {
		next := p.next
		p.signatures.close(m.BudgetField)
		for s := p.owned; s != nil; s = s.next {
			m.BudgetField.Release(s.header)
			s.header = 0
		}
		m.BudgetField.Release(p.header)
		p.header = 0
		p = next
	}
	m.sequencePlugin = nil
	if m.sourceClosing {
		return
	}
	for p := m.sequencePluginFailed; p != nil; {
		next := p.next
		p.signatures.close(m.BudgetField)
		for s := p.owned; s != nil; s = s.next {
			m.BudgetField.Release(s.header)
			s.header = 0
		}
		m.BudgetField.Release(p.header)
		p.header = 0
		p = next
	}
	m.sequencePluginFailed = nil
}
func (m *ASTManager) finalizeSequencePlugin() error {
	p := m.sequencePlugin
	if p == nil {
		return nil
	}
	if m.sourceClosing && p.sourceCause != nil {
		return p.sourceCause
	}
	for i := uint32(0); i < p.signatures.sizeSequence(); i++ {
		if p.signatures.words[2+i] == 0 {
			continue
		}
		s, e := p.signature(i)
		if e == nil {
			e = p.destroySignature(s, !m.sourceClosing)
		}
		if e != nil {
			if p.sourceCause == nil && m.sourceClosing {
				p.sourceCause = e
			}
			return e
		}
	}
	fields := [3]*astNodeRef{&p.stringSort, &p.char, &p.regex}
	for i, r := range fields {
		if r.slot == 0 || !m.sourceClosing && !p.acquired[i] {
			continue
		}
		consumed, e := m.decNode(*r)
		if consumed {
			*r = astNodeRef{}
			p.acquired[i] = false
		}
		if e != nil {
			if m.sourceClosing && p.sourceCause == nil {
				p.sourceCause = e
			}
			return e
		}
	}
	if !m.sourceClosing {
		for s := p.owned; s != nil; s = s.next {
			if e := p.destroySignature(s, true); e != nil {
				return e
			}
		}
	}
	return nil
}
func (m *ASTManager) registerSequencePlugin() (FamilyID, error) {
	name, e := m.Symbols.Intern("seq")
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
	bytes := uint64(reflect.TypeOf(astSequencePluginOwner{}).Size())
	if e = m.BudgetField.Reserve(bytes); e != nil {
		return NullFamilyID, e
	}
	p := &astSequencePluginOwner{owner: m, header: bytes, next: m.sequencePluginFailed}
	p.self = p
	m.sequencePluginFailed = p
	for i, text := range [...]string{"String", "Char"} {
		p.names[i], e = m.Symbols.Intern(text)
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
	if e = m.setPlugin(id, astSequencePlugin); e != nil {
		p.bodyCause = e
		return NullFamilyID, e
	}
	m.sequencePluginFailed = p.next
	p.next = nil
	m.sequencePlugin = p
	p.family = id
	p.published = true
	if m.bitVectorPlugin == nil {
		return id, ErrASTDomain
	}
	p.char, e = m.bitVectorPlugin.mkSort(8)
	if e != nil {
		p.bodyCause = e
		return id, e
	}
	if e = m.incNode(p.char); e != nil {
		p.bodyCause = e
		return id, e
	}
	p.acquired[1] = true
	param := astParameterRecord{Kind: ASTParameterAST, ast: p.char}
	p.stringSort, e = m.sequenceSortCreate(p, "String", 0, &param)
	if e != nil {
		p.bodyCause = e
		return id, e
	}
	if e = m.incNode(p.stringSort); e != nil {
		p.bodyCause = e
		return id, e
	}
	p.acquired[0] = true
	param.ast = p.stringSort
	p.regex, e = m.sequenceSort(p, 1, &param)
	if e != nil {
		p.bodyCause = e
		return id, e
	}
	if e = m.incNode(p.regex); e != nil {
		p.bodyCause = e
		return id, e
	}
	p.acquired[2] = true
	return id, nil
}
func (m *ASTManager) RegisterSequencePlugin() (FamilyID, error) {
	if e := m.lock(); e != nil {
		return NullFamilyID, e
	}
	defer m.mu.Unlock()
	m.formatExecuting = true
	defer func() { m.formatExecuting = false }()
	return m.registerSequencePlugin()
}
func (m *ASTManager) sequenceSortCreate(p *astSequencePluginOwner, text string, k int32, param *astParameterRecord) (astNodeRef, error) {
	name, e := m.Symbols.Intern(text)
	if e != nil {
		return astNodeRef{}, e
	}
	if e = m.beginFormatInfo(true, p.family, k, param); e != nil {
		return astNodeRef{}, e
	}
	defer m.endFormatInfo()
	return m.mkSort(name, ASTSortInfo{m.origin, math.MaxUint32 - 1, m.formatLexical.epoch})
}
func (m *ASTManager) sequenceSort(p *astSequencePluginOwner, k int32, param *astParameterRecord) (astNodeRef, error) {
	if e := m.initializeSequence(p); e != nil {
		return astNodeRef{}, e
	}
	if k == 2 {
		return p.stringSort, nil
	}
	if param == nil || param.Kind != ASTParameterAST || m.validNode(param.ast) != nil || m.nodeAt(param.ast.slot).kind != ASTSort {
		return astNodeRef{}, ErrASTDomain
	}
	switch k {
	case 0:
		if param.ast == p.char {
			return p.stringSort, nil
		}
		return m.sequenceSortCreate(p, "Seq", k, param)
	case 1:
		return m.sequenceSortCreate(p, "RegEx", k, param)
	}
	return astNodeRef{}, ErrASTDomain
}

func (m *ASTManager) initializeSequence(p *astSequencePluginOwner) error {
	if p.initialized {
		return nil
	}
	p.initialized = true // source publication precedes every recursive declaration
	k, e := m.registerUserName(util.NumericalSymbol(0))
	if e != nil {
		return e
	}
	A, e := m.userSort(k, nil)
	if e != nil {
		return e
	}
	strT := p.stringSort
	param := astParameterRecord{Kind: ASTParameterAST, ast: A}
	seqA, e := m.sequenceSort(p, 0, &param)
	if e != nil {
		return e
	}
	param.ast = seqA
	reA, e := m.sequenceSort(p, 1, &param)
	if e != nil {
		return e
	}
	param.ast = strT
	reT, e := m.sequenceSort(p, 1, &param)
	if e != nil {
		return e
	}
	boolT := m.boolSort
	if m.arithmeticPlugin == nil {
		return ErrASTDomain
	}
	intT := m.arithmeticPlugin.fields[1]
	locals := [2]astParameterRecord{{Kind: ASTParameterAST, ast: A}, {Kind: ASTParameterAST, ast: boolT}}
	predA, e := m.arraySortCore(0, astArrayParameters{locals: &locals})
	if e != nil {
		return e
	}
	if e = p.resize(44); e != nil {
		return e
	}
	type row struct {
		index      uint32
		name       string
		parameters uint32
		size       int
		domain     [3]astNodeRef
		rangeSort  astNodeRef
	}
	rows := [...]row{
		{0, "seq.unit", 1, 1, [3]astNodeRef{A}, seqA},
		{1, "seq.empty", 1, 0, [3]astNodeRef{}, seqA},
		{2, "seq.++", 1, 2, [3]astNodeRef{seqA, seqA}, seqA},
		{3, "seq.prefixof", 1, 2, [3]astNodeRef{seqA, seqA}, boolT},
		{4, "seq.suffixof", 1, 2, [3]astNodeRef{seqA, seqA}, boolT},
		{5, "seq.contains", 1, 2, [3]astNodeRef{seqA, seqA}, boolT},
		{6, "seq.extract", 1, 3, [3]astNodeRef{seqA, intT, intT}, seqA},
		{7, "seq.replace", 1, 3, [3]astNodeRef{seqA, seqA, seqA}, seqA},
		{10, "seq.indexof", 1, 3, [3]astNodeRef{seqA, seqA, intT}, intT},
		{8, "seq.at", 1, 2, [3]astNodeRef{seqA, intT}, seqA},
		{9, "seq.len", 1, 1, [3]astNodeRef{seqA}, intT},
		{13, "re.+", 1, 1, [3]astNodeRef{reA}, reA},
		{14, "re.*", 1, 1, [3]astNodeRef{reA}, reA},
		{15, "re.opt", 1, 1, [3]astNodeRef{reA}, reA},
		{16, "re.range", 1, 2, [3]astNodeRef{seqA, seqA}, reA},
		{17, "re.++", 1, 2, [3]astNodeRef{reA, reA}, reA},
		{18, "re.union", 1, 2, [3]astNodeRef{reA, reA}, reA},
		{19, "re.inter", 1, 2, [3]astNodeRef{reA, reA}, reA},
		{20, "re.loop", 1, 1, [3]astNodeRef{reA}, reA},
		{21, "re.complement", 1, 1, [3]astNodeRef{reA}, reA},
		{22, "re.empty", 1, 0, [3]astNodeRef{}, reA},
		{23, "re.all", 1, 0, [3]astNodeRef{}, reA},
		{24, "re.allchar", 1, 0, [3]astNodeRef{}, reA},
		{25, "re.of.pred", 1, 1, [3]astNodeRef{predA}, reA},
		{11, "seq.to.re", 1, 1, [3]astNodeRef{seqA}, reA},
		{12, "seq.in.re", 1, 2, [3]astNodeRef{seqA, reA}, boolT},
		{39, "str.indexof", 0, 3, [3]astNodeRef{strT, strT, intT}, intT},
		{29, "str.replace", 0, 3, [3]astNodeRef{strT, strT, strT}, strT},
		{27, "int.to.str", 0, 1, [3]astNodeRef{intT}, strT},
		{28, "str.to.int", 0, 1, [3]astNodeRef{strT}, intT},
		{30, "str.++", 1, 2, [3]astNodeRef{strT, strT}, strT},
		{31, "str.len", 0, 1, [3]astNodeRef{strT}, intT},
		{32, "str.contains", 0, 2, [3]astNodeRef{strT, strT}, boolT},
		{37, "str.at", 0, 2, [3]astNodeRef{strT, intT, intT}, strT},
		{33, "str.prefixof", 0, 2, [3]astNodeRef{strT, strT}, boolT},
		{34, "str.suffixof", 0, 2, [3]astNodeRef{strT, strT}, boolT},
		{35, "str.in.re", 0, 2, [3]astNodeRef{strT, reT}, boolT},
		{36, "str.to.re", 0, 1, [3]astNodeRef{strT}, reT},
		{40, "re.nostr", 0, 0, [3]astNodeRef{}, reT},
		{41, "re.allchar", 0, 0, [3]astNodeRef{}, reT},
		{38, "str.substr", 0, 3, [3]astNodeRef{strT, intT, intT}, strT},
		{43, "_re.unroll", 0, 2, [3]astNodeRef{reT, intT}, strT},
	}
	for _, r := range rows {
		sig, e := p.newSignature(r.name, r.parameters, r.domain[:r.size], r.rangeSort)
		if e != nil {
			return e
		}
		p.signatures.words[2+r.index] = sig.id // only completed new-expression return
	}
	return nil
}
