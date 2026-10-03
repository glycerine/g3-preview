# Standalone SAT clause storage: coordinator source review

This records the original coordinator source review. Its standalone storage
packet is now implemented and verified in SAT_STORAGE_PORT_STATUS.md; full SAT
solver functionality remains open.
The coordinator read all of sat_clause.h/.cpp, sat_clause_set.h/.cpp,
sat_watched.h/.cpp, sat_justification.h and sat_var_queue.h in the preserved
source. The complete solver, propagation, simplification, search, extension
callbacks and source solver tests remain required by GO_PORT_PLAN.md.

The packet should implement the actual clause allocator and temporary clauses,
clause wrappers and sets, watched records and owning watch lists, justification
records, and the activity-backed variable queue. All hot records must be concrete
and pointer-free. Clause IDs are separate from physical handles and may recycle
only at the source recycle site. No Go generics, unsafe or boxed hot records.

## Source details that tests must freeze before translation

- Clause construction copies all literals, sets capacity equal to initial size,
  initializes removed/used/frozen/reinit/inactivity/glue/psm, and calls
  mark_strengthened, which recomputes the variable approximation mask. Its source
  size assertion is greater than one. Shrink retains capacity and recomputes
  approximation only when size changes. Elim removes the first matching literal
  and shifts later elements left; it does not swap-remove.
- check_approx is not a read-only query: it saves the previous approximation,
  recomputes the real mask, then source-debug asserts may_eq. Preserve this
  mutation and distinguish release from source-debug execution.
- Inactivity is an eight-bit field and wraps. Glue and psm saturate at255.
  Freeze/unfreeze assert learned status and prior frozen status. Models retain
  true/false/undefined; an undefined literal does not satisfy a clause.
- Temporary clause Set destroys the previous clause before a larger allocation.
  Reusing existing capacity updates only size, learned and literal bytes. It does
  **not** call mark_strengthened or update_approx and does not reset other flags.
  Preserve stale approximation/flags on reuse; do not silently reconstruct the
  object on every Set. New temporary clauses use source ID UINT_MAX.
- bin_clause stores the second literal as `(index << 1) + learned`. Its equality
  compares the two encoded words directly in both orientations, including the
  unusual reverse comparison between unshifted and shifted fields. A conventional
  unordered pair equality would change defined source behavior.
- The source allocator allocates storage before obtaining a recyclable source
  ID and constructing the clause. Delete recycles the ID before physical release.
  Address-segment encoding is a platform storage contract, not a solver algorithm:
  replace it with explicit stable owner-local offsets/handles and record widths,
  provenance, reuse generations and limits. Do not reproduce pointer arithmetic
  or claim source address-segment exhaustion tests apply to the Go layout.
- Clause-set insert reserves its ID-to-position vector before checking an existing
  ID, publishes the position before pushing the borrowed clause, and has a source
  conditional invariant check. Erase uses swap-remove and updates the moved ID.
  Pop removes the last entry. Sets borrow clauses; they do not own clause cleanup.
  A post-position-publication push failure needs an explicit defined continuation
  adapter, not a hidden full-operation rollback claim.
- Watched ternary construction sorts literal indices and asserts they differ.
  Binary learned metadata occupies bit2; clause records pack the blocked literal
  above two tag bits. Preserve source width truncation versus debug round-trip
  assertions and expose typed guards only where source continuation is undefined.
- **Source watched external-index defect:** constructor puts the supplied index
  in m_val1 and EXT_CONSTRAINT(3) in m_val2, but get_ext_constraint_idx returns
  m_val2. Thus the source constructor round-trip assertion fails for indices other
  than3 in source-debug, and release access/display returns3. Do not silently fix
  this defect. Justification's external accessor instead correctly returns m_val1.
- watched_lt only orders binary before other records and ternary before clause/
  external; it supplies no ordering within those groups. Clause-watch erase removes
  the first matching offset and shifts subsequent records left. Ternary erase
  follows the actual source vector erase action, which must be reread.
- Justification uses NONE/BINARY/TERNARY/CLAUSE/EXT tags in the low three bits of
  its second word. Ternary second-literal shift is3, distinct from watched shift2.
  get_clause_offset/get_ext_justification_idx lack kind assertions in this source;
  do not fabricate a universal source getter assertion inventory.
- Variable-queue priority is descending activity with no extra variable-ID tie
  breaker. It uses the source heap with initial reserve128. MkVar reserves v+1
  before insert, increased activity performs decreased only if present, unassign
  inserts only if absent, deletion erases only if present, and reset retains heap
  storage. Activity must be a real explicitly borrowed concrete uint32 domain,
  not a fake solver callback or an uncharged copied snapshot.

## Ownership and verification contract to refine before implementation

Audit the source heap and all vector operations and identify actual test
registrations and consumers before approving API/storage details. Use actual
charged cold owners and pages/backings, with old/new overlap and bounded typed
slot/offset/generation adapters. Copy/move/reset/finalize and all implicit source
member actions require explicit concrete counterparts. Temporary clauses,
wrappers/sets/watch lists and clause-vector rendering must use the same real
clause owner; no parallel simplified clause representation.

Author source assertions and the defect fixtures first and observe missing-real-API
RED. Add independent bounded literal-array, model and membership/heap oracles,
source ordering fixtures and immediate ownership/failure-prefix sweeps. Source
tests that require the real solver remain open until that producer exists; a
test-only SAT oracle cannot stand in for source solver execution. Preserve exact
default display bytes while declaring stream-state integrations still open if
they are not translated. Run frozen full native/checked/debug, race, actual JS,
vet, WASI compilation and source/no-generics gates before the coherent commit.
