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
