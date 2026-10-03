# Resource-limit service

`util_rlimit.go` translates `src/util/rlimit.h/.cpp` (Microsoft 2015,
Nikolaj Bjorner). This is a cold owner service, not a solver or public status API.
It uses concrete types throughout. Source has no dedicated rlimit assertion
suite; the new tests are finite source-derived expectations and an independent
bounded stack oracle. Source consumer tests remain with their unported components.

| Source operations | Go counterparts / evidence |
| --- | --- |
| constructor and implicit vector destruction | newUtilReslimit, close; header, limit and child vectors budgeted |
| inc(), inc(unsigned), count | inc, incOffset, count; inclusive bounds and uint64 wrap |
| push/pop | push, pop; relative bounds, nested minima, count clamp, local cancel clearing |
| get_cancel_flag/get_cancel_msg | cancelFlag/cancelMessage; exact source messages, suspension distinction |
| cancel/inc_cancel/dec_cancel/reset_cancel/set_cancel | atomic uint32 state, process-wide cancellation mutex, recursive propagation |
| push_child/pop_child | borrowed child stack under the same process-wide mutex |
| scoped_rlimit | utilLimitScope with explicit deferred close |
| scoped_suspend_rlimit | utilSuspendLimit with explicit deferred close |
| scoped_limits | utilChildLimits with counted LIFO detachment |

Counting, push/pop, suspension and destruction require the operation owner's
confinement. Cancellation and child registration share the source OpenMP critical
section's process-wide scope; atomic loads/stores replace the C++ volatile flag
so asynchronous cancellation is race-safe in Go. Child services form an acyclic
borrow graph, outlive registrations, and must not close while registered. The
source has the same implicit graph/lifetime precondition. No individual hot
record contains an interface, closure or mutex. Copying initialized services or
active scope values is invalid; explicit owner lifetimes replace C++ destructors.

Preserved executable source behavior:

- Counting always increments even when canceled/exhausted/suspended, modulo 2^64.
- Push computes count+delta, maps zero/wrap to unlimited zero, then applies the
  literal source minimum. A nested delta=0 can therefore clear an outer bound.
- Pop clamps count to the departing nonzero bound before restoring its parent.
- Push/pop clear cancellation only locally; descendants keep their flags.
- Registration does not inherit a parent's existing flag. The next propagation
  overwrites every descendant with the parent's counter, including uint32 wrap.
- get_cancel_flag ignores exhaustion; get_cancel_msg ignores suspension.
- The suspension destructor sets false rather than restoring a previous value;
  nested suspension consequently clears the outer one. An explicit test freezes
  this source defect. Do not silently replace it with nesting depth.

Storage adaptation: constructors charge cold header bytes, and geometric vector
replacement charges old+new backing capacity before releasing old storage.
Budget failures preserve limits, cancellation and child publication. Popped
child pointers are cleared to avoid Go retention beyond the source borrow.
Close detaches borrowed children without destroying them and releases all owned
charges. Idempotent explicit scope/owner close prevents repeated deferred cleanup;
C++ RAII destruction would occur once. Invalid underflow, closed use, nil/self
children and unrepresentable capacities are development errors. Other cycles
remain an invalid caller precondition. These changes do not alter valid source
counting/cancellation semantics. Host runtime OOM recovery is not claimed.

TDD evidence: tests first failed with undefined service/scope constructors and
methods. Implementation reached GREEN. Tests cover all translated methods,
4000 deterministic bounded stack steps, source wrap/suspension quirks, late
registration, exact messages, failed constructor/push/registration, replacement
peak budgeting, complete cleanup, warmed zero allocations and four concurrent
balanced cancelers alongside counting. Native checked race/vet and actual
js/wasm main-package execution pass. Full integration gates are recorded in
PORT_PROGRESS.md; WASI compilation is not runtime acceptance.

The general common_msgs class (other strings), cancellation event-handler
families, timeout/memory services and domain consumer integration remain separate
source units. This packet supplies resource counting/cancellation only.

## Concrete cancellation event scope

`util_cancel_handler.go` translates `cancel_eh<reslimit>` and the event caller
IDs from `event_handler.h`. Its cold handler borrows a resource service and uses
explicit deferred close in place of its destructor. The first invocation fixes
the caller ID and increments cancellation once; later invocations have no effect.
Only an invoked handler decrements on close, and the original canceled/caller
fields remain observable. Nested handlers and external counter reset preserve
source behavior. Handler invoke/close must be serialized; counter propagation
retains the resource service's asynchronous safety. No template/generic or
interface dispatch is introduced. Other event-handler specializations and the
OS timer/signal dispatch remain separate unported units.

Added finite tests (no dedicated source suite) went RED on missing handler/caller
symbols before implementation, then GREEN. Tests cover first-caller order, quiet
handler cleanup, nested propagated cancellation, reset before cleanup, exact enum
numbers and zero warmed allocations. An isolated prospective-commit tree is used
for whole-module gates while other agents have intentionally RED tests in progress.
