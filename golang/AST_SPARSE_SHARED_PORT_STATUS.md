# Sparse AST marks and shared occurrences

Selected packet is implemented and integrated after coordinator review; six new Go files, twenty-one additional finite test groups. Source registered tests contain no `expr_sparse_mark` or `shared_occs` assertion fixture. These tests do not increase the registered-source assertion count. Whole AST, general object-table templates, and printer coverage remain open.

## Source methods and concrete counterparts

| Source | Concrete Go counterpart |
| --- | --- |
| ast.h `expr_sparse_mark` constructor | `NewASTExprSparseMark`: eager identity table capacity 8 |
| `is_marked`, unary/bool `mark`, `reset`, table destruction | `IsMarked`, `Mark(h,true)` / `Mark(h,flag)`, `Reset`, `Close` |
| shared_occs.h `shared_occs_mark` constructor/destructor | `NewASTSharedOccurrencesMark`, `Close`; local default-operator marker uses stack inline storage |
| `is_marked`, `reset_mark`, unary/bool `mark`, `reset` | `IsMarked`, `ResetMark`, `Mark(h,true)` / `Mark(h,flag)`, `Reset` |
| shared_occs constructor | `NewASTSharedOccurrences`, `NewASTSharedOccurrencesWithConfig` |
| private `insert`, `process` | actual private `insert`, `process` |
| both `operator()` bodies | `Collect`, `CollectWithMark`, private `collectLocked` |
| `is_shared`, `num_shared`, begin/end iteration | `IsShared`, `NumShared`, `BeginShared`, `EndShared`, concrete iterator operations |
| `reset`, `cleanup`, destructor | `Reset`, `Cleanup`, `Close` |
| `display` | OPEN: requires real `mk_ismt2_pp` / AST SMT2 printer; no substitute rendering |

Defaults are exactly TrackAtomic=false, VisitQuantifiers=true, VisitPatterns=false. Application children visit ascending; quantifier visits body, patterns, no-patterns. Parent frame cursor increments before processing each child, and frames are reread after growth. An atomic early return does not drain a preexisting release-mode stack. Source debug empty-stack/quantifier-visit assertions execute with g3debug, separately from handle checks. Reset does not clear the frame stack; Cleanup finalizes it.

The actual `obj_hash_entry<expr>` / `core_hashtable` specialization uses zero FREE, scalar sentinel DELETED, current node hash on every USED hash read, pointer-identity equality adapted to checked slot/generation, strict >75% growth before duplicate lookup, ascending move-table scan, wrapped linear probing, last-tombstone insertion, immediate-next FREE erase, and cleanup only after the DELETED erase branch. Its resource OOM skip uses the manager's constructor-bound resource callback (also bound in the real ID owner). No cached hashes, AST ID bitset, Go maps, or alternative table algorithm occur. Constructor, actual used find/contains/insert/insert-if-absent/erase/reset/finalize/rehash and physical iteration paths are represented. General inherited copy/assignment/swap/union/intersection/find-pointer/dump, optional HASHTABLE_STATISTICS, and remaining generic diagnostics are not claimed complete by this private specialization. Default source statistics are disabled.

## Ownership, layout and failure actions

All owners are confined to one real ASTManager. Sparse keys and visitation lists borrow nodes; they do not increment RC. Shared set publication precedes the unique real `incNode`. Reset invokes real DEC in ascending physical buckets, then table.reset through the collection helper equivalent, then the second source table.reset. Destruction resets first, then frees stack backing, then table backing. Visitation resets flags ascending and frees overflow after reset. The separate actual node `shared` flag never uses public mark1/mark2 or their debug owners. Failed visitation push leaves the already-published bit unlisted; Reset cannot silently clear it.

Cold headers reserve before constructing escaping Go objects. Actual reflect-sized headers, every 8-byte scalar entry/list element, every target-sized frame including padding, and all unused array capacity are charged. Marker inline16 is physically in its charged owner; local default visited inline16 is lexical stack storage with no self-slice. Overflow doubles 16→32→64. Frame svector uses actual 2→3→5… capacities. Its scalar growth helper preserves source unsigned32 capacity arithmetic, independently checks the selected LP64 pair extent16 plus two unsigned32 header words, and checks actual Go frame stride12 against the host allocation extent. Boundary fixtures exercise the exact source extent endpoint without allocating a large array. Ordinary replacement reserves new while old remains charged. Generation handles are eight bytes on both targets, an explicit physical adaptation of source pointers, not a claimed source pointer ABI.

Reset clears entries, refunds/deletes old backing, publishes halved capacity BEFORE allocation, then clears counts only after success. Finalize deletes old backing, allocates64, and publishes64 AFTER allocation. A scalar pending phase with nil backing preserves old count prefixes and these different capacity prefixes; reads reject the source undefined nil-entry access. Retry completes the physical replacement without repeating already-consumed DEC callbacks. A separate release cursor retains actual callback consumption when core deletion reports a failure after its owned emergency continuation. Its behavior is verified immediately using live node counts before retry/Close. No fake retained old backing is charged. Scalar epochs invalidate borrowed iterators only when the backing is replaced/freed. In-place insert, erase, and nonshrinking reset preserve the captured current position and endpoint, including at epoch exhaustion. A cleared current slot returns source null and an erased DELETED slot returns an explicit non-node sentinel scalar; dereferencing that token is an invalid AST domain. Actual relocation exhaustion returns typed bounds errors before old storage is touched, and a pending replacement can finish after its invalidation consumed the final epoch. Failed relocation reservation leaves the previous iterator alive.

Checked origin/generation/self guards replace invalid raw-pointer / implicit owning-copy domains. Implicit copies of cold owners are rejected, rather than exposing double-destruction. Explicit Close is required. After forced manager Close, external owners refund only their own physical storage and skip callbacks against freed AST nodes; iterators reject that lifetime. Iterator and scalar count views are owner-confined. These are Go lifetime/failure adapters, not source extra ownership or cleanup actions.

## Tests-first evidence and finite coverage

Initial isolated HEAD92c5d0a snapshot `.cache/release/ast-sparse-shared-tdd` observed missing-API RED, exit1 (session79239), before production. Iterator API missing-RED exit1 also preceded its implementation. Behavioral RED→GREEN regressions captured FREE-successor erase incorrectly attempting cleanup (session93307), the earlier structural-epoch exhaustion guard (28367, later revised to backing-only exhaustion), pending reset requiring an extra exhausted epoch (39526), and iterator access after forced manager Close (35718).

Twenty-one groups cover source defaults/multiplicity, all8 traversal policies, current hash and exact last tombstone, independent bounded bool-array membership oracle, ID reuse versus pointer identity, actual physical iteration and bucket-order deletion/LIFO ID reuse, frame reserve/cursor prefix, source shared-bit-before-list-growth, free-before-reserve reset/finalize, consumed deletion failure before retry, epoch limits, forced lifetime teardown, and warmed/constructor-refusal zero host allocations. Coordinator review exposed the initial blanket structural epoch adaptation as overrestricting valid entry-pointer lifetimes: corrected in-place iterator and final-epoch regressions observed behavioral RED32645. The source16/actual12 frame-bound helper observed missing-API RED before implementation. Successful and failed actual backing relocations, raw table erase, reset-null current slots, and captured endpoints are independently checked. A further sort/declaration facade regression observed RED41405: expression-only sparse Mark/IsMarked and collector IsShared now use the actual requireExpr guard, while the same test verifies AST-wide SharedOccurrencesMark accepts and marks both node kinds. No C++ runtime fixture or external testdata is required.

## Frozen verification

Isolated snapshot `.cache/release/ast-sparse-shared-final4` is committed HEADd4da3755bf33343698fc3f298876d5e1de31d3f5 (including real dependency owners) plus ONLY this packet's six Go files. Root-owned live drafts are excluded. Logs are `.cache/release/ast-sparse-shared-final4-logs/`. This final run supersedes the earlier17-group pre-review gates and intermediate20-group HEADa6b3cc3 and HEADd4da375 runs.

- Full module fresh native PASS; root2.454s, mpnum22.214s.
- Full module fresh actual g3checked+g3debug PASS; root2.738s, mpnum22.806s.
- Fresh named g3checked+g3debug race PASS: twenty-one RUN and twenty-one PASS, root1.092s (`race.log`).
- Actual named checked+debug JavaScript wasm execution PASS: twenty-one RUN and twenty-one PASS, root0.203s (`js.log`); original subtree TMPDIR/cache and `.cache/js-exec` used.
- Checked+debug whole-module vet and WASI build PASS (exit0).
- Source/no-generics audit PASS (exit0):2102units/677APIs,33ported/84partial/1985unported, reflecting committed metadata before this packet integration.

Immutable Go hashes:

```
0089a3fed0a28c0e875039f9450018c1844af4ea00ad137ad167f2ce4bc0bf66  ast_sparse_mark.go
4bceb5bba8eeb6db1203526da820faee93d2ce9f31064a04a6f9317db5d77dee  ast_sparse_mark_test.go
511fcc44dc55e448c2775e1874ca7c01ce6addda75b9ca67f9230dd3d8fcb2c6  ast_shared_occs.go
cec0f969060624f4663d7e626ef00de11dff96ae67afd9b217a6df4fe83b8d88  ast_shared_occs_mark.go
fc6f3f302814f102e619033a844b37471976c27a8deb130f194afdbbceb9a6f1  ast_shared_occs_iterator.go
0810502fba1adfb8ad02d392e6fd12b69f7ec8aa333e05e31da7576343b215bc  ast_shared_occs_test.go
```

All source, tests, logs, caches, snapshots and temporary files in this packet remain below golang. No existing core/marks/proof/dependency source, central metadata, or Git files were edited.

## Coordinator integration verification

The six hashes above were checked against both live files and final4. The prospective committed tree `.cache/review/ast-sparse-shared-integration-472e79b` contains HEAD472e79b plus only these six files and the final metadata; unrelated owning-reference/model/full-solver drafts are excluded. Fresh full native tests PASS (root2.346s, mpnum22.548s); fresh full g3checked+g3debug tests PASS (root2.742s, mpnum23.502s). The coordinator independently verified every one of the twenty-one exact RUN/PASS names in the immutable final4 race and actual JS logs. Vet/WASI gates remain unchanged-file evidence from final4; WASI is compilation only. Final source/no-generics inventory records33ported/87partial/1982unported units, with whole-source method/ownership completion flags still false for the newly partial units.
