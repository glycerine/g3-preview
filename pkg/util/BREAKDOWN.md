# Breaking `util` into focused Go packages

This document records the original analysis of direct C++ sources in
[`src/util`](../../src/util) and the corresponding Go port. The package split
described below was completed on October 10, 2026. The original source graph
and survey counts remain as historical context.

## Completed Go package boundaries

There are no Go files directly in `pkg/util` and no Go consumers import the
old root package. Its implementation and tests now live in 36 utility packages.
All utility production imports stay within the Go standard library, external
libraries, `arena`, and other utility packages; none imports a solver package.

The final extractions are:

| Package | Responsibility |
| --- | --- |
| `util/numeral` | Concrete integer, rational, binary and floating-point managers, ownership and invocation state |
| `util/params` | Parameter values, descriptors, catalog and global service |
| `util/diagnostic` | Assertions, debugger actions, standard streams, fatal policy and allocator diagnostics |
| `util/text` | Go-managed text, numeric streams and shared borrowed C-string boundaries |
| `util/memory` | Runtime accounting, allocation, release and concrete storage borrows |
| `util/hash` and `util/hash/runtime` | Pure project hashing and concrete runtime-backed reads |
| `util/sexpr` | S-expression values and their numeral/symbol ownership |
| `util/smt2` | SMT2 symbol quoting predicates and formatting |

Numerals depend on memory/text/hash; parameters depend on numerals, symbols,
text and diagnostics. Diagnostics have no parameter dependency. Consumers use
native slices/maps, `insdict.Dict` v0.10.1 where insertion order is observable,
and Go text builders/buffers. No root forwarding package was retained.

The full native module suite, checked and combined debug/checked compilation,
focused debug suites, vet and manifest validation passed after the final move.
This completes the package reorganization, not the outstanding C++ algorithm
port. Source coverage and completion credit remain in `port_manifest.json`.

## Original analysis

Splitting `pkg/util` is feasible and worthwhile. The C++ include graph has a
small, strongly connected infrastructure core and many feature units layered on
top of it. Most of those feature units can become Go packages without importing
one another. The split should proceed from those outer units inward.

The numerical implementation is the largest useful boundary. It accounts for a
large part of the current Go source and tests, but its Go ownership model is
more tightly coupled than the C++ include graph suggests. Move it first as one
cohesive package, after extracting the few low-level facilities it needs. Do not
try to split `mpn`, `mpz`, `mpq`, and `mpf` into separate Go packages in the
same operation.

Each live functional leaf should become a Go package leaf, even when small.
That makes consumers visible in the import graph and gives the facility an
enforceable boundary. Package by functionality rather than mechanically making
one package per C++ header. If a facility has no production C++ consumer,
delete or defer the port instead of giving dead code a new home.

The extraction work now includes `util/filepath`, `util/approx`,
`util/messages`, `util/natset`, `util/stack`, `util/bitset`, `util/luby`,
`util/statistics`, `util/control`, `util/prime`, and `util/text`.
The bitset
package owns the native variable-width, fast-reset, fixed-width, and unsigned
set implementations. It imports `arena` directly for fixed-width pooled
storage and keeps its source-compatible hash routine private beside its two
hash consumers. The unsigned set exposes typed ascending `ForEach` traversal;
the copied C++ iterator and hash-functor wrappers were removed. The redundant
`Runtime`/`Block` bit-container translations were deleted after a module-wide
consumer scan found no production user. The standalone `bit_util` translation
and its copied source tests were subsequently deleted for the same reason: the
live numeric ports already use `math/bits`, typed slices, and their concrete
arithmetic. `obj_mark` already lives in its concrete typed AST specializations,
so no generic utility wrapper remains to extract.

`util/checkedint` now contains the concrete checked `int64` specialization.
Overflowing multiplication uses native checked arithmetic, so the leaf no
longer imports rational ownership or numeric stream machinery merely to test
whether a product fits. Unchecked source instantiations use plain Go `int64`.

## Method and terminology

The graph was built from quoted includes in every direct `.h` and `.cpp` file
under `src/util`. An edge `A -> B` means a file belonging to logical unit `A`
includes `util/B.h` or another direct source belonging to `B`. A header and its
same-stem implementation file are treated as one logical unit. The existing
`src/util/lp` subtree is excluded because it is already a separate component.

There are two useful meanings of *leaf*:

* A **foundation leaf** has no outgoing edge to another direct `util` unit. It
  is at the bottom of the include graph.
* An **extraction leaf** has no incoming edge from another direct `util` unit.
  Nothing else in `src/util` depends on it, so it can be moved outward without
  first moving another `util` unit. Code elsewhere in Z3 may still consume it.

The second definition is the useful one for incremental package extraction.
The first identifies the primitives that must eventually sit below the other
packages.

This is a conservative source dependency analysis. Includes may exist only for
an inline function or a rarely instantiated template. Conversely, Go files can
have same-package dependencies that do not appear in C++ because the port uses
shared ownership and helper types. Every actual move therefore still needs a
Go compile check and focused tests.

## Graph size

| Measure | Result |
| --- | ---: |
| Direct C++ `.h` and `.cpp` files | 177 |
| Same-stem logical C++ units | 126 |
| Direct internal include edges | 275 |
| Strongly connected components | 116 |
| Nontrivial strongly connected components | 1 |
| Foundation leaves | 27 |
| Extraction leaves | 73 |
| Direct Go files in `pkg/util` | 944 |
| Production Go files | 413 |
| Go test files | 531 |

The 11-unit cycle is:

```text
buffer, debug, hash, hashtable, memory_manager, str_hashtable,
trace, util, vector, warning, z3_exception
```

This is the infrastructure knot. `str_hashtable` should not be reproduced in
Go: Go strings and maps replace it. Its presence in the C++ cycle explains the
C++ graph, not a desired Go abstraction.

The knot should be dissolved rather than moved intact. Most of its members are
C++ implementation machinery that Go already supplies. The C++ classes can
disappear, while the few observable behaviors they carry move into small,
one-way Go packages. The detailed replacement analysis is in
[Dissolving the 11-unit knot](#dissolving-the-11-unit-knot).

The highest fan-in units also show where package boundaries will be expensive:

| C++ unit | Direct `src/util` consumers |
| --- | ---: |
| `vector` | 38 |
| `util` | 28 |
| `debug` | 23 |
| `trace` | 15 |
| `memory_manager` | 14 |
| `hash` | 11 |
| `z3_exception` | 10 |
| `buffer` | 9 |
| `rational` | 9 |
| `hashtable` | 7 |
| `mpq` | 7 |
| `z3_omp` | 7 |
| `mpz` | 6 |
| `symbol` | 6 |

## Recommended package layers

The dependency direction should remain one-way:

```text
arena
  |
  v
memory-limit policy and true shared primitives
  |
  +-------------------+-------------------+
  v                   v                   v
numeric            symbols/params      independent algorithms
  |                   |                   |
  +-------------------+-------------------+
                      v
                  Z3 consumers
```

A root `util` compatibility facade would be dangerous if child packages also
imported it. That immediately creates cycles. During migration, move complete
facilities and update their consumers; do not make child packages depend on a
large parent package that re-exports them.

### 1. Independent algorithms and value types

These are the best early boundaries. Several implementations are already
self-contained in Go or depend only on `arena` or the standard library.

| Proposed package | C++ units | Recommendation |
| --- | --- | --- |
| `util/lbool` | `lbool` | High-value first move. The value type has no internal C++ dependency and already has many Go consumers. |
| `util/mincut` | `min_cut` | Self-contained live algorithm. Make it a package leaf even though its `muz` consumer has not yet been ported. |
| AST-local topological sort | `top_sort` | The generic Go adapter is gone. Implement the sole live `sort*` specialization beside the AST declaration collector when that consumer is ported. |
| `util/natset` | `nat_set` | Self-contained live representation and a natural package leaf. |
| Approximate numerals | `approx_nat` | Keep its numerical behavior with concrete numeral consumers; the unsigned `approx_set` now lives in `util/bitset`. |
| `util/backtrack` | `trail`, `union_find`, selected backtracking containers | `union_find` is widely used upstream. Keep trail ownership and undo behavior together. Specialize `backtrackable_set` for its actual consumer instead of keeping `any`. |
| `util/bitset` | `bit_vector`, `fixed_bit_vector`, `uint_set`, unsigned 64-bit `approx_set` | Extracted. Native typed storage; fixed-width vectors import `arena` directly. The concrete `ApproxSet` preserves residue collisions, ascending traversal, size and display. `bit_util` is a Go platform replacement and `obj_mark` is represented by concrete AST specializations. |
| `util/cliques` | `max_cliques`, its set helpers | Extracted. Typed slice rows and bit sets preserve the live source algorithm; SAT imports the leaf directly and the Runtime-backed duplicate is gone. |
| `util/heap` | live concrete `int32` specialization of `heap` | Extracted. Native typed slices hold values and inverse indices; Runtime/Block and diagnostic-emulation branches are gone. SAT retains its separately specialized activity queue for later conversion. |
| `util/idgen` | native `id_gen` | Extracted. A private typed `[]uint32` preserves LIFO reuse and imports only `arena`; polynomial, AST, and SAT use it directly. Runtime-backed owners preserve sticky OOM transport around typed growth without raw ID blocks. Dead hash/output diagnostics and the duplicate root generator are gone. |
| `util/random` | `random_gen` and concrete shuffles from `util.h` | Extracted. The typed source-compatible generator has no root dependency; SAT imports it directly. Its deterministic recurrence remains observable in seeded solver decisions. |
| `util/escaped` | `escaped` from `util.h/.cpp` | Extracted. Standard `io.Writer` and `strings.Builder` preserve live formatting behavior; the custom string buffer and its internal allocation-failure surface are gone. |
| `util/prime` | `prime_generator` | Extracted. Plain typed `[]uint64` slices replace the copied vector shape while retaining the source prime sequence, growth schedule, and budget-failure prefixes. The eager default binding and borrowed iterator live beside the generator; polynomial factorization imports the leaf directly. |
| `util/sortnet` | `sorting_network` | The Runtime-backed unsigned specialization was test-only and is deleted. The live concrete AST specialization stays in `ast`; revisit a leaf only when another ported production consumer can share a non-generic API. |
| `util/permutation` | live free operations from `permutation.h` | The polynomial coefficient/monomial cycles live in their concrete `math/polynomial` caller. The unused uint32-only adapter is deleted; port the distinct NLSAT clause-pointer cycle with its retained consumer. `permutation.cpp` has no production caller. |
| `util/stack` | `stack` | Put the arena-backed implementation in a package that imports `pkg/arena` directly. |

The `lbool` boundary is now direct: SAT, tactic, API, AST, and model code import
`util/lbool` themselves, and the root `util` alias facade has been deleted. The
unused root `mincut` facade was deleted as well; its implementation remains only
in `util/mincut`.

The extracted `util/backtrack` package also stands on its own now. Its root
`Trail` and `UnionFind` alias facades had no production consumer and were
deleted; the concrete implementations and tests live only in the leaf.

The root version-constant facade was likewise unused and has been deleted;
generated version values live only in `util/version`.

The `z3_exception` boundary now lives in `util/z3exception`. All current root
utility, AST, SAT, and SMT consumers import it directly. Ordinary Go error
values, wrapping, and caller-side formatting replace the source virtual
exception hierarchy and varargs/iostream constructor; immutable strings own
message storage.

The concrete `ScopedFastFloatVector` port was deleted after a full source scan
found that `scoped_mpff_vector` is instantiated only by `src/test/mpff.cpp`.
No production Z3 or Go code consumes it. Production scalar `scoped_mpff`
behavior and tests remain; vector capacity, allocator-layout, and retained-tail
tests do not justify shipping a test-only C++ container specialization.

Very small leaf facilities such as `file_path`, `luby`, and `version` should
still become distinct Go package leaves. This makes their consumers visible in
the import graph, gives them an enforceable boundary, and makes later deletion
straightforward. Package size alone is not a reason to leave unrelated code in
root `util`. These three leaves have now been extracted.

The former `any`-based Go versions of `list`, `scoped_vector`, and
`backtrackable_set` have been removed. Their live C++ instantiations belong to
SMT consumers that have not yet been ported. Implement those concrete element
types beside their consumers with typed nodes, slices, maps, or `insdict.Dict`
at v0.10.1 when stable iteration is required. Do not restore a generic root
adapter.

### 2. Symbols, parameters, statistics, and control

These should follow after the first independent moves:

* `util/symbol` now owns symbol identities, native Go spelling interning,
  cached symbol hashes, the explicit initialization/finalization service, and
  the live concrete int32 dictionary. Its only nonstandard dependency is
  `arena`; it imports neither root `util` nor any numerical package. Consumers
  import it directly, without a root compatibility facade. Further live
  `symbol_table` specializations belong here when their consumers are ported.
* `util/params` can own `params` and environment parameter parsing and import
  `util/symbol` if necessary.
* `util/statistics` now owns `statistics.h/.cpp`. String keys, typed slices,
  ordinary writers, and the approved insertion-ordered dictionary replace the
  C-string, vector, arena, and hash-table simulations. The package has no root
  `util` dependency. The separate source `stats` unit can join it when ported
  if its consumers need the same boundary.
* `util/stopwatch` now owns `stopwatch.h` and its platform clock adapters.
  It depends only on the standard library. Timer and time-reporting consumers
  exchange a typed `State` snapshot when they must persist the watch; all other
  consumers use `Stopwatch` directly. There is no root compatibility facade.
* `util/cooperate` now owns the selected single-threaded `cooperate.h/.cpp`
  profile and imports only `arena`. Current consumers import it directly.
* `util/control` now owns cancellation, event handlers, resource limits,
  timeouts, timers, and scoped control-C handling. The leaf imports only
  `arena` and `util/messages`; AST, SAT, tactics, and polynomial fixtures import
  it directly, with no root compatibility facade.

These APIs have broad consumers, so moving them will touch many packages. They
are valuable boundaries, but poor first experiments for the mechanical package
move process.

### 3. Memory-limit policy, without a C++ allocator model

Do not retain a general C++-style allocator, raw block, initialized-byte bitmap,
or vector-layout model as a foundation package. Native Go objects, slices, maps,
and [`pkg/arena`](../arena) should own actual storage.

Some source behavior remains visible: configured memory and allocation limits,
peak/current usage reporting, out-of-memory classification, and fatal-versus-
returned failure policy. If these behaviors are required by a ported consumer,
put only that policy and accounting in a small leaf such as `util/memlimit`.
Callers can reserve and release semantic amounts through `arena.Budget`; they
should not expose C++ addresses or allocation headers.

Exact C++ `malloc` sizes and failure points cannot be reproduced naturally by
the Go runtime. Treating them as a conformance surface forces every native data
structure back into a C++ memory simulator. Tests for externally visible limit
and failure behavior should remain; tests whose sole purpose is to reproduce an
internal C++ allocation layout should be retired as the relevant owner is
converted.

Do not turn the 11-unit C++ cycle into a permanent `util/base` package. That
would reproduce the current junk drawer under another name.

## Dissolving the 11-unit knot

The proposal is viable. All 11 units can disappear as C++ abstractions. A few
behaviors must survive in much smaller Go-native leaves, but none requires the
old dependency cycle.

| C++ unit | Go replacement | What, if anything, must survive |
| --- | --- | --- |
| `buffer` | Concrete slices for typed scratch storage; `[]byte` or `bytes.Buffer` for mutable bytes; `strings.Builder` for text | Explicit cleanup only when elements own a real resource. The inline C++ storage optimization, placement construction, capacity header, and destructor template disappear. |
| `vector` | Concrete slices with `append`, `copy`, and type-specific helpers | Preserve an algorithm's logical size and order. Do not preserve the universal vector class, growth formula, allocation header, or iterator classes. |
| `hashtable` | Built-in maps for unordered lookup; `insdict.Dict` v0.10.1 when traversal must be stable | Insertion order is the accepted stable replacement for C++ bucket order. Custom equality becomes a comparable key, stable ID, or canonical key. Cached hash slots, tombstones, probing, and rehash code disappear. |
| `str_hashtable` | `map[string]struct{}` | Nothing else. Go strings own their bytes and have value equality. |
| `memory_manager` | Go allocation, slices/maps, `arena`, and an optional small `memlimit` policy package | Preserve configured limits, usage statistics, and error/fatal policy where consumers expose them. Remove raw blocks, allocation headers, byte-initialization maps, placement construction, and C++ realloc emulation. |
| `debug` | Ordinary condition checks, `panic` or returned errors, and an optional tag set | Preserve user-facing assertion/debug toggles only if live consumers need them. Do not port debugger invocation, deliberate segmentation faults, macro expansion, or arena accounting for a tag map. |
| `trace` | A narrow `io.Writer`-based tracer, the later cross-language tracer, or no code when tracing is disabled | Preserve tag filtering and emitted records only where tracing is a supported feature. The lazy global C++ table and `.z3-trace` stream mechanics disappear. |
| `util` | Split functions among their consumers and standard packages such as `math/bits`, `io`, `strconv`, and `strings` | Preserve verbosity/fatal policy as small explicit services if live. There should be no Go package that corresponds to the miscellaneous C++ `util.h`. |
| `warning` | `fmt.Fprintf` to an `io.Writer`; `bytes.Buffer` or `strings.Builder` only when a complete message must be assembled first | Preserve enable/disable, destination selection, prefixes, and write errors if observable. Delete the C `va_list` formatter and custom character vector. |
| `z3_exception` | Go errors, sentinel errors, wrapping, and a small typed error-code value | Preserve public Z3 error codes and messages. Delete the C++ exception hierarchy and formatted exception constructor. |
| `hash` | Keep explicit Z3 hash computations beside their consumers | Go `map` hashing replaces hashes used solely to implement a table. Retain hashes that are observable data or guide an algorithm, without creating a hash-table or `utilhash` package. |

`bytes.Buffer` is not a general replacement for `vector<T>` or `buffer<T>`.
Use it for byte streams. A concrete slice is the direct replacement for typed
contiguous storage, and `strings.Builder` is appropriate for write-only text.
`bufio.Writer` supplies buffering around an `io.Writer` when output batching is
the actual requirement.

### Map order and hash behavior

Built-in Go map iteration is deliberately unspecified. Use it when an algorithm
only needs membership or lookup. Use `insdict.Dict` when a C++ table traversal
needs a stable order. Its insertion order does not reproduce the source table's
physical bucket sequence, but exact bucket order is not a porting requirement;
stability is. Do not retain a custom open-addressed table merely to reproduce
that incidental sequence, and do not sort keys merely to imitate it. Sorting is
still appropriate when a separate external format requires canonical order.

Likewise, `hash/maphash` and the runtime's internal map hash do not replace an
explicit Z3 hash value. They are intentionally seeded and their values are not
available as a stable source-compatible result. Keep operations such as
`Combine`, `Mix`, string hashing, and composite hashing in `util` while their
consumers remain there, or move each operation beside its consumer when that
consumer moves. Do not create a generic hash-table or `utilhash` package. Hash
helpers used only by deleted custom tables should be deleted.

### Current Go obstacle

The existing port has already spread the C++ storage model well beyond the
11 source units. At this audit point:

* 128 production files in `pkg/util` mention `Runtime` or `Block`.
* Outside `pkg/util`, `util.Block` appears in 82 production files and
  `util.Runtime` in 37. `util.Block` has 487 selector uses in those files.
* `TraceContext`, `WarningContext`, `PrintfFormatter`, `MemoryProcess`,
  `DebugGlobalService`, and `VerboseContext` currently have no production Go
  consumer outside `pkg/util`.
* `UintMap`, the direct custom-hashtable specialization, has only two production
  consumers outside `pkg/util`.

### Implementation progress

The direct `UintMap` specialization has now been removed. Its three live uses
were replaced according to their actual semantics:

* the API object registry uses `insdict.Dict[uint32, uint32]` because object
  destruction traverses the registry and needs a stable order;
* the floating-point power caches use `insdict.Dict[uint32, uint32]` because
  cache teardown traverses their entries; and
* the AST debug freed-ID set uses `map[uint32]struct{}` because it only performs
  membership checks.

The probing array, tombstones, cached hash values, iterator implementation, and
tests of the C++ table layout have been deleted. This removes one member of the
infrastructure knot rather than moving it into a child package.

The standalone `RuntimeUnsignedVector` wrapper has been deleted as well. Its
only production field was a permanently empty datatype-utility member; active
AST vectors already have concrete typed owners. The general Runtime column now
keeps only the growth-owner interface used by those concrete implementations.

The unused `UintPtrMap` template stand-in has also been deleted. Although the
source `uint_map<T>` template has live C++ instantiations, the Go wrapper had no
production consumer and stored every pointer as `any`, then used reflection to
recover a weak runtime type check. Each future consumer should use its concrete
pointer slice directly; the source operation is only indexed slice growth,
lookup, clearing, and reset.

Nine more unconsumed template stand-ins have been removed for the same reason:
`ScopedVector`, `PtrScopedBuffer`, `ScopedPtrVector`, `RefBuffer`, `List`,
`BacktrackableSet`, `TopSort`, `Pool`, and the related `RefVector` hierarchy.
They exposed `any` throughout their APIs and had no production Go caller. This
does not classify their upstream C++ facilities as dead. With the exception of
`ptr_scoped_buffer`, which is used only by a C++ test, they have live consumers
that must receive concrete Go implementations when those consumers are ported.
The natural homes are the consumer packages: sequence theory for its three
scoped-vector element types, AST for declaration topological sorting and AST
reference buffers, SMT for its quantifier backtracking set and enode pool, and
the relevant math packages for their owning pointer slices. Existing Go AST and
polynomial reference buffers are already concrete examples. Ordinary typed
slices and maps should replace the remaining source templates; no dynamically
typed root-util abstraction is needed.

The unused `ref_util.h` stand-in has likewise been deleted. Its Go API erased
map keys, map values, managers, and collections into `any`, while every source
operation is just a short typed decrement loop followed by clearing the owning
container. Those loops belong directly in each future concrete consumer, where
the compiler can check both reference types and the relevant manager method.

The unconsumed diagnostic ownership graph has also been removed: the
`MemoryProcess` orchestrator, `TraceContext`, `WarningContext`,
`PrintfFormatter`, and `DebugGlobalService`. These types only called one
another or appeared in their own tests. The live `DebugContext` remains because
vectors, heaps, parameter services, AST utilities, and assertion handling use
it. Future warning and trace entry points should be introduced with their first
real consumer and should use `fmt`, `io.Writer`, and ordinary string sets. The
deleted formatter was a manual C `printf`/`va_list` simulation and must not be
reintroduced. The unused `DebugContext` tag-runtime pointer and binding API
have also been removed; tag operations use the Go map directly.

Two more actual table implementations have been removed. The concrete
`Int32ChainTable`/`Int32ChainMap` port represented `chashtable`/`cmap`
instantiations that exist only in the upstream C++ table tests, and it had no
Go production consumer. `UintSetMap` also had no current Go consumer; its live
upstream symbolic-automata use should become a Go map keyed by a canonical,
comparable state-set representation when that consumer is ported. Neither
source use needs bucket arrays, collision chains, tombstones, or custom table
iterators in Go.

The live family-name `SymbolInt32Table` has been replaced at its AST consumer
with `insdict.Dict[Symbol, int32]`. Family lookup remains constant-time and
domain/range enumeration now follows stable registration order. The dedicated
probing table, scoped copy/rollback machinery unused by the family manager,
physical bucket accounting, and layout tests have been deleted.

The transient custom `statisticsMap` has been replaced in place with two
concrete `insdict.Dict` values keyed by ordinary Go strings. String value
equality preserves aggregation across distinct borrowed keys with the same
spelling, while insertion order gives deterministic internal rendering. The
probing array, cached hashes, growth code, and tests of physical bucket counts
have been deleted; normal and SMT2 rendering still sort keys explicitly because
those output formats require canonical order.

The live `SymbolDictionaryInt32` specialization now stores its values in a
built-in `map[Symbol]int32`. Its AST and tactic-manager consumers never iterate
the dictionary, so no ordering layer is needed. Symbol-owner validation remains
at dictionary operations, while bucket storage, tombstone cleanup, cached
hashes, simulated table allocation, and out-of-memory callbacks have been
removed.

The three global-parameter dictionaries have also shed their copied probing
tables. Module descriptors use `insdict.Dict[Symbol, uint32]` because display
traverses modules in stable registration order. Module parameter cells and
module descriptions use built-in maps because they are lookup-only. Destruction
walks the existing indexed owner slices, so cached hashes, tombstones, bucket
geometry, table allocation accounting, and shrink-failure states are gone.
Name-normalization scratch and its temporary string bridge are ordinary Go
storage and no longer consume the arena budget. The descriptor directory,
stable parameter cells, and borrowed description-region bindings also use
ordinary Go slices and heap objects; only their arena-owned descendants remain
in explicit memory accounting. Global-parameter diagnostics now compose text
with `strings.Builder` and checked symbol display instead of the translated C++
string-buffer owner.

`ParamsContext` formatting retains an ordinary Go byte slice for reuse. Growing
that slice no longer reserves arena budget or creates injectable failures, and
its capacity is no longer reported as arena-owned memory.

The symbol interner now owns immutable Go strings in a concrete entry slice,
with `map[string]uint32` lookup and stable, pointer-free IDs. Cached Z3 hash
values and signed-byte symbol comparison remain observable behavior; raw hash
prefix bytes, physical NUL storage, region descriptors, and arena growth
failures for spellings are gone. Borrowed parameter/stream views validate the
symbol manager on access, while returned Go strings remain safe after close.
Parameter name formatting and comparison now read native strings directly.
This removes the spelling-storage dependency on `region.h` without changing
symbol identity or source call ordering. Manager/service headers remain in
explicit budget accounting for now.

The interner and its identity/lifecycle tests are now physically in
`util/symbol`. Its full tests run independently of root numerical tests.
Parameters, S-expressions, and formatting no longer lock or inspect the
interner's internals; they acquire immutable Go spellings through checked
public methods. Runtime-backed producer reads and numeric stream adapters
remain beside their owning runtime/stream APIs, with concrete function
arguments rather than a reverse import into root `util`. The retained Z3
spelling-hash computation is private to the leaf and checked against existing
C++ golden values; it does not introduce a Go hash-table abstraction.


The live parameter-descriptor dictionary now uses
`insdict.Dict[Symbol, paramDescrInfo]`. Copy traversal follows stable insertion
order, while display continues to sort names explicitly for canonical output.
The separate names slice remains because the source API exposes registration
history, including names later erased from the live dictionary. Cached hashes,
tombstones, probing and rehash code, table geometry, and simulated allocation
failures for Go maps, slices, strings, and temporary sort/name buffers have
been removed. Arena-owned copies of descriptor text remain explicitly budgeted
and lifetime checked.

The first two counts make wholesale deletion unsafe. They describe migration
cost, not a reason to preserve the design. The latter two observations identify
good early simplifications, subject to checking future C++ consumers before
deleting functionality that has not yet reached Go.

The fast-float decimal formatter now uses four typed Go limb slices in place
of all its copied `sbuffer<unsigned>` storage, including the quotient buffer.
Its shifts, division, fractional digit generation, and full source precision
are unchanged. Inline masks, spill blocks, custom growth, Runtime bindings,
and the buffer-construction cursor are gone. Capacity is reused across calls
and released on explicit cleanup or close; these Go allocations are outside
arena accounting. Shared MPN character buffers and division workspaces remain
for subsequent migration.

S-expression bit-vector display now uses reusable Go byte scratch instead of
a copied inline-16/spill character buffer. Digit extraction, zero padding,
hexadecimal/binary selection, and values wider than the declared width retain
the source behavior. The printer emits the collected digits in reverse order
directly, without a temporary C terminator, custom growth policy, or arena
charges for Go bytes. Numeric temporaries still receive explicit deletion.

The S-expression manager also uses native slices for its traversal/deletion
stacks and output/string staging. Deletion retains the source LIFO order but
no longer preallocates one cleanup slot per record. Formatting grows with Go
append/make, and no scratch growth consumes the numeric arena budget. Paged
records, child handles, text payloads, and owned numerals retain their actual
arena/resource lifetimes.

The live parser `sexpr_ref_vector` specialization now owns a concrete Go slice
of scalar handles. Logical size is the slice length; append, erase compaction,
resize, reset, clone, and move use native storage. Reference increments/decrements
and structural-borrow invalidation remain explicit. The copied vector growth
helper, separate size field, and header/backing arena charges have been removed.
Its remaining upstream consumer is `src/parsers/smt2/smt2parser.cpp`.

### Dissolution order

1. Stop creating new `Runtime`, `Block`, custom vector, custom buffer, and
   custom table users. New ports use concrete slices/maps and native I/O.
2. Replace `str_hashtable`, trace/debug tag tables, and ordinary table consumers
   with `map` or `insdict.Dict` according to their required iteration order.
3. Replace warning, formatting, and text-buffer paths with `io.Writer`, `fmt`,
   `bytes.Buffer`, `strings.Builder`, and `bufio` as appropriate.
4. Delete hashes that existed only for custom tables. Keep explicit Z3 hash
   values beside the code that consumes them; do not form a `utilhash` package.
5. Convert one owning feature package at a time from `Runtime`/`Block` storage
   to concrete fields and slices. Move its focused tests at the same time.
6. Reduce memory management to `arena` plus a narrow limit/error policy. Remove
   allocation-layout and initialized-byte tests as their simulated owner goes
   away.
7. Replace the exception hierarchy with Go errors and move error codes to their
   API-facing leaf. Remove the final compatibility imports among the old knot
   units.

This order permits frequent green commits. It also prevents a second generic
container layer from forming: each converted feature declares its concrete data
types, and no new Go generics are required. The approved generic implementation
inside `insdict` remains the single exception.

### 4. The numerical subsystem

The numeric family is the largest coherent extraction. Source annotations in
the current port show the scale:

| C++ source family | Production Go files / lines | Test files / lines |
| --- | ---: | ---: |
| `mpf` | 75 / 13,249 | 178 / 19,161 |
| `mpz` | 58 / 11,711 | 73 / 14,978 |
| `mpq` | 34 / 6,405 | 53 / 11,737 |
| `rational` | 17 / 4,575 | 32 / 5,697 |
| `vector` | 28 / 6,381 | 27 / 5,725 |
| `scoped_numeral` | 6 / 1,623 | 9 / 3,050 |

Counts overlap when one Go file cites more than one C++ source, but the result
is still clear: numerical code dominates the package and test cost.

The main C++ dependency chain is:

```text
mpn <- mpz <- mpq <- rational
       ^      ^       |
       +-- mpf -------+
```

The floating-point and interval variants layer on those managers:

```text
mpff, mpfx, hwf, f2n
mpbq -> mpbqi
mpq_inf, mpzzp
s_integer -> inf_s_integer
rational -> inf_rational -> inf_eps_rational
         -> inf_int_rational
```

The Go port currently adds ownership edges that make a finer split unsafe:

* `IntegerManager` refers to rational and arbitrary-float services.
* `RationalManager` owns or refers back to integer and arbitrary-float managers.
* Arbitrary floats contain integer values whose representation is currently
  accessed across manager code.
* Numeric cells and allocation policy currently depend on `arena` and the
  shared C++-layout runtime simulation.

The first move should therefore be one package, tentatively `util/num`, holding
the live integer, rational, floating-point, interval, and numeral-lifetime code.
It should import `arena`, the narrow memory-limit policy if needed, and
independently extracted bit primitives. It must not import the parent `util`
package. Its `Runtime`/`Block` users should then be replaced internally with
native representations without exposing those compatibility types in the new
package API.

After that package is stable, a later audit can decide whether the manager
back-pointers can be removed cleanly enough for layers such as `num/core`,
`num/rat`, `num/float`, and `num/interval`. Creating those packages before the
ownership refactor would force internal representation fields and callback
interfaces into public APIs merely to avoid import cycles.

## Complete foundation-leaf inventory

These 27 logical C++ units include no other direct `src/util` unit:

```text
approx_nat        array              basic_interval     common_msgs
container_util    critical_flet      dec_ref_util        dlist
error_codes       event_handler                          file_path
luby                                 obj_ref             optional
pop_scopes        ref                ref_util            resource_limit
scoped_numeral    sstream            stats               stopwatch
stream_buffer     version            z3_omp
```

This list does not mean every item should become a Go foundation package.
`dlist` and `resource_limit` were confirmed dead and have already been removed
from the port inventory. `critical_flet`, `pop_scopes`, and `sstream` have no
known production source consumer and should be treated as deletion or deferral
candidates. `ext_gcd` is test-only in the surveyed C++ tree and its Go
translation has been removed. Several remaining
items are simple enough to replace with standard Go facilities.

## Complete extraction-leaf inventory

No other direct `src/util` logical unit includes any of these 73 units. The list
is grouped by likely disposition, not by a proposed one-package-per-row rule.

### Strong package candidates

```text
approx_nat             approx_set              backtrackable_set
cancel_eh              checked_int64           cooperate
dependency                                     env_params
f2n                    fixed_bit_vector        heap
hwf                    inf_eps_rational        inf_int_rational
                       lbool                   list
luby                   max_cliques             min_cut
mpbqi                  mpff                    mpfx
mpq_inf                mpzzp                   nat_set
numeral_buffer                                 plugin_manager
pool                   prime_generator         scoped_ctrl_c
scoped_numeral_buffer
scoped_ptr_vector      scoped_vector           sexpr
sorting_network        stack                   statistics
stopwatch              symbol_table            timeit
timeout
timer                  top_sort                union_find
version
```

Many of these are endpoints only within `src/util`; they have real consumers in
AST, SAT, SMT, tactic, solver, shell, or `muz` code. For example, `lbool`,
`statistics`, cancellation/cooperation, `heap`, and `union_find` all have broad
upstream use. Package boundaries must preserve that outward API.

`double_manager` was removed from the package candidates after a whole-source
consumer audit. Its only Z3 consumer is the double overload of
`linear_equation::get_a`, which merely copies an existing `double` coefficient.
The Go port should represent that operation as an ordinary `float64` return or
assignment. The manager's arithmetic maps directly to Go operators and `math`,
and its unused `atof`, `ostringstream`, parameter, and hash surfaces should not
be carried into Go.

The standalone concrete `Int32PArrayManager` was also removed after a
whole-source scan. Its exact int configuration occurs only in
`src/test/parray.cpp`. Production `parray_manager` instantiations belong to the
AST expression/dependency owners and subpaving bounds, so those concrete
consumer packages should own their typed implementations.

`s_integer` and `inf_s_integer` now form the pointer-free `util/smallint` leaf.
It retains the signed32 value and lexicographic infinitesimal algebra used by SMT
arithmetic and difference logic. Rational conversion stays at the future numeric
consumer boundary; C++ operator, functor, stream, and unused text-parser
compatibility surfaces are omitted.

`stats.h` has no live function call in the complete Z3 source tree. Several
translation units include it, but neither inline `print_stat` overload is ever
called. Its self-tested Go translation was removed instead of creating a
package for dead source.

`machine.h` is a C++ ABI macro used for allocator slot geometry and pointer
tagging. Go arena implementations define their own concrete geometry, and Go
pointers must remain visible to the garbage collector instead of being packed
into `uintptr` tags. The unused root alignment mirror was removed; a concrete
Go consumer can query `unsafe.Alignof` directly if alignment is relevant.

### Small units requiring a Go-native shape before packaging

```text
array                  array_map               container_util
dec_ref_util           ext_numeral             file_path
obj_mark               obj_pair_hashtable      obj_pair_set
obj_triple_hashtable   pop_scopes              ref
ref_buffer             ref_util
sstream                stream_buffer
```

These are often C++ templates, ownership wrappers, or compatibility helpers.
If live, their functionality should still have a visible package boundary, but
their Go form should be a concrete type, an ordinary slice/map, or a cohesive
package rather than a copied generic wrapper.

The unused `ExtNumeralManager` translation and its self-tests have been removed.
Its sole production source consumer is the unported interval subsystem, which
must receive a concrete typed extended-numeral implementation when it is ported.

### Dead, test-only, or requiring proof before retention

```text
critical_flet          dlist                   ext_gcd
obj_ref_hashtable      object_allocator        permutation
ptr_scoped_buffer      resource_limit          total_order
uint_map
```

`dlist`, `resource_limit`, `permutation.cpp`, and `obj_ref_hashtable.h` are
confirmed dead and their copied sources are deleted. Prior whole-tree
surveys found `ext_gcd`, `object_allocator`, `ptr_scoped_buffer`, and
`total_order` used only by tests. The Go translations of `ext_gcd`,
`object_allocator`, and `total_order` have been removed. `critical_flet` has no
known caller. `permutation.h` has live free-function consumers; retain only
that behavior. Recheck the entire
C++ tree before deleting a unit because a lack of current Go consumers alone
does not prove that upstream Z3 code is dead.

`uint_map.h` is reference-only: its sole include is the test-only
`total_order.h`. The `uint_map` in `src/test/uint_set.cpp` is a separate local
type over `util/map.h`, not an instantiation of this header.

These are source units rather than a claim that each file defines one class.
Some define templates or free functions. Includes are recorded at translation
unit and header granularity, so this is the reliable granularity for dependency
and package-boundary decisions. For example, `scoped_numeral_buffer` belongs
with numeric ownership even though a generic C++ buffer wrapper by itself would
not merit a package.

## Migration procedure

For each package extraction:

1. Confirm production consumers across the entire C++ tree and current Go
   module. Delete or defer code with no production role.
2. Define the smallest concrete Go API used by those consumers. Avoid `any`
   containers copied from C++ templates.
3. Move implementation and focused tests together. Preserve C++ source path and
   line annotations.
4. Make the new package import only lower layers. Never add an import back to
   root `util` to make a move compile.
5. Update one consumer group at a time and run its focused tests plus the new
   package tests.
6. Commit each green extraction. Run the full native suite only at the agreed
   interval, since its current cost makes it unsuitable for every move.
7. Record package test times before and after the move. The purpose is both a
   clearer dependency graph and faster focused verification.

The first extraction is `util/lbool`. Its implementation now lives in the leaf
package with no dependency on root `util`; root exports temporary aliases while
consumer packages migrate to the leaf API. Once those aliases are no longer
used, the compatibility file can be deleted. The same process can then be
applied to cliques, union-find/backtracking, sorting networks, and the larger
numeric boundary.

`util/mincut` is also extracted with its focused tests. It depends only on the
standard library and has no current Go production consumer; root keeps aliases
for the live upstream users that have not yet been ported.

`util/backtrack` now owns trail stacks and both union-find variants. Their tests
move with the implementation, and the leaf depends only on the standard
library. Root aliases preserve the API for future upstream consumers.

`util/version` owns the generated Z3 version tuple and its source-header
consistency test. Root constants remain as compatibility names.

`util/filepath` owns the source filename-extension rule and depends only on
the standard library. Its API accepts and returns Go strings and reports a
trailing-dot extension with a separate boolean; the old borrowed `[]byte`
C-string facade had no Go consumer and has been deleted rather than carried
across the package boundary. Future API and shell ports should import this
leaf directly.

`util/approx` owns the concrete approximate natural and 64-bit unsigned-set
implementations. Both depend only on the standard library. Formatting now uses
`String`, and set traversal uses a typed callback; the translated numeric
ostream edge and C++ iterator object were deleted. There were no current Go
consumers requiring a root compatibility facade.

`util/messages` owns the seven common Z3 message strings as constants. A
whole-source scan found no reassignment of the corresponding C++ static pointer
bindings, so the root mutable globals and tests of pointer rebinding were
deleted. Resource limits, SAT, rewriting, and model evaluation import the leaf
directly.

`util/natset` owns the fixed-domain timestamp set and depends only on the
standard library. It uses a concrete `[]uint32`, has no root facade, and is
ready for the future pattern-inference and arithmetic consumers to import
directly.

`util/stack` owns the live low-level LIFO byte allocator and its focused page
recycling tests. It imports `arena` directly, has no dependency on root `util`,
and exposes no compatibility facade; the future SMT2 parser port should import
this leaf.

The redundant Runtime-backed bit containers have been deleted:
`RuntimeBitVector`, `RuntimeFastResetBitVector`,
`RuntimeFixedBitVectorManager`, and `RuntimeUintSet`. A production-reference
scan found that each family was used only by its own implementation and tests,
while the same source functionality is already present in the typed native
`Vector`/`FastReset`, `FixedManager`, and `UintSet` implementations. This removes
four isolated `Runtime`/`Block` ownership graphs and their allocation-layout,
interrupted-destruction, and malformed-header tests without removing a live Go
consumer.

`util/stopwatch` owns `Stopwatch`, `ScopedWatch`, and the Linux/portable clock
adapters. SAT reporting imports the leaf directly. `Timer` now lives in the
same leaf and owns a concrete stopwatch directly;
its copied 32-byte C++ layout, Runtime binding, and byte serialization are gone.
`Timeit` has also moved into the leaf. It uses Go strings and `fmt` with an
`io.Writer`, plus an optional typed callback for memory statistics. Its copied
56-byte imp layout, Runtime dependency, root stream/parameter dependencies,
and persistent C++ ostream flags are gone. Reporting preserves clock, memory,
write, and flush errors, callback reentry guards, and borrowed writer lifetime.
Live upstream reports are in SMT context and Spacer context code. The leaf has
no project-package dependency.

`util/cooperate` owns the no-OpenMP cooperation section/task state and its
debug/release assertion profiles. It imports only `arena`; model evaluation,
SAT goal translation, and current test fixtures import it directly without a
root facade.

`RuntimeUnsignedRows` has left root `util`. Its sole production consumer was
SAT consequence/mutex output, so the concrete owner now lives privately in
`sat` as `satSolverLiteralRows`. SAT's existing fault campaigns retain the
observable partial-copy, outer-growth, source-finalization, and ownership-root
coverage; the util-only public facade and representation tests were removed.

The concrete unsigned32 Runtime column and its copy specialization are likewise
SAT-only and now live in `sat_solver_scalar_runtime.go`. The remaining Runtime column engine now lives privately in SAT as
`satRuntimeColumn`; AST sorting-network literal scratch uses native typed
slices. Root util no longer exports a Runtime vector or scalar-growth facade.


Only the currently consumed `StringBuffer64` specialization remains; the 32/128 variants and cross-width test matrix were removed. `StringBuffer64` now uses `bytes.Buffer`, and its callers no longer reach through a C++-style state/inline/overflow representation or charge Go-managed text storage to an arena budget.

The live `buffer<rational,16>` specialization used by AST polynomial rewriting now uses a concrete `[]OwnedRational`. Numerical element construction and ordered cleanup remain explicit because the elements own real resources; inline storage, doubling rules, relocation bookkeeping, and arena charges for the Go slice are gone.

Small-object allocator consolidation now gathers and sorts allocation IDs in concrete `[]uint32` scratch. Its two copied pointer-vector headers and Runtime blocks, vector growth/failure surface, and retained lexical scratch roots have been removed; only actual allocator chunks and pending allocations remain in Runtime ownership.

Consolidation now operates directly on those slices and uses Go `sort.Slice`
for scalar-address ordering. The custom vector wrapper, handwritten heapsort,
and no-op lexical destruction adapters have been deleted.

The concrete scoped integer, rational, and binary-rational vectors now use typed Go slices. Their elements still receive explicit deep copies and ordered deletion because they own numeric resources, while wrapper/backing arena charges and the C++ vector capacity schedule are gone. This keeps source-visible copy-failure prefixes and lifetime behavior without treating Go-managed slice storage as numeric arena memory.

The rational service's power-of-two cache now grows with native `append`. Cache entries remain concrete owned `Rational` values and are still destroyed in ascending order, but the copied `vector.h` growth routine, moved-from backing mutations, and `CacheBytes` arena accounting have been removed.

`ast.ASTHandleVector` now uses native slice growth for its borrowed, pointer-free handles. Stable order, erase compaction, and clearing the vacated slot remain; the common `vector.h` capacity helper and numeric-arena charges for this Go slice are gone.

`ast.ASTHandleBuffer` now uses the same native typed-slice representation for borrowed handles. Its inline-16/spill split, doubling logic, and AST-budget backing charge have been removed; reset and pop clear vacated handles while retaining ordinary Go slice capacity.

Stored-model label snapshots in `tactic` now use `[]util.Symbol`. The inline-16/spill buffer and its arena accounting are gone; construction still copies caller labels, application preserves insertion order, and close clears retained symbols.

The goal flattener's polarity worklist in `tactic` now uses a concrete slice as a LIFO stack. Its inline-64/spill representation, doubling code, and arena backing charges have been removed; popped and closed entries are cleared.

Bit-vector polynomial multiplication now stores its unsigned coordinates and borrowed sum views in concrete typed slices. Both inline-16/spill buffers and their arena accounting have been removed; close clears the slices.

The multiplication-hoist validity mask now uses `[]bool`; its inline-16/spill buffer, doubling path, and arena charge have been removed.

The model package's function-entry directory, function-interpretation entries, universe directory, array-expansion rows, and lexical function arguments now use native typed slices. Their shared `SourceVectorCapacity` calls and arena capacity counters have been removed; AST/model objects within those slices retain their existing ownership and cleanup.

The API object directory and recycled-ID stack now use native typed slices. Stable destruction order remains in the approved insertion-ordered dictionary; the auxiliary slices no longer use `SourceVectorCapacity` or AST-budget backing charges.

Tactic model labels and concatenated-converter child/size arrays now use native typed slices. Converter reference ownership and label order remain explicit, while `SourceVectorCapacity` and arena charges for those Go slices are gone.

AST as-array sort construction now stores its owning parameter descriptors in a typed slice. Parameter cleanup remains explicit; `SourceVectorCapacity`, capacity charges, and relocation bookkeeping are gone.

SAT goal translation now uses native typed slices for dependency roots,
interpreted atoms, traversal frames, result literals, and recovered literal
expressions. AST reference ownership remains explicit, while copied C++ vector
growth and arena charges for Go-managed backing storage are gone. These were the
last consumers of `SourceVectorCapacity`, so that shared helper and its
implementation-only test have been deleted.

AST sort and function-declaration metadata now stores its owning parameter
handles in `[]ASTParameter`. Parameter copies and ordered release remain
explicit, while the raw byte vector, embedded capacity words, and arena-backed
growth failures are gone. Lexical format metadata continues to borrow its
call-local parameters through the existing lexical record.

Polynomial variable-degree maps, maximum-degree variable lists, and variable
position maps now use concrete `[]uint32` storage. Sparse default values,
first-seen variable order, deep-copy assignment, move construction, scoped
reset, and display behavior remain explicit. C++ vector capacity growth,
`size_t` conversion, pending Runtime blocks, and arena failure states for these
Go-managed slices have been removed, along with the tests that existed only to
validate those representations.

Polynomial monomial-to-position maps now also use concrete `[]uint32` storage.
The arithmetic continuation still binds an unfinished write to its exact owner,
invocation, row, and value, while native slice relocation no longer appears as
observable block identity. This removed the final production use of the copied
`VectorGrowth` arithmetic, so that helper and its representation test are gone.

The live polynomial `scoped_numeral_buffer<mpz,128>` specialization now stores
its concrete owned `Int` values in `[]Int`. Deep copy, modular normalization,
ascending source deletion, stopped-cleanup state, and borrowed views remain;
inline/spill capacity, Runtime relocation blocks, and their allocation-failure
tests are gone. Future specializations should likewise be typed at their actual
consumer rather than restoring a universal buffer abstraction.

Integer and polynomial decimal character scratch now uses native `[]byte`.
The shared formatter reuses slice capacity and clears the active region before
conversion. Borrowed output still checks the exact owner and invocation, while
C++ inline/spill storage, Runtime character blocks, allocation-refusal states,
and destructor-failure states are gone. MPN numerical workspaces still retain
explicit resource ownership and failure cleanup; they are a separate remaining
step. AST and polynomial physical-root audits now count only those real numeric
allocations, and retained tests cover output, bounds, stale views, inner numeric
failures, and cleanup.

MPFF's retained significands, handle metadata, recycled-ID stack, conversion
scratch, and arithmetic scratch now use concrete `[]uint32` storage. Native Go
growth replaces the copied vector capacity schedule and its arena descriptors.
Generation checks, numeric slot recycling, original arithmetic/rounding, and
manager cleanup remain explicit. The division and decimal bridges now borrow
these slices directly; shared MPN resources remain separate. A focused test grows
through multiple pool sizes with arena allocation disabled, verifies retained
values, and verifies generation advancement after numeric slot reuse, including stale
handle rejection in the checked profile.

NaturalManager's Words storage dependency is gone. Neither arithmetic nor
formatting populated the legacy Words scratch fields; their actual resources
live in the existing Runtime buffers. The permanently empty scratch arrays,
duplicate capacity mirrors, unused growth/read helpers, and Words-backed
limb-view branch have been deleted. MPFF likewise no longer creates an unused
Words service. Physical ownership audits now inspect the remaining numeric
resources directly, and nested-owner tests verify that child cleanup preserves
the borrowed parent Runtime and its caller allocation. This removes dead
storage infrastructure without changing the remaining arithmetic buffers.

IntegerManager likewise no longer constructs an unused Words service. Its real
limbs already live in the numeric allocator/Runtime; no arithmetic or formatter
consumer populated the removed service. Constructor, destruction, statistics,
and physical-root audits now omit that empty owner. The Words field in
IntegerConfig remains as Natural output configuration, while independent live
Words users (parameter metadata, S-expression children, and SAT storage) remain
separate conversion work.

SAT binary-clause collection now stores its two complete literal indices in
`[]sat.BinaryLiteralPair`. Solver mutex discovery, maximum-variable scans,
inspection, and tactic goal recovery use slices directly. The BinClauseBuffer
wrapper, Runtime vector column, realloc tokens, standalone owner charge, and
stopped-destructor adapter are gone for this specialization. Watch traversal
order, original/learned filtering, caller-prefix appending, and full literal
bits remain covered by focused SAT tests.

Probing's assigned-literal set and conflict assumption membership now use
concrete `[]bool` and `[]sat.Literal` storage. Membership lookup and unique
first-insertion order are retained; reset reuses backing and finalize releases
the slices. These owners no longer bind shared Runtime vector columns. The
probing cache still applies its existing strict cutoff to Runtime allocations,
so Go-managed set memory is outside that cutoff and arena accounting. Focused
tests cover native growth, duplicates, reset/regrowth, the invalid wrapped
literal, finalization/reuse, and the retained cache-limit policy.

MUS and consequence literal sets now reuse the same concrete native literal
storage as probing. Intersection compacts in the original order, union appends
unseen values in source order, assignment clones membership and values, and
removal swaps the last value into the removed position. Captured set iteration
borrows a Go slice. Runtime vector copy remains only at still-live unsigned
column consumers; the MUS-named forwarding wrapper and unused request helpers
have been removed. Physical allocation/destruction tests for the converted sets
are replaced by order, independent-copy, native-growth, and scoped-cleanup tests.

The separate `satSolverLiteralSet` owner/iterator facade had no production Go
consumer anywhere in pkg/. It and its self-tests have been removed. This does
not mark C++ `sat::literal_set` dead: current probing/MUS/consequence consumers
use native storage, and `src/sat/sat_solver/inc_sat_solver.cpp` remains a future
consumer. The similarly named class in src/qe is its own `nlarith::util::literal_set`.

The retained unsigned set now uses ordinary slice growth for elimination todo,
model-converter variable collection, and integrity checks. Its cold owner
header remains budgeted; Go backing capacity is outside arena accounting.
Unused copy/union/intersection/removal facades, copied capacity arithmetic, and
iterator relocation epochs are removed. Elimination ordering ranges over the
captured values slice. Tests retain insertion order, deduplication, reset and
regrowth, finalization/reuse, constructor/lifetime checks at real consumers,
and independent membership/order comparison.

A second standalone SAT set prototype, `sat_sets.go`, was also consumed only by
its own tests. It has been removed alongside the unconsumed owning facade.
Current solver work uses the retained native literal/unsigned storage; unused
copy/pop/vector-wrapper APIs will be implemented only for a concrete future
consumer. The separate approximation mask is live in clauses and simplification;
its existing test was moved beside that implementation and retained.

Numeric output assembly now uses a reusable native `[]byte` owned by
NaturalManager. Integer, rational, MPFF, and infinitesimal formatting copy the
prepared bytes into caller buffers or independent Go strings. Growth preserves
already-written prefixes, including the numerator while preparing a denominator;
close drops the slice. The SmallBytes owner, indexed payload access, copied
capacity schedule, and text-only allocation refusal states are gone. This
Go-managed text is outside arena accounting. Numerical limb workspaces retain
their separate Runtime ownership and failure cleanup. Failure campaigns use
operands exceeding inline numerical storage so they continue testing real
allocation failures, alongside output, short-buffer, returned-string lifetime,
and warm-allocation checks.

S-expression child lists now use independently copied `[]SexprHandle`, and
string records retain native Go `string` values. This removes their Words and
SmallBytes owners, slot/generation word packing, and string copying through
indexed payload access. Recycled records clear their slice/string fields;
returned immutable strings remain valid after deletion and manager close.
Record pages and the manager header remain budgeted, while child/text payloads
are Go-managed. Source child order, duplicate reference counts, handle generation
checks, iterative deletion, numerical ownership, and display quirks remain.

Parameter descriptor descriptions, defaults, and module names now use immutable
Go strings with explicit presence bits. Missing and present-empty metadata remain
distinct, and C-string inputs still stop at NUL. Removed the Words service,
word packing, text copy/free loops, and text allocation-failure states. Descriptor
copy shares immutable strings safely across owner close while retaining insdict
insertion order and independent dictionary ownership. Header allocation remains
budgeted; native metadata, dictionary, and name storage are outside that budget.
Parameter-value strings and their borrowed views still use their separate Words
service and require a later conversion.

Parameter-value text now uses immutable Go strings in concrete owned lifetime
records. Returned views borrow those records; replacement/reset/close invalidate
the affected view and clear its bytes. Deep copies share immutable backing but
receive distinct lifetime records; the original null-destination copy shortcut
still shares the whole parameter block until copy-on-write. Borrowed symbols keep
their independent symbol-provider lifetime. Removed the ParamsContext Words
service, word packing, uint32 text-length codec, and text-only arena failures.
Entry/receipt arrays remain separately budgeted; native text is outside arena
accounting. Root util now has no Words owner; fixed-width bitset storage in its
already-separated child package remains an independent arena consumer.

The datatype plugin's constructor-only copied definition table, nine utility
object-table shells, and empty Runtime reference/trail vectors were consumed
only by their own representation tests. Removed that Go scaffolding after a
module-wide scan; retained actual plugin registration, family lookup, header
ownership, and finalization. C++ datatype definitions and utility operations
remain live and unported. Concrete native maps/slices should be introduced only
when their functional Go consumers are ported; empty source templates do not
justify another dependency on the container infrastructure.

## Native text buffers

`string_buffer.h` is a Go platform replacement. The former `util/text.Buffer`
wrapper and its unused floating-point append adapter were removed. Retained
callers use `bytes.Buffer`, `strings.Builder`, `strconv`, or local byte slices.
For example, AST fresh-declaration names use `bytes.Buffer`, symbol text uses
`strconv`, and AST model-value names retain a small inline byte array plus
charged overflow because that caller exposes the source allocation prefix.
The buffer disposition does not claim completion of the algorithms that use it.

## Floating-point numeral slice

The floating-point plugin's concrete scoped numeral storage now uses
`[]util.ArbitraryFloat` in `ast`. It no longer consumes RuntimeVectorColumn or
serializes numerals into 32-byte C++ rows. Native append preserves owned numeral
handles without acquiring another numeric reference. Copy still operates through
the numeric manager; source destruction and owning failure cleanup visit elements
in ascending order. Retained copy receipts protect actual numeral allocations,
including multiple failed copies. The native copy-chain path replaces its unused
Runtime-row facade, and the now-unconsumed vector allocator method is removed.
Numerical cell/limb ownership remains separate from Go slice backing.

## Native SAT consequence and mutex output rows

SAT's `satSolverLiteralRows` now stores independent `[][]Literal` results.
The copied outer pointer directory, Runtime-backed inner vectors, temporary
unsigned row, metadata accounting, moved-prefix history, and failed vector
constructor roots are gone. Runtime column inputs are still checked and copied
through their actual reader. Captured rows survive native outer growth;
finalization/reuse invalidates old views, and close releases the charged owner
header. Go slice backing is outside arena accounting. Solver consequence and
mutex refusal campaigns continue to audit real solver-owned allocations and
verify that previously published output rows remain unchanged.

## Native SAT Boolean marks

SAT variable marks and literal marks now use concrete `[]bool` fields in the
variable state. Conflict analysis and dynamic subsumption read and change those
flags directly, preserving duplicate-mark/unmark assertions and reached bounds
errors. Variable creation appends false defaults in its original action order.
Scope truncation clears removed flags, and newly created variables begin
unmarked when the backing slice is reused. Finalization drops both slices.
Neither array owns Runtime blocks or has copied C++ capacity/accounting state;
physical ownership audits count only the remaining real Runtime consumers.

Phase storage remains a distinct three-state byte representation; its live
assignment/decision/GC and asymmetric-branch snapshot consumers still require
conversion. The shared Runtime vector therefore remains live.

## Remaining Boolean SAT flags

Decision eligibility, eliminated status, external status, and assignment-since-GC
now also use `[]bool`. All six Boolean variable/literal arrays have left the
shared Runtime vector. Solver copy reads typed Boolean values; elimination,
assignment, and GC update the actual native flags. Source action order and
reached owner/bounds guards remain, while backing growth and cleanup use native
Go slice operations. Scope pop/regrowth preserves retained flags and initializes
new variables from their requested decision/external flags and false defaults.
Only assignment and the two three-state phase arrays still use Runtime byte
columns in variable state; asymmetric branching also retains its phase snapshot.

## Native SAT saved polarities

The two three-state phase arrays now use concrete `[]int8` fields, with the
source values 0, 1, and 2 (undefined). Assignment, decision selection, conflict
analysis, and dynamic GC use typed variable-state operations. Scope truncation
preserves retained polarities and initializes recreated variables to undefined.
Asymmetric branching takes an independent native snapshot and restores it only
on normal completion, matching `src/sat/sat_asymm_branch.cpp:70-113`. Cancellation
continues to retain the interrupted algorithm's phase changes. Snapshot cleanup
has no Runtime allocation/destruction or partial vector-constructor state.

Only assignment remains a Runtime byte column in SAT variable state; typed
level, activity, trail, justification, and queue columns remain separate work.
Native backing is outside arena accounting. The physical ownership tests now
count only the real remaining Runtime blocks.

Validation: complete SAT suites passed in native (1.06s) and checked/debug
(1.24s) builds, including phase scope truncation/regrowth, normal restoration,
cancellation, and allocation refusal campaigns. Module-wide test compilation,
SAT vet, and portcheck passed. The latest full native checkpoint remains the
18:07–18:12 UTC run; it was not repeated for this focused change.


## Native AST sorting-network literal scratch

`ast.ASTLiteralVector` now stores borrowed handles in a concrete `[]ASTHandle`.
It imports only `arena` for the existing owner-header charge; the native slice
has no numeric Runtime allocation, serialized handle payload, copied C++
capacity schedule, or acquisition history. Captured spans retain their slice
across growth and reject reads after owner finalization/close. Reset/pop clear
removed handles without changing AST reference counts.

The copied vector clone/assignment implementations had no production caller
and were deleted, along with their storage-specific tests. Unused reserve,
resize, capacity, and append-array scaffolding is gone. Sorting-network range
construction still reads input in order; a reached read error releases native
partial scratch and recursive frame headers. AST registration/reference owners
remain separate and keep their real allocation-failure audits. The unused root
`RuntimeVectorGrowthOwner` callback interface and hook branches were removed
with their last concrete implementation.

Validation includes the independent recursive clause oracle, every remaining
bound allocation refusal, the short-range cleanup regression, native capture
lifetime/reference checks, and full native AST/SAT tests. Root Runtime vector
columns now serve SAT alone; the numeric Runtime itself remains live elsewhere.

## Native SAT assignments

SAT literal assignments now use `[]lbool.LBool`, leaving no Runtime byte columns
in variable state. Creation appends paired undefined literals; assignment and
backtracking update each polarity in the original order, and scope pop clears
and truncates removed literals. Clause views borrow variable state and read its
current slice, so they remain correct across native growth. Typed reads/writes
retain owner, close, and reached bounds guards. Removed serialized byte-header,
spare-slot, capacity-schedule, and assignment allocator-failure tests; remaining
real Runtime allocation campaigns and their direct Runtime census stay live.
Native assignment mutation does not change arena/Runtime accounting.

Validation: complete native and checked/debug SAT suites, module-wide test
compilation, SAT vet, and portcheck passed. Numerical/typed Runtime columns are
still separate conversion work; root vector storage now serves SAT alone.


## SAT-only Runtime column engine removed from root util

The remaining typed Runtime columns and their copy/byte codec now live in SAT,
using a private `satRuntimeColumn`. The former SAT alias file and root utility
vector implementation are gone. Removed the unused byte push/read/write and
byte-growth branches after assignments left Runtime storage. Level, activity,
trail, justification, queue, and remaining solver scratch still use their real
Runtime allocations pending native conversion; no new common container package
was introduced.

Debug facilities now own `DebugBuild`, `ErrDebugAssertion`, and the context
budget/identity accessors. The unused unsigned-growth executing error and root
vector test census were deleted. The VERIFY macro test uses the actual Runtime
byte read/write operations directly, so debug testing does not retain a vector
codec dependency. Native/checked SAT and focused debug/sorting-network tests,
module-wide compilation, vet, and portcheck verify the new package boundary.

## Native SAT activity scores

The actual solver activity owner now stores `[]uint32` in `sat_solver_activity.go`.
Increment/decay retain uint32 overflow and threshold behavior; rescaling walks
the native slice. The queue borrows this concrete owner independently of its
own remaining Runtime backing. Queue construction receives that Runtime
explicitly, and copied/moved queues retain the same current-member activity
borrow. Scope pop clears/truncates removed scores and regrowth starts at zero.
The activity-specific Runtime allocation/destruction census and TLS refusal
fixture are gone; remaining real Runtime failures still have their own tests.

Validation: full native SAT (0.99s), checked/debug SAT (1.24s), module-wide
compilation, SAT vet, and portcheck passed. Existing threshold/wrap, decay,
rescaling, queue copy/move/growth, and native scope-regrowth tests remain live.
A regression verifies that score mutation/rescaling changes no allocator state.

## Native SAT decision levels and variable count

Decision levels now use `[]uint32`, initialized to the source maximum-unsigned
sentinel. Assignment updates the native level in source order; clause views read
the owner's current member. Variable counts now read that slice's length with
owner/lifetime checks instead of consulting a serialized header. All count
consumers in decision selection, probing, scopes, model creation, copy, display,
inspection, SCC, and integrity checks use this representation. Scope truncation
clears discarded levels and recreated variables receive the sentinel.

Deleted the level-specific capacity/spare-byte/TLS tests and malformed/freed
header fixtures. Native count/level regressions cover growth, current-member
borrowing, no allocation, bounds, copied/closed owners, and scope regrowth. The
model publication-order regression now uses an actual model allocation refusal,
which remains live, instead of corrupting eliminated level storage. Remaining
Runtime allocations in variable state belong to justification, trail, and queue.

Validation: full native (0.99s) and checked/debug (1.20s) SAT suites, module-wide
compilation, SAT vet, and portcheck passed. The full native module checkpoint
remains 18:07–18:12 UTC; no new whole-module execution was needed for this change.

## Native decimal MPN scratch

The three decimal-formatting limb buffers in `mpn_format_scope.go` now use
`[3][]uint32`. Invocation entry clears the requested spans; normal/error exit
truncates them for reuse, and explicit cleanup drops capacity. `mpn_cstring.go`
borrows these slices through the existing concrete limb view and retains the
original digit conversion arithmetic and full-precision operand sizes.

Formatting scratch no longer uses `SourceLimbBuffer`, inline16 growth policy,
Runtime blocks, initialized-byte maps, or partial C++ constructor/destructor
allocation states. Warm integer/rational and full-precision float formatting
require zero Go allocations and zero Runtime allocation events. The remaining
`SourceLimbBuffer` users are now only division (checked verification was removed
in the next chunk below); those buffers retain real allocation/failure ownership.
Polynomial and AST physical-root test helpers count only those remaining blocks.

Downstream fault tests tied exclusively to removed scratch allocations are
retired. Real numeral-copy, division and normal-classification allocation tests
remain. AST low-level printing independently checks large numeral text and
clean frame teardown; general visitor mark/history regression coverage remains.

Validation: full native util/AST, focused native polynomial display,
checked/debug affected features, module-wide compilation, affected-package vet,
and portcheck passed.

## Native checked division verification

`mpn_source_verify.go` now performs the original checked multiply/add comparison
using a reusable `[]uint32`. It clears the requested span on entry and truncates
it on every return, including an arithmetic assertion. Owning close releases
capacity. Cold scratch growth and warm verification do not enter Runtime;
warmed calls allocate nothing.

The verification holder/receipt types, primitive caller capability, private
buffer branches and polynomial/MPF/MPQ cleanup plumbing are gone. Normalization's
source-cause accessor was used only for that buffer destructor and is also gone.
Tests requiring verification allocation/release refusal are retired; real
integer/RAW and division allocation failure coverage remains. New tests cover
sizes crossing the former inline boundary, assertion cleanup, reuse and close.

`SourceLimbBuffer` served only the four division buffers after this chunk.
The following Algorithm D conversion removes it and its receipt machinery.

## Native Algorithm D workspace

All four division buffers now use `[4][]uint32`; `SourceLimbBuffer` is deleted.
Resize preserves the live prefix, clears newly exposed limbs, and retains
capacity on shrink. Algorithm D still performs the original normalization,
quotient estimation/correction, product, modular subtraction, conditional
add-back and remainder publication. The add-back workspace remains lazy.
Source and owning close release all native slices without Runtime events.

Removed the buffer receipt/identity types, primitive caller buffer capabilities,
PN/RAW/MPF/polynomial pending-buffer fields, acquisition histories and cleanup
loops. Actual numeral/cell/RAW ownership remains. Runtime physical-root audits
exclude Go scratch; RationalService's legacy `NaturalPayloads` statistic is
zero. Tests of C++ inline/doubling/failed expansion states are retired; independent
arithmetic, conditional add-back, cleared regrowth, close and zero warm allocation
checks cover the native representation.

MPN workspace no longer contributes a Runtime buffer dependency. Numeric cells,
allocator-backed numeral storage and the remaining concrete polynomial/container
buffers are still pending work for dissolving the infrastructure knot.

Validation: full native util (96.70s), full native polynomial (36.34s), AST
floating-point checks (0.07s), checked/debug division and affected callers
(util 1.07s, polynomial 0.53s), checked/debug fast-float (29.54s) and targeted
lifecycle/sexpr checks passed. The final dead pending-buffer predicates and
Natural source-cause field were pruned after the broad runs started; final
focused checks, module-wide compilation, affected-package vet and portcheck
passed for that pruning. No new full-module execution or broad checked
polynomial sweep was run; the latest full native module checkpoint remains
19:07–19:12 UTC. Manifest counts remain 2102 units / 677 APIs, with no new
whole-unit completion credit.

## Native polynomial GCD degree workspace

GCD's power pairs now use the concrete `polynomialGCDPowers` slice of
`MonomialPower`, and its variable-to-position cache is `[]uint32`. Neither
retains a buffer/vector allocation header, inline payload, pending growth block,
initialized mask or Runtime allocation event. The existing GCD scope stack
releases native pairs in source order; manager source/owning close clears the
cache. Variable collection retains first occurrence, maximum/minimum selection
and reset to the maximum-unsigned sentinel after each successful collection.

Go `sort.Slice` replaces the copied heapsort used to adapt `std::sort`. It uses
the same ascending unsigned variable-only and degree-only comparisons. Pair
identity and duplicate-key multiplicity are tested without requiring a C++
standard-library swap schedule. Large cold append and warmed reuse tests run
with Runtime reservations disabled. Recursive source-stop tests preserve older
native backing while a younger owned resource blocks cleanup.

The adjacent ICCP unsigned filters, GCD variable list, monomial reference buffer,
and polynomial vectors remain separate remaining conversions. No generic
buffer layer is introduced.

Validation: full native polynomial (35.97s), final checked/debug GCD/ICCP/manager
checks, module-wide test compilation, polynomial vet and portcheck passed.
The hourly full native module checkpoint remains 19:07–19:12 UTC; no additional
whole-module run or broad checked sweep was needed for this feature conversion.
Manifest counts remain 2102 units / 677 APIs, with 63 ported, 284 partial,
1727 unported and two dead/deleted.

## Native ICCP unsigned scratch and modular-GCD variables

ICCP's coefficient counters and first-occurrence degree list now use plain
`[]uint32`, as does the degree-sorted variable list borrowed by modular-GCD
recursion. `polynomial_content_buffer.go` and
`polynomial_gcd_variable_buffer.go` are deleted. Their inline/spill storage,
expansion copies, initialized masks, pending Runtime blocks and destructor
failure states no longer exist. Source/owning scope cleanup clears native
slices at the existing boundaries; actual numeral/reference children still
control whether older cleanup may proceed.

New independent content/primitive tests exercise degrees 33, 129 and 257 and
repeat the operation to check release of the native scratch. Recursive dense,
sparse, field and source-stop tests use borrowed Go slices. A deliberately
invalid terminal variable value leaves the Euclidean base-case result unchanged.
The stale ICCP receipt test now triggers a real large-coefficient numeral
allocation failure and requires an active replacement invocation, rather than
relying on eliminated filter growth. Its state comparison includes slice data.

The monomial reference buffer and remaining polynomial vectors still require
native storage conversions. This chunk adds no replacement container package.

Validation: full native polynomial (36.93s), focused native GCD/ICCP/manager
checks (3.16s), final checked/debug feature checks, module-wide test compilation,
polynomial vet and portcheck passed. The active replacement-receipt assertion
also passed in checked mode. The full native module checkpoint remains
19:07–19:12 UTC; no additional whole-module or broad checked sweep was run.
Manifest counts remain 2102 units / 677 APIs: 63 ported, 284 partial,
1727 unported and two dead/deleted.

## Native GCD monomial reference storage

GCD's concrete monomial reference buffer now stores `[]Monomial`. Removed its
inline/spill payloads, Runtime blocks, expansion copies, pending backing,
fallible publication and unjoined-increment cleanup. The existing scope stack
still owns its lifetime. Each append increments the real monomial reference
before publication and transfers generated ownership; reset/destruction drops
references in source order. A cleanup cursor and destructor-stop state remain
for actual monomial destruction, including explicit owning continuation.

New tests cover 4097 duplicate references, cold growth with Runtime reservations
disabled, reset/reuse and final release. Generated monomial ownership and a real
last-reference ID-recycling refusal exercise source-stop/retry/owning cleanup.
An independent GF11[x] content/primitive model uses 33 coefficient monomials to
exercise native growth through ICCP. Younger cleanup before an older stopped
restore is now checked by reference counts rather than deleted backing blocks.
The obsolete consumed-old-growth test is removed; its shared TLS owner teardown
helper remains for other live tests. Physical audits exclude native storage.

Validation: full native polynomial (35.99s), focused native GCD/ICCP/manager
checks (3.24s), checked/debug feature checks (4.43s), module-wide test compilation,
polynomial vet and portcheck passed. The full native module checkpoint remains
19:07–19:12 UTC; another hourly checkpoint is not yet due. Manifest counts remain
2102 units / 677 APIs: 63 ported, 284 partial, 1727 unported and two dead/deleted.
No whole-unit completion claim or replacement container package is added.

## Native composition degree positions

Composition's persistent degree-to-term-position member now uses `[]uint32`.
Removed its raw vector header, custom Runtime growth/pending blocks and member
free failures. New entries retain the maximum-unsigned missing-degree sentinel;
save/reset and Horner arithmetic keep source term positions and evaluation order.
Actual polynomial/monomial reads can still stop construction, so partial stored
position cleanup remains. Source/owning member destruction clears the native
slice at the existing lifecycle boundary; physical audits exclude its storage.

New tests check degree 257 with Runtime reservations disabled, actual coefficient
lookup for present degrees, missing-degree sentinels and reset/reuse. Independent
sparse p(y^2) results cover increasing/decreasing degrees 17, 65, 257, 33 and 17.
Existing cancellation, stopped child, ownership and actual allocation-failure
campaigns pass. The unsigned n+1 wrap safety test now expects arena.ErrBounds
in release mode; its source debug assertion and allocation-free failed-constructor
behavior remain. No whole-unit completion credit is added.

Validation: focused native composition/manager checks (0.12s), focused debug
checks (0.13s), full native polynomial, module-wide test compilation, polynomial
vet and portcheck passed. The hourly full native module checkpoint remains
19:07–19:12 UTC; another checkpoint is not yet due. Manifest counts remain
2102 units / 677 APIs: 63 ported, 284 partial, 1727 unported and two dead/deleted.
The lexicographic sorter still has nested vector scratch and a borrowed raw
permutation mapping; convert those together next, preserving its project sorting
algorithm and coefficient/monomial pairing.

## Native recursive lex-sort scratch

Polynomial lex sorting now stores degree buckets in `[][]uint32` and its mapping
in `[]uint32`. Deleted the outer class/binding directory, copied/moved class
metadata, custom Runtime headers/blocks, interrupted expansion state and nested
buffer free failures. Source and owning member destruction clear native slices
at their existing boundaries. Stopped-invocation cleanup clears only its scratch;
it does not undo polynomial publications.

The project recursive highest-variable/descending-degree algorithm and signed
in-place permutation cycles remain. Coefficient and monomial columns retain
actual borrowed payload guards and representation swaps. Independent native
checks cover cold/warm recursion and a 405-term three-variable grid with Runtime
reservations disabled. Big numeral identity, failed-null numeral representation,
cycle marking before a payload guard, captured object lifetime and wrapped-degree
bounds coverage remain.

Retired tests of eliminated vector allocation/capacity/spare-mapping topology.
Evaluator stale/early-invocation tests now use the existing wrapped-degree bounds
stop to retain a real LexSort generation. Their actual numeral Copy allocation
campaigns, receipt protection and root ownership checks remain. An early-route
check no longer searches allocation ordinals for the removed lex-vector failure.
The bounds fixture is an explicitly guarded invalid-input domain, not ordinary
mathematical exhaustion or defined C++ exception behavior.

Validation: full native polynomial (36.84s), focused native and debug lex/evaluator/
manager checks, module-wide test compilation, polynomial vet and portcheck passed.
The latest full native module checkpoint remains 19:07–19:12 UTC; its next hourly
run is not yet due. Manifest counts remain 2102 units / 677 APIs: 63 ported,
284 partial, 1727 unported and two dead/deleted. No whole-unit completion claim
or replacement container package is added.

## Native polynomial factor entries and multiplicities

`PolynomialFactors` now stores `[]Polynomial`, `[]uint32` multiplicities and
native byte-sized cleanup progress. Deleted the pointer/degree byte-vector
codecs, Runtime expansion/pending roots and charged phase growth. The last
shared polynomialUnsigned append/read helpers are gone. Actual polynomial INC,
DEC, destructor continuation, numeral constant ownership and invoked child
receipts remain. Push retains pointer/degree publication then total update then
INC; SetDegree retains the source index-zero assertion and unsigned wrap.
Source/owning destruction releases native slices at the original boundaries.

A cold/warm ownership regression grows through 4097 duplicate entries, updates
a multiplicity, checks exact total/reference counts and term-degree association,
then resets and reuses storage. Native storage is excluded from custom Runtime
root/byte audits. The zero-factorization no-change test snapshots slice contents.
Removed tests whose required failure existed only between pointer and degree
vector allocation, including the resulting quadratic borrowed dangling row.
Actual quadratic allocation/cancellation, factor multiplication/display/constant
allocation and stopped reference/numeral destructor campaigns remain.

Validation: focused native factors/quadratic checks (0.20s), corresponding debug
checks (1.08s), polynomial vet and portcheck passed. The hourly full native module
checkpoint ran 20:08–20:11 UTC and passed all packages. Manifest counts remain
2102 units / 677 APIs: 63 ported, 284 partial, 1727 unported and two dead/deleted.
No whole-unit completion claim or replacement container package is added.

Next concrete storage group: skeleton, SOM and Newton class-owner vectors still
encode child IDs in Runtime vectors while retaining the actual typed children
in separate directories. Replace that container bookkeeping with native typed
storage while preserving real child construction and source/owning destruction.

## Native polynomial child collections

SOM, Newton and skeleton owner collections now use typed Go pointer slices.
Removed child-ID byte vectors, duplicate lookup directories, charged directory
capacity and their expansion/free failure states. Actual child constructors,
references, numeral backing and stopped destructor continuation remain.
SOM keeps its children through source destruction for later owning completion.
Newton retains a separate slice only for actual unjoined resources left by a
completed source Flush. Skeletons retain published child contexts after source
replacement/destruction until their actual owning cleanup completes.

Native slice storage is excluded from Runtime root and custom byte censuses.
New checks grow skeleton storage through 4098 slots with Runtime reservations
disabled, verify child association and replacement/shrink lifetime, and confirm
that 129 lazy SOM children charge only their actual class headers. Real Newton
constructor failures, unjoined references and stopped child destructors remain
covered. Retired fixtures requiring eliminated pointer/directory allocations or
TLS failures in freed pointer-vector backing; updated the inventory accordingly.

Validation: full native polynomial passed (36.09s); final focused native and debug
owner/lifetime checks including the new growth tests passed (1.16s/1.12s).
Module-wide test compilation, polynomial vet and portcheck passed. The latest
full native module checkpoint remains 20:08–20:11 UTC; another hourly run is not
yet due. Counts remain 2102 units / 677 APIs: 63 ported, 284 partial,
1727 unported and two dead/deleted. The util knot and package split remain open.

Next: concrete skeleton payload arrays and polynomial reference/numeral vectors
still use byte-encoded Runtime backing. Replace their container mechanics while
preserving project algorithms and actual reference/numeral ownership.

## Native polynomial skeleton payload

`PolynomialSkeleton` now stores typed entries, uint32 powers and original
monomial handles in native slices. A borrowed local monomial slice and
`sort.Slice` replace the inline128 buffer and sorting adapter. The original
LexWithMinimum grouping, entry order, powers, unsigned indices, monomial INCs
and entries-before-originals DECs remain. Comparator dispatch order is not
part of the port contract.

Deleted byte codecs for these payload arrays, Runtime growth/free state,
retired-reference copies, custom slice charges and constructor transfer cursors.
A failed constructor retains its actual acquired references in the same typed
payload for explicit owning cleanup; it does not run the source class destructor.
Real generated-monomial construction failure and stopped reference destruction
remain covered. Native payload bounds replace reads of unused vector capacity;
unsigned index addition still wraps before checking the resulting index.

The 129-term pure-power regression permits only the owner-header reservation,
verifies no Runtime allocations, checks canonical power order and audits actual
references through source/owning destruction. Actual display callbacks test the
reached owner lifetime. Removed malformed raw uninitialized-entry/header fixtures
for storage that no longer exists. Physical censuses count real child resources.
The shared byte-backed monomial buffer still serves sparse interpolation and
cross-manager conversion; its existing code is isolated in
`polynomial_monomial_buffer.go` until those consumers are converted.

Validation: full native polynomial passed (35.67s), final focused debug skeleton/
sparse/GCD/conversion/imp checks passed (1.18s), and module-wide test compilation,
polynomial vet and portcheck passed. The full native module checkpoint remains
20:08–20:11 UTC, not yet due again. Manifest counts remain 2102 units / 677 APIs:
63 ported, 284 partial, 1727 unported and two dead/deleted. The complete util knot
and package split remain open. Next: remove the remaining shared monomial buffer
from its actual conversion/interpolation consumers, then remaining concrete
reference/numeral vector storage.

## Native borrowed monomial locals

Cross-manager conversion and sparse interpolation now collect borrowed monomial
handles in `[]Monomial`. Conversion's separate native orphan slice retains only
actual newly created zero-reference monomials, without custom capacity charges.
Actual coefficient/numeral ownership, solver algorithms, degree filtering,
monomial conversion and polynomial factory publication remain unchanged.

Deleted the last shared inline128/spill monomial buffer and its codecs, Runtime
expansion/pending/free paths, and the unused inline/raw factory span variants.
The borrowed factory view accepts native handles and its remaining concrete
polynomialVector consumer. Source destruction clears the native locals after the
appropriate younger numeral/solver destruction; owning continuation retains real
numeric/factory/monomial roots and exact foreign-owner protections.

A 4097-entry lexical conversion regression grows both native locals with Runtime
reservations disabled and checks the real returned factory root through source
and owning cleanup. The former TLS pointer-buffer-free stop is retired with that
removed backing. A 129-group single-sample sparse interpolation reconstructs the
independent polynomial twice and checks all coefficient/monomial associations.
Native borrowed-span tests preserve reset-before-read and unread-tail behavior.

Validation: focused native and debug conversion/sparse/indexed-factory/GCD/imp
checks passed (1.06s/1.10s), as did module-wide test compilation, polynomial vet
and portcheck. The last full native polynomial checkpoint passed at the previous
chunk (35.67s); the full native module checkpoint remains 20:08–20:11 UTC and is
not due again yet. Counts remain 2102 units / 677 APIs: 63 ported, 284 partial,
1727 unported and two dead/deleted. The complete util knot and package split
remain open. Next: remaining concrete polynomial reference and numeral vectors.
