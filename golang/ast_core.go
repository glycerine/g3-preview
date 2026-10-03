// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Owner-confined ast_manager default no-trace normal and format constructors.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

func NewASTManager(c ASTManagerConfig) (*ASTManager, error)       { return newASTManager(c, false) }
func NewASTFormatManager(c ASTManagerConfig) (*ASTManager, error) { return newASTManager(c, true) }
func newASTManager(c ASTManagerConfig, format bool) (*ASTManager, error) {
	if c.Symbols == nil || c.Numerals == nil || c.ProofMode > ASTProofEnabled {
		return nil, ErrASTDomain
	}
	// Validate borrowed owners before publishing the concrete cold header.
	if _, e := c.Symbols.Hash(Symbol{}); e != nil {
		return nil, e
	}
	if _, e := c.Numerals.Stats(); e != nil {
		return nil, e
	}
	id, e := astNextOrigin()
	if e != nil {
		return nil, e
	}
	bytes := uint64(reflect.TypeOf(ASTManager{}).Size())
	if e = c.Budget.Reserve(bytes); e != nil {
		return nil, e
	}
	m := &ASTManager{origin: id, budget: c.Budget, symbols: c.Symbols, numerals: c.Numerals, ownerBytes: bytes, proofMode: c.ProofMode, coercions: true, nextNode: 1, nextParameter: 1, nextInfo: 1, nextReference: 1}
	m.self = m
	for i := range m.nextKind {
		m.nextKind[i] = 1
	}
	m.exprIDs = newBudgetedUtilIDGen(0, c.Budget, c.OutOfMemory)
	m.declIDs = newBudgetedUtilIDGen(1<<31, c.Budget, c.OutOfMemory)
	// Completed constructor prefixes are owned immediately and unwound by the
	// same real destructor actions, without touching caller-owned services.
	fail := func(e error) (*ASTManager, error) { m.closeCore(); return nil, e }
	m.allocator, e = arena.NewSmallBytes(arena.SmallBytesConfig{Alignment: uint32(reflect.TypeOf((*ASTManager)(nil)).Size()), DirectOnly: astSourceDebug, Descriptors: arena.Config{Budget: c.Budget}})
	if e != nil {
		return fail(e)
	}
	m.vectorAllocator, e = arena.NewSmallBytes(arena.SmallBytesConfig{DirectOnly: true, Descriptors: arena.Config{Budget: c.Budget}})
	if e != nil {
		return fail(e)
	}
	m.exprArrays = astExprArrayManager{owner: m, allocator: m.allocator, budget: m.budget, id: m.origin, preserve: true}
	m.dependencies = astDependencyManager{owner: m, allocator: m.allocator}
	m.dependencyArrays = astDependencyArrayManager{owner: m, allocator: m.allocator, budget: m.budget, id: m.origin, preserve: true}
	m.families, e = NewFamilyManager(c.Symbols, c.Budget)
	if e != nil {
		return fail(e)
	}
	if e = m.initTable(); e != nil {
		return fail(e)
	}
	m.recFun, e = m.symbols.Intern(":rec-fun")
	if e != nil {
		return fail(e)
	}
	if !format {
		nested := c
		nested.ProofMode = ASTProofDisabled
		m.format, e = newASTManager(nested, true)
		if e != nil {
			return fail(e)
		}
	}
	if e = m.initManager(c.OutOfMemory); e != nil {
		return fail(e)
	}
	return m, nil
}
func (m *ASTManager) initManager(oom func() bool) error {
	for _, name := range [...]string{"basic", "label", "pattern", "model-value", "user-sort", "arith"} {
		s, e := m.symbols.Intern(name)
		if e != nil {
			return e
		}
		if _, e = m.families.MkID(s); e != nil {
			return e
		}
	}
	m.plugins[0] = astPluginBinding{kind: astBasicPlugin, present: true}
	if e := m.initBasic(); e != nil {
		return e
	}
	m.boolSort = m.basic.boolSort
	if e := m.incNode(m.boolSort); e != nil {
		return e
	}
	m.proofSort = m.basic.proofSort
	if e := m.incNode(m.proofSort); e != nil {
		return e
	}
	var e error
	m.undefProof, e = m.mkFamilyApp(0, ASTPrUndef, nil, nil)
	if e != nil {
		return e
	}
	if e = m.incNode(m.undefProof); e != nil {
		return e
	}
	// Label symbols are constructor-owned source scalar spellings; no node pin.
	m.builtin.labelPos, e = m.symbols.Intern("lblpos")
	if e != nil {
		return e
	}
	m.builtin.labelNeg, e = m.symbols.Intern("lblneg")
	if e != nil {
		return e
	}
	m.builtin.labelLit, e = m.symbols.Intern("lbl-lit")
	if e != nil {
		return e
	}
	m.plugins[1] = astPluginBinding{astLabelPlugin, true}
	m.plugins[2] = astPluginBinding{astPatternPlugin, true}
	m.plugins[3] = astPluginBinding{astModelValuePlugin, true}
	if e = m.builtin.userTable.init(m.symbols, m.budget, oom); e != nil {
		return e
	}
	m.plugins[4] = astPluginBinding{astUserSortPlugin, true}
	m.trueApp, e = m.mkFamilyApp(0, ASTOpTrue, nil, nil)
	if e != nil {
		return e
	}
	if e = m.incNode(m.trueApp); e != nil {
		return e
	}
	m.falseApp, e = m.mkFamilyApp(0, ASTOpFalse, nil, nil)
	if e != nil {
		return e
	}
	return m.incNode(m.falseApp)
}
func (m *ASTManager) Close() error {
	if m == nil || m.self != m {
		return ErrASTHandle
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	return m.closeCore()
}
func (m *ASTManager) closeCore() error {
	m.closing = true
	var failure error
	note := func(e error) {
		if failure == nil {
			failure = e
		}
	}
	for _, r := range []*astNodeRef{&m.boolSort, &m.proofSort, &m.trueApp, &m.falseApp, &m.undefProof} {
		if r.slot != 0 {
			consumed, e := m.decNode(*r)
			if consumed {
				*r = astNodeRef{}
			}
			note(e)
		}
	}
	if m.plugins[0].present {
		note(m.finalizeBasic())
	}
	m.destroyPluginBacking()
	// The source manager collects every physical table root after plugins are
	// gone. Dedicated charged directory marks replace the temporary ast_mark;
	// the root list is a charged source-growth vector retained for one pass.
	for m.table.size != 0 {
		for s := uint32(1); s < m.nextNode; s++ {
			if m.nodeAt(s).live {
				m.nodeAt(s).closeMark = false
			}
		}
		for bucket := uint32(0); bucket < m.table.slots; bucket++ {
			if m.table.cells[bucket].next == chainFree {
				continue
			}
			for j := bucket; j != chainEnd; j = m.table.cells[j].next {
				r := m.table.cells[j].node
				for c := uint32(0); c < m.nodeChildCount(r); c++ {
					child, e := m.nodeChild(r, c)
					if e != nil {
						note(e)
						continue
					}
					m.nodeAt(child.slot).closeMark = true
				}
				slot, g := m.infoForNode(r)
				if slot != 0 {
					info, e := m.info(slot, g)
					if e != nil {
						note(e)
						continue
					}
					for c := uint32(0); c < info.count; c++ {
						p, e := m.infoParameter(info, c)
						if e != nil {
							note(e)
							continue
						}
						v, e := m.parameter(p)
						if e != nil {
							note(e)
							continue
						}
						if v.kind == ASTParameterAST && v.ast.slot != 0 {
							m.nodeAt(v.ast.slot).closeMark = true
						}
					}
				}
			}
		}
		// Select physical bucket+chain order, preserving source USED iteration.
		var roots []astNodeRef
		var bytes uint64
		var head, tail astNodeRef
		emergency := false
		for bucket := uint32(0); bucket < m.table.slots; bucket++ {
			if m.table.cells[bucket].next == chainFree {
				continue
			}
			for j := bucket; j != chainEnd; j = m.table.cells[j].next {
				r := m.table.cells[j].node
				if m.nodeAt(r.slot).closeMark {
					continue
				}
				m.nodeAt(r.slot).closeNext = astNodeRef{}
				if tail.slot == 0 {
					head = r
				} else {
					m.nodeAt(tail.slot).closeNext = r
				}
				tail = r
				if emergency {
					continue
				}
				if len(roots) == cap(roots) {
					width := uint64(reflect.TypeOf(astNodeRef{}).Size())
					n, e := astDirectoryGrowth(uint32(cap(roots)), uint32(len(roots)+1), width)
					if e == nil {
						e = m.budget.Reserve(uint64(n) * width)
					}
					if e != nil {
						note(e)
						emergency = true
						continue
					}
					nextBytes := uint64(n) * width
					next := make([]astNodeRef, len(roots), int(n))
					copy(next, roots)
					m.budget.Release(bytes)
					bytes = nextBytes
					roots = next
				}
				roots = append(roots, r)
			}
		}
		if head.slot == 0 {
			m.budget.Release(bytes)
			return ErrASTDomain
		}
		// On source's unsafe allocation-exception continuation, the already charged
		// node directory keeps the exact root sequence; finish rather than orphan it.
		for r := head; r.slot != 0; {
			next := m.nodeAt(r.slot).closeNext
			m.nodeAt(r.slot).rc = 0
			note(m.deleteNode(r))
			r = next
		}
		m.budget.Release(bytes)

	}
	// Caller-owned info/parameter wrapper tokens are invalidated after dropping
	// their actual numeral payloads. Source allocator teardown has no callbacks;
	// this explicit Go extension closes all independently owned wrapper records.
	for s := uint32(1); s < m.nextInfo; s++ {
		r := m.infoAt(s)
		if r.live {
			note(m.dropInfoStorage(s, r.generation))
		}
	}
	for s := uint32(1); s < m.nextParameter; s++ {
		r := m.parameterAt(s)
		if r.live {
			note(m.dropParameter(ASTParameter{m.origin, s, r.generation}))
		}
	}
	if m.format != nil {
		note(m.format.Close())
		m.format = nil
	}
	m.exprIDs.cleanup(0)
	m.declIDs.cleanup(1 << 31)
	m.budget.Release(astChainBytes(uint32(len(m.table.cells))))
	m.table.cells = nil
	m.table.size = 0
	if m.vectorAllocator != nil {
		m.vectorAllocator.Close()
		m.vectorAllocator = nil
	}
	if m.allocator != nil {
		m.allocator.Close()
		m.allocator = nil
	}
	if m.families != nil {
		note(m.families.Close())
		m.families = nil
	}
	m.exprArrays.closeAfterNodeTeardown()
	m.dependencies.closeAfterNodeTeardown()
	m.dependencyArrays.closeAfterNodeTeardown()
	m.releasePages()
	m.closed = true
	m.closing = false
	m.budget.Release(m.ownerBytes)
	m.ownerBytes = 0
	return failure
}
