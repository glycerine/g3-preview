# LP port implementation and test coverage

The full `src/util/lp` port is **in progress**. The inventory includes all 110
files, including ignored-by-default `.hpp` implementation bodies. See
[PORT_STATUS.json](PORT_STATUS.json) for every source file's current targets and
tests; `port_manifest.json` remains the repository-wide source ledger.

There are currently 84 partial source units and 26 unported units. The floating
primal and dual cores expose Solve, and outer LU primal/dual integration is implemented.
Remaining source diagnostics and default tableau, exact, dual and LAR variants
remain incomplete. Go coverage does not measure unported C++ code.

Latest full native module checkpoint was observed complete at03:52UTC on
October11 with exit status0 and all64packages passing: LP979.695s,
88.2% (16034/18173). Log/profile:
`/mnt/oldrog/tmp/g3-breakpoint-module-native.{log,cover}`.
The run started03:33UTC at0d330f87 and covers exact bound-step, Harris epsilon,
basis dispatch and breakpoint record/slope additions. It predates the collector
and traversal additions, which have separate focused checks. Coverage measures
existing Go code, not unported C++ functionality.

Known source limitation: post-pop cleanup declares an empty local changed-cost
set, while basic-value updates record changes in a different member set. With
`2*x0+3*x1-x2=0`, clamping `x0` from0 to2 makes `x2=4` exceed upper1.
Its independently correct default-sign infeasibility cost is -1, but cleanup
leaves stored cost0 and reduced costs unchanged. Native returns that source
state; debug detects the cost mismatch in the real reduced-cost assertion.
Both profiles are tested; this behavior is preserved for source fidelity.

Known source limitation: implied-bound explanation selects witnesses using the
normalized pivot target sign while forwarding original-row coefficients. With
`-2*x0-3*x1-x2=0`, `x0` in `[-3,7]`, and `x1` in `[2,11)`, actual LU row analysis
correctly derives `x2` in `(-47,0]`. The source explanation instead reconstructs
0 for the claimed lower endpoint -47 and -47 for the claimed upper endpoint0.
The discrepancy is preserved and tested; direct-row proof tests do not establish
general LU explanation correctness.

Known source limitation: term substitution passes a zero-based offset to a helper
that asserts it is a full term ID. Debug/checked therefore rejects nonempty term
lists after pivot calculation, before substitution or analysis. Native looks up
the literal offset; an ordinary external variable numbered zero can skip the
first unregistered term. Both behaviors are preserved and tested. Empty-term LU
and tableau per-row dispatch succeed in both profiles.

Known source limitation: the outer primal path flips upper-only coordinates but
retains `ColumnUpperBound` with zero upper value. Its regression reports source
“optimal” cost -3 where independent enumeration proves optimum2. This behavior is
preserved for fidelity; successful low/boxed-bound tests do not establish correctness
for that upper-only profile.

Known source limitation: feasibility-only search can stop with `FLOATING_POINT_ERROR`
after changed infeasibility costs make the stored reduced cost stale. For
`-8*x+s0=-11`, `-x+s1=-3`, with nonnegative variables, a feasible witness is
`x=3,s0=13,s1=0`; the retained source loop stops after its first pivot at
`x=1.375,s0=0,s1=-1.625`. Tests independently predict this drift classification.

Known source limitation: dual DSE invokes only the prepared triangular U/R
solve, omitting full-solve scaling/tail transforms. For the retained scaled
triangular fixture, updated beta1 is `0.0517578125`; the independently computed
new inverse-row squared norm is `0.2666015625`. Tests preserve and distinguish
these results. This does not establish general beta-norm accuracy.

Known source limitation: a failed dual pivot can forbid the only infeasible
row. `OneIteration` resets `Unstable` to `Feasible` before pricing, without
clearing that set; skipping every forbidden row can then report `Optimal`.
A real stale-LU/drop-refusal regression reaches this status while the basic
value remains `-5e-6`. Tests preserve the source transition and distinguish it
from the120 independently verified bounded optima.

| Component implemented so far | Verification | Statement coverage |
| --- | --- | --- |
| Exact residual diagnostics and value reconstruction | 180 independent exact matrix/RHS worlds;120 known-solution permuted-basis dense/sparse LU systems; all stdout writer prefixes and second product reevaluation; live aliases and descending publication; stale/stopped LU/matrix; eight pooled/direct physical campaigns and four older-caller profiles | Residual69/78 (88.5%) native,70/78 (89.7%) debug/checked; reconstruction72/75 (96.0%) both. Gaps are protected capture, valid string/equality/destruction and native-disabled assertion guards. Exact reconstruction uses one solve. Focused3.579s/5.916s; core-value regressions1.304s/2.015s, both vets, module compilation and ledger pass. Exact advancement/pivot/cost scheduling, full Solve and LAR orchestration remain open |
| Exact core value update/restore and indexed residual early return | 320 independent tableau/LU models with real nonunit solves; shuffled source traversal; six live aliases; three older-caller profiles; six pooled/direct physical campaigns with completed-scalar prefixes and exact cleanup census; source backup and zero/index rules | New file87/93 (93.5%) native,88/93 (94.6%) debug/checked. Five shared gaps are protected capture, valid zero-sign and temporary destruction guards; native erases one size assertion. Focused1.413s/2.015s; basis/tableau/core-LU regressions42.693s/49.136s, both vets, module compilation and source ledger pass. Full residual diagnostics/reconstruction, exact advancement/pivot scheduling and Solve remain open |
| Precise leaving-variable ratio selection | 480 independent feasible rational models;240 infeasible homogeneous tableau models; circular indexed scans, all types, sparse-row ties and RNG consumption; live basis/entering output aliases; eight pooled/direct physical campaigns including inline minimum; older caller and metadata/source assertion prefixes | New file127/146 (87.0%) native,131/146 (89.7%) debug/checked. Canonical equality remains separate from ordering; multiplier and ratio locals survive child calls. Focused4.927s/8.597s and basis/bound/breakpoint regressions42.720s/63.209s pass. Both vets, module compilation and ledger pass. Full exact advancement, pivot scheduling, Solve and LAR orchestration remain open |
| Exact sorted-breakpoint traversal and native/debug leaving wrapper | 480 independent rational/queue-order/slope models; strict epsilon threshold, zero-slope RNG and exhaustion; live output/record aliases; six pooled/direct physical campaigns; real stopped-queue and older-caller comparison profiles | Traversal79/91 (86.8%) native,76/91 (83.5%) debug/checked. Debug exact wrapper asserts before fill; native-only wrapper/fallback paths account for profile differences. Other gaps are protected capture/copy/sign/destruction and child invariants. All breakpoint regressions2.982s/4.268s and both vets pass. Full precise leaving ratio scan, advancement and Solve remain open |
| Exact breakpoint collection | 600 independent pair delta worlds;200 actual two-row LAR models; eight pooled/direct physical campaigns; three older-caller comparison profiles; equality, indexed zero coefficients, source row order and signed boxed entering spans | Collector147/170 (86.5%) native,148/170 (87.1%) debug/checked. Focused breakpoint regressions2.128s/3.057s and both LP vets pass. Remaining gaps are protected capture/copy/sign/destruction and metadata guards. Randomized traversal, full precise leaving selection and Solve remain open |
| Exact breakpoint records, priorities, clearing and slope updates | 180 independent signed pair/priority worlds;360 slope worlds with actual LAR headings and aliases; fourteen pooled/direct ownership campaigns; signed-minimum cases; two older comparisons; real stopped-queue clear prefix | Native79.9% (123/154), debug/checked84.4% (130/154). Gaps qualify valid copy/zero/capture/destruction and opposite-profile guards. Full collection, randomized traversal and precise leaving/advance/Solve remain open |
| Exact basis-column theta dispatch | 2400 independent pair worlds across five column types, stored feasibility and both directions; sixteen scalar/output aliases; eighteen copy/physical-expression campaigns; five older-caller comparison profiles and source guards | Native86.8% (59/68), debug/checked88.2% (60/68). Gaps protect capture/sign/index, final zero comparison and valid cleanup. Full precise leaving/ratio, breakpoints, advancement and Solve remain open |
| Exact upper-negative, boxed infeasibility and feasible no-check basis-column limits | 1200 independent exact-pair worlds with output/bound aliases; five scalar aliases; fourteen physical campaigns; five older-caller comparison-owner profiles; source sign/stopped/nil guards | Native87.3% (96/110), debug/checked94.5% (104/110). Remaining debug gaps protect valid cleanup, sign reads, capture and zero comparison/copy. Full basis-column dispatch, precise leaving/ratio, breakpoints and Solve remain open |
| Precise Harris bound epsilon and scalar zeroing | 240 independent rational/epsilon pair worlds; eight pooled/direct physical sign/epsilon campaigns; live output/bound aliases; two older-caller comparison-owner profiles and stopped/nil guards | Both profiles86.9% (86/99). Remaining gaps are protected capture, inline constants, zero comparison/negation and valid temporary/final cleanup. Full basis-column limits, exact ratio/leave selection and Solve remain open |
| Precise infeasibility limits for negative/positive general bounds and positive low bounds | 1440 independent exact-pair worlds; nine live aliases; six pooled/direct physical campaigns; source sign/stopped guards and three older-caller comparison owners | Native82.1% (92/112), debug/checked92.9% (104/112). Remaining debug gaps protect capture/valid cleanup and validated sign reads. Full basis-column theta/ratio/breakpoint/advance/Solve integration remains open |
| Exact bound flips, theta minimum and tableau-row leaving selection | 480 independent rational/epsilon bound/direction worlds;160 minimum and live output/limit/bound alias worlds;160 minimum-member leaving models with epsilon-only violations and retained set order; twelve pooled/direct physical campaigns through success; four older-comparison ownership profiles; negative-step source profiles and assignment/unlimited prefixes | New file88.2% native (135/153),89.5% debug/checked (137/153). Remaining gaps are protected capture/cleanup/value guards, debug-preempted metadata returns and the native-only default zero target. Focused19.816s/23.905s native/debug; pricing/tableau-cost ownership regressions22.069s/24.786s pass. Both vets, module compilation and ledger pass. Full precise theta/ratio/breakpoint/advance/Solve integration remains open |
| Exact primal eligibility, rational sorting and entering-column pricing | 300 independent rational/epsilon eligibility cases and50 breakpoint threshold/endpoint cases;160 deterministic scan/rotation/RNG worlds; source sort/count-heading checks; twelve pooled/direct physical campaigns through success; four older-caller comparison-owner profiles; strict-improvement budgets, ties and direction inversion | New file90.2% native (165/183),95.6% debug/checked (175/183); ten assertion-dependent statements are unreached natively. Eight debug gaps qualify protected private capture/index/value guards, valid negation/destruction and assertion-bound comparison errors. Focused5.491s/6.252s native/debug; both vets, module compilation and ledger pass. Source zero-count sort comparison is retained without inventing portable zero/equal-key placement. Full exact advancement/Solve and LAR integration remain open |
| Exact/floating LAR preparation and concrete LU extension orchestration | Eight profiles across four strategies and two breakpoint settings;100 independent sparse-count worlds;100 floating dimension worlds;36 exact/floating basis-extension oracles with widened thresholds; actual dense tails and singular factor disposal; six pooled/direct physical campaigns through success; actual pending queue comparison and source failure prefixes | Exact preparation88.7% (47/53), floating preparation100% (27/27), LU orchestration95.8% (46/48), both profiles. Remaining gaps are valid ordinary-vector cleanup/resize guards, earlier-ready-protected init-count errors and valid-owner factor Close errors. Focused0.229s native/0.345s debug/checked; both vets, module compilation and ledger pass. Full exact optimization, LAR signature/Solve and touched-row propagation remain open |
| Changed-bound row detection and exact value/cost orchestration | 160 independent two-row tableau value/cost oracles;100 real nonunit LU row/value models; all basic membership transitions in four strategies; six pooled/direct physical campaigns through success; retained marks/domains, no-change membership, child ownership and source failure prefixes | New file88.1% native (96/109),97.2% debug/checked (106/109); ten covered debug assertion statements are disabled natively. Three gaps are valid-vector clear/resize errors, source empty/IsOK invariant and valid-delta destruction guards. Main27.502s/32.302s native/debug, additional guards0.042s both. Both vets, source ledger and module compilation pass. Full LAR Solve, touched-row propagation and exact optimization remain open |
| Whole-model LAR scopes and actual row/equation/infeasibility-set validation | 160 independent nested tableau models with scoped variable/term bounds;100 varied nonunit LU restorations;100 undecided deferred-term scopes; actual older/scoped-row pivots preserve surviving tableau bases;160 pair-equation and160 lexicographic membership oracles;14 allocating native/18 debug pooled/direct pop/validator campaigns through success, four erased-assertion native profiles, two real pending-bound Push prefixes, forward tail destruction and older caller identity | Scopes68.2% native (73/107),88.8% debug/checked (95/107);22 covered assertion statements are disabled natively. Validators91.2% both (73/80), Push/AxIsCorrect100%. Remaining gaps are protected capture/cleanup/assertion/metadata guards and later diagnostic writer errors. Main85.001s native,132.157s debug/checked; added scope/model0.167s/0.220s. Both vets, module compilation and ledger pass. Full LAR Solve/propagation scheduling and exact tableau optimization remain open |
| Post-pop exact infeasibility cleanup and real cost/reduced-cost diagnostics | 160 independent two-nonbasic tableau models,100 actual nonunit LU models,1600 independent strict lexicographic cost worlds and160 rational matrix oracles; source snapshot/swap-pop order, cost publication and separate empty local/member set;16 pooled/direct failure campaigns through success, older caller comparison identity, exact split streams, missing-reference and stopped-child guards | Combined focused profiles: cleanup81.2% native (69/85),91.8% debug/checked (78/85); validators/diagnostics82.5% native (175/212),92.9% debug/checked (197/212). Cleanup has nine debug-only covered statements; validator profiles differ by24 debug-only covered statements and two native-only returns. Remaining gaps qualify protected capture/cleanup/value guards or prior assertion/child invariants. Main21.929s/24.345s native/debug; strengthened campaigns7.302s/8.238s and print guards0.057s/0.041s pass. Outer whole-model scopes and optimization remain open |
| Exact infeasibility costs and tableau reduced-cost updates | 1600 independent lexicographic worlds across all five types, including inverted intervals and nonbasic early return; 160 two-row rational cost oracles; source zero-delta/publication prefixes, alias rereads and metadata guards; eight pooled/direct failure campaigns through success and older caller comparison identity | New file90.4% native (103/114),93.0% debug/checked (106/114). Native omits three assertion returns; remaining eight gaps are protected capture guards, allocation-free cleanup and valid-value zero/copy guards. Focused17.040s native,18.144s debug/checked; both vets, port ledger and module compilation pass. Complete post-pop cleanup, outer scopes and tableau optimization remain open |
| Exact feasibility clamp, membership and LAR basic-value delta updates | 1600 independent lexicographic worlds with all five types and untouched/aliased delta;160 actual clamp/tableau equation models and100 nonunit LU oracles;18 pooled/direct campaigns through success including comparison-before-publication, final copy after delta publication, real LU children and older caller comparison identity | Combined focused profiles: feasibility84.1% native (90/107),95.3% debug (102/107); basic updates92.0% both (69/75); difference expression90% both (9/10), membership100%. Twelve feasibility statements are disabled debug checks; remaining gaps are protected guard/child/cleanup errors. Main101.204s native,116.045s debug; added comparison/metadata15.472s/16.478s. Basis/scope regressions, both vets and native module compilation pass. Full post-pop cleanup/cost recomputation, outer scopes and optimization remain open |
| LAR tableau row/column removal and debug basis validation | 160 independent surviving models with basic/nonbasic removed terms, signed costs and row swaps; literal RHS tail pop and unchanged outer state; reverse multi-column pop, both basis swap-pop paths and first-link selection; eight pooled/direct physical campaigns through success, actual embedded-core ownership, metadata and stopped prefixes | Removal78.1% native (168/215),88.4% debug/checked (190/215); validator7.1% native (2/28, debug body disabled),92.9% debug (26/28). Remaining gaps are disabled assertions and protected guard/cleanup errors. Focused scope/removal regressions33.958s native,35.791s debug/checked. Whole LAR scopes, feasibility cleanup and optimization remain open |
| Exact tableau normalization, column elimination and reduced-cost pivot | 160 independent exact matrix/RHS/cost oracles; source empty-target false ordering, absent/zero pivots and metadata/stopped-matrix guards; eight pooled/direct physical campaigns through success with core/child ownership, normalized RHS and retained pivot-cost prefixes | New file117/126 (92.9%) in both profiles; native/basis/scope regressions28.670s, debug/checked33.938s. Nine gaps are provenance/earlier-ready protected errors, allocation-free cleanup errors and invariant-protected assertions. Full tableau optimization and LAR removal/scopes remain open |
| Exact basis changes and tableau row transposition | 1600 independent basis exchanges and trace cancellation; 80 randomized matrices with 30 row swaps each; detached-nonbasic recovery and source guard/publication prefixes; pooled/direct physical row-copy failure campaigns through success and exact root census | Broader exact-core/basis/scope regressions47.835s native and76.311s debug/checked; new file98.4% native (62/63),100% debug/checked (63/63). Native erases one assertion return. Matrix owns stopped copies after the published basis swap. Tableau pivot/column removal and outer LAR scopes remain open |
| LAR core scope lifecycle | 1600 independent nested snapshots; precise bound/type/count restoration and descending nonbasis reconstruction; actual factorization destruction and separate snapshot ownership; surviving value/extents, actual tableau bindings and current-mode profiles; four pooled/direct restoration campaigns plus pending child Push prefixes, stopped and metadata guards | Core scopes67.5% native (83/123),87.0% debug/checked (107/123). Twenty-four assertion-related statements explain the profile difference; remaining16 are child/cleanup/invariant guards. Outer tableau removal, feasibility cleanup and whole-model scopes remain open |
| Implied-bound original-row explanations | 160 independent exact constraint-combination proofs, signed targets/raw coefficient order/identity, fixed/strict endpoints; aliases and zero-map publication/skip ordering; refusal prefixes and owner guards; pooled/direct consumer-owned division and separately stopped bound construction; actual negative-basis LU counterexample | Explanation96% native (48/50),98% debug/checked (49/50). Sign error guard is protected by matrix provenance; native erases witness assertion. Opposite-endpoint LU source discrepancy is preserved; full explanation correctness is not claimed |
| Exact pivot calculation and per-row calculate/substitute/analyze dispatch | 100 independent B*D basis worlds and inverse-row oracle; permuted bases/cancellation; actual model strict bounds; source offset assertion/native collision and fallback; six inverse/accumulation/scheduling physical campaigns plus two native substitution and two native tableau-copy campaigns; metadata/stopped-entry guards | Union of expanded/supplemental profiles: exact pivot91.4% native (53/58),93.1% debug (54/58); LAR pivot90.2% native (55/61),75.4% debug (46/61). Debug blocks13 statements at earlier source assertions; two remaining resize guards follow successful clearing. Full optimization/scopes/explanations remain open |
| Solver-bound tableau and prepared pivot row analysis | 160 independent signed-coefficient interval worlds; strictness/term identity, strongest-bound repeats, row-length cutoff boundary and assertion/construction ordering; nil/foreign providers; four pooled/direct physical failure campaigns with stopped entries and borrowed propagator ownership | Dispatch87.5% native (35/40),92.5% debug/checked (37/40). Gaps are child/cleanup errors and erased assertions. Pivot computation, term substitution and full scheduling remain open |
| Real LAR exact/floating model and variable/term/bound registration | Four strategy profiles with300 variable steps each;120 independent exact row/value/constant worlds; nested tableau and actual row propagation; deferred empty terms and source guard/publication profiles;12 pooled/direct physical failure campaigns through success; older caller comparison retained through destruction | Core74.2% (23/31) both; model74.7% native/79.0% debug (121/162,128/162); terms76.1%/81.2% (150/197,160/197); bound registration94.1% (64/68) both. Gaps mainly qualify invariant/child cleanup guards. Full LAR Solve/scopes/strategy migration and explanations remain open |
| Exact mpq/impq core/primal construction, bounds and actual LU entry points | 100 independent basis and known-solution row/column systems;320 exact lexicographic interval worlds, basic infeasibility membership and delta signs; source assertion/publication and header-borrow profiles;16 pooled/direct physical failure campaigns through success; older caller comparison survives destruction | Base76.0% native (149/196),82.1% debug/checked (161/196); precise primal76.8% (76/99) both. Gaps mainly qualify metadata/cleanup guards, erased assertions and small-integer epsilon-construction failures. Exact pricing/tableau/Solve and full LAR integration remain open |
| Exact row-bound analyzer and retained static entry point | 240 independent rational elimination worlds; strict endpoint exclusion, all five column types, single/multiple missing bounds, cursor/early-stop/persistent counters and literal RHS profiles; actual row/pivot iterators, foreign-manager and borrowed-provider guards; six pooled/direct physical refusal campaigns through success | Native80.9% (190/235), debug/checked84.7% (199/235). Gaps mostly child cleanup/guard errors and fresh-zero assertion failures. Full LAR row/tableau solver dispatch and explanations remain open |
| Exact LAR bound propagation and typed virtual providers | 1200 independent strongest-bound worlds with term aliases, strict ties and first-discovery evidence order; term constant restoration before filter, live model directories/getters, output and separately owned hook division; 14 pooled/direct physical refusal campaigns through success with map/copy/publication prefixes | Native/debug 89.5% (119/133). Remaining gaps mostly invariant-protected lookup/child-cleanup errors. Row analysis, full LAR model/explanation and SMT integration remain open |
| LAR exact bound dispatch, all five column-type updates and scoped witnesses | 300 independent exact open/closed interval worlds; source conflict/status/witness prefixes, unknown relations, invalid metadata/debug profiles, real scoped owners/aliased RHS and multi-column behavior; 30 pooled/direct physical refusal campaigns through success | Native 80.5% (293/364), debug/checked 85.7% (312/364); scoped witness specialization 100% (31/31). Remaining gaps mostly child-error/invariant guards; native omits debug assertion statements. Full registration, matrix/basis, whole-solver scopes, propagation and optimization remain open |
| Owning LAR variable/term/general constraints and bound witnesses | 120 independent exact copy/borrow worlds; all relations, duplicate/zero coefficients and empty vectors; seven physical refusal campaigns through success with partial element/evidence prefixes, source isolation and exact root census; stopped-term/foreign-manager guards; witness value copies and stream refusal prefixes | Native/debug constraints 92.8% (77/83); witnesses/stream 100% (10/10). Full LAR constraint insertion/scopes, bound updates and quick_xplain solving remain open |
| Exact LAR implied bounds and relation kinds | 160 independent big.Rat/evidence worlds, all six equality fields, copy isolation/self-assignment, strict direction/formatting, actual constructor/assignment allocation refusals and partial-owner cleanup | Native/debug 91.4% (53/58). Remaining guards qualify child cleanup errors or prior pending checks. Row analyzer, propagation integration and explanations remain open |
| Floating MPS dual creation and primal/dual comparison workflow | 120 mixed-cost parsed models in both minimum and maximum modes against independent vertex optima; stored bounds/ranges/fixed fixture; model/reader isolation, IsOK-independent creation and partial-owner publication guards | Reader93.7% native (357/381),94.5% debug (360/381). Both concrete solvers share actual base-model population. Exact/LAR creation and full original CLI remain open |
| Floating dual outer two-stage solve and solution reconstruction | 120 independent shifted/boxed/fixed/scaled/mixed-cost vertex optima; analytic free/equality models, infeasible/unbounded classification, both phase clock stops, real numerical/metadata/writer and owner publication tests | Native89.2% (116/130), debug90.0% (117/130). Gaps mostly propagate invariant-protected child errors or qualify assertion profiles. Source stage-two iteration-exhaustion dispatch omission is preserved; default tableau remains a missing-LU error. Exact integration and original drivers remain open |
| Floating dual core Solve, iteration and row pricing | 120 bounded covering LPs against independent exhaustive vertex minima;500 independent record-budget pricing worlds; RNG/strict limits/time/output/debug guards and actual numerical failure/forbidden-row transitions | Native77.9% (81/104), debug97.1% (101/104). Native excludes20 assertion-only statements;3 remaining guards are protected by earlier same-thread checks. Source forbidden-row Optimal discrepancy below is preserved. Outer dual stages/exact variants remain open |
| Floating dual actual pivot, beta update and rollback | 150 independent triangular-basis systems through actual LU replacement/refactor and rollback; full bound-flip pivot, real dual-feasibility/singular/numerical recovery and strict/IEEE/metadata/writer prefixes | Native84.3% (182/216), debug85.6% (185/216). Gaps include rare numerical snap/recovery, child errors and assertion-profile paths. Prepared source DSE beta discrepancy below is preserved; full pricing/iteration optimization remains open |
| Floating dual Harris ratio selection and bound-flip residual updates | 240 independent breakpoint-group models,120 Harris-window systems,120 flip/equation worlds, real nonunit-basis pivot-row/ratio/flip integration; strict/IEEE/source fallthrough and real child/metadata publication checks | Ratio92.9% native (260/280),95.0% debug (266/280); flips85.9% native (55/64),98.4% debug (63/64). Remaining gaps mostly qualify other-profile assertions or invariant-protected child propagation. Full dual pricing/pivot/beta/basis-update loop remains open |
| Floating dual feasibility initialization and leaving-row pricing | 160 independently reconstructed systems through actual LU;160 pricing worlds with reordered bases; fixed strict reduced-cost signs, endpoint short circuits, primal tolerance boundaries, IEEE beta division, stopping limits and diagnostic/error publication prefixes | Shared predicates/assignment88.9% native (56/63),95.2% debug (60/63); initialization/pricing88.2% native (45/51),100% debug (51/51). Full pricing/ratio/flip/pivot optimization loop and exact variants remain open |
| Floating dual two-stage preparation and real core construction/recomputation | 160 independently prepared models and100 independent dual-vector systems through actual LU; source column eligibility, cost/bound transformations, signed slack/artificial basis, restoration, IEEE/resize/backup and error publication checks | Preparation84.5% native (256/303),85.5% debug (259/303); core98.1% in each (53/54). Full dual optimization/stage dispatch and exact variants remain open; these tests do not claim dual optima |
| Floating MPS parser and primal construction | 120 independent parse-to-optimum models; stored fixed/bounds/ranges fixture; equality/greater-than integration; all bound types, signed/zero/IEEE ranges, fixed-field fallbacks, continuation/duplicate/marker publication, malformed records and real I/O refusals | Native93.9% (353/376), debug/checked94.7% (356/376). Native Go numeric grammar/writers are explicit boundaries. Exact/dual/LAR creation and complete upstream CLI remain open |
| Floating original zero-low constructor and initial-X validator | 2,000 independent predicate worlds; strict/IEEE/output and constructor publication checks; original test_lp_0/1 via actual constructor with empty upper vectors and independent optima under explicit LU | Native90.7% (49/54), debug/checked100% (54/54). Native gaps are erased assertions. Unchanged default strategy reaches source tableau assertion in debug / missing LU guard in release; complete original driver remains open |
| Floating feasibility-only entry point and nonbasic diagnostics | 160 independent two-row systems:58 feasible exits,102 independently predicted source drift stops; debug mode/status ordering, erased release check, all column types, real writer/stdout failures | Native85.4% (41/48), debug/checked95.8% (46/48). Exact/LAR integration remains open; source drift stops are fidelity evidence, not successful feasibility credit |
| Floating named solution feasibility validators | 200 independent dense row systems; all equality/inequality and bound tolerance edges, IEEE, name/metadata failures, output/publication prefixes and120 actual solved outer models | Native/debug97.4% (76/78); gaps are opposite-profile assertion/guard branches. Native classic-six-digit writer boundary is explicit; exact validators and complete parser/test-driver consumers remain open |
| Floating outer primal LU solve and reconstruction | 120 independent low/boxed-coordinate LP optima with fixed costs, shifts, scaling and source-equation checks; infeasibility, constructor/solve/output publication, value/cost/Close ownership and source upper-only counterexample | Native/debug85.7% (54/63). Gaps propagate child errors under invalid metadata. Upper-only source type/flip inconsistency is explicitly preserved, not credited as correct optimization; default tableau/exact variants, diagnostics and validators remain open |
| Floating stage-one solve and iteration/statistics control | Two original numerical fixtures and100 independent bounded standard-form LPs compared with exhaustive dense vertex enumeration; feasible-only/infeasible/unbounded/cancellation/limit exits, seeded entering budgets, output/clock failures, IEEE and mutation guards | Solve72.6% native (61/84),78.6% debug (66/84); statistics78.9% (56/71),94.4% (67/71). Gaps include rare numerical rechecks, qualified child errors and other-profile assertions. Default floating/tableau profile, exact/dual/LAR solvers and outer diagnostics/validators remain open |
| Floating primal pivot comparison and advance orchestration | 180 independent ray/basis worlds through sparse/forced-dense LU; bound flips, feasibility-cost transitions, degenerate pivots, actual stale-LU recovery, singular refusal/rollback, drift and two-observation status transitions; mutation/output guards | Native82.0% (137/167), debug/checked90.4% (151/167). Gaps qualify child/index errors or opposite-profile assertions. Floating solve/iteration/resource/statistics loop is implemented; exact/tableau advance remains open |
| Floating primal reduced-cost refresh and steepest-edge updates | 160 independent basis-coordinate worlds with sparse/forced-dense LU, independent transposed beta solves, new-basis norms and reduced costs; drift/profitability/tolerance classification, periodic seeded reset, infeasibility-cost transitions, fixed/duplicate/IEEE and mutation guards | Native94.6% (105/111), debug/checked100% (111/111). Native gaps are erased source assertions. Floating pivot orchestration is implemented; floating optimization is implemented; exact/tableau paths remain open |
| Floating Harris and breakpoint leaving/step selection | 500 independent Harris bound worlds and600 signed breakpoint crossing worlds; independent hinge slope jumps, source double first-crossing behavior, queue/order/publication guards, bound flips and full floating dispatch | Harris98.5% native/100% debug; breakpoint/dispatch93.3% native (112/120),95.8% debug (115/120). Remaining gaps propagate queue errors behind valid metadata or belong to the other assertion profile. Floating optimization is implemented; exact/tableau paths remain open |
| Floating primal stage-one initialization and entering pricing | 150 independent basic/nonbasic bound worlds (3,300 column decisions); real feasibility/objective cost lifecycle and seeded norms, 100 independent rank worlds with45 choices each, budget/rotation/counter rules, IEEE and publication/error prefixes; actual120 prepared-primal run initializations | Initialization95.8% native (158/165),98.2% debug (162/165); pricing89.9% (89/99),100% (99/99). Gaps guard inherited valid boxed bounds/int-set invariants or belong to debug-only bound assertions. Harris and floating breakpoint leaving/step selection are implemented; norm/price updates are implemented; optimization remains open |
| Floating primal values, pivot rows and numerical basis/X updates | 100 independent known-solution reconstruction worlds; 120 dense independent B*d/nullspace worlds through sparse/forced-dense LU, real pivot rows, Suhl and full refactors; singular feasible-only/rollback, nonzero-step restoration, stale factors, drop-based refusal, output/borrowed-input prefixes, literal correction sign and IEEE residuals | Values95.2% native (118/124),96.0% debug (119/124); numerical updates87.8% (79/90),91.1% (82/90); pivot rows96.2% (25/26),100% (26/26). Gaps are other-profile assertions, invariant-protected child errors and final non-OK refactor after restored feasible X. Full optimization and exact core remain open |
| Floating basis changes, reduced costs and entering-column preparation | 16,000 independent basis exchanges/restorations and trace steps; 64 known-dual matrix worlds with six refactor/entering-column rounds each, sparse/forced-dense LU, source publication prefixes, stopped children, descending/duplicate arithmetic and IEEE | Native94.8% (92/97), debug95.9% (93/97). Gaps are invariant-protected row guards and false source is_OK stub assertions; native also excludes the debug entering assertion. Numerical pivot/X update and optimization remain open |
| Floating core construction, basis headings and feasibility | 160 independent reordered diagonal bases with actual optional LU and replacement; 41,665 exact dyadic rational tolerance comparisons, all column types/IEEE/strict boundaries, cached infeasibility updates, debug/release publication guards; actual 120-world prepared-primal integration | Native88.8% (135/152), debug97.4% (148/152). Remaining debug gaps guard enormous int-set extents, valid heading invariants and other-profile short-basis domains. Exact core, pivots, reduced costs and optimization remain open |
| Floating primal initial residual basis and bound/cost preparation | 120 independent [A,D]X=B worlds; real sparse/forced-dense LU solves in both orientations, structural costs/bounds, negative initial values, unsigned wrap, resize/IEEE/empty-program and publication/output-error cases | 96.2% (101/105) native and debug. Gaps propagate real host-clock/invariant-protected child errors or guard enormous negative converted extents. Actual core construction/optimization and exact preparation remain open |
| Floating solver presolve, pinning and fixed/zero elimination | 100 independent exact box-corner worlds and1500 relation/RHS decisions; strict directional tolerance/IEEE/unknown/zero bounds, pin and RHS publication prefixes, deferred erasure, actual cleanup pass-count semantics, writer failures and cleanup-to-scale integration | Native97.8% (133/136), debug98.5% (134/136). Native gaps are debug-only assertion/error propagation; debug gaps guard null pointers after successful source metadata lookup. Exact presolve and complete drivers remain open |
| Floating common solver preparation and retained column metadata | 120 independent original-system worlds across shifted/flipped/boxed/free columns; external/core maps, scaled solutions/objectives, reconstruction, counters, stale name aliases, bound tolerance/IEEE and source error/publication prefixes | Solver96.6% native (196/203),98.0% debug (199/203); retained column metadata89.6% (43/48),100% (48/48). Gaps include other-profile assertions, real host-clock failure and invariant-protected reconstruction/false-balance propagation. Primal/dual optimization and exact preparation remain open |
| Floating ratio/geometric and power-of-two scaler | 160 independent known-solution matrix worlds; diagonal-factor oracles, strict thresholds, both ordering branches, original-ratio and reps+1 fixtures, borrowed inputs, RHS/scale publication failures, zero/NaN/infinity and source log stop | Native86.9% (226/260), debug88.1% (229/260). Gaps propagate invariant-protected child errors, belong to the other assertion profile or lie beyond the source unimplemented balance stop. Precise explicit scale/full solver integration remain open |
| Prepared X, dense T and indexed T LU solves for all three retained types | 40 independent U*R*x worlds with sparse/forced-dense factors; pair coordinates, descending append, settings/tiny/IEEE cases, U/R error prefixes, stopped/foreign caller ownership; 12 physical refusal campaigns (128 native /206 debug) | Exact100% (46/46), floating94.6% (35/37) in both profiles. Floating gaps guard bounds established by successful U/R operations |
| Three retained factorization replacement functions and remaining upstream LU numerical scenarios | 12 independent reordered-basis worlds with sparse/forced-dense owners; nil/replacement/invalid constructor lifetimes; four pooled/direct failure campaigns (292 native /532 debug refusals), successful re-entry and146/260 actual consumed-scratch guards; original larger/holes/random fixtures and all replacements at three density policies, independent factorization and dense/indexed/refined row plus column solutions | Replacement88.2% (15/17) in both profiles; uncovered returns propagate exact-owner cleanup errors behind valid ownership invariants. All four upstream mathematical LU scenarios now pass; diagnostic printing/CLI and full solver integration stay open |
| LU dimension extension for retained floating/rational-pair consumers | 24 independent basis worlds, two growth rounds each; affected-column maps, dimensions, tail factorization identity, floating/pair column and dense/indexed row solutions; original small-LU four updates plus two-row extension, real dense-tail query, degeneration and guards; two pooled/direct fault campaigns (22 native / 40 debug refusals), real stopped U/Q/R children and named-owner census | Floating91.4% (53/58) native /94.8% (55/58) debug; exact93.4% (57/61) /98.4% (60/61). Gaps are the other assertion profile, invariant-protected source column/child bounds, and floating child propagation. Scalar mpq/mpq extension has no retained caller and is excluded |
| Dense/indexed LU row error-check and refinement | 64 independent basis worlds across all three types, strict120/30 thresholds and every floating routing branch; one-correction approximate factors, 1,200 exact residual/union worlds, 96 post-Suhl replacements, RHS-only residual/IEEE/tolerance/error prefixes; 12 pooled/direct allocation campaigns (1,196 native / 2,304 debug refusals), caller-owned stopped receipts | Floating90.9% (110/121), precise routing100% (27/27) in both profiles; remaining floating returns propagate child errors behind successful marker/extent invariants. Full solver integration remains open |
| Exact rational/pair Suhl LU column updates | 96 independent exact basis worlds with eight updates each (768 total); scalar/pair column and dense/indexed row solutions, tiny/ignored-pivot/cancellation/hidden-index cases, 21-row sparse column integration; 14 pooled/direct refusal campaigns through success (2,506 native / 5,014 debug), partial row/diagonal tails and older caller ownership | 87.7% native (265/302), 88.4% debug/checked (267/302); both public updates 94.1%. Gaps propagate valid-scalar cleanup/child errors, protect invariants or belong to the other assertion profile; full upstream drivers remain open |
| Floating Suhl LU column updates | 64 independent basis worlds with 12 repeated nonsingular updates each through sparse/forced-dense initial factors; original four test_small_lu replacements; 241-row indexed column integration; degeneration, strict tolerance/IEEE/underflow/assertion and reached publication/error prefixes | 91.0% native (141/155), 93.5% debug/checked (145/155); remaining returns propagate invariant-protected child/bounds errors or belong to the other assertion profile. Full driver remains open |
| LU entering-column preparation and indexed/dense column solves | 48 independent original-matrix/known-solution worlds across all three types; strict10/120 boundaries, actual sparse and dense routes, pair faster caller, alias/hidden-data/drop/order cases, first upstream prepare stage; 28 pooled/direct fault campaigns (3,765 native / 7,354 debug refusals), source/caller ownership and reached input prefixes | After prepared-stage extraction: exact column file96.1% (74/77), floating100% (23/23) both profiles; all public column methods100%. Remaining returns guard invariant-protected resize/index bounds or propagate valid-child/zero-predicate errors; source matrix/source-template functionality is not inferred from Go coverage |
| Exact scalar/pair initial LU factorization and solves | 56 independent rational matrix worlds run sparse and forced-dense; exact factorization identity and scalar/pair column plus dense/indexed row solutions; 32 pooled/direct refusal campaigns through first success (4,595 native / 8,718 debug refusals), borrowed-source and named-cell census, failed eta/dense publication, precise tiny values, constructor/owner guards and older comparison ownership | Initial-factorization coverage snapshot before Suhl ownership additions: shared coefficient orchestration 83.9% native (115/137), 87.6% debug/checked (120/137); typed owners 87.5% (42/48), current full-native exact solve file100% (74/74); historical pre-prepared-extraction debug solve snapshot100% (82/82). Remaining initial statements propagate cleanup/child errors or belong to the other assertion profile; full upstream driver remains open |
| Floating initial sparse/dense LU factorization and solves | 180 independent rectangular-source/basis worlds each run sparse and forced-dense; verify tail-product identity and known column/dense-indexed row solutions; two upstream initial-basis fixtures, singular dense prefix, malformed/stopped/closed inputs, strict tiny-pivot/drop, density-wrap and ignored unit-shortening boundaries | Current core82.3% native (107/130),85.4% debug/checked (111/130); current full-native solve orchestration83.8% (31/37); historical pre-prepared-extraction debug80.9% (38/47). Remaining paths propagate child errors behind valid construction invariants or belong to the other assertion profile; full upstream driver remains open |
| Exact dense scalar/pair tail applications, conjugation and diagnostics | 120 independently factored rational worlds verify five application routes after conjugation; duplicate/hidden/trailing/growth/cancellation behavior, 22 genuine-fraction allocation campaigns, actual pair X-before-Y and index-before-copy failures, stopped caller and debug-view refusal | Scalar applications 190/230 (82.6%) and pair 88/105 (83.8%) in each profile; debug L check 24/26 (92.3%), view 20/21 (95.2%); gaps propagate cleanup/child errors or guard invariant-protected mappings; native L check returns true without debug traversal |
| Exact dense-block sparse parent publication | 120 independent factorization worlds verify sparse upper and dense lower matrices, reciprocal links and row maxima; staged missing-diagonal/common/cancellation/fill tests, physical refusal campaigns and older comparison ownership | 91.5% (108/118) in native and debug/checked; gaps propagate cleanup/capture errors or guard invariant-protected predicates/mappings |
| Exact dense-block construction and pivot stages | 120 independent rational elimination worlds checked at every pivot; pooled/direct actual allocation refusals through success, partial L publication, borrowed parent and older comparison ownership; precise tiny coefficients and reached bounds/assertion prefixes | 82.8% native (207/250), 87.2% debug (218/250); gaps propagate child/cleanup errors, guard invariant-protected offsets or belong to the other profile; remaining solver integration and diagnostics remain open |
| `int_set.h` | Original `test_int_set`; dense swap order, resizing, reached invalid-domain clear prefix, 10,000-step independent map oracle, output/error cases | 100% |
| Four live `stacked_value.h` scalar types | Original `test_stacked_unsigned`; signed, status and strategy restoration, nested/excess/zero pops, unsigned wrap | 100% |
| Three concrete `stacked_vector.h` element types | Original `test_stacked_vector` with strengthened assertions; 12,000 operations against independent snapshots, repeated replacements, popped tails, baseline journals and checked scope/index guards | 100% |
| Exact `numeric_pair<mpq>` arithmetic | 400 arbitrary-precision worlds against `math/big.Rat`, 625 lexical comparisons, aliases, missing-cell error prefixes and strengthened ten-term source accumulation | 100% |
| Floating/unsigned indexed vectors | 12,000 operations against independent sparse maps; source equality/self-assignment/debug-prefix quirks, IEEE/tolerance edges and public-index guards | 99.0% per profile; only the opposite-profile assertion path is uncovered |
| Exact rational/pair indexed vectors | 5,000 operations against independent rational maps; exact values below floating tolerance; copy/assignment/self-assignment and bounds; real set/assign/clone/add/drop allocation refusals with named-root and direct-object census, including genuine fractional scalar/pair normalization failures with nested integer GCD/division receipts | 90.0%; remaining statements are child error propagation and the other profile assertion path |
| Exact `stacked_vector<numeric_pair<mpq>>` | 1,600 operations against independent snapshots; big-value ownership; real append/replace/restore allocation refusals with live-cell/raw-block census; older/replacement pending-root identity | 92.8%; uncovered return paths propagate cleanup/capture errors |
| Shared permutation index algorithms and double/double applications | Original `--test_perm`; 2,304 distinct compositions and 288 transpositions against independent dense products; all 120 size-five permutations across six dense/four sparse directions; literal alias and shrunk-domain prefixes | Index algorithms 91.2% native / 93.0% debug; floating applications 87.7% / 92.6% |
| Exact rational/rational and rational/pair permutation applications | All 24 size-four permutations across six dense/four retained rational sparse directions against independent exact matrix products; ordered indices including zeros; 1000-digit allocation refusals through first success; named-cell/direct-object census; stopped destination ownership and every pending mutation guard | 82.8% native / 91.5% debug; native excludes debug dense copies and exact comparison |
| Native int/unsigned/double priority queues and int/unsigned pair queues | Original `-tbq` and strengthened pair smoke scenario; 12,000 independent map operations per type (60,000 total); exact tie/LIFO order, source mutation prefixes, extent wrap, NaN/infinity and diagnostic gaps | 85.6% scalar algorithms; 92.2% pair wrappers; index helpers 87.7/89.5% heap, 90.6/88.7% pair |
| Exact rational/pair priority queues and strict pair relations | 5,000 operations per type against independent exact ordering; retained-priority/copy ownership, ties, exact fractions; actual new/increase/decrease/dequeue reservation campaigns; by-value argument stops, named-cell/direct-object census and older comparison identity | 86.2% queue code, 84.6% call receipts in each profile; pair arithmetic/relations file remains 100% |
| Floating sparse coefficient lists, column/row eta matrices and actual tail interface | 2,000 independent dyadic matrix worlds across four dense/sparse directions; all 24 size-four permutation conjugations; mixed tail chains; strict configurable/fixed tolerance edges, exact index order, duplicates, signed zeros, IEEE values and reached bounds/conjugation prefixes | Eta and row eta 98.9% native / 100% debug; sparse coefficients 91.7% / 100%; interface binding 100% |
| Exact rational sparse coefficient lists and division ownership | 120 arbitrary-precision list worlds against `math/big.Rat`; first-match lookup, duplicate/zero entries, tiny divisors and live divisor aliasing; real append/lookup/positive and negative divide/alias-divide refusal campaigns through first success; named-cell/direct-object census and older raw invocation identity | Coefficient storage 87.1% native / 88.7% debug; exact matrix call receipts 86.2%; remaining statements propagate child/cleanup errors or the opposite-profile assertion |
| Exact rational/rational and rational/pair column eta applications and tail bindings | 240 independent rational matrix worlds; all 24 size-four conjugations and element views; mixed permutation/eta dense/sparse chains; source duplicate/self-term/index and bounds prefixes; real coefficient, diagonal, small/large-pivot and zero-destination refusal campaigns; stopped-destination and older raw ownership | Eta 89.4% native / 89.8% debug; concrete rational expression helpers 90%; exact tail bindings 100%. Gaps include child/cleanup errors and the unreachable precise branch that drops a nonzero pivot after division by a valid finite diagonal |
| Exact rational/rational and rational/pair row eta applications | 240 independent rational matrix worlds across dense/sparse directions and pair left; all 24 size-four conjugations/element views; mixed permutation/column/row tail chains; source snapshot/live-reference, duplicate/self-term and descending conjugation prefixes; actual reservation campaigns through numerical success, including index-before-result-copy stops; stopped-destination/older-call ownership | 91.4% native / 91.8% debug; remaining statements propagate child/cleanup errors or the opposite-profile assertion |
| Floating static matrix storage, queries, row pivots and dimension scopes | 12,000 independent map mutations; 2,000 independent row pivots; reciprocal links, fill/cancellation, IEEE underflow, duplicates, literal offset/scoping/debug quirks and valid mutation prefixes | 90.9% native / 93.5% debug; complete solver integration remains open |
| Exact rational static coefficient storage and row elimination | 1,800 independent mutations; 240 exact matrix worlds with shuffled term order and multiple cancellations; tiny exact coefficients, live aliases, literal mutation prefixes; eight actual reservation campaigns through first success with named-root/direct-object census and older invocation ownership | 90.8% native / 92.2% debug; exact term-with-basis fill is implemented; complete solver integration remains open |
| Exact rational/pair static products, column extraction and absolute extrema | 160 independent arbitrary-precision matrix worlds across scalar/pair rows and scalar columns; output/live-alpha aliases, literal sparse order, retained dense slots, duplicates and bounds prefixes; ten physical reservation campaigns with named-root/direct-object census and older comparison identity | 91.6% native / 91.6% debug; remaining statements are child/cleanup errors and profile-only guards |
| Exact static dimension scopes, row transposition and diagnostics | 1,400 operations against independent dimension/coordinate maps; all three owning transpose copy stages under real numeric refusal; self/empty transposes, scope prefixes, source diagnostic gaps and writer errors | 88.7% native / 94.0% debug; release omits the final source diagnostic block, remaining gaps propagate child errors or are tautological checks |
| Retained rational row/column/indexed/pivot-row iterators | 4,800 independent mixed cursor operations; literal publication/end/size quirks, four physical copy-failure campaigns, borrowed live values, index-only reservation exclusion and older comparison ownership | 94.8% in each profile; remaining statements propagate child/cleanup errors |
| Exact LAR term ownership, arithmetic and substitution | 1,600 independent map/order operations; explicit zero/cancellation/reinsertion and live pivot prefixes; eight physical refusal campaigns, partial construction/export and borrowed-owner census | 90.0% in each profile; remaining statements propagate child/cleanup errors |
| Exact term-with-basis iterator | Basis-first and coefficient copy publication; live values, cursor failure, stopped-owner guards and actual copy receipt census | 96.3% in each profile |
| Exact static last-row fill with basis elimination | 240 independent rational elimination worlds; literal zero/duplicate/stale workspace behavior; three physical refusal campaigns through success, owner-specific publication prefixes and older comparison identity | 85.0% in each profile; remaining guards include unreachable child Reset/bounds failures for the retained concrete term iterator |
| Floating sparse basis construction and permutation queries | Standalone source test_init_U fixture; 600 independent sparse worlds, duplicate/zero/IEEE values, reciprocal offsets, 16 permutation rounds per world and malformed-input prefix checks | 100% in each focused profile; complete solver integration remains open |
| Exact sparse basis coefficient storage and permutation queries | 120 independent arbitrary-precision sparse worlds; 12 permutation rounds each, duplicate/zero/first-match storage, independent source/row/column owning-cell census; two real allocation-refusal campaigns through success, column-before-row stops and older comparison identity | 94.2% in both focused profiles; remaining statements propagate cleanup errors or guard fresh constructor state |
| Floating sparse mutation, row elimination, maximum ordering and division | 6,000 independent map mutations and 800 dense pivot worlds; reciprocal links, shuffled permutations, exact cancellation/fill, strict tolerances, live divisor, IEEE values, duplicate publication/removal and reached bounds prefixes | 94.4% native / 95.6% debug/checked focused; remaining gaps propagate bounds errors on malformed permutation metadata or belong to the opposite assertion profile |
| Exact sparse mutation, row elimination and live-reference division | 1,200 independent mutations; 160 independent rational pivot worlds; row/column divisor aliases, tiny exact values, reciprocal ownership census, eleven pooled/direct physical failure campaigns through success and older comparison identity | 90.0% native / 90.9% debug/checked focused; uncovered returns propagate cleanup/valid-scalar errors, guard malformed metadata, or belong to the opposite assertion profile |
| Exact maximum ordering and floating/exact pivot preparation/selection | 120 independent rational/float queue worlds; stable maxima, minimum admissible scores, inverse permutations, strict thresholds, stale/inactive candidates, exhaustion recovery, unsigned wrap, six physical failure campaigns and older comparison identity | Exact 89.9% native / 90.4% debug; float 88.2% / 89.7%; remaining statements propagate child errors, protect malformed metadata or are opposite-profile guards |
| Floating/exact active shortening and eta-driven sparse elimination | 160 independent incidence/queue worlds with permutations and eta refresh; 100 exact dense elimination worlds; early zero-column/empty-target exits, stale candidates, inactive eta reinsertions, signed score profiles, bounds publication prefixes and pooled/direct physical failure campaigns with borrowed eta census | Exact 94.2% native / 96.1% debug; float 93.6% / 95.7%; remaining statements propagate queue errors under malformed queue state or belong to the opposite assertion profile |
| Sparse-column eta construction and repeated-pivot integration | 180 independently ordered/permuted column worlds; literal duplicate/unit/zero/tiny/IEEE and publication prefixes; four physical failure campaigns with matrix/eta ownership census; 120 repeated pivot worlds checked against exact dense residuals and permutation-expansion determinants | Exact 94.1% and floating 96.7% in each profile; remaining statements propagate unreachable Append assertion or child/capture/cleanup errors |
| Floating/exact sparse column replacement | 1,600 independent coordinate-map replacements with permutations, row maxima and reciprocal links; literal duplicate/unindexed/tolerance/IEEE inputs; ten physical failure campaigns, borrowed-owner guards and source publication prefixes | Exact 88.3%, floating 95.2% in each profile; remaining statements propagate child/cleanup errors or assert the nonempty row guaranteed by successful insertion |
| Floating/exact physical-row restoration from logical indexed workspace | 800 independent coordinate-map/permutation worlds; reciprocal links and maximum ordering; duplicates, hidden values, tiny exact coefficients, strict tolerance/IEEE and bounds prefixes; six pooled/direct physical failure campaigns with row-before-column and cleared-index publication plus stopped caller ownership | Exact 90.3% in both profiles; float 93.0% native / 97.7% debug/checked; gaps propagate child/cleanup errors or belong to the other assertion profile |
| Dense unit-upper triangular scalar/pair solves and floating residual correction | 220 independent exact RHS/solution/residual worlds with permutations, tiny/zero terms and trailing slots; literal diagonal/duplicate/IEEE and publication prefixes; six physical refusal campaigns with required pair X-before-Y stop and caller ownership census | Exact 92.8%, floating 97.2% in each profile; remaining statements propagate scalar/cleanup errors or are invariant-protected correction bounds |
| Indexed graph traversal and floating/exact scalar triangular solves | 240 independent nilpotent inverse-polynomial worlds with sparse reachability/index membership; literal postorder/output prefixes, flag lifetimes, zero/tiny/unindexed inputs; four physical failure campaigns and borrowed-owner guards | Graphs 94.4% native / 97.2% debug each; exact solve 89.3%, floating 93.4% in both profiles; remaining statements propagate child/cleanup errors, protect index rebuild bounds or belong to the opposite assertion profile |
| Floating indexed residual correction/refinement | 240 independent exact inverse-polynomial and dense-residual worlds with shuffled permutations; strict density boundary, sparse/dense tolerance divergence, hidden/trailing data, fixed cleanup/order/IEEE and reached error prefixes | 91.4% native and debug/checked (32/35); remaining returns guard invariant-protected correction errors |
| Sparse identity extension, stored-entry count and LU permutation composition | 120 worlds with eight repeated dense-oracle permutation/growth rounds; retained workspace/queue/counters and reciprocal links; pooled/direct inline-one growth under disabled numerical reservations; native/debug shrink, real stopped-owner and publication/error prefixes | Floating 96.6% native / 100% debug; exact 91.3% / 93.5%; remaining returns propagate inline-one/capture/cleanup errors or belong to the opposite assertion profile |
| LU one-coordinate diagonal tails for floating/rational/pair paths | 240 independent matrix/conjugation worlds; exact tiny values, strict tolerance/IEEE/index quirks, constructor ownership, fourteen pooled/direct reservation campaigns, stopped caller receipts and actual pair X-before-Y failure; cached debug view | Floating 97.3% native / 100% debug; exact 86.1% / 88.9%; debug view 100%; gaps propagate child/cleanup/capture errors, test an unreachable exact-zero quotient or belong to the other profile |
| Floating LU dense-block factorization, parent update and tail applications | 180 independent exact elementary-elimination worlds checking E and E*A*P, column swaps, sparse parent values/backlinks and all four tail routes after conjugation; literal tolerance/IEEE/duplicate/trailing/growth cases and reached publication errors | Core 90.5% native / 93.2% debug; applications 81.9% / 83.2%; debug L check 91.7%, view 93.8%; remaining gaps are metadata/child propagation and assertion-profile paths |
| Settings defaults, stats, float tolerance families, random state and resource limits | All numerical/strategy defaults, source RNG golden vectors, mutable thresholds, strict CPU-time boundary, borrowed limit/error propagation and stats reset | 96.3%; uncovered constructor return propagates a real host-clock start error |
| Settings enum values and three formatting/parsing functions | All column/status cases, source missing cases, parser/output asymmetry, native/debug assertion profiles | 100% |
| Existing LP parameter collector | All five exact keys/kinds/defaults/descriptions/module names; each reached conflicting-kind error prefix; native backing under disabled Runtime reservations | 100% |

The complete native module suite passed at20:03 UTC on2026-10-10, including
all current floating primal preparation, core initialization/iteration/value/pivot/
numerical updates and primal initialization/pricing. LP passed in332.181s with
**90.0%** statement coverage (**10056/11174 existing Go statements**). Other
unchanged packages reused valid Go test cache entries. Authoritative log/profile:
`/mnt/oldrog/tmp/g3-primal-core-module-native.log` and
`/mnt/oldrog/tmp/g3-primal-core-module-native.cover`. This replaces the historical
19:01 snapshot; it does not measure unported C++ algorithms.

Current lifecycle/upstream numerical checks pass0.812s native/1.294s debug.
Same-manager re-entry after a source Add/SubMul allocation refusal preserves
missing scratch and returns the real dead-slot guard; owning cleanup is not a
source-state repair. Existing numeral re-entry tests establish that contract.

Current extension and floating/exact initial-LU regression checks pass2.419s native
and2.911s combined debug/checked. Both LP vets pass. These focused results
complement the current complete native module run above.

Focused refinement allocation campaigns pass4.735s native and9.284s combined
`g3debug g3checked`; final strengthened property/guard checks pass1.324s/1.495s.
Both LP vets and source inventory/evidence/no-generics checks pass. The component
table combines matching current focused coverage blocks where necessary; no
allocation campaigns are rerun solely to collect combined reports. The last
complete debug/checked LP coverage snapshot was91.7% before sparse basis storage;
subsequent components pass focused debug/checked tests as recorded above. The
full native Go coverage percentage does not measure unported source algorithms.

Coverage is measured with native and combined `g3debug g3checked` builds. It
applies to the current Go code only. Debug assertions use the existing explicit
Go error profile; process-global debugger notification/stream binding is not
claimed. Go slices and maps replace STL/container internals. Printing uses Go
`io.Writer`, with file ownership and flushing left to its caller.

Floating static row balancing reaches the source double logarithm’s unimplemented
fatal path for nonzero terms. Go returns `ErrLPBalanceNotImplemented` at that
point; no invented logarithm or balancing algorithm is credited.

The next layers retain the full requested scope:

1. Concrete scoped vectors, actual numeral/numeric-pair types, priority-queue integration and
   indexed vectors, with independent state/ordering/arithmetic tests.
2. Sparse/static/dense matrices, remaining permutation integration, eta transformations and LU,
   checked against independent matrix operations and exact residuals.
3. Scaling, settings, random updates and primal/dual simplex, tested with feasible,
   infeasible, degenerate and unbounded programs and independent certificates.
4. LAR constraints, scopes, terms, bound propagation and explanations, including
   strict bounds and evidence validation.
5. MPS input and the remaining upstream test scenarios, followed by coverage and
   retained-consumer audits across the complete implementation.

`src/test/lp.cpp` has command-selected test branches and an unconditional return
before several trailing test calls. Those trailing calls are not credited as
executed upstream tests. Reachability of project templates must be checked before
adding concrete Go instantiations; definition-only templates and STL code are
not port targets.

Focused checks (from the repository root):

```sh
go -C pkg test ./util/lp -run 'TestLPFloat(Presolve|Solver|ColumnInfo|Scaler|StaticMatrix)' -coverprofile=/tmp/g3-lp-native.cover
go -C pkg test -tags 'g3debug g3checked' ./util/lp -run 'TestLPFloat(Presolve|Solver|ColumnInfo|Scaler|StaticMatrix)' -coverprofile=/tmp/g3-lp-debug.cover
go tool cover -func=/tmp/g3-lp-native.cover
go -C pkg vet ./util/lp
go -C pkg run ./cmd/portcheck
```

Prepared-solve and floating/initial-exact/entering-column integration checks pass
25.210s native and49.067s combined debug/checked. Both LP vets pass. These are
focused component profiles; their aggregate percentage is not full LP coverage.
The complete native module snapshot above includes these prepared entry points.

Floating scaler plus all floating static-matrix regressions pass0.204s native
and0.199s combined debug/checked. Both LP vets pass. Profiles and authoritative
logs are `/mnt/oldrog/tmp/g3-scaler-{native,debug}.{cover,log}`. The complete
native run above includes the scaler and common floating caller restoration.
The precise no-op entry point, exact caller preparation and complete simplex/LAR
scaling integration remain open.

Floating solver preparation, column metadata, scaler and all floating static
matrix regressions pass0.233s native/0.244s combined debug/checked. Both LP
vets and source inventory/evidence/no-generics checks pass. Authoritative logs and
profiles: `/mnt/oldrog/tmp/g3-solver-assembly-{native,debug}.{log,cover}`.
The hourly full native run above includes common preparation and presolve.
Common preparation does not supply primal/dual optimization or exact owners.

Floating presolve plus solver/column/scaler/static regressions pass0.216s in
both native and combined debug/checked profiles. Authoritative focused logs
and coverage: `/mnt/oldrog/tmp/g3-presolve-{native,debug}.{log,cover}`.

Floating primal preparation plus floating presolve/solver/column/scaler/static
and LU regressions pass0.769s native/0.766s combined debug/checked. Logs and
profiles: `/mnt/oldrog/tmp/g3-primal-prepare-{native,debug}.{log,cover}`. The
current full native snapshot includes primal preparation and all core additions.
Preparation methods initialize matrix/vectors; current integration tests construct
the actual primal core and execute its initialization. Optimization remains open.


### Floating Harris step limits and leaving selection — 2026-10-10

Ported retained double/double Harris epsilon control, feasible/infeasible
one-sided and boxed step limits, indexed Harris theta, randomized maximum-pivot
leaving selection, and entering-column bound flips. Preserved global cached
feasibility, strict pivot tolerances, first-operand std::min IEEE behavior,
random circular tie order, boxed lower crossing's upper-bound epsilon scale,
flip output even on rejection, and error publication prefixes. Empty leaving
selection consumes RNG and leaves epsilon zero before the typed undefined-domain
guard; successful selection restores settings epsilon. Definition-only checked
feasible-positive helper is excluded. Exact/tableau variants, breakpoints and
full find_leaving_and_t / optimization integration remain open.

500 independent bound-intersection worlds verify theta and randomized strongest
admissible pivot/step. Independent infeasible boundary cases cover all column
types and directions; targeted tests cover IEEE/ties, skipped pivots, zero-step
short circuit, missing bound/index guards, entering flips and native/debug
assertion differences. Focused core/primal/presolve/solver/column/scaler/static/LU
checks pass native1.029s and debug/checked0.837s. Harris statements134/136 (98.5%)
native and136/136 (100%) debug; native gaps are erased assertions. Profiles:
/mnt/oldrog/tmp/g3-harris-{native,debug}.cover. Whole LP remains65 partial/45
unported. The20:03 full native90.0% aggregate predates this Harris addition.

Native and combined debug/checked LP vet, source inventory/evidence/no-generics
verification and diff checks pass. Regenerated graph stays64 packages with
unchanged edges and the new Harris production file recorded.


### Floating breakpoint selection and full leaving/step dispatch — 2026-10-10

Ported concrete double breakpoint records, strict sign/exact-zero gates,
row-bound and entering boxed full-span construction, queue lifecycle, slope
changes and double first-crossing selection. Connected actual breakpoint and
Harris/entering-flip/unlimited routes through FindLeavingAndT. Preserved signed
breakpoint deltas, index-before-queue append publication, fixed-bound double
slope jump except at zero, cached global infeasibility dispatch and the source
double behavior that always stops at the first dequeued breakpoint, even if
slope remains beneficial. Exact randomized multi-breakpoint continuation,
tableau/precise leaving paths, diagnostic printing and optimization remain open.

600 independent movement-ray/half-space crossing worlds check ordered records,
nearest absolute distance, signed steps, zero crossings and boxed entering span.
Independent hinge slope jumps cover fixed/low/upper and both signs. Integration
checks actual Harris basic pivot, entering flip, unbounded direction and flagged
infeasible breakpoint selection. IEEE and reached invalid extent/heading/queue
checks retain earlier record/queue mutation and output prefixes. Native focused
core/primal/presolve/solver/column/scaler/static/LU tests pass0.845s; combined
debug/checked pass0.926s. New code112/120 (93.3%) native,115/120 (95.8%) debug;
gaps propagate queue errors under inconsistent metadata or are opposite-profile
assertions. Profiles: /mnt/oldrog/tmp/g3-breakpoint-{native,debug}.cover.
Both LP vets pass. One source unit becomes partial: LP66 partial/44 unported.
The20:03 full native module90.0% aggregate predates Harris/breakpoint additions;
no immediate full rerun is needed. Source/evidence/no-generics and diff checks
pass; graph regenerated with unchanged edges and the new production file.


### Floating reduced-cost and steepest-edge updates — 2026-10-10

Ported floating pivot-row reduced-cost updates, entering-cost refresh with
source drift/profitability return codes, cost-switch predicate, actual indexed
LU transposed beta solve, indexed entering norm, norm update/reset counter and
steepest-edge recurrence. Source nonbreakpoint infeasibility transition subtracts
then zeros leaving cost after publishing the new reduced cost. Preserved
source descending refresh sum, sparse duplicates, native IEEE comparisons,
first-operand std::max on NaN, fixed-column dot product before the no-write
branch and mutation prefixes at malformed native bounds. Uncalled debug
calculate_column_norm_exactly is excluded. Actual pivot/advance/optimization
orchestration and exact/tableau updates remain open.

160 independent upper-triangular basis/coordinate worlds run actual sparse and
forced-dense LU, compare beta against independent transposed substitution and
compare recurrence norms/reduced costs against explicit new-basis coordinates.
Actual new-basis refactor/entering solves then verify refreshed costs. Tests
cover drift/profitability thresholds, source RNG periodic resets, cost-mode
switches, infeasibility-cost changes, norm floor/fixed columns/duplicates/IEEE
and reached publication/error guards. Focused native core/primal/presolve/solver/
column/scaler/static/LU checks pass0.936s; debug/checked pass0.975s. New code105/111
(94.6%) native and111/111 (100%) debug; native gaps are erased assertions.
Profiles: /mnt/oldrog/tmp/g3-refresh-{native,debug}.cover. Both LP vets and source/
evidence/no-generics verification pass. LP stays66 partial/44 unported. The20:03
full native90.0% aggregate predates these additions; no full repeat is due.
Graph regenerated with unchanged edges and the new production file.


### Floating primal pivot/advance orchestration — 2026-10-10

Ported floating column/row pivot consistency codes, bound-flip advance,
entering/leaving pivot orchestration and complete floating entering advance.
Actual LU beta/pivot/refactor, X/basis updates, cost/norm transitions and pricing
list mutations execute in source order. Preserved zero-pivot code1, strict
normalized Harris comparison, signed delta handling, numerical refusal versus
Go child errors, feasibility-only early returns, restored-basis versus failed
new-basis ownership, and two-observation tentative-unbounded/unstable escalation.
Borrowed Go DebugOut emits the source failure diagnostic after rollback/counter
publication. Whole solve/iteration/resource/statistics orchestration, precise/
tableau advances and complete original smoke drivers remain open.

180 independent known-basis ray worlds through sparse/forced-dense LU verify
entering selection, limiting basic variable, new X/basis/cost and equation
residuals for both movement signs. Separate worlds cover flips, zero steps,
feasibility-cost repair/restoration and list retention, source drift decisions,
actual stale-basis recovery/refactor, real singular rollback/refusal, actual
drop-based restore/diagnostic refusal, IEEE pivot comparison and reached output/
mutation guards. Distinct ray distances avoid treating roundoff-split exact ties
as exact ordering; Harris tests separately check source randomized equal pivots.
Focused native core/primal/presolve/solver/column/scaler/static/LU checks pass
0.951s and combined debug/checked0.973s. New code137/167 (82.0%) native and151/167
(90.4%) debug; remaining statements qualify child/index errors under malformed
metadata or belong to the other assertion profile. Profiles:
/mnt/oldrog/tmp/g3-advance-{native,debug}.cover. Both LP vets and source/evidence/
no-generics verification pass. LP stays66 partial/44 unported. Last full native
module snapshot remains20:03 UTC90.0%, predating these additions; no immediate
full rerun. Regenerated graph has unchanged edges plus the new production file.


### Floating stage-one core solve and iteration control — 2026-10-10

The retained double stage-one core now exposes Solve, connecting actual
initialization, randomized entering budgets, one-iteration status decisions,
pivot/flip advance, optimum refactor/recheck and tentative-unbounded/unstable
recovery. Preserved source do-while <= limits without inventing an exhausted
status, early feasible-only return before residual validation, normalized
objective costs, post-increment per-core/global statistics and cancellation
status. Floating cost/zero-nonbasis assertions and actual U/static nonzero
counts are implemented. Native Go writers and classic six-digit number text
replace STL stream state; no locale/flag emulation. Definition-only done() is
excluded. Original upper-only constructor/default tableau smoke profile,
find_feasible_solution diagnostic wrapper, exact/tableau loops and outer primal/
dual/LAR solvers remain open.

Both original test_lp_0/test_lp_1 numerical fixtures now solve through the
retained stage-one LU profile, with optimum/equation/nonnegative assertions
strengthened beyond the source smoke calls.100 independent positive-constraint
bounded LP worlds also solve to exhaustive dense Gauss-Jordan vertex maxima.
These are numerical fixture ports, not completion credit for the original
constructor/default profile or complete upstream driver. Tests cover infeasible,
unbounded, feasible-only/residual rejection, source iteration-limit exits,
actual borrowed cancellation/host errors, unsigned seeded entering-budget
arithmetic, counter wrap, printed cadence/cost/nonzero/IEEE text and writer/
metadata mutation guards. Repeated Solve with default cost backup reaches a
source debug nonempty-backup assertion; clock tests construct fresh owners.

Focused native core/primal/presolve/solver/column/scaler/static/LU checks pass
0.922s; combined debug/checked1.052s. Solve61/84 (72.6%) native,66/84 (78.6%)
debug; statistics54/69 (78.3%),65/69 (94.2%). Gaps include rare numerical status
rechecks, child-error propagation under invalid metadata and opposite-profile
assertions; no replacement numerical callbacks fabricate those paths. Profiles:
/mnt/oldrog/tmp/g3-solve-{native,debug}.cover. Both LP vets and source/evidence/
no-generics verification pass. LP remains66 partial/44 unported. Last full
native90.0% snapshot remains20:03 UTC, predating these additions; no immediate
full rerun. Graph regenerated with unchanged edges and both new production files.


### Floating outer primal LU integration and source limitation — 2026-10-10

Ported outer FindMaximalSolution / SolveWithTotalInf orchestration with actual
presolve, assembly, scale, residual-basis preparation, core construction/Solve,
status/iteration publication and core lifetime cleanup. External column and
original objective reconstruction include shifts, scaling and fixed columns;
numeric external indices choose a deterministic source unordered objective sum.
Partial Go constructor owners close before error return without publication;
a core whose Solve fails remains published while outer status/iteration fields
retain their source prefix. Default tableau and exact variants, source solution
validators/diagnostics and complete original drivers remain open.

120 independent bounded low/boxed-coordinate LP worlds compare outer solutions
with exhaustive dense vertex maxima, including shifted bounds, physical fixed
columns/costs, actual scaling, equations and box feasibility. Actual infeasibility,
empty/preexisting-status routes, output/clock failures, early value lookup,
objective error prefixes and explicit Close ownership also pass.

End-to-end upper-only worlds exposed a source inconsistency, verified against
lp_primal_simplex.hpp:146-165 and column_info.h:100-102: assembly flips an upper-only
variable but bounds retain upper_bound with zero upper value. Regression problem
maximize -x subject to x<=3,z>=0,-x+z<=2 reproduces source status OPTIMAL at x=3,
cost=-3; independent vertex enumeration proves optimum2 at x=-2,z=0. The port
preserves this source behavior and records it prominently rather than silently
correcting the algorithm or claiming upper-only correctness. Fixed columns use
actual SetFixedValue; merely coincident bounds do not set the source fixed flag.

Focused native core/primal/presolve/solver/column/scaler/static/LU checks pass
0.857s, combined debug/checked0.891s. New outer integration54/63 (85.7%) in both
profiles; remaining statements propagate child errors under invalid metadata.
Profiles: /mnt/oldrog/tmp/g3-outer-{native,debug}.cover. Both LP vets and source/
evidence/no-generics verifier pass. LP stays66 partial/44 unported. Last full
native snapshot remains20:03 UTC90.0%, predating these additions; no immediate
full rerun. Regenerated graph records the new production file with unchanged edges.

### Precise infeasibility bound limits — 2026-10-11

Precise infeasibility bound limits pass 1440 independent math/big pair worlds, nine live output/input/scalar aliases, six pooled/direct physical refusal campaigns through success, source sign and stopped-owner guards, and three actual older-caller comparison profiles. General-bound equality assigns zero even over a negative old theta; satisfied low-bound equality leaves theta and unlimited unchanged. Binary pair division retains its scalar copy and full difference/ratio temporaries with the primal owner; delegated comparisons keep their base receipts. Native coverage92/112 (82.1%), combined debug/checked104/112 (92.9%); twelve native gaps are debug assertions. Eight debug gaps qualify protected capture/valid close returns and sign-read errors behind validated manager inputs. Focused4.923s native/5.574s debug; bound-step/pricing/tableau-cost regressions46.940s/53.471s pass. Full precise basis-column theta, leaving ratio, breakpoints, advancement and Solve remain incomplete.

Profiles: `/mnt/oldrog/tmp/g3-inf-limit-{native,debug}.cover`; regression profiles: `/mnt/oldrog/tmp/g3-inf-regression-{native,debug}.cover`. The latest full native checkpoint predates these additions. Source inventory remains84 partial/26 unported.
Both LP vets, module-wide compilation and source-ledger verification pass. The production import graph includes the new file with unchanged dependency edges.

### Exact Harris bound epsilon — 2026-10-11

Exact Harris bound epsilon and zero_harris_eps pass240 independent math/big pair worlds including lexicographic absolute signs, zero/negative/nonzero scalar settings and live bound/output aliases; eight pooled/direct positive/negative-bound and zero/nonzero-epsilon physical refusal campaigns run through success, with two real older-caller comparison profiles and stopped/nil guards. The full (1+abs(bound)/10)*epsilon/3 expression is retained even for zero epsilon. Pair intermediates and copied-left rational expressions remain with the primal owner for cleanup; output assignment occurs after expression evaluation. New file86/99 statements (86.9%) in both native and combined debug/checked profiles. Thirteen gaps qualify capture protected by ready, inline constant setters, zero-comparison/negation guards, valid temporary destruction and final valid-owner cleanup. Focused0.717s native/0.944s debug/checked. Full basis-column limits, exact leaving/ratio selection and Solve remain incomplete.

Profiles: `/mnt/oldrog/tmp/g3-harris-exact-{native,debug}.cover`. The latest full native checkpoint predates this addition. Source inventory remains84 partial/26 unported.

Source audit for subsequent work: `restore_harris_eps` at `lp_primal_core_solver.h:328` constructs a rational from the floating tolerance; `src/util/rational.h:56` marks that constructor UNREACHABLE. Restoration and its exact leaving-selector consumer are not credited by the bound-epsilon implementation.

Both LP vets, module-wide compilation and source-ledger verification pass. The production graph records the new file with unchanged dependency edges.

### Exact basis-step limit helpers — 2026-10-11

Exact upper-negative, positive/negative boxed infeasibility and positive/negative feasible no-check basis-step helpers pass1200 independent math/big pair worlds with live X/low/upper output aliases, five scalar/output aliases, ten pooled/direct full-expression physical campaigns and four directed direct-allocation boxed comparison-prefix campaigns through success, five older-caller comparison-owner profiles and source sign/stopped/nil guards. The source upper-bound epsilon for upward motion from below a boxed low bound is preserved; feasible helpers clamp negative theta while boxed helpers do not. The source commented boxed sign assertions remain erased. Epsilon remains owned by the primal through subsequent delegated comparisons, ratio expressions, assignments and clamping; base comparison receipts remain separate, including failures that retain no raw comparison receipt. New helper file96/110 statements (87.3%) native and104/110 (94.5%) combined debug/checked; eight native gaps are debug assertions. Six debug gaps protect valid epsilon destruction, sign reads, capture and zero comparison/copy. Focused9.720s native/13.722s debug; prior Harris/infeasibility/bound-step regressions24.722s/30.436s pass. Full basis-column dispatch, precise leaving/ratio selection, breakpoint integration and Solve remain incomplete.

Profiles: `/mnt/oldrog/tmp/g3-basis-limit-{native,debug}.cover`; prior-helper regression logs: `/mnt/oldrog/tmp/g3-basis-regression-{native,debug}.log`. The latest full native checkpoint predates this addition. Source inventory remains84 partial/26 unported.

Both LP vets, module-wide compilation and source-ledger verification pass. Production graph edges are unchanged; the graph records the new helper file.

### Exact basis-column theta dispatch — 2026-10-11

Exact basis-column theta dispatch passes2400 independent math/big pair worlds across all five column types, stored feasible/infeasible-set states and both directions, including live X/low/upper output aliases. Sixteen scalar/output alias profiles, eighteen pooled/direct copy/expression campaigns through success, five actual older-caller comparison profiles and nil/index/stopped/unknown-type/zero-direction guards pass. Native free-column scalar copies can share storage and succeed with allocation disabled; the arithmetic profiles exercise physical refusal. The by-value multiplier remains owned through child expressions and final output clamping, public reentry is guarded while it is live, and delegated base comparison receipts remain separate. Dispatch uses the stored infeasibility set even when X disagrees; fixed uses the boxed source branches; the final limited-theta clamp also applies to free/no-limit directions. Native unknown column assertions are erased; debug rejects them. New file59/68 statements (86.8%) native and60/68 (88.2%) combined debug/checked. Eight debug gaps qualify protected capture/sign/index guards, final zero comparison and valid cleanup; native also erases the unknown-type assertion. Focused21.396s native/34.671s debug; previous basis/Harris/infeasibility-limit regressions14.893s/20.874s pass. Precise leaving/ratio selection, breakpoint integration, advancement and full Solve remain incomplete.

Profiles: `/mnt/oldrog/tmp/g3-basis-dispatch-{native,debug}.cover`; prior-helper regression logs: `/mnt/oldrog/tmp/g3-dispatch-regression-{native,debug}.log`. The latest full native checkpoint predates this addition. Source inventory remains84 partial/26 unported.

Both LP vets, module-wide compilation and source-ledger verification pass. Production graph edges are unchanged; the graph records the new dispatcher file.

### Exact breakpoint records and slope updates — 2026-10-11

Exact breakpoint record construction, priority enqueue, clearing and slope updates pass180 independent signed rational/epsilon record/absolute-priority worlds and360 independent slope worlds with actual LAR basis headings, fixed zero/nonzero doubling, entering increments, zero coefficients/directions and Ed/breakpoint-output aliases. Fourteen pooled/direct ownership campaigns run through success, including inline signed-minimum negation and separately owned queue comparison refusals; two older-caller comparison profiles and real queue-stop-before-Clear/error-prefix cases pass. Construction retains by-value/constructor temporaries through their source scopes and destroys construction/expression temporaries before priority enqueue; it appends the record before absolute-priority enqueue and preserves that record on failure. Queue numerical receipts stay separate from primal receipts. Source default zero-capacity enqueue fails after record publication; no automatic capacity fix is invented. Record clearing precedes queue clearing and source ascending record destruction is preserved. New file123/154 statements (79.9%) native and130/154 (84.4%) combined debug/checked. Remaining debug gaps qualify valid copy/zero/capture/destruction guards and a native-only heading return; native additionally erases source assertions. Focused0.709s native/0.867s debug; previous basis-dispatch/basis-limit/Harris/infeasibility-limit regressions37.127s/57.680s pass. Exact breakpoint collection, randomized slope traversal and full precise leaving/advance/Solve integration remain incomplete.

Profiles: `/mnt/oldrog/tmp/g3-breakpoint-exact-{native,debug}.cover`; prior-helper regression logs: `/mnt/oldrog/tmp/g3-breakpoint-regression-{native,debug}.log`. The latest full native checkpoint predates this addition. Source inventory remains84 partial/26 unported.

Both LP vets, module-wide compilation and source-ledger verification pass. Production graph edges are unchanged; the graph records the new exact breakpoint file.
