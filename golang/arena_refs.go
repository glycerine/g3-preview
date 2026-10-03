package g3

import "github.com/glycerine/g3/golang/arena"

// These are distinct internal ownership domains, not interchangeable IDs.
// Source AST IDs and Boolean literals are separate from storage references.
type astRef rawRef
type enodeRef rawRef
type satClauseRef rawRef
type smtClauseRef rawRef
type nlsatClauseRef rawRef
type justRef rawRef
type vecRef rawRef
type dependencyRef rawRef

// Public token bits always retain a generation. Public wrappers must also carry
// their owning context and validate closure/domain/kind before entering core.
// Wrappers are introduced with their real domain manager, not as API stubs.
func makeToken(slot arena.Slot, generation arena.Generation) uint64 {
	return uint64(generation)<<32 | uint64(slot)
}
func validToken[T any](s *arena.Slab[T], token uint64) bool {
	return s.IsLive(arena.Slot(uint32(token)), arena.Generation(token>>32))
}
