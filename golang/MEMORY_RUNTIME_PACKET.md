# Global memory runtime: source contract and next TDD packet

Coordinator-owned prerequisite for the complete SAT/SMT runtime. This document
records inspected source bodies. The scalar transition component is implemented;
see MEMORY_ACCOUNTING_PORT_STATUS.md. Full memory manager/global runtime
integration remains open. The proof builder prerequisite is committed at a18b6ce. Existing
arena.Budget is owner-local predictable storage accounting and cannot replace
memory::get_allocation_size. Go runtime.MemStats and process RSS cannot replace
that getter either.

## Complete source body scope

memory_manager.h/cpp declare and implement initialization/finalization,
is_out_of_memory, exit_when_out_of_memory, set_high_watermark,
above_high_watermark, set_max_size, set_max_alloc_count, get_allocation_size,
get_max_used_memory, get_allocation_count, both display methods, allocate,
reallocate and deallocate. Include debug allocation/deallocation wrappers,
source out_of_memory_error and both limit failure transports. Concrete typed
allocation/destruction adapters replace template wrappers; no Go generics.
Generated mem_initialize/mem_finalize and PROFILE_MEMORY/TRACE dependencies
remain separately required, rather than silently replaced by empty callbacks.

## Ordered source behavior to preserve

- initialize updates the limit unless max_size equals UINT_MAX. Only the first
  initializer clears OOM and runs mem_initialize; concurrent initialization
  waits until full initialization. finalize runs actual mem_finalize while
  finalizing=true, then clears initialized; it does not reset every counter or
  fully_initialized. Do not invent a zero-counter reset.
- get_allocation_size returns max(global signed allocation bytes, 0).
  The getter does not flush thread-local counters. Maximum used memory records
  the global peak; allocation count is cumulative and not a count of live blocks.
  High watermark is disabled at zero and otherwise uses strict watermark < size.
- Every raw allocation includes its actual size_t prefix. Deallocation reads
  the original stored extent; shortening a logical consumer does not shrink it.
  Raw byte payloads and typed backing storage need real ownership and physical
  budget charges; no fake counter-only successful allocations.
- The thread-local branch synchronizes only when local bytes > 100000 after
  allocation/reallocation, or local bytes < -100000 after deallocation.
  Synchronization adds BOTH local size and local count, updates peak, checks
  limits, then resets ONLY local size. Local allocation count remains cumulative
  and is added again at later synchronizations. Preserve this source behavior.
- Thread-local allocate calls malloc and writes its size header BEFORE local
  counter changes and synchronization. A synchronization limit failure thus
  follows a real allocation prefix. Go must explicitly expose/own that prefix.
- Thread-local realloc updates local size delta and count BEFORE synchronization
  and BEFORE actual realloc. A limit failure leaves the old physical allocation
  intact while the counter prefix is published. A later allocator failure also
  preserves the prior counter mutation. Successful realloc updates the header.
- Non-thread-local allocate/realloc update GLOBAL size/count/peak and check
  memory then allocation-count limits BEFORE the physical allocator operation.
  Limit or allocator refusal must not roll back those source counter prefixes.
  Non-thread-local deallocate subtracts the stored extent before free.
- Synchronization while deallocating never throws memory/count-limit failures.
  Allocating checks memory before count. OOM transport sets the global OOM flag;
  allocation-count transport does not silently set it. Exit policies and output
  channels are distinct source contracts.
- display_i_max_usage ignores its supplied stream and writes stdout. Debug
  deallocate performs actual deallocation BEFORE conditional TRACE, including
  the finalizing guard. Debug allocate traces only after successful allocation.

CMake Linux x86_64 selects _USE_THREAD_LOCAL; scripts/mk_util.py selects it for
64-bit Linux. Both conditional bodies remain in scope for Go native/wasm builds;
explicit runtime lanes replace implicit OS-thread-local ownership. Host word
width, actual Go backing sizes and extra owner metadata must be documented
separately from source ABI sizes. Invalid pointers, signed overflow and host
unrecoverable OOM need typed/explicit boundaries without changing finite paths.

## Actual consumers and integration requirement

SAT solver.h checkpoint checks global allocation bytes every tenth call against
its configured maximum; probing.cpp checks actual accounting to clear/stop its
cache. Solver diagnostics and statistics.cpp also read this service. Runtime
ownership must therefore connect real allocations across participating owners
and preserve synchronization/limit ordering. A getter covering only one budget,
only a test raw-byte allocator, or only the solver clause directory is not full
integration. Existing typed owners still need an audited allocation-hook plan;
aggregate budget reservation must not be mislabeled as an exact C++ malloc call.

Generated initialization uses component-sorted header traversal, stable numeric
initializer-priority ordering, and collected finalizer order (not an invented
reverse initializer order). Full registered modules are required eventually.

## Tests before implementation

Write and observe missing-API RED for actual owned raw storage and source counters
in both selected branches: strict threshold edges, no getter flush, repeated
local-count contribution, negative global clamp, watermark equality, real
realloc data/header preservation, counter-before-allocator refusal, old-block
ownership after failed realloc, physical allocation publication after TLS limit
refusal, deallocation without throwing limits, memory-before-count selection,
OOM flag policy, and actual initialization/finalization callback ordering.
Add independent bounded allocation/counter models and immediate physical-root
checks before rendering or cleanup. Tests must distinguish source prefixes from
Go cleanup/continuation adapters. Complete native/debug/race/actual JS/vet/WASI
compile/source/no-generics gates on a frozen packet before any coherent commit.

src/test/memory.cpp is an empty function outside Windows. Its Windows API loop
requires the real config/global-params/context/BV-sort/error-handler/reset stack
and runs until OOM for limits10/20/30. Do not claim that direct allocator fixtures
execute this registered Windows test; retain that obligation separately.

## Inspected source fingerprints

- `src/util/memory_manager.h`: `78b004da89d38686fdf0ad34807ad77f844e9d7ac275834ef312adc2c7bc1839`
- `src/util/memory_manager.cpp`: `3b884edd5463aa7ddfe68d6e0badc0695eff2b6b6fd90736850dd740f43e29d6`
- `src/test/memory.cpp`: `39c8d989b1dc67fca95d08a1559e27a28faf82c05d0ea63620ed21c0ec7f046b`
- `src/sat/sat_solver.h`: `7ec8a9c01ca2757723192c3e1098891043a91c7e719695fd2c806f717233c1c4`
- `src/sat/sat_probing.cpp`: `3dcf3eecc14646c45c2e9ed55210c6e7b87e1a459f77c548344700580694910e`
- `src/util/statistics.cpp`: `ee1b62f6d3daaa8e3b1b21731620f567f23bc084e32181b141aa3951d44bbb16`
- `scripts/mk_genfile_common.py`: `86cdb958696d411d48d04b74fc56e379a6ad41eba03d1f52cf76c53813f15482`
- `scripts/mk_util.py`: `4b1921c9cdf40923f8d12b81c6319cab2f9cfbb8dd5a3abcc48deeb94b3179fd`
- `CMakeLists.txt`: `832e85a5d21ec0baecb1fa08f166037719fdf363e09fd48a69f3f7ab4efca6bd`

The counter component now exposes explicit32/64 size_t promotion helpers.
Physical adapter input deltas must use source unsigned arithmetic at that width.
On ILP32, shrink subtraction wraps as uint32 before LL64 promotion; on LP64,
counter extent arithmetic is unsigned64 before the selected signed conversion.
Generated callbacks, physical allocator hooks and failure transports remain open.

Runtime clock prerequisite: Linux stopwatch uses CLOCK_PROCESS_CPUTIME_ID,
not elapsed wall time; get_seconds while running performs stop/start sampling.
Windows/Apple/BSD branches and JS/WASI clock-domain adapters need their actual
source contracts. Do not substitute time.Now/time.Since for process CPU time.
