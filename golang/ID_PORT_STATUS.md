# Source ID generator prerequisite

`util_id_gen.go` translates `src/util/id_gen.h`: constructor, mk/recycle,
reset/cleanup, show_hash, get_id_range, set_next_id, and free-ID display. IDs stay
uint32 in both handle builds and are distinct from arena slots/generations.
Source UINT_MAX wraparound and duplicate recycle behavior are preserved;
this low-level generator does not invent globally unique/permanent IDs.

The owner controls the source memory-service check through one cold callback;
recycle skips enqueueing while that service reports out-of-memory. The full
process/owner resource service remains unported. Free-ID backing arrays are owned,
reset retains capacity and cleanup drops it. The generator must not be shallow
copied. Budgeted construction now charges each actual free-list backing array;
the embedded generator header is charged once by its enclosing owner. The
standalone constructor retains a nil/unlimited budget.

There is no dedicated source id_gen assertion suite. Finite added tests check
known allocation/recycle sequences, resource skip, debug skip over free IDs,
range, reset/cleanup, source wraparound, duplicate behavior, display and explicit
little-endian byte representation of hash input. A 3,000-operation independent
live-ID set tests uniqueness when the caller obeys source ownership. A warmed
mk/recycle loop allocates zero Go objects. Hash ordering follows the established
utilhash baseline; the empty-list hash avoids the source's invalid address-of-
element-zero expression without changing the result. Oversized hash byte length
is explicitly rejected instead of truncating to uint32.

Tests first failed with undefined constructor, then passed after translation in
release and checked builds. No test reads reference files or runs C++.
Source AST register/delete/compress call sites were read for integration context
but remain unported; this primitive does not implement an AST manager.

## Budgeted reclamation workspace

Four new groups first failed to compile for the missing budgeted API, then passed
after implementation. Source vector capacities are 2,3,5,8,12,...; each growth
reserves the entire new uint32 array while the old array remains charged, copies
logical free IDs, publishes the new array, then refunds the old one. The exact
8+12 byte overlap requires a limit of20;19 rejects it before logical mutation.
Reset retains the charged capacity; repeated cleanup refunds it and leaves the
helper reusable. The source OOM callback runs before reservation or enqueue.

reserveFreeIDs is an explicit Go ownership adapter: the future AST owner can
prepare fallible reclamation storage before publishing nodes. It changes no
logical IDs and retains any successful physical growth prefix on failure.
Recycle returns allocation errors; enclosing owners must handle them or preflight
sufficient capacity. Source uint32 wrap, duplicate IDs, LIFO order and hashes
retain their existing behavior. Physical array sizes are checked against both
uint32 and host-int bounds before allocation. One hundred warmed batches of64
live IDs recycle within preallocated capacity with zero host allocations and
no budget reservation, even with the next reservation forced to fail.

New tests: TestIDGeneratorBudgetSourceGrowthAndReset,
TestIDGeneratorBudgetExactOverlapAndFailedPublication,
TestIDGeneratorBudgetReservationPrefixesAndOOMSkip, and
TestIDGeneratorBudgetWarmEmbeddedValueAndCapacityBounds. Together with the four
existing suites these cover the helper, not actual AST registration/deletion or
the complete resource service. Diagnostic free-ID rendering still uses host
formatting and remains a cold unbudgeted diagnostic path; no claim is made that
it is an AST-owned printer implementation.

## Coordinator integration gate

The prospective integrated Go tree in `.cache/release/id-chain-integrated-final`
contains the frozen string-buffer, integer facade, budgeted ID and chain-table
packets together. Full fresh CGO-disabled native and g3checked+g3debug modules
passed (mpnum9.631s/9.742s). All25 named ID/chain suites passed fresh checked-debug
race and actual JS execution, recorded in `.cache/id-chain-integrated-race.log`
and `.cache/id-chain-integrated-js.log`. Full checked-debug vet and WASI package
compilation passed. WASI compilation is not runtime evidence. Coordinator
verified every frozen production/test SHA and each named RUN/PASS registration.
