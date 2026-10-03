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

## Current verification scope

The user requests faster porting while the implementation uses a single
goroutine. Defer JavaScript/WASM and WASI builds or execution, and race checks,
at this stage. Do not include them as gates for current port chunks. Use native
tests, checked/debug tests, appropriate static checks, and the source/evidence
and no-generics audit. Preserve the plan's portable architecture; deferred
platform and concurrency verification remains a later completion obligation.
