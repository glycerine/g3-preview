# Breaking `util` into focused Go packages

This document analyzes the direct C++ sources in [`src/util`](../../src/util) and
the corresponding Go port in this directory. It is a guide for later package
moves; it does not propose moving files mechanically or preserving C++ utility
code that has no production consumer.

## Conclusion

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
`util/messages`, `util/natset`, `util/stack`, `util/bitset`, `util/luby`, and
`util/statistics`.
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
| `util/toposort` | `top_sort` | Live in the AST code. Specialize the present `any`-based Go API for its real node type rather than preserving a generic C++ template shape. |
| `util/natset` | `nat_set` | Self-contained live representation and a natural package leaf. |
| `util/approx` | `approx_nat`, `approx_set` | Cohesive approximate-number/set functionality. Their display methods currently touch shared numeric formatting; replace that edge with ordinary `fmt`/`io` behavior or a narrow formatter. |
| `util/backtrack` | `trail`, `union_find`, selected backtracking containers | `union_find` is widely used upstream. Keep trail ownership and undo behavior together. Specialize `backtrackable_set` for its actual consumer instead of keeping `any`. |
| `util/bitset` | `bit_vector`, `fixed_bit_vector`, `uint_set` | Extracted. Native typed storage; fixed-width vectors import `arena` directly. `bit_util` is a Go platform replacement and `obj_mark` is represented by concrete AST specializations. |
| `util/cliques` | `max_cliques`, its set helpers | Already has SAT consumers. A focused package would isolate the algorithm and its tests. |
| `util/sortnet` | `sorting_network` | Standalone algorithm with production consumers outside `util`. |
| `util/permutation` | live free operations from `permutation.h` | Extracted as concrete `uint32` slice operations. `permutation.cpp` itself has no production caller. |
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

Very small leaf facilities such as `file_path`, `luby`, and `version` should
still become distinct Go package leaves. This makes their consumers visible in
the import graph, gives them an enforceable boundary, and makes later deletion
straightforward. Package size alone is not a reason to leave unrelated code in
root `util`. These three leaves have now been extracted.

The current Go versions of `list`, `scoped_vector`, and `backtrackable_set` use
`any`. Each C++ template has a small set of concrete production instantiations.
That use of `any` was an earlier porting mistake and should be eliminated.
Replace these containers with concrete slices, Go maps, or
`insdict.Dict` at v0.10.1 when stable iteration is required. When a wrapper still
has meaningful behavior, specialize it for its real element type. Do not
preserve unbounded template generality in the port.

### 2. Symbols, parameters, statistics, and control

These should follow after the first independent moves:

* `util/symbol` can own `symbol`, `symbol_table`, and their Go-native string
  handling. It must not reintroduce C++ string classes.
* `util/params` can own `params` and environment parameter parsing and import
  `util/symbol` if necessary.
* `util/statistics` now owns `statistics.h/.cpp`. String keys, typed slices,
  ordinary writers, and the approved insertion-ordered dictionary replace the
  C-string, vector, arena, and hash-table simulations. The package has no root
  `util` dependency. The separate source `stats` unit can join it when ported
  if its consumers need the same boundary.
* `util/control` can own cancellation, event handlers, resource limits,
  timeouts, timers, scoped control-C handling, and cooperation checks. These
  form a behavioral subsystem even though they are not one C++ include cycle.

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
reintroduced.

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
numeral_buffer         parray                  plugin_manager
pool                   prime_generator         scoped_ctrl_c
scoped_numeral_buffer
scoped_ptr_vector      scoped_vector           sexpr
sorting_network        stack                   statistics
                       symbol_table            timeit
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

### Dead, test-only, or requiring proof before retention

```text
critical_flet          dlist                   ext_gcd
obj_ref_hashtable      object_allocator        permutation
ptr_scoped_buffer      resource_limit          total_order
```

`dlist` and `resource_limit` are confirmed dead and deleted. Prior whole-tree
surveys found `ext_gcd`, `object_allocator`, `ptr_scoped_buffer`, and
`total_order` used only by tests. The Go translations of `ext_gcd`,
`object_allocator`, and `total_order` have been removed. `critical_flet` and `obj_ref_hashtable` have no
known caller. `permutation.h` has live free-function consumers, while
`permutation.cpp` does not; retain only the live behavior. Recheck the entire
C++ tree before deleting a unit because a lack of current Go consumers alone
does not prove that upstream Z3 code is dead.

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
