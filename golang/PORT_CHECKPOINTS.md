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

Review found two concrete factorization corrections still required: Berlekamp
pivot vectors currently allocate exact size instead of source geometric resize;
and a factors owner whose Close has finished Clear but failed constant cleanup
must reject ordinary mutation before retrying Close. The existing backend cannot
yet inject that leaf cleanup refusal. These issues and deeper coverage are next
work, not reasons to leave the implemented algorithms untracked.
