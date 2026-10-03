# Concrete scoped symbol table

Owned files: util_symbol_table.go, util_symbol_table_test.go and this report.
No edits to the existing symbol service, global ledger, manifest or git metadata.
This packet translates the complete source symbol_table<int> successful behavior
to SymbolInt32Table. Other template payload types remain separate prerequisites;
it does not complete parser/AST/solver symbol consumers.

## Source mapping

All src/util/symbol_table.h and src/test/symbol_table.cpp were read, together with
the applicable full core_hashtable policies and vector storage operations.

| Source | Concrete behavior |
| --- | --- |
| symbol_table.h:33-55 key_data constructors | scalar Symbol/int32 entry, default scalar initialization |
| :57-70 hash/equality | symbol cached-prefix hash read through borrowed manager, comparable scalar identity |
| :72-106 hash_entry methods | null FREE, DummySymbol DELETED; no stored hash/state field; sentinel changes preserve inactive data |
| :115-119 reset | Reset clears table, then trail/limits lengths, retaining vector capacity |
| :121-133 find/contains | Find, FindInto and Contains; miss preserves caller output |
| :135-137 get_scope_level | ScopeLevel |
| :139-158 insert | scoped overwrite trails previous pair and bypasses table growth; absent scoped binding trails marked key then normal insertion; unscoped normal insertion |
| :160-162 begin_scope | BeginScope appends scalar trail limit |
| :164-182 end_scope | EndScope pops limit before reverse undo; new binding unmarked before erase; old pair inserted before each trail pop |
| :184-188 append | Append traverses source physical buckets in ascending order and inserts through scoped policy |
| :190-200 get_range/get_dom | AppendRange/AppendDom preserve caller prefixes and ascending used-bucket order |
| test/symbol_table.cpp:24-39 | all ten ENSURE assertions preserved |
| hashtable.h:28,244-255 | default8, power-of-two initial capacity (including1) |
| :160-220,222-241 | rehash source bucket order, double expansion and same-capacity tombstone cleanup |
| :286-308 | empty reset no-op; pre-clear FREE overhead decides strict >75% half shrink when capacity>16 |
| :324-351 | ascending used-bucket iteration |
| :353-401 | strict 4*(used+deleted)>3*capacity growth before duplicate lookup; linear probing/wrap, last tombstone wins |
| :481-507 | hash+identity lookup stopping at FREE |
| :533-580 | erase becomes FREE only when next slot FREE, otherwise DELETED; cleanup only deleted>used AND deleted>64 |
| vector.h:59-97,215-224,343-365 | trail/limit cap2 then (3*cap+1)/2; reset/pop retain capacity |

Raw source ABI/pointer tags are represented by existing scalar Symbol tokens.
Source uninitialized inactive int payloads start at zero in Go; retained initialized
payloads remain unchanged when an entry becomes FREE/DELETED. Scope limits store
the uint32 bit pattern of the source int-vector/unsigned trail length.

## Ownership, checked preconditions and budgets

The table borrows one live SymbolManager. The manager and its spelling storage
are never charged or closed by this table. Explicit common owner domain replaces
the source global symbol interner. Foreign spelling tokens, dead owners, null and
Dummy sentinel keys and marked keys are rejected before storage changes. Empty
EndScope returns ErrSymbolTableScope. Foreign-manager Append is rejected even
when symbols are numerical. Source self-Append paths that keep the iterator valid
are supported; an unscoped self-Append requiring rehash returns
ErrSymbolTablePrecondition before action, guarding source iterator invalidation
rather than inventing snapshot semantics.

Owner header, actual entry backing, retained trail backing and retained scope
backing are fully charged. The source vector header counters become owner fields
already included in the Go header charge; there is no fictitious second charge.
Constructor failure releases all prepared storage. Close is idempotent, refunds
the owner's graph and does not require its borrowed manager to remain open.
Reset/EndScope retain source vector capacity. Returned domain/range buffers belong
to the caller and do not expose internal arrays; their output allocation is outside
retained-owner accounting. No Go generics, unsafe, per-entry callbacks, boxing or
Go pointers appear in hot records. Cold mutexes serialize table operations and
Append lock ordering; shared Budget access by different owners remains caller
serialized. Public operations lock the borrowed manager while reading hash bytes.

Insertion prepares trail/table replacements before publication and accounts full
old+new peaks. Potentially allocating multi-step undo/Append uses charged temporary
transaction storage and preserves original table/trail/limits on predictable
failure. Physical copies retain all FREE/DELETED/USED buckets and inactive payloads;
they are NOT source copy-constructor rehashes. The staged undo pops its limit before
body replay. Temporary transaction storage is an explicit Go failure-atomicity
adaptation to source throwing allocations and memory::is_out_of_memory cleanup
guards. Budget failures return errors instead of exposing partially undone scopes.
Successful Reserve attempts consume the usual FailAfter prefix even if later
preparation fails; only Used charges are refunded.

Conservative direct EndScope proof: reverse restoration overwrites extant bindings,
and erase never increases used+deleted. Initially <=75% occupied and
deleted+marked-new-records<=64 excludes both growth and tombstone cleanup, so replay
is failure-free without workspace. Append direct replay similarly bounds maximum
occupancy by initial used+deleted+missing keys and checks retained trail capacity
against every possible push. Otherwise it stages. Warm nonrehash scope/lookup and
Append/self-Append tests measure exactly zero Go allocations, including FailAfter0.

## TDD and validation

First source assertions and independent tests failed compilation due to absent
concrete API (RED). After implementation, all source assertions and initial tests
passed; added source-state/failure coverage before final handoff. Fourteen finite
tests cover 2000-operation nested snapshot oracle, independent2500-operation bucket
oracle, collisions/wrap/last tombstones, retained inactive payloads, dynamic cached
symbol hash prefix, source strict growth/overwrite exception, real >64 tombstone
cleanup, vector growth/retention, domain/range order/prefixes, constructor unwinds,
both trail/table preparation failures, staged undo and Append failure sweeps with
successful endpoints, ownership/sentinel guards, source-valid self-Append,
conservative zero-allocation paths,1000-cycle plateau, concurrent cold mutex, and
exact undo peak (one byte below fails unchanged; exact amount succeeds).

- Full CGO-disabled native and g3checked go test ./... passed.
- Final fourteen SymbolInt32Table tests passed native, g3checked and native race.
- go vet ./... passed; final go vet . passed after the final peak regression.
- js/wasm checked and wasip1/wasm root test builds passed.
- Actual checked Node wasm execution printed all fourteen new test names, all PASS.
- go run ./cmd/portcheck passed, including no-generics syntax audit.
- Normal workspace compiler environment; Node-only minimal environment preserved
  TMPDIR. All cache/temp/output stayed under golang/.cache.

Tests require no source checkout or reference executable. No solver/domain
algorithms, other symbol-table payload types or whole-port completion are claimed.

Source SHA256:

- symbol_table.h: ebb97249e3df7344be8926974cc3fbee266cf2d802bbbb456b536b262862a38a
- test/symbol_table.cpp: ba72eb9b1ae2aa8fd66adfbd340ccb8d8f4eb01405a893ed3a6e24c6829b8321
- hashtable.h: bdc562a7db2c0d914dd896577c43e0648f501538873460d2e1a6c2d0932f64f2
- vector.h: aafafc0fe6ca0afd7bafe0c781342da259f9c34c0b94b16732c415cf2733066e

## Independent second source review

Re-read complete symbol_table.h/tests against the final concrete implementation.
Added one meaningful boundary test before fixing the preflight: an empty scope
at >75% occupancy incorrectly required staging and failed under FailAfter0 (RED).
The source only pops a limit in that case, including when retained tombstones
exceed64 (a second RED assertion in the same regression, fixed with direct empty
limit pop). Marked-only undo similarly cannot
trigger insertion growth; scoped Append with only existing keys bypasses growth
checks in every source insert. The conservative direct proofs now account for
these facts without changing bucket states, mark history or undo order.
TestSymbolInt32TableHighOccupancyNoGrowthScopes covers all three cases and measures
zero warmed allocations for empty high-occupancy scopes (GREEN).

All fifteen symbol-table tests passed in isolated symbol-file builds, native,
g3checked and race; isolated vet and checked JS/WASI compilation passed. Actual
checked Node JS printed all fifteen names, all PASS. Isolated builds explicitly
used util_symbol.go, util_symbol_table.go and their two test files because concurrent
IntHeap tests were in their intentional RED phase. This is not a claim of a new
full-workspace gate after that unrelated edit. The earlier full-workspace gates
remain historical evidence before the review adjustment.

Exact completed source methods for this int32 specialization: key_data's three
constructors; key_data_hash_proc::operator(); key_data_eq_proc::operator();
hash_entry constructor/get_hash/is_free/is_deleted/is_used/get_data (mutable and
const)/set_data/set_hash/mark_as_deleted/mark_as_free; symbol_table implicit
constructor/destructor and reset/find/contains/get_scope_level/insert/begin_scope/
end_scope/append/get_range/get_dom. No named symbol_table source method remains
untranslated for this payload. Other payload specializations and source raw ABI,
source allocation exception exposure and invalid sentinel/self-iterator behavior
remain the documented adaptation/separate-scope items above.

## Prospective isolated release gate

Created golang/.cache/release/scoped-table-final with git archive HEAD:golang at
5c4498f302873cdc51df1da38ec41d99d8ea74ac. Overlaid only this packet's three files;
all uncommitted unrelated root work was excluded. Fresh build cache/temp/runtime
output stayed inside that tree's .cache. Full CGO-disabled native ./...,
g3checked native ./..., and actual g3checked JS/wasm ./... all passed (count1,
verbose). Native sessions6387, checked-native5719 and checked-JS45950 finished
exit0. All fifteen new names were compared against the test declarations and
found registered and PASS in every log. All seven tested packages passed; cmd
portcheck has no tests. Normal workspace compiler environment and Node-only
minimal wrapper with retained TMPDIR were used.

Verbose logs: .cache/release/scoped-table-final/.cache/native.log,
checked-native.log, checked-js.log. No manifest or git writes were made.

## Source implicit deep copy and assignment completion

Added Clone(destinationBudget *arena.Budget) and Assign(other *SymbolInt32Table)
after re-reading symbol_table.h implicit member operations, core_hashtable's copy
constructor/destructor/assignment, and complete vector copy_core/copy constructor/
assignment/destruction paths. First new tests observed missing-method compile RED;
implementation and four new source-policy/lifetime/failure tests are GREEN.

Clone maps core_hashtable copy constructor (hashtable.h:261-269): exact source
table capacity, USED-only rehash in source bucket order, deleted count0. It copies
both vectors at exact source capacities with live marked history and limits
(vector.h:102-120,156-164). Spare vector records are not constructed/copied in C++
and are zero-initialized in Go. This differs intentionally from the physical
transaction copy, which preserves every original bucket and inactive record.
The owner and all arrays are independent, borrow the same SymbolManager, and charge
the supplied destination Budget (nil is unlimited). Fresh cold mutexes are created;
no Go mutex is copied as a value. Source destruction maps Close's own-graph release.

Assign maps member assignment in source declaration order: core_hashtable::operator=
(hashtable.h:620-627) first RESETs the destination and then inserts source USED
buckets through ordinary hash-table growth policy. Assignment table capacity is
therefore determined by destination reset/shrink/grow history, rather than cloned
source capacity. Only afterward are trail and limit vector capacities/live elements
copied via vector::operator= (vector.h:190-203). Assign retains destination manager
and Budget, requires the same borrowed manager, and self-assignment is a no-op
after lifetime checks. Never-allocated source vectors release destination retained
capacity; allocated-but-empty source vectors retain their source capacities in the
copy. This source distinction was found during the audit and tested explicitly.

Assignment stages a charged physical destination graph, executes source table
reset/replay and vector copies in it, and commits only after successful preparation.
Source reset-first allocation exceptions could expose partial assignment; Go
predictable failures instead preserve destination state. All simultaneous old,
staged and newly prepared arrays are charged. Clone constructor unwind releases
header and all successfully prepared arrays. Successful fault prefixes consume
the existing Budget controller policy; Used charges unwind on later failure.
Non-self copies allocate as the source operations require, plus the documented
assignment transaction workspace; no warm zero-allocation copy promise is made.

New tests: SourceCloneScopedHistory, SourceAssignDestinationPolicy,
CopyFailureSweeps, CopyEmptyAndExactPeaks. They cover tombstone removal and exact
source-capacity clone bucket replay; pointer independence; marked nested history
undo independently after copy; destination half shrink/ordinary growth on assign;
source close while clone remains usable; shared and separate destination budgets;
empty-but-retained versus never-allocated source vectors; clone/assignment fault
sweeps with both failed and successful endpoints; self/foreign/dead-owner guards;
and exact clone/assignment old+new peaks (one byte below fails unchanged, exact
amount succeeds). Existing fifteen tests still pass, including no-allocation paths.

Prospective final tree: golang/.cache/release/scoped-table-copy-final, from
git archive HEAD:golang at8a8ae0a70582d7779dc1b4e6d301e7ff637b72b1, overlaid ONLY
the three owned scoped-table files. Full native and g3checked native ./...,
actual full g3checked JS/wasm ./..., full vet, checked scoped-table races,
root WASI test build and no-generics portcheck all PASS. All nineteen scoped-table
test names were compared with declaration names and found registered/PASS in
each native/checked-native/actual-JS verbose log. Native sessions83880,
checked-native83646, actual-JS66549, vet48677, WASI47515 completed exit0;
checked race54379 and portcheck80046 also exit0. Logs are tree/.cache/native.log,
checked-native.log and checked-js.log. Workspace full gates during the same turn
were blocked by the unrelated numeric Root packet's intentional compile RED;
the complete green gates above use the isolated prospective tree. All fresh
cache/temp/runtime outputs remain under golang/.cache, with normal compiler
environment and minimal Node-only wrapper preserving TMPDIR.

The source implicit deep copy/assignment dependency for this concrete int32 table
is now covered. Other payload specializations, domain consumers and any separate
move/swap API remain outside this packet. No manifest/progress/git or other owner
files were changed.

Central scope audit: this chunk completes every explicitly declared symbol_table
method for int32. Synthesized copy construction/assignment still require explicit
owned APIs; other payload specializations remain open. The source header remains
partial in the manifest. Full combined native, checked races (root corrected
heap fixture rerun), actual checked JS and WASI compilation also pass.
