# Concrete S-expression prerequisite

Owned files: `util_sexpr.go`, `util_sexpr_test.go`, `util_sexpr_debug.go`, `util_sexpr_nodebug.go`, `util_sexpr_debug_test.go`, and this status. The three conditional files were explicitly approved after the initial packet. No existing source, manager, service, manifest, or progress files changed.

## Source scope and inventory

Read completely: `src/util/sexpr.h/.cpp`, `obj_ref.h`, `ref_vector.h`, applicable `vector.h` and the actual `buffer<char,false,16>` instantiation in `buffer.h`, `rational.h` arithmetic/formatting routes, and `util.h/.cpp::escaped`. There are no direct sexpr registered test calls or assertion sites in `src/test`. `main.cpp` registers `escaped`; `escaped.cpp` has thirteen print-only calls and zero assertions. The five nonnull default-trim/default-indent inputs relevant to sexpr are fixed-output fixtures here. The helper's null input, trim=true variants, and general indentation are not claimed as an independent complete escaped-helper port.

Actual `sexpr_ref_vector` specialization usage is SMT2 parser's `m_sexpr_stack`: push/get/back/subrange access/shrink/pop. `sexpr_ref` is declared as an `obj_ref<sexpr,sexpr_manager>` typedef with no additional concrete use found. The concrete wrappers nevertheless cover the complete applicable header surface. No other generic `obj_ref`/`ref_vector` specialization, ref-vector hash utility requiring `get_id`, or parser implementation is claimed.

## Method counterparts

| Source | Concrete Go counterpart |
|---|---|
| enum and base constructor | `SexprKind`, exact composite/numeral/BV/string/keyword/symbol order; records initialized with RC0 and explicit line/pos |
| inc_ref/get_ref_count/get_line/get_pos/get_kind | `IncRef`, `RefCount`, `Line`, `Pos`, `Kind` |
| all six is_* predicates | `IsComposite`, `IsNumeral`, `IsBVNumeral`, `IsString`, `IsKeyword`, `IsSymbol` |
| get_numeral/get_bv_size/get_symbol/get_string | `Numeral` (owning copy adapter), `BVSize`, `Symbol`, `String` |
| get_num_children/get_child/get_children | `NumChildren`, `Child`, `ChildrenInto` (caller snapshot replacing raw borrow) |
| display_atom/display | private `displayAtom`, safe explicit `DisplayAtom`, `Display` with defined-prefix error contract |
| manager constructor/all seven factories (two string overloads) | `NewSexprManager`; `MkComposite`, `MkNumeral`, `MkBVNumeral`, `MkString` preserving full bytes, `MkCString` NUL truncation, `MkKeyword`, `MkSymbol` |
| manager inc_ref/dec_ref/del | `IncRef`, `DecRef`, private `inc/dec/recycle` with source ascending child decrement/push and LIFO deletion |
| obj_ref constructors/copy/move/destructor/get_manager/m/get/arrow/deref/bool/not | `NewRef` (including null), `CloneRef`, `MoveRef`, `CloseRef`, `RefManager`, `GetRef`, `IsNull`; dereference through checked manager accessors |
| obj_ref assignments/reset/swap/steal/equality | `AssignRef`, `AssignRefFrom`, `ResetRef`, `SwapRef`, `StealRef`, `EqualRef` |
| ref_vector constructors/copy/move/destructor/manager | `NewRefVector`, `NewRefVectorFrom`, `Clone`, `Move`, `Close`, `GetManager` |
| reset/finalize/resize overloads/reserve/shrink | `Reset`, `Finalize`, `Resize`, `ResizeFill`, `Reserve`, `Shrink` |
| push/pop/back/size/empty/get(default)/[]/c_ptr/begin/end | `PushBack`, `PopBack`, `Back`, `Size`, `Empty`, `Get`, `GetDefault`, `CopyInto`, scalar `Begin/End/IteratorGet/IteratorNext` |
| set/erase(index or element)/contains/append overloads/swap(indices)/reverse | `Set`, `EraseIndex`, `EraseHandle`, `Contains`, `Append`, `AppendVector`, `SwapIndices`, `Reverse` |
| vector swap and set(vector) | `Swap`, `Assign` (source reset then sequential append, with partial prefix on failure) |
| element_ref assignments/get/arrow/deref/equality | scalar `Element` token; explicit-vector `GetElement`, `SetElement`, `AssignElement`, `EqualElement` and checked manager accessors |

All default source coordinates are represented by explicit `SexprDefaultPosition = MaxUint32`; callers pass line and pos. No implicit Go global source manager exists.

## Source rendering and conditional actions

Composite display uses the source iterative frame/index traversal. Empty composites print `)` including nested empty composites. The source top-level atom branch emits `display_atom` then lacks a return and casts the atom as a composite, which is undefined behavior. `Display` preserves the fully defined atom prefix and returns `ErrSexprDisplayContinuation`; no fabricated successful source continuation is claimed. `DisplayAtom` explicitly exposes the private source atom operation safely. Predictable rendering failures return the completed emitted prefix and error.

Numerals use basic `RationalService.FormatInto`, matching source rational `operator<<` calling `to_string`; pretty formatting is not substituted. A short caller staging buffer causes a budgeted growth and a repeated pure preparation via the existing public formatter API; numeric operand copies/arithmetic do not change. No intermediate Go strings are retained or allocated for rendering. Symbols borrow the live SymbolManager byte region under its lock, preserve null `null` and source signed numerical `k!` spelling. Strings stop display at their first NUL, insert backslash only before quote, and preserve backslashes/newlines/other bytes unchanged, as `escaped` defaults require. Their getter preserves the full stored bytes, including embedded NUL and UTF8 continuation bytes.

BV uses the exact function-local `sbuffer<char>` inline sixteen bytes. Heap overflow doubles sixteen→thirty-two→sixty-four with old-plus-new reservation, and is destroyed on success and every failure before the owned val destructor; no digits buffer is cached. A direct 95-byte budget rejects the 32+64 growth peak without mutating the old buffer. Fifteen hex digits plus NUL fit inline with no heap allocation, sixteen plus NUL spill to32, and thirty-two plus NUL spill to64. The production failure fixtures preserve prefix `#x` before any digit emission, and both source inline operation and final caller output allocate only the latter. BV display writes `#`, creates an owned rational copy, writes x/b, performs source Rem and IDiv result creation, Copy assignment and expression destruction, and retains c through digit-buffer push. It runs source hex comparison against rational9, reverses the buffer, pushes its terminating NUL before digit emission, and destroys val afterward. It never truncates values exceeding width. Negative values produce zero padding, width zero selects hex and can produce `#x`, and positive noninteger rem/idiv domains return the existing typed rational error after the completed prefix. Checked capacity arithmetic replaces unrepresentable physical buffers with typed bounds errors.

`g3debug` evaluates source `SASSERT(rational(0)<=c && c<rational(16/2))` after quotient publication and before c-to-digit branching. The lower test short-circuits the upper; the zero temporary survives RHS evaluation, and the RHS temporary is destroyed before zero at the full-expression end. The actual source rational comparisons and owned temporary constructors/destructors are used; false diagnostics return `ErrSexprBVInvariant`. Nondebug builds omit these actions, as the source does. The conditional arithmetic diagnostics are implemented, not left pending.

## Ownership, storage, and explicit adaptations

Factories return RC0. Composites acquire children in ascending order. `IncRef` retains exact uint32 wrapping: MaxUint32→0 does not delete. A subsequent `DecRef(0)` returns the typed source assertion error. Wrapped-zero records remain generation-valid until a later valid 1→0 deletion or explicit owner teardown. A nonallocating maximum-counter fixture verifies this.

Hot records, child handle pairs, element/iterator tokens and stack frames are pointer-free scalar structs. Cold pages, array directories, owner headers and all cleanup/display/staging/ref-vector backing arrays are charged with exact architecture sizes/capacities and old-plus-new growth peaks. A pre-reserved deletion stack bounded by allocated record capacity makes accepted cleanup of a valid acyclic RC graph allocation-free. Invalid cleanup after counter-overflow corruption can return typed errors; ordinary unsigned semantics are preserved. Record recycling increments generations and retires exhausted slots instead of reusing a public identity.

Record allocation precedes factory value construction, as the source does; failed constructors unwind copied numeral and payload resources and recycle their unpublished record. Physical pooled pages can remain charged after failure for reuse. SymbolManager and RationalService are borrowed and must outlive this manager. All numeral fields own a source Copy; accessors return owning copies rather than exposing mutable source raw borrows. Manager Close destroys every still-live owned numeral once, including RC0 and wrapped-zero records, before releasing physical storage. This is explicitly an owner-teardown adaptation: source implicit allocator teardown does not invoke outstanding node value destructors. Borrowed services are never closed by the manager.

`obj_ref` preserves null-aware callbacks, copy/assignment inc-before-dec, move/swap/steal without RC actions, and checked owner provenance. `ref_vector` source callbacks unconditionally dereference null manager arguments; the approved Go wrapper adapts them to no-ops, allowing source Reserve/Resize null placeholders followed by Set. Raw manager null callbacks still return typed errors. Source Reserve changes logical size. Capacity follows initial2 then (3*c+1)>>1. Vector growth preflights capacity before acquiring a reference rather than leaking an increment if source allocation throws; sequential multi-entry actions preserve accepted prefixes. Self-append of a nonempty vector would grow its own source loop indefinitely until resource overflow; Go returns typed bounds instead of inventing a finite self-append algorithm. Raw pointers, mutable c_ptr access and pointer iterators become checked scalar views/caller snapshots; structural mutation invalidates views. Vectors own their cold headers/backing independently and must be closed explicitly, including after manager teardown.

## Tests first and gates

Observed RED: missing manager/kind/factory/wrapper API in isolated HEAD+tests; subsequent UTF8 byte-copy behavioral mismatch; record-before-string-construction resource error priority; single finalization epoch boundary; missing conditional debug helper API, and wrongly allocated BV15digits-plus-NUL under zero available heap reservations. Faithful translations/corrections produced GREEN. No tests use production Go maps, generic code, or external C++ fixtures.

Fourteen base groups cover fixed six-kind source branches, factory RC/metadata and accessor copies, atom undefined continuation and empty lists, BV width/sign/domain quirks, actual deletion LIFO via slot reuse, scalar RC wrap, concrete refs/vector method branches and capacity/reserve semantics, independent finite 25-node DAG rendering/lifetime, embedded-NUL/UTF8 strings, seven complete allocation-failure sweeps, immediate public numeric cells/Words live counts before any String/Del/Close, independent BigInt rendering-prefix oracle, warm zero allocation/budget plateau, stale/foreign/generation/epoch failures, constructor prefix refunds, exact complete physical manager graph charge/refund, five relevant registered print inputs, scalar iterator/accessor behavior, and exact temporary BV inline16/doubling/terminator/cleanup boundaries.

Two g3debug groups cover lower/upper invariant boundaries for bases16 and2, lower-test short-circuit under FailAfter0, and upper comparison's genuine allocated-temporary failure prefixes with immediate cell/Words accounting. All owner teardown checks refund complete shared physical graphs. This adds no invented source assertion count: direct registered sexpr assertion sites remain zero.

Final isolated snapshot `.cache/release/sexpr-final` uses HEAD `d964d45` plus only the five owned Go files. All writes/build caches/temp/output stay beneath `golang`. Final amended gate results are recorded below after completion.

Amended frozen-byte gates (including actual source inline16 BV temporary buffer):

- Full native `go test -count=1 ./...`: PASS, root1.736s/mpnum10.090s.
- Full `g3checked g3debug` module tests: PASS, root1.706s/mpnum9.929s.
- `g3checked g3debug` race, all sixteen Sexpr groups: PASS,1.401s.
- Fresh actual `g3checked g3debug` JavaScript/Wasm execution: all sixteen groups individually RUN/PASS,0.248s.
- Full checked/debug vet: PASS.
- Checked/debug WASI compilation: PASS; no WASI runtime claim.
- `go run ./cmd/portcheck`, including the no-generics syntax audit: PASS.

All defined sexpr header methods and applicable concrete reference specialization methods are represented. Defined release and BV diagnostic C++ algorithm branches are implemented; the undefined top-level atom continuation remains explicitly unsupported rather than purportedly repaired. Platform pointer addresses, allocator layout, and unsafe raw borrows have the checked scalar/storage adaptations described above.

## Parent integration verification

The exact frozen five Go files were integrated with committed parameters and
formatter dependencies in `.cache/release/sexpr-buffer-debug-integrated`.
Full CGO-disabled native and full g3checked+g3debug module tests passed.
All sixteen named groups explicitly RUN/PASS under native checked-debug race
and actual checked-debug JavaScript/Wasm execution. Full vet and WASI build
passed. Final Go bytes matched the snapshot hashes; source ledger/fingerprint
and no-generics audit passed separately. WASI is a build check, not a runtime
claim. The broader solver/AST/interpolation implementation remains open.
