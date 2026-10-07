# Package reorganization audit

Audited 2026-10-07 at commit `fba0ad7` against parent `d959ab8` and the local C++ sources. Module: `pkg/go.mod`, `github.com/glycerine/g3/pkg`.

Follow-up scope: A1 is declined. A2–A5 are repaired. All attempted A6 source-header changes were reverted at the user's request. The later D1/D2 study below updates recommendations only; their implementations and A6 headers remain untouched.

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

Follow-up study at `e89da4f`: recommendations only; no Go files, source annotations, or coverage claims were changed.

### D1. Keep the concrete AST specialization in ast; distinguish it from solver integration

**Recommendation:** Retain `PsortASTContext`, `PsortASTNetwork`, and their AST literal-vector support in `pkg/ast/`. Close the placement question as an acceptable concrete specialization. Record that the context ports a C++ test adapter and that production solver integration remains a separate obligation. Do not move the context alone into `pkg/test`, introduce a callback/interface layer solely to relocate it, or move AST-dependent code into util.

**What is actually being ported:**

- `src/test/sorting_network.cpp:150-181` defines `ast_ext2`, a clause-collecting AST adapter. It borrows an ast_manager, owns reference-counted clause/trail vectors, creates fresh `"x"` Booleans, and builds AND/OR/NOT terms. That corresponds to `pkg/ast/util_psort_ast_context.go`.
- The cardinality/sorting algorithm is production utility code, `src/util/sorting_network.h:120-1046`, instantiated in Go for ASTHandle by the `util_psort_ast_{network,bounds,costs,sorting}.go` files. Its location in ast is a consequence of its concrete element type and ownership, not evidence that the C++ utility belongs to the AST directory.
- Current repository callers of NewPsortASTContext/NewPsortASTNetwork are AST tests. There is no production integration caller outside this specialization. The tests consume the context's private clause vectors and use independent propositional oracles.

**Why the proposed context-only move is unsuitable:** `NewPsortASTNetwork` accepts `*PsortASTContext` directly (`pkg/ast/util_psort_ast_network.go:79-95`). Moving that type to a package that imports ast would require ast to import its own client, creating a cycle. The context also uses private manager locking/origin data and private acquired/published push-receipt fields (`pkg/ast/util_psort_ast_context.go:33-58,85-99,243-250`). Its lifetime code is part of the tested AST ownership boundary, not just a small adapter over public MkAnd/MkOr calls. RuntimeASTLiteralVector additionally carries a private psort construction-frame link (`pkg/ast/util_ast_literal_vector.go:30`). Moving the whole specialization instead would require a larger ownership/API extraction, without a demonstrated current consumer that benefits from it.

**Production C++ contexts are materially different:**

| C++ consumer | Required context behavior |
| --- | --- |
| `src/ast/rewriter/pb2bv_rewriter.cpp:386-397` | Fresh `"sn"` variables are registered in the enclosing rewriter's declaration vector; clauses are appended to its lemma vector. |
| `src/opt/sortmax.cpp:136-154` | Fresh declarations are tracked and inserted into a model filter; clauses are asserted directly into the optimization solver. |
| `src/smt/theory_pb.cpp:1165-1217` | The network works on SMT literals rather than ASTHandle values, internalizes Boolean terms, and emits justified auxiliary clauses into the SMT context. |
| `src/tactic/arith/card2bv_tactic.h:35-73` | Declares its own AST rewriter context, including lemma/trail ownership and its own fresh/clause operations. |

Consequently, the test collector cannot simply stand in for all of these consumers. In particular, exporting the current test-backed constructor does not establish model filtering, lemma publication, SMT clause justification, or the full registered C++ sorting-network driver.

**A concrete integration limitation:** The Go context exposes Clause to append a clause but has no public clause-count/read/reset API. The existing tests can inspect `c.clauses` because they are inside ast; an ordinary external caller cannot. Where the encoder introduces fresh variables, asserting only its returned result and omitting its generated clauses does not enforce the intended cardinality constraint. This is an incomplete external integration surface, not a demonstrated failure in the existing test consumers.

**Recommended follow-up when porting the first production consumer:**

1. Keep its concrete fresh-variable bookkeeping and clause/lemma/model-filter actions with that consumer. Port those actions from its actual C++ context; do not replace them with ast_ext2's collecting semantics.
2. If an external AST consumer deliberately uses a collecting encoder, provide bounded borrowed clause reads and an explicit reference-safe reset operation in ast. Do not expose mutable owning vectors or private ownership receipts merely to make relocation possible. Define context/result lifetimes and require all emitted clauses to reach the solver before checking the result.
3. Keep production specializations concrete and non-generic. The SMT-literal specialization will need its own scalar storage/owner integration. Any sharing of AST encoders should follow an actual consumer requirement and preserve the timing of source factory and publication actions.
4. Put registered solver-driven tests in `pkg/test` once the relevant real solver entry points exist. Keep the current AST ownership/failure tests close to the internals they audit.

**Evidence and acceptance:** Existing tests exercise actual factory/trail ownership, constructor refusals, unpublished clause cleanup and stopped destructors; the bounded cardinality oracle covers sizes 0–4 and the recursive oracle covers selected sizes 5, 10 and 12. These tests passed during this study. Before claiming production integration, add tests that assert the collected clauses through the real consumer, validate model filtering/lemma lifetime where relevant, and run the original solver-driven push/assert/check/pop cases. Preserve the ledger's current partial sorting-network/test-driver statuses.

### D2. Extract the shared negated-literal clique utility into util

**Recommendation:** Extract the clique engine and its unsigned storage into `pkg/util/`, retaining a thin SAT caller/conversion layer. Use a concrete complement-paired uint32 implementation, for example `NegatedLiteralCliques` in `pkg/util/max_cliques.go`. This is justified shared production functionality, rather than a move based only on the header's directory. Complete the extraction and SAT integration as one green implementation chunk before porting the SMT consumer; do not leave a second divergent copy behind.

**The second production consumer changes the assessment:**

- `src/sat/sat_solver.cpp:3113-3150` constructs max_cliques<neg_literal>, collects/deduplicates binary clauses, converts its requested SAT literals to unsigned indices, and converts unsigned output rows back to SAT literals.
- `src/smt/smt_consequences.cpp:577-614` independently constructs the same utility. It converts internalized AST expressions to SMT literal indices, reads the SMT watch lists, and converts the returned rows back to reference-counted AST expressions.
- Both negate policies flip the low bit, as shown by `src/sat/sat_types.h:76-85` and `src/smt/smt_literal.h:52-85`. There is no need for a generic type parameter or callback to support the two source instantiations currently present.

The present Go type is private to sat. Sharing it would otherwise require exporting SAT-specific storage machinery to smtcore or copying the algorithm into that package. The source utility takes unsigned vertices and does not know about solver objects or AST ownership. Neither collection policy belongs inside the shared utility.

**Correct abstraction:** This is not a general maximum-clique solver or an enumeration of all maximal cliques. `src/util/max_cliques.h:34-80,91-134` implements a greedy partition using complement-aware reachability and cached traversal. It appends groups of size greater than one and drops singletons. The concrete API should explicitly state the `vertex ^ 1` complement encoding and preserve this algorithm, output order and append behavior.

**The actual extraction seam:**

| Part | Recommended owner |
| --- | --- |
| Clique adjacency, cached closure, reachable/seen sets, todo list, greedy partition | util |
| Nested unsigned rows and their concrete Runtime allocation/copy/destruction support | util |
| SAT binary-watch collection, learned/user clause handling, pair deduplication, Literal conversion, enclosing mutex frame | sat |
| SMT internalization checks, SMT watches, literal2expr conversion and owning expression rows | smtcore, preserving its deliberate amalgam |

`RuntimeUintColumn` in sat is already an alias of `util.RuntimeUintColumn` (`pkg/sat/sat_solver_scalar_runtime.go:8`), so the graph's scalar vectors are not intrinsically SAT types. The difficult dependency is `satSolverLiteralRows`: its raw same-Runtime unsigned storage is shared with SAT literal presentation (`pkg/sat/sat_solver_literal_rows.go:15-30,93-126,234-267`). The graph uses its unsignedRow/reserveEmptyRows/appendColumn operations rather than SAT literal semantics.

**Concrete proposed implementation:**

1. Add a util-owned concrete unsigned-row container, extracting the existing raw storage and ownership protocol. It must retain real Runtime headers, scalar slot/generation provenance, charged cold metadata, failed-construction roots, and source/owning finalization. Keep SAT literal-span/read/conversion helpers in sat. Share the extracted storage where the protocols are identical instead of widening SAT internals for util access.
2. Move the clique engine, its raw-row operations, and its concrete embedded bit-set helper to util. Inputs/output rows remain unsigned; no util-to-sat or util-to-smtcore import is needed. The useful public surface is construction with Runtime/Budget, AddEdge, Cliques, and checked cleanup; avoid exposing the mutable graph internals.
3. Adapt satMutexFrame's graph and intermediate unsigned results to those util types. Keep `findMutexes` edge gathering and final row conversion exactly where they are. Preserve its current allocation/publication sequence and cleanup cursor (`pkg/sat/sat_solver_find_mutexes.go:10-20,34-78,140-247`). Do not introduce an additional whole-graph or whole-result copy just to cross a package boundary.
4. Move graph-only and unsigned-container tests into util; retain SAT watch-to-mutex integration and enclosing failure-root audits in sat. Tests that currently inspect private graph fields should move with those fields, rather than motivating public mutable debug fields.
5. Wire the SMT consumer to the same utility when its real context implementation is ported. `src/smt/smt_consequences.cpp` currently remains unported in the ledger; this extraction does not implement that caller or justify promoting its status.

**Preserve the existing source details:** Keep ascending uint-set selection, duplicate edge handling, wrapped unsigned max-plus-one, reserve/resize publication timing, append-to-existing-output behavior, and retained m_seen2/m_tc caches. In particular, `TestSATMaxCliquesRetainsGoalSpecificSourceCacheAcrossCalls` records that reusing a graph with a different goal can differ from a fresh graph. Both real C++ callers construct a fresh graph for each find_mutexes invocation. Do not reset the cache or "improve" that behavior during relocation.

Existing `util.RuntimeUnsignedVector` and `util.RuntimeUintSet` are useful storage references, but their standalone constructors charge separate owners and implement their own continuation/guard protocols. They are not automatic drop-in replacements for the engine's embedded columns/sets. First preserve the current embedded allocation/destructor sequence; any further storage unification needs its own failure-prefix evidence.

**Acceptance criteria:** Retain the independent eight three-vertex mutex-graph cases, output order, repeated-query/cache tests, unsigned-wrap boundaries, warmed budget plateau, nested resize refusal, stopped-owner cleanup and every actual SAT findMutexes reservation refusal. The existing independent graph oracle uses canonical negative-pair clauses; extend independent coverage to mixed literal polarities and transitive implication paths. Add a util-only consumer test that constructs and uses the graph without importing sat; add SMT-level conversion/integration tests with the later SMT port. Check that util has no higher-level imports, that no generics enter the new API, and run the full native suite before committing an extraction. Moving files alone is insufficient.

**Study validation:** Native and focused checked/debug runs of `go test -count=1 -run 'TestPsortAST|TestPsortASTCost|TestSATMaxCliques|TestSATSolverFindMutexes|TestSATMutexPairSet' ./ast ./sat` passed (with `-tags g3checked,g3debug` on the second run). This validates the existing baseline; it does not claim that the proposed extraction has been implemented or tested.

**Implementation (2026-10-07):** D2 is extracted into `pkg/util/negated_literal_cliques.go`, with `RuntimeUnsignedRows` providing the shared raw row owner. SAT retains watch collection, pair deduplication, the enclosing mutex frame, and the literal adapter; its intermediate graph/results are util owners. The SAT adapter uses the same concrete storage address and adds no owner reservation. Graph traversal, cached closure, embedded bit sets, unsigned wrapping, raw vector copy, and source/owning destruction order are preserved. Physical ownership is available through cold read-only diagnostic snapshots, rather than public mutable graph fields.

Graph-only and raw row failure tests now reside in util. The SAT captured-literal lifetime test and enclosing every-reservation audits remain in sat. Added independent Boolean enumeration for mixed signs/transitive paths, a util-only external consumer, and refusal at every measured graph-construction/query reservation. Focused native and checked/debug tests, whole-module vet, and source/no-generics ledger validation pass. The full native suite is the remaining commit gate. Ledger scope/status counts are unchanged; the SMT consequences consumer remains unported. D1 stays in ast to avoid an import cycle and needs no relocation.

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
