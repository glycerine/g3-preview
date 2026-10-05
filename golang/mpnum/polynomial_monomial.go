// Source: src/math/polynomial/polynomial.cpp; src/math/polynomial/polynomial.h. Licensed under the MIT license.
// Actual standalone polynomial.cpp monomial owner. This is not polynomial::imp.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/utilid"
	"math"
	"reflect"
	"runtime"
	"sync/atomic"
)

var ErrMonomialOwner = errors.New("mpnum: invalid monomial owner")
var ErrMonomialAssertion = errors.New("mpnum: monomial source assertion")
var ErrMonomialPending = errors.New("mpnum: retained monomial construction")
var ErrMonomialLive = errors.New("mpnum: live caller monomials")
var monomialOrigins atomic.Uint64

type Monomial struct {
	owner            uint64
	slot, generation uint32
}
type MonomialPower struct{ Var, Degree uint32 }
type MonomialInfo struct{ RefCount, ID, TotalDegree, Size, Hash uint32 }
type MonomialManagerConfig struct {
	Budget    *arena.Budget
	Memory    *memory.Runtime
	Allocator *memory.SmallAllocator
	Profile   memory.SmallAllocatorProfile
}
type monomialLocation struct {
	arithmeticCohort                   *polynomialArithmeticScope
	arithmeticInvocation               uint64
	conversionCohort                   *polynomialGlobalConversionScope
	conversionSlot                     uint32
	compositionCohort                  *polynomialCompositionScope
	compositionSlot                    uint32
	squareRootCohort                   *polynomialSquareRootScope
	squareRootSlot                     uint32
	derivativeCohort                   *polynomialDerivativeScope
	derivativeSlot                     uint32
	unreferencedCohort                 *polynomialSubstitutionState
	unreferencedSlot                   uint32
	raw                                memory.ObjectRef
	sourceRaw                          memory.Block
	sourceOffset                       uint64
	generation, requested, id          uint32
	stage                              uint8
	live, headerInitialized, published bool
}
type monomialCreation struct {
	coefficientParent        *polynomialCoefficientScope
	coefficientInvocation    uint64
	generation               uint64
	parent                   *MultivariatePolynomialCore
	tableIndex, slot, id     uint32
	stage                    uint8
	idConsumed, tableRemoved bool
}
type monomialPowerVector struct{ raw, pending memory.Block }
type MonomialManager struct {
	allocatorInvocation                                                       monomialAllocatorScope
	unitDropConsumed                                                          bool
	unitDropCause                                                             error
	continuationGeneration                                                    uint64
	borrowedGeneration                                                        uint64
	borrowedParent                                                            *MultivariatePolynomialCore
	borrowedTemp                                                              *monomialTemp // actual borrowed imp temporary pointer; no ownership
	self                                                                      *MonomialManager
	config                                                                    MonomialManagerConfig
	memory                                                                    *memory.Runtime
	allocator                                                                 *memory.SmallAllocator
	ids                                                                       utilid.Generator
	table                                                                     monomialTable
	temps                                                                     [4]monomialTemp
	powers                                                                    monomialPowerVector
	renameTable                                                               monomialTable
	renameFound                                                               memory.Block
	locations                                                                 []monomialLocation
	creation                                                                  monomialCreation
	unit                                                                      Monomial
	origin, header                                                            uint64
	nextVar, managerRefs, publishedObjects                                    uint32
	completedTemps                                                            uint8
	idsInitialized                                                            bool
	allocatorOwned, memoryOwned, completed, closing, closed, sourceFinalizing bool
	sourceStage                                                               uint8
	originalCause, cleanupCause                                               error
}

func NewMonomialManager(c MonomialManagerConfig) (*MonomialManager, error) {
	if c.Budget == nil {
		return nil, arena.ErrConfig
	}
	r := c.Memory
	owned := false
	if c.Allocator != nil {
		if _, e := c.Allocator.AllocationSize(); e != nil {
			return nil, e
		}
		if r == nil {
			r = c.Allocator.RuntimeOwner()
		}
		if r != c.Allocator.RuntimeOwner() {
			return nil, arena.ErrConfig
		}
	}
	if r != nil {
		if r.Budget() != c.Budget {
			return nil, arena.ErrConfig
		}
		if e := r.CheckOpen(); e != nil {
			return nil, e
		}
	} else {
		var e error
		r, e = memory.NewRuntime(memory.RuntimeConfig{Budget: c.Budget, SizeTBits: c.Profile.PointerBits})
		if e != nil {
			return nil, e
		}
		owned = true
	}
	if c.Profile.PointerBits == 0 {
		c.Profile = memory.SmallAllocatorProfile{PointerBits: r.SizeTBits(), Debug: modularSourceDebug, Windows: runtime.GOOS == "windows"}
	}
	if c.Profile.PointerBits != r.SizeTBits() {
		if owned {
			_ = r.Close()
		}
		return nil, arena.ErrConfig
	}
	h := uint64(reflect.TypeOf(MonomialManager{}).Size())
	if e := c.Budget.Reserve(h); e != nil {
		if owned {
			if cleanup := r.Close(); cleanup != nil {
				return nil, errors.Join(e, cleanup)
			}
		}
		return nil, e
	}
	m := &MonomialManager{config: c, memory: r, memoryOwned: owned, header: h}
	m.self = m
	for {
		old := monomialOrigins.Load()
		if old == math.MaxUint64 {
			c.Budget.Release(h)
			if owned {
				_ = r.Close()
			}
			return nil, arena.ErrSlotLimit
		}
		if monomialOrigins.CompareAndSwap(old, old+1) {
			m.origin = old + 1
			break
		}
	}
	fail := func(e error) (*MonomialManager, error) {
		m.originalCause = e
		m.cleanupCause = m.constructorUnwind()
		return m, m
	}
	if e := m.table.construct(m, 8, 2); e != nil {
		return fail(e)
	}
	if e := m.ids.Init(0, r); e != nil {
		return fail(e)
	}
	m.idsInitialized = true
	for i := range m.temps {
		if e := m.tempConstruct(&m.temps[i]); e != nil {
			return fail(e)
		}
		m.completedTemps++
	}
	m.allocator = c.Allocator
	if m.allocator == nil {
		a, e := memory.NewSmallAllocator(memory.SmallAllocatorConfig{Runtime: r, Profile: c.Profile})
		if e != nil {
			return fail(e)
		}
		m.allocator = a
		m.allocatorOwned = true
	}
	unit, e := m.MkMonomial(nil)
	if e != nil {
		return fail(e)
	}
	m.unit = unit
	if e = m.incRefCore(unit); e != nil {
		return fail(e)
	}
	m.completed = true
	return m, nil
}
func (m *MonomialManager) Error() string {
	if m == nil {
		return "mpnum: failed monomial construction"
	}
	if m.originalCause != nil {
		return m.originalCause.Error()
	}
	if m.cleanupCause != nil {
		return m.cleanupCause.Error()
	}
	return ErrMonomialPending.Error()
}
func (m *MonomialManager) Unwrap() error {
	if m == nil {
		return nil
	}
	return m.originalCause
}
func (m *MonomialManager) OriginalCause() error {
	if m == nil {
		return nil
	}
	return m.originalCause
}
func (m *MonomialManager) CleanupCause() error {
	if m == nil {
		return nil
	}
	return m.cleanupCause
}
func (m *MonomialManager) check() error {
	if m == nil || m.self != m {
		return ErrMonomialOwner
	}
	if m.closed || m.closing || m.sourceFinalizing || m.originalCause != nil {
		return arena.ErrClosed
	}
	return m.memory.CheckOpen()
}
func (m *MonomialManager) RuntimeOwner() *memory.Runtime {
	if m == nil || m.self != m {
		return nil
	}
	return m.memory
}
func (m *MonomialManager) NumVars() (uint32, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	return m.nextVar, nil
}
func (m *MonomialManager) IsValidVar(v uint32) (bool, error) {
	if e := m.check(); e != nil {
		return false, e
	}
	return v < m.nextVar, nil
}
func (m *MonomialManager) MkVar() (uint32, error) {
	if e := m.check(); e != nil {
		return 0, e
	}
	v := m.nextVar
	m.nextVar++
	return v, nil
}
func (m *MonomialManager) Unit() (Monomial, error) {
	if e := m.check(); e != nil {
		return Monomial{}, e
	}
	return m.unit, nil
}
func (m *MonomialManager) location(a Monomial) (*monomialLocation, error) {
	if a.owner != m.origin || a.slot == 0 || uint64(a.slot) > uint64(len(m.locations)) {
		return nil, ErrMonomialOwner
	}
	l := &m.locations[a.slot-1]
	if !l.live || l.generation != a.generation {
		return nil, ErrMonomialOwner
	}
	return l, nil
}
func (m *MonomialManager) headerCore(a Monomial) (MonomialInfo, error) {
	var out MonomialInfo
	l, e := m.location(a)
	if e != nil {
		return out, e
	}
	values := [5]*uint32{&out.RefCount, &out.ID, &out.TotalDegree, &out.Size, &out.Hash}
	for i, p := range values {
		v, e := m.allocator.ReadUint32(l.raw, uint64(i)*4)
		if e != nil {
			return out, e
		}
		*p = v
	}
	return out, nil
}
func (m *MonomialManager) Info(a Monomial) (MonomialInfo, error) {
	if e := m.check(); e != nil {
		return MonomialInfo{}, e
	}
	return m.headerCore(a)
}
func (m *MonomialManager) powerCore(a Monomial, i uint32) (MonomialPower, error) {
	var out MonomialPower
	l, e := m.location(a)
	if e != nil {
		return out, e
	}
	if modularSourceDebug {
		n, e := m.allocator.ReadUint32(l.raw, 12)
		if e != nil {
			return out, e
		}
		if i >= n {
			return out, ErrMonomialAssertion
		}
	}
	out.Var, e = m.allocator.ReadUint32(l.raw, 20+uint64(i)*8)
	if e != nil {
		return out, e
	}
	out.Degree, e = m.allocator.ReadUint32(l.raw, 24+uint64(i)*8)
	return out, e
}
func (m *MonomialManager) powerIndexGuard(n, i uint32) error {
	if i >= n {
		return arena.ErrBounds
	}
	return nil
}
func (m *MonomialManager) ReadPower(a Monomial, i uint32) (MonomialPower, error) {
	if e := m.check(); e != nil {
		return MonomialPower{}, e
	}
	return m.powerCore(a, i)
}
func (m *MonomialManager) prepareLocation() (uint32, error) {
	for i := range m.locations {
		l := &m.locations[i]
		if !l.live && l.generation < math.MaxUint32 {
			generation := l.generation + 1
			*l = monomialLocation{generation: generation}
			return uint32(i + 1), nil
		}
	}
	if uint64(len(m.locations)) == uint64(math.MaxUint32) {
		return 0, arena.ErrSlotLimit
	}
	if len(m.locations) == cap(m.locations) {
		n := uint64(cap(m.locations)) * 3 / 2
		if n == 0 {
			n = 2
		}
		if n <= uint64(cap(m.locations)) {
			n = uint64(cap(m.locations)) + 1
		}
		stride := uint64(reflect.TypeOf(monomialLocation{}).Size())
		if n > uint64(^uint(0)>>1)/stride {
			return 0, arena.ErrBounds
		}
		if e := m.config.Budget.Reserve(n * stride); e != nil {
			return 0, e
		}
		next := make([]monomialLocation, len(m.locations), int(n))
		copy(next, m.locations)
		m.config.Budget.Release(uint64(cap(m.locations)) * stride)
		m.locations = next
	}
	m.locations = append(m.locations, monomialLocation{generation: 1})
	return uint32(len(m.locations)), nil
}
func (m *MonomialManager) incRefCore(a Monomial) error {
	l, e := m.location(a)
	if e != nil {
		return e
	}
	n, e := m.allocator.ReadUint32(l.raw, 0)
	if e != nil {
		return e
	}
	if e = m.allocator.WriteUint32(l.raw, 0, n+1); e != nil {
		return e
	}
	l.acknowledgeArithmeticProduct(a)
	if d := l.compositionCohort; d != nil && uint64(l.compositionSlot) < uint64(len(d.orphans)) && d.orphans[l.compositionSlot] == a {
		d.orphans[l.compositionSlot] = Monomial{}
		l.compositionCohort = nil
		l.compositionSlot = 0
	}
	if d := l.squareRootCohort; d != nil && uint64(l.squareRootSlot) < uint64(len(d.orphans)) && d.orphans[l.squareRootSlot] == a {
		d.orphans[l.squareRootSlot] = Monomial{}
		d.orphanInvocations[l.squareRootSlot] = 0
		l.squareRootCohort = nil
		l.squareRootSlot = 0
	}
	if d := l.conversionCohort; d != nil && uint64(l.conversionSlot) < uint64(len(d.orphans)) && d.orphans[l.conversionSlot] == a {
		d.orphans[l.conversionSlot] = Monomial{}
		l.conversionCohort = nil
		l.conversionSlot = 0
	}
	// Source INC is complete. The actual cohort slot alone grants ownership;
	// the cold location tag only routes this infallible, allocation-free transfer.
	if d := l.unreferencedCohort; d != nil && uint64(l.unreferencedSlot) < uint64(len(d.orphans)) && d.orphans[l.unreferencedSlot] == a {
		d.orphans[l.unreferencedSlot] = Monomial{}
		l.unreferencedCohort = nil
		l.unreferencedSlot = 0
	}

	if d := l.derivativeCohort; d != nil && uint64(l.derivativeSlot) < uint64(len(d.orphans)) && d.orphans[l.derivativeSlot] == a {
		d.orphans[l.derivativeSlot] = Monomial{}
		d.orphanInvocations[l.derivativeSlot] = 0
		l.derivativeCohort = nil
		l.derivativeSlot = 0
	}
	return nil
}
func (m *MonomialManager) IncRef(a Monomial) error {
	if e := m.check(); e != nil {
		return e
	}
	return m.incRefCore(a)
}
func (m *MonomialManager) DecRef(a Monomial) error {
	if e := m.check(); e != nil {
		return e
	}
	return m.decRefCore(a)
}

type monomialDecrementOutcome struct{ written, last bool }

func (m *MonomialManager) decRefCore(a Monomial) error { return m.decRefCoreOutcome(a, nil) }
func (m *MonomialManager) decRefCoreOutcome(a Monomial, out *monomialDecrementOutcome) error {
	l, e := m.location(a)
	if e != nil {
		return e
	}
	if l.stage != 0 {
		return m.deleteCore(a, l)
	}
	n, e := m.allocator.ReadUint32(l.raw, 0)
	if e != nil {
		return e
	}
	if modularSourceDebug && n == 0 {
		return ErrMonomialAssertion
	}
	n--
	if e = m.allocator.WriteUint32(l.raw, 0, n); e != nil {
		return e
	}
	if out != nil {
		out.written = true
		out.last = n == 0
	}
	if n == 0 {
		return m.deleteCore(a, l)
	}
	return nil
}
func (m *MonomialManager) deleteCore(a Monomial, l *monomialLocation) error {
	if l.stage == 0 {
		n, e := m.allocator.ReadUint32(l.raw, 12)
		if e != nil {
			return e
		}
		l.requested = monomialObjectSize(n)
		if e := m.table.erase(m, monomialKey{kind: monomialObjectKey, slot: a.slot}); e != nil {
			return e
		}
		l.stage = 1
	}
	if l.stage == 1 {
		id, e := m.allocator.ReadUint32(l.raw, 4)
		if e != nil {
			return e
		}
		l.id = id
		if e := m.ids.Recycle(l.id); e != nil {
			return e
		}
		l.stage = 2
	}
	if l.raw != (memory.ObjectRef{}) {
		progress, e := m.allocator.DeallocateTracked(uint64(l.requested), l.raw)
		if progress.PhysicalReleased || progress.PooledLinked {
			l.raw = memory.ObjectRef{}
		}
		if e != nil {
			return e
		}
	}
	if l.published {
		m.publishedObjects--
		l.published = false
	}
	l.live = false
	l.stage = 0
	return nil
}
