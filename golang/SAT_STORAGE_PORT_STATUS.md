# Standalone SAT storage: concrete implementation status

The approved standalone storage packet is implemented in 16 new Go files, with
40 permanent named test groups. All production, tests, reference probes, build
caches and output stay under golang. Existing SAT types, heap/ID/SmallBytes
utilities, central metadata and Git remain parent-owned and untouched.
No Go generics, unsafe, production math/big, hot boxed records, solver stand-ins
or extension callbacks were added. Full propagation/search/simplification/solver
consumers and the actual sat_user_scope solver test remain OPEN.

## Selected complete scope and prerequisites

Implemented the executable standalone methods in sat_clause.h/.cpp,
sat_clause_set.h/.cpp, sat_watched.h/.cpp, sat_justification.h and sat_var_queue.h:
real clause owner and temporary clause, binary/wrapper scalars and owning vectors,
borrowed clause set, watched scalars and owning list, justification, and variable
queue borrowing a real mutable activity vector. Read all these files, sat_types.h,
heap.h, vector.h, approx_set.h, id_gen.h, small_object_allocator.h/.cpp, the existing
Go SmallBytes/IDGen/IntHeap/literal/model/approximation implementations and real
registered heap/sat_user_scope/small allocator tests and SAT consumers.

Available genuine dependencies: satLiteral/satBoolVar/satModel/lbool/satApproxSet;
arena.SmallBytes with descriptor generations/physical budgets/pooling/direct
branches; budgeted utilIDGen; source heap algorithms already translated in
IntHeap (but source action-order difference below prevents blindly wrapping it);
NumericTextStream for real ostream flags/width/base/numeric emission; exact
source vector growth/copy capacities. No new solver abstraction is necessary.

Owned files: sat_clause_storage.go/test, sat_clause_set.go/test,
sat_watched.go/test, sat_justification.go/test, sat_var_queue.go/test and two
sat_storage_debug.go/nodebug.go compile-profile files. Add concrete vector
implementations in those files, or one explicitly coordinated sat_storage_vector
file if organization warrants it. This document remains the single packet
status. Existing shared storage, SAT types, heap and ID implementations are
root-owned and will not be edited without coordination.

## Concrete storage and APIs

Use existing lowercase sat literal/model types and concrete owner-local `sat...`
names within package g3, with exported method names where useful for inspection.
No public raw backing slices or Go pointers in retained SAT records.

- satStoredClauseRef: private scalar ownerID uint64 plus storageKind/slot/generation.
  Existing arena_refs.go satClauseRef (rawRef foundation) is untouched and is NOT
  the completed clause owner locator; future consumers must adopt this real type.
  satClauseOffset is a DISTINCT scalar type with the same checked locator fields.
  Source clause ID is another uint32 scalar stored in clause data, never derived
  from slot/offset/generation. Zero refs are invalid; physical slot0 remains valid
  when generation/ownerID are nonzero. OwnerID uses a cold monotonic atomic
  namespace, never wraps/reuses; self-identity rejects owning Go shallow copies.
- satClauseAllocatorConfig: one Storage arena.SmallBytesConfig, whose
  Descriptors.Budget charges the COMPLETE owner graph; source allocation profile
  default follows the selected Linux g3debug/nondebug profile, with explicit
  pooled/direct diagnostic overrides only as recorded storage test adapters.
  Optional borrowed OutOfMemory func()bool feeds existing utilIDGen's real source
  memory::is_out_of_memory branch. Its cold function field is included in header,
  its externally owned environment is borrowed, no closure is allocated here.
  Nil retains existing standalone IDGen's non-OOM contract; no invented global
  memory-service state or Budget-error-to-global-OOM equivalence.
- NewSATClauseAllocator(config) → *satClauseAllocator,error; MkClause(literals,
  learned) → satStoredClauseRef,error; Delete(ref) error; Offset(ref) → scalar offset;
  GetClause(offset) → scalar ref; IsLive(ref), ID(ref), Size(ref), Capacity(ref),
  Literal(ref,index), SetLiteral(ref,index,value), AppendLiterals(ref,callerBuffer).
  Every clause member below is an owner method receiving ref. Close reclaims
  tracked storage; source has no allocator Reset, so no invented logical-reset
  algorithm is claimed. IDs recycle ONLY Delete's actual source recycle site.
- Internal satClauseHotHeader contains uint32 ID/size/capacity/approx/flag lanes;
  all retained data is pointer-free. The canonical five-word header+literal
  byte encoding lives in SmallBytes directly, with scalar typed word accessors.
  No duplicate uncharged literal snapshot or second parallel clause model.
  Physical byte charges are real allocation lengths and real descriptor/directory
  metadata, not assumed C++ struct stride. The five-word codec is a Go storage
  adapter; its selected-baseline size coincidence does not assert ABI equivalence.
  Object extent is checked uint64 `20+4*n`, including n0 and UINTMAX n
  (17179869200 bytes). Extents <=UINTMAX use real SmallBytes; larger host64
  extents use the concrete direct backend in sat_clause_large.go. Actual host
  byte bounds precede Reserve/make, not an invented 4GB clause limit.
  The full scalar locator adds storageKind to owner/slot/generation. A wide
  cold directory is embedded in the charged parent (no second owner header),
  capacity*actual reflect stride, with exact `5+n` uint32 payload words charged
  once. Unpublished directory preparations refund on payload failure; fault
  attempt counters never rewind. Free rows retain generation ledger; exhausted
  rows retire, independent of source ID recycling. Finite backend fixtures test
  these real algorithms at 28 bytes; separate huge-count fixtures prove extent
  and branch arithmetic plus budget refusal without multi-GB allocations.
- satTmpClause is a cold unique owner with nullable ref. NewSATTmpClause(Budget),
  Get, Set(literals,learned), SetBinary(l1,l2,learned), SetBinClause(encoded), Close.
  Reuse the SAME clause codec/access methods through a direct-only storage owner;
  temporary allocation is source alloc_svect(char,size), not SmallBytes pooled
  placement. New tmp ID is UINT_MAX; it never calls IDGen.Mk/Recycle. The public
  allocator rejects releasing another owner's temporary ref. Destruction releases
  its direct storage; no source rational/resource destructor callback exists.
- satBinClause: two source encoded uint32 lanes, constructor, Literal1/2, Learned,
  Equal. satClauseWrapper: scalar clause ref or first literal plus sentinel second
  literal, Binary, Size, Literal, ContainsLiteral/Var, ClauseRef; binary wrapper
  allocates no clause. Null second literal makes a source binary wrapper's union
  unusable as a clause pointer: typed dangling-union guard, no invented tag fix.
- satClauseVector and satClauseWrapperVector: cold owners, concrete scalar backing,
  constructors, Push/Get/Set/Pop/ResizeFill/Shrink/Append/AppendTo/Reset/Finalize/Close,
  Clone/Assign/MoveClone/MoveAssign and cold checked iteration. ClauseVector has
  source DisplayInto; wrapper scalar has its own display, no invented dedicated
  source wrapper-vector display overload.
  Borrow one real allocator; copy raw refs without touching clause data, and
  preserve defined cross-owner aliasing via cold borrowed allocator bindings if
  copying different domains is allowed (do not fake owned copies of clauses).
- satClauseSet: NewSATClauseSet(real allocator,Budget), Contains, Empty, Size,
  Insert, Erase(ref), Pop() → ref, Reset, Finalize, CheckInvariant and checked
  bucket-position iteration/AppendTo, Clone/Assign/MoveClone/MoveAssign, Close.
  Membership follows SOURCE ID, not generation; borrowed clause validity is a
  separate typed guard before dereferencing. A set is bound to one allocator
  ID domain; foreign allocators have independent IDs and must not silently share
  one ID-position vector. Defined source same-ID membership/overwrite quirks
  remain; source debug invariants diagnose ID aliasing.
- satWatched: source val1/val2 uint32 lanes plus scalar provenance/generation
  companions only for clause offsets. Binary/Ternary/Clause/External constructors;
  all source predicates/getters/setters, MarkNotLearned, Equal/NotEqual and Less.
  Binary/ternary/external have zero companions and exact two-word scalar equality;
  clause offsets use full locator identity instead of accidental cross-owner
  slot equality. This widens hot record size; actual reflect stride is charged.
  No offset bits are sacrificed to tags: source offset lives in val1, literal/tag
  packing remains exactly source val2. Raw source equality against invalid/dangling
  old addresses is not promised as a checked-handle domain operation.
- satWatchList: charged owner/backing, exact vector lifecycle/copy/move and full
  scalar access/append/set-end/erase helpers, comparator and DisplayInto(real
  allocator,real NumericTextStream). EraseClauseWatch first matching clause offset,
  EraseTernaryWatch constructs source-sorted ternary record then vector.erase.
  Both shift later entries LEFT and preserve order (neither swap-removes).
- satJustification: default None, Binary/Ternary/Clause/External factories,
  Kind/predicates/literal getters, raw ClauseOffset/ExternalIndex, DisplayInto.
  Clause offset widened as above; scalar assignment/copy only, no owning allocation.
- satActivityDomain: genuine concrete charged owner of mutable uint32 vector,
  Push/Resize/Get/Set/Len/Reset/Finalize/Clone/Assign/MoveClone/MoveAssign/Close.
  Source uint32 activity arithmetic belongs to caller; Set NEVER automatically
  invokes queue callbacks. Queue borrows this OWNER, so backing growth/assignment
  reads through current activity storage; no priority snapshot or hidden closure.
- satVarQueue: constructor(activity,Budget), ActivityIncreased, MkVar, DelVar,
  UnassignVar, Reset, Empty, NextVar, Clone and MoveClone, Close/Stats plus cold
  diagnostic Contains/CheckInvariant for tests. Comparator activity[a]>activity[b]
  has NO variable-ID tiebreaker. New concrete private activity heap holds int32
  values/indices and borrowed domain pointer. No source copy/move ASSIGN API:
  comparator reference member deletes those operations, verified below.

Source vector specialization validity matters: watched, bin_clause and
clause_wrapper have NO default constructor. Their source vector(size) and
resize(size) default-object forms are ill-formed if instantiated; expose actual
ResizeFill/size+value/source-array constructors, not fabricated zero records.
Clause-pointer vector default resize creates null refs; activity default resize
creates0. Common executable vector operations include Size/Capacity/Empty/Get/
Set/GetDefault/SetX/Back/Push/Pop/Insert/ReserveFill/ResizeFill/Shrink/SetEnd/Erase
(first matching, shifts)/Contains/Append/Fill/Reverse/Swap and forward/reverse
checked iteration, plus constructor/copy/move/reset/finalize routes. Source
nonempty self Append is nonterminating: typed adapter before actions; empty self
valid. The implementation is for valid concrete instantiations, not every theoretical
uninstantiable template method. Cold watch-list owners own only concrete scalar
record arrays; future solver's outer watch-list collection must use scalar list
references/common owner metadata, not hot []Go-owner-pointers. That outer solver
storage is not implemented or claimed by this standalone packet.

The spelling is concrete and source mapping is enumerated below; no arithmetic
or solver algorithms are represented by standalone storage tests.
Caller append/output buffers and final strings are caller-owned boundaries.

## Every clause method and exact lifetime/control flow

sat_clause.cpp:27-42 allocates header/literal capacity exactly initial size;
initial removed/used/frozen/reinit/inactivity/glue/psm false/zero, learned argument,
memcpy all literals, mark_strengthened recomputes variable approximation. Source
approx is unsigned32 mask of variable&31, not literal-index mask. Constructor
check_approx is inside SASSERT and therefore executed only in source-debug;
size>1 assertion is LAST after constructor body.

sat_clause.h:58-95: ID/Size; both indexed literal overloads; Learned/UnsetLearned;
Shrink; Strengthened/MarkStrengthened/UnmarkStrengthened; Elim; Removed/SetRemoved;
Approx/UpdateApprox/CheckApprox; begin/end via checked scalar iteration;
ContainsLiteral/Var; SatisfiedBy; Used mark/unmark/query; inactivity inc/reset/query;
Frozen/Freeze/Unfreeze; static ApproxLiterals; Glue/PSM setters/getters;
ReinitStack getter/setter. Literal writes MUST NOT implicitly refresh approx or
strengthened. Shrink recomputes only on a decrease, never changes capacity.
Elim removes first matching literal and shifts remainder, then decrements size
and marks/recomputes. Satisfaction reads actual three-valued model and returns
true only positive=true or negative=false; undefined satisfies neither. Missing
model index is genuine source OOB and gets a typed guard at actual read.
Inactivity wraps eight bits; glue/psm saturate255, source flag bits remain distinct.

CheckApprox cpp:55-60 saves old mask, recomputes/mutates real approximation, then
source-debug asserts equality. Stale mask debug error occurs AFTER the mutation;
release returns true after mutation. Do not implement a read-only validator.

Tmp Set cpp:105-124 destroys old object and clears pointer BEFORE larger allocate.
If capacity suffices, only size/learned/literal bytes change: ID/approx/strengthened/
removed/used/frozen/reinit/inactivity/glue/psm remain stale/preserved. No call to
mark_strengthened. New temporary uses normal constructor body and UINT_MAX ID.
Budget failure after old release leaves Get nil; successful capacities retained.
Source tmp copy is raw-pointer aliasing and double-cleanup-unsafe; user destructor
suppresses actual implicit move. Allocator implicit shallow copy similarly aliases
owned SmallBytes chunks/free lists. Neither has a safe source owning Clone contract:
Go rejects shallow owners by self identity, rather than inventing source deep copy.

Allocator cpp:198-205 allocates actual object bytes BEFORE obtaining source ID,
then constructs. Delete cpp:208-220 recycles ID BEFORE destructor/storage release;
if recycle array allocation fails, clause remains live. If true source OOM observer
skips recycle, physical deletion still occurs. ID generator wraps/reuses according
to real existing source operations; IDs are never silently retired at generation
exhaustion. Descriptor generations can retire physical slots independently. Source
allocator destructor requires caller destruction/deletion of live clauses (solver
cpp:60-73 does this); Go tracked-owner Close reclaims remaining inert clause storage
as explicit cleanup adaptation, without fake extension/destructor callbacks.

Clause display cpp:223-238: '(' space-separated literals ')' then x if removed,
+ if strengthened,* if learned. Clause vector emits each clause then LF. Wrapper
display cpp:259-267 lacks all flag suffixes. Literal output is SOURCE two insertions:
(sign?"-":"") then unsigned variable. Empty sign CString insertion still consumes
Width; do not flatten literal into one host string or use dimacs variable+1.
All renderers use real NumericTextStream actions, preserving flags/base/width/
alignment/fill and exact default bytes; source intermediate host strings absent.

## Release/debug domains and packed quirks

Preserve defined release bodies, not blanket stronger preconditions:

- Clause construction zero/one sizes is defined in release; source-debug sz>1
  fails after body. Preserve allocation/ID/initialized-header prefix at that site
  with typed assertion diagnostics; expose produced scalar ref where needed.
  For Tmp new(mem) constructor failure, source pointer assignment has not completed:
  Get stays nil, allocated storage remains owner-tracked for Close/explicit
  diagnostic cleanup. Do not pretend that field publication preceded assertion.
- Elim absent on a NONEMPTY clause is release-defined: no shift, decrements size
  and drops last literal, then refreshes approximation. Debug error precedes that
  body mutation. Empty absent Elim would underflow and then scan OOB: typed guard.
- Shrink greater than size is release no-op/debug error. Invalid learned/frozen
  state for UnsetLearned/Freeze/Unfreeze performs ordinary release bit write;
  source-debug diagnoses at actual pre-body assertion. Indexed OOB/dangling refs
  receive typed guards in all profiles because actual source continuation is UB.
- BinClause val2=(second.index<<1)+learned, uint32 wrap; no source roundtrip assert.
  Equal compares BOTH encoded words directly in each orientation. It is not
  ordinary unordered pair equality. Preserve raw high-bit loss in second literal.
- Watched ternary sorts indices BEFORE packing, asserts distinct and second-literal
  roundtrip in debug; val2=TERNARY+(l2.index<<2). Clause constructor blocked literal
  similarly shifts2 and source-debug roundtrip asserts. RELEASE high two bits
  truncate; debug diagnosis is source site, not a universal input cap.
  SetBlockedLiteral/SetClause do NOT have roundtrip assertion even in debug;
  preserve their uint32 truncation. Binary Learned tests val2>>2==1, not merely
  bit2!=0; BinaryNonLearned is exact val2==0. MarkNotLearned asserts learned then
  rewrites binary tag and verifies result.
- Watched external constructor stores index in val1/tag3 in val2; getter RETURNS
  val2. Release getter/display therefore return3 for every index. Debug constructor
  roundtrip fails except supplied3. Do not fix getter to val1. Standalone setter/
  equality/raw-record tests distinguish external index0/3/UINTMAX and debug errors.
- Watched Less: binary before nonbinary, ternary before clause/external, false
  within groups or between clause/external. Source solver.cpp:1251-1255 uses
  stable_sort, not arbitrary sorting; if exposing standalone SortByKind, use
  allocation-free concrete STABLE grouping and test within-group order. Do not
  claim solver::sort_watch_lits ported merely from this storage helper.
- Justification None/Binary/Clause/External predicates compare exact val2 where
  source does; Kind masks low3. Ternary val2=TERNARY+(l2.index<<3), truncates top3
  bits in release AND debug, no constructor roundtrip assertion or sorting.
  ClauseOffset and ExternalIndex getters have NO kind assertions: preserve raw
  val1 result with scalar locator companions, which cannot resolve if not a real
  live clause handle. Only three literal-getter source assertions exist.

Wrong-kind watched/literal justification getters also have SASSERT-only checks:
preserve their defined scalar release extraction and diagnose in debug, instead
of making every getter an unconditional ErrKind. A scalar raw word interpretation
is distinct from an actual invalid clause-pointer dereference, which is genuinely
undefined and receives an all-profile checked-locator guard.

Use dedicated g3debug profile constants and typed ErrSATAssertion, with optional
source tag conditions for heap/clause_set CASSERTs. They are selected assertion
instrumentation adapters, not invented solver callbacks or a fake global debug
runtime. Validate source-debug and release behaviors separately in permanent
hermetic tests. No production/tests may depend on C++ compiler/reference source.

## Set, vectors, activity heap and failed-publication continuations

Source vector:0→2 then(3*old+1)/2; exact source-capacity deep copy, ascending raw
record copy, Reset retains backing, Finalize releases. Resize expands repeatedly
BEFORE length/fill publication; retain each successful capacity prefix on failure.
Cold owners reflect-sized once; actual arrays capacity*reflect stride with full
old+new overlap, host-byte/source-count checked before Reserve/make. No fictitious
source pointer/vector header bytes added when represented by actual Go owner
scalars. Embedded IDGen metadata is charged in owner header, its free array once.
SmallBytes nested headers/pages/directory/descriptor charges are not duplicated.
Tmp direct allocation metadata/generation and any owner cleanup logs are charged.
Borrowed activity/allocator/stream/Budget/callback environment remain external.

ptr_vector/svector have move constructors, but ONLY const-copy assignment overload:
ClauseVector/WrapperVector/ClauseSet/Activity move construction transfers backing;
move assignment COPYs and leaves source unchanged. watch_list is vector<watched>,
which has genuine move assignment: release destination then steal source backing.
Source ClauseSet implicit copy/assignment operates ID positions THEN borrowed
clause vector; destructor reverse order. Failure after first-member publication
remains a prefix; do not globally stage everything atomically. Raw scalar vector
copy never resolves live clause bytes, including stale/closed borrowed identities.
Cold owner binding transfer preserves aliasing, not deep object copies.

ClauseSet insert cpp:23-31 resizes ID positions to id+1 filledUINT_MAX BEFORE
membership check, then publishes position BEFORE pushing clause ref. Same-ID
insert is early return after reserve, not generation-based set equality. Erase
cpp:34-51 swaps last reference into removed position and updates that moved ID;
Pop cpp:54-63 removes last and marks its ID sentinel. Reset/finalize act ID vector
then clause vector. CheckInvariant cpp:66-90 debug checks forward AND reverse
maps; release body returns true without traversal. CASSERT calls only run when
source clause_set tag enabled. Source UINTMAX tmp ID makes id+1 wrap then index
OOB: typed guard, not valid set insertion or a billion-element fixture.

Source position publication/push allocation failure leaves Contains true though
no clause ref was appended. Preserve that observable raw prefix. Retry may return
already-present; no hidden repair/rollback. Define typed incomplete-publication/OOB
continuations at the actual endangered vector access; keep completed writes,
allow safe raw Size/Contains and Reset/Finalize cleanup. Independent diagnostics
can inspect consistency in all profiles, but do not claim source release
CheckInvariant detects such a ghost. Every retained partial array is budget-owned.

Queue constructor source heap.cpp/header: values.push(-1) THEN index bounds.resize128;
index capacity comes from repeated source vector growth (not make capacity128).
MkVar reserves v+1 BEFORE insert; priority comparator reads actual borrowed
uint32 activity with strict > and no tie breaker. ActivityIncreased calls heap
DECREASED only if present. DelVar erases only if present; Unassign inserts only if
absent (it does NOT reserve). Reset retains arrays/sentinel. NextVar asserts
nonempty then source erase_min. Signed heap value/index/source shifting domain is
int32: guard genuine negative/INT_MAX signed-overflow/OOB at actual operations,
not narrowing uint32 activity values. Activity writes and notification remain
separate source actions, including unsigned wrap/invariant consequences.

Move_up/down must preserve parent/left/right comparison/read/order and exact ties:
right child wins only if strictly preferred; equal children choose left; equal
priority does not move. Erase chooses up iff last value preferred to parent,
otherwise down. Erase_min special one-value branch has no unnecessary comparator.

IMPORTANT source order must be preserved, not a generic heap assumption: source heap.h:231-240 sets indices[val]=newIdx BEFORE
fallible values.push_back. On growth failure, ghost inverse position remains and
Contains reports true. Empty Reset returns early and can retain this ghost.
Source check_invariant_core checks live forward values/parents only, not a full
reverse-only ghost scan; stronger Go diagnostics must be labeled separately.
Retain source action prefixes and typed guards for later actual OOB continuation,
not atomic Insert/automatic ghost repair. Heap bounds resize likewise preserves
every successful capacity growth before eventual failure. Root committed the existing IntHeap source-prefix and forward-only invariant
fix as 6b41257; that shared file remains root-owned. A concrete activity
specialization still avoids boxed comparators/allocated closures by directly
borrowing satActivityDomain.

Queue's reference comparator deletes copy/move assignment: DO NOT add successful
Assign/MoveAssign methods. Clone copies heap arrays at exact capacities and borrows
same current activity owner; MoveClone transfers arrays with same reference. Cold
shallow copies fail self identity. Cross-budget transfers reserve destination
charges before mutation and refund prepared Used on failure; source logical
membership/ID actions do not change. Budget fault attempt counters never rewind.

## TDD and actual source test/consumer inventory

No standalone sat_clause/sat_watched/sat_var_queue storage test or TST(sat) exists
in preserved src/test. There are ZERO source ENSURE sites in the six scoped SAT
storage headers/cpps. Textual source SASSERT inventory is59: clause.h10,clause.cpp13,
clause_set.cpp7,watched.h25,justification.h3,var_queue.h1. Five clause.cpp sites
(140,154,161,187,189) are pointer-segment platform checks replaced by explicit
checked locator/provenance/storage adapters, not pretend translated address tests.
ClauseSet has two actual CASSERT calls (31/51); watched has four static tag-range
assertions. Heap dependency has15 SASSERT/19 CASSERT textual sites, not59 extra
SAT assertions. Exact source assertions, including multiple assertions on one
line, must be ported FIRST; publish actual observed registration and counts.

Registered heap test main.cpp:149 has11 actual ENSURE sites in test/heap.cpp:
source N10000, three seeds0..2, 30000 insertion trials and100000 dynamic trials
per seed. Existing IntHeap tests already cover real source integer comparators.
For new descending-activity specialization, port applicable membership/invariant/
extraction assertions through the actual domain (label comparator adaptation);
independent max-activity oracle must not claim source ascending-comparator fixture
was unchanged. No per-trial print stream or huge log is required for source
assertions; printed diagnostics are distinct from algorithm/assertion counts.

Registered small_object_allocator test main.cpp:166 is actual allocation/free
consumer with zero ENSURE; its source operations already covered by genuine
SmallBytes, so clause tests use actual dependency rather than duplicatefakepool.
Registered sat_user_scope main.cpp:239 has ONE solver-coherence ENSURE at test:76;
source tst loops while(true). It compares actual incremental/rebuilt solvers and
cannot be claimed by clause-storage oracle or invented finite miniSAT. This real
solver test remains OPEN until solver functionality and an explicit finite test
execution adaptation are implemented.

Real consumers audited: solver.cpp:135-155 activity/queue variable creation;
256-336 real ternary/nary allocation/watch/justification and source literal writes;
566ff traversal/compaction;1251-1255 stable watch sort;2576 unassign and2704 delete;
solver.h:518-520 activity += increment THEN queue notification; simplifier.h:53/58
clause set and tmp owner, simplifier.cpp:452/695/820 tmp reuse; sat_integrity_checker.cpp:200ff and
sat_model_converter.cpp:155ff wrapper traversal. These actual consumers remain unported;
storage methods must support their real aliasing and order, not fake their results.

Implemented source-first groups and independent permanent oracles:

1. Clause ctor/flags/capacity/literal writes/approx collision32 and CheckApprox
   mutation; release/debug size0/1, absent Elim, state flags at actual assertion
   sites; ordered first-match elimination; tiny model true/false/undef oracle.
2. Temporary grow destroy-before-allocate failures, reuse preserves all stale
   flags/mask, UINTMAX ID, three Set routes, debug constructor publication site,
   exactly-once owning Close and rawref liveness/generation reuse.
3. Real allocator tiny SmallBytes chunks/pooled/direct/diagnostic, storage-before-ID,
   ID recycle-before-free including recycle failure/OOM-observer skip, independent
   physical handles/provenance/generation vs logical IDs, rawoffset roundtrip and
   reuse/reset/exhaustion bounds. No fabricated seven-address-segment tests.
4. Binary encoded equality pathological reverse comparison and width truncation;
   wrapper null-sentinel union guard, source display versus clause suffixes;
   owning vectors copy/move/reset/finalize/capacity/failure/borrow aliases.
5. ClauseSet independent ID-position/membership/swap-remove/pop oracle, actual
   reserve-before-duplicate, position/push failure ghost and retry/erase/cleanup
   continuation, scoped debug invariant tags, stale/foreign/closed borrowers.
6. Watched every constructor/getter/setter/kind/roundtrip including EXTERNAL
   defect release3/debug-non3 error, setter truncation, precise source Less and
   stable group fixture; ordered clause/ternary erase; source display all kinds;
   all owning watchlist copy/move/finalization/storage roots.
7. Justification every tag/packed ternary raw high-bit domain/default/unguarded
   raw getters, exact ostream action/value fixtures and scalar copy, no extension.
8. Actual borrowed activity queue with source sentinel/reserve128 growth capacities,
   comparator rereads writes, same-priority traversal, present/absent events,
   source insert ghost, empty reset ghost, min/up/down action order, bounded
   independent membership/max oracle and applicable real heap assertions.
   Deep Clone and MoveClone tests and reflection/compile-surface regression that
   deleted Assign/MoveAssign are absent; invalid Go shallow-owner rejection.
9. Every cold header/backing/old+new peak and constructor/member/binding failure
   prefix immediately after error BEFORE Close hides leaks; owner/source ID/gen
   limits, simulated32bit host byte bounds, pointer-freedom architecture audit,
   warmed non-growth operations0Goalloc, no closure/per-record allocations.
10. Full native/checked+debug, targeted race and actual named Node JS RUN/PASS,
    checked vet/WASI and real source/no-generics audit, isolated committedHEAD+
    owned files only. Frozen hashes and exact source registration evidence.

## Reference probe and source hashes

Bounded scratch probe `.cache/sat-storage-traits.cpp` includes the ACTUAL preserved
SAT headers. `TMPDIR=golang/.cache/tmp g++ -std=c++17 -D_AMD64_ -Isrc -fsyntax-only`
passed: source sizeof(sat::clause)==20 in this selected AMD64 baseline; var_queue
copy/move constructors available, copy/move assignment deleted. This is external
reference evidence only, not Go layout/size proof or a production/test dependency.
Permanent Go regressions above preserve the corresponding lifecycle distinctions.
No external scratch files/compiler outputs were written.

| Source | SHA256 |
| --- | --- |
| src/sat/sat_clause.h | b66b1e936801b63d3d93f504cd934089ed755a4720081d03c94effc1a5c57707 |
| src/sat/sat_clause.cpp | 862cd6ea1c065ee1ef4e1477380e4f93d2e1a66facbcf8ec627f7ec496b7ec82 |
| src/sat/sat_clause_set.h | 4b69e9601f4fb27e16cf80727d32d6cbca5793c32baccd626deb4c9d6d76622d |
| src/sat/sat_clause_set.cpp | ca1fe1b4e757849ab7b0d2a6dce8e0391ec0ebcb3d174df28a0e5e4585001db0 |
| src/sat/sat_watched.h | 1d2969f295a5f9ab042c5997fdc0717dc6cfe74711c6037802a780c78c72bf28 |
| src/sat/sat_watched.cpp | 8ffc73b9e0e8cd99b2d5a663af51f3aa629f522d4bfdce6e896c066be3037d27 |
| src/sat/sat_justification.h | 0a3af366718264233e8a9bc7598d871fccf150356a36c010cee3462396258a48 |
| src/sat/sat_var_queue.h | 642aaa8dd58e029bef06825e7ca0a8662c00fea89f79dcfd75e17b8ada6ded68 |
| src/util/heap.h | 6451a4522708f62df4ead0d6bc893820987d60012e532decb92d57646b8249bd |
| src/util/id_gen.h | 1708ba4b1bf14008c6e60e656b2b964cc3ab1dd7e0768f25242b347cd37da04f |
| src/util/small_object_allocator.h | 57b0a3bbc2d9fd10f008f371e3b5b9bee5535d39ab5cfbcb4afd9324018a140b |
| src/util/small_object_allocator.cpp | 70cf910ed0a3871ea110ee01ff8d23a687f74af5972ab995dce32ba3e6e35994 |
| src/test/heap.cpp | 3706ff918ca7b64a8cb44ec3647a2507a71db4fb02c79dd35d73e73183bde8fc |
| src/test/sat_user_scope.cpp | c86c383ea199a89a5311bae5f8186e0b59e6953aa17bb7d675e221aa0c5edffc |

Implementation and final ownership/gate evidence below await parent integration.
Full solver/backend/extension functionality remains OPEN; complete standalone
storage does not complete its real solver consumers.

## Exact completed mapping and additional boundary adaptations

| Source site | Concrete implementation |
| --- | --- |
| clause.h:56-95; clause.cpp:27-103 | five-word clause construction; all ID/size/indexed read/write, learned/strengthened/removed/used/frozen/reinit flags, capacity, approx, first-match Elim/Shrink, model satisfaction, 8-bit inactivity and glue/psm methods in sat_clause_storage.go |
| clause.cpp:44-60; clause.h:75-76,87 | SATApproxLiterals/SATApproxLiteralsN, UpdateApprox/CheckApprox, checked LiteralIter; no exposed raw slice |
| clause.h:114-135; clause.cpp:105-124 | encoded satBinClause and all three tmp Set routes; source destroy-before-grow/reuse and unique Close |
| clause.cpp:125-220; clause.h:143-174 | real SmallBytes-backed allocator, budgeted utilIDGen, generation/provenance offsets, wide direct directory, Delete and tracked-owner Close |
| clause.h:182-210; clause.cpp:241-267 | sentinel satClauseWrapper, binary scalar reads, source literal/variable Contains and suffix-free DisplayInto |
| clause.cpp:223-239 | clause and clause-vector NumericTextStream display; exact sign CString then unsigned-variable action order |
| clause_set.h:30-65; clause_set.cpp:23-92 | Insert/Erase/Pop/Contains/Size/Empty, source Reset/Finalize order, source-profile CheckInvariant, iteration, actual implicit Clone/Assign/MoveClone/MoveAssign |
| watched.h:53-119 | every encoded constructor/tag/predicate/getter/setter, source external getter defect, equality/inequality |
| watched.h:125-130; watched.cpp:23-83 | comparator, owning watchlist, ordered first-match clause/ternary erase and all four NumericTextStream display branches |
| justification.h:27-86 | scalar default/factories/tags/predicates/literal/raw offset/index access and complete display switch |
| var_queue.h:33-75; heap.h:122-263 | genuine borrowed uint32 activity, source sentinel/bounds constructor, MkVar/Del/Unassign/ActivityIncreased/Reset/Empty/NextVar, actual up/down and priority reads |
| vector.h:129-217,230-513 | concrete watch, clause-pointer, wrapper, binary and activity owners: valid default/fill/array constructors, source capacities/deep copy/move distinctions, scalar access, Push/Pop/Insert, ResizeFill/ReserveFill, Shrink/SetEnd, Append/Reverse/Fill/Swap, iteration and cleanup |

Concrete methods and explicit thin-route mappings cover the valid vector bodies.
Source reserve(n) on activity/clause-pointer vectors maps to ReserveFill(n,0) or
ReserveFill(n,satStoredClauseRef{}); clear() maps to Reset(). Raw c_ptr, iterator
raw pointer comparisons and mutable reference escape are intentionally replaced
by checked concrete iterator values/Get/Set, not advertised as raw pointer APIs.
No completeness of an unrelated obj_hashtable template is claimed: clause_set is
the actual pair of source vectors, not a hash-table substitution. Watch/bin/wrapper
source default-size/default-resize instantiations are ill-formed, so are absent.
Wrapper has no source operator== in C++17; wrapper-vector Contains/Erase(value)
are likewise ill-formed and absent (scalar wrapper Contains(literal/var) remains).
Insert(value) is the source push alias; InsertAt and EraseAt are explicit Go
index adapters. Default-sized activity and pointer-vector construction uses exact
capacity n, whereas size+value/array constructors use source repeated growth.
Source fill(size,value) recurses into itself after default resize; FillSize on
the valid activity/pointer specializations keeps that resize prefix then reports
ErrSATNonterminating, rather than silently repairing the source overload.

Release clause literal reads/writes past current size but below physical capacity
remain defined retained-literal accesses; debug diagnoses idx<size at its source
site. Binary wrapper release indices 2/UINTMAX still extract second; debug errors.
Scalar svector/ptr-vector retained-capacity access/size writes are preserved,
while watch-vector access consults explicit initialized-lifetime metadata.
Source erase(iterator) shifts and decrements WITHOUT a destructor, so its trailing
initialized watched record remains live and release Get/Set are permitted there.
Pop/Shrink/SetEnd/Reset destroy only their actual logical ranges; erase→pop can
leave a dead hole below a still-live erase tail, so a single highwater cannot model
this. Debug idx<size assertions still occur before lifetime/OOB guards.
Source UINTMAX SetX performs resize(0) first, then typed guard at endangered
index write; no early guard hides that completed prefix. Absent Elim on size
UINTMAX would increment UINTMAX to zero then write index UINTMAX: scalar boundary
helper guards that actual OOB after debug assertion, distinct from valid last match.

Clause-vector owners retain explicit charged cold allocator binding directories
for cross-owner copies, with stable scalar origins. Raw copies/MoveClone retain
closed identities without dereference. Public BindAllocator checks live owner;
copy adoption checks identity only. Rendering resolves the real borrowed owner
and returns ErrClosed for dead storage; no owned spelling/clause copy is made.
Binding allocation prefixes are explicit Go ownership adapters, retained and
charged immediately; Close releases them. Set intentionally stays in one source
ID domain, preventing unrelated allocators' colliding IDs from forming a fake set.

Heap contains receives source int32: unsigned event >INTMAX converts to negative
then would index negative; ActivityIncreased/Del/Unassign/MkVar have typed OOB
guards, distinct from valid high positive absent-event no-ops. Heap move really
leaves values size0, so source Empty (size==1) is false after move. Diagnostic Size
returns0, NextVar guards endangered empty backing, and source Reset recreates
sentinel via real growth while retaining completed failure prefixes. Reference
comparator forbids Assign/MoveAssign; permanent reflection/lifecycle checks verify
this, rather than copying a cold mutex/reference as a Go value.

Each owner and its borrowed activity/Budget are caller-serialized; no global
solver callback/state or per-record closure exists. Cold self identity rejects
unsafe shallow owners. Iterator epochs saturate on an operation reaching UINTMAX64
and reject subsequent structural operations, never wrap to resurrect iterators.
Source vector expansion checks source byte-domain and actual host-byte bounds;
Go widened scalar-record strides are charged physically, not assumed C++ stride.
Pointersegment packing/native address hashes are explicitly replaced by full
scalar checked locators, not seven fabricated address segments. Optional tag
CASSERT runtime and TRACE streams are not reconstructed as global solver runtime.

## TDD and frozen test evidence

Initial real isolated RED: missing storage API in committed HEAD snapshot plus
only initial source/lifetime tests. Behavioral RED then GREEN: binary wrapper
release extraction, retained clause literal reads, foreign-owner vector copy
rendering, scalar retained vector bodies, unsigned negative queue event. Missing
source APIs were observed RED before concrete bin vector, valid constructors,
FillSize recursion adapter, iterator/static count helper, unsigned Elim helper.
Epoch wrap had behavioral RED then saturation fix. Existing methods' integration
fixtures that immediately passed are not falsely described as production RED.

Constructor failure sweeps, 28-byte real wide backend ledger/retirement/rollback,
independent 12,000-event mutable priority oracle, 3,000-event concrete vector
oracle, all 27 finite 3-valued model valuations, release/debug packed boundary
fixtures, old+new backing peak refusal and source failed-publication immediate
roots are permanent hermetic Go tests. Warm real pooled clause Allocate/Delete
and queue events have zero warmed Go allocation checks. No reference source or
compiler is required at ordinary test time. No solver test was invented.

Initial freeze1 full native/checked-debug, 32 named race/actual JS, vet, WASI
and source/no-generics audit passed. Final unsigned-negative event fixtures then
exposed DelVar's missing high-int guard; that real behavioral RED was corrected
before the complete final reruns below. Only terminal final evidence is used
for this final freeze.


## Iterator allocation review and tests-first correction

The parent/numeric supplemental audit found12 factory-returned iterator pointers
escaping to uncharged host heap storage (10 concrete vector forward/reverse
factories, ClauseSet.Iter and ClauseAllocator.LiteralIter). A compiler method-
expression contract regression first failed in isolated HEAD9f8974d+this packet
(test80593, exit1) against pointer-returning factories. All12 then changed to
source-faithful concrete value returns. The original traversal bodies and Next
pointer receivers, owner identity and mutation epochs remain unchanged.

The permanent escaping-value test assigns each iterator into a typed global
value sink, with no interface boxing, under Budget.FailAfter(0). Each of the12
factories measures0 host allocations with testing.AllocsPerRun, leaves Used
unchanged, and preserves the armed fault prefix. Actual traversal from the
escaped values and immediate complete owner cleanup are checked. Both new groups
pass native, checked-debug race and actual checked-debug JS. Other source storage
bodies remain frozen; source self-append nontermination guards retain the prior
explicit documented adapter. Previous fa840a5 gate evidence is superseded below
by the full9f8974d+corrected iterator packet reruns, which include committed MPFF.

## Scalar vector supplemental source review

Full remaining vector review found two concrete source-route defects; both have
permanent tests written before production corrections. Source vector.h:288-299
set_end has no size assertion for existing scalar backing, whereas shrink:389
asserts new size<=old size in debug. The four scalar SetEnd methods now publish
size independently, allowing initialized retained capacity in both profiles.
TestSATScalarSetEndSourcePublication first failed all four checked/debug cases
(RED58565); its release cases already passed. WatchList's true destructor loop
retains its separate guarded lifetime domain; its implementation was not changed.

Source vector.h:74-90 trivial-reallocate growth preserves full physical backing,
including initialized scalar tails retained after Shrink. The four scalar grow
methods now copy full old capacity rather than current length. A failure fixture
initializes size/capacity3, shrinks to1, allows expansion3→5 and fails expansion5→8
before any size/constructor publication. TestSATScalarExpansionRetainsBackingPrefix
first failed with lost tail values for all four variants (RED93306); the corrected
packet preserves each tail and verifies immediate live length/capacity/budget
charges before any repair or teardown. Both new groups then passed checked/debug
GREEN19984 and the final release/debug/race/actualJS reruns below. No source
assertion count is invented from these independent Go regressions. The latest
reruns supersede iterator-only freeze gates for final vector/test bytes.

## Watch erased-tail lifetime review and correction

The independent numeric audit found that watched remains trivially copyable even
though its watch_list uses CallDestructors=true. Source vector.h:371-384 erase
shifts/decrements size without ending the trailing object's lifetime; expansion
uses trivial reallocate and retains its bytes. The release behavioral regression
TestSATWatchEraseRetainedLifetimesAndGrowthPrefix first failed Get(4) after two
erases from size/capacity5 (RED19924). It now verifies release reads/writes with
debug assertions, successful5→8/failed8→12 growth retention, mixed erase→pop
holes, copy-only-logical-size construction, move/swap lifetime transfer and
immediate complete cleanup. TestSATWatchLifetimeMetadataBudget separately audits
actual reflected owner/data/bool bytes, old+new peaks, armed clone unwind,
logical-only Reset destruction and Finalize/Close release. Forty targeted SAT
checked-debug groups were GREEN37301 before freezing final gates.

satWatchList owns a physically charged []bool array with one flag per physical
capacity slot. Its slice header lives in the reflected cold owner. The watched
array and bool array use one aggregate reservation before either new allocation;
this is explicit cold Go allocation-action aggregation, preserving source repeated
growth order and completed successful backing prefixes. Host/source bounds remain
on actual individual allocations, not an invented combined host-size cap. The
complete old watched+bool backing remains charged through preparation, and each
successful replacement releases both old charges. Stats.BackingBytes includes
both actual arrays. Copy constructs flags only for logical source elements;
growth copies full data/lifetime capacity; move and swap carry both arrays and
charges. Erase does not clear flags; scalar-vector tails are unchanged. Existing
failure/peak tests now include bool byte stride rather than silently uncharging
metadata. Source object lifetimes remain distinct from Go array allocation.

The final40-group gates supersede earlier38-group scalar and36-group iterator
freezes. Only vector implementation, vector tests and the existing independent
model's exact watch stride charge changed in this final watch-lifetime correction;
other packet/source bytes stay fixed. No solver or extension algorithm added.

## Final frozen gates (isolated HEAD9f8974d; iterator, scalar-vector and watch-lifetime corrections)

Snapshot: golang/.cache/release/sat-storage-iterator-final, `git archive 9f8974d:golang`
plus ONLY the 16 new Go files below. Other agents' live drafts excluded.
All GOCACHE/GOTMPDIR/TMPDIR and output remained below golang/.cache.

| Gate | Terminal result |
| --- | --- |
| full native `go test ./... -count=1` | PASS, 44356; main1.110s, mpnum22.841s |
| full `g3checked g3debug` tests | PASS, 53274; main1.107s, mpnum23.623s |
| named checked-debug `-race -count=1 -v` | PASS, 39761; all40 exact RUN/PASS, sat-storage-watch-final-race.log |
| checked-debug JS compile + actual minimal Node wrapper | PASS, compile88807 + actual exit0; all40 exact RUN/PASS, sat-storage-watch-final-js.log |
| full checked-debug vet | PASS, exit0 |
| full WASI checked-debug production cross-build | PASS, exit0 |
| source/no-generics portcheck | PASS, 96910; 2102units/677APIs,33ported/68partial/2001unported; snapshot HEAD metadata unchanged |

No source storage ENSURE or SAT solver assertion is invented from the 40 Go
groups. Exact named registration was independently checked in BOTH verbose race
and actual JS logs, not inferred from a cached compile or aggregate package PASS.
Native and checked-debug source profile cases are exercised separately. Header/
array ownership includes callback function field; OutOfMemory environment is
borrowed and must be readonly/nonreentrant, caller-serialized and nonpanicking.
Caller-owned iterator values/output objects are cold boundaries, not retained SAT records. All12 iterator factories return concrete values, matching source iterators and IntHeap.Begin/End. Next remains a pointer receiver updating caller state; epochs, provenance and traversal bodies are unchanged. No iterator owner allocation or new retained charge is introduced.

All source files/hashes above remain unchanged. No manifest/progress/Git changes
or writes outside golang occurred in this packet. Code/tests are frozen for
parent review/integration; remaining global debug tags/TRACE and pointer ABI
representations are explicit adapters/open runtime prerequisites, not solver code.

| Frozen Go file | SHA256 |
| --- | --- |
| sat_clause_large.go | 8f334dc1c2756206dd0eb1fbba275667713cdbfe68093dba13b9707d52b199ce |
| sat_clause_set.go | e20438dee423e6e10455c15e25aa594a15b5f0fe7aadea25eb605b2eb5f2ca92 |
| sat_clause_set_test.go | 35f4c78e9a6de0c4a419eddbfec33e8398505e959317b21a64af822236117fec |
| sat_clause_storage.go | 2b8bd2e824897c4156a8ec29c660f5c5ce49d499fc62b8253aeb8578f82a7c92 |
| sat_clause_storage_test.go | 186c2a42cd5175c17943b36048afeb89bcd90911ef13520772a4c9f5dc527b66 |
| sat_justification.go | 50028968fe1ec7d0efcfe243695a7812866a6d2e483cb27992c9fdbfbfc91c81 |
| sat_justification_test.go | fb1765483221c26a858bce143750796e4d72051b6633fe2665b9a98cf955bc66 |
| sat_storage_debug.go | 1c38fe9557a945f6abe43db93def02bb1f6e58fd4566ecf5f5ee05921000c102 |
| sat_storage_model_test.go | ab5af90853aa552ed0827ec4e96e848b7c64dc780dc15cb51d7dce796f12ddb9 |
| sat_storage_nodebug.go | 3de2d4c58ba58efeda5abc21e893a4fe82b2aa8ff6d62545e8efeddfa4f734bf |
| sat_storage_vector.go | 495f424ffc16bac31fb94264e3ef1af0ee96171cb602dc221b3f593700bdca63 |
| sat_storage_vector_test.go | 7e12d9c8459adbfcdd121bde9d3fcc0f0d21bb0ed29f5b7eda1f3703799c2eb6 |
| sat_var_queue.go | 207fe621df027c805dc711b697372a7c810cf61b3e172417120e0ffd14e28e71 |
| sat_var_queue_test.go | ef65322dd234e2d93299f8b419de910e5b697bdf0fcd79ab9acec6a0951c819a |
| sat_watched.go | f14235f1e2190cd01de89cc13f8c0f2f4b286b72feca9c9d1eb7eefa9eb742be |
| sat_watched_test.go | 0a6fac8b4946b9fdf805bb2a3350bfa0dee822aa7b7b9ded543b741191ed1e42 |

Exact permanent named groups (40):

- TestSATBinClauseVectorRealSpecialization
- TestSATBinaryEncodedEqualityAndWrapper
- TestSATBinaryWrapperReleaseIndexBody
- TestSATClauseCheckedIterationAndApproxCount
- TestSATClauseElimUnsignedWrapBoundary
- TestSATClauseReleaseDebugBodiesAndModels
- TestSATClauseReleaseRetainedLiteralAccess
- TestSATClauseSetActualCopyMoveFailureOrder
- TestSATClauseSetSourceOrderAndGhostPrefixes
- TestSATClauseSourceIDsLocatorsAndDeleteFailure
- TestSATClauseSourceLifecycleFlagsAndApprox
- TestSATClauseVectorBorrowedForeignCopyLifetime
- TestSATClauseWideExtentAndBudgetRefusal
- TestSATConcreteSourceConstructorsAndFillRecursion
- TestSATConcreteVectorIndependentOperations
- TestSATConcreteVectorSourceCapacityAndCopyPolicies
- TestSATIteratorConcreteValueContracts
- TestSATIteratorEscapingValuesNoAllocations
- TestSATJustificationSourceAllKindsAndRawGetters
- TestSATLargeClauseBackendLedgerBudgetAndFaults
- TestSATPackedSourceGetterSetterDomains
- TestSATScalarExpansionRetainsBackingPrefix
- TestSATScalarSetEndSourcePublication
- TestSATScalarVectorReleaseRetainedBodies
- TestSATStorageOwnerIdentityExhaustionAndOOMRecycle
- TestSATStoragePointerFreeOwnershipAndConstructorFailure
- TestSATStorageSourceTextAndWidthActions
- TestSATStorageVectorFailurePrefixesAndBudgetPeaks
- TestSATStorageWarmNoAllocationsAndClosedBorrow
- TestSATTmpClauseSourceReuseAndDestroyBeforeGrow
- TestSATVarQueueConstructorCloneFailureAndMovedReset
- TestSATVarQueueIndependentMutableActivityModel
- TestSATVarQueueRealActivityPriorityEventsAndLifecycle
- TestSATVarQueueSourceGhostAndEmptyReset
- TestSATVarQueueUnsignedNegativeIndexBoundary
- TestSATVectorCrossBudgetSwapAndMoveFailure
- TestSATVectorIteratorEpochRetiresWithoutWrap
- TestSATWatchEraseRetainedLifetimesAndGrowthPrefix
- TestSATWatchLifetimeMetadataBudget
- TestSATWatchedSourceTagsWidthDefectAndErase

Coordinator integration uses committed AST-core HEAD07a6182 plus exactly these
16 frozen Go files. Source rows are partial for global conditional debug-tag/
TRACE/OOM policy and real solver/extension consumers. The full standalone SAT
solver remains the next algorithmic packet; these storage tests produce no SAT
result. Central metadata is separately audited on the exact prospective tree.

Coordinator prospective HEAD07a6182+16-file validation passed full native
(54940, root2.461s/mpnum22.721s), full checked/debug (67106, root2.436s/
mpnum23.190s), and central source/no-generics audit (35728:2,102 units,677 APIs,
33 ported,77 partial,1,992 unported). The agent's frozen forty-group race/actual
JS, vet and WASI evidence above covers identical SAT Go bytes. The coordinator
revalidated every file against the sixteen-entry frozen SHA inventory.
