// Source: src/util/mpf.cpp:61-67,1519-1569; src/util/mpf.h:65-73,241-259;
// src/util/map.h:285; src/util/hashtable.h:245-257.
// Source author: Christoph Wintersteiger (2011-12-01).
// Licensed under the MIT license; see LICENSE.txt.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/utilhash"
	"reflect"
)

// This is the actual MPF constructor/destructor dependency, distinct from the
// fixed-precision mpff implementation. Pointer values in powers2 are compressed
// manager-local mpz-wrapper IDs. No wrapper exists in default empty maps.
// MPF numeral algorithms and populated powers2 operations remain separate.
type ArbitraryFloatManager struct {
	rationalSetHistories     *arbitraryFloatRationalSetHistory
	rationalSetHistoryBytes  uint64
	rationalSet              arbitraryFloatRationalSetScope
	formattedSet             arbitraryFloatFormattedSetScope
	formattedSetHistories    *arbitraryFloatFormattedSetHistory
	formattedSetHistoryBytes uint64

	roundHistories                                          *arbitraryFloatRoundHistory
	roundHistoryBytes                                       uint64
	roundScope                                              arbitraryFloatRoundScope
	roundSqrtScope                                          arbitraryFloatRoundScope
	inputBinding                                            arbitraryFloatInputBinding
	inputInvocation, inputHistoryBytes                      uint64
	inputCalls                                              *arbitraryFloatValueInputCall
	conversionHistory                                       *arbitraryFloatRationalConversionHistory
	conversionHistoryBytes                                  uint64
	rationalConversionInvocation                            uint64
	rationalConversion                                      arbitraryFloatRationalConversionScope
	formatScopes                                            *ArbitraryFloatFormatInvocation
	formatInvocation                                        uint64
	powerWrappers                                           *arbitraryFloatPowerWrapper
	powerNextID                                             uint32
	powerWrapperBytes                                       uint64
	self                                                    *ArbitraryFloatManager
	rational                                                *RationalManager
	integer                                                 *IntegerManager     // source reference to the exact same mpq base
	powers                                                  [4]utilhash.UintMap // p, pn, pm1, pm1n; actual inline owners
	memory                                                  *memory.Runtime
	budget                                                  *arena.Budget
	valueExecuting                                          bool
	valueCopy                                               integerCopyAllocationReceipt
	valueBinding                                            arbitraryFloatCopyBinding
	valueCalls                                              *arbitraryFloatValueCopyCall
	valueInvocation, valueHistoryBytes                      uint64
	valueCopyAcknowledged                                   bool
	failedChild                                             error
	bodyCause, sourceCause, ownCause                        error
	header                                                  uint64
	completedMaps                                           uint8
	rationalReady, constructorComplete, sourceEnded, closed bool
}

func NewArbitraryFloatManager(c IntegerConfig) (*ArbitraryFloatManager, error) {
	if c.Cells.Budget != c.Words.Descriptors.Budget {
		return nil, arena.ErrConfig
	}
	b := c.Cells.Budget
	bytes := uint64(reflect.TypeOf(ArbitraryFloatManager{}).Size())
	if e := b.Reserve(bytes); e != nil {
		return nil, e
	}
	m := &ArbitraryFloatManager{budget: b, header: bytes, memory: c.Memory}
	m.self = m
	q, e := NewRationalManager(RationalConfig{Integers: c, Synchronized: false})
	if e != nil {
		m.failedChild = e
		var child ConstructionFailure
		if errors.As(e, &child) {
			m.memory = child.RuntimeOwner()
		}
		m.bodyCause = e
		return nil, m
	}
	m.rational = q
	m.memory = q.RuntimeOwner()
	m.integer = q.z
	q.mpfParent = m
	m.integer.mpfParent = m
	m.rationalReady = true
	for i := range m.powers {
		if e = utilhash.InitializeInlineUintMap(&m.powers[i], utilhash.UintMapConfig{Budget: b}); e != nil {
			m.bodyCause = e
			// Only completed source members unwind; no powers2 destructor BODY is
			// invoked on this incomplete object. Map members unwind in reverse.
			for j := int(m.completedMaps) - 1; j >= 0; j-- {
				m.powers[j].Close()
			}
			if dt := q.SourceFinalize(); dt != nil {
				m.sourceCause = dt
			} else {
				m.rationalReady = false
			}
			return nil, m
		}
		m.completedMaps++
	}
	m.constructorComplete = true
	return m, nil
}
func (m *ArbitraryFloatManager) Error() string { return "mpnum: retained arbitrary float manager" }
func (m *ArbitraryFloatManager) Unwrap() error {
	return errors.Join(m.bodyCause, m.sourceCause, m.ownCause)
}
func (m *ArbitraryFloatManager) OriginalCause() error { return m.bodyCause }
func (m *ArbitraryFloatManager) CleanupCause() error  { return m.ownCause }
func (m *ArbitraryFloatManager) RuntimeOwner() *memory.Runtime {
	if m == nil || m.self != m {
		return nil
	}
	return m.memory
}
func (m *ArbitraryFloatManager) Cleanup() error { return m.Close() }
func (m *ArbitraryFloatManager) SourceFinalize() error {
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	if m.rationalSet.executing || m.rationalSet.finishing || (m.integer != nil && m.integer.rationalServiceReadCallbackActive()) || m.formattedSet.executing || m.formattedSet.finishing || m.rationalConversion.executing || m.rationalConversion.finishing || m.valueExecuting || m.formatInvocationExecuting() || (m.integer != nil && m.integer.mpfBorrowedSourceCallbackActive()) {
		return ErrPolynomialPending
	}
	// The caller-owned format context is a younger lexical scope. Query only
	// actual issued frames before touching cached class fields; ownership of
	// its unjoined leaves remains with the text caller.
	for f := m.formatScopes; f != nil; f = f.next {
		if !m.matchesFormatFrame(f) {
			return memory.ErrRuntimeOwner
		}
		if e := f.SourceReady(); e != nil && !f.owned {
			return e
		}
		if m.formatFrameUnfinished(f) {
			return ErrPolynomialPending
		}
	}
	if !m.constructorComplete {
		return arena.ErrClosed
	}
	if e := m.finishRationalSetSource(); e != nil {
		if m.sourceCause == nil {
			m.sourceCause = e
		}
		return e
	}
	if e := m.finishFormattedSetSource(); e != nil {
		if m.sourceCause == nil {
			m.sourceCause = e
		}
		return e
	}
	if e := m.finishRoundSqrtSource(); e != nil {
		if m.sourceCause == nil {
			m.sourceCause = e
		}
		return e
	}
	if e := m.finishRoundSource(); e != nil {
		if m.sourceCause == nil {
			m.sourceCause = e
		}
		return e
	}
	if m.sourceCause != nil {
		return m.sourceCause
	}
	if m.sourceEnded {
		return nil
	}
	// All source NUMs and exact allocation children have already ended.
	// Retire only their now-empty cleanup protocol before the numerical
	// owner disappears; the caller's stream/result storage remains untouched.
	for f := m.formatScopes; f != nil; f = f.next {
		if !f.owned {
			f.owned = true
			f.active = false
			f.ownCursor = 5
		}
	}
	// The reached conversion is younger than this owner's cached values.
	// Completed source locals permit the class suffix even when exact unjoined
	// allocation roots still await separate owning acknowledgment.
	if m.rationalConversion.manager == m && m.rationalConversion.invocation != 0 {
		if e := m.finishRationalConversionSource(); e != nil {
			m.sourceCause = e
			return e
		}
	}
	if e := m.destroyPowerWrappers(false); e != nil {
		m.sourceCause = e
		return e
	}
	for i := 3; i >= 0; i-- {
		m.powers[i].Close()
	}
	if e := m.rational.SourceFinalize(); e != nil {
		m.sourceCause = e
		return e
	}
	m.rationalReady = false
	m.sourceEnded = true
	return nil
}
func (m *ArbitraryFloatManager) Close() error {
	if m == nil || m.self != m {
		return memory.ErrRuntimeOwner
	}
	if m.rationalSet.executing || m.rationalSet.finishing || (m.integer != nil && m.integer.rationalServiceReadCallbackActive()) || m.formattedSet.executing || m.formattedSet.finishing || m.rationalConversion.executing || m.rationalConversion.finishing || m.valueExecuting || m.formatInvocationExecuting() || (m.integer != nil && m.integer.mpfBorrowedSourceCallbackActive()) {
		return ErrPolynomialPending
	}
	// Registered diagnostic metadata is not ownership. Refuse only actual
	// unfinished lexical/owning children before retiring older class fields.
	for f := m.formatScopes; f != nil; f = f.next {
		if !m.matchesFormatFrame(f) || m.formatFrameUnfinished(f) {
			return ErrPolynomialPending
		}
	}
	if m.closed {
		return nil
	}
	stop := func(e error) error {
		if e != nil && m.ownCause == nil {
			m.ownCause = e
		}
		return e
	}
	if e := m.CleanupFormattedSet(); e != nil {
		return stop(e)
	}
	if e := m.CleanupRoundSqrt(); e != nil {
		return stop(e)
	}
	if e := m.CleanupExponentRationalSet(); e != nil {
		return stop(e)
	}
	if e := m.CleanupRound(); e != nil {
		return stop(e)
	}
	// Retire the exact younger lexical conversion before older class members.
	// Its helper acknowledges completed children, owns local NUMs, then aborts
	// only the captured unjoined allocation children.
	if m.rationalConversion.manager == m && m.rationalConversion.invocation != 0 {
		if e := m.CleanupRationalConversion(); e != nil {
			return stop(e)
		}
	}
	if e := m.cleanupValueInputs(); e != nil {
		return stop(e)
	}
	if e := m.cleanupValueCopyCalls(); e != nil {
		return stop(e)
	}
	// A stopped value copy retains only the exact acquired child invocation.
	// Owning cleanup acknowledges it before destroying the older mpq member;
	// the immutable receipt is retained through the enclosing context end.
	if m.valueCopy.invocation != 0 && !m.valueCopyAcknowledged {
		if m.integer == nil || !m.integer.matchesCopyAllocationInvocation(&m.valueCopy) {
			return stop(ErrPolynomialPending)
		}
		if e := m.integer.cleanupCopyAllocationInvocation(&m.valueCopy); e != nil {
			return stop(e)
		}
		m.valueCopyAcknowledged = true
	}
	if m.constructorComplete && !m.sourceEnded && m.sourceCause == nil {
		if e := m.SourceFinalize(); e != nil {
			return stop(e)
		}
	}
	if m.failedChild != nil {
		var child ConstructionFailure
		if errors.As(m.failedChild, &child) {
			if child.RuntimeOwner() != m.memory {
				return stop(memory.ErrRuntimeOwner)
			}
			if e := child.Cleanup(); e != nil {
				return stop(e)
			}
		}
		m.failedChild = nil
	}
	if e := m.destroyPowerWrappers(true); e != nil {
		return stop(e)
	}
	for i := int(m.completedMaps) - 1; i >= 0; i-- {
		m.powers[i].Close()
	}
	if m.rational != nil {
		if e := m.rational.CloseChecked(); e != nil {
			return stop(e)
		}
	}
	m.releasePowerWrapperHistory()
	m.releaseValueCopyHistory()
	m.releaseRationalSetHistories()
	m.releaseFormattedSetHistories()
	m.releaseRoundHistory()
	m.releaseValueInputHistory()
	m.releaseRationalConversionHistory()
	m.budget.Release(m.header)
	m.header = 0
	m.closed = true
	return nil
}

// OwnedStorageBytes reports this exact context's actual cold/payload storage.
// The borrowed Runtime is excluded; its physical block directory is charged by
// the shared Runtime owner once. This is accounting, not root acquisition.
func (m *ArbitraryFloatManager) OwnedStorageBytes() uint64 {
	if m == nil || m.self != m {
		panic(memory.ErrRuntimeOwner)
	}
	n := m.header + m.rationalSetHistoryBytes + m.formattedSetHistoryBytes + m.roundHistoryBytes + m.valueHistoryBytes + m.powerWrapperBytes + m.conversionHistoryBytes + m.inputHistoryBytes
	var z *IntegerManager
	if m.rational != nil {
		n += m.rational.headerBytes
		z = m.rational.z
	} else {
		var failed *IntegerManager
		if errors.As(m.failedChild, &failed) {
			z = failed
		}
	}
	if z != nil {
		n += z.headerBytes
		if z.allocator != nil {
			n += z.allocator.Stats().TotalBytes
		}
		if z.cells != nil {
			s := z.cells.Stats()
			n += s.OwnerBytes + s.RecordBytes + s.MetadataBytes + s.DirectoryBytes
		}
		if z.words != nil {
			s := z.words.Stats()
			n += s.OwnerBytes + s.PayloadBytes + s.BlockMetadataBytes + s.DirectoryBytes + s.Descriptors.OwnerBytes + s.Descriptors.RecordBytes + s.Descriptors.MetadataBytes + s.Descriptors.DirectoryBytes
		}
		if z.natural != nil {
			n += z.natural.headerBytes
			if z.natural.text != nil {
				s := z.natural.text.Stats()
				n += s.OwnerBytes + s.PayloadBytes + s.MetadataBytes + s.DirectoryBytes
			}
		}
	}
	for i := range m.powers {
		n += m.powers[i].StorageBytes()
	}
	return n
}

// formatFrameUnfinished reads only exact issued source locals and returned
// child receipts. Completed erased invocations and owning-acknowledged
// diagnostic records do not hold the older numerical manager alive.
func (m *ArbitraryFloatManager) formatFrameUnfinished(f *ArbitraryFloatFormatInvocation) bool {
	if f.owned {
		return false
	}
	if f.sourceCursor != 5 || polynomialNegNumericPending(&f.numeric) || f.transferred.Block() != (memory.Block{}) {
		return true
	}
	if polynomialNegNumericPending(&f.decimal.numeric) || f.decimal.transferred.Block() != (memory.Block{}) {
		return true
	}
	for _, buffers := range [][4]naturalPNBufferReceipt{f.buffers, f.decimal.buffers} {
		for _, b := range buffers {
			if b.pending != (naturalPNBufferReceipt{}).pending {
				return true
			}
		}
	}
	for _, v := range []*naturalPNVerificationReceipt{&f.verification, &f.decimal.verification} {
		if m.integer.natural != nil && m.integer.natural.matchesPNVerification(v) {
			return true
		}
	}
	if f.decimal.insertion.active && (!f.decimal.insertion.sourceDone || f.decimal.insertion.characterOwned || f.decimal.insertion.formattingOwned) {
		return true
	}
	if f.decimal.parent != nil && !f.decimal.ended && !f.decimal.owned {
		return true
	}
	for _, r := range []*integerRawInvocationReceipt{&f.gcd, &f.division, &f.extended, &f.decimal.division} {
		if m.integer.matchesRawInvocation(r) && (!r.frame.ended || r.frame.pending(m.integer)) {
			return true
		}
	}
	return false
}
