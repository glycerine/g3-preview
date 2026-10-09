// Source: src/math/polynomial/polynomial.cpp:2104-2119; src/util/vector.h.
package polynomial

import (
	"github.com/glycerine/g3/pkg/util"
	"sort"
)

// std::sort contributes its descending graded-lex order, not library internals.
func (b *PolynomialSOMBuffer) sortMonomials() error {
	var cause error
	sort.Slice(b.buffer.ms, func(i, j int) bool {
		if cause != nil {
			return false
		}
		var order int
		order, cause = b.storage.monomials.GradedLex(b.buffer.ms[i], b.buffer.ms[j])
		return cause == nil && order > 0
	})
	return cause
}

// Owning abort resumes only actual untransferred local fields, separately from
// the sticky source raw-destructor history. No source cleanup is retried here.
func (b *PolynomialSOMBuffer) cleanupSortNumerals() error {
	v := &b.sortNumerals
	if b.sortCleanupIndex < b.sortTransferred {
		b.sortCleanupIndex = b.sortTransferred
	}
	for b.sortCleanupIndex < v.initialized {
		a, e := v.coefficient(b.storage, b.sortCleanupIndex)
		if e != nil {
			return e
		}
		e = b.storage.integer.Del(&a)
		if w := v.putCoefficient(b.storage, b.sortCleanupIndex, a); w != nil {
			return w
		}
		if e != nil {
			return e
		}
		b.sortCleanupIndex++
	}
	v.initialized = 0
	return nil
}
func (b *PolynomialSOMBuffer) finishSortSource() error {
	if !b.sortActive {
		return nil
	}
	if b.sortSourceCause != nil {
		return b.sortSourceCause
	}
	v := &b.sortNumerals
	// Actual initialized published fields transfer before their raw destructor.
	// The source mpz element has no destructor; no Del is invented here.
	for b.sortTransferred < v.initialized {
		a, e := v.coefficient(b.storage, b.sortTransferred)
		if e != nil {
			b.sortSourceCause = e
			return e
		}
		if a != (util.Int{}) {
			if b.retiredNumeralCount >= uint32(len(b.retiredNumerals)) {
				return ErrPolynomialPending
			}
			b.retiredNumerals[b.retiredNumeralCount] = a
			b.retiredNumeralCount++
		}
		b.sortTransferred++
	}
	v.initialized = 0 // ownership moved; physical source size/header is unchanged.
	if v.raw != (util.Block{}) {
		if e := b.storage.memory.Deallocate(&v.raw); e != nil {
			b.sortSourceCause = e
			return e
		}
	}
	if v.pending == (util.Block{}) && v.old == (util.Block{}) && v.retiredActive == 0 {
		*v = polynomialVector{width: 12}
		b.sortActive = false
		b.sortTransferred = 0
	}
	return nil
}
func (b *PolynomialSOMBuffer) SortGradedLex() (result error) {
	if e := b.ready(); e != nil {
		return e
	}
	if b.buffer.owner == nil {
		return ErrPolynomialOwner
	}
	if e := b.sortMonomials(); e != nil {
		return e
	}
	// The default local is constructed after the sort, at the actual lexical
	// site. A previous stopped local is not silently repaired or reused.
	if b.sortActive {
		return ErrPolynomialPending
	}
	b.sortActive = true
	b.sortTransferred = 0
	b.sortCleanupIndex = 0
	b.sortOriginalCause = nil
	b.sortSourceCause = nil
	b.sortNumerals = polynomialVector{width: 12}
	defer func() {
		b.sortOriginalCause = result
		e := b.finishSortSource()
		if result == nil {
			result = e
		}
	}()
	n, e := b.buffer.ms.size(b.storage)
	if e != nil {
		return e
	}
	if n > ^uint32(0)/2 {
		return ErrPolynomialPending
	}
	if e = b.prepareRetirement(b.retiredNumeralCount + 2*n); e != nil {
		return e
	}
	for i := uint32(0); i < n; i++ {
		term, e := b.buffer.ms.monomial(b.storage, i)
		if e != nil {
			return e
		}
		pos, e := b.buffer.positions.Get(term)
		if e != nil {
			return e
		}
		if e = b.sortNumerals.appendCoefficient(b.storage, util.Int{}); e != nil {
			return e
		}
		old, e := b.buffer.as.coefficient(b.storage, pos)
		if e != nil {
			return e
		}
		local, e := b.sortNumerals.coefficient(b.storage, i)
		if e != nil {
			return e
		}
		if e = b.storage.modular.Swap(&local, &old); e != nil {
			return e
		}
		if e = b.sortNumerals.putCoefficient(b.storage, i, local); e != nil {
			return e
		}
		if e = b.buffer.as.putCoefficient(b.storage, pos, old); e != nil {
			return e
		}
		if e = b.buffer.positions.Reset(term); e != nil {
			return e
		}
		if e = b.buffer.positions.Set(term, i); e != nil {
			return e
		}
	}
	b.buffer.as, b.sortNumerals = b.sortNumerals, b.buffer.as
	return nil
}
