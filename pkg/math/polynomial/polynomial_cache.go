// Source: src/math/polynomial/polynomial_cache.cpp:83-92,129-139,203-227.
package polynomial

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
)

// PolynomialCache owns canonical representatives used by the retained NLSAT
// cache. PSC-chain and factor result memoization are separate remaining paths.
type PolynomialCache struct {
	self      *PolynomialCache
	manager   *MultivariatePolynomialManager
	refs      polynomialReferenceVector
	buckets   map[uint32][]Polynomial
	ids       map[uint32]bool
	header    uint64
	resetting bool
	closing   bool
	closed    bool
}

func NewPolynomialCache(m *MultivariatePolynomialManager) (*PolynomialCache, error) {
	if e := m.ready(); e != nil {
		return nil, e
	}
	header := uint64(reflect.TypeOf(PolynomialCache{}).Size())
	if e := m.core.integer.Budget().Reserve(header); e != nil {
		return nil, e
	}
	c := &PolynomialCache{manager: m, header: header, buckets: make(map[uint32][]Polynomial), ids: make(map[uint32]bool)}
	c.self = c
	return c, nil
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
	if e := c.refs.reset(c.manager.core); e != nil {
		c.resetting = true
		return e
	}
	clear(c.buckets)
	clear(c.ids)
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
	if e := c.refs.close(c.manager.core); e != nil {
		return e
	}
	c.manager.core.integer.Budget().Release(c.header)
	c.header = 0
	c.buckets = nil
	c.ids = nil
	c.closed = true
	return nil
}
