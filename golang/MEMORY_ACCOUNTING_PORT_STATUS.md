# Memory counter transition component

This implements the scalar component of MEMORY_RUNTIME_PACKET.md:
global/lane counter updates and synchronization, deallocation counter phases,
strict limit selection and OOM flag, nonflushing size/count/peak getters and
strict/disabled watermark. It is not a complete raw allocator or global runtime.
Physical malloc/realloc/free, size_t headers, generated lifetime callbacks,
configuration, concurrent global ownership, source exits/diagnostics/TRACE and
all participating allocation hooks remain required. The SAT checkpoint must
not substitute counter-only state for real runtime-wide reporting.

The component creates no cold owners or backing arrays. Its pointer-free fields
are intended for embedding and charging once in the real runtime owner; the
containing service must synchronize global access and confine explicit lanes.
No generic code is used.

Five initial transition groups observed missing-API RED17938 before code;
the signed-domain fixture was written before initial implementation. A seventh
phase fixture observed missing-method RED410bbe before implementing counter
subtraction before free and synchronization after free. These fixtures test
counter state only, not successful raw allocations or the Windows API campaign.
Actual adapters must call TLS-malloc accounting AFTER physical success and
TLS-realloc/non-TLS allocation accounting BEFORE their physical operations.
They must retain each source completed counter/physical ownership prefix.

Eight groups cover strict ±100000 thresholds, repeated local count contribution,
no getter flush, memory-before-count failure, limit-failure prefixes, negative
clamp, retained peak, watermark equality/zero, signed-overflow prefixes and the
physical deallocation boundary. A 300-event independent three-lane model includes
shrink-reallocation deltas. Signed synchronization/count overflow and ILP32-promoted LL64 overflow are
undefined: Go stops at the actual failing addition, preserving prior mutations.
LP64 size_t updates instead use unsigned64 modular arithmetic and the selected
two's-complement signed conversion. ILP32 realloc subtraction occurs as uint32
before LL64 promotion; a shrink can therefore add a large positive delta.
A compiled GCC source-promotion probe confirmed both distinctions; the new
size_t-width fixture observed missing-API RED before those helpers were added.

Frozen .cache/review/memory-accounting-green is HEAD92c5d0a plus only these2 Go
files, excluding evolving dependency/core and solver drafts. Final full native (root2.463s/mpnum22.781s), full checked/debug
(root2.731s/mpnum22.594s), eight checked/debug race groups (1.016s), eight actual
checked/debug JavaScript RUN/PASS events, vet and WASI compilation passed.
WASI execution and actual process-wide concurrent accounting are not claimed.
Initial source fingerprint audit passed; final metadata/no-generics audit is
recorded after its terminal result.

memory_manager.h/.cpp remain partial. Registered src/test/memory.cpp is empty
outside Windows; its Windows config/context/BV-sort/OOM campaign remains
unported. No direct counter fixture is labeled as that registered test.

Frozen Go hashes:

- util_memory_accounting.go: ac2fa02a442c038784c5843d154793a42a651fe4db4b7451dbd690f90c1ecfab
- util_memory_accounting_test.go: 3515aa3baf6126219229774e406319c43ad06676d096a116485deeffbd33a1e7

Final source/declaration/no-generics audit passed2102 units/677 APIs:
33 ported,80 partial,1989 unported. Only memory_manager.h/.cpp move from
unported to partial. Whole-method/ownership flags and API coverage remain false.
Frozen byte hashes matched before commit; original source fingerprints unchanged.
