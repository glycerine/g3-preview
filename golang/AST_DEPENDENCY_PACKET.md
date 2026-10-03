# Concrete AST dependencies and persistent dependency arrays

Selected configuration: `dependency_manager<ast_manager::expr_dependency_config>` and `parray_manager<ast_manager::expr_dependency_array_config>` from `src/util/dependency.h`, `src/util/parray.h`, and the actual AST manager wrappers in `src/ast/ast.h`/`ast.cpp`. These are real AST callbacks, not the region-backed `scoped_dependency_manager<void*>` used by old intervals/Grobner.

## Source methods and public contracts

`ASTDependency` is a pointer-free, comparable owner-qualified allocator token. Fresh leaves and joins have source RC0; callers retain them explicitly with `DependencyInc` and release with `DependencyDec`. Null is source empty. Public methods are `DependencyEmpty`, `DependencyLeaf`, `DependencyJoin`, `DependencyJoinValues`, `DependencyInc`, `DependencyDec`, `DependencyRefCount`, `DependencyIsLeaf`, `DependencyContains`, `DependencyLinearizeRaw`, and `DependencyLinearize`. Leaf null is the AST wrapper fast path, rather than a generic null-valued leaf. Join null/equal fast paths do not increment. The 30-bit counter wraps exactly; zero DEC is a source debug assertion, with defined release unsigned underflow.

The actual constructor, leaf/join allocator/callback order, iterative LIFO destructor, shared FIFO queue, root/child marking order, early-found unmarking, and raw linearization of dependency identities are implemented. Distinct leaves containing the same expression remain distinct dependencies. AST-level linearization then uses the real `ASTFastMark1` and source `remove_duplicates`: mark, optional in-place `set(j)`, advance j, final shrink. It deduplicates the entire preexisting output, preserving stable order and completed compaction prefixes on failure. No staging/replay/maps are used in production.

`ASTDependencyJoinProgress{Completed, PendingLeaf}` exposes the actual loop prefix when constructing the next leaf succeeds but allocating its join refuses. Both returned nonnull RC0 roots remain owned by the caller; no silent DEC of the unjoined leaf occurs. Each nonnull input is checked at its actual iteration; a later foreign or stale token returns the preceding completed leaf/AST INC prefix. The borrowed input slice is never persisted.

`ASTExprVector` is the actual non-owning expression-pointer output specialization: `NewExprVector`, `ExprVectorPush`, `ExprVectorGet`, `ExprVectorSize`, `ExprVectorReset`, and `DelExprVector`. Shared scalar codecs with the reference vector invoke no reference callbacks here. A caller must keep expressions alive separately; output tokens are not owning AST references. Output owners require explicit NewExprVector and must not be shallow-copied or coerced into an owning reference-vector token; the full generic ptr-vector copy API is not claimed. Persistent-array owners likewise use explicit Copy, not Go scalar assignment. ExprVectorGet checks actual AST lifetime. Additional generic vector methods are not claimed.

`ASTDependencyArray` is an owner-qualified scalar with source update counter. The full selected persistent-array branches are implemented: `DependencyArrayMk`, `Del`, `Copy`, `Size`, `Empty`, `Get`, `Set`, `SetFrom`, `Push`, `PushFrom`, `Pop`, `PopFrom`, `Unshare`, `Unfold`, and `Reroot` (all names have the `DependencyArray` prefix). Internal ROOT/SET/PUSH/POP encoding, RC30, both scratch vectors, copy-values callbacks, source requested growth `0->2->(3c+1)/2`, ROOT preservation, update threshold `>size`, trail `>16`, and reroot split `size/2` follow the source. Its value manager is the actual ASTManager, whose callbacks route to its dependency manager. The source `manager()` accessor is represented by the existing containing manager; no borrowed raw callback owner is exported.

The selected configuration is reference-counted with preserve_roots=true; false/reference-disabled/different value-manager configurations remain separate. Public logical Get/Set assertions are debug-only. Release can read a retained initialized pointer or construct a nonroot SET at an arbitrary uint32 index. Source empty nonroot POP can publish a POP with index0 and subsequent Size UINT32_MAX; debug rejects at !empty. Root/null empty-pop physical undefined behavior is guarded. Copy-only Get returns a retained scalar even if its dependency lifetime has ended; actual dependency queries/callbacks then diagnose an invalid handle rather than dereferencing a source dangling pointer. No universal logical bound replaces these distinctions. At an invalid retained old callback, rset preserves the incoming INC completed before the lifetime guard; tests recover it explicitly. Generation validation remains strict after a lifetime ends, even if storage is reused; raw source transparent pointer replacement/ABA is not emulated.

## Physical ownership and failure adaptations

All hot records contain scalars, not Go pointers. The existing manager header charge automatically includes the two actual embedded owners and their source scratch-slice headers once. Both main and formatting managers bind them eagerly. No new allocation is invented by empty owner construction.

Selected source 64-bit leaf/join sizes are16/24; charged Go encoded sizes are24/32. Each has an actual8-byte intrusive continuation link. Source flags/RC are encoded at0, value/children at8, continuation at16/24. This additional physical backing is an explicit allocation-exception cleanup adaptation. Ordinary deletion grows the source LIFO todo only at its real push, after decrement. On first reserve refusal, already queued entries and the failed entry are linked in exactly the same LIFO order; the same finite deletion completes without another reservation, freeing every dependency and performing every AST DEC. The first error is propagated after completion. AST emergency destructor errors similarly consume ownership while finishing the remaining dependency graph. No all-live scratch preflight, replay, ignored allocation refusal, or orphaned lexical node substitutes for this.

The FIFO traversal queue is a real charged concrete slice, with complete old/new backing overlap and full-capacity scalar copying. Its source unsigned32 allocation-byte bound and actual host backing-byte bound are checked separately before reserve; arena.ErrBounds is the concrete checked overflow/memory-domain adapter, not a claim to the source default_exception class. An explicit unsafe source allocation-exception adapter clears the completed dependency marks (including a root marked before an initial push refusal); appended output remains a source completed prefix. This is distinct from FastMark1's source defect: when its mark-before-buffer-push refuses, the current unlisted AST mark remains dirty after lexical Close. It is not repaired by linearization. Tests demonstrate explicit caller recovery.

Dependency-array cells retain the source selected24-byte scalar layout; value locators are8 bytes. Source backing has an8-byte capacity header; actual Go backing charges12 bytes including initialized high-water metadata. Growth copies all old capacity bytes without inventing initialization. Only actual writes/logical copy construction advance initialization. Logical copy-values increments only copied values. Physical/initialized/allocator uint32 bounds diagnose source invalid memory access; the actual12-byte limit differs from source8. Shared-root reversal publishes before the requested operation, preserving normal allocation/refcount prefixes. Reroot PUSH storage preparation before destructive union retagging is the existing explicit source-unsafe exception adapter, not a second algorithm. Temporary rematerialized blocks/cells are cleaned lexically on failure; consumed callback errors finish publication and clear consumed caller tokens. Owner/provenance checks guard Go raw-handle misuse before numeric/domain callbacks.

Manager Close follows existing forced AST teardown before allocator/member reclamation. Escaped dependency/array/output records are bulk reclaimed without callbacks against freed ASTs; retained todo/get/reroot backing is then refunded exactly once. Constructor-prefix cleanup also handles unbound empty owners. Borrowed symbols/numerals/allocators are never separately closed by the embedded managers.

## Tests-first evidence and source attribution

Initial complete actual API test produced isolated compilation RED (session24055, HEAD07a6182 plus new test, undefined dependency/array/output APIs). Six initial tests plus constructor binding passed after implementation (22351). Added source RC30 release regression produced behavioral RED50797, then native/checked-debug GREEN86342. Constructor-prefix test exposed a real nil-owner teardown panic (30215), fixed and GREEN97826. Empty nonroot POP release regression was RED14808, then full targeted release/checked-debug GREEN17998. Later-input prevalidation was caught RED26908 and corrected to preserve prior source prefixes (GREEN11382); invalid retained-old rollback was caught RED31080 and corrected (GREEN93275). Scalar source/host scratch-bound test was missing-helper RED before the concrete helper (GREEN62650). Full gates below supersede intermediate development runs.

There are **zero registered direct AST-valued dependency or dependency-array source assertions**. Added tests cover actual AST callbacks, source FIFO/LIFO action order, duplicate values/shared DAGs, independent finite reachable-value sets, aliases, real failure prefixes, physical allocator live counts before Close, constructor teardown, and warm allocation-free raw traversal/output reuse. Stable dedup uses the separately audited real cold FastMark owner; its actual heap header allocation is charged and is not claimed allocation-free.

`TestASTDependencyArrayAdaptedRegisteredParrayAssertions` ports the three active `src/test/parray.cpp` preserve-roots=true action/condition bodies, substituting retained dependency values for ints. It executes **15+213+128=356** conditions. The source original int configuration is reference-disabled with max_trail8; this actual AST specialization is reference-enabled with max_trail16. These are explicitly **adapted shared-template assertions**, not original specialization completion. The false-mode calls, disabled tst4, original million-update expr-array tst5/heap diagnostics, and TRACE pointer display are not newly claimed by this packet.

## Remaining boundaries

`expr_dependency_ref`/its obj_ref specialization, dependency ref-vector/ref-buffer aliases, region-backed scoped dependency configurations, native pointer-address `display_info`/TRACE/ostream state, and original int/void* dependency consumers remain open. The AST dependency manager has no source Reset; none is invented here. Full AST/proof checking, solver, polynomial/Grobner/interval integration, cancellation/context propagation, and externally registered consumer ports remain independent unfinished work. Existing source units remain partial where these configurations/adapters are outstanding.

## Final verification

The initial prospective tree was HEAD92c5d0a:golang plus only the13 owned Go files, at `.cache/release/ast-dependency-final`, with22 new groups plus the updated eager constructor-binding test. Those historical gates are recorded below; the correction appendix supersedes their frozen hashes and final-tree evidence. No central metadata or Git edits are part of this packet.


Frozen Go verification (all terminal PASS):

| Gate | Session | Result |
| --- | --- | --- |
| Full native `go test -count=1 ./...` | 99985 | g3 2.402s; mpnum22.256s; all packages PASS |
| Full `g3checked,g3debug` module tests | 96461 | g3 2.888s; mpnum23.008s; all packages PASS |
| Named checked-debug race,22 new + eager binding | 65402 | all23 RUN/PASS; 1.297s |
| Actual checked-debug JS/wasm,22 new + eager binding | 36331 | all23 RUN/PASS; 0.278s |
| Checked-debug `go vet ./...` | direct terminal | PASS |
| Portcheck including no-generics syntax audit | 44439 | PASS; committed ledger baseline33 ported/78 partial/1991 unported |
| Checked-debug WASI root test compilation | 14688 | PASS; compile only |
| Checked-debug WASI full-module `go build ./...` | 20925 | PASS; compile only |

Initial accidental WASI compilation in the active workspace (83630) failed on unrelated SAT draft missing APIs; it is not a packet gate. The isolated snapshot excludes drafts and passed both authoritative WASI compilation gates. All writes/artifacts remained below golang. No live gate sessions remain.

Ordered SHA256 inventory (13 Go files frozen throughout final gates):

```text
e287b426cfbb22b930029b962d935a1457c7d62b3d7fc0664c700a13f25b9484  ast_core.go
af04f558a564b6ae08c6fff25f80c07b9f27f1c525fe8f1cbec56815786065f5  ast_core_dependency_ctor_test.go
47fcfe3d368a05184adbbc9ec4c34cf109cf2220cd173dd576b62ff8cb40a061  ast_core_storage.go
df227a19ab62faca0a0b17b811585d00d5bec103504c70a4ca713198ca773edb  ast_dependency.go
f2340921362abc7150ae5a21b637cca00f9f0b9f35bb9052f4028463dfe84336  ast_dependency_api.go
96ff3f1f6890d1169ab2350b892dc0efc454a14aee118e8db4cbb65a20b50a0b  ast_dependency_array.go
d1d8b6931c07a098c75eced88ac6a0dd2aafd9961a9bd8e97fa45c81ac7d565c  ast_dependency_array_api.go
a13f3f825db0edc0b5d970ec604f8a6ba22500788825fc56eda6620086985772  ast_dependency_array_test.go
42fe64e431882d66fd9db8a9c5d9af56fc259fd5d97550b3d254b58fac49e9bd  ast_dependency_array_values.go
2701cf978ea7973284563857f22ef5119033eeec7c572fef6b2ecdcbeb0895a3  ast_dependency_failure_test.go
9351797a34352042e634e6d8b31ebc341121490590aa05b439d5619ca7282121  ast_dependency_test.go
73983ac29d620c0ec8a4745140fae49eb02bcb6a87836eeee9b01dd70dc1b83a  ast_dependency_traversal.go
a0a7726ba057f259fd7dbce62cfbcbe5f8d99820cbd4dc6d9616016375d90e31  ast_dependency_vector.go
```


## Source-order and refusal correction

The final corrected prospective tree is HEADa6b3cc3:golang plus the same13 owned Go files at `.cache/release/ast-dependency-corrected`. Only `ast_dependency_array.go`, `ast_dependency_array_api.go`, `ast_dependency_array_values.go`, `ast_dependency_failure_test.go`, and this document were thawed. No central metadata or Git changes are included.

`copy_values` now writes each destination scalar before executing its actual INC, matching parray.h198–204. A failed INC does not confer ownership of that current scalar; lexical cleanup decrements only the preceding successful INC prefix. Read/write guard failures also release that prefix and its temporary backing. A valid root's logical slots are initialized, so later raw-read refusal is not an ordinary source path: the additional malformed-metadata regression explicitly checks this safety adapter. Its behavioral RED45202 reported copied-reference count2 instead of1 before teardown; corrected GREEN19714 passes.

A dependency DEC refused by its source debug RC0 assertion now reports consumed=false. This keeps the root SET slot intact at the source assertion rather than publishing its replacement. The regression seeds the independently tested MAX30 boundary, then calls actual array Push to perform the source wrapping INC to0; it does not invent a callback. Debug RED55414 reported the incorrectly replaced slot; corrected GREEN50444/19714 passes. Release preserves unsigned0→MAX30 decrement and replacement. Both fixtures check immediate physical ownership before Close. There are now25 new test groups plus the existing eager binding test; the source adapted-condition total remains356.

### Explicit remaining debug diagnostics

The selected release algorithms and public logical/RC0 assertions are implemented; this packet does **not** claim every internal parray SASSERT. Remaining exact diagnostic sites include capacity-allocation equality (line41), private cell kind accessors (72–75), some internal next/reference-count checks and helper ROOT/capacity/null-target assertions (128,169,177,184,194,199,215,253), construction/unshared/update-counter/reference-count postconditions in Copy/Set/Push/Pop (281,326,357–358,376,386,410–411,417,428,438,460–461,467,480,488,507), and Unshare/Unfold structural postconditions (511–512,520).

In particular Reroot's exact source assertions after unfold (`c` ROOT), at each inverse step (`c` ROOT, `p` nonROOT, `p->next==c`), and at its final publication (`c==r.m_ref`, ROOT, size==initial r_sz, then counter reset and r.root()) remain open at parray.h541,546,549–550,585–589. Existing typed switch/lifetime/storage guards are not equivalents for these source diagnostic checkpoints. Source finite inverse PUSH/POP index quirks remain preserved in release; no claim of complete malformed-graph debug diagnosis is made. Registered original dependency-specialization assertion count remains zero.

Copy-only `DependencyArrayGet` now matches ast.h1958: release returns the retained raw scalar without an invented dependency dereference. Actual RefCount/IsLeaf later validate its ended lifetime; debug logical Get refuses before copying the retained slot. Regression RED68137 demonstrated the extra lifetime guard.


### Corrected frozen verification

All corrected Go files are byte-identical between the workspace and isolated HEADa6b3cc3 snapshot. Final full native62943 PASS (g3 2.308s, mpnum21.447s); full g3checked,g3debug30332 PASS (g3 2.746s, mpnum22.923s); named26 checked-debug race87039 PASS1.261s with all RUN/PASS; actual named26 checked-debug JS97161 PASS0.244s with all RUN/PASS. Checked-debug vet terminal PASS; WASI root test compilation50565 and full-module build3980 terminal PASS (compile only). No-generics/portcheck12515 terminal PASS: committed ledger baseline33 ported/82 partial/1987 unported. No live gate sessions remain. The unrelated active user-scope draft compilation failure before the isolated RED is not a packet gate. All artifacts stayed below golang.

Corrected final13 Go SHA256 inventory (supersedes the historical inventory):

```text
e287b426cfbb22b930029b962d935a1457c7d62b3d7fc0664c700a13f25b9484  ast_core.go
af04f558a564b6ae08c6fff25f80c07b9f27f1c525fe8f1cbec56815786065f5  ast_core_dependency_ctor_test.go
47fcfe3d368a05184adbbc9ec4c34cf109cf2220cd173dd576b62ff8cb40a061  ast_core_storage.go
df227a19ab62faca0a0b17b811585d00d5bec103504c70a4ca713198ca773edb  ast_dependency.go
f2340921362abc7150ae5a21b637cca00f9f0b9f35bb9052f4028463dfe84336  ast_dependency_api.go
eb38137cbb91c52e8bd22fdfa107a015446b73c4df07ffeb4b9e699d48cef769  ast_dependency_array.go
1f5adb304a1fec587dfc8bc3cd9e93191912e6b7ff30702401d4e0b093f8ee66  ast_dependency_array_api.go
a13f3f825db0edc0b5d970ec604f8a6ba22500788825fc56eda6620086985772  ast_dependency_array_test.go
ce77106c1f53803e1f3ece65a92ebc84161c51bdf07135d71b4e982691846cdd  ast_dependency_array_values.go
98af9d9a84b63f43ca17d5b1f6b5ddc9a85ebbad90480d8ce5a684a064903564  ast_dependency_failure_test.go
9351797a34352042e634e6d8b31ebc341121490590aa05b439d5619ca7282121  ast_dependency_test.go
73983ac29d620c0ec8a4745140fae49eb02bcb6a87836eeee9b01dd70dc1b83a  ast_dependency_traversal.go
a0a7726ba057f259fd7dbce62cfbcbe5f8d99820cbd4dc6d9616016375d90e31  ast_dependency_vector.go
```

## Coordinator integration

Coordinator read the selected dependency, persistent-array and actual manager
wrapper source, plus the concrete production/API/traversal/ownership changes.
Independent cross-review found write-before-INC order, debug RC0 consumption,
and an omitted debug diagnostic inventory. Coordinator additionally required
copy-only retained Get to return its raw token before a later actual dereference.
The corrected packet keeps those source effects and explicitly identifies the
remaining private parray/Reroot SASSERT sites, rather than counting typed
handle guards as complete source diagnostics.

All13 live Go files match the final prospective tree and final ordered hashes.
The full native/debug, vet and WASI gates above use HEADa6b3cc3 plus exactly
those13 files. Coordinator captured fresh logs at
`.cache/review/ast-dependency-integration-a6b3cc3/.cache/`: exactly26 RUN/PASS
names were independently verified under checked/debug race (1.270s) and
actual JavaScript (0.232s). Final metadata/source fingerprint/no-generics and
declaration checks passed at33ported/84partial/1985unported. Only dependency.h
and the selected ast_util.h dedup helper become partial; other touched AST and
parray units remain partial. These gates verify the isolated committed chunk;
concurrent solver and marking drafts are separate ongoing work.
