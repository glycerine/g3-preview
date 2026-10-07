# Package reorganization audit

Audited 2026-10-07 at commit `fba0ad7` against parent `d959ab8` and the local C++ sources. Module: `pkg/go.mod`, `github.com/glycerine/g3/pkg`.

Follow-up scope: A1 is declined. A2–A5 are approved for repair. All attempted A6 source-header changes were reverted at the user's request; A6 and placement questions D1/D2 remain untouched.

Repair verification: A2–A5 are complete. Full uncached native tests (`go test -count=1 ./... -timeout=30m`), full native vet, focused checked/debug tests for containers, capacity bounds, proofs, rewriters and filter models, and `go run ./cmd/portcheck -source ..` passed. All pre-existing Go source headers were checked against the pre-task commit `c954fc0` and remain unchanged. The original audit evidence below is historical.

The initial audit changed no implementation or test files. At the user's subsequent request, an intentionally failing regression test was added for A1 and later removed when the finding was declined. `smtcore/` is deliberately an amalgam of C++ packages to avoid Go import cycles. Its mixed source origins are not placement defects.

## Findings at a glance

P1 means a production behavior or required verification blocker. P2 means a reproducible utility defect or lost audit capability. P3 means incorrect provenance or a placement decision needing clarification.

| ID | Priority | Finding |
| --- | --- | --- |
| A1 | P1 | Global parameter lookup depends on which packages the executable imports; test-only registrations conceal the failure. |
| A2 | P1 | Three generic declarations violate the no-generics rule and fail the port checker. |
| A3 | P2 | PointerVector charges dynamic element size but releases static element size. |
| A4 | P2 | Zero-width elements cause division-by-zero panics in the new container helpers. |
| A5 | P2 | The port manifest and progress ledger were removed rather than migrated. |
| A6 | P3 | Several source headers name C++ files that do not exist. |

Two additional placement questions appear below. They are not confirmed runtime bugs.

## A1. Production global parameter registration is incomplete unless consumers arrange imports

**Evidence:** `pkg/util/util_gparams_collector_registry.go:14-42` holds a process-wide registry populated by package initialization. `pkg/util/util_gparams_catalog_dispatch.go:23-25` requires `catalogCollectContextParams` before any other collector. Its production registration is in `pkg/smtcore/gparams_catalog_collectors.go:380-385`.

An executable importing `util` and `arena`, constructing a valid symbol manager, rational service, params context, catalog, and global params object, then calling `GetValueName("timeout")`, returns:

```text
arena: invalid configuration: parameter collector catalogCollectContextParams is not linked
```

Construction succeeds; the lookup fails. A second executable with the same construction and blank imports of all collector-owner packages succeeds. All dispatched collector names have production implementations; this is a linkage/bootstrap problem, not an absence of collector implementations.

`pkg/util/gparams_collector_source_fixtures_test.go:1505-1539` registers the collectors using duplicated test-only implementations. Consequently, the util tests run with a complete registry that an ordinary util consumer does not have. `pkg/doc.go` does not supply a module-wide bootstrap.

**C++ comparison:** `src/util/gparams.cpp:89-98` calls the statically linked generated `gparams_register_modules()` on first use. Available parameter metadata does not depend on incidental caller imports.

**Impact:** Production lookups and module enumeration can fail despite a green package test suite. Importing just one owner is insufficient because dispatch requires the complete collector set.

**Initialization clarification and regression test:** This reproducer does not depend on initialization order. Imported packages finish initialization before the test runs; collector owners outside the import graph never initialize. The root-package test `TestGlobalParamsTimeoutWithProductionImports` imports arena/util without util's test-only fixtures. Running `cd pkg && go test -count=1 -run '^TestGlobalParamsTimeoutWithProductionImports$' -v .` fails on the missing context collector, as intended. The earlier full-suite pass below predates this requested red test.

**Proposed fix:** Provide a deterministic composition/bootstrap entry point above the collector-owner packages, or inject a complete concrete collector table from that layer. Preserve the generated C++ dispatch order. Keep `util` below its clients so that it does not acquire cyclic imports. If standalone util catalogs require explicit composition, make that requirement part of the constructor/API contract and provide the complete production entry point. Do not silently omit unlinked modules.

**Validation after fixing:** Add an external production integration test or separate executable that exercises the actual entry point without the test-only registry. Verify global lookup, representative module lookups, full metadata enumeration, and failure/cleanup behavior. Retain independent metadata expectations; importing production collectors into the existing fixture test package directly would collide with its duplicate registrations.

[decision: do not fix; this is an artificial by product of only importing util, which no real program will ever do because that is insufficient to do anything of interest].

## A2. Generics violate the implementation rule and leave portcheck red

**Evidence:** New generic declarations are present at:

- `pkg/util/pointer_buffer.go:12`: `PointerBuffer[T any]`.
- `pkg/util/pointer_vector.go:9`: `PointerVector[T comparable]`.
- `pkg/math/polynomial/numeric_borrowed_guards.go:15`: `borrowedPolynomialCaller[T any]`.

Consumers include `pkg/ast/ast_proof_manager_rules.go:8` and polynomial caller guards. Running the retained checker from the module root fails:

```text
$ CGO_ENABLED=0 go run ./cmd/portcheck
math/polynomial/numeric_borrowed_guards.go:15:1: Go generics prohibited by AGENTS.md
exit status 1
```

**Impact:** The ordinary test suite is green, but the project-specific verification command is red and the explicit no-generics implementation requirement is violated. This finding does not assert an independently measured performance regression.

**Proposed fix:** Replace the declarations and their instantiations with concrete implementations. Keep ASTHandle-specific buffers/vectors in `ast`, with any independent storage primitives in `arena` or `util`. Use concrete polynomial caller lookup functions in `math/polynomial`. Do not move AST-dependent types into util or introduce util-to-ast imports to share the implementation.

**Validation after fixing:** Run the no-generics checker over the entire module, affected ownership/allocation-failure tests, and the full native suite. Resolve A5 as well before expecting the complete portcheck command to pass.

**Repair:** The generic util containers are replaced by concrete `ASTHandleBuffer` and `ASTHandleVector` in `pkg/ast/`, and all production users now use those concrete types. Polynomial caller lookups now use concrete pointer assertions and preserve the original binding traversal. The no-generics check passes through the complete portcheck command.

## A3. PointerVector allocation accounting uses inconsistent widths

**Evidence:** `pkg/util/pointer_vector.go:13` obtains `reflect.TypeOf(d).Size()`, the dynamic value width. Allocation and release during growth use that width. `Close`, at `pkg/util/pointer_vector.go:39-40`, releases capacity multiplied by `reflect.TypeFor[T]().Size()`, the static slice element width.

This valid instantiation reproduces the defect on the audited 64-bit host:

```go
b := arena.NewBudget(^uint64(0))
var v util.PointerVector[interface{}]
if err := v.Push(b, 1); err != nil { panic(err) }
fmt.Println(b.Used()) // 16: capacity 2 times dynamic int width 8
v.Close(b)           // panic: arena: budget release underflow
```

The slice requires two interface slots, each 16 bytes. The reservation is 16 bytes, whereas Close releases 32. Different dynamic values can also make growth account with a different stride from an earlier allocation.

**C++ comparison:** `src/util/vector.h:59-99` bases allocation on static `sizeof(T)`. The generic Go implementation does not preserve that rule for interface elements.

**Impact:** The exported utility can undercharge its arena budget and panic during cleanup. Reviewed concrete AST handle uses do not demonstrate this particular mismatch; the reproducer establishes a defect in the admitted utility API.

**Proposed fix:** In the concrete replacements required by A2, use a single fixed element width for every reserve and release. Preserve accounting for simultaneous old and new storage during growth and publish new backing only after successful reservation/allocation. Do not infer slice slot size from a stored value.

**Validation after fixing:** Verify actual storage charges through initial growth, later growth, failed reservation, erasure, and Close. Verify that cleanup returns the budget to its initial value.

**Repair:** The concrete AST containers reserve and release using the same static ASTHandle width. Interface instantiations are no longer admitted. Migrated container tests verify source growth, erasure and balanced cleanup; an additional vector test verifies old-plus-new peak refusal, unchanged published values/backing after failure, and clearing the erased slot.

## A4. Zero-sized elements and zero stride panic

**Evidence:** `pkg/util/pointer_buffer.go:36-37` divides by `reflect.TypeFor[T]().Size()` without checking for zero. `pkg/util/source_vector_capacity.go:6-12` likewise divides by its public `stride` argument.

Reproduced:

```go
var v util.PointerBuffer[struct{}]
for i := 0; i < 17; i++ {
    if err := v.Push(nil, struct{}{}); err != nil { panic(err) }
}
// Seventeenth push: panic: runtime error: integer divide by zero
```

The first 16 pushes fit inline; the spill path divides by zero. `SourceVectorCapacity(0, 0)` reaches the same unchecked division.

**C++ comparison:** C++ object types have nonzero `sizeof(T)`. The Go `any` constraint introduces zero-sized types that the source container did not have to handle.

**Proposed fix:** Use concrete nonzero-width element storage as required by A2. If SourceVectorCapacity remains an exported helper, validate stride before division and return a documented configuration/bounds error. Retain the normal source growth rules.

**Validation after fixing:** Cover the inline-to-spill boundary and zero-stride public helper input. Do not claim the panic proves a defect in existing ASTHandle buffering; that concrete element has nonzero width.

**Repair:** Concrete AST buffers cannot contain zero-sized elements. `SourceVectorCapacity` now rejects zero stride with `arena.ErrConfig` and invalid/overflowing capacities with `arena.ErrBounds` before arithmetic. The new zero-stride test first reproduced the panic and passes after the fix; the migrated buffer test covers the seventeenth insertion and failed spill publication.

## A5. Port coverage ledger and recorded outstanding work were lost

**Evidence:** Parent commit `d959ab8` contains `golang/port_manifest.json` and `golang/PORT_PROGRESS.md`. Neither has a migrated counterpart in the current tree. `pkg/cmd/portcheck/main.go:16` still defaults to `port_manifest.json`; its run function still loads, validates, and verifies evidence from that manifest after the syntax check.

**Impact:** Fixing the generics alone will not restore the full checker: its default ledger is absent. The prior distinction between ported, partial, and unported source units, API coverage, and recorded remaining obligations cannot be checked at the new paths. Passing Go tests does not replace that source coverage record.

**Proposed fix:** Restore the previous manifest and progress record from the parent commit, then migrate their targets and test-evidence paths to the new package layout. Preserve statuses and outstanding obligations instead of generating a fresh ledger that loses their history. Check each method/API target against its actual new owner and tests. Keep source paths without introducing C++ content hashes.

**Validation after fixing:** Run the complete portcheck command, including evidence validation. Confirm that all referenced Go files/tests exist, and that incomplete entries remain explicitly incomplete unless their behavior has actually been implemented and validated.

note: I restored these files from the parent comment at the top level of the repo.

**Repair:** The restored files stay at the repository root. All manifest Go targets are migrated to `pkg/...`, including extracted dictionary/deletion/module-parameter helpers and production collector owners. Five renamed dictionary tests are mapped to their current declarations. Methods spanning packages list explicit additional `go_targets`; verification still rejects missing tests and does not implicitly borrow unit-wide scope. No unit/method status, ownership claim, assertion/conditional inventory, or API completion entry was promoted. Portcheck verifies all 2,102 units and 677 API entries against the current reference: 37 ported, 270 partial, 1,795 unported. The progress record now identifies the current module/root-ledger layout and treats older paths as historical.

## A6. Incorrect C++ provenance paths

The following headers are demonstrably invalid against both the canonical source tree and its package mirrors:

| Go file and line | Current reference | Correct source |
| --- | --- | --- |
| `pkg/sat/sat_solver_model_test.go:1` | `src/util/util.LBool.h` | `src/util/lbool.h` |
| `pkg/ast/ast_mpf_format_numeric_stop_source_test.go:3` | `src/ast/fpa/fpa_decl_plugin.cpp` | `src/ast/fpa_decl_plugin.cpp` |
| `pkg/sat/sat_solver_report_params_fault_test.go:1` | `src/util/param_descrs.cpp` | `src/util/params.cpp` and, for declarations, `src/util/params.h` |

The param_descrs implementation is visible at `src/util/params.cpp:49` and `src/util/params.cpp:210` onward. `util.LBool` is a Go qualified name, not part of the C++ filename.

**Impact:** Readers cannot follow the stated source references. These typos do not establish that the Go files themselves are in the wrong package.

**Proposed fix:** Repair those paths and use `file:line` or `file:start-end` formatting for line references. Expand ambiguous `.h/.cpp` shorthand when updating other headers, checking the actual declaration/implementation before assigning a line range. A path-existence check can catch this without hashing source contents. Exclude intentionally synthetic paths in portcheck's own fixture tests.

## Placement decisions requiring review

### D1. A sorting-network test adapter lives in production ast

`pkg/ast/util_psort_ast_context.go:1` cites `src/test/sorting_network.cpp:150-181`, the `ast_ext2` test adapter, rather than an AST production implementation. Its public PsortASTContext owns clause/trail state and is used by the production AST sorting-network specialization.

This is a real mismatch of source category, but not enough evidence to declare the package placement wrong: the port may intentionally promote the adapter into a reusable context.

**Proposed resolution:** Decide whether this is a test fixture or an intended production adapter. If test-only, move the adapter to `pkg/test` and decouple the AST sorting network from its concrete test context using a narrow non-generic contract. If intentional production functionality, document that promotion, its owning behavior, and its difference from the original test adapter. Keep the actual network algorithm's provenance as `src/util/sorting_network.h`.

### D2. The independent max-cliques algorithm is embedded in sat

`pkg/sat/sat_solver_max_cliques.go` and `pkg/sat/sat_solver_max_cliques_rows.go` implement `src/util/max_cliques.h`; `pkg/sat/sat_solver_max_cliques_set.go` supplies its set support. The SAT caller is `src/sat/sat_solver.cpp:3113-3117`. The Go implementation is coupled to SAT literal rows/runtime storage.

**Proposed resolution:** Review whether the graph algorithm should be a concrete util implementation with a SAT conversion wrapper, or remain an explicitly documented SAT specialization. Extract only the independent graph operations if moving it; do not introduce util-to-sat dependencies. Current placement alone is not evidence of incorrect clique results.

## Placement differences that should not be mechanically reversed

| Go family | C++ origin | Assessment |
| --- | --- | --- |
| `pkg/smtcore/` | Multiple SMT, command-context, parameter, and related directories | Deliberate amalgam to avoid import cycles; retain. |
| `pkg/arena/` | Go port storage infrastructure | Separate leaf is intentional; no matching C++ directory is required. |
| `pkg/ast/ast_ref*.go`, `ast_ref_vector*.go`, `util_ast_literal_vector*.go` | `src/util/obj_ref.h`, `ref_vector.h`, `vector.h` | Concrete AST types and ownership belong with ast; moving these into util would invert dependencies. |
| AST BV character buffers, FPA scoped values/tables | util buffer/numeral/hash-table templates | AST-specific adaptations; util source provenance does not by itself make their placement wrong. |
| `pkg/ast/util_psort_ast_{bounds,costs,network,sorting}.go` | `src/util/sorting_network.h` | AST specialization is reasonable; review the separate test context above. |
| Polynomial content/GCD/PSC buffers, refs, vectors, skeleton/scoped numeral storage | util buffer/ref/vector/numeral templates | Concrete polynomial/numeric ownership supports placement in math/polynomial. |
| `pkg/model/model_constant_map.go` | `src/util/obj_hashtable.h` | Model-owned AST-handle map is a specialization, not a displaced generic util port. |
| SAT approximate sets, assignment storage, index maps/sets, literal rows, scalar runtime, variable queue | util set/vector/map/heap templates | SAT-specific scalar types and lifetimes make current specialization placement reasonable. |
| `pkg/sat/tactic/sat_goal_ast_map.go` | `src/util/hashtable.h` | The specialized map belongs with its goal-to-SAT owner. |
| `pkg/util/numeric_source_delete.go` | Polynomial caller deletion guards plus util numeric implementations | Primitive numeric destruction belongs below its polynomial callers; retain lower-layer ownership. |
| `pkg/util/owned_string.go`, `util_numeric_stream_string_view.go` | Shared string semantics, including `src/smt/params/qi_params.cpp` call sites | Shared storage helpers can remain in util; a higher-level call site is not necessarily implementation ownership. |
| `pkg/clibrand/`, `pkg/testoracle/`, `pkg/portcheck/` | Go RNG adaptation and port verification infrastructure | Go-only support packages do not require a one-to-one C++ directory counterpart. |

## Validation and limits

Completed checks with `CGO_ENABLED=0` and external temporary Go build/cache directories:

```sh
cd pkg
go test -count=1 ./... -timeout=30m
go vet ./...
go test -tags g3checked,g3debug -run 'TestBoolRewriterStar|TestBetaReducer|TestRewriterEngine' -count=1 ./ast/rewriter
go run ./cmd/portcheck
```

The full native suite, vet, and focused checked/debug rewriter tests passed. Portcheck failed as documented in A2. No WASM, WASI, race, or broad checked/debug sweep was run. No C/C++ oracle executable was used.

Independent production probes reproduced A1, A3, and A4. Linking all production collector-owner packages made the A1 lookup succeed. A typed nil pointer passed to PointerVector did not reproduce a failure; it is not reported as a pointer-nil defect.

The inventory covered 2,105 Go files. A structural comparison with the parent matched 15,664 function bodies by file/function identity; 13,831 remained structurally unchanged after removing comments, identifier spelling, and import qualifiers. This is triage evidence only: renamed identifiers can change semantics, and structurally changed bodies may correctly implement extracted helpers. Test bodies are included in those counts.

Source-header triage produced 61 placement candidates, including legitimate domain specializations and malformed/shorthand references. The tables above distinguish the actionable questions from moves that would recreate import cycles. Focused reads compared parameter registration, new container growth/accounting, numeric caller guards, AST payload/symbol boundaries, and rewriter extraction with their relevant source/pre-reorganization implementations.

All 1,929 package-mirror C++ files compared byte-for-byte with their corresponding canonical `src/` files; none differed or lacked a canonical counterpart after accounting for nested directories and amalgams. This comparison used no source hashes. A correct mirror does not prove that its Go implementation is faithful.

This is a repository-wide inventory and test audit with targeted behavioral/source review, not a line-by-line proof of all ported algorithms. No new solver/model correctness defect was established beyond the specific findings above. Existing unported or partially ported functionality should be assessed using the restored ledger rather than inferred to be complete from passing tests.

## Reproducing A1 without test-only registrations

Save the following as a temporary file outside the module and run `go run /absolute/path/to/probe.go` from `pkg/`. It uses the same small storage configurations as the parameter tests. No source-tree modifications are needed.

```go
package main

import (
    "fmt"
    "github.com/glycerine/g3/pkg/arena"
    "github.com/glycerine/g3/pkg/util"
)

func must(err error) { if err != nil { panic(err) } }

func main() {
    b := arena.NewBudget(^uint64(0))
    s, err := util.NewSymbolManager(util.SymbolConfig{
        Budget: b, PageSize: 64, DescriptorPageSize: 2,
    })
    must(err); defer s.Close()
    words := arena.VectorConfig{
        PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b},
    }
    q, err := util.NewRationalService(util.RationalServiceConfig{
        Integers: util.IntegerConfig{
            Cells: arena.Config{PageSize: 2, Budget: b}, Words: words,
        },
    })
    must(err); defer q.Close()
    p, err := util.NewParamsContext(s, q, util.ParamsConfig{
        Budget: b, PageEntries: 1, Text: words,
    })
    must(err); defer p.Close()
    text := words; text.PageSize = 64
    c, err := util.NewGlobalParamsCatalog(s, util.GlobalParamsCatalogConfig{
        Budget: b, Text: text,
        Descriptions: arena.BytesRegionConfig{
            PageSize: 64, Descriptors: arena.Config{PageSize: 2, Budget: b},
        },
    })
    must(err); defer c.Close()
    g, err := util.NewGlobalParams(s, p, util.GlobalParamsConfig{Budget: b, Catalog: c})
    must(err); defer g.Close()
    _, err = g.GetValueName("timeout")
    fmt.Println(err)
}
```
