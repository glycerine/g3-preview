# Complete selected univariate root isolation and refinement packet

The approved selected numerical implementation is frozen for coordinator review over committed `96e3058`; the evidence below distinguishes initial and final bases. This is not whole upolynomial/algebraic-number completion. Actual sources: `src/math/polynomial/upolynomial.cpp`2081–2518 and2599–2948, corresponding complete public/protected declarations in `upolynomial.h`, `src/test/upolynomial.cpp`68–278/491–598/654–722/817–830/main1059–1094, `util/mpbq.h/.cpp`, `util/vector.h`, `util/scoped_numeral_vector.h`, and actual consumers in `math/polynomial/algebraic_numbers.cpp`603–630/694/977/1423/1533/1643. Repository-wide search finds **no source `refine_intervals` method**: callers iterate intervals and invoke the existing single-interval methods. Do not invent a source method or count for that spelling.

The selected packet is the entire actual DRS/Sturm isolation, square-free/zero/default dispatch, endpoint-to-refinable conversion, both one-step and precision refinement overloads, rational-to-binary interval conversion and interval normalization. No isolation through known-root lookup, numerical sampling, alternate Sturm implementation or rational-only replacement. Algebraic cell construction/factorization/multivariate conversion/printers remain genuine later dependencies; this packet makes their actual univariate numerical paths possible.

## Dependencies and actual defaults

Committed source owners already provide all arithmetic algorithms needed: IntegerManager/IntegerModularManager, polynomial storage/set/normalize/square_free/signed remainder, all transforms/bounds, sequence/Sturm and sign evaluation, BinaryRationalManager Add/Sub/Neg/Div2/Mul2/Mul2k/Power/Copy/Swap/Reset/Del/comparisons/LessOneDiv2k, FromRational, RefineUpper/RefineLower, rational comparisons and ToRational. Binding uses the actual same IntegerManager, real utilReslimit and `_NO_OMP_` utilCooperation; diagnostic BinaryRationalManager uses actual RationalService as required by g3debug. Existing numeric wrappers' documented failure-fidelity debt remains open; it is not made exact by an isolation wrapper.

Real missing storage is owning BinaryRational vectors plus DRS/SS frames; no new polynomial scratch arrays replace source members. Source `sqf_nz_isolate_roots` calls **DRS**, with the Sturm alternative only commented out. Both outer DRS and Sturm wrappers select actual `#if 1` Knuth bounds; `#else` Cauchy branches are disabled inventory, not run-time options to pretend were active. Source DRS core chooses `pos_k=max(neg_k,pos_k)` for its positive auxiliary copy, then uses original neg_k for negative input transformation. Preserve this asymmetry and direct mutation of the mutable core input. No restore transaction is present.

The sole existing numerical edit factors RemoveZeroRoots into its unchanged shared state body; no other numerical/core/service changes were required. New external binary vectors borrow BinaryRationalManager and its numerical/diagnostic owners, all of which must outlive vectors. This explicit caller lifetime contract avoids pretending the current BinaryRationalManager has a live-vector counter; a counter/Close enforcement would require separately coordinated existing-file permission. Source raw numerals similarly borrow a manager.

## Concrete API and physical storage

Approved new production files:

- `mpnum/upolynomial_binary_vector.go`: concrete `BinaryRationalVector`, private typed `upolyBinaryVectorState`, actual source-style growth/size/initialized metadata and stats.
- `mpnum/upolynomial_isolation.go`: all DRS/public dispatch bodies, exact packed DRS frames and output adjustments.
- `mpnum/upolynomial_sturm_isolation.go`: actual SS frame stack and full Sturm isolation bodies.
- `mpnum/upolynomial_refinement.go`: all refining/normalizing/conversion overloads.

Implemented constructor `(*BinaryRationalManager).NewVector() (*BinaryRationalVector,error)`; owner methods `Size() (uint32,error)`, `PushCopy(*BinaryRational) error`, `Read(i uint32,out *BinaryRational) error`, `Set(i uint32,*BinaryRational) error`, `Reset() error`, `Shrink(uint32) error`, `Resize(uint32) error`, `Reserve(uint32) error`, `Close() error`, `Stats() BinaryRationalVectorStats`. Private pushDefault publishes an actual default value and size **before** algorithm Copy/Swap. PushCopy maps `_scoped_numeral_vector<mpbq_manager>::push_back`, not a prebuilt boxed value. Algorithms take three explicit vector owners `(roots,lowers,uppers *BinaryRationalVector)`; do not collapse them into a combined reservation/publication. A single composite output is unnecessary and would obscure source prefixes. Source output-alias behavior is not silently replaced with a blanket three-distinct requirement; shared valid owner arguments must preserve actual reads, mutations and assertion sites, with typed guards only at undefined accesses.

Cold vector owner fields: self pointer, borrowed actual BinaryRationalManager, typed state, charged header count, closed scalar. Hot backing records contain actual `BinaryRational` (actual typed Int plus uint32 k), never per-element pointers or reencoded owning numeric tokens. State uses a typed slice and scalar size/initialized metadata; capacity is the actual backing capacity. No public borrowed slot view or invented relocation directory is exported. Actual reflected owner/header and actual full capacity stride are charged; selected LP64 source `sizeof(mpbq)=24` and unsigned header8 are distinct source extent checks. No fictitious source malloc charge. Growth source2→3→5… performs literal unsigned source old/new extent comparison, separately tested by `upolyIsolationSourceCapacity`. `upolyIsolationNextCapacity` then applies a distinct Go full-capacity domain guard: wide source stride×capacity+8 must fit UINT_MAX, and actual typed bytes must fit host allocation. This additional rejection is not a source overflow comparison. Source-allowed wrapped extents can allocate an undersized raw block before subsequent out-of-bounds moves/writes; their raw allocation/header/publication prefix is unimplemented, rather than claimed as a defined typed backing topology. Reserve full new backing while old remains charged, publish/move source fields in actual order, release old; no numerical clone/Del during moves. `mpbq` contains nontrivial `mpz`, so this is the real nontrivial move path, not a trivial memcpy claim. Metadata directory, if needed for scalar views, is concrete and fully charged.

Public Read is an explicitly owning copy adapter. Internal numerical actions borrow actual slots at each source dereference, never capture an owning Go slice or raw pointer across growth. Initialized high-water permits defined release access to retained initialized slots; g3debug applies actual logical index assertions. Set into a live initialized slot invokes real Copy; Reset destroys ascending and retains capacity; Shrink destroys ascending suffix before publishing requested source size; Resize follows the actual scoped wrapper calls, including source resize fill behavior. Complete generic vector template/copy/swap/iterator surfaces are not claimed by this selected storage API. Implicit owning Go owner copies rejected; no unsafe/type punning/generics.

Private DRS frame: two uint32 fields `parent` and packed `sizeFlags`, actual source size30/first1/left1; construction masks to the source30-bit width rather than inventing a precondition. Selected source and actual record stride8. Source parent UINT_MAX sentinel and ascending child push/right-first LIFO behavior retained. SS frame: actual two typed BinaryRational values plus two uint32 sign variation counts, selected LP64 source stride56; actual Go stride reflected separately (profile-dependent). No owner pointer inside frames. Frame stacks are lexical typed arrays, concrete scalar metadata and charged full backings/peaks, not per-frame allocations. SS growth moves numerical ownership; popped frames Del lower then upper; stack destructor loops ascending remaining frames, lower then upper, then frees storage. Packed DRS frame has no numeric destructor.

Stats separately identify cold/vector bytes, physical backing capacities and actual live owned numerator roots. Combined immediate numerical audit adds manager polynomials/cache + BinaryRationalManager scratch + caller numeric outputs + binary vectors + live SS frames + lexical legitimate outputs; the same backing is never charged or counted twice. Source array relocation and borrow generations are checked only at actual reads. Failure cleanup releases live numerical companions including owner failed sentinels by existing approved cleanup APIs; no automatic missing TMP repair.

## Complete method/action map

Every public numerical API returns error in addition to source result. Boolean means the source boolean (true interval, false precise root stored in lower), not success/failure status. Receiver is `*UPolynomialManager`; shared `v UPolynomialView,bqm *BinaryRationalManager` prefix, explicit output vectors or endpoint pointers.

| Actual source body | Concrete API/private counterpart and action order |
|---|---|
| drs_frame ctor; pop_top_frame | packed scalar ctor; `popDRS`: assert nonempty, top size bound, repeatedly Del polynomial stack back then pop, finally frame pop |
| push_child_frames | private `pushDRSChildren`: parent index captured; actual m_push_tmp Set→ComposeTwo→Normalize; each left default-push then Copy, then left frame; Translate from actual last left coefficient span into m_push_tmp→Normalize; right default-push then ownership Swap, then right frame. No pre-reserving both siblings or input snapshot |
| add_isolating_interval | local raw lower0/upper1; walk exact parent chain, left Div2 lower/upper or right AddInteger1 lower/upper then Div2 lower/upper; push lower default then upper default, then Swap lower then upper. Failure between pushes preserves unequal lengths |
| add_root | local raw half1/2, parent walk right AddInteger1 before Div2; roots default-push then Swap. Source local raw mpbq has no automatic numeric destructor; safe Go error cleanup explicitly documented |
| drs_isolate_0_1_roots | private full DRS0_1; Descartes0→return,1→lower0 push then upper1 push,otherwise lexical q,p_stack,frames; initial half root appended before removal/child frames; actual while checkpoint, first flag cleared before bound, exact pop/interval/half-root/child branches. Root and parent frame remain until both children consumed |
| adjust_pos/adjust_neg/swap_lowers_uppers | actual vector range starting at captured previous counts; positive Mul2k; negative Mul2k then Neg; lower/upper swap only after both adjustment traversals; same-size debug assertions at actual sites |
| mutable drs_isolate_roots overload | `DRSIsolateRootsWithBounds(v,neg_k,pos_k,bqm,roots,lowers,uppers) error`: lexical aux Copy; pos=max; positive composition/isolation/three adjustments; mutate actual v with PMinusX then negative composition; negative isolation/three adjustments/endpoint swaps. Completed input/output mutation retained on refusal/cancel |
| const drs wrapper | `DRSIsolateRoots(v,bqm,roots,lowers,uppers) error`: debug output size/nonzero checks; lexical p1 Set+Normalize; Knuth bounds on original raw p, not normalized copy; mutable core on p1. TRACE Cauchy/display actions remain printer-dependent open |
| ss_frame_stack destructor; pop_ss_frame | ascending remaining lower→upper Del, then member backing; pop does lower→upper Del before frame pop |
| ss_add_isolating_interval | lower default push, upper default push, Copy lower, Copy upper; failures retain actual partial metadata/numeric prefix |
| sturm_isolate_roots_core | `SturmIsolateRootsWithBounds`: source nonzero assertion; lexical seq,mid,currLower,currUpper; real Sturm; frame stack declared after Sturm; variation minus,zero,plus then no-root shortcut; Power2 lower→Neg→Power2 upper; source macro body exactly below; while checkpoint, read counts, Swap current lower/upper from top frame, pop, strict difference assertion, Add/Div2 midpoint, real variation, left macro then right macro |
| PUSH_SS_FRAME macro | debug distinct endpoints first; equal counts no action; difference1 EvalSign upper before decide precise-root push/default/Copy vs interval helper; otherwise frame default-push before Copy lower/upper before scalar count stores. No early all-or-nothing frame construction |
| sturm outer wrapper | `SturmIsolateRoots`: size assertion, lexical Set/Normalize, active Knuth bounds on original input, actual mutable core. #else Cauchy remains disabled inventory |
| sqf_nz_isolate_roots | private source nonzero assertion then DRS (not Sturm) |
| sqf_isolate_roots | `SquareFreeIsolateRoots`: actual bqm reset roots, then lowers, then uppers; zero branch roots.Push(0) before lexical nz_p/removezero/assert/core; nonzero directcore |
| isolate_roots | `IsolateRoots`: debug sz>0, lexical sqf_p, actual SquareFree algorithm, then SquareFreeIsolateRoots. Not sample roots, not polynomial literal special cases |
| isolating2refinable | `IsolatingToRefinable(v,bqm,a,b) (bool,error)`: EvalSign a then b; case1 opposite nonzero immediate return; cases2/3 midpoint loop and endpoint swaps exactly; case4 b1/a2/newa1/newb2 construction, first midpoint Copy before Eval, then alternating two-sided probes/swap/updates. Return root in a, no invented checkpoint in these loops |
| refine_core single | `RefineCore(v,sign_a,bqm,a,b)`: source debug recomputes a then b signs/assertions; lexical mid Add→Div2→Eval; exact root swaps into a/returnsfalse; sign_a swap into a, otherwise debug opposite then swap into b |
| refine single | `Refine(v,bqm,a,b)`: actual EvalSign a/debugnonzero then core |
| refine_core precision | `RefineCoreToPrecision(v,sign_a,bqm,a,b,prec_k)`: real debug sign checks; lexical w; checkpoint BEFORE width Sub; LessOneDiv2k shortcut, else single core. Strict width <2^-k, not <= |
| refine precision | `RefineToPrecision`: Eval a, debug Eval b/opposite/nonzero then precision core |
| convert_q2bq_interval | `ConvertRationalToBinaryInterval(v,qm,a,b,bqm,c,d)`: source EvalRational a then b, debug signs; scoped lower/upper; exact FromRational(a) swap/eq branch or source bracketing/refine-upper loops with sign tests and optional d publication; then missing-d b branch/refine-lower loops. Precise root swaps c, explicit Del lower and upper before false return (then scoped cleanup), not an omitted double-cleanup action. Final actual source inequalities/sign checks. qm must share source actual IntegerManager; no import-clone adapter inserted |
| normalize_interval_core | `NormalizeIntervalCore(v,sign_a,bqm,a,b)`: IsNeg(a)&&IsPos(b) short-circuit; source INT_MIN unknown-sign sentinel, actual Eval/debug branch; zero-root false without invented reset; else EvalZero and Reset exactly a or b; final opposite-sign debug check |
| normalize_interval | `NormalizeInterval` delegates sentinel to core; all source non-crossing intervals return true untouched |

Source stack/raw mpbq locals have no owned destructor except scoped wrappers. Go always cleans temporary numeric roots on typed error/cancellation as the explicit safe ownership adaptation; normal completion cleanup/action order remains literal. Errors preserve completed scalar/numeric/output prefixes. No promises of operation rollback, identical C++ unsafe exceptional continuation, or callbacks outside actual checkpoint sites. All g3debug conditionals execute actual source predicate/numerical calls and short-circuit order; g3checked is only handle validation. TRACE/display/decimal print collection remains open until a real printer/event packet; registered numerical routes may execute without claiming those printer actions.

## Complete registered consumer inventory / tests-first plan

Phase-one tests own NEW `upolynomial_isolation_source_test.go`, `upolynomial_refinement_source_test.go`, `upolynomial_isolation_model_test.go` and optional `mpnum/upolynomial_isolation_storage_test.go` for nonallocating extent/packed-frame cases. No existing fixture or production changes. First missing-API RED overlays only these onto the final committed Sturm prerequisite. Root literal fixtures use actual upolySourceOwnerServices, actual compatible BinaryRationalManager+RationalService diagnostics and owning vectors. Multivariate `polynomial_ref` expression construction/to_numeral_vector is explicitly a **dense input adapter**, not claimed converted source functionality. Hermetic exact coefficients are derived through actual dense factor products or independent test-only big integer expansion. No testdata C++ dependency. Phase two begins only after API/layout review.

### Actual isolation datasets

`tst_isolate_roots()` contains **12 calls**, invoked **twice** in registered main1078/1088. Preserve both invocations and the exact expected rational approximants/precisions, not just one representative dataset.

| Dataset dense recipe | Expected matching list from source | precision |
|---|---|---|
| (x-1)(x-2) | 1,2 |5|
| (x-1)^2*x^3 |1,0|5|
| x^5-x-1 |11673039/10000000 (approx)|5|
| (x-1)(x+1)(x+2)(x+3)(x-3)^2 |1,-1,-2,-3,3|5|
| (10000x-31)(10000x-32) |31/10000,32/10000|10|
| previous times(10000x-33) |31/10000,32/10000,33/10000|10|
| (x^5-x-1)(x^5+x-1)(1000x-1167) |11673039/10000000,75487766/100000000 (approx),1167/1000|10|
| (x-2)(x-4)(x-8)(x-16)(x-32)(x-64)(2x-1)(4x-1)(8x-1)(16x-1)(32x-1) |2,4,8,16,32,64,1/2,1/4,1/8,1/16,1/32|10|
| dataset7 squared |same three|10|
| P17=x^17+5x^16+3x^15+10x^13+13x^10+x^9+8x^5+3x²+7 | -413582/100000,-170309/100000,-109968/100000 (approx)|10|
| P17*(x^5-x-1)^2*(x³-2)^2 |previous three plus11673039/10000000,125992/100000 (approx)|10|
| (x^5-1000000000)^3*(3x-10000000)^2*(10x-632)^2 |630957/10000 (approx),10000000/3,632/10|10|

Their expected list sizes sum43 per invocation,86 across the two calls. The expected values are used only **after actual algorithm output**, with source strict containment or exact equality and actual visited-slot bookkeeping. The approximations are not asserted to be exact roots. They must not choose production branches or supply outputs.

`tst_isolate_roots5`817–830 is separately active and executes the **real degree70** polynomial:

`x^70 -6x^65 -x^60 +60x^55 -54x^50 -230x^45 +274x^40 +542x^35 -615x^30 -1120x^25 +1500x^20 -160x^15 -395x^10 +76x^5 +34`.

It runs the same isolation/checking helper with precision10 and no supplied root list. Do not replace it with the degree14-in-x^5 mathematical shortcut in production. Tests can independently compare root counts with exact rational Sturm, but actual full-degree source paths still run. `#if 0`747–815 isolate2/isolate3 stay disabled; main's commented isolate4/gcd2 are not registered executions.

The active common helper68–147 constructs Sturm on original input, square-free component and Fourier on that component, then verifies **actual algorithm outputs**:

- line104 ENSURE exact output count = Vminus−Vplus (once per25 helper calls:24 list cases+degree70).
- line107 ENSURE every exact root EvalSign==0 (R executions, determined by actual split).
- line121 ENSURE each interval has a root endpoint OR Sturm variation difference1, with literal short-circuit action order (I executions).
- line132 VERIFY same endpoint checks OR Fourier difference positive odd, after unconditional fsvLower/fsvUpper evaluation (I executions).
- line143 VERIFY DescartesBoundInterval on square-free component ==1, after actual bound call (I executions).

Check_roots158–182: ENSURE outputcount==expected once per24 list calls; for each of86 expected entries, scan exact roots then intervals in source order. A matched slot executes ENSURE(!visited) and VERIFY(!found), marks that exact slot, then final ENSURE(found). Successful uniqueness means258 conditional assertions across86 entries, independent of exact/interval split. Preserve original visited grow/fill and strict rational comparisons via real existing ToRational/source-wrapper conversion; do not turn matching into a Go map. These count invariants give base307 executions plus actual `R+3I` common per-output actions (since line121 ENSURE +two VERIFY per interval). Print-only refinements are not added to source assertion totals. Record R/I and total after the actual algorithms finish, not a fabricated preselected DRS split.

### Refinable/refine/conversion registered calls

- `tst_refinable`491–557: nine calls. First polynomial (x−1)(4x−11)(x−3) at(1,3),(2,3),(5/2,3); second (x−1)(5x−11)(x−3) at(1,3),(2,3),(3/2,3),(1,5/2),(3/2,5/2); third (x−1)(x−2)(x−3) at(1,3). Each executes exactly one conditional source assertion: true route VERIFY nonzero opposite endpoint signs, false route ENSURE EvalSign(a)==0. Nine executions total, with branch split determined by actual case2/3/4 probes. Direct finite fixtures separately exercise case1 and every return/update branch.
- `tst_refine`559–598: two calls, **zero ENSURE/VERIFY**. x^5−x−1,(1,2),prec20; x²−2,(1,2),prec200. Run actual precision loop; independent rational width/endpoints/root oracles validate without inventing registered assertion count.
- `tst_convert_q2bq`654–722: nine calls, **zero ENSURE/VERIFY**. Polynomial(x−1)(x−2)(x−3)(x−4); intervals(1/3,7/5),(1/2,7/5),(3/7,3/2),(0,3/2),(0,23/21),(7/2,5),(999/1000,1001/1000),(9999/10000,10001/10000),(39999/10000,40001/10000). Actual Boolean/c/d outputs and bracketing/root-sign checks use independent assertions, not claimed source counts. Include exact and inexact rational branches, source found_d publication, zero/negative bracketing and raw component aliases in additional bounded cases.
- There are **13 static ENSURE/VERIFY sites** in these selected registered consumers: five common helper sites, six check_roots sites and two conditional refinable sites. Counting executed routes is conditional as above. The 21 transform/half-root sites in the committed prior packet are unchanged and not recounted here.

## Independent model, ordering, lifetime and refusal coverage

Tests first before each corresponding implementation:

1. Whole roots and intervals independently evaluated with test-only rational arithmetic/exact Euclidean Sturm; monotonic disjoint interval/real-root uniqueness without using expected roots as outputs. Both actual DRS and Sturm public/core paths, defaultDRS and active Knuth selection. Outputs can have different exact/interval splits; do not demand an invented sorted order (source is right-first LIFO with positive then negative traversal). Record frozen source order fixtures for finite cases.
2. Degree70 full actual path, repeated-root/zero-root removal, constant/nonzero/zero input source conditional domain, zero-width frame/uint32 source limits and actual packed30-bit behavior. DRS frame parent sentinel/left-right order; SS half-open endpoint roots and macro prefix ordering.
3. Prefilled outputs demonstrate wrappers' reset order versus append-only lower cores; metadata refusal between lower/upper append before numeric Swap/Copy; default token publication before refused coefficient Copy. Existing output owner alias cases must be frozen at defined actual source paths, not globally rejected.
4. Source unsigned8-header extent versus actual physical stride for Binary24,DRS8,SS56, and host extent without giant allocations. Full old/new peak reservation, charged cold headers before allocation, same-owner compatible diagnostics, shallow cold owner rejection and zero/closed lifetime. Captured indexed vector borrows checked at actual read; growth/free versus retained initialized-slot Reset behavior.
5. Actual cancellation at DRS/checkpoints/Sturm stack/refinement width loops, keeping real output and mutated input prefixes. Loops without source checkpoint (isolating2refinable, q2bq, scalar refine) must not invent cooperation calls. Test immediate cells/Words graphs before fault disable/format/Del/Close; each failure accounts legitimate output/vector/frame roots while lexical cleanup releases companions. One successful sweep endpoint required; finite ordinal sweeps are not exhaustive all-input claims.
6. Actual destruction order: scoped binary-vector ascending Del; DRS polynomial stack pop reverse; SS frame pop lower→upper, remaining stack destructor ascending lower→upper; lexical seq/mid/currentLower/currentUpper/frame declaration/destruction order. Probe actual manager slot/Words reuse, not mirrored cleanup or only bytes after full owner close.
7. All normalization cases crossingzero/noncrossing, unknown-sign sentinel and supplied-sign debug checks; returnedfalse for zero root retains endpoints. Refinable four-case loops retain successive swaps; refine strict width and exact midpoint; rational conversion explicit Del-before-return and found_d retained prefix. Independent aliases and failed numerical-publication tokens use real cleanup guards, no strong transaction or scratch repair.
8. Full frozen native/checkeddebug/race/actualcheckedJS named RUN/PASS, vet/WASI/source-noGenerics, compiler escape and actual-service warm physical plateau measurements. Genuine per-call lexical backings are measured candidly, not cached away to forcezero allocations.

## Implementation and evidence before final freeze

Nine Go files constitute the selected packet: the four production and four test files listed above, plus the explicitly approved existing `mpnum/upolynomial_root_bounds.go` factoring. The public `RemoveZeroRoots` descriptor guards remain unchanged; its literal body now delegates to `removeZeroRootsState`, reused by lexical isolation. Ordinary, raw late-read prefix, and in-place alias regressions passed before factoring. There are no other existing-file edits in this packet.

Phase-one missing-API RED was observed against committed `e3a76a6`: `.cache/isolation-phase1/missing-api-red.log` (session 30817, exit 1), isolated tree `.cache/release/upoly-isolation-phase1-red-e3a76a6`. It lacked the actual binary vector, frames, DRS/Sturm and refinement methods. Initial selected source/arithmetic GREEN is `.cache/isolation-phase1/initial-production.log` (29442, exit 0); subsequent model/failure tests initially confirmed the implemented source behavior. They are not falsely labeled algorithm-defect RED. The pre-factoring raw-read fixture initially used an incorrect expected error; the actual initialized-extent `arena.ErrBounds` expectation was corrected and passed before factoring.

Actual registered execution counts from the full algorithms: the twelve datasets invoked twice yield R=26 exact roots/I=60 intervals, 306 ENSURE+206 VERIFY=512; full degree70 yields R=0/I=6, 7 ENSURE+12 VERIFY=19; nine refinable calls yield 4 ENSURE+5 VERIFY=9. Total **540 executions: 317 ENSURE+223 VERIFY**, from the thirteen static selected source sites. The two precision and nine q2bq registered calls are print-only source routines; their independent assertions add no claimed source counts. No expected root list supplies production results.

Nineteen named new groups cover all public dispatch/refinement/normalization methods, scalar packed/source-extent limits, actual cold-header refusal before host allocation, reserve logical-size growth, retained-capacity warm plateau, owning vector copies/failures, old RemoveZeroRoots behavior, defined aliased outputs, zero reset/publication prefix, actual SS pop/destruction order and ascending vector destruction. `.cache/isolation-phase1/deep-prefix-first.log` records real bound128 two-root DRS 131 refused ordinals and Sturm 19 refused ordinals, each followed by a successful endpoint; immediate live numeral/Words graphs are checked before disabling injection, formatting or cleanup. `.cache/isolation-phase1/refinement-fault-first.log` records precision refinement 13 refusal ordinals and q2bq 7, then success; these are finite workload sweeps, not exhaustive all-input coverage. Actual SS lower→upper pop and ascending remaining-frame destruction are tested through real numeric cell reuse, not mirrored cleanup. DRS reverse coefficient pop is translated literally and independently source-reviewed; there is no separate direct reuse test claiming its lexical cleanup order.

`.cache/isolation-phase1/publication-site-debug.log` and `publication-site-release.log` verify zero-output reset/append before lexical polynomial reservation refusal and q2bq equal output aliases: release publishes the source result, debug fails at the actual final strict endpoint assertion after publication. Failed-source raw mpz/mpbq locals receive explicit Go temporary cleanup on error, retaining caller mutations; this does not claim C++ allocation-exception cleanup equivalence.

A final source-route review corrected q2bq's two sign checks to actual underlying `IntegerManager.IsNeg`, matching cpp2839/2890 `m_manager.is_neg`, rather than the modular facade's extra normalization assertion. Ordinary Z fixtures do not distinguish this route; this correction is documented source-review evidence, not fabricated behavioral RED. Final validation includes coordinator commit `96e3058` with direct Power/Copy/Mul dependencies. All immediate failure/root checks pass without weakening: precision still13 refusals+success, q2bq now6 refusals+success rather than the historical7, reflecting the corrected sequential Copy dependency topology.

Final isolated native/checked-debug/race/actual JS/vet/WASI/escape and actual-service performance evidence is recorded below; source/no-generics final evidence is also terminal GREEN. Full algebraic cell/factorization/multivariate/printer and disabled alternative/TRACE source branches remain OPEN. Public owning Read and checked scalar owner/lifetime guards are explicit Go adaptations, not complete generic vector surfaces. All written artifacts and compiler caches/temp reside below golang; no generic, unsafe, production math/big/map, fake callback or mutable global manager is introduced.


## Frozen final gate evidence

Snapshot `.cache/release/upoly-isolation-final-96e3058` archives exact committed `96e3058` plus only the nine approved Go files. It excludes concurrent historical drafts. Final logs are `.cache/isolation-final/`; all compiler/cache/temp paths explicitly remain below golang. No source code changed during these final gates.

| Gate | Terminal result and log |
|---|---|
| New selected groups, `g3checked,g3debug` | 75715 exit0; `target-debug.log`, exactly19 RUN/19 PASS |
| Full module native | 55061 exit0; `native.log`; root3.280s, mpnum22.417s |
| Full module checked/debug | 50509 exit0; `debug.log`; root3.907s, mpnum22.608s |
| Checked/debug race, all polynomial regression groups | 33403 exit0; `race.log`, exactly77 RUN/77 PASS |
| Actual checked/debug JavaScript wasm execution | 72379 exit0; `js.log`, exactly77 RUN/77 PASS; real `.cache/js-exec` wrapper preserves subtree TMPDIR |
| Full checked/debug vet | 69126 exit0; `vet.log` |
| Full checked/debug WASIp1 wasm build | 78212 exit0; `wasi.log` |
| Baseline metadata/source/no-generics audit | 98747 exit0; `audit.log`; actual source `../../../..` from frozen snapshot, baseline metadata only (new selected method linkage remains coordinator-owned) |
| Compiler escape audit | 8807 exit0; `escape.log`; actual charged cold vector/header and typed dynamic payloads escape; no uncharged lexical numeric/frame metadata moved-to-heap finding |
| Actual-service warm benchmark | 41247 exit0; `perf.log`; separate unchanged-Go `.cache/release/upoly-isolation-perf-96e3058` |

One200ms amd64 native run of full default `IsolateRoots(x²−2)` after20 warm calls with actual utilReslimit, actual `_NO_OMP_` cooperation, actual RationalManager/BinaryManager/diagnostic RationalService: **6172ns/op,120B/op,6allocs/op**, same complete shared budget used7544B throughout every timed call; separately polynomial owner+three outputs2656B. Source lexical arrays are allocated/destroyed per call; they are not cached away to forcezero. Benchmark input literals are caller-owned, no comparison/speedup claim. Actual cleanup returns the complete shared budget to0. Existing vector warm reset/copy fixture separately proves0Goalloc and FailAfter0 retained-capacity behavior.

Immutable hashes (also machine-readable `.cache/isolation-final/inventory.json`):

| File | SHA256 |
|---|---|
| `mpnum/upolynomial_binary_vector.go` | `f7c8bde551ac0093f6f51d2d09e58bcfb580a4965951306cd1c86e4e9c7277de` |
| `mpnum/upolynomial_isolation.go` | `d04f21dfbfd992c572846df7439090594eb24873f3ef46c102f51724d56ef8eb` |
| `mpnum/upolynomial_sturm_isolation.go` | `40a0b5bd27fa1dd65ba6c6d739ac4c692b8d7fa4f76cc07a745fa317510f9839` |
| `mpnum/upolynomial_refinement.go` | `8964ec4ce20e6229e9a6cbf89e11d1761f9bcd1500e0f02b15d625fef6b483ed` |
| `mpnum/upolynomial_isolation_storage_test.go` | `6191a4b6d946701cacdba3a98f40d64a06be3ee78801f4b59dd143d90256249c` |
| `upolynomial_isolation_source_test.go` | `48889f75fef486db62eefdd70560a0254f1085cb2646ef166834d85cc302a3b1` |
| `upolynomial_refinement_source_test.go` | `cbdd819cddc97cc4f0c512b855480c34a6967f7f7b5795c4eadb266b837c0c06` |
| `upolynomial_isolation_model_test.go` | `124ece9fe56d06502553f5a16c69a5697d016061d4c9442c38d22fec746eb6a5` |
| `mpnum/upolynomial_root_bounds.go` | `bd44f4a215d24a657bdb06d1505da54a853e33d8135961f4f0e6b44e9921e477` |


## Coordinator final verification

Coordinator independently froze96e3058 plus the exact nine Go files and
integrated manifest/progress/documents at
.cache/review/upoly-isolation-final-96e3058. Full native80114 and checked/debug
50944 exited0 (mpnum21.612s/22.045s). Race10048 and actual JavaScript39579
exited0 with independently extracted exact77 RUN/PASS inventories. Vet78172,
full WASI build54956, root WASI test compile93036 and mpnum test compile49865
exited0. Integrated source-evidence/no-generics audit77029 exited0 at
2102units/677APIs,33ported/145partial/1924unported. All2102 recorded source/tool
SHA256 values were independently matched to the unchanged reference files.
Final nine Go hashes remained unchanged throughout gates; only this document
changed afterward. Commit/index names and hashes are checked against
.cache/isolation-parent-final-inventory.json. The full solver goal remains
open; all six touched source units retain partial status.
