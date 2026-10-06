# g3 Go implementation rules

## No Go generics

The user explicitly requires no Go generics in this project. This overrides
parameterized Go types and functions suggested in GO_PORT_PLAN.md, including
its arena sketches. Apply this rule to production code, tests, tools, and
generated Go code. Use concrete typed implementations instead.
The user's reasons are performance and keeping code easy to read, debug,
and profile. Apply this rule throughout the continuing port goal.

Preserve pointer-free hot records, fixed-width handles, explicit ownership,
source algorithms, and test coverage when replacing generic infrastructure.
Do not substitute unsafe storage or per-record interface boxing for generics.
Shared scalar allocation metadata may be factored into ordinary concrete types.

Slabs, vectors, regions, handle-validation helpers, shuffle and their tests
use concrete types. Preserve this rule in later domain specializations.
Run the no-generics syntax audit through `go run ./cmd/portcheck`.

## Workflow

Keep active implementation work visible. Use the main working tree or visible
Git worktrees under `golang/worktrees/`; do not develop implementation in
`.cache`. Keep new clean verification copies visible under `golang/worktrees/`
as well. Reserve `.cache` for build caches, logs, and machine-readable evidence;
historical immutable verification copies may remain there as evidence. Keep
existing unfinished work visible until it can be integrated into a green commit.
Do not start or continue parallel agent work: the user now requires serial work.

Write project files and build/cache/temp artifacts only below this golang
subtree. Port source assertions to Go tests before translating the corresponding
implementation, and retain independent bounded correctness checks. After each
coherent green chunk, make a git commit with a pithy one-to-three-sentence
message. Git metadata writes necessary for those requested commits are allowed.
The user requests frequent small commits, but the latest instruction forbids
leaving the suite RED across implementation commits. Keep tests-first RED in
the working tree until the corresponding implementation and required current
verification checks are GREEN. Do not hide failing tests with tags/skips or
weaken their assertions. Existing genuinely missing-API pending campaigns
remain explicit completion obligations, with commands and removal tracked in
PORT_PROGRESS.md.

Maintain ALL narrative port status, TODO/FINISHED lists, source maps, failure
evidence and handoff in PORT_PROGRESS.md. Do not create separate PACKET, STATUS,
CHECKPOINT or progress documents. Keep machine-readable inventories synchronized
with justified evidence. A documentation-only audited consolidation commit is
explicitly required before resuming the paused implementation work.

## Current plan: serial end-to-end API delivery

The user's latest instruction supersedes the earlier parallel-agent workflow
and mandatory large checked/debug failure sweeps. Work serially; do not spawn,
resume, or delegate work to sub-agents. Stop further expansion of unrelated
numeric, printing, or allocation-lifetime lanes. Preserve their existing work
and evidence; integrate only what the next solver API milestone actually needs.

The next milestone is a usable Go path through context creation, Boolean AST
construction, solver creation, assertion, satisfiability checking, and model
retrieval. Port the original context/solver/factory/SAT/SMT algorithms needed
for this path. Establish source-derived end-to-end tests for SAT with a checked
model and UNSAT before implementation. Preserve the source's distinct UNKNOWN
result and error behavior; do not substitute a toy solver, canned results, or
composed algorithms for the original implementation.

First inspect existing dependency coverage and choose the next missing API
operation on that path. Finish each needed dependency only to the extent
required for correct source behavior on the milestone. Do not make exhaustive
allocation-history instrumentation or unrelated arithmetic helpers prerequisites
for usable API progress. Then expand supported logic and public API coverage
along working end-to-end paths. The complete patched solver remains the final
objective; this milestone changes implementation order, not final scope.

## Current verification scope

Do not launch or continue large debug allocation-failure/refinement sweeps.
Remove them from current integration and commit gates. Preserve their existing
tests, original precision, horizons and recorded evidence for later use; do not
truncate or weaken them to manufacture a passing result. No full combined
checked/debug sweep is required for current commits. Use focused ordinary
unit/integration tests for the changed behavior, bounded independent correctness
checks, and targeted checked/debug tests only when they are small and directly
relevant. Run appropriate vet and source/no-generics checks. Avoid repeated
verification copies and duplicate broad runs without a concrete reason.

Continue deferring JavaScript/WASM, WASI, and race verification at this stage.
Preserve the portable architecture and record deferred verification obligations.

## Green commit requirement

Do not commit known failing implementation changes. Port source tests first,
close their failures, and run the normal uncached native suite with
`CGO_ENABLED=0 go test -count=1 ./... -timeout=30m` before an implementation
commit. The large checked/debug sweeps are explicitly deferred by the latest
user instruction and are not a current commit gate. Do not hide failures in
ordinary required tests with tags, skips, or weaker assertions. Commit small
coherent API/dependency changes frequently with a pithy one-to-three-sentence
message; do not amend or rebase.

Keep fixtures self-contained and compiler caches outside verification sources.
Use a clean module verification copy when fixture isolation actually needs it,
rather than making new copies for every scoped check. Files and artifacts stay
below golang/, and tests must not depend on an existing ignored cache directory.

## Fixed source provenance

Keep originating repository-relative source paths in ported files. The C++ reference is fixed: do not compute, store, or verify its content hashes, checksums, or fingerprints. Omit Microsoft copyright lines from current and new ported headers; retain license notices and `LICENSE.txt`, and preserve other authors’ notices. This policy does not change solver algorithm hashing/fingerprints or Go working-tree SHA verification.

## Pure Go port

The port, tests, tools, and fixture generators must use Go: no C/C++ sources,
cgo, or external C compiler dependency under golang. Keep recorded JSON test
expectations and originating reference paths; the fixed original C++ source
outside golang remains read-only reference material. Preserve source-compatible
algorithms implemented in Go, including the shared random stream. Run current
verification gates with CGO_ENABLED=0.
