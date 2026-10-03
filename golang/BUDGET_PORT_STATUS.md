# Concrete owner budget integration

Arena constructors now charge the actual cold Go owners for concrete slabs,
descriptors, word/typed vector pools, regions (including separately allocated
identity), constructed pools and byte storage. Stats.OwnerBytes is additive
across nested owners; SmallBytes explicitly flattens its descriptor owner once.
Already charged directory elements include page slice headers; they are not
charged again. Embedded configs/policies, borrowed shared Budget and caller
callback environments are not separate per-owner allocations.

The integer manager and private numericCellSlab likewise reserve their actual
cold headers, unwind failed nested construction, retain ownership during Reset,
and release it exactly once at Close. Integer/cell/Words/descriptor/page counts
sum to the measured Budget. Constructor-failure sweeps verify every reservation
prefix through completion and complete rollback. Record algorithms, pointer-free
representations, generations and arithmetic do not change. These tests began
RED on absent owner accounting before the implementations.

Caller-owned scope values remain caller resources. Standalone MPN scratch,
String formatting, ID free lists, other containers and domain services still
need their own complete budgets/lifetimes; this packet does not close P1 or the
solver memory gate. The byte counts model predictable logical storage ownership,
not Go allocator rounding, retained closed wrapper objects, GC metadata or host
runtime OOM. The source ownership adaptations remain in each component status.

## Word owner accounting transfer

Words.SwapBudgets transfers complete word/descriptor graphs between accounting
destinations while preserving storage, scalar refs and source ownership actions.
Checked aggregate sums precede reservations; both destinations charge incoming
storage while outgoing storage still exists. Failure refunds prepared Used bytes
and changes no owner pointers. Successful reservation attempts still consume the
ordinary FailAfter prefix. Success rebinds both owner and descriptor budgets and
refunds old graph charges. Shared/nil budgets and closed owners are tested.

This hook lets concrete key-owning maps swap table/pool ownership without
inventing key copies or destructors. Map table charges are prepared separately
before this atomic hook. Its prospective commit passes full native tests,
checked races and vet; all five transfer tests execute under js/wasm, and WASI
compilation passes. Experimental dependent packets are excluded from that
isolated commit tree until they independently reach GREEN.
