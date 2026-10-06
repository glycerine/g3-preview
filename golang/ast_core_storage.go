// Source: src/ast/ast.h; src/ast/ast.cpp:1354,2213-2238.
// Source author: Leonardo de Moura (2006-09-18; 2006-09-28).
// Licensed under the MIT license; see LICENSE.txt.
// Concrete pointer-free storage for src/ast/ast.h. Domain destruction is
// performed by ast_manager action translations, never by page reclamation.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/mpnum"
	"github.com/glycerine/g3/golang/utilhash"
	"math"
	"reflect"
	"sync"
	"sync/atomic"
)

var ErrASTHandle = errors.New("g3: invalid AST owner, kind or lifetime")
var ErrASTDomain = errors.New("g3: AST source precondition")
var ErrASTPending = errors.New("g3: AST release continuation is pending")
var astIdentity atomic.Uint64

type ASTHandle struct {
	origin           uint64
	slot, generation uint32
}

func (h ASTHandle) Origin() uint64     { return h.origin }
func (h ASTHandle) Slot() uint32       { return h.slot }
func (h ASTHandle) Generation() uint32 { return h.generation }
func (h ASTHandle) IsNull() bool       { return h == ASTHandle{} }

type astNodeRef struct{ slot, generation uint32 }
type ASTKind uint16

const (
	ASTApp ASTKind = iota
	ASTVar
	ASTQuantifier
	ASTSort
	ASTFuncDecl
)

type ASTProofMode uint8

const (
	ASTProofDisabled ASTProofMode = iota
	ASTProofEnabled
)

type ASTManagerConfig struct {
	Symbols     *SymbolManager
	Numerals    *mpnum.RationalService
	Budget      *arena.Budget
	ProofMode   ASTProofMode
	OutOfMemory func() bool
}
type ASTManagerStats struct {
	LiveNodes, LiveParameters, LiveInfos, LiveReferences, ArrayLiveCells uint32
	AllocatedBytes, OwnerBytes, PageBytes, DirectoryBytes                uint64
	TableSlots, TableUsedSlots, TableCollisions                          uint32
}
type ASTParameter struct {
	origin           uint64
	slot, generation uint32
}
type ASTSortInfo struct {
	origin           uint64
	slot, generation uint32
}
type ASTFuncDeclInfo struct {
	origin           uint64
	slot, generation uint32
}
type ASTRef struct {
	origin           uint64
	slot, generation uint32
}
type ASTExprArray struct {
	origin  uint64
	cell    arena.SmallBytesRef
	updates uint32
}
type ASTParameterKind uint32

const (
	ASTParameterInt ASTParameterKind = iota
	ASTParameterAST
	ASTParameterSymbol
	ASTParameterRational
	ASTParameterDouble
	ASTParameterExternal
)

type ASTFuncFlags struct{ LeftAssociative, RightAssociative, FlatAssociative, Commutative, Chainable, Pairwise, Injective, Idempotent, Skolem bool }
type ASTQuantifierSpec struct {
	Forall               bool
	Sorts                []ASTHandle
	Names                []Symbol
	Body                 ASTHandle
	Weight               int32
	QID, SkolemID        Symbol
	Patterns, NoPatterns []ASTHandle
}

// All continuation fields are actual charged directory metadata. A zero-RC
// node awaiting destruction remains owned by this manager's deletion graph;
// it cannot be recycled while current/queued/recursive-frame state owns it.
type astNodeRecord struct {
	generation, nextFree, id, hash, rc, kindIndex uint32
	kind                                          ASTKind
	live, mark1, mark2, shared, closeMark         bool
	markOwner1, markOwner2                        uint64
	deleteNext, deleteParent, closeNext           astNodeRef
	deletePhase, deleteIndex                      uint32
	deleting                                      bool
}
type astSortRecord struct {
	nextFree uint32
	name     Symbol
	info     ASTSortInfo
}
type astFuncDeclRecord struct {
	nextFree  uint32
	name      Symbol
	info      ASTFuncDeclInfo
	domain    arena.SmallBytesRef
	arity     uint32
	rangeSort astNodeRef
}
type astAppRecord struct {
	nextFree                    uint32
	decl                        astNodeRef
	args                        arena.SmallBytesRef
	count, depth                uint32
	ground, quantifiers, labels bool
}
type astVarRecord struct {
	nextFree, index uint32
	sort            astNodeRef
}
type astQuantifierRecord struct {
	nextFree                                   uint32
	body                                       astNodeRef
	sorts, names, patterns, noPatterns         arena.SmallBytesRef
	count, patternCount, noPatternCount, depth uint32
	weight                                     int32
	qid, skolemID                              Symbol
	forall, unusedVars, labels                 bool
}
type astParameterRecord struct {
	generation, nextFree uint32
	kind                 ASTParameterKind
	live                 bool
	integer              int32
	ast                  astNodeRef
	symbol               Symbol
	rational             mpnum.OwnedRational
	double               float64
	external             uint32
}
type astInfoRecord struct {
	generation, nextFree            uint32
	live, isSort, private, attached bool
	retainedByDebug                 bool
	family                          FamilyID
	kind                            int32
	parameters                      arena.SmallBytesRef
	count, capacity                 uint32
	size                            SortSize
	flags                           ASTFuncFlags
}
type astRefRecord struct {
	generation, nextFree uint32
	live                 bool
	node                 astNodeRef
}

const astPageSize = 64

type astNodePage [astPageSize]astNodeRecord
type astSortPage [astPageSize]astSortRecord
type astFuncDeclPage [astPageSize]astFuncDeclRecord
type astAppPage [astPageSize]astAppRecord
type astVarPage [astPageSize]astVarRecord
type astQuantifierPage [astPageSize]astQuantifierRecord
type astParameterPage [astPageSize]astParameterRecord
type astInfoPage [astPageSize]astInfoRecord
type astRefPage [astPageSize]astRefRecord

type astPluginKind uint8

const (
	astBasicPlugin astPluginKind = iota
	astLabelPlugin
	astPatternPlugin
	astModelValuePlugin
	astUserSortPlugin
)

// The five real implementations use a cold tagged registry. No per-node
// interface or callback is stored. Plugin backing/cache ownership is explicit.
type astPluginBinding struct {
	kind    astPluginKind
	present bool
}
type astBasicPluginState struct {
	boolSort, proofSort astNodeRef
	declarations        [ASTLastBasicPR]astNodeRef
	eq, oeq, ite        []astNodeRef
	proofCaches         [ASTLastBasicPR][]astNodeRef
}
type astBuiltinPluginState struct {
	labelPos, labelNeg, labelLit Symbol
	userNames                    []Symbol
	userTable                    astUserDictionary
}

type ASTManager struct {
	traceAcquisitionHistory                                                                            *astTraceAcquisition
	traceAcquisitions, traceAcquisitionExecuting                                                       *astTraceAcquisition
	traceAcquisitionSerial                                                                             uint64
	mu                                                                                                 sync.Mutex
	self                                                                                               *ASTManager
	origin                                                                                             uint64
	budget                                                                                             *arena.Budget
	symbols                                                                                            *SymbolManager
	recFun                                                                                             Symbol
	numerals                                                                                           *mpnum.RationalService
	families                                                                                           *FamilyManager
	allocator                                                                                          *arena.SmallBytes
	vectorAllocator                                                                                    *arena.SmallBytes
	format                                                                                             *ASTManager
	proofMode                                                                                          ASTProofMode
	coercions, closed, closing, debugNondeterminism                                                    bool
	sourceDTStarted, sourceDTCompleted, sourceClosing, traceEOFReached                                 bool
	sourceDTCause, errorOwnClose                                                                       error
	constructorComplete                                                                                bool
	constructorBodyCause, constructorUnwindCause, constructorOwnCause                                  error
	constructorMemberStage                                                                             uint8
	constructorMembers                                                                                 uint32
	constructorPendingFormat                                                                           *ASTManager
	constructorPendingFile                                                                             *astTraceFile
	traceFile                                                                                          *astTraceFile
	traceStream                                                                                        *NumericTextStream
	traceOwner                                                                                         bool
	lowLevel                                                                                           astLowLevelFrame
	traceExecuting                                                                                     bool
	debugRefCount                                                                                      bool
	debugFreeIndices                                                                                   utilhash.UintMap
	debugFreeComplete                                                                                  bool
	exprIDs, declIDs                                                                                   utilIDGen
	nodes                                                                                              []*astNodePage
	sorts                                                                                              []*astSortPage
	decls                                                                                              []*astFuncDeclPage
	apps                                                                                               []*astAppPage
	vars                                                                                               []*astVarPage
	quantifiers                                                                                        []*astQuantifierPage
	parameters                                                                                         []*astParameterPage
	infos                                                                                              []*astInfoPage
	references                                                                                         []*astRefPage
	nextKind, freeKind                                                                                 [5]uint32
	nextNode, freeNode, nextParameter, freeParameter, nextInfo, freeInfo, nextReference, freeReference uint32
	liveNodes, liveParameters, liveInfos, liveReferences                                               uint32
	pageBytes, directoryBytes, ownerBytes                                                              uint64
	table                                                                                              astChainTable
	plugins                                                                                            [5]astPluginBinding
	basic                                                                                              astBasicPluginState
	builtin                                                                                            astBuiltinPluginState
	boolSort, proofSort, trueApp, falseApp, undefProof                                                 astNodeRef
	deletionHead, deletionCurrent                                                                      astNodeRef
	pendingDeletionHead, pendingDeletionCurrent                                                        astNodeRef
	exprArrays                                                                                         astExprArrayManager
	// Actual AST-valued dependency manager and persistent-array specialization;
	// borrowed owners are bound eagerly and scratch backing is charged on use.
	// Source fresh_id and lexical func_decl_info storage; included in owner header.
	refVectorPushSerial uint64
	freshID             uint32
	freshInfo           astInfoRecord
	freshInfoEpoch      uint32
	freshInfoActive     bool
	dependencies        astDependencyManager
	dependencyArrays    astDependencyArrayManager
}

func astNextOrigin() (uint64, error) {
	for {
		old := astIdentity.Load()
		if old == math.MaxUint64 {
			return 0, arena.ErrBounds
		}
		if astIdentity.CompareAndSwap(old, old+1) {
			return old + 1, nil
		}
	}
}
func astDirectoryGrowth(old, needed uint32, width uint64) (uint32, error) {
	n := uint64(old)
	if n == 0 {
		n = 2
	}
	for n < uint64(needed) {
		n = (3*n + 1) / 2
		if n > math.MaxUint32 {
			return 0, arena.ErrBounds
		}
	}
	if n > uint64(^uint(0)>>1)/width {
		return 0, arena.ErrBounds
	}
	return uint32(n), nil
}
func (m *ASTManager) nodeAt(slot uint32) *astNodeRecord {
	return &m.nodes[(slot-1)/astPageSize][(slot-1)%astPageSize]
}
func (m *ASTManager) sortAt(slot uint32) *astSortRecord {
	index := m.nodeAt(slot).kindIndex
	return &m.sorts[(index-1)/astPageSize][(index-1)%astPageSize]
}
func (m *ASTManager) declAt(slot uint32) *astFuncDeclRecord {
	index := m.nodeAt(slot).kindIndex
	return &m.decls[(index-1)/astPageSize][(index-1)%astPageSize]
}
func (m *ASTManager) appAt(slot uint32) *astAppRecord {
	index := m.nodeAt(slot).kindIndex
	return &m.apps[(index-1)/astPageSize][(index-1)%astPageSize]
}
func (m *ASTManager) varAt(slot uint32) *astVarRecord {
	index := m.nodeAt(slot).kindIndex
	return &m.vars[(index-1)/astPageSize][(index-1)%astPageSize]
}
func (m *ASTManager) quantifierAt(slot uint32) *astQuantifierRecord {
	index := m.nodeAt(slot).kindIndex
	return &m.quantifiers[(index-1)/astPageSize][(index-1)%astPageSize]
}
func (m *ASTManager) parameterAt(slot uint32) *astParameterRecord {
	return &m.parameters[(slot-1)/astPageSize][(slot-1)%astPageSize]
}
func (m *ASTManager) infoAt(slot uint32) *astInfoRecord {
	return &m.infos[(slot-1)/astPageSize][(slot-1)%astPageSize]
}
func (m *ASTManager) referenceAt(slot uint32) *astRefRecord {
	return &m.references[(slot-1)/astPageSize][(slot-1)%astPageSize]
}
func (m *ASTManager) validNode(r astNodeRef) error {
	if r.slot == 0 || r.slot >= m.nextNode {
		return ErrASTHandle
	}
	n := m.nodeAt(r.slot)
	if !n.live || n.generation != r.generation {
		return ErrASTHandle
	}
	return nil
}
func (m *ASTManager) internal(h ASTHandle) (astNodeRef, error) {
	r := astNodeRef{h.slot, h.generation}
	if h.origin != m.origin {
		return astNodeRef{}, ErrASTHandle
	}
	return r, m.validNode(r)
}
func (m *ASTManager) external(r astNodeRef) ASTHandle {
	if r.slot == 0 {
		return ASTHandle{}
	}
	return ASTHandle{m.origin, r.slot, r.generation}
}
func (m *ASTManager) lock() error {
	if m == nil || m.self != m {
		return ErrASTHandle
	}
	if m.traceExecuting {
		return ErrASTPending
	}
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return arena.ErrClosed
	}
	if m.sourceDTStarted {
		m.mu.Unlock()
		return ErrASTPending
	}
	if !m.constructorComplete {
		m.mu.Unlock()
		return ErrASTDomain
	}
	return nil
}
func (m *ASTManager) reservePage(bytes uint64) error {
	if e := m.budget.Reserve(bytes); e != nil {
		return e
	}
	m.pageBytes += bytes
	return nil
}
func (m *ASTManager) prepareDirectory(old, needed int) (int, error) {
	if old >= needed {
		return old, nil
	}
	width := uint64(reflect.TypeOf((*astNodePage)(nil)).Size())
	n, e := astDirectoryGrowth(uint32(old), uint32(needed), width)
	if e != nil {
		return 0, e
	}
	if e = m.budget.Reserve(uint64(n) * width); e != nil {
		return 0, e
	}
	return int(n), nil
}
func (m *ASTManager) publishDirectory(old, next int) {
	width := uint64(reflect.TypeOf((*astNodePage)(nil)).Size())
	m.budget.Release(uint64(old) * width)
	m.directoryBytes += uint64(next-old) * width
}
