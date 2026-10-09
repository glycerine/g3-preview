// Source: src/ast/ast.cpp:1297-1479; src/ast/ast.h:1454-1490.
// Owner-confined ast_manager default no-trace normal and format constructors.
package ast

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func NewASTManager(c ASTManagerConfig) (*ASTManager, error)       { return newASTManager(c, false) }
func NewASTFormatManager(c ASTManagerConfig) (*ASTManager, error) { return newASTManager(c, true) }
func newASTManager(c ASTManagerConfig, format bool) (*ASTManager, error) {
	m, e := constructSourceASTManager(c, format, nil, nil)
	if e != nil && m != nil {
		if cleanup := m.AbortConstruction(); cleanup != nil {
			return m, errors.Join(e, cleanup)
		}
		return nil, e
	}
	return m, e
}

// NewSourceASTManager retains the real failed placement-new context. A failed
// constructor invokes only reverse completed member DT, never ast_manager's
// destructor body. Body-acquired file/format pointers remain named for Abort.
func NewSourceASTManager(c ASTManagerConfig, filename *util.ParamsStringView) (*ASTManager, error) {
	var borrowed TraceFilename
	if filename != nil {
		borrowed = *filename
	}
	return constructSourceASTManager(c, false, borrowed, nil)
}

func constructSourceASTManager(c ASTManagerConfig, format bool, filename TraceFilename, borrowedTrace *util.NumericTextStream) (*ASTManager, error) {
	if c.Symbols == nil || c.Numerals == nil || c.ProofMode > ASTProofEnabled {
		return nil, ErrASTDomain
	}
	// Validate borrowed owners before publishing the concrete cold header.
	if _, e := c.Symbols.Hash(util.Symbol{}); e != nil {
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
	m := &ASTManager{origin: id, BudgetField: c.Budget, Symbols: c.Symbols, numerals: c.Numerals, ownerBytes: bytes, proofMode: c.ProofMode, coercions: true, nextNode: 1, nextParameter: 1, nextInfo: 1, nextReference: 1}
	m.Self = m
	// reslimit is the first automatic source member. Its zero-valued scalar
	// constructor owns no vector payload and has no separate owner allocation.
	m.Limit.InitEmbedded(c.Budget)
	for i := range m.nextKind {
		m.nextKind[i] = 1
	}
	m.ExprIDs = util.NewBudgetedIDGen(0, c.Budget, c.OutOfMemory)
	m.declIDs = util.NewBudgetedIDGen(1<<31, c.Budget, c.OutOfMemory)
	// Completed constructor prefixes are owned immediately and unwound by the
	// same real destructor actions, without touching caller-owned services.
	fail := func(e error) (*ASTManager, error) {
		if m.constructorBodyCause == nil {
			m.constructorBodyCause = e
		}
		unwind := m.unwindConstructorMembers()
		return m, errors.Join(e, unwind)
	}
	m.allocator, e = arena.NewSmallBytes(arena.SmallBytesConfig{Alignment: uint32(reflect.TypeOf((*ASTManager)(nil)).Size()), DirectOnly: ASTSourceDebug, Descriptors: arena.Config{Budget: c.Budget}})
	if e != nil {
		return fail(e)
	}
	m.constructorMembers |= astCtorAllocator
	m.families, e = NewFamilyManager(c.Symbols, c.Budget)
	if e != nil {
		return fail(e)
	}
	m.constructorMembers |= astCtorFamilies
	m.vectorAllocator, e = arena.NewSmallBytes(arena.SmallBytesConfig{DirectOnly: true, Descriptors: arena.Config{Budget: c.Budget}})
	if e != nil {
		return fail(e)
	}
	m.constructorMembers |= astCtorVectorAllocator
	m.exprArrays = astExprArrayManager{owner: m, allocator: m.allocator, budget: m.BudgetField, id: m.origin, preserve: true}
	m.dependencies = astDependencyManager{owner: m, allocator: m.allocator}
	m.dependencyArrays = astDependencyArrayManager{owner: m, allocator: m.allocator, budget: m.BudgetField, id: m.origin, preserve: true}
	m.constructorMembers |= astCtorArrays
	if e = m.initTable(); e != nil {
		return fail(e)
	}
	m.constructorMembers |= astCtorTable | astCtorIDGenerators
	m.DebugFreeIndices = make(map[uint32]struct{})
	m.debugFreeComplete = true
	m.constructorMembers |= astCtorDebugFree
	m.recFun, e = m.Symbols.Intern(":rec-fun")
	if e != nil {
		return fail(e)
	}
	m.TraceStream = borrowedTrace
	if filename != nil {
		file, err := newASTTraceFileBorrowed(c.Budget, filename)
		if err != nil {
			m.constructorPendingFile = file
			return fail(err)
		}
		m.TraceFile = file
		m.TraceStream = &file.stream
		m.traceOwner = true
	}
	if !format {
		nested := c
		nested.ProofMode = ASTProofDisabled
		child, err := constructSourceASTManager(nested, true, nil, m.TraceStream)
		if err != nil {
			m.constructorPendingFormat = child
			return fail(err)
		}
		m.Format = child
	}
	if e = m.initManager(c.OutOfMemory); e != nil {
		return fail(e)
	}
	m.constructorComplete = true
	return m, nil
}
func (m *ASTManager) initManager(oom func() bool) error {
	for _, name := range [...]string{"basic", "label", "pattern", "model-value", "user-sort", "arith"} {
		s, e := m.Symbols.Intern(name)
		if e != nil {
			return e
		}
		if _, e = m.families.MkID(s); e != nil {
			return e
		}
	}
	if e := m.setPlugin(0, astBasicPlugin); e != nil {
		return e
	}
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
	m.builtin.labelPos, e = m.Symbols.Intern("lblpos")
	if e != nil {
		return e
	}
	m.builtin.labelNeg, e = m.Symbols.Intern("lblneg")
	if e != nil {
		return e
	}
	m.builtin.labelLit, e = m.Symbols.Intern("lbl-lit")
	if e != nil {
		return e
	}
	if e := m.setPlugin(1, astLabelPlugin); e != nil {
		return e
	}
	if e := m.setPlugin(2, astPatternPlugin); e != nil {
		return e
	}
	if e := m.setPlugin(3, astModelValuePlugin); e != nil {
		return e
	}
	if e = m.builtin.userTable.Init(m.Symbols, m.BudgetField, oom); e != nil {
		return e
	}
	if e := m.setPlugin(4, astUserSortPlugin); e != nil {
		return e
	}
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
	if m == nil || m.Self != m {
		return ErrASTHandle
	}
	if m.traceExecuting || m.formatExecuting {
		return ErrASTPending
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Closed {
		return nil
	}
	if e := m.cleanupExprVisitFrame(); e != nil {
		return e
	}
	if !m.constructorComplete {
		return m.abortConstructionCore()
	}
	if m.sourceDTCompleted {
		return m.closeSourceOwnedStorageCore()
	}
	e := m.closeCore(false)
	if e != nil && m.errorOwnClose == nil {
		m.errorOwnClose = e
	}
	return e
}
func (m *ASTManager) closeCore(source bool) error {
	if e := m.cleanupAppCoercion(!source); e != nil {
		return e
	}
	if !source {
		if e := m.cleanupArrayCardinality(); e != nil {
			return e
		}
	}
	if e := m.llFrameDestroy(!source); e != nil {
		return e
	}
	if !source {
		if e := m.cleanupTraceAcquisitions(); e != nil {
			return e
		}
	}
	m.closing = true
	m.sourceClosing = source
	defer func() { m.sourceClosing = false }()
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
			if source && failure != nil {
				return failure
			}
		}
	}
	// Source finalizes concrete registered plugins in ascending registry order.
	if len(m.plugins.words) != 0 {
		for id := uint32(0); id < m.plugins.words[1]; id++ {
			p := m.pluginAt(FamilyID(id))
			if !p.present {
				continue
			}
			var e error
			switch p.kind {
			case astBasicPlugin:
				e = m.finalizeBasic()
			case astFormatPlugin:
				e = m.finalizeFormatPlugin()
			case astArithmeticPlugin:
				e = m.finalizeArithmeticPlugin()
			case astBitVectorPlugin:
				e = m.finalizeBitVectorPlugin()
			case astFloatingPointPlugin:
				e = m.finalizeFloatingPointPlugin()
				if e == nil {
					e = m.destroyFloatingPointMembers(!source)
				}
			case astDatatypePlugin:
				e = m.finalizeDatatypePlugin()
			case astSequencePlugin:
				e = m.finalizeSequencePlugin()
			}
			note(e)
			if source && failure != nil {
				return failure
			}
		}
	}
	if !source {
		if e := m.cleanupBitVectorPower(); e != nil {
			return e
		}
	}
	if e := m.destroyFloatingPointMembers(!source); e != nil {
		return e
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
			if m.table.cells[bucket].next == util.ChainFree {
				continue
			}
			for j := bucket; j != util.ChainEnd; j = m.table.cells[j].next {
				r := m.table.cells[j].node
				if e := m.validNode(r); e != nil {
					return e
				}
				for c := uint32(0); c < m.nodeChildCount(r); c++ {
					child, e := m.nodeChild(r, c)
					if e != nil {
						note(e)
						if source && failure != nil {
							return failure
						}
						continue
					}
					m.nodeAt(child.slot).closeMark = true
				}
				slot, g := m.infoForNode(r)
				if slot != 0 {
					info, e := m.info(slot, g)
					if e != nil {
						note(e)
						if source && failure != nil {
							return failure
						}
						continue
					}
					for c := uint32(0); c < info.count; c++ {
						p, e := m.infoParameter(info, c)
						if e != nil {
							note(e)
							if source && failure != nil {
								return failure
							}
							continue
						}
						v, e := m.parameter(p)
						if e != nil {
							note(e)
							if source && failure != nil {
								return failure
							}
							continue
						}
						if v.Kind == ASTParameterAST && v.ast.slot != 0 {
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
			if m.table.cells[bucket].next == util.ChainFree {
				continue
			}
			for j := bucket; j != util.ChainEnd; j = m.table.cells[j].next {
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
						e = m.BudgetField.Reserve(uint64(n) * width)
					}
					if e != nil {
						note(e)
						if source && failure != nil {
							return failure
						}
						emergency = true
						continue
					}
					nextBytes := uint64(n) * width
					next := make([]astNodeRef, len(roots), int(n))
					copy(next, roots)
					m.BudgetField.Release(bytes)
					bytes = nextBytes
					roots = next
				}
				roots = append(roots, r)
			}
		}
		if head.slot == 0 {
			m.BudgetField.Release(bytes)
			return ErrASTDomain
		}
		// On source's unsafe allocation-exception continuation, the already charged
		// node directory keeps the exact root sequence; finish rather than orphan it.
		for r := head; r.slot != 0; {
			next := m.nodeAt(r.slot).closeNext
			m.nodeAt(r.slot).rc = 0
			note(m.deleteNode(r))
			if source && failure != nil {
				return failure
			}
			r = next
		}
		m.BudgetField.Release(bytes)

	}
	// Caller-owned info/parameter wrapper tokens are invalidated after dropping
	// their actual numeral payloads. Source allocator teardown has no callbacks;
	// this explicit Go extension closes all independently owned wrapper records.
	if !source {
		for s := uint32(1); s < m.nextInfo; s++ {
			r := m.infoAt(s)
			if r.live {
				note(m.dropInfoStorage(s, r.generation))
				if source && failure != nil {
					return failure
				}
			}
		}
		for s := uint32(1); s < m.nextParameter; s++ {
			r := m.parameterAt(s)
			if r.Live {
				note(m.dropParameter(ASTParameter{m.origin, s, r.generation}))
				if source && failure != nil {
					return failure
				}
			}
		}
	}
	if m.Format != nil {
		if source {
			if e := m.Format.SourceFinalize(); e != nil {
				note(e)
				return failure
			}
		}
		note(m.Format.Close())
		if source && failure != nil {
			return failure
		}
		m.Format = nil
	}
	if m.traceOwner && m.TraceFile != nil {
		if !m.traceEOFReached {
			m.traceEOFReached = true
			note(m.TraceStream.AppendCString("[eof]\n"))
			if source && failure != nil {
				return failure
			}
		}
		note(m.TraceFile.Close())
		if source && failure != nil {
			return failure
		}
		if m.TraceFile.Closed {
			m.TraceFile = nil
			m.TraceStream = nil
			m.traceOwner = false
		}
	}
	if m.debugFreeComplete {
		m.DebugFreeIndices = nil
		m.debugFreeComplete = false
	}
	m.declIDs.Cleanup(1 << 31)
	m.ExprIDs.Cleanup(0)
	m.BudgetField.Release(astChainBytes(uint32(len(m.table.cells))))
	m.table.cells = nil
	m.table.size = 0
	if !source && m.vectorAllocator != nil {
		m.vectorAllocator.Close()
		m.vectorAllocator = nil
	}
	m.dependencyArrays.closeAfterNodeTeardown()
	m.dependencies.closeAfterNodeTeardown()
	m.exprArrays.closeAfterNodeTeardown()
	if m.families != nil {
		note(m.families.Close())
		if source && failure != nil {
			return failure
		}
		m.families = nil
	}
	if m.allocator != nil {
		m.allocator.Close()
		m.endSequenceContexts()
		m.ppContextEnded()
		m.allocator = nil
	}
	// The first source member is destroyed last, after m_alloc.
	m.Limit.Close()
	if source {
		m.sourceDTCompleted = true
		m.closing = false
		return nil
	}
	m.releaseTraceAcquisitionHistory()
	m.releaseExprVisitHistory()
	if !source {
		if e := m.releasePPFrames(); e != nil {
			return e
		}
	}
	m.releasePages()
	m.Closed = true
	m.closing = false
	m.releaseFloatingPointHistory()
	m.releaseAuxiliaryPluginHistory()
	m.releaseDatatypePluginHistory()
	m.releaseArrayCardinalityHistory()
	m.BudgetField.Release(m.ownerBytes)
	m.ownerBytes = 0
	return failure
}
