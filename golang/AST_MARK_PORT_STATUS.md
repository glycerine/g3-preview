# Concrete AST marks packet

Selected declared mark APIs are complete and frozen for coordinator integration. This packet contains six Go files and sixteen additional finite test groups. It does not claim whole `ast.h`, `ast.cpp`, `obj_mark.h`, or `bit_vector` completion.

## Source and method map

| Source surface | Concrete Go counterpart |
| --- | --- |
| `util/obj_mark.h`: default `obj_mark<expr>` constructor; reference/pointer `is_marked`; `mark(flag)` and `mark(true)` overloads; `reset` | `NewASTExprMark`, `IsMarked`, `Mark(h, flag)` (`true` selects the no-flag overload), `Reset`, `Close` |
| `ast.h`: `ast::mark1/mark2`, `reset_mark1/reset_mark2`, `is_marked1/is_marked2`, including actual `Z3DEBUG` ownership assertions | `ASTManager.MarkNode1/2`, `ResetNodeMark1/2`, `IsNodeMarked1/2`; node constructor initialization is the existing real AST core |
| `ast_fast_mark<1/2>` constructors/destructors, `is_marked`, `reset_mark`, both `mark` overloads, `reset`, `get_level`, `set_level`; expr typedef aliases | `NewASTFastMark1/2`, `Close`, `IsMarked`, `ResetMark`, `Mark`, `Reset`, `GetLevel`, `SetLevel` |
| `ast_ref_fast_mark<1/2>` constructors/destructors and all declared methods; expr typedef aliases | `NewASTRefFastMark1/2`, `Close`, `IsMarked`, `ResetMark`, `Mark`, `Reset`; actual manager INC/DEC callbacks |
| `ast.cpp:3232–3250`: `ast_mark::is_marked/mark/reset` plus virtual destructor | `NewASTMark`, `IsMarked`, `Mark`, `Reset`, `Close`; independent expr/declaration bitmaps |
| `ast.cpp:3258–3294`: both `scoped_mark::mark` overloads, `reset`, `push_scope`, both `pop_scope` overloads; constructor and empty derived destructor | `NewASTScopedMark`, `Mark`, `Reset`, `PushScope`, `PopScope`, `PopScopes`, `Close` |
| Instantiated `bit_vector` subset required by these `obj_mark`s | Private budgeted `astMarkBits`: default construction, exact resize(false)/get/set/reset/destruction |
| Instantiated `ptr_buffer<ast,16>` and `ast_ref_buffer` operations | Actual cold-owner inline16 scalar array, heap doubling, ASC reset/destruction; no separate inline slice or per-entry object |

The expression map uses actual Expr IDs. The AST declaration map uses unsigned `get_id() - (1u << 31)` and the expression map remains separate. Marks survive node deletion and ID recycling as explicitly warned by source `ast_mark`.

## Storage, ownership, and failure actions

All owners bind a real borrowed ASTManager. Headers (including named inline16 arrays), complete bitmap word capacities, fast-buffer overflow, scope limits, and real scoped ASTRefVector metadata/payloads are charged to that manager's budget. Growth reserves the whole replacement while old storage remains charged, then refunds the old storage. Bitmap capacities follow source `((num_words(new_size)*3+1)>>1)`; fast buffers double 16→32→64; scope limits follow unsigned-vector 2→3→5 growth. Reset retains backing.

Header reservation and owner-identity preparation now occur **before** the escaping Go cold object is allocated. The final allocation fixture proves all seven constructors refuse FailAfter(0) with zero Go allocations.

Ordinary ID maps and nonref fast lists do not retain AST references. Ref-fast lists retain actual nodes. Fast marking sets the node bit before list insertion, and ref-fast increments before list growth. Failed unlisted mark/INC prefixes remain visible; Reset/Close cannot clear/decrement an entry that was never inserted. Explicit caller balancing remains possible. `reset_mark` and `mark(false)` clear a fast bit without decrementing its retained reference or removing a list entry.

Ref-fast Reset clears **all** node marks ascending before the real ref-buffer decrements nodes ascending. Its completed release order is tested through actual recycled IDs. Scoped marking pushes an owning stack entry before growing/setting its bitmap; retries repeat source INC and push, retaining completed prefixes. Scoped pop clears bitmap entries ascending, pops the limit, then resizes/decrements the actual stack. PopScopes preserves completed scopes before a later underflow guard. Scoped Reset clears declaration/expr maps before resetting the owning stack and then limits. Scoped Close follows source reverse member destruction: limits, stack, base declaration bitmap, base expression bitmap. It does not invent a Reset call.

## Explicit Go adapters and remaining source scope

* Checked origin/slot/generation handles replace source raw pointers. Foreign, null, or freed handles fail safely. Nonref fast lists retain checked borrowed generations; stale-borrow Reset/Close reports a typed error and leaves the owner open until forced manager teardown permits physical cleanup.
* Cold owners are caller-confined. Operations that touch the manager hold its mutex. Private `newASTFastMark1Locked`, `markLocked`, `nodeMarkedLocked`, `resetMark`, and `closeLocked` permit real dependency traversal under an already-held lock, with the same algorithms and budgeted owner.
* A unique scalar owner identity replaces the debug `this` pointer comparison. Assertions execute only with `g3debug` at the actual source mark/read/reset sites; release ignores owner assertions. Raw low-level APIs accept an opaque uint64 owner token.
* Implicit owning C++ copy/assignment is not implemented in this selected packet. Go owner copies are explicitly rejected by self identity; declaration/bitmap ID semantics are not changed to generation keys.
* Source uint32 resize arithmetic and completed publication are preserved even at unsigned wrap. Source-undefined physical accesses return typed errors. A behavioral RED showed a later physical read panic after retained resize publication; the final guard reports ErrBounds without erasing that prefix.
* After forced ASTManager Close, marker Close refunds only its external cold/backing storage and avoids callbacks against the freed AST graph. Consumed deletion failures use the core's owned emergency continuation; these source unsafe-exception continuations are explicit adapters.
* `expr_sparse_mark`, the shared-occurrence private mark/functor, other generic `obj_mark` BV/functor instantiations, implicit source copies, and whole AST/util units remain open. Bit-vector TRACE diagnostics remain part of the pending whole bit-vector surface.

No registered mark-specific test was found in `src/test`. `src/test/bit_blaster.cpp` uses ast_mark during real traversal; that integration remains dependent on its actual BV plugin. The sixteen tests here are additional coverage, not invented registered source assertions. They include all five real node kinds, a bounded independent boolean oracle, source growth/ownership order, deletion/ID reuse, immediate physical accounting at constructor failure prefixes, partial scopes, stale/foreign owner guards, and warmed zero-Go-allocation operations under FailAfter(0).

## Observed RED and final GREEN

1. Initial HEAD07a6182 + only mark tests: missing agreed constructors and node mark APIs, compile exit 1 (session42058).
2. Locked helper test: missing `newASTFastMark1Locked`, compile exit 1.
3. UINT32-wrap continuation test: actual index-out-of-range panic (`[134217727] with length 2`), behavioral exit 1 (session96776); physical guard added after retaining source resize publication.
4. Header-refusal allocation test: constructor0 allocated one uncharged header before FailAfter(0), behavioral exit 1 (session76132); all constructors changed to reserve before allocating.

Final immutable snapshot: `.cache/release/ast-mark-final-a18`, committed **a18b6ce** + only the six listed Go files. It excludes evolving dependency/core bindings and solver drafts; proof code is included from committed HEAD. All caches/temp remained beneath golang.

| Final gate | Evidence |
| --- | --- |
| Full native `go test -count=1 ./...` | exit0 session16389; `native.log`; root package2.528s, mpnum21.357s |
| Full actual checked/debug `go test -tags 'g3checked g3debug' -count=1 ./...` | exit0 session67264; `debug.log`; root package2.668s, mpnum21.710s |
| All16 named checked/debug race groups | exit0 session17878; `race.log`; sixteen explicit RUN/PASS, root1.180s |
| Actual JS/wasm execution, checked/debug | exit0 session54249; `js.log`; sixteen explicit RUN/PASS, root0.219s; real `.cache/js-exec`, `-count=1 -v` |
| Checked/debug vet all packages | exit0; `vet.log` |
| WASIp1/wasm checked/debug build all packages | exit0 session44376; `wasi.log` |
| Actual reference source inventory/fingerprint and no-generics audit | exit0 session61430; `source-audit.log`; `go run ./cmd/portcheck -source /mnt/oldrog/home/jaten/go/src/github.com/glycerine/g3` |

Final inventory at snapshot HEAD: 2102 source/tool units, 677 APIs, 33 ported, 77 partial, 1992 unported. This packet did not edit central metadata.

## Frozen Go hashes

`AST_MARK_SHA256SUMS` digest: `0a579a5f42e4512a987f883f5a8ca2e0b161212ba44b5c091d41b92b86f75dfc`.

```text
7ca1ebda8779a54a5c53f4eb989226f1ad71c877e74befc796d08b9fef0e58ed  ast_mark.go
162fc2ac3684029a1a9b42a23b3090c7fbb6a4894466ce8969d641354fd2ac80  ast_mark_doc.go
224f9b7f9d0468545fdfa1c8c1e68f732bcc2d0ea4d30ba46e31e066b2b01963  ast_mark_fast.go
a5b6490693b0848749ccf80797858b9ec4a5170a717b0a1b2f4e3d804f6c97fd  ast_mark_node.go
8ca548efc83ca0ad0e3c3038a97502f03990708885fa6daa559392d749658a7f  ast_mark_scoped.go
397474631ce3e95b72d2049fa44c8a28d2c4853c7ed671bc2f3d0b1d1b8d54c0  ast_mark_test.go
```

## Coordinator integration

Read-only source/implementation review found no further defined-path mismatch
after the constructor allocation-before-reservation correction. Latest snapshot
.cache/release/ast-mark-integration-9fcbeff includes committed expression-array
retained-storage correction9fcbeff plus these exact6 Go files. Full native
(root2.307s/mpnum22.909s) and checked/debug (root2.753s/mpnum23.236s) passed.
Final declaration-evidence/no-generics audit passed2102 units/677 APIs:
33 ported,78 partial,1991 unported. Only obj_mark.h moves unported→partial;
its other type/BV/functor specializations remain open. Whole AST/source audit
flags and API coverage remain unchanged.
