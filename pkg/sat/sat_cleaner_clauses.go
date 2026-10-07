// Source: src/sat/sat_cleaner.cpp:87–166.
package sat

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func (c *satCleaner) clauseVector() *[]StoredClauseRef {
	if c.learned {
		return &c.lifecycle.learned
	}
	return &c.lifecycle.Original
}
func (c *satCleaner) cleanupClauses(learned bool) error {
	if e := c.open(); e != nil {
		return e
	}
	l := c.lifecycle
	if l.cleanerContinuation != nil {
		return ErrSATOwner
	}
	c.learned = learned
	vec := c.clauseVector()
	c.cursor = 0
	c.out = 0
	c.end = uint32(len(*vec))
	c.drainPrefix = 0
	c.drainSuffix = 0
	c.interrupted = true
	c.running = true
	defer func() { c.running = false }()
	l.cleanerContinuation = c
	a := l.propagation.clauses
	p := l.propagation
	v := p.variables
	for i := uint32(0); i < c.end; i++ {
		// The original scalar pointer is captured before source front compaction.
		r := (*vec)[i]
		sz, e := a.Size(r)
		if e != nil {
			return e
		}
		p.cleanerCounter = int32(uint32(p.cleanerCounter) + sz)
		j := uint32(0)
		satisfied := false
		for k := uint32(0); k < sz; k++ {
			lit, e := a.Literal(r, k)
			if e != nil {
				return e
			}
			value, e := v.valueLiteral(lit)
			if e != nil {
				return e
			}
			if value == util.LTrue {
				satisfied = true
				break
			}
			if value == util.LFalse {
				c.elimLiterals++
			} else {
				if e = a.SetLiteral(r, j, lit); e != nil {
					return e
				}
				j++
			}
		}
		if satisfied {
			c.elimClauses++
			if e = l.delClause(r); e != nil {
				return e
			}
			c.cursor = i + 1
			continue
		}
		if satStorageSourceDebug {
			f, e := a.Frozen(r)
			if e != nil {
				return e
			}
			if !f && j < 2 {
				return ErrSATAssertion
			}
		}
		if j < 2 {
			if satStorageSourceDebug {
				f, e := a.Frozen(r)
				if e != nil {
					return e
				}
				if !f {
					return ErrSATAssertion
				}
			}
			if j == 0 {
				p.setConflict(satJustification{}, DefaultLiteral())
			} else {
				lit, e := a.Literal(r, 0)
				if e != nil {
					return e
				}
				if e = p.assign(lit, satJustification{}); e != nil {
					return e
				}
			}
			if e = l.delClause(r); e != nil {
				return e
			}
			c.cursor = i + 1
		} else {
			if satStorageSourceDebug {
				for k := uint32(0); k < 2; k++ {
					lit, e := a.Literal(r, k)
					if e != nil {
						return e
					}
					x, e := v.valueLiteral(lit)
					if e != nil {
						return e
					}
					if x != util.LUndef {
						return ErrSATAssertion
					}
				}
			}
			if j == 2 {
				x, e := a.Literal(r, 0)
				if e != nil {
					return e
				}
				y, e := a.Literal(r, 1)
				if e != nil {
					return e
				}
				learn, e := a.Learned(r)
				if e != nil {
					return e
				}
				if e = l.mkBinClause(x, y, learn); e != nil {
					return e
				}
				if e = l.delClause(r); e != nil {
					return e
				}
				c.cursor = i + 1
			} else {
				if e = a.Shrink(r, j); e != nil {
					return e
				}
				(*vec)[c.out] = r
				c.out++
				c.cursor = i + 1
				f, e := a.Frozen(r)
				if e != nil {
					return e
				}
				if !f {
					if j == 3 {
						_, e = l.attachTerClause(r)
					} else {
						_, e = l.attachNaryClause(r)
					}
					if e != nil {
						return e
					}
				}
			}
		}
	}
	*vec = (*vec)[:c.out]
	c.interrupted = false
	l.cleanerContinuation = nil
	return nil
}

// Owning abort is separate from source set_end and never reruns clause cleaning.
// Destroy exactly front retained owners then unread original suffix, retaining
// each cursor on refusal. Dead/duplicated pointer slots are not owning roots.
func (c *satCleaner) cleanupInterrupted() error {
	if !c.interrupted {
		return nil
	}
	if c.lifecycle == nil || c.lifecycle.self != c.lifecycle {
		return ErrSATOwner
	}
	l := c.lifecycle
	if l.cleanerContinuation != c {
		return ErrSATOwner
	}
	vec := c.clauseVector()
	if uint32(len(*vec)) < c.end {
		return arena.ErrBounds
	}
	a := l.propagation.clauses
	for c.drainPrefix < c.out {
		if e := a.Delete((*vec)[c.drainPrefix]); e != nil {
			return e
		}
		c.drainPrefix++
	}
	if c.drainSuffix < c.cursor {
		c.drainSuffix = c.cursor
	}
	for c.drainSuffix < c.end {
		if e := a.Delete((*vec)[c.drainSuffix]); e != nil {
			return e
		}
		c.drainSuffix++
	}
	*vec = (*vec)[:0]
	c.interrupted = false
	l.cleanerContinuation = nil
	return nil
}
