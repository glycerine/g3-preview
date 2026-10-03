# Work checkpoints

Save tests-first work in smaller commits instead of leaving it untracked while
an entire source family is completed. A checkpoint preserves work; it does not
claim that an unfinished source unit is ported or that all tests are green.
Keep ordinary green implementation commits and source-ledger promotions subject
to their existing evidence requirements.

The allocator prerequisites (capacity-only physical cells, inline16 limbs and
inline1024 characters) have nine passing native/debug tests, vet and source
checks. Their logs are `.cache/mpz-source-backing-nine-{native,debug,vet,source}.log`.
Mandatory Integer/Natural/output integration remains unfinished, as described in
`MPZ_BACKEND_IMPLEMENTATION_STATUS.md`.

Two tests-first campaigns need APIs that are not implemented yet. Explicit tags
preserve their actual RED state without breaking ordinary package builds:

- `go test -tags port_pending_mpz ./mpnum` runs the mandatory allocator/owner
  integration campaign in `mpnum/mpz_allocator_phase_b_test.go`.
- `go test -tags port_pending_sat .` runs the finite registered SAT user-scope
  campaign in `sat_solver_user_scope_test.go`.

These campaigns must become ordinary tests when their implementations land.
They remain required work; a default green run does not cover them. Proposal
packets retain their historical review context and are not current completion
claims. Source coverage and the full solver goal remain unchanged.

The factorization checkpoint contains the source factors owner, finite-field
factorization, Hensel lifting, degree/subset search and integer recombination.
The registered 37 numeric recipes exercise 74 source numeric ENSURE executions;
37 extra dense qualification checks do not port the missing multivariate
construction/assertion. Earlier family runs and the corrected independent-core
Hensel test pass. Cold-binding refusal, additional alias/lifetime coverage and
allocator-backed cleanup-refusal retention remain unfinished review items.
No whole factorization source unit or multivariate solver completion is claimed.

The integrated factorization/helper tree passes `go test ./...`,
`go test -tags g3debug ./...` and `go vet ./...`; logs are
`.cache/checkpoint-{native,debug,vet}.log`. The explicitly tagged pending
campaigns still fail compilation at their missing APIs, as expected; logs are
`.cache/checkpoint-pending-{mpz,sat}-red.log`.

Review found two concrete factorization corrections at the checkpoint: Berlekamp
pivot vectors currently allocate exact size instead of source geometric resize;
and a factors owner whose Close has finished Clear but failed constant cleanup
must reject ordinary mutation before retrying Close. The existing backend cannot
yet inject that leaf cleanup refusal. These issues and deeper coverage are next
work, not reasons to leave the implemented algorithms untracked.

The Berlekamp pivot correction now follows source geometric resize, including
retained capacity with logical size still zero after an interrupted expansion.
The finite empty-input release path appends its empty factor; checked/debug
rejects at the source entry assertion. Tests-first missing-helper and behavior
RED preceded translation. Full native/debug suites, vet and source/evidence/
no-generics audit pass in the exact four-file overlay on 5b414f9; logs are
`.cache/pivot-parent-{native,debug,vet,source}.log`. The factors closing guard
and genuine allocator-backed cleanup-refusal coverage remain separate work.


`mpnum/mpz_backend_constructor_test.go` now also uses `port_pending_mpz` while
mandatory physical-cell wiring is in progress. Before tagging, its isolated
API-absence RED was observed on 5b414f9 with only this test overlay (log
`.cache/mpz-backend-constructor-isolated-red.log`). It specifies the six actual
source cell objects, constructor swap/order, initialized physical words and
default Runtime ownership. Remove this test's tag as soon as those constructor
APIs land; the larger transitive owner-integration campaign remains separately
unfinished. The pending tag preserves this work without including unrelated
allocator implementation drafts in a green checkpoint.


Factors owning Close now freezes ordinary operations before Clear starts, and
uses cleanup-only validation plus its completed-Clear cursor when retried.
The explicit partial-Clear/member-destruction protocol fixtures use real
constructed numeric/polynomial owners and callbacks that reject unexpected
resource calls. They do not fabricate a numeric Del refusal. Tests first
observed ordinary mutation allowed in the old completed-Clear state; native/
debug protocol and real utility-bound lifecycle tests pass after the correction.
The exact two Go files also pass full native/debug suites, vet and source/
evidence/no-generics audit on 0c9d695; logs are
`.cache/factors-close-parent-{native,debug,vet,source}.log`. Actual leaf cleanup
refusal and retained lexical factorization contexts remain backend integration
obligations. The independent allocator draft was excluded from these gates.
