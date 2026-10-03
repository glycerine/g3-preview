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


Imported-factor Hensel lifting now adopts newly created prime-core A/B backing
at the actual lifted-core reads. A fourth donor store exposed a genuine owner
error before this correction. Isolated HEAD083b174 plus only the two Hensel
files passes seven Hensel groups in native/debug profiles. The strengthened
immediate-root sweep passes 151 release and 155 debug refusal ordinals plus
success, including the actual-read binding reservation at ordinal 78/80.
Logs: `.cache/factors-owner/hensel-read-binding-{native,debug}.log`.
The separate unfinished physical-cell backend was excluded from these checks;
its existing root-audit migrations remain required. No source-unit promotion.


Unfinished mandatory physical-cell backend checkpoint: the constructor test's
`port_pending_mpz` tag is removed because its actual APIs now execute. Numeric
cells use real SmallAllocator objects; four genuine Natural Words buffers
remain until the next integration step. Nine selected native/debug groups,
module vet and source/no-generics audit pass on the frozen 11-file snapshot
`.cache/release/mpz-cell-checkpoint-0c9d695`. Hash inventory:
`.cache/mpz-cell-checkpoint-sha256.txt`.

Broader 97-group runs remain RED; logs are
`.cache/mpz-cell-checkpoint-97-{native,debug}.log`. Held fixtures still assume
cell-backed Words, universal reservation refusal, or allocator-initialized size.
These failures remain visible and require individual source-based correction.
SourceFinalize, Natural inline storage, output lifetime, transitive cleanup,
shared adoption and global runtime bindings remain unfinished. The separate
`port_pending_mpz` phase-B campaign is still pending. This commit preserves
implementation progress and does not claim full-suite success or source
completion. Details: `MPZ_BACKEND_IMPLEMENTATION_STATUS.md`.


RationalService payload statistics now count actual numeric allocator objects,
retained pending-cell objects and genuine Natural buffers. Separate counters
identify live numeric objects, pending cell objects, retained descriptors and
Natural payloads; released descriptors are not live physical objects and flags
do not duplicate an object. Tests first observed a successful two-cell copy
absent from the old Words-only payload count. Two new tests pass native/debug,
including actual allocation before descriptor exhaustion and complete cleanup.
Vet and source/evidence/no-generics audit pass on the exact two-file overlay
`.cache/release/service-stats-8ade92d`. Logs:
`.cache/rational-service-stats-isolated-{native,debug}.log` and
`.cache/rational-service-backend-stats-{red,vet,source}.log`.

Native AST proof and S-expression ownership consumers also pass after this
accounting correction. Debug consumers reach separately held Memout101 category
expectations; the Params sweep reaches a separately held warm-pool no-reservation
case. Full native baseline remains RED in `.cache/post-cell-native.log`; this
statistics correction does not claim completion of backend integration.


Two physical-cell fixtures now match actual source initialization and wrapped
allocation. Normalize initialization explicitly publishes pointer and size;
the UINT_MAX scratch request distinguishes a rounded eight-byte replacement
from a retained four-byte object whose capacity write is out of bounds. Tests
verify old descriptor generation/object invalidation, initialized capacity,
uninitialized size, physical prefixes, actual span guards and complete owning
cleanup. Named native/debug RED preceded this correction. Eleven relevant
groups, vet and source/no-generics audit pass on the exact two-file overlay
`.cache/release/mpz-fixtures-final-8ade92d`; logs:
`.cache/mpz-fixtures-eleven-{native,debug}.log` and
`.cache/mpz-fixtures-final-{vet,source}.log`. No numerical algorithm changed;
remaining broader backend failures remain separate integration work.


Mandatory Natural inline-buffer tests are saved tests-first with temporary tag
`port_pending_mpn`. The actual missing-API RED was observed on HEAD95a8236 plus
only `mpnum/mpn_backend_test.go`; log `.cache/natural-missing-api-red.log`.
Run `go test -tags port_pending_mpn ./mpnum` to expose the pending API campaign.
Remove this tag as soon as RuntimeOwner, the actual four inline16 division
buffers and CloseChecked execute. These tests are not covered by ordinary
green checks while tagged. `NATURAL_BACKEND_NEXT_PACKET.md` records the full
source action order, conditional addback and post-output debug lexical scope;
formatting and transitive cleanup remain required alongside this integration.


AST proof, Params and S-expression numeric reservation sweeps now distinguish
cold metadata refusal from exact source physical Memout101. Params copying
fixtures use genuine direct-sized numerals for two previously entirely warm
pool routes, so every required sweep observes a real reservation boundary.
The S-expression audit independently enumerates persistent scratch roots and
Natural buffers, validates live slots/checked generations and retains the
public-root expectation when source free-before-allocation leaves a cache hole.
It does not treat arbitrary lost cells as acceptable. Native, debug and
checked/debug selected consumer groups pass; module vet and source/no-generics
audit pass on the exact four-file overlay
`.cache/release/numeric-consumers-8d9b20f`. Logs:
`.cache/numeric-consumer-isolated-{native,debug,checked-debug,vet,source}.log`.
The broader backend baseline remains unfinished and RED separately.


Imported Hensel exponent16 now has an independent descriptor/object/raw-block
census before fault disable or cleanup. The sweep covers 208 release and 228
debug refusals plus success, including genuine big numeric roots and one
pending physical object in each profile. Unpublished/cleanup-pending branches
are supported by the census but were not exercised by this campaign.

Debug ordinal157 leaves the actual lower-bound failed token. A focused direct-
allocator fixture proves ordinary factors Close remains guarded and retains
its closing owner. A separate explicit caller SetZ changes only the source mode
flag, preserves bounds/physical roots and permits cleanup retry. This caller
action is not credited as Hensel recovery. Native/debug tests, vet and source/
no-generics pass in `.cache/release/hensel-physical-8ade92d`; logs are
`.cache/factors-owner/hensel-physical-final-{native,debug}.log` and
`.cache/factors-owner/hensel-physical-{profile-final,vet,source}.log`.
The older Words-per-cell family audits remain RED pending their exact physical
root migration; source lexical vector ownership needs further qualification.
