// Licensed under the MIT license; see LICENSE.txt.
// Source: src/util/mpz.cpp:1385-1410; src/util/mpn.cpp:371-399.
// Source author: Leonardo de Moura.
package mpnum

import "github.com/glycerine/g3/golang/arena"

// Concrete cold caller receipt. Tuples survive retirement; flags grant no
// authority without exact service, frame and nonwrapping source invocation.
type NumeralInsertionReceipt struct {
	service                                             *RationalService
	natural                                             *NaturalManager
	character                                           *integerCharacterScope
	formatting                                          *naturalFormattingScope
	invocation, formattingInvocation, displayInvocation uint64
	characterOwned, characterCompleted, formattingOwned bool
	active, sourceDone, retired                         bool
	bodyCause, sourceCause, ownCause                    error
}

func (r *NumeralInsertionReceipt) Active() bool { return r != nil && r.active }
func (s *RationalService) CompleteNumeralInsertion(r *NumeralInsertionReceipt, owning bool) error {
	if e := s.lock(); e != nil {
		return e
	}
	defer s.mu.Unlock()
	return completeNumeralInsertion(r, s.manager.z.natural, s, owning)
}
func completeNumeralInsertion(r *NumeralInsertionReceipt, natural *NaturalManager, service *RationalService, owning bool) error {
	if r == nil || !r.active || r.service != service || r.natural != natural || !natural.matchesIntegerCharacterScope(r.character) || r.invocation == 0 || (r.characterOwned && r.invocation != r.character.invocation) || r.displayInvocation != r.character.displayInvocation {
		return arena.ErrInvalidSlot
	}
	n := r.natural
	if r.character.callExecuting {
		return arena.ErrInvalidSlot
	}
	if r.formattingOwned && r.formatting != nil && (r.formatting != r.character.formatting || r.formattingInvocation != r.formatting.invocation) {
		return arena.ErrInvalidSlot
	}
	stop := func(e error) error {
		if e != nil && r.ownCause == nil {
			r.ownCause = e
		}
		return e
	}
	if !owning {
		if r.sourceDone {
			return nil
		}
		if r.formattingOwned && r.formatting != nil {
			if e := n.unwindFormattingScope(r.formatting); e != nil {
				return stop(e)
			}
		}
		if r.characterOwned && r.characterCompleted {
			if e := n.destroyIntegerCharacterScope(r.character); e != nil {
				return stop(e)
			}
			r.characterOwned = false
		}
		r.sourceDone = true
		return nil
	}
	if !r.sourceDone {
		return arena.ErrInvalidSlot
	}
	if r.formattingOwned && r.formatting != nil {
		if e := n.cleanupFormattingScope(r.formatting); e != nil {
			return stop(e)
		}
	}
	if r.characterOwned && r.character.pending {
		if e := r.character.buffer.finalize(n.runtime); e != nil {
			return stop(e)
		}
		r.character.reset()
	}
	r.characterOwned = false
	r.formattingOwned = false
	r.active = false
	r.retired = true
	return nil
}
