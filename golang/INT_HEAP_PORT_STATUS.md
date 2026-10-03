# Concrete integer heap

This packet translates src/util/heap.h into IntHeap and preserves every active
assertion in src/test/heap.cpp. No Go generics, unsafe, interfaces or Go pointers
appear in the int32 value/inverse-index arrays. Cold owner fields contain the
Budget, borrowed Less callback, checked epoch and retained query workspace.
An owning heap must not be copied as a Go value; use Clone or Assign. Callers
serialize operations and shared budgets, and keep comparator environments alive.
Comparators must be strict, nonreentrant and nonpanicking.

## Source method and storage audit

- Constructor pushes sentinel -1 before resizing the inverse vector; destructor
  maps to idempotent Close. empty/contains/get_bounds/min_value map to
  Empty/Contains/Bounds/Min. Invalid keys/empty minima return typed preconditions.
- move_up keeps strict comparator ordering; move_down chooses right only when
  strictly less than left, retaining the source left-child tie rule.
- insert/erase/erase_min/decreased/increased preserve source value and inverse
  position assignments and comparator notification directions. No priority is
  copied into a heap record. Source mutable-priority callers update the borrowed
  priority environment then notify the appropriate direction.
- reset/clear preserve both capacities and zero the complete logical inverse
  vector. set_bounds/reserve retain source shrinking/resizing/no-op behavior;
  unsafe live-key shrink is refused before mutation. Newly exposed indices are
  zeroed, including regrowth of retained capacity. Negative Reserve is a no-op.
- Source vector resize repeatedly expands 2,3,5,8,12,18,27,41,..., rather than
  allocating the requested minimum directly. Each successful expansion remains
  retained on later failure; logical bounds remain unchanged. Source unsigned32
  vector allocation-size overflow (including two source header words) and host
  int bounds are checked before allocation. Source vector header metadata is
  represented by cold owner slice fields, already included in the header charge.
- begin/end map to cold checked owner/epoch iterators; Next/Value/Equal and
  AppendValues preserve heap-array order without returning an internal slice.
  Raw mutable pointers become readonly checked values; callers may not corrupt
  the source inverse-index invariant by changing a borrowed array element.
- swap exchanges only values and inverse indices. Comparators, budgets, headers
  and retained FindLE scratch stay with their owners; incoming comparators must
  be compatible as required by source debug invariant checks. Cross-budget swap
  reserves both complete incoming vector graphs while outgoing charges remain,
  rolls back the first on second failure, then publishes infallibly.
- Source implicit deep copying is explicit Clone/Assign. Clone copies comparator
  and exact vector capacities, but not query scratch (source locals). Assign
  stages a charged clone before publishing, preserving destination on predictable
  failures rather than exposing source partially destroyed vectors. Its peak
  includes a temporary cold owner header. Destination query scratch is retained.
- find_le uses the source LIFO traversal: append node, push left then right;
  output is right-first and unsorted. Source local todo and caller-result growth
  become retained budgeted todo/found vectors followed by caller publication.
  Source pushes of representable beyond-heap children are retained. Signed-shift
  overflow children are discarded instead of invoking source undefined behavior.
  Failed preparation leaves all caller output bytes/elements unchanged while
  successful scratch capacity prefixes remain charged. Returned output belongs
  to the caller; no internal backing escapes.
- display preserves recursive preorder, indentation and decimal/newline spelling.
  Its result is caller-owned diagnostic output. check_invariant checks the source
  parent ordering and inverse mappings, plus the independent reverse mapping.

Every retained backing capacity and reflect-sized cold owner header is charged.
Replacement allocation charges full old+new capacity. Close refunds the complete
graph even after partial growth and at exhausted epoch; it drops comparator borrows
without invoking callbacks. Structural/reordering mutations refuse epoch wrap
before publication; readonly/no-op actions remain available.

## TDD and gates

Four initial tests failed on missing heap APIs before translation (RED), then
passed (GREEN). Both original fixtures preserve N=10000 and seeds0/1/2, the
30000 insertion attempts, original100000 mutable-priority commands, unchanged RNG
draw sequence, all active source assertions, and exact100100-line print shape
captured in test-only output. Additional sorted extraction and an independent
5000-step membership/minimum oracle check successful behavior.

A source review then identified wrong direct-minimum resize allocation. The
capacity32-to41 regression failed before the fix, then passed with exact later
62/140/12137 capacities and retained successful failure prefixes. Copy and
allocation-size APIs independently failed compilation before implementation.
Nine tests cover failure prefixes/constructor unwind, staged query/copy/assignment,
exact vector capacities, old+new finite peak refusal, both cross-budget swap
reservation failures, comparator retention, bounds/regrowth, stale iterators,
epoch exhaustion, output order and warm zero-allocation mutation/traversal.

Combined full native module tests pass. Actual g3checked races pass for all
packages: after correcting a finite-peak test fixture that itself prevented
construction, the changed root package was rerun successfully; the other packages
passed the full race run. Full actual g3checked JS/WASM module execution passes,
with all9 heap,15 scoped-table,12 numeric-storage and18 string-table test names
explicitly checked for registration and PASS in the verbose log. Checked vet,
WASI cross-build and source/no-generics audit pass. WASI is compilation evidence;
its previously documented runtime startup gap remains open.

This completes the integer heap utility, not its solver consumers or any
AST/solver/proof/interpolation milestone. The full port goal remains active.
