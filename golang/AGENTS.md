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

Write project files and build/cache/temp artifacts only below this golang
subtree. Port source assertions to Go tests before translating the corresponding
implementation, and retain independent bounded correctness checks. After each
coherent green chunk, make a git commit with a pithy one-to-three-sentence
message. Git metadata writes necessary for those requested commits are allowed.
The user requests frequent small commits, but the latest instruction forbids
leaving the suite RED across implementation commits. Keep tests-first RED in
the working tree until the corresponding implementation and full native and
checked/debug suites are GREEN. Do not hide failing tests with tags/skips or
weaken their assertions. Existing genuinely missing-API pending campaigns
remain explicit completion obligations, with commands and removal tracked in
PORT_PROGRESS.md.

Maintain ALL narrative port status, TODO/FINISHED lists, source maps, failure
evidence and handoff in PORT_PROGRESS.md. Do not create separate PACKET, STATUS,
CHECKPOINT or progress documents. Keep machine-readable inventories synchronized
with justified evidence. A documentation-only audited consolidation commit is
explicitly required before resuming the paused implementation work.

## Current verification scope

The user requests faster porting while the implementation uses a single
goroutine. Defer JavaScript/WASM and WASI builds or execution, and race checks,
at this stage. Do not include them as gates for current port chunks. Use native
tests, checked/debug tests, appropriate static checks, and the source/evidence
and no-generics audit. Preserve the plan's portable architecture; deferred
platform and concurrency verification remains a later completion obligation.

## Green commit requirement

The user's latest instruction supersedes the earlier permission for RED
checkpoints: do not leave the test suite failing across commits. Keep tests-first
RED observations in the working tree, implement the corresponding fix, and
require passing full native and checked/debug suites before committing. Preserve
meaningful assertions; do not hide existing failures with tags or skips. Continue
committing small coherent changes frequently once these checks are green.
