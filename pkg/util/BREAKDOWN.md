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

Tiny C++ headers are not automatically good Go packages. If a facility has one
consumer, prefer putting a concrete implementation with that consumer. If it
has no production consumer, delete or defer the port instead of giving dead
code a new home.

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
small runtime and memory primitives
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
| `util/mincut` | `min_cut` | Self-contained algorithm. Move when its `muz` consumer is ported; there is no current production Go consumer. |
| `util/toposort` | `top_sort` | Live in the AST code. Specialize the present `any`-based Go API for its real node type rather than preserving a generic C++ template shape. |
| `util/natset` | `nat_set` | Self-contained representation. Move only with a live consumer. |
| `util/approx` | `approx_nat`, `approx_set` | Cohesive approximate-number/set functionality. Their display methods currently touch shared numeric formatting; replace that edge with ordinary `fmt`/`io` behavior or a narrow formatter. |
| `util/backtrack` | `trail`, `union_find`, selected backtracking containers | `union_find` is widely used upstream. Keep trail ownership and undo behavior together. Specialize `backtrackable_set` for its actual consumer instead of keeping `any`. |
| `util/bitset` | `bit_util`, `bit_vector`, `fixed_bit_vector`, `obj_mark` | Natural bit-oriented cluster. Check whether fixed-width manager state needs the shared allocator before moving it. |
| `util/cliques` | `max_cliques`, its set helpers | Already has SAT consumers. A focused package would isolate the algorithm and its tests. |
| `util/sortnet` | `sorting_network` | Standalone algorithm with production consumers outside `util`. |
| `util/permutation` | live declarations from `permutation.h` | Only move the live free permutation operations. `permutation.cpp` itself has no production caller. |
| `util/stack` | `stack` | Put the arena-backed implementation in a package that imports `pkg/arena` directly. |

Very small facilities such as `file_path`, `luby`, and `version` do not justify
a package merely because they are graph leaves. Put them with their consumers,
or leave them in a small root package until several related facilities form a
coherent boundary.

The current Go versions of `list`, `scoped_vector`, and `backtrackable_set` use
`any`. Each C++ template has a small set of concrete production instantiations.
When their consumers arrive, specialize them there or create concrete packages.
Do not preserve unbounded template generality in the port.

### 2. Symbols, parameters, statistics, and control

These should follow after the first independent moves:

* `util/symbol` can own `symbol`, `symbol_table`, and their Go-native string
  handling. It must not reintroduce C++ string classes.
* `util/params` can own `params` and environment parameter parsing and import
  `util/symbol` if necessary.
* `util/stats` can own `statistics` and `stats`. Its use of symbols, maps, and
  numeric values must be made explicit rather than reaching back into root
  `util`.
* `util/control` can own cancellation, event handlers, resource limits,
  timeouts, timers, scoped control-C handling, and cooperation checks. These
  form a behavioral subsystem even though they are not one C++ include cycle.

These APIs have broad consumers, so moving them will touch many packages. They
are valuable boundaries, but poor first experiments for the mechanical package
move process.

### 3. Runtime and memory primitives

The allocator, failure, exception-equivalent, and buffer code needs a narrow
home below all feature packages. A possible package is `util/mem`, importing
[`pkg/arena`](../arena) and the standard library only. Its public surface should
represent Go ownership and failure behavior rather than C++ allocation syntax.

Do not turn the entire 11-unit C++ cycle into a permanent `util/base` package.
That would reproduce the current junk drawer under another name. Standard Go
maps, slices, strings, errors, logging, and testing should replace the C++
container and platform wrappers where their semantics are sufficient.

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
* Numeric cells and allocation policy depend on `arena` and shared runtime
  failure behavior.

The first move should therefore be one package, tentatively `util/num`, holding
the live integer, rational, floating-point, interval, and numeral-lifetime code.
It should import `arena`, the narrow memory/runtime package, and independently
extracted bit primitives. It must not import the parent `util` package.

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
error_codes       event_handler      ext_gcd             file_path
luby              machine            obj_ref             optional
pop_scopes        ref                ref_util            resource_limit
scoped_numeral    sstream            stats               stopwatch
stream_buffer     version            z3_omp
```

This list does not mean every item should become a Go foundation package.
`dlist` and `resource_limit` were confirmed dead and have already been removed
from the port inventory. `critical_flet`, `pop_scopes`, and `sstream` have no
known production source consumer and should be treated as deletion or deferral
candidates. `ext_gcd` is test-only in the surveyed C++ tree. Several remaining
items are simple enough to replace with standard Go facilities.

## Complete extraction-leaf inventory

No other direct `src/util` logical unit includes any of these 73 units. The list
is grouped by likely disposition, not by a proposed one-package-per-row rule.

### Strong package candidates

```text
approx_nat             approx_set              backtrackable_set
cancel_eh              checked_int64           cooperate
dependency             double_manager          env_params
f2n                    fixed_bit_vector        heap
hwf                    inf_eps_rational        inf_int_rational
inf_s_integer          lbool                   list
luby                   max_cliques             min_cut
mpbqi                  mpff                    mpfx
mpq_inf                mpzzp                   nat_set
numeral_buffer         parray                  plugin_manager
pool                   prime_generator         scoped_ctrl_c
scoped_numeral_buffer
scoped_ptr_vector      scoped_vector           sexpr
sorting_network        stack                   statistics
stats                  symbol_table            timeit
timeout
timer                  top_sort                union_find
version
```

Many of these are endpoints only within `src/util`; they have real consumers in
AST, SAT, SMT, tactic, solver, shell, or `muz` code. For example, `lbool`,
`statistics`, cancellation/cooperation, `heap`, and `union_find` all have broad
upstream use. Package boundaries must preserve that outward API.

### Small units best placed with a consumer or cohesive neighbor

```text
array                  array_map               container_util
dec_ref_util           ext_numeral             file_path
obj_mark               obj_pair_hashtable      obj_pair_set
obj_triple_hashtable   pop_scopes              ref
ref_buffer             ref_util
sstream                stream_buffer
```

These are often C++ templates, ownership wrappers, or compatibility helpers.
Their Go form should be a concrete consumer-owned type, an ordinary slice/map,
or part of a larger behavioral package.

### Dead, test-only, or requiring proof before retention

```text
critical_flet          dlist                   ext_gcd
obj_ref_hashtable      object_allocator        permutation
ptr_scoped_buffer      resource_limit          total_order
```

`dlist` and `resource_limit` are confirmed dead and deleted. Prior whole-tree
surveys found `ext_gcd`, `object_allocator`, `ptr_scoped_buffer`, and
`total_order` used only by tests. `critical_flet` and `obj_ref_hashtable` have no
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

The best proof of the approach is a small first extraction such as `lbool`. It
has real consumers, no numerical ownership complications, and no reason to
depend on the remaining root package. Once that move is green, the same process
can be applied to cliques, union-find/backtracking, sorting networks, and then
the larger numeric boundary.
