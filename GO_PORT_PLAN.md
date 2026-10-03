# g3: faithful Go port of the patched Z3 4.7.1 fork

This document specifies the port of **this repository's complete solver**, including Ken McMillan's interpolation implementation. The implementation is a translation of the existing algorithms, proof objects, theories, tactics, fixedpoint engines, models, parsers, and APIs. The intentional architectural change is storage and ownership suitable for Go. GoIvy compatibility is an early acceptance milestone; it does not narrow the final scope.

**Memory decision:** use owner-local, typed, pointer-free slab arenas with compact internal handles; pooled word pages for variable-size payloads; and marked regions only for data whose C++ lifetime is actually a region. Preserve logical reference counting and explicit cleanup where the source uses them. Do not make AST lifetime equal to solver scope lifetime. Do not require `unsafe`, cgo, `mmap`, experimental Go arenas, or `weak`.

The document has three layers: [requirements](#a-requirements-and-source-baseline), [architecture and implementation](#b-architecture-and-implementation), and [testing and acceptance](#c-testing-and-acceptance). Its code sketches specify contracts, not an implementation already present.

## A. Requirements and source baseline

### A1. What was surveyed, and what is still unverified

The available checkout is `/Users/jaten/go/src/github.com/glycerine/g3`; the Linux path originally supplied is the intended corresponding checkout on another host. At survey time `golang/` contains `go.mod` and `.placeholder`, with no solver implementation. The existing module is:

```text
module github.com/glycerine/g3/golang
go 1.26.4
```

Keep this module path and Go directive unless the user subsequently changes them. Commands below run from `golang/`. No additional root module is needed.

The survey covered the build component graph, API headers and dispatch, AST interning/release, region/trail allocation, SAT and SMT clauses and reasons, SMT backtracking, theory/quantifier/model interfaces, numerical representations, interpolation translation and scope handling, fixedpoint/tactic/optimization boundaries, generated registrations, and representative test assertions. It did **not** execute or validate the C++ reference, or inspect every algorithm line. Implementation tasks must read each function and its tests before translating it; this document is not a claim that all existing C++ behavior is sound or that every theory is complete.

`scripts/mk_project.py` declares version **4.7.1.0**. The source contains `src/interp/`, `src/api/api_interp.cpp`, and `src/api/z3_interp.h`. Do not replace this baseline with modern upstream Z3 or reconstruct interpolation from papers.

There are 1,619 files under `src/` with extensions `.cpp`, `.h`, `.hpp`, or `.pyg`, totaling 526,299 physical lines, including test files and parameter specifications. These numbers are inventory measurements, not estimates of Go implementation size. A reproducible content fingerprint is:

```text
565f0ba019a9925d84f3ed7f227e19e8bdccf026cfb20df473fec5bdf8da01e2
```

Fingerprint procedure: sort those paths lexicographically as repository-relative POSIX paths; feed each path's UTF-8 bytes, a NUL byte, and the raw 32-byte SHA-256 digest of that file into one SHA-256 accumulator. This avoids dependence on Git metadata or checkout location. Record a new baseline explicitly if the source changes.

Useful individual source checksums:

| Source | SHA-256 |
| --- | --- |
| `src/ast/ast.cpp` | `ac1de339a71c12fabaf951db2b0936bb1a84990b337ce268490633cb7bb5ff52` |
| `src/smt/smt_context.cpp` | `f41e7f4d9d46c32bcc78455bc324d7c6a1e3a01f57907d00bfd451028aba1a91` |
| `src/api/api_interp.cpp` | `de9d72951e3382dfb76acc188dde145d6f3d8412c66c39d414230baaea82d264` |
| `src/interp/iz3translate.h` | `2ae64fbee825268f7cc1b8c6e2a4aa5e813de7c188e8df3805220c9b8f6f9784` |

### A2. Behavioral fidelity contract

1. Preserve each implemented solver algorithm, its control flow, restart/branching policies, numerical semantics, proof rules, default parameters, and supported/unsupported cases. Preserve legacy and strategic solver factories separately.
2. Preserve satisfiable/unsatisfiable/unknown as three distinct results. Preserve reasons for unknown, exception categories, invalid-input behavior, and partial model behavior where the source permits it. Allocation failure, cancellation, timeout, unsupported interpolation, and a satisfiable interpolation problem are different events.
3. Preserve incremental assertions, decision levels, user scopes, assumption scopes, theory scopes, model/proof dependencies, and reset behavior. A source cleanup action remains an action in Go, even if physical deallocation changes.
4. Preserve exact rational, integer, algebraic, bit-vector, and floating-point semantics. Native floating point may be used only at the places where the C++ already uses approximate arithmetic or heuristics.
5. Preserve identity comparisons for simultaneously live objects and structural interning rules. Arena location, source AST ID, and logical equality are separate concepts. Numeric IDs may be recycled or compressed at the source's existing sites; never infer they are permanent global identities.
6. Preserve declaration names, kinds, sorts, de Bruijn indices, quantifier attributes, patterns/no-patterns, annotations, proof parameters, and reconstruction information. They are observable by GoIvy and by interpolation.
7. Preserve model/proof/dependency converters through tactics. A transformation with correct SAT status but a wrong reconstructed model or proof is an incorrect port.
8. Keep parameter names, types, defaults, descriptions, and unknown-parameter handling consistent with the fork. Generated tables count as functionality.
9. The implementation must build and run without a C/C++ compiler, GMP, libz3, OpenMP, Emscripten, a neighboring repository, or a reference executable. Its semantic core must compile unchanged for native Go, `js/wasm`, and `wasip1/wasm`.
10. Preserve source copyright and license notices in translated files, and retain `LICENSE.txt`. Inspect per-file notices as files are ported, including McMillan, Gurfinkel, and other contributors.

For deterministic workloads, aim to preserve source traversal, hash-table iteration, source RNG, and tie-breaking order. Go's unspecified map iteration must not choose solver branches. Raw process addresses, runtime memory statistics, wall-clock profiling strings, and OS signal machinery cannot be identical across languages. Record those representational differences; do not use them to justify changing algorithms.

Different valid models, cores, or interpolants can be semantically acceptable at an external interface, but that is a testing accommodation, not permission to substitute algorithms. If an unexpected structural difference appears, investigate the translation first.

No C ABI is promised by an all-Go implementation. Every implemented C++ API operation needs a Go semantic counterpart; old C/Java/.NET/ML/Python client binaries cannot link unchanged to a Go package. Translate client-visible behavior and examples into the Go API. Inventory build scripts, language bindings, developer tools, and OS adapters separately, recording the Go replacement or their role as source/reference material. Do not silently exclude production C++ components because GoIvy does not currently call them.

### A3. Complete component coverage

Use both `scripts/mk_project.py` and every `src/**/CMakeLists.txt` as coverage inputs. They expose components that would be missed by following GoIvy calls alone. This table is a work breakdown; none of these rows is optional for final completion.

| Source component family | Behavior to preserve | Planned Go file prefixes / package |
| --- | --- | --- |
| `util` | vectors/buffers, hashes/maps, sets, heaps, lists, symbols, parameters, resources, timers, tracing, allocation contracts, arithmetic support | `util_*`; selected independent packages below |
| `util/lp` | sparse LP matrices, exact/approximate arithmetic paths, feasibility and bounds | `lp_*` |
| `math/polynomial` | multivariate/univariate polynomial managers, factors, roots, algebraic numbers | `polynomial_*`, `upolynomial_*`, `algebraic_*` |
| `math/{hilbert,simplex,automata,interval,realclosure,subpaving,euclid,grobner}` | each manager, numerical domain, and associated tactic | respective prefixes |
| `ast` | all node kinds and plugins: basic/proof, arithmetic, arrays, BV, datatypes, finite domains, PB, sequences/strings/regex, FPA, recursive functions | `ast_*`, `*_decl_plugin.go` with collision-safe prefixes |
| `ast/{rewriter,macros,normal_forms,substitution,pattern,proofs,fpa}` | rewriting with proof/dependency tracking; binding and substitution; pattern inference; proof checks; FPA lowering | `rewriter_*`, `macro_*`, `normal_*`, `subst_*`, `pattern_*`, `proof_*`, `fpa_*` |
| `ast/rewriter/bit_blaster` | bit-blasting algorithms and proof/conversion behavior | `bit_blaster_*` |
| `model` and `smt/proto_model` | interpretations, universes, evaluator, factories, completion, model reconstruction | `model_*`, `proto_model_*` |
| `sat`, `sat/tactic`, `sat/sat_solver` | standalone SAT, simplification/inprocessing, extension constraints, incremental API, wrappers | `sat_*` |
| `nlsat`, `nlsat/tactic` | nonlinear arithmetic SAT, atoms, explanations, assumption dependencies, tactic | `nlsat_*` |
| `solver` | abstract result/factory contracts, combined solvers, tactic adapters, translation | `solver_*` |
| `smt`, `smt/params`, `smt/tactic` | legacy SMT core; congruence, relevancy, conflict resolution; all theory plugins; E-matching, MBQI; tactics | `smt_*`, `theory_*`, `qi_*`, `mam_*` |
| `interp` | complete iZ3 implementation, proof translation, scopes, checker, localization, interpolation APIs | `interp_iz3*` |
| `cmd_context`, `cmd_context/extra_cmds`, `parsers/{util,smt2}` | commands, scoping, tokenization, SMT-LIB and interpolation I/O | `cmd_*`, `parser_*` |
| `tactic` and all children | core, arithmetic, BV, AIG, UFBV, SLS, SMT logics, nonlinear combinations, FPA, portfolios; tacticals/probes/converters | `tactic_*`, `probe_*`, `converter_*` |
| `qe` sources under `src/qe` | quantifier elimination, guarded definitions, QSAT, theory projection | `qe_*` |
| `duality` | recursive proof-based fixedpoint/interpolation engine | `duality_*` |
| `muz/{base,dataflow,transforms,rel,pdr,spacer,clp,tab,bmc,ddnf,duality,fp}` | all registered fixedpoint engines and transforms, not just Spacer | `muz_*`, `pdr_*`, `spacer_*`, `datalog_*` |
| `opt` | optimization, MaxSMT engines, objective combinations, Pareto/lexicographic/box behavior, bounds | `opt_*`, `maxsmt_*` |
| `api` | all implemented entry points, AST containers, algebraic/polynomial/RCF, fixedpoint/Spacer, optimize, interpolation, FPA, and legacy APIs | `api_*`, public methods in main package |
| `shell` | actual frontend dispatch, flags, parameters, diagnostics, exit status | `shell_*` plus `cmd/g3` |
| `src/test`, `src/test/fuzzing`, examples and developer tools | existing tests, generators, runnable semantic examples; tool behavior classification | `*_test.go`, `test_vectors/`, `cmd/...` |

**Inventory correction:** the top-level directory summary in A1 does not enumerate every directory individually; `src/qe` must be included explicitly in the manifest and work queue. Always derive completeness from actual files, not this table alone.

The shell dispatch in `src/shell/main.cpp` includes SMT-LIB2, Datalog, DIMACS, WCNF, OPB, Z3 logs, and MPS. Its help also contains legacy wording. Port the implemented dispatch and parsers, rather than assuming every advertised old format has a working implementation. TPTP is an example/tool in `examples/tptp`, not evidence of a core shell frontend.

### A4. Interpolation requirements grounded in this fork

Read `src/api/z3_interp.h` and `src/api/api_interp.cpp` together with `src/interp/*`. The header explicitly identifies **the legacy SMT solver**, EUF, linear integer arithmetic, arrays, and allowed quantifiers: quantified AUFLIA. It also says interpolation may fail for other operators or unsuitable proofs. Full solver support for BV, strings, FPA, nonlinear arithmetic, or an arbitrary tactic does not imply that iZ3 can interpolate their proofs.

Important source choices to preserve:

- `src/interp/iz3translate.h` defines `IZ3_TRANSLATE_FULL`. Translate the full translator as the default. The direct translator and alternative code remain inventoried; do not switch defaults to simplify the port.
- `src/interp/iz3interp.cpp` defines `BINARY_INTERPOLATION` and computes individual cuts by constructing binary problems from the tree/sequence and translating the refutation for those cuts. Preserve this implementation. Do not replace it with a newly designed CHC or PDR API.
- `Z3_compute_interpolant` temporarily enables proof mode, creates a solver through `mk_smt_solver_factory`, applies timeout/resource/cancellation settings, and returns the check result plus interpolants or a model. The fork ignores some interpolation parameter arguments; preserve the actual behavior, documenting it instead of implementing newer imagined semantics.
- `Z3_get_interpolant` accepts a refutation and marker pattern. Premises absent from the pattern become background theory. Result ordering and repeated marker occurrences are observable.
- The API comment calls its traversal “pre-order” while describing children before parents. Follow executable traversal, and lock down occurrence/order behavior with tests. Do not resolve this documentation inconsistency by changing code.
- `iz3translate.cpp` scans skolem origins, handles quantified proof rules, localizes mixed terms, and introduces quantifiers. Localization order and existential/universal polarity are semantic, not allocation details.
- Its `add_quants` walks localization variables in reverse creation order and uses existential quantification for a variable originating inside the current A frame range, universal quantification otherwise. Preserve the simplifications in `apply_quant` as well as this polarity/order.
- Short scope endpoints in `iz3scopes.h`, sentinel values, and signed/unsigned conversions require explicit porting. Oversized interpolation trees must never silently wrap into an incorrect scope.
- Preserve interpolation problem reading/writing/checking and profile/error facilities as semantic or diagnostic APIs, respectively.

This code provides the missing quantified solving and quantified interpolation paths that motivated the backend change. It does not make first-order solving decidable or promise that every unsatisfiable quantified problem produces an interpolant.

### A5. GoIvy acceptance slice

The requirement source is the existing `golemma/IVY_REQS.md`, supplemented by the actual GoIvy code. That file describes the previous backend proposal and contains shorthand about scoped declarations. For g3, **the C++ lifecycle is authoritative**: solver pop removes assertions/internalization state; it does not invalidate every AST created since push. Parser declaration scope is another independent mechanism.

Map the following client operations to the ported APIs before changing GoIvy's adapter:

| Client requirement | Source implementation path | Acceptance obligation |
| --- | --- | --- |
| contexts, sorts, declarations, construction and inspection | `api_context`, `api_ast`, `api_quant`, theory API files; `ast/*` | context ownership, identities, sorts, declaration kinds, bound variables and attributes |
| solver assertions/scopes/assumptions/cores | `api_solver`; `solver/*`; `smt/smt_context*` | assertion order, repeated checks, assumptions scoped to a check, core membership and reset |
| models, universes, function tables and evaluation | `api_model`; `model/*`; proto-model generation | retain models across later checks/pop; evaluate all client-supported forms |
| Bool, UF, Int/Real, BV, arrays, enumerations, strings | corresponding plugins, rewriters and theory/tactic implementations | exact source construction, solving and reconstruction |
| forall/exists and substitution | `ast`, `ast/rewriter/var_subst`; quantifier manager, `mam`, `qi_queue`, model checker/finder | de Bruijn correctness, triggers, MBQI, relevant proof terms |
| parse/render with supplied declarations | `api_parsers`, parser/cmd context, AST/model printers | supplied names and IDs, escaping, round trips, canonical client rendering |
| binary/tree interpolation | `api_interp`, legacy SMT proofs, `interp/*` | marker semantics, proof retention, vocabulary, binding and conversion back to Ivy |
| interrupt/limits/check counters | `rlimit`, timer and API machinery; adapter hooks | safe cancellation, next-check recovery, deterministic client counters |

`goivy.Solver.BinaryInterpolant(clauses2, clauses1)` sets **A = clauses2**, **B = clauses1**, computes `(and (interp A) B)` in a fresh interpolation context, and converts the result back before closing that context. Preserve orientation. It deliberately retranslates into the new context; existing context handles cannot be reused there.

The GoIvy convenience wrapper currently turns SAT/unknown interpolation results into errors. The g3 core must retain the richer source status/model/error contract; the adapter can reproduce GoIvy's convenience behavior. GoIvy's module caches, nat/range guards, small-model search, stable activation names, minimized-core policy, and trace counters belong in its adapter, not new solver algorithms.

### A6. Completion criteria

There are two explicit milestones:

- **GoIvy milestone:** all client requirements above, including quantified interpolation, work using only Go; relevant Ivy regression/conformance tests pass on native and wasm runners.
- **Full-fork milestone:** every production source component, generated registration, implemented API operation, source test, and semantic tool/example has a documented translated counterpart or a justified platform/build replacement. Nothing is silently “phase two” after declaring completion.

A buildable package containing stubs is not a milestone. Internal unported operations must fail explicitly during development; they must not masquerade as SAT, UNSAT, a valid interpolant, or a source-compatible UNKNOWN.

## B. Architecture and implementation

### B1. Package and file layout

Keep solver-domain code in **one `package g3` in `golang/`**. The C++ component graph contains mutual conceptual dependencies: AST plugins, models, solvers, tactics, command contexts, and interpolation all meet at manager-owned types. Recreating every C++ namespace as a Go package would introduce cycles or force broad interface redesign.

```text
g3/
  GO_PORT_PLAN.md
  src/                         retained C++ baseline
  golang/
    go.mod                     existing module
    doc.go                     package g3, lifecycle and fidelity contract
    arena_handles.go
    arena_handle_release.go    !g3checked
    arena_handle_checked.go    g3checked
    arena_ast.go               domain ownership, AST records and cleanup
    arena_sat.go
    arena_smt.go
    arena_nlsat.go
    arena_trail.go
    ast_*.go                   main package; source-derived filenames
    smt_*.go
    interp_iz3*.go
    ...                        other prefixes from A3
    api_*.go
    port_manifest.json         per-file and per-operation progress
    port_deviations.md         reviewed representational/platform differences
    arena/                     storage primitives only
    mpnum/                     independent exact numerical managers
    utilhash/                  source hash algorithms / generic hash tables
    bitset/                    independent dense/sparse bit sets
    internal/
      portcheck/               manifest and static invariant checks
      testoracle/              independent bounded interpreters/enumerators
      gen/                     deterministic Go generators
    cmd/
      g3/                      CLI wrapper
      g3wasm/                  thin browser worker entry point
      portcheck/               progress/coverage checker
    test_vectors/
      source_tests/
      regressions/
      interp/
      ivy/
      numeric/
      fuzz_seeds/
      benchmarks/
```

These are planned files, not files created by this survey. For tests in a utility package, prefer loading shared fixtures through a test-only helper or embedding a package-local `test_vectors/` directory. Never add runtime fixture dependencies to production utilities.

Import direction is `cmd/* -> g3 -> independent utilities`; `mpnum` may use `arena`, `utilhash`, or `bitset`; no utility imports `g3`. A generic storage package knows slots and byte budgets, not AST kinds or theory cleanup. Add another package only when it can compile/test independently without solver-domain types. Do not move an unfinished subsystem into a package merely to reduce file count.

`internal/testoracle` defines its own small formula/value representations and imports no production `g3` algorithms or types. Test harnesses convert into those representations. This both preserves oracle independence and prevents a test import cycle from the main package back to itself.

Use file names that retain source paths by prefixes, e.g. `smt_context.go`, `smt_theory_array.go`, `interp_iz3translate.go`, and `muz_spacer_context.go`. Prefix collisions between namespace-local types, constants, and helpers consistently. Avoid filenames accidentally interpreted as architecture/OS suffixes. Keep constants and generated tables traceable to the source.

### B2. Mechanical translation rules

Before translating a source unit, record its constructor/destructor actions, ownership, aliases, exceptions, source tests, compile-time options, and generated dependencies. Translate its current algorithm first. Refactoring a heuristic or combining engines is separate future work.

| C++ construct | Go translation rule |
| --- | --- |
| namespace / class | prefixed type and receiver methods in the main package |
| multiple inheritance | explicit embedded state plus interfaces; preserve method dispatch, especially overridden defaults |
| virtual plugin / factory | narrow interface and a small number of long-lived implementation pointers; no interface per AST/variable/clause |
| templates | concrete types or generics where semantics match; preserve comparator/reset/destructor hooks |
| `unsigned`, `int`, `short`, bitfields | explicit `uint32`, `int32`, `int16` and masks as applicable; do not map everything to platform-sized `int` |
| `size_t` / vector length | checked length arithmetic; use `uint64` for budget/overflow calculations, then convert to `int` after bounds checks |
| union / tagged pointer / `void*` | explicit discriminant plus scalar or typed-handle payload; preserve sentinel distinctions |
| placement `new`, flexible trailing arrays | arena record plus payload allocation; no byte-to-struct casts |
| `ref`, `obj_ref`, `ast_ref`, scoped reference | explicit logical retain/release; carefully handle assignment, move, vector shrink, erase and overwrite |
| destructor / RAII scope | deterministic cleanup method or scope guard; `defer` at function/scope boundaries, explicit release in long loops |
| exceptions | typed internal failures with source catch boundaries, cleanup on unwind; translate to public errors/source error codes |
| macros for assertions/trace | checked-build invariant calls and optional tracing; assertions cannot contain required production side effects |
| global / thread-local manager | owner-local manager where it owns mutable memory; explicit config or runtime service for genuinely global source semantics |
| unordered table / pointer hash | source hash/probe/iteration algorithm over IDs where solver order matters; never rely on Go map range |
| OpenMP / native locks | preserve synchronization and algorithmic task boundaries in Go; provide portable serial execution of the same work |

Internal typed panics can model source exception unwinding where returning an error through every hot helper would distort the translation. Recover **only recognized port failure types** at the corresponding source boundary; programming panics, corrupt handles and invariant failures must not become UNKNOWN. Cleanup guards must restore proof mode, resources, temporary roots and partially initialized objects. Public ergonomic APIs return errors; compatibility methods preserve source error codes where needed.

Do not assume copying a Go struct is the source's copy constructor. Arena-backed mutable numerals, owned vectors, solver state, and reference wrappers often need explicit `Copy`, `Move`, `Swap`, or `Reset`. Add ownership comments to every such type.

Preserve source RNG arithmetic and seeds from `util/random*` rather than using `math/rand` for production heuristics. Preserve wraparound intentionally used by source hashing; do not reproduce signed-overflow undefined behavior without an explicit, tested choice.

### B3. Why the memory design is hybrid

The source demonstrates five distinct reclamation mechanisms:

1. `ast_manager::register_node_core` hash-conses nodes and retains their children, declaration information, sorts, quantifier bodies and patterns. `delete_node` removes the node from the intern table, recycles source IDs, runs plugin deletion hooks, releases children iteratively and frees storage. The intern table is not an owning reference to every node.
2. `util/region` and `trail_stack` reclaim scoped allocation after reverse undo. `smt::context` region-allocates enodes, trail entries and many justifications, with subsystem callbacks and destructors before reclaiming pages.
3. `sat::clause_allocator` uses a small-object allocator and packs C++ addresses into 32-bit offsets with architecture-specific segment logic. It is **not** a preexisting MiniSat contiguous relocating word arena. Binary/ternary watch representations and external constraints are specialized.
4. SMT clauses use a different variable-size representation: optional activity, deletion callback, justification and saved signed AST atoms. Learned clauses can survive decision backtracking and be reinitialized; source base-level transitions decide when they die. NLSAT has another clause/assumption representation.
5. `util/object_allocator.h` maintains worker-local pages/free lists. Recycle invokes `ResetProc`; an object returned from the free list is not constructed again. With `CallDestructors`, destruction occurs on pool reset/teardown, not each recycle. Preserve this pool contract separately from ordinary object destruction.

Consequently, a single arena truncated at every `Pop` would be incorrect. A new tracing collector over all handle graphs would require a complete root inventory and change source deletion timing. Logical reference counting is compatible with pointer-free arenas: it reclaims records and payloads inside large Go-managed backing arrays without involving Go GC for each term.

Go still owns and eventually collects those backing arrays. Pointer-free record and payload arrays reduce scanning work; they do not eliminate heap accounting, allocation cost, GC scheduling, or retained capacity. This design follows the runtime's distinction between pointerful graph data and pointer-free storage. [Go GC guide](https://go.dev/doc/gc-guide), [compiler/runtime optimization notes](https://go.dev/wiki/CompilerOptimizations).

### B4. Options considered and decision

| Option | Consequence for this source | Decision |
| --- | --- | --- |
| One Go object/pointer per C++ object; GC replaces all refcounts | simplest-looking translation, but scans the graph, loses deterministic logical deletion, and cannot safely emulate non-owning caches by default | permit only cold control objects; reject for hot solver data |
| Monotonic context arena; free everything on context close | easy identity and low scanning, but long-lived GoIvy sessions accumulate dead transient nodes/clauses | reject as sole allocator; use only where source lifetime truly is owner lifetime |
| One mark/truncate arena for all objects | invalidates proofs, models, API terms and learned clauses retained across pop | reject |
| Custom global mark/sweep or moving handle GC | possible later, but changes root management and deletion behavior throughout the port | do not introduce for the faithful port |
| Flat word arena with raw offsets everywhere | attractive for literals, but relocation must update every watcher, reason, cache and external constraint | use words for payloads, with stable handle indirection |
| Typed slabs + payload pools + source RC + scoped regions | preserves source ownership, limits scanned pointers, permits safe payload relocation and source-shaped translation | **selected** |
| `weak.Pointer` intern table | weak pointers describe Go heap object reachability, not logical uint32 arena slots; an arena backing array remains live as a whole | reject for term lifetime; [weak package](https://pkg.go.dev/weak) |
| `GOEXPERIMENT=arenas`, unsafe/off-heap allocation, or disabling GC | unnecessary platform/runtime dependency; object visibility and wasm complicate correctness | reject as a requirement; historical experimental arena API is build-gated [Go source](https://go.googlesource.com/go.git/+/refs/tags/go1.22.12/src/arena/arena.go) |
| Always use 64-bit generational internal references | stronger continuous stale-reference detection, but doubles common edge/watcher-reference storage | checked build uses this; release uses 32-bit internal references |

Start with array-of-records for source-shaped fixed records, plus separate variable payloads. Do not redesign the entire solver into structure-of-arrays before translating it. Split particularly hot fields into parallel arrays only where the source already does so or measurement justifies a storage-only change. Typed pointer-free structs are sufficient for no pointer scanning of their elements.

### B5. Ownership domains and lifetimes

An API context owns an AST manager and numerical/symbol services. Multiple solver objects in that context share AST identity, but each has its own search regions, trails, clause databases and theory state. Models, proofs, goals, converters and API containers retain their own AST roots. A nested/translated manager is a separate ownership domain, even if source code calls it a formatting or auxiliary manager.

| Domain | Data | Reclamation rule |
| --- | --- | --- |
| runtime/config | immutable registrations, version metadata, selected global parameter semantics | immutable or explicitly synchronized; never stores hot mutable solver terms |
| AST manager | interned AST/proof DAG, declaration information, expression dependencies, manager pins | source logical reference counts/hooks; final bulk close |
| numeric manager | numeral cells and limb buffers, rational caches | explicit ownership/reset matching source numeral managers; final manager close |
| solver persistent | clauses/lemmas, proof roots, theory structures surviving decisions, solver result objects | source deletion/reset/database reduction, not decision-region truncation |
| solver scoped | enodes, theory-variable attachments, trail entries, region justifications, scope-local queues/caches | exact source undo/callback order, then marked region rollback |
| operation scratch | rewrite stacks, temporary proof translation tables, polynomial scratch, parser buffers | operation mark/restore or reusable buffers after all retained outputs are promoted/rooted |
| external API result | returned ASTs, model/proof/goal/vector handles | source RC/non-RC context contract or explicit ergonomic ownership policy |

Maintain a per-type ownership ledger in the manifest. Every pointer field from C++ must be classified as owned, borrowed, weak/non-owning, intrusive-list link, scoped, or shared reference. Every container mutation must state whether removing an element releases it. A Go slice holding integer handles is **not** a Go GC root for the represented objects; the port's logical ownership must keep them alive.

### B6. Internal handles, public tokens, and source IDs

Use distinct domain types: `astRef`, `enodeRef`, `satClauseRef`, `smtClauseRef`, `nlsatClauseRef`, `justRef`, `vecRef`, `dependencyRef`, `numericCellRef`, and so on. Keep literal and Boolean-variable encodings distinct from allocator handles. Do not accept one domain's integer in another domain by an unchecked cast.

The representation contract:

- **Release internal reference:** a `uint32` slot, zero reserved as null. Reuse slots only at the source's valid destruction/reclamation sites. No bit-packed generation that imposes an arbitrary 16-million-slot limit.
- **Checked internal reference (`g3checked`):** a `uint64` holding `generation << 32 | slot`. All stored references and accessor parameters in that domain use the widened type, including trails, hash buckets and reason payloads. Slot lookup checks live state and generation.
- **Public token in all builds:** a `uint64` slot/generation pair plus an owning context/manager reference in the opaque wrapper. Validate owner, live state, generation, kind, sort as appropriate, and context closure before entry into the core.
- Every recycled slot carries a generation in pointer-free metadata. Increment it before reuse; retire a slot instead of wrapping generation back to a previously issued token. Exhaustion is an explicit resource failure.
- **Source AST ID** is a separate `uint32` field with the fork's expression/declaration ID generators. `RECYCLE_FREE_AST_INDICES`, `c_first_decl_id`, `compress_ids`, and their source use sites must be preserved. Do not expose arena slots as `GetASTID`.

Example layout contracts:

```go
// Release-only definition; checked file defines rawRef as uint64.
type rawRef uint32
type astRef rawRef
type enodeRef rawRef

// Public wrapper: deliberately opaque and context-owned.
type Expr struct {
    ctx   *Context
    token uint64
}

type slotMeta struct { // pointer-free
    generation uint32
    nextFree   uint32
    live       uint32
}
```

Use `makeRef(slot, generation)`, `refSlot`, and `refGeneration` helpers behind the build tags; domain-specific constructors enforce type separation. Literal bits and source numeric IDs remain fixed-width in checked builds. Watch/reason payload records containing handles widen through their typed fields; do not truncate a checked handle into a raw `uint32` payload. Test both representations.

**Honest limitation:** a live-bit check and a side generation table cannot detect a stale **release** internal slot after it has been reused, because the old reference carries no expected generation. The checked build can. Correct source ownership is therefore mandatory in release too. Public tokens always detect stale reuse. This trades release memory for debug detection; it does not claim that Go bounds checking proves solver correctness.

Internal handles belong to an implicit manager/solver at their call site; they do not embed owner IDs. Checked entry points assert owner/domain where objects cross managers. Public handles always carry ownership. Translation between managers traverses/reinterns the DAG; copying integer handles across contexts is forbidden.

### B7. Typed slab allocator contract

Implement `golang/arena/slots.go`, `slab.go`, `budget.go`, and their tests first. The independent allocator API operates on slots and generations, leaving domain-handle encoding and cleanup to `g3` or `mpnum`.

```go
type Slot uint32
type Generation uint32

// Domain wrapper decides when construction, release and destruction occur.
type Slab[T any] struct { /* fixed-size pages, metadata, free list */ }

// Required operations, with errors/typed failures fixed consistently:
// Allocate() -> (slot, generation)
// Read(slot) -> T; mutable access has a documented short borrow
// IsLive(slot, generation) -> bool
// Recycle(slot)   // only after domain cleanup
// TrimFreePages() // cannot move live slot identities
// Reset/Close(); Stats()
```

Rules:

1. Allocate fixed-size pages of records; use power-of-two slot indexing for predictable lookup. Start with 1,024 records/page for metadata/AST-like records, 4,096 for small scalar records, and benchmark rather than embedding assumptions about cache sizes.
2. Page directory slices are pointerful, but each backing record page is pointer-free. There is one header/reference per page, not per represented node. Cold manager/plugin pointers are ordinary Go pointers.
3. `T any` cannot prove pointer-freedom. A test/static audit must recursively reject pointers, strings, slices, maps, interfaces, functions, channels and `unsafe.Pointer` in hot record types. Fixed arrays of allowed scalar records are acceptable. Do not hide a pointer in `uintptr`.
4. Slab record locations do not change while live. Free slots join explicit free lists; a free page may be dropped while retaining a minimal generation/tombstone ledger needed to prevent old public tokens becoming valid after page reallocation. Reclaimed backing pages are reusable within the owner.
5. Slot/generation metadata must survive payload/page reclamation whenever a token could still be checked. Keep a paged generation ledger and occupancy metadata separate from releasable record pages. If metadata retention becomes material, measure it; never reset generations just to reclaim ledger bytes.
6. For ordinary destroyed/reconstructed objects, initialize every source-required field before publishing a recycled record. For constructed-object pools, preserve the reset-only contract and do not run the constructor/destructor again on every reuse. Poison only storage whose source state is actually dead; poisoning pooled state retained across reuse would be incorrect. Checked builds assert the applicable construction/reset/destruction counts and balanced ownership.
7. Do not keep `*T` to a slice element in another stored object. A temporary borrow must not survive allocation, callback, rollback, reset or compaction. Prefer getter/setter methods for recursive algorithms; reacquire by handle after potentially mutating calls.
8. Growth and free-list mutations use checked arithmetic. All owner budgets include allocated capacity, metadata, scratch and a reserve for cleanup. Fail before changing observable state when a predictable allocation exceeds the budget.
9. Storage recycle does not decide domain destruction. Ordinary domain `Destroy` releases references/callbacks first; constructed-object pools instead call their source reset procedure and defer final destruction to pool teardown. Track initialized pooled records even when on a free list, and destroy each exactly once when the source does. The allocator must not import the domain package.
10. Provide injectable tiny page sizes, slot limits and allocation failures for tests. Production defaults are tunable storage parameters, not solver heuristic parameters.

Go runtime OOM is not reliably a recoverable exception. The port can reproduce source memory-limit failure through conservative arena budget checks; it must not claim to recover every host/runtime exhaustion. Similarly, freeing logical records does not immediately return pages to the OS.

### B8. Variable payload arenas and compaction

Implement `arena/words.go`, `vectors.go`, and `region.go`. Use `[]uint32` word pages for SAT/SMT/NLSAT literal payloads and numeric limbs; use typed pointer-free pages for vectors of domain handles, so checked builds can widen handles without special packing. Strings/symbol spelling use byte pools with offset/length descriptors.

Start word pages at 16,384 words (64 KiB). A variable-vector descriptor holds page ID, offset, length, capacity and allocation class. Large objects get dedicated pages. Handle-vector descriptors use the same idea with their element type's actual size. Preserve length versus capacity: source clause shrink does not shrink allocated capacity, and release must use the original allocation extent.

Pool blocks by geometric size class. The free-list links and descriptor tables are scalar data. A caller sees a stable vector/record handle, never a raw backing address. Small empty vectors can use a distinguished empty value rather than a separate allocation. Repeated watcher/parent-list edits need reservable capacity and source-compatible ordering.

For clauses, keep header records in typed slabs and literal sequences in word pages. This preserves source fields and behavior while removing C++ flexible-array/alignment machinery. It is an arena design inspired by compact clause storage; it is **not** a replacement SAT algorithm and is not a 100-line whole-memory subsystem.

**Optional payload compaction has a narrow contract:**

1. Run only at a documented owner safe point with no borrowed slices/pointers and no active callbacks. First translate the source free-list allocator behavior; compaction is added only after relocation invariants pass.
2. Determine live payloads from live owning descriptors, not a guessed proof/SAT root walk. Preserve record handles and source IDs.
3. Allocate destination pages and construct an entire new descriptor-location table before publishing it. Charge peak old+new storage to the budget. If that cannot fit, skip compaction or release wholly free pages.
4. Copy logical contents in deterministic descriptor order, preserving capacities required by the source. Atomically publish the new location mapping within the single owner operation, then drop old pages.
5. Watchers, reasons, lemmas, API handles and proof references still name stable record handles; only payload locations change. A subsystem storing raw payload offsets must register and update every such reference or be excluded from compaction.
6. Do not compact scoped regions across live marks. Do not move live slab records or renumber source AST IDs as a side effect of payload compaction.

`ast_manager::compact_memory()` in this fork consolidates the small allocator and may rebuild the AST hash table. It does **not** move all AST objects or compress AST IDs. `compress_ids()` is a separate source operation. Preserve that distinction in Go.

### B9. AST/proof interning and logical reclamation

Implement `arena_ast.go` and `ast_manager.go` around the source's node kinds. Use a common pointer-free header with kind, source ID, hash, logical reference count, flags and kind-specific record/payload handle. A tagged union of scalar fields is acceptable; do not put one interface object behind each node. Derived app flags and quantifier metadata must match the C++.

Interning keys must include exactly the structural information used by source hash/equality: kind, declarations/sorts, parameters, child order, bound-variable index, quantifier body/binder metadata/patterns, and relevant plugin metadata. Distinct declarations with the same printed spelling are not automatically the same declaration. The source rejects certain declaration recycling with a changed range; preserve that check.

Port interning in the following order:

1. Validate input ownership/sorts as the source does and create a candidate record under an allocation guard.
2. Compute source-compatible structural hash and probe an equality-checked intern table. Hash equality alone is insufficient.
3. If an existing node wins, release candidate storage without retaining its children twice.
4. Otherwise assign the source ID, retain child/metadata references, initialize derived flags, and publish the new record. Audit source construction order rather than assuming this conceptual outline permits reordering observable callbacks.
5. Return the source-equivalent borrowed/owned result. A newly interned node may have zero external logical references; temporary roots and source scope guards must protect it until stored/retained.

Release follows `ast_manager::delete_node` iteratively: remove the intern entry before making its slot reusable; run plugin hooks; release each owned child/body/sort/pattern/parameter dependency; recycle the source ID at the source site; then recycle storage. Reference counts are not atomic inside a single-owner manager. Checked builds detect underflow, overflow and duplicate release.

Explicit root owners include API last-result/trail behavior, assertions, rewriter caches with owning semantics, models, universes, function entries, goals, converters, proof results, interpolation work tables, pattern databases and module/session caches. Non-owning tables must erase entries before destruction or test liveness as the source does. Audit all variants of `obj_ref`, `ref_vector`, and scoped vectors: vector truncate/destruction often has semantic release actions.

Proofs are AST applications with proof-sort declarations/parameters and premise/fact structure in this source. Preserve that representation and all rule identities; do not introduce a disconnected “simplified proof DAG” for interpolation. SAT/SMT justifications are separate objects that may construct/retain those AST proofs.

Reference counting works for the source's owned DAG edges; do not assume it collects arbitrary cycles. Recursive datatype declarations, dependency structures and engine graphs may use special source finalizers/manager teardown. Identify those cycles and preserve their source ownership protocol instead of inventing RC edges that leak forever.

### B10. Backtracking regions and trail implementation

Port `util/trail.h`, `smt::context::push_scope/pop_scope_core`, the quantifier manager, relevancy, theory callbacks, and asserted-formula scopes as a coordinated unit. Keep user/base levels and search/decision levels distinct. Assumption initialization has its own temporary scope and cleanup path.

A marked Go region consists of typed append-only stores and payload pages. A mark records each store's used length/page cursor and relevant cleanup-list/trail limits. Rollback recycles or truncates storage only after every object allocated after the mark has had its source cleanup performed. Region generation metadata must still reject handles from discarded slots in checked mode.

Do not implement trail entries as closures capturing pointers into slices. Use scalar tagged entries with an explicit owner/subsystem/field selector, object handle, index and old scalar/handle value. For a complicated source trail subclass, use a dedicated pointer-free payload record and dispatch its `Undo` through the owning subsystem. A few cold control callbacks can be interfaces, but millions of trail entries cannot be closures/interfaces.

```go
type trailEntry struct {
    kind      trailKind
    target    rawRef       // interpreted only by kind
    index     uint32
    oldScalar uint64
    oldRef    rawRef
    payload   rawRef
}
```

This is a sketch: some trails need two values, a vector descriptor, a numeral, or an owned AST reference. Record each variant's ownership explicitly. The trail itself may own the old value until undo/destruction. An old slot number without a retain is not sufficient when the C++ trail held an `ast_ref` or `rational`.

For `smt::context::pop_scope_core`, preserve the actual ordering, including:

1. cached generations, quantifier manager and case-split queues;
2. crossing base-level cleanup of lemmas/conflict/proof state;
3. auxiliary clauses, relevancy and fingerprints;
4. unassignment and reverse trail undo;
5. theory pop callbacks;
6. justification deletion hooks/destructors;
7. asserted-formula cleanup and propagation queue reset;
8. region rollback and level updates;
9. clause reinitialization and unit reassertion.

This list is a reading aid, not permission to omit intervening source actions. Translate the source body, including exception cleanup and calls made by the public `pop_scope` wrapper. In particular, region justifications can own vectors, and enode cleanup adjusts child parent lists. Go GC does not perform those actions.

Use region allocation for enodes and their scoped theory attachments; cyclic union-find/congruence links are typed handles. Preserve source trail updates of roots, class links, parents, merge explanations, theory-variable maps, Boolean associations, generations and relevancy. Ordinary solver pop need not delete the underlying immutable AST.

Query scratch regions must be nested safely: an MBQI auxiliary solve or interpolation proof rewrite cannot rewind the caller's scratch mark. Outputs copied into a persistent owning container retain/promote their referenced objects before operation rollback. Never let a persistent lemma refer to a region justification that will disappear; source assertions in `smt_clause.cpp` already distinguish this case.

### B11. Clauses, watches and reasons

Preserve the three separate clause domains. Share storage utilities, not algorithms or record semantics.

| Domain | Required representation and lifetime details |
| --- | --- |
| standalone SAT | source clause ID, size/capacity, removed/learned/used/frozen/reinit flags, glue/activity/approximate variable sets; literal words; binary and ternary special cases; external constraint indices |
| legacy SMT | clause kind, source capacity and activity, reinit/reinternalize flags, optional justification, saved atom handle plus sign, deletion-handler ID; decision/base-level lifetime and atom-release protocol |
| NLSAT | source clause ID, size/capacity, learned/activity fields, assumption-set handle and nonlinear reason dependencies |

Represent standalone SAT watch payloads as the source's two-word layout in release where possible, replacing `clause_offset` address segments with a stable `satClauseRef`. Preserve blocked literal, learned bit, ordering and external-constraint meaning. Checked builds use typed widened handle fields. No address bits or segment reconstruction are required in Go.

Represent SMT `b_justification` and equality justifications as explicit tags and payloads. Source null-clause justification, axiom, binary clause, and general justification are different states; zero must not accidentally conflate them. Port each conversion and comparison.

Clause destruction has a domain cleanup method that first detaches watches/reasons as required, invokes the source deletion event, releases saved atoms/proofs/assumption sets, and then recycles header/payload storage. Locked reasons or a source deferred-deletion flag prevent premature destruction. Preserve source binary/ternary clause behavior: do not allocate a large-clause record for every binary just to use a uniform API.

Learned-clause reduction must follow source retention thresholds and ordering. Physical compaction is optional and uses stable handles. Deleting a clause is not equivalent to deleting every proof term that once used it; proof ownership remains independent. Clause shortening preserves allocation capacity until storage release.

### B12. Exact numerics and mathematical managers

Create `golang/mpnum/` for the self-contained numerical layer. Port `mpn`, the internal `mpz` backend, `mpq`, and source rounding/fixed-precision support rather than replacing algorithms with approximate arithmetic. This fork's CMake defaults `USE_LIB_GMP` to OFF and selects `_MP_INTERNAL`; the source also contains GMP/MSBIGNUM paths. Inventory them, preserve their public numerical contracts through the portable internal implementation, and record removal of external arithmetic bindings as a platform replacement.

Keep **32-bit limbs** (`mpn_digit`/internal `digit_t`) and explicit 64-bit intermediate arithmetic. This gives one numerical representation across native and wasm. Maintain small inline integers and allocate limb cells only for big values. A numeral value is a scalar record with small value/sign and an owned numeric-cell handle; its limb buffer resides in word pages.

Required manager operations mirror the source: `Set`, `Copy`, `Move`, `Swap`, `Reset/Del`, arithmetic, comparison, conversion and rendering. Mutable numerals cannot be shallow-copied or aliased by Go assignment. Preserve allowed output/input aliasing at each source operation; scratch buffers must not overwrite a live input when a destination aliases it.

The C++ `rational` layer has a global synchronized manager. Make its storage owner explicit in Go, passing a numeric owner/service through AST and mathematical managers. A rational moving between owners is copied into the destination's manager. Immutable common constants can be initialized per owner; process-wide mutable numeral caches would reintroduce synchronization, lifetime coupling and wasm interference.

Do not prematurely extract AST-dependent algebraic/polynomial/QE code into `mpnum`. Polynomial objects can use numerical managers independently where possible; AST-aware adapters and Groebner integration stay in `g3` to avoid imports back into the solver.

Port signed division/remainder/modulus functions separately. Go's `math/big.Int.QuoRem` and `DivMod` have different division conventions; use the corresponding operation only as an independent test oracle after checking the source contract. Copy oracle operands into distinct destinations to avoid oracle aliasing assumptions. [math/big documentation](https://pkg.go.dev/math/big).

Port the source `mpf`, `mpff`, `mpfx`, `mpbq`, interval endpoints, infinitesimal rationals, algebraic numbers and real-closed-field routines with their own representations/rounding rules. `big.Float` or hardware `float64` is not an interchangeable implementation of arbitrary IEEE FP semantics. FPA theory lowering, NaNs, signed zero, subnormals, exception-free bit interpretation and rounding modes must follow the fork.

Numerical scratch is manager-owned and reusable; persistent numeral cells are not reclaimed with an unrelated SMT region. Tests must cover every path where source numeral destructors or reset hooks run, including object recycling, vector truncation, errors and manager close.

### B13. Solvers, theories, quantifiers and models

Port the source abstract solver and result contracts before their concrete factories. `Z3_mk_simple_solver` and interpolation use `mk_smt_solver_factory`; `Z3_mk_solver` uses the strategic factory. Keep both choices. Tactic-backed solvers and combined solvers need source state transitions, not a single universal replacement solver.

Keep theory interfaces close to `smt/smt_theory.h`: internalize atom/term, equality/disequality events, assignment/relevancy, propagation, final check, scope push/pop, model construction and proof/explanation hooks. Theory instances are a small number of ordinary Go objects. Their variables, rows, enodes, edges, bounds and constraints are pointer-free records/payloads.

Port all theory variants, including arithmetic's rational/approximate LP paths, integer cuts/branching, nonlinear integration, difference logic/UTVPI, arrays and array variants, BV, datatypes, sequences/strings, PB, FPA, and optimization-specific theories. Each variant has a manifest entry even if a default configuration does not select it. Preserve source selection parameters and fallbacks.

Quantified solving is a system of components:

- quantifier ASTs and correctly scoped variable substitution/normal forms;
- pattern inference and the matching abstract machine (`mam`);
- quantifier-manager plugins, fingerprints, generations and instantiation queues;
- E-matching/relevancy/equality event interaction;
- proto-model construction, model finder and model checker for MBQI;
- auxiliary solver/context lifetimes and proof-producing instantiation.

Translate all of them. A standalone finite-grounding helper is a test oracle, not the production quantified solver. Preserve source SAT/UNKNOWN/RESTART outcomes of model checking and their effect on search.

Models must retain their AST/numerical interpretations independently of mutable search state. Preserve complete versus partial function entries, else-values, array/as-array encodings, finite universes, fresh values and evaluator model-completion mutation. An exported old model can remain valid after another check or pop if it was retained according to the API contract. Do not store a borrowed pointer/handle to region-only enode state in a persistent model.

Goals/tactics carry proofs and dependencies with every formula as the source specifies. Port model/proof/dependency converters, tactic failure and cleanup, translation to another manager, and probe/tactical registration. Parallel portfolios must use source-equivalent ownership and cancellation between branches; mutable AST managers are not concurrently modified by arbitrary goroutines.

### B14. iZ3 implementation order and ownership

Give interpolation a dedicated work queue, not a late API wrapper. Files retain their source roles:

1. `iz3mgr`, hash wrappers, pretty printing, profiling and AST/operator abstractions;
2. `iz3scopes` range/tree calculations and `iz3base` assertion/vocabulary/frame analysis;
3. `iz3proof` translated proof records and `iz3proof_itp` interpolation rules;
4. `iz3translate` full proof translation, including arithmetic rewriting, quantifier instantiation, skolem-origin scan and mixed-term localization;
5. direct translator source path and its current limitations;
6. `iz3interp` orchestration and result quantification;
7. `iz3checker` and API/command problem I/O.

Use explicit operation-owned storage for translated proof nodes and memo tables, with AST roots retained in the source's owning collections. Result interpolants are retained into an API vector before the translator/scratch stores are destroyed. Do not retain raw input solver-region explanations after the source has converted them into AST proofs.

Maintain a **proof rule matrix** in the manifest: rule ID, constructor, producer(s), checker behavior/side conditions, full translator branch, direct translator branch, interpolation rule, unsupported outcome and tests. Many code paths are recursive and memoized; preserve memo insertion timing and localization creation order. A blanket “all proof rules supported” check is insufficient.

Pattern traversal is by occurrence where the source requires it. Identical interned formula/marker nodes can occur several times in the pattern and must not be deduplicated into one result. Background theory changes allowed vocabulary. Fresh skolems/local variables must not escape without the proper quantification and binding indices. Preserve the source's treatment of annotations, including unsupported labels.

For oversized short-based scope ranges, use wider intermediates and validate representability before assigning source-width fields. Do not silently turn overflow into a valid but wrong interpolant. Record the boundary behavior as a representational safety difference if the source has no defined supported behavior there.

### B15. Fixedpoint, QE, optimization and frontends

Translate all `muz` engines and transforms behind the source registration/factory interface. Spacer, PDR, Duality, relational Datalog, BMC, CLP, TAB, DDNF and their support code remain separate. This full port includes CHC capabilities; they do not replace `Z3_compute_interpolant` or GoIvy's binary interpolation semantics.

Keep predicate transformers, proof obligations, reachability facts, frame/level data, rule sets and certificates owner-local. Their C++ reference-counted graphs need audited cycle/cleanup behavior. An engine object may be pointerful; its potentially numerous facts, lemmas and obligations should use typed records and scalar handles. QE/MBP algorithms, array/arithmetic projection and guarded definitions must preserve the fork's exact choices.

Optimization includes MaxSMT backends, objective transformations, infinitesimal/infinite bounds, callbacks and multiple objective priorities. A model merely satisfying hard constraints is not sufficient acceptance for optimization. Preserve source proof/model conversion when objectives are lowered to SAT/SMT forms.

Parsers and command contexts can be ordinary cold Go objects, with interned symbols and reusable byte/token buffers. Port byte-level tokenization and escaping before making Unicode assumptions: SMT string/regex behavior follows the fork's implemented character model, not a newly selected Go rune model. Keep parser declarations/scopes separate from immutable manager ASTs and solver assertions.

Port all actual shell frontends and semantic developer tools/examples. Build-system operations such as dynamic library loading, foreign ABI export and platform resource sampling get explicit Go/platform replacements in the manifest. Use `io.Reader`/`io.Writer` interfaces for core parser/printing operations; path opening, process exits, signal handlers and host resources stay in CLI/platform adapters.

### B16. API ownership and concurrency

Provide an explicit compatibility context mode for the source RC contract, including last-result pinning and object tables; provide a non-RC mode with source-equivalent context AST trails. Translate `api/api_context.h/.cpp`, `save_ast_trail`, `save_object` and their release behavior before exposing public values.

For an ergonomic Go layer, choose context-owned returned AST values initially: each distinct exported AST is pinned in that context until close, and copying `Expr` values is safe. Make this retention cost explicit. Clients needing reclamation within long-lived contexts use an explicit owned-result/container or the RC compatibility API. Do not add `Expr.Release` to a freely copied value and pretend every copy owns an independent retain. Solvers/models/vectors/goals have explicit `Close` or reference operations with documented idempotence and ownership.

When implementing RC last-result replacement, preserve `save_ast_trail`'s temporary increment before resetting the old result: the new result can be the same AST whose only pin was the old result. Releasing the old pin first would free it before re-pinning. Include this exact regression for AST and analogous API object replacement.

No finalizer or `runtime.AddCleanup` is required for correctness. A leaked context can ultimately be collected as a whole by Go, but deterministic cleanup and budget reclamation use explicit close/reset. Closing the context invalidates its public values and releases backing pages even if wrappers still reference the closed context object. Results that must survive close are explicitly translated/serialized, not accidentally kept valid.

One mutable owner runs one solver/AST-manager operation at a time. A context mutex may serialize public operations and several solvers sharing it; internal routines do not reacquire that mutex. Independent contexts can operate concurrently. Source algorithms that intentionally run parallel tasks need isolated mutable stores or the same source synchronization protocol, not concurrent writes to generic slabs.

`Interrupt` sets an atomic cancellation state **without waiting for the operation mutex**. Port source polling/counting points and nested resource scopes. Deadline polling must work in a tight loop even where timer callbacks cannot execute. Close waits for active work/cancels as documented; callbacks must not reenter the same locked context unexpectedly. Test close/interrupt/check races under `-race`.

Keep source process-global parameter behavior in an explicit synchronized registry/snapshot where its API requires it. Memory arenas, symbol caches, RNG search state and numeral scratch are not process-global mutable singleton data.

### B17. Wasm and resource portability

The core has no `syscall/js`, OS process launch, native signals, thread pinning, cgo or dynamic native libraries. Browser and WASI wrappers import the same `g3` package and use Go's supported wasm targets. [Official WebAssembly guide](https://go.dev/wiki/WebAssembly).

Use byte-count budgets and checked `uint64` arithmetic before converting lengths into Go `int`, including limb/record/page multiplication. A 32-bit handle does not imply that every target has the memory to allocate its full namespace. Configure wasm budgets with room for the Go runtime, stacks, scratch, old+new compaction pages, input/output buffers and host overhead. Never rely on Emscripten pointer layouts or assumed native word size.

For browser use, execute a solver in a dedicated Web Worker so synchronous Go search does not block the UI event loop. Use source polling plus elapsed-time/resource limits; a message to that worker alone cannot necessarily interrupt a tight running check. For prompt host-driven interruption use a supported shared cancellation channel if the host supports it, or terminate/restart the worker as an explicit host-level operation. Worker termination discards the entire context; it is not equivalent to graceful source cancellation and recovery. Do not redesign the search into a resumable PDR engine merely to add UI responsiveness.

WASI supports a serial execution path for source parallel facilities. Native Go parallel scheduling may improve throughput, but the serial path must preserve logical work and all features. Limit/cancellation testing on wasm must execute a module; a successful cross-build alone is inadequate.

### B18. Generated code and coverage ledger

Create `port_manifest.json` as a complete inventory, initially marking all source units unported. Include source path/hash, component, Go targets, exported operations/types, source tests and assertions, generated inputs, conditional branches, ownership ledger, supported state, unit/property/fixture tests and approved deviations. A file-level “ported” tick must not hide unimplemented methods.

Generators must cover `.pyg` parameter specifications, `ADD_TACTIC`/probe/simplifier registrations, initializers/finalizers, API operation enumeration, version/error/decl-kind constants, and logging/replay tables. Read `scripts/mk_genfile_common.py`, `mk_util.py`, and `update_api.py` as specifications; generated C++ files absent from `src/` are still required behavior.

Commit generated Go tables and a deterministic generator or declarative input so ordinary builds/tests do not require Python/C++ or the neighboring checkout. A generator may read the preserved reference source when deliberately regenerating; the resulting Go tests/builds must not. An eventual Go-only source release contains all required tables and permanent fixtures.

`cmd/portcheck` verifies that new reference files/API operations cannot vanish from the ledger, source hashes have not drifted silently, and completed units name executable Go tests. The checker must distinguish source-existence inventory from hermetic runtime tests; only a maintainer inventory mode reads `../src`. CI semantic tests can run from the `golang/` subtree alone.

### B19. Implementation phases and dependency gates

Every task ships source-linked code, actual source-test ports, independent expectations, checked-handle tests where applicable, manifest updates and an ownership note. A partially translated source unit remains incomplete. Keep patches small enough that a reviewer can compare its algorithm to the source.

| Phase | Deliverable | Gate to the next dependent work |
| --- | --- | --- |
| P0: baseline | complete source/API/test/conditional inventory; source fingerprints; conventions and planned deviations | no component omitted; known source defects recorded, not silently fixed |
| P1: foundations | arena primitives and handles; numeric managers; hashes/containers; RNG/limits/params; registrations | unit/property tests; both handle modes; native and wasm build of utilities; allocation-failure cleanup |
| P2: AST | all AST base kinds/plugins, intern/RC, binding/substitution, source printers/parser scaffolding, proof constructors/checker | identity/lifetime/binding tests; public owner validation; proof rule matrix started |
| P3: solver spine | goals/converters/result/factory contracts, standalone SAT, legacy SMT congruence/search/trails/clauses | bounded SAT oracle; scope state machine; retained proofs/models; no fake unported outcome |
| P4: quantified AUFLIA | source arithmetic and arrays, model generation, E-matching and MBQI, iZ3 complete default path, API facade | quantified interpolation, independent vocabulary/binding checks, core/model checks; GoIvy milestone candidate |
| P5: remaining theories/math/tactics | BV/FPA/string/datatype/PB variants; polynomial/NLSAT/QE/RCF/subpaving; all tactics/portfolios | each source component's tests and independent semantic/property checks |
| P6: engines/tools | all fixedpoint engines/transforms, optimization backends, every frontend and semantic tool/API/example | fixedpoint certificates/reachability and optimum oracles; API/registration completeness |
| P7: closure | all ledger entries completed or justified platform replacements; full conformance, stress and performance reports | full-fork milestone, native + both wasm execution paths, Go-only build/test artifact |

These phases are dependency labels, not a requirement to wait for all of P4 before beginning unrelated P5 work. P4 includes only the vertical slice needed for Ivy; other production code still must be translated. Arithmetic/proofs/quantifiers/interpolation have significant dependencies, so do not promise a shallow API milestone before their producers exist.

Follow the per-component build graph at finer granularity: `ast` depends on polynomial support, arithmetic declaration plugins can need algebraic-number managers, and proof checking uses rewriting infrastructure. Move these mathematical prerequisites into the foundation/AST work queue before declaring their dependent plugins complete; P5 contains the remaining mathematical algorithms and solver integration. Sharing a Go package removes import cycles, not algorithmic prerequisites. An API/interface scaffold is not an implemented prerequisite.

### B20. Parallel sub-agent work plan

Parallelism is useful after shared contracts are fixed. This section authorizes an implementation coordinator to propose concrete sub-agent tasks when that session's user/tool policy permits delegation; it is not a request to create agents during this planning survey.

**Coordinator responsibilities:** own arena/handle/API contracts, source manifest schema, shared base types/interfaces, phase gates and final integration. Assign explicit file ownership and a named dependency gate to each agent. Agents may report needed interface changes, but must not independently redesign common types or edit another task's files. Reserve shared constants/registration tables and manifest integration for one owner.

| Work wave | Parallel tasks | Prerequisites and boundaries |
| --- | --- | --- |
| W0: read-only inventory | source/API/build inventory; test assertion inventory; GoIvy call/trace inventory; ownership/conditional-branch audit | all can begin immediately; each returns separate reports/manifest fragments, one coordinator merges |
| W1: foundations | `arena` primitives; `mpnum`; `utilhash`/`bitset`/source containers; independent `internal/testoracle` and fixture schemas | agree slot/generation/budget/numeric contracts first; these packages never import `g3`; oracle code cannot import production solver logic |
| W2: AST follow-ons | base AST/interning (one owner); plugin families (separate owners); parser/tokenizer/printer work; proof-rule/test inventory | AST header/ref/plugin interfaces frozen; plugin agents own their prefixed files; parser AST calls integrate only against declared real implementations |
| W3: two solver cores | standalone `sat_*`; legacy SMT congruence/region/trail/search; model/evaluator and converter infrastructure; independent mathematical managers | AST/numerical/solver result contracts ready; **SAT and legacy SMT are separate agents**, not alternative implementations of one core |
| W4: first theories and quantifiers | arithmetic/LP; arrays; E-matching/quantifier queue/MBQI; iZ3 scopes/base/translated-proof layer | freeze theory callback and explanation interfaces; MBQI waits for usable model/auxiliary solver gates; iZ3 producer-dependent branches remain incomplete until real proofs are available |
| W5: broader theories | BV/bit-blasting; datatype/recursive-function support; sequences/strings/regex; PB; FPA lowering | core plugins/theories/models stable; each agent owns source-linked theory/rewriter/plugin files and its tests; generated common enum/registry changes go through coordinator |
| W6: mathematics and higher engines | polynomial/NLSAT/RCF; QE/MBP; optimization; relational Datalog; PDR; Spacer; Duality; other fixedpoint engines | shared source solver/tactic/rule contracts ready; **serialize edits to `qe_*`, `muz_base_*`, and shared transforms** or assign those prerequisites first; separate engine prefixes alone do not remove dependency overlap |
| W7: integration and coverage | API/header groups and examples; CLI frontend groups; GoIvy adapter/conformance; wasm runner; performance/GC instrumentation; independent property suites | coordinator owns core API context/solver factory and shared registries; API agents port disjoint operation families; test agents add independent expectations without modifying algorithms |

Work that **must have a single design/integration owner**:

- handle representation and checked/release build tags;
- AST interning/RC and deletion ordering;
- SMT scope rollback and justification/lemma lifetime;
- numeric ownership and copy/alias contracts;
- shared proof/decl-kind IDs and generated registration tables;
- exception-to-result/error conversion and public context lifecycle;
- source scope/occurrence interpretation for iZ3 tree markers;
- performance changes to shared hot loops after semantic integration.

Once their contracts are reviewed, implementations/tests can be delegated in disjoint files, but competing versions of these foundations must not be integrated simultaneously.

Provide each sub-agent a task packet containing:

1. exact source files, active conditional branches and prerequisite revisions/contracts;
2. Go files it owns and shared files it may only request changes to;
3. source assertions it must preserve and independent oracle/expected fixtures;
4. allocation/root/cleanup contracts and checked-build obligations;
5. commands appropriate to that task, including wasm utility builds when relevant;
6. manifest fragment with per-method status and unresolved source behavior;
7. a finite definition of done and a concise handoff explaining observable differences.

A read-only test/oracle agent can work alongside a port agent for a difficult theory. It must derive expected results from mathematics/enumeration/source assertions, not copy the new implementation's outputs. Treat tests that call the same solver twice as consistency checks, not independent correctness evidence.

Integration rules:

- Prefer disjoint files in one checkout only if agents coordinate reliably. If an implementation session uses isolated checkouts/worktrees, follow that session's authorization and repository rules; never reset, rebase, or create Git backups of someone else's work.
- Do not let agents “make the build green” by adding competing shared stubs or deleting another agent's tests. Agree minimal compiling shared interfaces centrally.
- An agent may finish a utility before dependents exist; its own meaningful tests must run. Solver-domain contributions with unresolved producers stay marked partial even if they compile.
- Integrate prerequisite work first, run its gate, then integrate dependents. With a shared package, compile checks after each integration catch collisions early; broad solver tests run after the relevant vertical slice exists.
- Every C++ comparison made by any agent is converted into a permanent hermetic Go test with independent expected results before its task is complete. The coordinator backfills comparisons inherited from earlier work.
- Make corrective additional commits when needed; do not rebase or back up history in Git. No automated plan task should push or publish without authorization for that action.

## C. Testing and acceptance

### C1. Test ownership and fixture rules

Permanent inputs, expected outputs, regression cases, oracle provenance and benchmark workloads live in **`test_vectors/`**, either centrally under `golang/` or under the independent package that owns them. Never create a project fixture directory named `testdata/`. The user reserves that name for ephemeral Go fuzzing data.

For every ported source feature:

1. Read its actual source unit-test assertions, including calls in `src/test/main.cpp`, the CMake test list, examples and disabled/conditional branches. Record which assertions are active in the baseline configuration.
2. Port those checks into finite Go tests with the same semantic expectations. Preserve source aliasing, reset, ownership and error cases, not just example formulas.
3. Supplement print-only, empty, stress-only or self-consistency tests with explicit independent expectations. Label extra tests as added coverage rather than pretending the source asserted them.
4. Add unit tests for storage/lifecycle adaptation and property tests where the state space or aliasing warrants them.
5. Ensure the tests run without reference C++ files, executable, compiler or sibling checkout. Ordinary tests must not run Python generators or contact a network.

**Every on-the-fly C++ comparison must become a committed Go unit regression with independently justified expected results.** Save input, source baseline/build options, observed reference result, expected semantic assertion and independent derivation. A golden copied from C++ is useful conformance evidence but is not by itself an independent correctness oracle. Backfill earlier comparisons during implementation; this planning survey made none.

Source/reference experiments are optional maintainer tools and cannot be the only test for a ported behavior. Do not invoke `compile.sh` casually: it deletes its build directory. Use an isolated reference build directory and record its configuration if comparison work is later authorized/needed.

### C2. What the existing tests actually establish

The baseline has more tests than a “SAT/EUF/interpolation” feature list suggests, but several are incomplete. Examples inspected during this survey:

| Source | Actual assertion/behavior | Go port requirement |
| --- | --- | --- |
| `src/test/ast.cpp` | repeated `(and a c)` applications compare identical, an `(or a c)` differs; persistent expression-array checks sizes and elements; old numeral/variable examples are commented out | preserve active application/array checks; add explicit lifetime/hash-collision/binder tests, and inventory commented cases separately |
| `src/test/mpz.cpp` | large-number arithmetic/comparison identities, `INT_MIN` negation, in-place aliasing and bug regressions; some cases only print | port all active assertions; independent decimal/limb answers and alias tests for print-only cases |
| `src/test/object_allocator.cpp` | recycled coefficient resets to zero; worker allocator contents survive operations; allocation/destruction counters balance | preserve logical reset/destruction checks with Go storage; add deterministic operation scripts and threaded cases where applicable |
| `src/test/small_object_allocator.cpp` | allocate/free/reuse exercises several sizes, largely trace output | add content/liveness/non-overlap/reuse checks rather than claiming the source checked addresses |
| `src/test/region.cpp` | `tst1()` contains only `// TODO` | there is no substantive source region assertion to inherit; add comprehensive mark/undo/lifetime tests |
| `src/test/sat_user_scope.cpp` | seeded incremental solver status equals a freshly rebuilt solver; outer loop is infinite | preserve the coherence invariant in a finite deterministic state-machine test; add independent exhaustive SAT oracle |
| `src/test/check_assumptions.cpp` | `(p or q or r)` checked with `not p` is SAT; then assertion `not p` and assumptions `not q,not r` exercise a crash regression without a final result assertion | preserve the SAT and no-crash checks; independently establish and assert UNSAT for the final query |
| `src/test/proof_checker.cpp` | constructs complementary hypotheses, unit resolution and a tautological lemma; checker acceptance is asserted | port acceptance; add corrupted premises/conclusions and side-condition checks |
| `src/test/model_retrieval.cpp` | retrieves an array model, reuses a value in an assertion and prints results | add explicit SAT and interpretation correctness plus retained-model lifetime assertions |
| `src/test/quant_solve.cpp` | validates guarded substitutions by solving the negated implication; some old validation code is disabled | port active guarded-definition checks and distinguish self-consistency from independent finite/arithmetic semantics |
| `examples/c/test_capi.c::interpolation_example` | computes interpolation for `PredA and PredB` versus `not PredB and PredC`, printing the result | add asserted UNSAT, one result, shared vocabulary `{PredB}`, and independent truth-table Craig checks |
| GoIvy `z3bridge_interp_test.go` | checks two Craig obligations for `x<0` versus `x>2` using the same backend; expects SAT interpolation to report an error | retain adapter behavior; add vocabulary/binding checks and an independent semantic oracle |

Do not port the infinite outer loop literally into `go test`, count print statements as assertions, or activate previously disabled algorithm tests without noting the change. Inventory source tests even when they are not registered in the default runner.

### C3. Arena, ownership and backtracking unit tests

Implement these first, against tiny injectable capacities as well as production defaults:

- allocation across page boundaries, large dedicated payloads, zero/empty values, size classes, alignment-independent records, live/non-live lookup and closed owner;
- generation increment, double free, generation exhaustion/slot retirement, stale public token, checked stale internal token, and reuse after fully releasing a page;
- domain/type separation and cross-context rejection; `GetASTID` distinct from slot/generation; source ID recycling/compression does not invalidate live arena handles;
- overflow at size/capacity/page/byte calculations and conversions; budget charges for retained capacity, metadata, destination compaction pages and scratch;
- failures injected before allocation, between candidate/intern publication steps, during child retain/copy, and during compaction preparation; no partial publication and no leaked owned refs;
- free-list and payload data integrity when size-class blocks are reused; reset/destructor counts and numeric zeroing;
- nested region push/pop including empty scopes, bulk pop, pop zero, invalid depth, base scope, marks crossing pages and allocation after rollback;
- trail undo in reverse order, deletion hooks before reclaim, captured old AST/numeral values kept live, level-zero cleanup and region justification-owned vectors;
- source SMT pop ordering through observable test hooks; lemma persistence across decision pop, deletion across base pop, reinit clauses, unit reassertion and saved-atom sign handling;
- RC last-result replacement with the identical AST/object, where the old result is its only existing pin; worker pool recycle invokes reset without premature destruction/reconstruction;
- proof/model/API result surviving solver pop and later checks under its ownership contract; context close invalidates tokens; translating/copying before close preserves the destination;
- payload compaction changes locations without changing logical content, handles, watches, reasons, proofs, source IDs or markers; allocation failure leaves old locations usable;
- pointer-freedom audit of every hot record including fields imported from `mpnum`, checked-build representations and fixed arrays;
- ordinary GC forced by tests at borrow-free safe points: live backing arrays remain reachable via owners, no hidden Go pointers are needed, and close drops owner references.

Do not depend on when the runtime returns memory to the OS. Assert owner counters/free lists/capacity and liveness directly; use runtime metrics only for benchmarks and coarse trends.

### C4. Unit and semantic suites by subsystem

| Area | Required coverage, beyond arena adaptation |
| --- | --- |
| containers/hashing/RNG | source collision handling, erase/reinsert/tombstones, comparator ties, deterministic iteration where used, seed/output vectors, buffers/resizing/destructor hooks |
| integers/rationals | small/big transitions, signed division families, zero/one/min/max, normalize/gcd, parsing and rendering, every allowed input/output alias pattern, overflow-free exact identities |
| fixed/FP/interval/algebraic | rounding modes, precision changes, endpoints/open bounds, infinities/infinitesimals, NaNs/zero/subnormals, roots and isolating intervals, algebraic comparison and RCF transcendental representations from source |
| polynomial/LP/simplex/Groebner/etc. | manager identities and deletion, coefficient aliasing, polynomial operations/factoring/roots, exact feasible/infeasible LP instances and bounds, all source math-component regressions |
| AST/plugins | each kind, metadata, declaration constraints, interning/hash collisions, coercions, sorts, enumerations/datatype cycles, annotations and fresh symbols |
| binding/substitution | nested forall/exists, shadowing, multiple sorts, index shift under binders, capture avoidance, patterns/no-patterns, reverse binding order and source abstraction helpers |
| rewriting/normal forms | source normal forms and canonical ordering; proof/dependency links; semantic equivalence; ITE, arrays, BV, arithmetic, datatype, sequence and FPA rewrite rules |
| SAT | empty/unit/binary/ternary/long clauses, duplicate/complement literals, propagation, conflicts, learned/reduced/frozen/locked clauses, assumptions, user scopes, simplifiers/inprocessing/extensions, source model reconstruction |
| SMT/EUF | equality propagation and explanations, congruence, relevancy, theory attachment, backjump/restart, conflicting equalities/disequalities, combined theories and all scope transitions |
| arithmetic | rational/integer bounds, strict inequalities, difference/UTVPI variants, divisibility, mixed Int/Real coercions, integer cuts, nonlinear fallback and model reconstruction |
| arrays | select/store/const/map/as-array behaviors implemented by source, extensionality, nested arrays, array variants, proof generation and model evaluation |
| other theories | full BV edge widths/shifts/conversions; PB counting; datatype constructor/accessor/tester cases; sequence/string/regex byte/character cases; FPA bit-level cases; recursive functions |
| quantifiers | E-matching patterns/multipatterns, repeated matches/fingerprints, congruence-driven matches, trigger inference, instance generations, queue limits, skolemization, MBQI SAT/UNKNOWN/RESTART and auxiliary models |
| models | const/function interpretations, else/default values, universes, arrays, partial models, completion effects, old model retention and cross-manager translation |
| proofs | every constructor/rule/producer mapping, reference lifetime, checker rejection, side conditions, parameter decoding and proof/model converters |
| tactics/solvers | goal splitting and closure, proof/core/model converters, tactic failure/reset/cleanup, factory selection, translated/combined solver behavior, probes/tacticals and every registry entry |
| QE/projection | source guarded definitions, valid substitutions, existential equivalence in supported domains, MBP model containment and sound under-approximation rather than claiming full QE |
| fixedpoint | parser/rules/transforms, finite Datalog closure, CHC safe/unsafe/unknown outcomes, frame scopes, reachable facts, cover deltas, certificates, each registered engine and source cancellation behavior |
| optimization | hard/soft constraints, weighted MaxSMT, all objective priorities, exact optimum and bound encodings, unsat/unbounded/Pareto cases and source callbacks |
| API/CLI | all declared-and-implemented operations, invalid inputs, retained objects, error codes, supplied parser declarations, parameter descriptions, actual input dispatch, exits/diagnostics/log replay and semantic examples |

The source tests include named suites for algebraic numbers, `mp*`, interval/rational, arithmetic rewriters, substitution, bit blasting, API bugs, models, fixedpoint relations/tables/PDR/DDNF, nonlinear arithmetic, QE, LP/simplex and containers. Port each active assertion as its component arrives; the table does not substitute for the full test inventory.

### C5. Independent oracles and property-based testing

Use deterministic table/property tests plus Go's built-in fuzzing. `testing/quick` is suitable for simple scalar properties; structured solver input should use a reproducible bounded decoder/generator and a shrinkable byte-operation script. Do not add a property library solely to avoid writing a domain-specific oracle.

Every property states its domain, expected result and resource bound. Exhaustive checks over an explicitly finite domain prove that finite case; checks over a few values of an unbounded sort are only bug detectors. Do not report bounded integer sampling as a proof of arbitrary quantified LIA.

| Property target | Generator and independent oracle | Required invariant |
| --- | --- | --- |
| arena state machine | bounded allocate/free/borrow/reuse/compact scripts; simple map-of-copied-records model outside the allocator | live values/identities match; dead public/checked handles rejected; budget/free counts balance |
| region/trail state machine | push, mutation, nested allocation, pop, reset; deep-copy snapshot model and event log | exact pre-mark logical state restored; destruction/undo order correct; no retained reference to dead region object |
| numerical managers | signed decimal/limb operands and alias cases; `math/big` plus small direct arithmetic oracle | exact result, source division convention, normalization and destination alias equivalence |
| AST interning | bounded typed DAGs, deliberate hash collisions, retains/releases; independent structural serialization map | equal structures share identity while live; unequal structures never merged; RC roots and child counts remain valid |
| binding | well-sorted nested binders; independent named-variable interpreter converted separately to de Bruijn form | evaluation preserved by substitution/abstraction/renaming; no variable capture or free index |
| rewriting | small well-typed formulas/terms; truth tables, exact integers/rationals, finite arrays and bit-vector evaluation | source rewrite result is equivalent in the generator's complete domain; proof fact and dependencies agree |
| SAT | CNF with up to 8 variables/controlled clauses; enumerate all assignments | correct SAT/UNSAT, returned model satisfies input, incremental script agrees with active-formula oracle |
| finite EUF/quantifiers | Bool/explicit enumeration sorts, small functions/relations, bounded binders; enumerate all total interpretations | complete finite satisfiability agrees; models satisfy formulas; quantified binding and instantiation sound |
| BV | widths 1–4 for exhaustive cases and selected larger widths; masked integer evaluator | correct signed/unsigned ops, shifts/division edge cases, overflow and solving |
| arrays | finite explicit index/value domains, enumerate total arrays | select/store/extensionality and model/interpolant evaluation match complete semantics |
| cores | bounded SAT/finite-theory assumption sets; subset enumeration | core uses supplied assumptions/tracked assertions and is unsatisfiable with persistent assertions; minimality only when requested/implemented |
| LIA | planted inconsistent bounds and exact small linear systems; rational/Farkas certificates plus integer-specific arguments | source result/explanations valid; do not accept real feasibility as proof of integer feasibility |
| MBP/QE | formulas restricted to a domain with a separate evaluator/eliminator; explicit satisfying model | model lies in projection and projection implies existential input; QE equivalence only for exact QE routines |
| interpolation | planted contradictory typed partition trees; truth tables/finite interpretations or independent certificate checks | Craig obligations, allowed vocabulary, binding and tree/sequence result indexing |
| finite Datalog/CHCs | finite domains, small rule systems; naive exhaustive least-fixedpoint evaluator | queries and returned reachable facts agree; certificates are checked against original rules |
| optimization | finite Bool/BV/enumeration objectives and small bounded arithmetic instances; exhaustive feasible objective enumeration | exact optimum, priority behavior, soft weights and Pareto membership |
| parser/API | valid generated scripts plus malformed tokens/operation streams; independent AST serializer/evaluator and declaration environment | round-trip semantic preservation, appropriate errors, no corruption/leaked roots after failure |

For finite uninterpreted-sort tests, either use actual finite enumeration sorts or assert domain coverage and distinctness so enumeration is complete. Merely considering universes of size one or two for an arbitrary uninterpreted sort is not an oracle for full satisfiability. Prefer small explicit domains when generating quantified semantic tests.

The independent oracle establishes the logical answer, while the source engine can still be incomplete or resource-limited. In those fragments, check soundness of every decided result and record explicit expected UNKNOWN cases separately. Never count UNKNOWN as a successful proof of the oracle's answer, or silently allow it in a regression that is required to return a definite result. Generic properties must not impose a new completeness guarantee that the baseline solver lacks.

Preserve satisfying/unsatisfying witnesses across shrink. Useful metamorphic checks include renaming private symbols, adding a tautology, duplicating a premise, reordering clauses when semantics permit, push/assert/pop restoration, translating to a fresh manager, and forcing payload relocation. Metamorphic agreement is additional evidence, not an independent SAT oracle.

### C6. Interpolation tests: mandatory detailed matrix

For a binary result I, separately verify:

```text
A and not I is unsatisfiable
I and B is unsatisfiable
UninterpretedVocabulary(I) is contained in the allowed shared/background vocabulary
I is well-sorted and all variable indices are correctly bound
```

Vocabulary extraction walks declarations/sorts under binders; it must distinguish interpreted theory operators, uninterpreted function/predicate/constant declarations, sort declarations, generated local/skolem symbols and background-theory exceptions. Printed name intersection is insufficient. The vocabulary checker is independent of iZ3's scope code.

Include these unit fixtures with mathematically stated expectations. Assert semantic obligations rather than a single exact printed interpolant unless a source-conformance fixture specifically requires it:

| Fixture | Independently justified expectation |
| --- | --- |
| A=`PredA and PredB`, B=`not PredB and PredC` | UNSAT; shared vocabulary is only `PredB`; over all Boolean valuations I must be equivalent to `PredB` |
| A=`x<0`, B=`x>2` over Int | UNSAT; valid shared-x separator; validate arithmetic implication with an independent restricted arithmetic checker, not just another call to g3 |
| A=`x<=0 and a=x+1`, B=`x>=1` | private `a` does not occur in I; exact integer contradiction and Craig obligations |
| A=`a=b and f(a)=c`, B=`f(b)!=c` | private `a` eliminated; EUF consequence `f(b)=c` separates the partitions |
| A=`store(a,i,v)=b`, B=`select(b,i)!=v` | array read-over-write contradiction; no private `a` in I |
| A=`P(a)`, B=`forall x. not P(x)` | quantified interpolant may be `exists x. P(x)`; private `a` cannot escape; check finite enumeration instances exhaustively and retain a hand-justified general witness argument |
| A=`forall x. P(x)`, B=`not P(b)` | private `b` eliminated; universal/shared-P separator; de Bruijn and polarity checks |
| sequence `p`, `p=>q`, `not q` | two correctly ordered cuts; local sequence entailments and final inconsistency checked by truth table |
| branched tree with p/q leaves and a conflicting root | correct child-before-parent occurrences, per-node entailments and root false; also verify cut vocabulary |
| duplicated marker occurrence with shared interned subformula | output count/order tracks occurrences; no accidental DAG deduplication |
| proof with premises omitted from pattern | background premises affect entailment and allowed vocabulary exactly as the API specifies |
| satisfiable pattern | core reports SAT/model and no interpolant; GoIvy adapter returns its expected SAT-interpolation error |
| canceled/resource-limited query | UNKNOWN/reason/source error mapping and clean next-check recovery; no empty result mislabeled as successful UNSAT interpolation |
| invalid/non-Boolean marker or mismatched proof premises | appropriate API failure; no malformed AST published |
| unsupported proof/operator cases | match the fork's actual unsupported path; do not assert every BV/string formula must fail, since simplification can remove unsupported operators |

Additional obligations:

- Construct or obtain examples for each active `iz3translate` proof-rule branch and each quantifier/skolem/localization branch. Keep a branch coverage ledger tied to rule IDs.
- Test proof-enabled rewriting, resolution/lemma steps, congruence/equality explanations, arithmetic coefficients and array lemmas through the real producer path.
- Check every residual proof-checker side condition; a checker returning true with unchecked side conditions is not a certificate of the whole proof.
- Compare a produced tree/sequence against the source contract using node-local obligations, not only independent binary cuts. If the baseline exposes a defect, record it as a source issue and keep it visible; do not weaken the checker to make the port pass.
- Test nested quantifiers of different sorts, private functions, multiple local witnesses, skolem-origin heuristics, localization order, alpha-renaming and fresh-name collisions.
- Run with forced small arena pages, checked handles and safe-point compaction, retaining the refutation/interpolants while the input solver undergoes pop/reset or later checks.
- Cover interpolation I/O parent vectors, invalid cycles/order, sequences, background theory and scope-endpoint limits.

For unbounded quantified formulas, finite testing is supplemented by the ported checker and targeted mathematical arguments/certificates. The ported solver/checker share code ancestry; they are useful consistency checks but not fully independent proof of all first-order validity. Report that distinction in acceptance results.

### C7. GoIvy integration and conformance

Place reusable backend-independent formulas and expected results in `test_vectors/ivy/`. Translate the current `goivy/smt` unit expectations and bridge conformance cases into g3 semantic fixtures; do not make g3 tests import a neighboring GoIvy checkout.

The integration suite in GoIvy then runs its existing bridge atop the g3 adapter with:

- context/session-cache identity and multiple solver objects;
- enum native and Boolean encodings, nat/range guards, numerals and clamping;
- quantifier and interpretation translation in both directions;
- Herbrand/model mining, sort universes, relation/function values and small-model search;
- activation assumptions, soft deletions, core minimization and ordering;
- binary interpolation orientation, fresh interpolation contexts and conversion before close;
- canonical rendering, trace counters/Merkle inputs and direct SMT-LIB paths;
- native and wasm execution without the old CGo/Emscripten backend.

Keep exact client trace fixtures where output is contractual. For noncanonical solver models/cores/interpolants, use a documented semantic comparator; do not normalize away wrong binders, private vocabulary, missing proof premises or altered declaration identity. Record any address/runtime-only differences explicitly.

### C8. Fuzz infrastructure and regression promotion

Keep permanent seeds under `test_vectors/fuzz_seeds/`, load/embed them and call `f.Add`. Built-in Go fuzzing automatically uses `testdata/fuzz` for minimized failures and the Go cache for generated corpus; those files are ephemeral in this project. After a failure, promote the minimized input and independently verified expectation into `test_vectors/regressions/` plus a normal Go unit test, then load it as a seed. Do not commit permanent project fixtures under `testdata/`. [Go fuzzing documentation](https://go.dev/doc/security/fuzz/).

Use a fresh owner/context per fuzz input, deterministic seed/decoder, bounded formula size, finite operation counts and per-case resource limits. Negative inputs may produce typed API failures; invariant panics, data races, bad handles and solver corruption must fail the target. Do not blanket-recover every panic in a fuzz harness.

Suggested targets, implemented with their components:

```text
FuzzArenaOps                FuzzRegionTrailOps
FuzzMPZAliasing             FuzzRationalOps
FuzzASTInternRelease        FuzzBoundSubstitution
FuzzRewriteFinite           FuzzSATScopes
FuzzFiniteQuantifiedSolve   FuzzArrayFiniteSolve
FuzzBVFiniteSolve           FuzzCoreAssumptions
FuzzInterpolationPattern    FuzzInterpolationScopes
FuzzFiniteFixedpoint        FuzzOptimizeFinite
FuzzSMT2Tokens              FuzzAPIStateMachine
```

Implement custom shrinking/decoding that preserves typing and bound variable scope. If truncating a byte stream changes the intended property domain, decode into a valid smaller case or classify it explicitly as a malformed-input test. Re-run permanent seeds in ordinary `go test` and on wasm; run coverage-guided mutation on supported native runners.

### C9. Build, race and portability checks

Run phase-appropriate checks; do not rerun the full project suite after every low-level typo. At integration gates use:

```sh
# From golang/; commands are separate jobs, not a shell script to run blindly.
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 go test -tags=g3checked ./...
go test -race ./...
go vet ./...

# Portable package compilation and actual command artifacts.
CGO_ENABLED=0 GOOS=js GOARCH=wasm go build ./...
CGO_ENABLED=0 GOOS=wasip1 GOARCH=wasm go build ./...
CGO_ENABLED=0 GOOS=js GOARCH=wasm go build -o /tmp/g3-browser.wasm ./cmd/g3wasm
CGO_ENABLED=0 GOOS=wasip1 GOARCH=wasm go build -o /tmp/g3-wasi.wasm ./cmd/g3

# Select one fuzz target/package per job; bounded native fuzzing example.
go test -run='^$' -fuzz='^FuzzSATScopes$' -fuzztime=60s .
```

The race detector is a native development check and may require a host C toolchain; that does not introduce a production cgo dependency. Keep the production/test hermetic CGO-disabled jobs separate from the optional native race job. Also run checked handles with the race detector at suitable native integration gates.

Run `js/wasm` tests with the toolchain-matched `go_js_wasm_exec` wrapper and Node, using Go's official wasm execution setup. Run the WASI test binary with a pinned compatible runtime/wrapper; expose fixture files explicitly or embed them. Record toolchain/runtime versions. Include browser worker execution, deadline expiration, interruption policy, model/proof/interpolant serialization and repeated create/close. Native and wasm compare semantic answers and permanent fixture expectations. Do not assume identical timing or parallel schedules.

CI should include Linux/macOS/Windows native platforms as available, amd64/arm64, and a supported native 32-bit build/run where practical. Fixed integer widths and length checks need actual non-64-bit coverage, not just an amd64 test. Browser wasm and WASI are separate execution jobs. Use the repository's Go directive, with explicitly pinned alternate compiler versions only for compatibility testing.

Test fixture embedding keeps execution portable. `go test ./...` from a copy of `golang/` alone must pass the hermetic suites after the full port; maintainer source-inventory/reference-comparison tools are opt-in and clearly separated.

### C10. Performance, allocation and GC tests

Set performance baselines after the first correct vertical slice; do not claim a speedup from handle width alone. Record workload, source version/build, Go version, platform, parameters, seeds and cold/warm state. Any new C++ comparison still receives the independent Go regression required by C1.

Benchmarks must include:

- term interning and hit/miss mixes, small/high-arity nodes and quantifiers;
- reference release through deep/wide DAGs and repeated create/release in a long-lived manager;
- SAT propagation/conflict/reduction, both standalone SAT and legacy SMT;
- EUF merges/rollback, arithmetic rows/pivots, array propagation;
- E-matching/MBQI, model evaluation and repeated checks;
- proof production and quantified interpolation on realistic Ivy partitions;
- growing/reducing clause databases, payload compaction peak memory, and backtracking region churn;
- each major higher engine and numerical manager, including NLSAT/Spacer/optimization;
- parser/printing overhead, large symbols and public result retention;
- a long-lived GoIvy session with many push/check/pop cycles;
- native and wasm separately.

Measure `ns/op`, `B/op`, allocations, arena live/capacity bytes by domain, generation-ledger bytes, free/fragmented words, logical object counts, GC cycles/assist CPU, scannable heap bytes, peak heap, and relevant native RSS/wasm memory. Use `runtime/metrics.All()` to select available metric names; metrics are process-wide, so isolate benchmark processes and compare deltas. `/gc/scan/heap:bytes` reports scannable heap space, not a per-query count of actually scanned bytes. [runtime/metrics documentation](https://pkg.go.dev/runtime/metrics).

Performance acceptance targets:

1. No per-literal/term/clause allocation of separate Go heap objects in the hot representation. Page/payload growth is amortized; public cold wrappers are accounted separately.
2. A warmed bounded inner operation should allocate zero Go objects when it stays within preallocated pages/scratch. Growing workloads may allocate pages; do not assert zero allocations for an entire arbitrary solve.
3. Repeated logical deletion/backtracking with bounded simultaneously live state must plateau in arena live storage and reusable capacity, apart from explicitly measured generation/retired-slot ledgers and retained public results. No monotonic dead-AST payload growth hidden by context longevity.
4. After close/reset, logical roots and releasable page references return to the defined baseline. Do not assert immediate RSS reduction.
5. Scannable memory should grow primarily with owners/page directories/control objects, rather than every hot DAG edge. Validate the actual compiler/runtime layout with metrics and pointer-freedom tests.
6. Establish workload-specific speed/memory envelopes from measurements before enforcing CI thresholds. Regressions must explain whether source behavior, allocator layout, cache iteration, compiler escape or a new fixture caused them.

Use allocation profiles, CPU profiles and compiler escape diagnostics to find hot pointerful records, borrowed-slice escapes, hidden interface boxing and repeated buffer growth. First correct ownership/storage; only then tune page sizes, SoA splits, bounds-check-friendly loops and growth policies. A library must not change the embedding application's global GC percentage/memory limit or disable GC to make benchmarks attractive.

### C11. Completion checklist for an implementation task

Before a task is marked complete:

- every assigned source method/branch has a counterpart or recorded platform replacement;
- constructors, assignments, callbacks, destructors and vector cleanup have audited ownership;
- actual C++ assertions and applicable examples have finite Go tests;
- every reference experiment has a permanent hermetic regression with independent expectations;
- property domain/oracle/resource bound are stated, and failures are promoted to fixtures;
- checked and release handles agree semantically, including growth/pop/reuse;
- source proof/model/core reconstruction and unsupported/error behavior are preserved;
- tests appropriate to the component pass, plus its dependency integration gate;
- no fixture, generated table or ordinary test depends on C++/Python/network/sibling files;
- manifest, deviation record and source-linked comments are current;
- shared interfaces changed only through the coordinator; handoff identifies unresolved prerequisites;
- performance data is supplied for storage changes in established hot paths, without unmeasured speed claims.

Full-port acceptance additionally requires that the coverage ledger contain no unresolved production components/API operations, the GoIvy milestone pass including quantified interpolation, and all native/wasm execution and hermetic release checks succeed. A passing SAT subset or a successful wasm cross-build cannot close this project.

### C12. First implementation-session task packets

Use these concrete starting assignments once delegation is authorized in the implementation session:

1. **Coordinator / baseline owner:** build the complete manifest and source/API/conditional inventories; fix common naming, handle, numeric and budget contracts; scaffold `doc.go`, the main-package prefix conventions and checked/release helper signatures. Do not insert misleading solver stubs.
2. **Arena agent:** implement independent slot/slab/word/vector/region primitives, generation ledger, budgets and failure injection with state-machine oracles. Own only `arena/*` and agreed arena tests; domain destruction remains outside this package.
3. **Numerics agent:** implement `mpnum` from internal `mpn/mpz/mpq`, source assertion ports and independent arithmetic/aliasing properties. Report every source copy/global-manager adaptation. Its Go tests must not require the C++ source.
4. **Containers/test infrastructure agent:** implement source hashes/RNG/bitsets/needed containers and bounded SAT/finite-interpretation oracles with permanent fixture loading. Keep production random/hash algorithms distinct from independent test generation.

Run W1's gate, then assign AST ownership to one agent/coordinator and distribute plugin/test work from W2. Subsequent agents receive the explicit source and file sets from B20 and the component-specific obligations in C4–C6. This sequence lets later, lower-rigor sessions follow fixed contracts rather than rediscover memory semantics or improvise solver architecture.
