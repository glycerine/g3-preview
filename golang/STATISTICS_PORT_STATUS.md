# Statistics container, rendering and real resource-limit collector

Implemented the complete declared statistics container/rendering surface and
`get_rlimit_statistics`, with explicit Go lifetime/error adapters. Whole source
`statistics.h/.cpp` units remain **PARTIAL**: `get_memory_statistics` still needs
the real process-memory service. Solver statistics producers and their collector
call chains remain unported; no fake memory or solver counter was introduced.

Owned packet: `util_statistics.go`, `util_statistics_render.go`,
`util_statistics_test.go`, and this status file. Existing stream/buffer/region,
resource-limit, hash, central metadata and Git files were not edited. All writes,
compiler caches, temporary files and wasm outputs stayed under `golang/`.
Production/test/storage types are concrete; no Go generics, unsafe, production
math/big, hot interface boxing, per-record callbacks or hidden key strings.

## Complete method/source map

| Source | Implemented operation | Preserved behavior |
| --- | --- | --- |
| statistics.h:26-30 implicit default construction | NewStatistics | independent initially empty uint/double vectors |
| statistics.h members, vector.h:99-116,160-174 | Clone | independent backing at exact source capacities, borrowed keys and binding identities |
| vector.h:192-202; svector:531-543 | Assign | uint member then double member; destroys old member before exact-capacity copy; self-assignment no-op |
| vector.h:169-171; svector:534 | MoveClone | steals vector backing, leaving source vectors empty; separate actual graph budget transfer |
| svector:538-542 const-copy assignment only | MoveAssign | copies, leaves source unchanged: svector declares no move-assignment overload |
| statistics.cpp:26-29 | UpdateUint | append nonzero uint32 contribution; never dereference key or aggregate at update |
| statistics.cpp:31-34 | UpdateDouble | append nonzero IEEE64 contribution; ±0 excluded, NaN/Inf accepted |
| statistics.cpp:36-39 | Copy | append uint contributions then double contributions, not assignment |
| statistics.cpp:41-44 | Reset | clear lengths only, retain both capacities and stale backing records |
| statistics.cpp:46-58 | private maps | current contribution + old aggregate in vector order; uint32 wrap; no zero pruning |
| statistics.cpp:60-66 | scalar key buffer | uint-map physical USED buckets then double-map buckets, no cross-domain deduplication |
| statistics.cpp:68-81 | SMT2 key rendering | colon first, skip one leading colon, replace nonsimple bytes with hyphen |
| statistics.cpp:83-85,90-100 | comparison/maximum width | strcmp original CString bytes; maximum strips one leading colon |
| statistics.cpp:102-143 | DisplaySMT2Into/DisplaySMT2 | full maps/sort/lookup/padding/values, brackets and final LF |
| statistics.cpp:145-171 | DisplayInto/Display | colon stripping, colon suffix, stripped width, values and LF |
| statistics.cpp:173-201 | DisplayInternalInto/DisplayInternal | both maps first; uint physical buckets then doubles; finite empty-key output; nonempty source loop typed guard |
| statistics.cpp:203-205 | Size | contribution counts added in unsigned32, not distinct keys |
| statistics.cpp:207-209 | IsUint | idx < uint-vector length, including false for out-of-range indices |
| statistics.cpp:211-216 | Key | raw scalar borrowed key; uint vector first, then double vector |
| statistics.cpp:218-228 | UintValue/DoubleValue | source index/domain guards translated to ErrBounds |
| statistics.cpp:230-237 | collector uint64 branch | <=UINT_MAX uint32, otherwise float64 conversion |
| statistics.cpp:249-251 | GetRlimitStatistics | actual utilReslimit.count(), including UINT_MAX crossover |
| statistics.cpp:239-247 | OPEN get_memory_statistics | actual max/used/allocation-count getters absent; no BudgetUsed/zero surrogate |

Source vector growth is 0→2, then (3*old+1)/2; eight contributions retain capacity
eight. Clone copies capacity even after Reset leaves size zero. Clone/MoveClone
have an internal retained-owner constructor: copying scalar char-pointer identity
never reads key bytes, even after a primary or secondary region has closed.
Public NewStatistics and BindKeyRegion require live real regions. Closed/stale
keys can still be copied/queried raw; actual rendering later returns the region
lifetime error. No cold mutex is copied as a Go value.

## Borrowed keys, maps and exact rendering

StatisticsKey has private scalar origin/ref/offset fields and no pointers. Its
creation validates allocation/offset, not an invented immutable spelling. Plain
updates accept null, unknown foreign and stale scalar tokens: source update does
not read char*. Zero contributions skip all key work. BindKeyRegion explicitly
borrows additional real byte owners through a sorted cold identity directory.
Copy/Assign/Clone adopt existing primary/secondary bindings without reading text;
unknown raw tokens remain unknown. Regions are never closed/freed by Statistics.
Rendering scans first NUL and reads real borrowed bytes; writes to a spelling
before rendering remain visible. Owner IDs prevent numeric-ref collisions from
silently adopting a foreign region. Unterminated/out-of-bounds Cstrings return
typed errors, not fabricated text.

Private concrete temporary maps translate map.h/str_hashtable.h/hashtable.h:
seed17 string_hash with existing signed-tail/little-endian hash adapter, cached
hash, strcmp equality, capacity8 eager allocation, forward/wrapped linear probes,
source uint32 strict >75% grow-before-insert (including overwrite), doubling and
old physical USED order rehash. Equal-key replacement retains the incoming key
identity. These maps do not erase: no claim of a new complete generic hash-table
port is made. Uint/double records share a concrete pointer-free entry layout;
charges reflect actual Go stride, including all physically present fields.

Keys buffer embeds16 scalar refs and doubles overflow, with full actual owner
and old/new backing charges. Source std::sort is an external library boundary:
a concrete allocation-free in-place heap sort preserves observable ascending
original-CString order; equal spelling identity order remains unspecified.

Both-domain spelling appears twice and is looked up in uint map first on BOTH
lines, so its uint aggregate prints twice. A shadowed double does not set fixed
or precision2. Other double lines persistently set fixed and precision2 on the
borrowed NumericTextStream. Uint stream base/show flags, width/align/fill and
float flags all remain actual stream state. Normal padding uses stripped key
length; SMT2 padding uses original length including a leading colon. Sanitization
does not change sort order or width calculation. Empty SMT2 output is ()+LF;
empty ordinary/internal output is empty. No temporary text snapshots or hidden
unbudgeted key arrays are allocated. Whole-CString insertion consumes width once;
region bytes fill the stream's charged reusable scratch before insertion.
Convenience output functions produce one final caller-owned string.

Double addition keeps source contribution+old order and host IEEE64 arithmetic.
Arithmetic NaN sign/payload propagation is target-dependent: the tested invalid
infinity sum renders -nan on native AMD64 and nan on wasm. Tests require NaN
classification for that arithmetic path, not universal NaN bits. Explicit signed
NaN contributions retain their raw bits and render exact nan/-nan; ±Inf and
signed-zero exclusion are exact. This arithmetic target adapter is separate from
the NumericTextStream selected C-locale nearest/libstdc++ precision ABI profile.
Other locales, stream implementations and process fenv remain external OPEN.

## Source internal nontermination adapter

statistics.cpp:178-184 while(*key) NEVER increments key. Nonempty stripped keys
repeat their FIRST byte forever; lowercase inserts the INT ('A'+...), not a char.
For example first 'a' repeatedly inserts numeric65, not finite "A" or "65".
No silent key++ correction was made. DisplayInternalInto constructs/aggregates
both actual maps, visits source bucket order, and returns
ErrStatisticsNonterminating exactly at a nonempty loop entry. Prior empty-key
entries' space/value/LF output remains visible. Empty and single-colon spellings
are finite; doubles retain caller float mode/precision instead of fixed2.
No optional unbounded continuation or artificial iteration cap is exposed.
This is an explicit finite Go adapter of a source defect, not an assertion that
the original source nonempty internal display terminates.

## Physical budgets, prefixes and ownership

Every actual heap owner header is reflect-sized/reserved before allocation:
Statistics; both temporary map owners; key buffer including inline16. Every actual
contribution/map/overflow/binding backing is charged at actual capacity*stride.
Borrowed Budget, region owners, stream owners, stack scalar arrays and caller
returned text are not double charged. Array byte-size/host/source-count bounds
are checked before Reserve/make. The binding size helper has finite simulated
32bit byte-bound fixtures without huge allocation. Growing arrays overlap full
new and old storage before publication/refund. Reset retains charges. Close
releases double then uint, bindings and own header; no key destructors.

Updates/append Copy preserve already completed vector publications on failure.
Assignment destroys uint before copying it, then destroys/copies double. If the
second allocation fails, newly copied uint remains and double is coherently
empty. Clearing the failed member avoids source fatal-OOM dangling-pointer state;
this recoverable Go failure adapter does not make assignment transactional.
Cold binding growth/adoption can fail before a retained record; completed binding
prefixes remain charged/usable. Constructor failures release every partially
owned backing/header. MoveClone reserves destination graph charges first when
budgets differ; failure refunds the new owner and leaves source untouched.
No Budget.FailAfter attempt counter rollback or special transaction bypass.
Nonempty self append Copy is explicitly rejected before actions because source
other.size() grows forever; empty self Copy and self assignment remain no-ops.

Render temporaries clean up in source reverse lifetime order on every error.
Caller output prefix and its owned storage survive stream failures. Shared
external Budget remains owner-confined: these cold mutexes do not make unrelated
owners sharing it globally thread-safe. Cross-statistics operations use one cold
transfer mutex before both owners, without pointer-order tricks or hot callbacks.

GetRlimitStatistics reads actual private utilReslimit owner count. Nonzero calls
explicitly allocate/write "rlimit count"+NUL in the caller's borrowed primary
BytesRegion, then append at the UINT_MAX domain threshold. No hidden literal
cache, global mutable state, fake solver counter or BudgetUsed approximation.
Each call uses a fresh scalar allocation identity instead of C's static literal
address, a documented key-domain adapter; equal text still aggregates identically.
A failed contribution append retains this completed caller-owned key allocation.
Stats.Close never frees it; region Reset/Close is the actual key lifetime boundary.
Zero count skips allocation/update, matching the source zero contribution guard.

## TDD, source assertions and independent evidence

Tests preceded implementation: isolated committed module+new test file failed
with undefined Statistics APIs. Review regressions then recorded actual RED:
closed-secondary raw Copy returned ErrStatisticsKey; closed-primary Clone returned
ErrStatisticsKey. Identity-only internal transfer/constructor fixed both. The
checked-backing helper first recorded missing-helper RED before integration.
Earlier arithmetic-NaN JS failure and vet's test mutex-copy finding were corrected;
no initial failed gates are reported as PASS.

Source has five actual SASSERT sites: statistics.cpp:69 nonnull rendering key;
219/220 uint index/domain;225/226 double index/domain. All five semantic guards
have finite Go fixtures. There is no standalone test/statistics.cpp or registered
TST(statistics); no statistics ENSURE count or source fixed printed fixture was
invented. Registered algebraic/heap_trie/cnf_backbones consumers print statistics
but require genuine unrelated mathematics/solver producers; their unrelated
assertions are not counted as statistics assertions.

Nineteen final groups cover raw/null/foreign/stale/closed domains; independent
500-step vector model and1000-step exact modulo aggregate; high-byte seed17 oracle;
map capacities/probes/overwrite growth/incoming key identity; both-domain shadow;
all three render routes and persistent flags; internal prior-empty prefix;
±Inf/signedNaN/ordered finite and arithmeticNaN; embedded NUL/alias mutation;
copy/assign/move exact capacities and cross-budget/cross-region binding;
22 fault prefixes through maps/growth/keys scratch cleanup; constructor/member/
append/move failures with immediate Used roots before teardown; full old/new peaks;
real resource-limit UINTMAX crossover/literal prefix; pointer-free record audits;
warmed update/getter/hash/reset zero Go allocations; host32 bounds; invalid cold
self/nil/closed guards. Source tests remain hermetic and never require C++ or the
reference source tree.

Final frozen gates in `.cache/release/statistics-packet` (committed HEAD archive
at packet start, overlaid ONLY the three owned Go files, excluding live drafts):

- Full native ./... -count=1 PASS, session50197; root package1.959s/mpnum10.438s.
- Full g3checked+g3debug ./... -count=1 PASS,68840; root1.901s/mpnum10.711s.
- Targeted checked/debug race PASS,81381, all19 explicit RUN/PASS,1.063s.
- Actual checked/debug JS Node runtime PASS,93514, all19 explicit RUN/PASS.
- Full checked/debug vet PASS; WASI checked/debug module cross-build PASS.
- Real source/no-generics audit PASS,11732:2102 units,677 APIs;
  33 ported,60 partial,2009 unported. This is the snapshot's existing central
  metadata accounting, not a claim that this unintegrated packet changed it.
- JS wrapper used minimal env with TMPDIR below golang; all Go caches/GOTMPDIR/
  TMPDIR and archive/wasm outputs stayed below golang.

Frozen SHA256:

| File | SHA256 |
| --- | --- |
| util_statistics.go | a2ba50adec47660c04e89f023260194a4c92c64a077b68ac6a540d4388d7bb21 |
| util_statistics_render.go | 0a502180d70d04e219bffe0e1acdc6c6675fb2c99d49d99880a024cd13046282 |
| util_statistics_test.go | ae2cc59da4c327068778ddefcf84b43a972c379a7cc09818d55f938577a78e2b |

Audited active source SHA256:

| Source | SHA256 |
| --- | --- |
| src/util/statistics.h | c6793c4216c5c54787bd3c22a6a5479030bc08ff2519d961c8635ef15b1712f2 |
| src/util/statistics.cpp | ee1b62f6d3daaa8e3b1b21731620f567f23bc084e32181b141aa3951d44bbb16 |
| src/util/vector.h | aafafc0fe6ca0afd7bafe0c781342da259f9c34c0b94b16732c415cf2733066e |
| src/util/hashtable.h | bdc562a7db2c0d914dd896577c43e0648f501538873460d2e1a6c2d0932f64f2 |
| src/util/map.h | 62db3b3819aa71f67bababdb3d5583d14fecebbde6f2d22866c28a86215aa850 |
| src/util/buffer.h | f5c876511a1fd5674ba5f9fa462d9206ffaad979ee90cc30e7d1b5fc9c4e07e1 |
| src/util/hash.cpp | 05f32077835d6def70963af8ef79f5344b233cff55ca2f12ce38dc2dfde89d3a |
| src/util/str_hashtable.h | e6bbfd47cff10b6def5cc1ab1ea5a4d3a47da70e39a9d51de2f063b59dca22a5 |

## Parent independent frozen integration

Parent verified identical three Go hashes in a separate committed HEAD3a9be91+
ONLY three-owned-Go-file snapshot `.cache/release/statistics-integrated`:
full native57658 PASS (mpnum10.616s), full checked/debug44202 PASS
(mpnum10.903s), exact19 named checked/debug race4501 RUN/PASS1.056s,
exact19 actual checked/debug JS97845 RUN/PASS0.200s, full vet PASS and
WASI17604 terminal PASS. Parent updated exactly the two statistics source rows
to PARTIAL, retaining process-memory and genuine solver producers OPEN.
Parent real source/no-generics audit82871 terminal PASS:33 ported,65 partial,
2004 unported. Those central metadata edits were parent-owned; this agent did
not edit manifests/progress/deviations or Git. No additional production/test
changes followed the final freeze. Status documentation is now ready for parent
source audit and commit.
