# Actual raw memory allocation paths

This packet implements the real allocate/reallocate/deallocate bodies and
size/count/peak/watermark configuration/getters from src/util/memory_manager.cpp,
using the committed utilMemoryAccounting transitions. It owns actual byte
backings with stored SIZE_T prefixes. It is not the complete process runtime:
generated initialize/finalize, participating typed-owner allocation hooks,
multiple shared lanes, process exit/stdout/TRACE and full solver integration
remain OPEN. Budget.Used is physical Go bookkeeping, never the source getter.

newUtilMemoryRuntime creates an explicitly charged, confined cold owner. It
borrows its Budget and contains one real accounting lane, a scalar origin, and
a charged directory of physical allocations. Public block tokens contain only
origin/slot/generation. Each allocation has its actual data array, including
four/eight stored extent bytes, and a charged initialized-byte bitset. Slice
headers belong to the cold allocation directory. No host map, generic, unsafe,
boxed hot record or per-payload pointer object replaces this storage.

Allocate adds the source SIZE_T prefix with selected unsigned32/64 arithmetic.
The non-TLS profile changes counters/checks limits before physical storage.
TLS allocates real data and writes its header before counters/synchronization.
A TLS limit failure returns the completed owned block along with the source
error classification: the caller must Deallocate it. Source native throw would
leave that pointer unpublished; exposing its ownership is an explicit Go
cleanup adaptation, preserving immediate physical/counter evidence.

Reallocate reads the old stored extent, updates counters before physical work
in both profiles, allocates new actual backing, copies the surviving bytes and
initialized bits, then frees the old physical backing and writes the new header.
The actual Go backend always relocates, a permitted realloc outcome. Successful
relocation advances the checked pointer generation. Predictable physical
refusal preserves the old block while retaining counter/OOM prefixes. Exhausted
checked generations refuse before source actions, an explicit token-domain
boundary. Old+new data and bitset peaks are charged without inventing source
allocation-count events for cold metadata or checker bits.

Deallocate reads the actual stored extent, subtracts before free, then performs
TLS synchronization only after free. It never throws memory/count limits while
deallocating. Header lifetime guards run at the actual read. Scalar metadata
State/Stats are diagnostic snapshots; AllocationSize uses the source signed
clamp and does not flush local counters. MaximumUsed/AllocationCount and strict
AboveWatermark use the actual source transitions. Setters select signed
conversion for LP64 and checked valid size_t inputs for the32-bit profile.

Unsigned32 shrinking realloc deltas can add nearly2^32 to signed64 accounting;
this is preserved. TLS synchronizes strictly above100000 or below-100000, resets
only local size, and repeatedly contributes cumulative local count. Memory
limit failure precedes count limit; OOM sets its flag, count failure does not.
Errors use actual Z3Error classifications101/113. Native exit/diagnostic policy
is still an OPEN transport boundary, not silently executed by a library test.

Wrapped too-short extents still allocate real bounded physical storage before
refusing at the unsafe header write. Allocate returns its owned partial token;
reallocate publishes the new owned binding after freeing the old backing.
Source continuation is undefined. CleanupPartial explicitly releases only this
bad-header backing without a fabricated counter subtraction. Ordinary
Deallocate refuses because it cannot read the source header. Uninitialized raw
payload reads are diagnosed; Go's zeroed backing is not advertised as a source
malloc value. Shrink/regrow retains only the surviving initialized prefix.

Close refuses live external block owners before cleanup, then releases directory
and cold header exactly once. It does not reset source counter prefixes or
close borrowed services. Nil/shallow/foreign/stale/closed owners have checked
boundaries. One owner/lane and its Budget require caller confinement; this is
not yet runtime-wide concurrent source accounting.

Tests FIRST reached missing-API RED17542, then missing-getter RED. Generation
refusal regression52202 exposed a physical/counter publication before token
refusal and was fixed after observed RED. Wrapped-header API RED and behavioral
RED61820 exposed missing actual allocation prefix; header guards now occur after
physical publication. The first fixture incorrectly attempted a watermark
larger than32-bit size_t; it was corrected to a separate64-bit owner without
changing the source setter contract. Working snapshot memory-runtime-red-2c89d3f
was reused for development; immutable final gates use a separately frozen tree.

Eleven named groups cover both physical/counter profiles and size widths,
stored header/data realloc preservation, threshold equality and cumulative
count, real limit/allocator refusal roots, old-block preservation, generation
retirement, byte lifetime, wrapped partial cleanup and cold owner domains.
The independent1,200-action model checks each physical live extent and complete
retained budget graph before cleanup, including signed/local/global counters.
Twenty-four shrink/regrow cases cross bitmap/header boundaries. These are
independent source action/ownership fixtures, not a claimed Windows API OOM
campaign: src/test/memory.cpp is empty outside Windows. Its real Windows
context/config/BV/error-handler integration remains OPEN.

Final selected race/actual JS/vet/WASI gates are running on frozen snapshot
.cache/review/memory-runtime-final-53047b9. Exact two Go hashes and eleven names
are in .cache/memory-runtime-final-hashes.json. Complete combined integration
and final source/no-generics evidence will be appended after terminal results.

Reference SHA256:

- src/util/memory_manager.cpp — 3b884edd5463aa7ddfe68d6e0badc0695eff2b6b6fd90736850dd740f43e29d6
- src/util/memory_manager.h — 78b004da89d38686fdf0ad34807ad77f844e9d7ac275834ef312adc2c7bc1839
- src/test/memory.cpp — 39c8d989b1dc67fca95d08a1559e27a28faf82c05d0ea63620ed21c0ec7f046b

## Final coordinator integration

Frozen snapshot `.cache/review/memory-runtime-integration-4745701` adds only
the two verified Go files to committed complete catalog/lazy-list4745701.
Full native30618 and checked-debug84600 passed (mpnum21.514s/22.096s).
Selected checked-debug race11242 passed all eleven exact RUN/PASS groups
(1.367s); actual checked-debug JS87197 passed all eleven (0.240s). Full vet8160
and WASI build61420 passed on the earlier53047b9 snapshot with identical raw
implementation/tests and dependencies; the subsequently added lazy-list had
its own full gates. Source/no-generics audit90371 passed with
33ported/142partial/1927unported. No raw runtime methods/global unit are
advertised as complete beyond the explicitly selected allocation paths.

Standalone storage benchmark artifact
`.cache/review/memory-runtime-performance-4745701/memory_runtime_perf_test.go`
ran the exact frozen implementation: 73-byte Allocate/Deallocate cycles on
Linux amd64, 200ms target, measured160.0ns/op,112B/op,2Goallocs/op. The two
allocations are actual data and initialized-byte tracking. This records the
translated storage cost, not a C++ speed comparison. Logical owned extents
charge81 data bytes+16 tracker bytes; Go heap size-class rounding is outside
both stored source extent and requested-backing budgets.

Exact frozen Go hashes:

- util_memory_runtime.go — 830ec80548d240ee91bdb28d1feedbcc223a2c96cd53b06f5f42ec2112d1aba4
- util_memory_runtime_test.go — bb0f9cf39a8ca70e09ca2036b7ff06b97764c24f7f2697c01757bb74864d2aa8

Exact eleven named groups:

- TestMemoryRuntimeActualBlocksHeadersAndReallocate
- TestMemoryRuntimeTLSStrictThresholdAndRepeatedCounts
- TestMemoryRuntimeLimitBeforePhysicalAndTLSOwnedFailure
- TestMemoryRuntimeFailedReallocateKeepsActualOldBlock
- TestMemoryRuntimeSourceUnsignedDeltaWatermarkAndGuards
- TestMemoryRuntimeReallocateGenerationRefusalPrefix
- TestMemoryRuntimeActualGettersAndUninitializedBytes
- TestMemoryRuntimeIndependentPhysicalAndCounterModel
- TestMemoryRuntimeImmediateFaultGraphAndOwnerDomains
- TestMemoryRuntimeWrappedHeaderActualPhysicalPrefix
- TestMemoryRuntimeShrinkRegrowByteLifetime
