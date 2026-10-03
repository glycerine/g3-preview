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
