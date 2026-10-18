// Source: src/math/polynomial/polynomial_cache.cpp:35-92,129-227.
package polynomial

import (
	"errors"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

// PolynomialCache owns canonical representatives and memoized results used by
// the retained NLSAT cache.
type PolynomialCache struct {
	pscCall        polynomialPSCInvocationReceipt
	factorCall     polynomialFactorCoreInvocationReceipt
	self           *PolynomialCache
	manager        *MultivariatePolynomialManager
	refs           polynomialReferenceVector
	buckets        map[uint32][]Polynomial
	ids            map[uint32]bool
	psc            map[polynomialPSCKey]*polynomialPSCEntry
	factors        map[Polynomial]*polynomialFactorCacheEntry
	pendingFactors *PolynomialFactors
	pendingPSC     bool
	header         uint64
	imp            *polynomialCacheImpOwner
	resetting      bool
	closing        bool
	closed         bool
}

type polynomialPSCKey struct {
	p, q Polynomial
	x    uint32
}

type polynomialPSCEntry struct {
	result   []Polynomial
	complete bool
}

type polynomialFactorCacheEntry struct {
	result   []Polynomial
	complete bool
}

func NewPolynomialCache(m *MultivariatePolynomialManager) (*PolynomialCache, error) {
	if e := m.ready(); e != nil {
		return nil, e
	}
	header := uint64(reflect.TypeOf(PolynomialCache{}).Size())
	if e := m.core.integer.Budget().Reserve(header); e != nil {
		return nil, e
	}
	c := &PolynomialCache{manager: m, header: header, buckets: make(map[uint32][]Polynomial), ids: make(map[uint32]bool), psc: make(map[polynomialPSCKey]*polynomialPSCEntry), factors: make(map[Polynomial]*polynomialFactorCacheEntry)}
	c.self = c
	if e := c.newImp(207); e != nil {
		m.core.integer.Budget().Release(header)
		return nil, e
	}
	return c, nil
}

// Factor returns distinct factors, dropping multiplicities and the source
// factorization constant. The result cache is keyed by the canonical input.
func (c *PolynomialCache) Factor(p Polynomial, out *PolynomialRefVector) error {
	if e := c.ready(); e != nil {
		return e
	}
	if out == nil || out.owner != c.manager.core {
		return ErrPolynomialOwner
	}
	if e := out.Reset(); e != nil {
		return e
	}
	var e error
	p, e = c.MkUnique(p)
	if e != nil {
		return e
	}
	if entry, ok := c.factors[p]; ok {
		// Source entries start with result size0; interruption alone does not
		// invalidate a later empty hit. Unwritten published slots are not values.
		if !entry.complete && len(entry.result) != 0 {
			return ErrPolynomialPending
		}
		if e := out.Reset(); e != nil {
			return e
		}
		for _, value := range entry.result {
			if e := out.Push(value); e != nil {
				return e
			}
		}
		return nil
	}
	entry := &polynomialFactorCacheEntry{}
	c.factors[p] = entry
	c.factorCall = polynomialFactorCoreInvocationReceipt{}
	fs, e := NewPolynomialFactors(c.manager)
	if e != nil {
		if fs != nil {
			return c.finishFactorOwner(fs, e)
		}
		return e
	}
	if e := c.manager.factorWithReceipt(p, fs, DefaultUPolynomialFactorParams(), &c.factorCall); e != nil {
		return c.finishFactorOwner(fs, e)
	}
	n, e := fs.Distinct()
	if e != nil {
		return c.finishFactorOwner(fs, e)
	}
	entry.result = make([]Polynomial, n)
	for i := uint32(0); i < n; i++ {
		value, e := fs.factor(i)
		if e != nil {
			return c.finishFactorOwner(fs, e)
		}
		value, e = c.MkUnique(value)
		if e != nil {
			return c.finishFactorOwner(fs, e)
		}
		if e := out.Push(value); e != nil {
			return c.finishFactorOwner(fs, e)
		}
		entry.result[i] = value
	}
	// The cache is filled before the local source factors destructor runs.
	entry.complete = true
	if e := c.finishFactorOwner(fs, nil); e != nil {
		return e
	}
	return nil
}

func (c *PolynomialCache) finishFactorOwner(fs *PolynomialFactors, body error) error {
	if body != nil {
		if e := c.cleanupFactorInvocation(); e != nil {
			c.pendingFactors = fs
			return errors.Join(body, e)
		}
	}
	if e := fs.CloseChecked(); e != nil {
		c.pendingFactors = fs
		return errors.Join(body, e)
	}
	return body
}

// CleanupFactor completes a temporary factor owner after an interrupted call.
func (c *PolynomialCache) CleanupFactor() error {
	if c == nil || c.self != c {
		return ErrPolynomialOwner
	}
	if c.closed {
		return arena.ErrClosed
	}
	if c.pendingFactors == nil {
		return nil
	}
	if e := c.cleanupFactorInvocation(); e != nil {
		return e
	}
	if e := c.pendingFactors.CloseChecked(); e != nil {
		return e
	}
	c.pendingFactors = nil
	return nil
}

func (c *PolynomialCache) finishPSC(body error) error {
	if e := c.cleanupPSCInvocation(); e != nil {
		c.pendingPSC = true
		return errors.Join(body, e)
	}
	return body
}

// CleanupPSC completes the manager's source PSC scope after an interrupted
// cache miss. The caller still owns its output reference vector.
func (c *PolynomialCache) CleanupPSC() error {
	if c == nil || c.self != c {
		return ErrPolynomialOwner
	}
	if c.closed {
		return arena.ErrClosed
	}
	if !c.pendingPSC {
		return nil
	}
	if e := c.cleanupPSCInvocation(); e != nil {
		return e
	}
	c.pendingPSC = false
	return nil
}

// PSCChain memoizes the source ordered pair and variable after canonicalizing
// both inputs. A pre-publication failure leaves the source empty cache entry.
// Published arrays with unwritten elements remain unavailable for safe replay.
func (c *PolynomialCache) PSCChain(p, q Polynomial, x uint32, out *PolynomialRefVector) error {
	if e := c.ready(); e != nil {
		return e
	}
	var e error
	p, e = c.MkUnique(p)
	if e != nil {
		return e
	}
	q, e = c.MkUnique(q)
	if e != nil {
		return e
	}
	key := polynomialPSCKey{p: p, q: q, x: x}
	if entry, ok := c.psc[key]; ok {
		// Preserve the source's size0 hit before result-array publication.
		if !entry.complete && len(entry.result) != 0 {
			return ErrPolynomialPending
		}
		if out == nil || out.owner != c.manager.core {
			return ErrPolynomialOwner
		}
		if e := out.Reset(); e != nil {
			return e
		}
		for _, value := range entry.result {
			if e := out.Push(value); e != nil {
				return e
			}
		}
		return nil
	}
	entry := &polynomialPSCEntry{}
	c.psc[key] = entry
	c.pscCall = polynomialPSCInvocationReceipt{}
	if e := c.manager.pscChainWithReceipt(p, q, x, out, &c.pscCall); e != nil {
		return c.finishPSC(e)
	}
	n, e := out.Size()
	if e != nil {
		return c.finishPSC(e)
	}
	entry.result = make([]Polynomial, n)
	for i := uint32(0); i < n; i++ {
		value, e := out.Get(i)
		if e != nil {
			return c.finishPSC(e)
		}
		value, e = c.MkUnique(value)
		if e != nil {
			return c.finishPSC(e)
		}
		if e := out.setWithReceipt(i, value, nil); e != nil {
			return c.finishPSC(e)
		}
		entry.result[i] = value
	}
	entry.complete = true
	return nil
}

func (c *PolynomialCache) ready() error {
	if c == nil || c.self != c {
		return ErrPolynomialOwner
	}
	if c.closed || c.closing {
		return arena.ErrClosed
	}
	if c.resetting {
		return ErrPolynomialPending
	}
	if c.pendingFactors != nil {
		return ErrPolynomialPending
	}
	if c.pendingPSC {
		return ErrPolynomialPending
	}
	return c.manager.ready()
}

// MkUnique returns the first equal representative inserted since Reset. The
// returned handle is borrowed; the cache's reference keeps it alive.
func (c *PolynomialCache) MkUnique(p Polynomial) (Polynomial, error) {
	if e := c.ready(); e != nil {
		return Polynomial{}, e
	}
	info, e := c.manager.core.Info(p)
	if e != nil {
		return Polynomial{}, e
	}
	if c.ids[info.ID] {
		return p, nil
	}
	h, e := c.manager.Hash(p)
	if e != nil {
		return Polynomial{}, e
	}
	for _, old := range c.buckets[h] {
		equal, e := c.manager.core.Equal(p, old)
		if e != nil {
			return Polynomial{}, e
		}
		if equal {
			return old, nil
		}
	}
	if e := c.refs.push(c.manager.core, p); e != nil {
		return Polynomial{}, e
	}
	c.buckets[h] = append(c.buckets[h], p)
	c.ids[info.ID] = true
	return p, nil
}

// Reset releases cached representatives and permits new first representatives.
// CleanupReset completes a stopped owning release before further cache use.
func (c *PolynomialCache) Reset() error {
	if e := c.ready(); e != nil {
		return e
	}
	clear(c.psc)
	clear(c.factors)
	if e := c.refs.reset(c.manager.core); e != nil {
		c.resetting = true
		return e
	}
	clear(c.buckets)
	clear(c.ids)
	c.releaseImp(232)
	if e := c.newImp(233); e != nil {
		c.resetting = true
		return e
	}
	return nil
}

func (c *PolynomialCache) CleanupReset() error {
	if c == nil || c.self != c {
		return ErrPolynomialOwner
	}
	if c.closed {
		return arena.ErrClosed
	}
	if !c.resetting {
		return nil
	}
	if e := c.refs.resetMode(c.manager.core, true); e != nil {
		return e
	}
	clear(c.buckets)
	clear(c.ids)
	clear(c.psc)
	clear(c.factors)
	c.releaseImp(232)
	if c.imp == nil {
		if e := c.newImp(233); e != nil {
			return e
		}
	}
	c.resetting = false
	return nil
}

func (c *PolynomialCache) Close() error {
	if c == nil || c.self != c {
		return ErrPolynomialOwner
	}
	if c.closed {
		return nil
	}
	c.closing = true
	if e := c.CleanupPSC(); e != nil {
		return e
	}
	if e := c.CleanupFactor(); e != nil {
		return e
	}
	if e := c.refs.close(c.manager.core); e != nil {
		return e
	}
	c.releaseImp(211)
	c.manager.core.integer.Budget().Release(c.header)
	c.header = 0
	c.buckets = nil
	c.ids = nil
	c.psc = nil
	c.factors = nil
	c.closed = true
	return nil
}

// A refused cache call conveys no authority over the manager's public root.
func (c *PolynomialCache) cleanupFactorInvocation() error {
	if e := c.manager.cleanupFactorCoreInvocation(&c.factorCall, false); e != nil {
		return e
	}
	return c.manager.cleanupFactorCoreInvocation(&c.factorCall, true)
}

// Compatibility cleanup is selected only after the cache's concrete accepted
// owner/frame/nonce matches. A refused or retired tuple cannot select any work.
func (c *PolynomialCache) cleanupPSCInvocation() error {
	core := c.manager.core
	if !core.matchesPSCInvocation(&c.pscCall) {
		return nil
	}
	generation := c.pscCall.frame.sourceGeneration
	if e := core.cleanupPSCCompleted(generation); e != nil {
		return e
	}
	return core.cleanupPSCAborts(generation)
}
