# Shared memory uint32 codec prerequisite

Concrete helper adaptation, not a new C++ allocator algorithm or a completed MPZ backend. Frozen baseline45f5d4b plus ONLY memory/uint32_codec.go and memory/uint32_codec_test.go. No existing production, byte API, metadata, Git, tracked-release or MPZ/Natural/service implementation was changed by this chunk.

## Exact implemented surface

Runtime.ReadUint32/WriteUint32 use the real initialized byte backing and SIZE_T header validation. Complete four-byte bounds use offset<=extent and extent-offset>=4, avoiding offset+4 wrap. Read checks every initialized byte; write checks the entire word before publishing its four bytes. Explicit little-endian shifts are the selected internal32-digit physical adapter, with no unsafe casts or slices escaping. Unaligned byte offsets work as explicit codec positions. No allocation/reservation/count/free events are emitted.

SmallAllocator.ReadUint32/WriteUint32 preserve ordinary byte API requested-size bounds and actual owner/generation/closed guards. They resolve the one genuine raw Block+offset and use the same runtime codec. ObjectExtent returns real rounded pool-class block bytes or direct Runtime.Size, distinct from requested bounds. Future MPZ typed access must check its own word against this physical object extent, never the entire underlying chunk or the source claimed capacity. LP64 pooled request4 rounds8 and fits the actual capacity header at offset4; ILP32 pool/direct4 truly endanger that access. Existing byte Write(ref,4) still rejects request4.

The helpers own no new header, backing, interface or closure. Actual Budget graph and source Runtime.State stay unchanged across warm reads/writes/extent queries. They do not interpret Budget reservations as malloc calls.

## Tests first and meaningful coverage

Original missing API RED against6ffd717 is .cache/mpz-backend-phase-b-memory-red.log, compile exit1 undefined ReadUint32/WriteUint32. Expanded physical-extent/lifecycle fixtures compile against these real owners; four final groups:

- TestMemoryUint32CodecActualBackingAndInitializedBytes: real bytes and LE word, wholly/partially uninitialized reads, uint64 endpoint bounds, no counter events.
- TestMemoryAllocatorUint32CodecUsesIntrusivePayload: actual shared capacity bytes, genuine pooled address reuse with new generation, old token rejected, no raw event on recycle.
- TestMemoryMPZRoundedPhysicalObjectExtent: LP64/ILP32/direct profiles, real rounded extent, requested byte boundary unchanged and LP64 actual capacity header publication.
- TestMemoryUint32CodecGuardsAndZeroAlloc: unaligned backing, foreign/copied/nil/closed/freed identities, failed end-word write preserves all four bytes, armed Budget plateau and0hostallocs for actual escaping owners. Wrong successful-read values are captured by a scalar flag independently from nil error.

Initial executed fixtures exposed a test configuration mistake: NewBudget(0) refuses charges rather than expressing an unlimited budget. Tests now use bounded real budgets; no production change was made to bypass Budget. The initial three tests were compilation RED, not falsely claimed runtime RED for the already correct new fourth fixture. First codec-only native GREEN is .cache/memory-codec-first-green.log.

## Frozen hashes and isolation

- memory/uint32_codec.go: ae90b5722be60408e470d13c1b8881d8bc95a176a1455254d6bca29464afb4f6
- memory/uint32_codec_test.go: 4dfe9b824f1c57899bfb4b24f5f2da74b9f577b9c1f7b872784d8decde2a4668

Snapshot .cache/release/memory-codec-final-45f5d4b excludes all live tracked-release and MPZ/Natural missing-API draft tests. Exact49 lower-package groups are recorded in .cache/memory-codec-final-names.txt: original45 plus four codec groups. All caches/temp/output remain below golang/.cache. Final gate evidence follows after terminal completion; central source/evidence integration is parent-owned and no source unit promotion is implied.


## Terminal final gates

All results below use exactly45f5d4b+the two frozen Go files, normal compiler environment and cache/temp paths below golang. No unrelated live RED drafts are included.

| Gate | Terminal handle | Result | Log |
| --- | --- | --- | --- |
| Full native ./... | 4290 | PASS exit0 | .cache/memory-codec-final-native.log |
| Full g3checked+g3debug ./... | 76760 | PASS exit0 | .cache/memory-codec-final-debug.log |
| Complete memory g3checked+g3debug race | 37823 | exact49 RUN/PASS exit0 | .cache/memory-codec-final-race.log |
| Complete memory actual checked-debug JS | 90788 | exact49 RUN/PASS exit0 | .cache/memory-codec-final-js.log |
| Full checked-debug vet | 2249 | PASS exit0 | .cache/memory-codec-final-vet.log |
| Full checked-debug WASI build | 4919 | PASS exit0 | .cache/memory-codec-final-wasi.log |
| Frozen-tree CheckNoGenerics AST audit | terminal command | PASS exit0 | .cache/memory-codec-final-nogen.log |

RUN/PASS sets in both race and JS logs were compared exactly to all49 selected names; both frozen workspace and snapshot SHA values were rechecked unchanged. The no-generics audit was run from the workspace module against the isolated tree after an initial external-helper invocation hit Go's internal-package path restriction; the final audit is terminal PASS. Production/tests/doc are released for parent review/integration. Full actual MPZ/Natural source storage and tracked release remain separately required pending approved implementation; this helper does not close those units.


## Parent terminal integration on rational-division base

Independent snapshot `.cache/review/memory-codec-parent-70067bd` is committed70067bd plus exactly two Go files and three documentation/ledger files. Full native62526 and checked/debug98367 passed (mpnum22.650s/23.130s). Checked/debug race83294 and actual JS97687 each executed exactly49 lower-memory RUN/PASS groups matching the frozen inventory. Vet68448, full WASI build52798 and checked/debug test compilation (root98476, mpnum2746, memory immediate) passed. Integrated source/evidence/no-generics audit7354 exited0 with33ported/145partial/1924unported. Both reviewed Go hashes remained unchanged through gates. Concurrent tracked-release, factor/prime and MPZ/Natural/SAT drafts were excluded; the live worktree is not claimed green. Documentation-only final evidence was appended after verification. No whole source unit is promoted.
