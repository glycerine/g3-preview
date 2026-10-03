# Source ID generator prerequisite

`util_id_gen.go` translates `src/util/id_gen.h`: constructor, mk/recycle,
reset/cleanup, show_hash, get_id_range, set_next_id, and free-ID display. IDs stay
uint32 in both handle builds and are distinct from arena slots/generations.
Source UINT_MAX wraparound and duplicate recycle behavior are preserved;
this low-level generator does not invent globally unique/permanent IDs.

The owner controls the source memory-service check through one cold callback;
recycle skips enqueueing while that service reports out-of-memory. The full
process/owner resource service remains unported. Free-ID backing arrays are owned,
reset retains capacity and cleanup drops it. The generator must not be shallow
copied. Full owner budget plumbing for free-list growth remains open.

There is no dedicated source id_gen assertion suite. Finite added tests check
known allocation/recycle sequences, resource skip, debug skip over free IDs,
range, reset/cleanup, source wraparound, duplicate behavior, display and explicit
little-endian byte representation of hash input. A 3,000-operation independent
live-ID set tests uniqueness when the caller obeys source ownership. A warmed
mk/recycle loop allocates zero Go objects. Hash ordering follows the established
utilhash baseline; the empty-list hash avoids the source's invalid address-of-
element-zero expression without changing the result. Oversized hash byte length
is explicitly rejected instead of truncating to uint32.

Tests first failed with undefined constructor, then passed after translation in
release and checked builds. No test reads reference files or runs C++.
Source AST register/delete/compress call sites were read for integration context
but remain unported; this primitive does not implement an AST manager.
