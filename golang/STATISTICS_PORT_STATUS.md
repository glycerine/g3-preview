# Statistics source audit and dependency proposal

Read-only audit/proposal: no statistics implementation or tests are added yet.
Only this assigned status file was written; no central metadata or Git changes.
Source statistics.h/.cpp Microsoft2011 notices must be retained on translation.
All future production/tests/tools must be concrete, pointer-free in hot storage,
physically budgeted, and free of Go generics/unsafe/boxed hot records. Completing
this utility is not implementing solver statistics producers or a solver runtime.

## Full declared and helper method map

| Source | Proposed concrete operation | Source algorithm |
| --- | --- | --- |
| statistics.h:26-30 members/default constructor | NewStatistics(StatisticsConfig) | two independently empty contribution vectors, unsigned32 and float64 |
| statistics.cpp:26-29 update unsigned | UpdateUint(StatisticsKey,uint32) error | append key/value only if inc!=0; never aggregate at update |
| statistics.cpp:31-34 update double | UpdateDouble(StatisticsKey,float64) error | append only if inc!=0.0; both signed zeros excluded, NaN included |
| statistics.cpp:36-39 copy | Copy(*Statistics) error | APPEND source uint contributions, then APPEND source double contributions |
| statistics.cpp:41-44 reset | Reset() error | reset both lengths, retain their storage/capacity; no key destruction |
| statistics.cpp:46-58 mk_map | private uint/double aggregation maps | iterate each input in contribution order; find by CString bytes; replace with currentContribution+oldValue, otherwise insert |
| statistics.cpp:60-66 get_keys | private concrete scalar key buffer | append physical-bucket USED keys from uint map then double map; duplicate spellings across maps retained |
| statistics.cpp:68-81 display_smt2_key | private byte renderer | assert nonnull; emit colon; skip one original leading colon; replace every nonsimple byte with hyphen |
| statistics.cpp:83-85 str_lt | private CString comparator | strcmp byte order, not Unicode/locale collation |
| statistics.cpp:90-100 get_max_len | private width helper | longest first-NUL key length AFTER stripping at most one initial colon |
| statistics.cpp:102-143 display_smt2 | DisplaySMT2Into/DisplaySMT2 | build/sort local keys, opening paren, lines separated LF+space, key/padding/value, closing paren+LF |
| statistics.cpp:145-171 display | DisplayInto/Display | same maps/keys/sort/max; strip one colon in key, emit barekey+colon/padding/space/value/LF |
| statistics.cpp:173-191 template display_internal | private uint/double bucket renderer | unsorted physical USED buckets; strip one colon; nonempty key loops forever on its FIRST byte (no increment); empty key emits space/value/LF |
| statistics.cpp:193-201 display_internal | DisplayInternalInto/DisplayInternal | build both maps, emit entire uint map then entire double map |
| statistics.cpp:203-205 size | Size() uint32 | sum contribution lengths, NOT unique/aggregated key count |
| statistics.cpp:207-209 is_uint | IsUint(uint32) bool | idx<uint-vector length only; no idx<size assertion |
| statistics.cpp:211-216 get_key | Key(uint32)(StatisticsKey,error) | combined raw vectors, uint entries first then doubles; scalar borrow replaces raw key pointer |
| statistics.cpp:218-222 get_uint_value | UintValue(uint32)(uint32,error) | both index and uint-domain implementation preconditions |
| statistics.cpp:224-228 get_double_value | DoubleValue(uint32)(float64,error) | both index and double-domain implementation preconditions |
| statistics.cpp:230-237 get_uint64_stats | private updateUint64 helper | <=UINT_MAX stores unsigned, greater stores float64 conversion |
| statistics.cpp:239-247 get_memory_statistics | real memory-service collector dependency | read peak/used; unsigned64 multiply100 then integer divide1048576; update max memory, memory doubles then num allocs |
| statistics.cpp:249-251 get_rlimit_statistics | real resource-limit collector dependency | read actual reslimit.count and updateUint64(rlimit count) |

All entries are proposals/unported. Rendering APIs depend on explicit numeric
stream-state/float support below; do not implement fake successful float output.
Close and storage Stats are Go ownership/inspection adapters. Source implicit
copy construction/assignment and move construction/assignment inherit real
vector.h semantics, separate from its explicit append-copy method: deep vector
backing, same borrowed keys, copy exact source capacity; assignment uint member
then double member, vector assignment destroys old before copy. These must be
mapped explicitly if included in the complete owning wrapper packet. Do not
rename append-copy to Assign or export an unsafe Go shallow owner copy.

## Exact contribution, map and rendering quirks

Unsigned inputs are unsigned32, not uint64 counters. Aggregation addition wraps
modulo2^32 as defined source arithmetic. Double additions follow insertion order
and the source expression contribution+old, not a reassociated sum. All IEEE64
finite/NaN/infinite values are valid update inputs except positive/negative zero
which do not append. A later sum may be zero, negative zero, infinity or NaN and
still must print; there is no zero-pruning at aggregation. Getters expose original
contributions. Duplicate input keys remain distinct in raw vectors.

Each display constructs unsigned map FIRST, double map SECOND, aggregates uint
vector THEN double vector, and creates local pointer-key buffer afterwards.
Maps are separate map<char const*,unsigned/double,str_hash_proc,str_eq_proc>
(default_map_entry, core_hashtable), not scoped SymbolInt32Table or chained map.
Use source string_hash(first-NUL bytes,length,17), strcmp equality and cached
entry hash. Default capacity8, strict (size+deleted)*4>capacity*3 growth before
insert even for an overwrite, forward/wrapped linear probing, USED bucket
traversal and source doubling rehash order must be preserved. map replacement
sets both incoming key and value; if equal-byte keys came from different
allocations, the last contribution's borrowed key identity replaces earlier
identity. General null keys are invalid for hashing/display; key!=0 is asserted
by display_smt2_key. Numeric Symbols are not a source CString key.

Get_keys concatenates both maps without cross-domain deduplication. A spelling
present in both appears twice after sorting; display/display_smt2 look in uint
map first for EACH occurrence, so they print the uint aggregate twice, never the
double aggregate for that spelling. DisplayInternal reaches each map's own value only for empty post-colon
keys; nonempty keys never reach the value insertion due to the loop defect. Equal-key std::sort stability is unspecified; equal spellings have
identical normal/SMT2 output, so no invented stable key-identity promise.

Normal/SMT2 sort by original key before colon removal or hyphen sanitization.
Distinct keys can produce equal rendered keys and remain separate lines. Max
width strips one leading colon. Normal display also strips before its len/pad.
SMT2 padding uses ORIGINAL strlen(k), including that leading colon; preserve
this one-space discrepancy rather than normalize alignment. Empty normal/internal
output is empty; empty SMT2 output is exactly () followed by LF.

Critical internal-output quirk: out << ('A'+(*key-'a')) inserts an INT, not a char.
With ordinary decimal stream state, lowercase first byte a emits65 repeatedly,
b66 repeatedly, z90 repeatedly. The while(*key) body NEVER increments key.
Therefore a b does NOT render finite65_66: it emits65 forever. Nonlowercase
first byte repeats unchanged, space repeats underscore. Strip one leading colon
only; empty post-colon key is finite and emits space/value/LF. Do not silently
insert key++ while translating. The integer code insertion inherits caller
base/show/width flags; decimal-code prefix is only the ordinary initial profile.
A future explicit typed nontermination guard or finite-budget output-prefix
adapter must be separately approved/tested. This discovered source defect blocks
claiming an ordinary finite DisplayInternal for nonempty keys.

## Key storage and concrete representation proposal

Source m_stats/m_d_stats store BORROWED char pointers. Update/copy/reset/destructor
never allocate, copy, free or own key text. Typical solver callers use stable
string literals, and helper name arguments remain borrowed. Copy shares those
same key pointers. Rendering's temporary keys and map entries borrow them too;
there is no source interning or key deduplication at append time.

Preferred faithful Go storage: one explicit borrowed *arena.BytesRegion key owner
in cold StatisticsConfig{Keys *arena.BytesRegion, Budget *arena.Budget}; scalar
StatisticsKey{originID uint64, ref arena.BytesRegionRef, offset uint32}. Numeric
values use concrete pointer-free uint/double records. Validate origin/ref/liveness
and scan first NUL on key access; do not cache a spelling snapshot or use
Symbol.String intermediates. This can preserve actual borrowed key aliasing,
including externally written bytes observed at later rendering. No key slice or
pointer escapes. Key owner outlives storage/operations; Close does not free keys.
Keys can occupy distinct allocations with identical spelling. Cold caller helper
can allocate literal bytes+NUL in its explicit region; that is an owned key-domain
adapter, not a fake solver producer or hidden global literal cache.

A SymbolManager-bound alternative would need plain named symbols and explicitly
restrict key spelling mutation to canonical immutable interning; it is weaker
than source raw borrowed text. Existing Symbols/SMT2 APIs are available but are
not grounds to silently substitute symbol hashing for source CString hash17.
StatisticsKey scalar origin/ref proposal awaits root choice. Foreign key-owner
copy requires an explicit approved cross-owner key policy; same owner default
copy retains borrowed refs, not deep text. Zero update should remain an exact
no-op before attempting key read/validation, matching source inc guard.

## Physical budget and lifecycle proposal

Charge actual cold owner reflect-size once, including two slice headers/scalars,
mutex/self identity/borrowed owner pointers. Do not double-charge borrowed Budget,
key region, or caller text. Actual concrete contribution backing is charged
capacity*sizeof(record); both record types audited pointer-free. Source svector
uses vector.h capacity0->2 then (3*old+1)/2, two unsigned header slots stored with
source backing. Translate control slots into actual Go header scalars rather
than inventing an8-byte backing allocation that does not physically exist.
Record stride is actual Go layout, not source pointer-size assumptions. Full
new backing overlaps old at growth; source realloc becomes explicit safe copy
adapter with actual peak charged. Reset keeps capacity and borrowed record bytes.
Close releases double vector then uint vector (reverse source member destruction),
then header, without destroying borrowed key bytes. Explicit copy appends one
record at a time uint then double, preserving source completed prefixes on
allocation failure. Nonempty self Copy loops forever in source because other.size
increases: reject this undefined/nonterminating ownership case before actions,
not snapshot self. Empty self is a valid no-op.

Rendering temporary maps require TWO charged owner headers+backing capacity8
allocated in source order even when empty. Aggregation scratch lifetime remains
local; map entries do not own key allocations. Concrete scalar key buffer has
source buffer<char*,false,16> inline16 then doubled overflow; charge actual
owner/inline once and old/new overflow peaks. Sort in-place, no uncharged key
slice/String intermediates. Budgeted output buffer appends source actions;
convenience String has one final caller-owned text. Cleanup on error must release
all local graphs without changing source contribution vectors. Scratch/output
errors may retain caller output prefixes and stream-state manipulator effects
already completed. Constructor unwind, copy second-vector failures, maps/key-sort
scratch failures and final text boundary need independent immediate accounting.
Self identity disallows shallow Go owner copies. Borrowed external Budget still
requires confinement even with a cold owner mutex.

## Float and ostream dependency gate

Available PrintfFormatter implements integers/C strings only. It deliberately
rejects float formats and is NOT a valid statistics float backend. fmt.Sprintf,
strconv.FormatFloat or mathematical rational display cannot silently replace
source C++ num_put/stream floating behavior. No float implementation is approved
by this audit.

Normal and SMT2 double branch performs out<<fixed<<setprecision(2)<<d_value.
Flags and precision persist AFTER the method; no restoration exists. If no actual
double branch executes (including double keys shadowed by uint map), float flags
and precision are unchanged. DisplayInternal uses existing caller stream state
without applying fixed or precision. Caller integer base/show flags also affect
uint counters and internal lowercase numeric-code insertion. Formatted widths
reset after formatted insertions; fill/alignment, locale decimal/grouping,
exception masks and stream badbit/custom buffers can influence actual output.
A fresh default C-locale stream has defaultfloat precision6 and decimal integers;
fixed2 must correctly handle IEEE rounding, subnormals, negative zero, inf/NaN,
large exponent values and source platform formatting distinctions. Repeated
rendering on the SAME stream is observably stateful.

Proposed prerequisite: a separately audited concrete cold numeric text/stream
profile exposing uint32/int code insertion and IEEE64 defaultfloat/fixed/scientific
modes, precision and persistent state. Explicit supported C-locale baseline and
platform output adapter require fixtures/oracles before rendering is claimed.
Caller-custom locale/stream-state behavior stays an explicit unported adapter
surface unless completely modeled. Statistics DisplayInto APIs should accept
that real concrete stream/profile state plus charged output destination; after
first double, publish fixed/precision2 before attempting its numeric output.
Do not falsely mark statistics.cpp complete with a uint-only renderer or a typed
unsupported-double exit. A finite coherent first packet could finish storage,
copy/query and independent aggregation while renderer/collectors remain clearly
partial, followed by the real float dependency and full source rendering packet.

## Resource collectors are actual dependencies

get_memory_statistics uses process memory_manager getters, not local arena Used.
Source memory_manager.cpp:162-185 reports actual tracked allocation size (negative
clamped0), peak and global allocation count. The helper's100*value is unsigned64
with defined wrap BEFORE MiB integer division; then converts quantized counts to
double/100.0. Counter<=UINT_MAX goes to uint, otherwise double loses low bits as
source conversion does. Order max memory, memory, num allocs is observable.
Do not fake these sources with a supplied zero or a single owner budget.

reslimit::count is implemented in the existing concrete internal utilReslimit;
real GetRlimitStatistics can consume that actual owner after interface approval.
Its uint64 count follows the same UINT_MAX threshold. No solver/resource collector
scaffold needed. Full process memory service remains a separate dependency.
A pure helper over explicit finite snapshot values can test arithmetic but must
be labeled adapter/oracle, not the completed global getter algorithm.

## Real consumers and tests

No standalone src/test/statistics.cpp or TST(statistics) was found. Source
statistics.cpp has five SASSERT sites: nonnull display key at69; index+domain
checks at219-220 and225-226. No direct source ENSURE asserts this container.
Test/algebraic.cpp:98-100 and113-115 print solver-produced SMT2 statistics;
main.cpp:207 registers algebraic, while its ENSUREs check algebraic mathematics,
not statistics bytes. Test/heap_trie.cpp:17-30 prints collected statistics;
main.cpp:224 registers heap_trie, without a fixed statistics output expectation.
Test/cnf_backbones.cpp:17-27 prints real solver/time output, registered as
TST_ARGV(cnf_backbones) at main.cpp:247. Do not invent deterministic output or
borrow those unrelated assertion counts. Real collector prerequisites must be
ported before source print routes can execute faithfully.

Actual consumers inspected: cmd_context.cpp:1860-1876 combines actual time,
memory, resource limit and solver/check-result/optimizer producers then SMT2
renders; tactic_cmds.cpp:162 adds elapsed time; shell/dimacs_frontend.cpp:40-41;
smt_context_pp.cpp:356-401 updates counters, recursively collects quantifier/
theory/assertion producers then normal/internal display; solver_pool.cpp:154-155
and262-268 combines timers/counters. API api_stats.cpp:26-37 SMT2 rendering strips
exactly final LF, then publishes context-owned external text; getters validate
bounds/domain and API RC owns the statistics wrapper (api_stats.h:26-32).
These API/context/solver algorithms remain unported by the utility proposal.
Actual producers include SLS floats per second (sls_engine.cpp:84-93), algebraic
manager counters, Hilbert basis, bitvector simplification and many solver counters.
Zero/NaN/infinite float domains cannot be excluded merely because example counters
are positive integers.

## Proposed tests-first finite packet

Before production: independent raw contribution model, duplicate append order,
zero/negative-zero/NaN/inf policies, uint overflow and ordered double sums;
copy uint prefix followed by double failure, reset capacity retention, exact
svector growth/old-new peaks, borrowed key alias/foreign/stale/owner-close guards,
implicit copy capacities if in scope, nonempty self append guard. Queries preserve
raw contribution values and source IsUint lacks total bounds. Test all5 actual
source preconditions without inventing source ENSURE counts.

Aggregation/render tests after backend approval: hash17/forward bucket order,
grow-before-overwrite, last equal-byte key identity, same-key uint/double shadowed
duplicate lines; original key sorting, leading-colon padding discrepancy, empty
outputs, key sanitization collisions, internal no-key-increment nontermination and exact repeated FIRST-code prefix
(or approved typed guard), plus finite empty-key output. Fixed2/default
float independent finite IEEE fixtures and persistent state across repeated calls,
including shadowed double causing NO state mutation. Failure sweeps inspect
source vectors, local scratch/graph budgets, output byte prefixes and published
state immediately before teardown hides leaks. Test real rlimit count thresholds;
process memory collectors remain gated until real service exists. Full native/
checked-debug/race/actual named JS/vet/WASI/source-noGen gates after each coherent
packet, all caches/probes under golang, no fake solver interfaces.

## Source SHA256

```
c6793c4216c5c54787bd3c22a6a5479030bc08ff2519d961c8635ef15b1712f2  src/util/statistics.h
ee1b62f6d3daaa8e3b1b21731620f567f23bc084e32181b141aa3951d44bbb16  src/util/statistics.cpp
62db3b3819aa71f67bababdb3d5583d14fecebbde6f2d22866c28a86215aa850  src/util/map.h
e6bbfd47cff10b6def5cc1ab1ea5a4d3a47da70e39a9d51de2f063b59dca22a5  src/util/str_hashtable.h
aafafc0fe6ca0afd7bafe0c781342da259f9c34c0b94b16732c415cf2733066e  src/util/vector.h
```


## Coordinator review correction

Complete source review found statistics.cpp display_internal body173-191 contains no key++.
The original audit wrongly described traversing an entire key. That claim is
withdrawn and corrected above. Statistics remains unimplemented; no production
loop fix or fabricated finite source output is introduced. Future policy must
preserve nontermination explicitly, or classify a checked guard/finite-budget
prefix adapter; full statistics display completion stays gated on that review.
