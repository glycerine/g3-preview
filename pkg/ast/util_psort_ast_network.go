// Source: src/util/sorting_network.h:120-1046; src/test/sorting_network.cpp:150-181.
package ast

import (
	"errors"
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

var ErrPsortUninitializedComparison = errors.New("g3: reached uninitialized psort comparison mode")

const (
	psortLE uint8 = iota
	psortGE
	psortEQ
	psortGEFull
	psortLEFull
)

type PsortASTStats struct{ CompiledVariables, CompiledClauses, ClauseVariables uint32 }
type PsortASTNetwork struct {
	self            *PsortASTNetwork
	context         *PsortASTContext
	runtime         *util.Runtime
	budget          *arena.Budget
	debug           *util.DebugContext
	header          uint64
	mode            uint8
	modeInitialized bool
	stats           PsortASTStats
	frames          *psortASTVectorFrame
	serial          uint64
	running, closed bool
	destructorCause error
	sourceDone      bool
}
type psortASTVectorFrame struct {
	next                                 *psortASTVectorFrame
	owner                                *PsortASTNetwork
	invocation, header                   uint64
	vectors                              []RuntimeASTLiteralVector
	cursor                               uint32
	sourceDone                           bool
	bodyCause, destructorCause, ownCause error
}

// psortASTPointer denotes a call-local source pointer. Small literal arrays
// remain actual inline values; dynamic pointers keep their actual Runtime token.
type psortASTPointer struct {
	span   ASTLiteralVectorSpan
	inline *[3]ASTHandle
	offset uint64
}

func (p psortASTPointer) at(i uint32) (ASTHandle, error) {
	index := p.offset + uint64(i)
	if index < p.offset {
		return ASTHandle{}, arena.ErrBounds
	}
	if p.inline != nil {
		if index >= 3 {
			return ASTHandle{}, arena.ErrBounds
		}
		return p.inline[index], nil
	}
	if index > math.MaxUint32 {
		return ASTHandle{}, arena.ErrBounds
	}
	return p.span.At(uint32(index))
}
func (p psortASTPointer) add(n uint32) psortASTPointer { p.offset += uint64(n); return p }
func psortVectorPointer(v *RuntimeASTLiteralVector) (psortASTPointer, error) {
	p, e := v.CPtr()
	return psortASTPointer{span: p}, e
}
func NewPsortASTNetwork(c *PsortASTContext, r *util.Runtime, b *arena.Budget, d *util.DebugContext) (*PsortASTNetwork, error) {
	if c == nil || c.self != c || r == nil || b == nil || c.budget != b || r.Budget() != b || d != nil && (!d.ValidOwner() || d.Budget() != b) {
		return nil, arena.ErrConfig
	}
	if e := c.open(); e != nil {
		return nil, e
	}
	if e := r.CheckOpen(); e != nil {
		return nil, e
	}
	bytes := uint64(reflect.TypeOf(PsortASTNetwork{}).Size())
	if e := b.Reserve(bytes); e != nil {
		return nil, e
	}
	s := &PsortASTNetwork{context: c, runtime: r, budget: b, debug: d, header: bytes}
	s.self = s
	return s, nil
}
func (s *PsortASTNetwork) open() error {
	if s == nil || s.self != s {
		return ErrASTHandle
	}
	if s.closed || s.sourceDone {
		return arena.ErrClosed
	}
	if s.running {
		return ErrASTPending
	}
	if s.destructorCause != nil {
		return s.destructorCause
	}
	return s.context.open()
}
func (s *PsortASTNetwork) assertions() (bool, error) {
	if !util.RuntimeVectorSourceDebug {
		return false, nil
	}
	if s.debug == nil {
		return true, nil
	}
	return s.debug.AssertionsEnabled()
}
func (s *PsortASTNetwork) assertion(line uint32, text string) error {
	return util.DebugAssertionFailure(s.debug, "src/util/sorting_network.h", int32(line), text)
}
func (s *PsortASTNetwork) comparison() (uint8, error) {
	if !s.modeInitialized {
		return 0, ErrPsortUninitializedComparison
	}
	return s.mode, nil
}
func (s *PsortASTNetwork) beginFrame(n uint32) (*psortASTVectorFrame, error) {
	if s.serial == math.MaxUint64 {
		return nil, arena.ErrSlotLimit
	}
	bytes := uint64(reflect.TypeOf(psortASTVectorFrame{}).Size()) + uint64(n)*uint64(reflect.TypeOf(RuntimeASTLiteralVector{}).Size())
	if e := s.budget.Reserve(bytes); e != nil {
		return nil, e
	}
	s.serial++
	f := &psortASTVectorFrame{owner: s, invocation: s.serial, header: bytes, vectors: make([]RuntimeASTLiteralVector, n), next: s.frames}
	s.frames = f
	for i := range f.vectors {
		v := &f.vectors[i]
		v.self = v
		v.budget = s.budget
		v.debug = s.debug
		v.raw.Bind(s.runtime)
	}
	return f, nil
}
func (s *PsortASTNetwork) sourceFrame(f *psortASTVectorFrame) error {
	if f.owner != s || f.invocation == 0 || f.invocation > s.serial {
		return ErrASTHandle
	}
	if f.destructorCause != nil {
		return f.destructorCause
	}
	for f.cursor != 0 {
		v := &f.vectors[f.cursor-1]
		if e := v.SourceFinalize(); e != nil {
			f.destructorCause = e
			return e
		}
		f.cursor--
	}
	f.sourceDone = true
	return nil
}
func (s *PsortASTNetwork) ownFrame(f *psortASTVectorFrame) error {
	if f.owner != s || f.invocation == 0 || f.invocation > s.serial {
		return ErrASTHandle
	}
	for i := len(f.vectors) - 1; i >= 0; i-- {
		v := &f.vectors[i]
		var e error
		if v.destructorCause != nil {
			e = v.CleanupStopped()
		} else {
			e = v.Close()
		}
		if e != nil {
			if f.ownCause == nil {
				f.ownCause = e
			}
			return e
		}
	}
	link := &s.frames
	for *link != nil && *link != f {
		link = &(*link).next
	}
	if *link != f {
		return ErrASTHandle
	}
	*link = f.next
	s.budget.Release(f.header)
	f.header = 0
	return nil
}
func (s *PsortASTNetwork) finishFrame(f *psortASTVectorFrame, body error) error {
	if f.bodyCause == nil {
		f.bodyCause = body
	}
	// A younger lexical destructor stop bars all older source actions.
	if s.destructorCause != nil {
		return errors.Join(body, s.destructorCause)
	}
	e := s.sourceFrame(f)
	if e != nil {
		if s.destructorCause == nil {
			s.destructorCause = e
		}
		return errors.Join(body, e)
	}
	// Source epilogues never invoke owning abort. Incomplete constructors and
	// unjoined acquisitions remain named until an explicit caller Close.
	for i := range f.vectors {
		v := &f.vectors[i]
		if !v.constructed || !v.sourceDone || v.raw.Block != (util.Block{}) || v.raw.Pending != (util.Block{}) || v.copyAcquisitions != nil || v.growthAcquisitions != nil {
			return body
		}
	}
	link := &s.frames
	for *link != nil && *link != f {
		link = &(*link).next
	}
	if *link != f {
		return errors.Join(body, ErrASTHandle)
	}
	*link = f.next
	for i := range f.vectors {
		f.vectors[i].closed = true
	}
	s.budget.Release(f.header)
	f.header = 0
	return body
}
func (s *PsortASTNetwork) Close() error {
	if s == nil || s.self != s {
		return ErrASTHandle
	}
	if s.closed {
		return nil
	}
	if s.running {
		return ErrASTPending
	}
	s.running = true
	defer func() { s.running = false }()
	for s.frames != nil {
		f := s.frames
		if !f.sourceDone && f.destructorCause == nil {
			if e := s.sourceFrame(f); e != nil {
				if s.destructorCause == nil {
					s.destructorCause = e
				}
				return e
			}
		}
		if e := s.ownFrame(f); e != nil {
			return e
		}
	}
	s.budget.Release(s.header)
	s.header = 0
	s.closed = true
	return nil
}
func (s *PsortASTNetwork) Stats() (PsortASTStats, error) {
	if s == nil || s.self != s || s.closed {
		return PsortASTStats{}, ErrASTHandle
	}
	return s.stats, nil
}
func (s *PsortASTNetwork) fresh() (ASTHandle, error) {
	s.stats.CompiledVariables++
	return s.context.Fresh()
}
func (s *PsortASTNetwork) maximum(a, b ASTHandle) (ASTHandle, error) {
	if a == b {
		return a, nil
	}
	s.stats.CompiledVariables++
	return s.context.Max(a, b)
}
func (s *PsortASTNetwork) minimum(a, b ASTHandle) (ASTHandle, error) {
	if a == b {
		return a, nil
	}
	s.stats.CompiledVariables++
	return s.context.Min(a, b)
}
func (s *PsortASTNetwork) not(a ASTHandle) (ASTHandle, error) { return s.context.Not(a) }
func (s *PsortASTNetwork) clause(n uint32, p psortASTPointer) (result error) {
	s.stats.CompiledClauses++
	s.stats.ClauseVariables += n
	f, e := s.beginFrame(1)
	if e != nil {
		return e
	}
	defer func() { result = s.finishFrame(f, result) }()
	tmp := &f.vectors[0]
	if e = s.rangeVector(f, 0, n, p); e != nil {
		return e
	}
	span, e := tmp.CPtr()
	if e != nil {
		return e
	}
	return s.context.Clause(n, span)
}
func (s *PsortASTNetwork) clause2(a, b ASTHandle) error {
	ls := [3]ASTHandle{a, b}
	return s.clause(2, psortASTPointer{inline: &ls})
}
func (s *PsortASTNetwork) clause3(a, b, c ASTHandle) error {
	ls := [3]ASTHandle{a, b, c}
	return s.clause(3, psortASTPointer{inline: &ls})
}
func (s *PsortASTNetwork) clauseVector(v *RuntimeASTLiteralVector) error {
	n, e := v.Size()
	if e != nil {
		return e
	}
	p, e := psortVectorPointer(v)
	if e != nil {
		return e
	}
	return s.clause(n, p)
}
func (s *PsortASTNetwork) appendPointer(v *RuntimeASTLiteralVector, n uint32, p psortASTPointer) error {
	for i := uint32(0); i < n; i++ {
		h, e := p.at(i)
		if e != nil {
			return e
		}
		if e = v.Push(h); e != nil {
			return e
		}
	}
	return nil
}
func (s *PsortASTNetwork) cmpGE(x1, x2, y1, y2 ASTHandle) error {
	n, e := s.not(y2)
	if e != nil {
		return e
	}
	if e = s.clause2(n, x1); e != nil {
		return e
	}
	n, e = s.not(y2)
	if e != nil {
		return e
	}
	if e = s.clause2(n, x2); e != nil {
		return e
	}
	n, e = s.not(y1)
	if e != nil {
		return e
	}
	return s.clause3(n, x1, x2)
}
func (s *PsortASTNetwork) cmpLE(x1, x2, y1, y2 ASTHandle) error {
	n, e := s.not(x1)
	if e != nil {
		return e
	}
	if e = s.clause2(n, y1); e != nil {
		return e
	}
	n, e = s.not(x2)
	if e != nil {
		return e
	}
	if e = s.clause2(n, y1); e != nil {
		return e
	}
	n, e = s.not(x1)
	if e != nil {
		return e
	}
	m, e := s.not(x2)
	if e != nil {
		return e
	}
	return s.clause3(n, m, y2)
}
func (s *PsortASTNetwork) compare(x1, x2, y1, y2 ASTHandle) error {
	mode, e := s.comparison()
	if e != nil {
		return e
	}
	switch mode {
	case psortLE, psortLEFull:
		return s.cmpLE(x1, x2, y1, y2)
	case psortGE, psortGEFull:
		return s.cmpGE(x1, x2, y1, y2)
	case psortEQ:
		if e = s.cmpGE(x1, x2, y1, y2); e != nil {
			return e
		}
		return s.cmpLE(x1, x2, y1, y2)
	}
	return nil
}
func (s *PsortASTNetwork) Ge(full bool, k, n uint32, p ASTLiteralVectorSpan) (ASTHandle, error) {
	if e := s.open(); e != nil {
		return ASTHandle{}, e
	}
	s.running = true
	defer func() { s.running = false }()
	return s.ge(full, k, n, psortASTPointer{span: p})
}
func (s *PsortASTNetwork) Le(full bool, k, n uint32, p ASTLiteralVectorSpan) (ASTHandle, error) {
	if e := s.open(); e != nil {
		return ASTHandle{}, e
	}
	s.running = true
	defer func() { s.running = false }()
	return s.le(full, k, n, psortASTPointer{span: p})
}
func (s *PsortASTNetwork) Eq(full bool, k, n uint32, p ASTLiteralVectorSpan) (ASTHandle, error) {
	if e := s.open(); e != nil {
		return ASTHandle{}, e
	}
	s.running = true
	defer func() { s.running = false }()
	return s.eq(full, k, n, psortASTPointer{span: p})
}
func (s *PsortASTNetwork) Sorting(n uint32, p ASTLiteralVectorSpan, out *RuntimeASTLiteralVector) error {
	if e := s.open(); e != nil {
		return e
	}
	if out == nil || out.self != out {
		return ErrASTHandle
	}
	s.running = true
	defer func() { s.running = false }()
	return s.sorting(n, psortASTPointer{span: p}, out)
}

func (v *RuntimeASTLiteralVector) constructionBorrow() bool {
	f := v.constructionFrame
	return f != nil && f.owner != nil && f.owner.self == f.owner && f.owner.running && f.invocation != 0 && f.invocation <= f.owner.serial && v.constructionIndex == f.cursor && int(v.constructionIndex) < len(f.vectors) && &f.vectors[v.constructionIndex] == v && !v.constructed
}
func (s *PsortASTNetwork) defaultVector(f *psortASTVectorFrame, i uint32) error {
	if f.owner != s || f.invocation == 0 || f.invocation > s.serial || i != f.cursor || int(i) >= len(f.vectors) {
		return ErrASTHandle
	}
	f.vectors[i].constructed = true
	f.cursor++
	return nil
}
func (s *PsortASTNetwork) defaultVectors(f *psortASTVectorFrame) error {
	for f.cursor < uint32(len(f.vectors)) {
		if e := s.defaultVector(f, f.cursor); e != nil {
			return e
		}
	}
	return nil
}
func (s *PsortASTNetwork) rangeVector(f *psortASTVectorFrame, i, n uint32, p psortASTPointer) error {
	if f.owner != s || f.invocation == 0 || f.invocation > s.serial || i != f.cursor || int(i) >= len(f.vectors) {
		return ErrASTHandle
	}
	v := &f.vectors[i]
	v.constructionFrame = f
	v.constructionIndex = i
	for j := uint32(0); j < n; j++ {
		h, e := p.at(j)
		if e != nil {
			return e
		}
		if !v.constructionBorrow() {
			return ErrASTHandle
		}
		if e = v.raw.Open(); e != nil {
			return e
		}
		if e = v.pushReached(h); e != nil {
			return e
		}
	}
	v.constructionFrame = nil
	v.constructed = true
	f.cursor++
	return nil
}

func (s *PsortASTStats) Reset() { *s = PsortASTStats{} }
func (s *PsortASTNetwork) ResetStats() error {
	if e := s.open(); e != nil {
		return e
	}
	s.stats.Reset()
	return nil
}

// SourceFinalize performs only reached lexical source destruction, never
// owning acquisition/archive release. Stopped source history remains sticky.
func (s *PsortASTNetwork) SourceFinalize() error {
	if s == nil || s.self != s {
		return ErrASTHandle
	}
	if s.running {
		return ErrASTPending
	}
	if s.closed || s.sourceDone {
		return nil
	}
	if s.destructorCause != nil {
		return s.destructorCause
	}
	s.running = true
	defer func() { s.running = false }()
	for f := s.frames; f != nil; f = f.next {
		if e := s.sourceFrame(f); e != nil {
			if s.destructorCause == nil {
				s.destructorCause = e
			}
			return e
		}
	}
	s.sourceDone = true
	return nil
}
