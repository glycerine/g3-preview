# SAT lazy clause use list — selected implementation

`sat_solver_clause_use_list.go` translates the actual `LAZY_USE_LIST` selected
by `src/sat/sat_clause_use_list.h/.cpp`, using the committed clause allocator
and clause-vector owners. No solver, child-solver callback, runtime constructor,
or Check substitute is introduced. The source headers remain **partial**:
implicit move APIs, the inactive non-lazy branch, and STRACE integration remain
open. The full SAT solver and its user-scope consumer are still open.

## Implemented source surface

| Source | Concrete Go API/action |
| --- | --- |
| `clause_use_list()` | `newSATClauseUseList(allocator, budget)` |
| `size()` / `empty()` | `Size()` / `Empty()` read the independent unsigned live counter |
| `insert(c)` | `Insert(ref)`: source debug contains/removed assertions, vector push, then counter increment |
| `erase_not_removed(c)` | `EraseNotRemoved(ref)`: source assertions, physical erase, then decrement |
| `erase(c)` | `Erase(ref)`: source assertions, counter-only lazy decrement |
| `reset()` | `Reset()`: vector finalize before publishing counter zero |
| `check_invariant()` | `CheckInvariant()`: actual removed-flag traversal in **both** builds; final equality assertion only in debug |
| `mk_iterator()` / iterator constructor | `Begin()`: capture physical vector length, then immediate real `consume()` |
| `at_end()` | `AtEnd()`: plain `i == capturedSize`, no backing/lifetime dereference |
| `curr()` | `Current()`: actual current-vector scalar read, no additional removed-flag callback |
| `next()` | `Next()`: source debug assertions, `i++`, `j++`, then consume |
| `consume()` | real skip-removed loop; each surviving current reference is written to compacted position j before returning |
| iterator destructor | explicit `Close()`: finish all remaining Next actions, then vector Shrink(j) |
| legal implicit copy constructor / assignment | `Clone(budget)` / `Assign(source)` via actual vector copy, followed by live counter copy |
| implicit list/vector destruction | owner `Close()`: release actual owned vector/header, never delete referenced clauses |

Every source debug assertion site is exercised. Source release bodies are
preserved: duplicate/removed insertions can execute, missing physical erases
still decrement, repeated lazy erased-reference decrements wrap unsigned32,
and invariant equality does not become a new release precondition.

The class has no explicit move declaration; legal implicit move inventory is
left open until its exact compiler-instantiated layout/copy-forwarder routes
are separately tested. In particular, the existing ptr_vector move-assignment
API is a source copy forwarder and is not silently used as underlying vector&&
transfer. The actual simplifier `use_list::init` resets/destroys the outer list
elements **before** resizing, so its ordinary growth starts with zero logical
elements; it does not require relocating live list instances in this packet.

## Iterator and lifetime contract

The iterator is returned **by value**, with a borrowed stable `*satClauseVector`
and scalar captured-size/i/j/closed fields. It does not allocate an uncharged
heap iterator, capture a source backing pointer, pin the original backing,
cache a current clause reference, or reject vector growth by epoch. Reads and
writes resolve the vector object's **current** backing on every source action.
Underlying source operator[] debug guards use the **current** logical size;
release reads/writes use the existing concrete vector's physical bounds and
retained scalar slots. Tests demonstrate growth, backing Swap, and public
implicit list assignment during an active iterator.

Captured size is vector length, not the lazy live counter. Begin already skips
removed prefixes and publishes the first live scalar to compacted index zero.
Each Next performs the same immediate publication; Close completes traversal
even if the caller stops early. Close does not repair the live counter if the
caller appends while iterating: its captured-end final Shrink can discard that
appended physical tail while leaving the increased counter, as source does.

Go has no automatic C++ lexical destructor. Callers must explicitly Close a
successfully constructed iterator at source lifetime end and must not copy a
live owning iterator. Idempotent Close/closed-state errors are explicit Go
lifetime adapters. AtEnd is still plain scalar comparison after vector closure;
actual destructor Shrink then reports the concrete closed-vector error.

Current copies a retained scalar without dereferencing its clause payload.
An actual subsequent was_removed/other clause access performs the allocator
generation/lifetime guard. A stale payload encountered by consume or invariant
traversal reports the actual guard error after prior skips/publications.
Failed Begin returns the partial scalar iterator alongside its error; Next and
Close expose their already completed index/write prefixes and do not roll them
back. An interrupted invalid-payload construction/traversal is not advertised
as a valid, generally retryable C++ iterator. No clause ownership is lost:
clauses remain with their actual allocator/caller, and list Close can release
its own backing independently. No hidden recovery traversal or always-true
removed callback substitutes for the source.

## Storage and budget

One cold list header owns one actual `satClauseVector` header/backing. Both
actual `reflect.TypeOf(...).Size()` headers are charged once; the list header
contains only cold owner pointers and the scalar live counter. Clause tokens
in backing remain pointer-free. The vector's real allocator-binding ledger is
retained/copied as necessary; copied callbacks resolve the actual borrowed
allocator, including across two real allocator owners. No clause payload or
source phantom inline header is charged to the list twice.

Underlying vector geometry retains source requested capacity growth and source
pointer-width limits separately from actual wider Go token byte limits. Full
old and new backing charges overlap during expansion. A permanent one-byte-
below-peak fixture refuses capacity2→3 growth with the old vector/counter and
caller-owned clause unchanged. Reset releases backing before counter zero;
Close releases only owned headers/backing/bindings. Source unsigned size is
not normalized, clamped, or recomputed by these lifecycle methods.

Constructors and Clone reserve all actual headers/backing and unwind predictable
allocation failure without losing caller clause roots. Assign preserves the
source member order: vector finalization/copy precedes the scalar counter copy.
Failure can leave an empty destination vector with the previous live counter;
it does not invent an atomic source assignment. Warm iteration and explicit
destruction allocate zero heap objects and make no new Budget reservation.

All actual allocator callbacks are the existing Removed/Get/Set operations.
Payload, owner, epoch-exhaustion and physical-bounds errors are typed safety
adapters from those concrete owners, observed at actual action boundaries.
There are no per-element interface boxes, Go generics, unsafe operations,
mock child solvers, fake counters, or production truth-table algorithms.

## Source tests and verification

No registered direct `clause_use_list` source test/assertion was found in
`src/test`; original source assertions claimed here: **zero**. The source
simplifier remains an actual unported consumer. New tests derive complete
finite action/assertion cases from the preserved methods and include an
independent stable-subsequence oracle over all 64 removal patterns, with
different early iterator destruction points.

Tests were written first and observed actual missing-API RED on isolated
committed HEAD `2c89d3f53dc790bf3af247d20702dd683ad6e3c6`: terminal handle 2142,
`.cache/sat-clause-use-list-red.log`. Initial14-group native GREEN2767 and
checked/debug GREEN49700 passed. Expanded18-group native27562/debug50870 passed.
The final 20-group inventory additionally verifies physical peak accounting and
public assignment/current-backing/real cross-allocator binding semantics.

Final snapshot is `.cache/release/sat-clause-use-list-final`, the same committed
base plus **only** the two owned Go files. Existing configuration/registry,
storage owners, central metadata, and concurrent memory/polynomial drafts are
unchanged/excluded. Final named inventory, hashes and terminal gate evidence
are recorded below when completed.

Only writes/artifacts under golang; no Git/central manifest edits.

## Frozen final gate evidence

All terminal PASS on the final two Go file hashes below:

| Gate | Handle | Result |
| --- | --- | --- |
| Full native `./...` | 29103 | PASS; g3 2.588s |
| Full `g3checked,g3debug ./...` | 55228 | PASS; g3 3.005s |
| Exact20 checked/debug race | 98554 | PASS; 1.070s, all expected RUN/PASS registrations |
| Actual exact20 checked/debug JS wasm | 28077 | PASS; 0.176s, all expected RUN/PASS registrations |
| Checked/debug vet `./...` | final synchronous result | PASS |
| Checked/debug WASI test compile | 65460 | PASS; `.cache/sat-clause-use-list-final.wasi.test` |
| Source + no-generics audit | 19773 | PASS; committed baseline 33/102/1967 |

The prior19-test round also passed fullnative48280/fullcheckeddebug27158,
race59658/actualJS79704/vet45216/WASI45339/audit17691. The final test-only
cross-allocator assignment addition passed targeted51819 before refreshing
all final gates; no production bytes changed during either final round.

Logs are `.cache/sat-clause-use-list-final-{native,debug,race,js,vet,wasi,audit}.log`.
`.cache/sat-clause-use-list-final-testnames.txt` records the exact20 named
inventory; both race and actual JS RUN/PASS sets were checked against those
names, not inferred from a package-only success line. The working files and
prospective snapshot bytes match exactly; `gofmt` and source/no-generics are clean.

Frozen SHA256 inventory `.cache/sat-clause-use-list-final-sha256.txt`:

```text
71750ac813ee38690ea479ea27c0e1d3eebf4d9efe23179986c7c7842cd965b0  sat_solver_clause_use_list.go
cc98929cca2fb59816b6cb47310400ab341755e803fdb18f5807abaa8bebbe9c  sat_solver_clause_use_list_test.go
```

The parent owns source metadata/integration/commit. Selected methods are
verified; whole-source headers remain partial with the explicit open profiles
and implicit move inventory above.

## Coordinator integration

Snapshot `.cache/review/sat-clause-use-list-integration-53047b9` adds only two
frozen Go files to the committed complete catalog53047b9. Full native15409
and checked-debug96734 passed (mpnum21.972s/22.804s). Coordinator verified both
Go hashes and all twenty exact race/actual JS RUN/PASS records. Updated
source/no-generics audit11844 passed, ledger33ported/142partial/1927unported.
Only selected explicit/copy/lazy iterator methods are covered; implicit moves,
nonlazy/TRACE and complete default SAT/SMT/proof/interpolation remain open.
