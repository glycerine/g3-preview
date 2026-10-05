// Source: src/math/polynomial/upolynomial.h; src/math/polynomial/upolynomial.cpp.
// Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
	"sync/atomic"
)

var ErrUPolynomialOwner = errors.New("mpnum: invalid polynomial owner")
var ErrUPolynomialLive = errors.New("mpnum: live polynomial owners")
var ErrUPolynomialCanceled = errors.New("mpnum: polynomial resource canceled")
var ErrUPolynomialAssertion = errors.New("mpnum: polynomial source assertion")
var upolyOrigins atomic.Uint64

type UPolynomialConfig struct {
	Integer     *IntegerManager
	Checkpoint  func() bool
	Cooperation func() error
}

// The two callbacks borrow actual source resource/cooperation owners. They
// are required, even though operations without a checkpoint do not call them.
// Cold ownership and every actual typed backing are charged to Integer's budget.
type UPolynomialManager struct {
	callbackDepth    uint32
	mignotteCaller   *upolyMignotteScope
	squareFreeCaller *upolyFactorSquareFreeScope
	self             *UPolynomialManager
	origin           uint64
	config           UPolynomialConfig
	modular          *IntegerModularManager
	// Actual source members: 16 core and five derived numeral vectors.
	basic, div1, div2, exactDiv, gcd1, gcd2, cra   upolyVectorState
	mgcd                                           [6]upolyVectorState
	sqf1, sqf2, pw                                 upolyVectorState
	db, dbab1, dbab2, tr, push                     upolyVectorState
	pages                                          []*upolyDescriptorPage
	store                                          *UPolynomialCoefficientStore
	bindings                                       []upolyStoreBinding
	live                                           uint32
	liveSequences                                  uint32
	liveFactors                                    uint32
	header                                         uint64
	closed, closing                                bool
	sourceFinalizing, sourceFinalized, sourceGuard bool
	sourceStage                                    uint8
	sourceIndex                                    uint32
	sourceCause                                    error
}

func NewUPolynomialManager(c UPolynomialConfig) (*UPolynomialManager, error) {
	return newUPolynomialManager(c, nil)
}
func newUPolynomialManager(c UPolynomialConfig, store *UPolynomialCoefficientStore) (*UPolynomialManager, error) {
	if c.Integer == nil || c.Checkpoint == nil || c.Cooperation == nil {
		return nil, arena.ErrConfig
	}
	if c.Integer.closed {
		return nil, arena.ErrClosed
	}
	n := uint64(reflect.TypeOf(UPolynomialManager{}).Size())
	if e := c.Integer.budget.Reserve(n); e != nil {
		return nil, e
	}
	z, e := NewIntegerModularManager(c.Integer)
	if e != nil {
		c.Integer.budget.Release(n)
		return nil, e
	}
	var id uint64
	for {
		old := upolyOrigins.Load()
		if old == math.MaxUint64 {
			z.Close()
			c.Integer.budget.Release(n)
			return nil, arena.ErrSlotLimit
		}
		if upolyOrigins.CompareAndSwap(old, old+1) {
			id = old + 1
			break
		}
	}
	if store == nil {
		store, e = newUPolynomialCoefficientStore(c.Integer, id)
	} else {
		e = store.retain()
	}
	if e != nil {
		z.Close()
		c.Integer.budget.Release(n)
		return nil, e
	}
	m := &UPolynomialManager{origin: id, config: c, modular: z, header: n, store: store}
	m.self = m
	return m, nil
}
func (m *UPolynomialManager) check() error {
	if m == nil || m.self != m || m.config.Integer == nil {
		return ErrUPolynomialOwner
	}
	if m.closed || m.closing || m.sourceFinalizing || m.config.Integer.closed {
		return arena.ErrClosed
	}
	return nil
}
func (m *UPolynomialManager) IntegerManager() *IntegerManager {
	if m == nil || m.self != m {
		return nil
	}
	return m.config.Integer
}
func (m *UPolynomialManager) checkpoint() error {
	m.callbackDepth++
	defer func() { m.callbackDepth-- }()
	if !m.config.Checkpoint() {
		return ErrUPolynomialCanceled
	}
	return m.config.Cooperation()
}
func (m *UPolynomialManager) descriptor(p UPolynomial) (*upolyDescriptor, error) {
	if e := m.check(); e != nil {
		return nil, e
	}
	if p.origin != m.origin || p.slot == 0 {
		return nil, ErrUPolynomialOwner
	}
	i := p.slot - 1
	if uint64(i/32) >= uint64(len(m.pages)) {
		return nil, ErrUPolynomialOwner
	}
	d := &m.pages[i/32].rows[i%32]
	if !d.live || d.generation != p.generation {
		return nil, ErrUPolynomialOwner
	}
	return d, nil
}
func (m *UPolynomialManager) NewPolynomial() (UPolynomial, error) {
	if e := m.check(); e != nil {
		return UPolynomial{}, e
	}
	for i, page := range m.pages {
		for j := range page.rows {
			d := &page.rows[j]
			if !d.live && d.generation != math.MaxUint32 {
				if d.generation == 0 {
					d.generation = 1
				}
				d.live = true
				m.live++
				return UPolynomial{m.origin, uint32(i*32 + j + 1), d.generation}, nil
			}
		}
	}
	if uint64(len(m.pages)+1)*32 > math.MaxUint32 {
		return UPolynomial{}, arena.ErrSlotLimit
	}
	if len(m.pages) == cap(m.pages) {
		next, bytes, e := upolyDirectoryNext(uint64(cap(m.pages)), uint64(reflect.TypeOf((*upolyDescriptorPage)(nil)).Size()), uint64(int(^uint(0)>>1)))
		if e != nil {
			return UPolynomial{}, e
		}
		n := int(next)
		if e := m.config.Integer.budget.Reserve(bytes); e != nil {
			return UPolynomial{}, e
		}
		a := make([]*upolyDescriptorPage, len(m.pages), n)
		copy(a, m.pages)
		m.config.Integer.budget.Release(uint64(cap(m.pages)) * uint64(reflect.TypeOf((*upolyDescriptorPage)(nil)).Size()))
		m.pages = a
	}
	bytes := uint64(reflect.TypeOf(upolyDescriptorPage{}).Size())
	if e := m.config.Integer.budget.Reserve(bytes); e != nil {
		return UPolynomial{}, e
	}
	page := &upolyDescriptorPage{}
	m.pages = append(m.pages, page)
	page.rows[0] = upolyDescriptor{generation: 1, live: true}
	m.live++
	return UPolynomial{m.origin, uint32((len(m.pages)-1)*32 + 1), 1}, nil
}
func (m *UPolynomialManager) DelPolynomial(p *UPolynomial) error {
	if p == nil {
		return ErrUPolynomialOwner
	}
	d, e := m.descriptor(*p)
	if e != nil {
		return e
	}
	if e = m.reset(&d.state); e != nil {
		return e
	}
	m.freeBacking(&d.state)
	d.live = false
	if d.generation != math.MaxUint32 {
		d.generation++
	}
	m.live--
	*p = UPolynomial{}
	return nil
}
func (m *UPolynomialManager) Close() error {
	if m == nil || m.self != m || m.config.Integer == nil {
		return ErrUPolynomialOwner
	}
	if m.mignotteCallPending() {
		return ErrUPolynomialCleanupPending
	}
	if m.closed || m.config.Integer.closed {
		return arena.ErrClosed
	}
	if m.live != 0 || m.liveSequences != 0 || m.liveFactors != 0 {
		return ErrUPolynomialLive
	}
	m.closing = true
	if m.sourceFinalizing && !m.sourceFinalized && m.sourceCause == nil {
		if e := m.SourceFinalize(); e != nil {
			return e
		}
	}
	// Owning cleanup acknowledges actual retained release cursors separately;
	// stopped literal source cause/stage/history are never repaired or advanced.
	// Derived destructor reset order, followed by implicit reverse members.
	derived := [5]*upolyVectorState{&m.db, &m.dbab1, &m.dbab2, &m.tr, &m.push}
	for _, s := range derived {
		if e := m.reset(s); e != nil {
			return e
		}
	}
	for i := len(derived) - 1; i >= 0; i-- {
		m.freeBacking(derived[i])
	}
	core := [16]*upolyVectorState{&m.basic, &m.div1, &m.div2, &m.exactDiv, &m.gcd1, &m.gcd2, &m.cra, &m.mgcd[0], &m.mgcd[1], &m.mgcd[2], &m.mgcd[3], &m.mgcd[4], &m.mgcd[5], &m.sqf1, &m.sqf2, &m.pw}
	for _, s := range core {
		if e := m.reset(s); e != nil {
			return e
		}
	}
	for i := len(core) - 1; i >= 0; i-- {
		m.freeBacking(core[i])
	}
	if e := m.modular.Close(); e != nil {
		return e
	}
	b := m.config.Integer.budget
	b.Release(uint64(len(m.pages)) * uint64(reflect.TypeOf(upolyDescriptorPage{}).Size()))
	b.Release(uint64(cap(m.pages)) * uint64(reflect.TypeOf((*upolyDescriptorPage)(nil)).Size()))
	m.store.release()
	for _, binding := range m.bindings {
		binding.store.release()
	}
	b.Release(uint64(cap(m.bindings)) * uint64(reflect.TypeOf(upolyStoreBinding{}).Size()))
	m.bindings = nil
	b.Release(m.header)
	m.pages = nil

	m.closed = true
	return nil
}
