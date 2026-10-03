# Complete standalone SAT solver packet

Status: source inventory and concrete implementation design; solver production is not implemented. The complete standalone source solver is the objective. Existing clause/watch/queue storage tests do not establish search, propagation, conflict analysis, incremental scope, model, or core correctness. No generic Go, unsafe, boxed hot records, fake extension, or replacement DPLL algorithm is permitted.

## Audited source and evidence

Read the complete `src/sat/sat_solver.h` (578 lines) and `.cpp` (3770 lines), the complete registered `src/test/sat_user_scope.cpp`, configuration and four parameter-generator descriptions, and the complete dependency headers. The method index below records every solver definition, including overloads and helper bodies; inline header bodies are inventoried separately. The complete cleaner, model_converter, SCC, elim_eqs, probing, asymm_branch, MUS, integrity_checker, simplifier, clause_use_list, par, and max_cliques bodies have now also been read. Inactive iff3-finder and external/generated runtime algorithms require their own complete audits before translation; this document does not claim those algorithms have been audited merely from their interfaces or call sites.

`src/test/main.cpp:239` registers `sat_user_scope`. Its source test has ONE textual ENSURE site, inside an infinite campaign. One explicit finite outer cycle executes 42 coherence comparisons: seven repetitions of three pushes/two pops, then seven final pops. It creates 168 three-literal clauses, initially seven variables (variable zero is unused in the formulas), and compares incremental search to a separately reconstructed solver using DEFAULT parameters. A finite-cycle harness is an explicit test termination adaptation, not an assertion-count invention. Both solvers must additionally agree with an independent exhaustive truth-table oracle for the six formula variables, and every returned model must satisfy original active clauses. The oracle is test-only; production retains the real source algorithms.

`literal(r(6)+1,r(2)==0)` has platform-dependent argument evaluation order. The Go fixture will choose and document one deterministic order and preserve the actual source random_gen recurrence and 15-bit outputs. Coherence and independent satisfiability, rather than an invented fixed source CNF transcript, are the assertions. A permanent selected-reference transcript requires a separately identified tool/profile if introduced.

## Complete executable surface

| Source region | Required translation and observable behavior |
| --- | --- |
| solver.cpp:34–160 | Actual owner constructor/destructor, source shared params assignment, component initialization order, nonlearned clause copy, full MkVar publication sequence. |
| 161–495 | Mutable caller clause sort/compaction; user guard append; zero/unit/binary/ternary/nary creation; attach/detach; watch selection; learned clause allocation and reinitialization. |
| 496–722 | Conflict publication, assignment then queue/phase/ext notifications, clause status, binary/ternary/nary/external propagation, prefix watch compaction, update-glue branch. |
| 723–932 | Check initialization, burst search, exact simplify/restart/GC schedule, abort/error distinctions, real parallel solver copies/unit exchange/cancellation. |
| 933–1190 | 15-bit RNG case splits, exact phase policy, extension final-check outcomes, assumptions/user guards and MUS core routing, initialized search state. |
| 1191–1377 | Default cleanup/SCC/simplifier/probing/asymmetric branching, stable watch sort, real model converter, check_model, geometric/Luby restarts. |
| 1378–1674 | All five GC policies, source stable tie order, active/frozen/used/inactivity/glue/PSM handling, live-justification deletion exclusions. |
| 1675–2407 | FIRST-UIP conflict analysis; all justification kinds; unsat cores/init-conflict helpers; antecedent marking and activity; minimization, level approximate set, unmark cleanup. |
| 2408–2530 | Actual dynamic subsumption resolution against binaries/ternaries and probing cache; source mark and lemma mutation order. |
| 2531–2762 | Decision and user scopes, reverse unassignment, reinit stack, assumption reassertion, clause/bin/variable GC after removing user guards. |
| 2763–2851 | All parameter update/descriptor routes, component statistics/reset differences, activity rescaling, binary collection's actual learned flag semantics, integrity checker. |
| 2852–3118 | Mark checks, all finite clause/watch/unit/assignment/justification/DIMACS/WCNF displays, clause counts, missed-propagation checks, explicit cleaner/simplifier/SCC wrappers. |
| 3119–3637 | Actual mutex graph/max-cliques algorithm, bounded and ordinary consequences, antecedent index sets, model/core fixup, exact helper traversal. |
| 3638–3770 | Asymmetric-branch wrapper, sticky stream status/stat formatting, all source stats reset/collect and operator insertion. |

Header inline methods remain real methods: scoped detach constructor/destructor/deletion and checkpoint-disable lifetime; basic getters, assign dispatch, conflict overload, offset/ref conversions, checkpoint/canceled/set_par/config, mark/unmark, can_delete, all activity actions, iterator values, and clause overloads. Source raw pointers/references become checked scalar locators and by-value iterator views; caller output arrays/text are explicit boundaries. This is an ABI/lifetime adapter, not permission to omit body actions.

`solver::resolve_weighted` is declared at solver.h:349 but has NO definition anywhere under src/sat. It must be inventoried as source-declared/unavailable rather than supplied an invented algorithm. The commented CLONE_BEFORE_SOLVING alternative and disabled iff3-finder call are conditional/inactive baseline inventory, distinct from active bodies. `if(false && asms.empty())` consequences helper branch remains source-inactive; its defined helper bodies are still inventoried.

## Required dependencies: genuine algorithms and runtime

| Dependency | Need and current gate |
| --- | --- |
| Existing SAT types/sets/approx/clauses/temp/wrappers/watches/justifications/activity queue | Real foundations; retain scalar clause storage owner/slot/generation/kind separately from logical source IDs. Frozen storage files are root-owned during integration. |
| cleaner | Full root-level unit/binary/nonlearned/learned cleanup, counters and attach/detach policy; mandatory default Check path. |
| simplifier + clause_use_list | Full occurrence lists, subsumption, blocked-clause and resolution elimination, todos, costs, allocation/finalization, learned/nonlearned passes; default enabled policies cannot be disabled to green tests. |
| SCC + elim_eqs | Actual binary implication SCC traversal and literal substitution/model converter integration; enabled by default. |
| probing | Real probes/cache/implications/limits and source typo `probing_chache_limit`; enabled by default. |
| asymm_branch | Real source clause rounds/detach/propagate/strengthening and limits; enabled by default. |
| model_converter | Real blocked/eliminated variable reconstruction, entries/nested clause arrays, reverse conversion, copy/check/display/reset; cannot return a partial assignment as a substitute. |
| MUS | Complete source core algorithms, scoped assumption state, model copies, support/assignment sets and restart limit. Default core minimization false does not eliminate declared executable surface. |
| integrity_checker | Real clauses/watches/variables/justifications invariant checks, selected debug routes, no always-true placeholder. |
| max_cliques | Actual utility algorithm used by FindMutexes; concrete specialization, owned adjacency and index sets. |
| par | Actual solver/unit exchange and child resource-limit topology. Cold Go concurrency/scheduler adapter needs explicit policy; do not silently force parallel_threads=1. |
| ParamsContext/ParamsRef + generated sat parameter modules | Existing local typed getters and shared Ref lifecycle. Source generated getters also read global gparams module fallback. Real owner-local registry/runtime service or explicit coordinator-approved global service is required; defaults alone do not implement fallback. |
| utilReslimit | Existing real private resource owner provides increment/cancel/children/scoped limits; borrowed by solver. Its public facade and lifetime/cold concurrent usage must be coordinated rather than reimplemented. |
| memory_manager | Mandatory checkpoint/probing/status accounting. `memory::get_allocation_size` is allocator accounting, NOT RSS. Source allocations include sizeof(size_t) prefix, max/current/count bookkeeping, selected thread-local synchronization profile, and allocation-failure prefixes. A real shared runtime accounting service is required; Budget.Used, runtime.MemStats, and zero getters are not equivalent. |
| random_gen | Existing utilhash.Random is exact source uint32 LCG214013+2531011 and `(state>>16)&0x7fff`; embed cold scalar state, avoid an uncharged extra heap owner. |
| get_luby + stopwatch | Source floating log/pow/recursive Luby and real clock elapsed time; audit UINTMAX index wrap and float→uint domains. Runtime clock can be cold service but cannot be fabricated. |
| NumericTextStream/Statistics/parameter descriptors | Existing real render/storage services, borrowed stream state and borrowed key regions; solver statistics keys must be explicitly allocated in a genuine key owner. No hidden literal/key cache or fabricated process-memory collector. |
| extension | Nine real cold dispatcher methods: propagate, get_antecedents, asserted, check, push, pop, simplify, clauses_modifed, get_phase. One dispatcher per owner is allowed; hot watches keep scalar extension constraint IDs. Nil is the real source core case. No mock extension claimed as production. |
| verbose/TRACE/debug tags/ostream flush | Explicit runtime IO/diagnostic dependencies and selected profile inventory; no fake command runtime. |

CMake active solver dependencies include the inprocessing and support translation units above. Their headers are fully read; cleaner/model_converter/SCC/elim_eqs/probing/asymm/MUS/integrity_checker/simplifier/clause_use_list/par and max_cliques bodies are fully read, and remaining algorithm bodies will be read in full before each independent translation packet, with hashes below pinning the audited revision. Pyg defaults and actual extra params must remain source exact, including core minimization and optional parallel/diagnostic modes.

## Concrete API proposal and ownership

Proposed cold `satSolver` owner, public constructor `NewSATSolver(satSolverConfig) (*satSolver,error)` with `{ParamsContext *ParamsContext; Params ParamsRef; RLimit *utilReslimit; Budget *arena.Budget; Runtime *satRuntime; Extension *satExtensionDispatcher}`. The lifetime domain is explicit: parameter and runtime/resource services outlive the solver, ParamsRef is retained through actual shared Assign/Del, and extension is borrowed. Scalar source Boolean/literal/model/result representations remain existing concrete types.

Methods first required by the registered source test: `MkVar(ext,dvar bool)(satBoolVar,error)`, `MkClause([]satLiteral) error` (source mutates its array), `Check([]satLiteral)(lbool,error)`, `UserPush() error`, `UserPop(uint32) error`, `Close() error`. Full surface adds every source method above with typed error returns for allocation, closed/foreign owner, genuine bounds/lifetime/overflow danger and source-debug assertion sites. `Check` returns lUndef only at actual source abort/limit policies; cancellation/max-memory are source solver exceptions, not silently false. Model/core access copies into caller buffers or by-value checked iterators; no arena slice escapes. Source nonlearned Copy is distinct from invented generic Clone/Assign.

A solver cannot store `[]*satWatchList` or a hot `[]satWatchList` containing Go slices. Proposed one cold charged watch-store has pointer-free descriptors (list identity/generation, payload/lifetime refs, logical size/capacity, epoch) and concrete payload pages. The outer source vector owns scalar list descriptors with source 2,3,5,... growth and real move/destructor semantics. Transient by-value views may hold one cold store pointer on the caller stack, never in retained hot records. Full-capacity initialized watch tails and erased-versus-destroyed holes must preserve the frozen watch lifetime contract. Actual backing/descriptor headers/directory/ledger/lifetime bytes and all old+new growth peaks are charged once. Source payload capacity policy and physical arena capacity classes are separately documented; no pointer arithmetic or unsafe codec is introduced.

All variable arrays, scope records, justification arrays, nested occurrence/model/consequence lists, assumptions/sets, clauses/reinit vectors, activity queue, component owners, temporary stable-sort storage, scope guards and parallel graphs are real charged allocations. Borrowed Budget/runtime/callback environment/services are not double charged. Published growth prefixes survive later allocation failure exactly where source bodies publish; no blanket transaction rewinds owner state or Budget failure counter. Unpublished constructor preparations unwind their actual owned graph in lexical/source lifetime order. Cold mutexes/owner IDs reject copied owner values; no per-record closure/interface.

## Default policies and fidelity traps

Source default phase caching (on400/off100), Luby restart initial100/factor1.5/maxUINTMAX, random frequency .01/seed0, burst100, maximum conflicts UINTMAX, GC glue_psm(initial20000/increment500/smallLBD3/k7), minimization and dynamic subsumption true, core minimization false, one parallel thread, max memory UINTMAX MB. Simplification initial multiplier300, growth1.5/max500000; SCC, probing and asymmetric branching enabled. Simplifier resolution/subsumption and variable elimination enabled, blocked-clause elimination false. Exact parameter definitions and overrides must be tested; optional policies remain part of full scope.

Important observed action order:

- MkVar marks model stale and increments stats BEFORE sequential watch/assignment/justification/decision/level/activity/etc vector publications and queue/todo updates (cpp:135–160).
- MkClause base propagation may return before binary watches/stat counters; ternary/nary stats precede real clause allocation, attachments precede vector publication (195–370).
- Propagate increments qhead before watch iteration, publishes replacement clause literals before destination watch Push, and compacts only completed prefix when conflicts/extension outcomes exit (553–710).
- SimplifyProblem invokes cleaner → SCC → nonlearned/learned simplifier → stable watch sort → probing → asymm → extension, then assumptions; none can be a placeholder (1191–1250).
- No-extension get_max_lvl returns scope_lvl immediately; conflict analysis cannot replace this with an apparently tighter literal scan (2088+).
- Pop calls extension before debug bound assertion, shrinks scopes before reinit and uses retained scalar scope fields; unassignment retains justification/level/phase history (2531–2619).
- UserPop pop-to-base precedes zero-count handling; removes guarded clauses/binaries and shrinks variable arrays in actual source order, then propagates from qhead0 (2620–2751).
- CollectBinClauses(false) excludes learned; true includes BOTH. FindMutexes true then false deduplicates its pair map (2818+,3119+).
- WCNF maximum weight+1 wraps uint32; soft literal formatting uses ordinary source variable numbering, unlike DIMACS (2957+). Keep this defined source quirk.
- Watched external extraction retains the already documented source release value3/debug round-trip assertion behavior; real extension continuation must not silently repair it.
- Simplifier default elim_blocked_clauses=false still executes blocked-clause elimination when m_num_calls equals default elim_blocked_clauses_at=2, the third nonlearned call. This is an active default dependency, not an optional feature that can be skipped (simplifier.cpp:190–194).
- Probing cache availability publishes before individual cached literal pushes; operator() calls finalize after each completed run, which frees assigned/to-assert/cached bins (probing.cpp:41–57,243–246,261–265). Preserve this cleanup rather than inventing a permanently retained cache.
- Integrity checker check_assignment is genuinely return-true; check_learned_clauses counts frozen entries in NONlearned vector before checking learned; selected SASSERT calls run only in debug (integrity_checker.cpp:113–137,225+). Do not silently correct these source bodies.
- Asymmetric branching catches solver_exception to preserve unvisited clause pointer tail before rethrow and skips final saved-phase assignment on that exception path; scoped detach lifetime still runs (asymm_branch.cpp:82–114).
- Stats.reset clears16 counters; collect emits15 excluding nonlearned generation; solver reset_statistics does not reset SCC although collection includes it.
- Source hot integer additions/multiplication/wrapping and signed/float conversion domains require source-defined behavior/debug guards before genuine UB adapters; no universal early bounds narrowing.

## Full-scope TDD roadmap

1. Registered finite user-scope cycle FIRST observed compile RED against real missing solver API; permanent independent truth-table/model oracle and charged fixture lifetime. Tests stay RED until genuine default paths work. No reduced solver executable or bypass-default green claim.
2. Exact configuration/global module/runtime boundary tests FIRST; real nested vector/watch-store and model-converter/occurrence support lifetimes with failure injection and immediate charge/root checks.
3. Full source constructor/Copy/MkVar/MkClause/attach/detach/assignment/propagation algorithms, all clause widths and source watch quirks; bounded propagation closure oracle and publication fault tests.
4. Decision scopes/search/FIRST-UIP/backjump/minimization/dynamic subsumption/restarts/GC; truth-table differential matrices, learned-clause implication checks, exact phase/RNG/action traces and failure prefixes.
5. Genuine cleaner/SCC/elim_eqs/simplifier/probing/asymm/model conversion integrated with DEFAULT Check and scopes. Component packets may run in parallel after shared scalar/view contracts are agreed, but all source default dependencies remain required.
6. Assumptions/core/MUS/consequences/mutex graph, all auxiliary displays/stats/descriptor methods, explicit real extension routing, full parallel exchange/resource cancellation. Independent exact core/consequence/mutex oracles are tests, not production substitutes.
7. Native, checked-debug, named checked-debug race and actual JS RUN/PASS, vet, WASI, no-generics/source audit. Freeze files and report source method/accounting mappings; no claim complete solver while any executable source dependency is still open.

Full solver production files will be coordinated new `sat_solver*.go` and concrete dependency specializations only. Existing utilities, frozen storage, runtime ownership edits, manifests, progress and Git are parent-owned. The source body inventory is a plan/evidence boundary, not implementation status.

## Pinned source hashes

| Source file | SHA256 |
| --- | --- |
| `src/sat/sat_solver.h` | `7ec8a9c01ca2757723192c3e1098891043a91c7e719695fd2c806f717233c1c4` |
| `src/sat/sat_solver.cpp` | `ea35f45c69be4d179aec4e1af034b2295f2a026bb01c37a2d9f8c59abf519655` |
| `src/test/sat_user_scope.cpp` | `c86c383ea199a89a5311bae5f8186e0b59e6953aa17bb7d675e221aa0c5edffc` |
| `src/test/main.cpp` | `e18f789f7c4794ed08c2988aa637a778d58b72bb7c1bc616b8710a4c54c9e993` |
| `src/sat/sat_extension.h` | `18ee5002876191ffa8a3b50681d5f93b58ee912140956d228b7e0c69bff087b9` |
| `src/sat/sat_config.h` | `4207a3caa3bc54a06e88064859db569b2f122d7a4c189e41bffe45175d8388ca` |
| `src/sat/sat_config.cpp` | `cfbc032cf10a16acb1f913a18286ef5d1fd846edd249081013d3f1b0ca58122f` |
| `src/sat/sat_params.pyg` | `23c5c38b82c1184089a6e77dd8498a03b5607eb44f00ca972475fb18b5cf85df` |
| `src/sat/sat_scc_params.pyg` | `dc0a71c2aa4d5e8f1138a4f0bc46e9896c246b9a105e1daf6b75c8d0336deed7` |
| `src/sat/sat_asymm_branch_params.pyg` | `aa1fc70459542f193c2efb5bd54c985c83ed2e7a5cbdb5a339bd842d7b731f0e` |
| `src/sat/sat_simplifier_params.pyg` | `e3b1e74369b98a67bc0508882111d89f94cdc4695bbf277b17fab565d72dbda9` |
| `src/util/memory_manager.h` | `78b004da89d38686fdf0ad34807ad77f844e9d7ac275834ef312adc2c7bc1839` |
| `src/util/memory_manager.cpp` | `3b884edd5463aa7ddfe68d6e0badc0695eff2b6b6fd90736850dd740f43e29d6` |
| `src/util/luby.cpp` | `b11c52b8b4070fa8c3068abe87c9e1b2e1048b86a7e83515343796887392dab7` |
| `src/util/stopwatch.h` | `d04639e66c71315e72c145add2ceb6c52614a88d701170649f94b9b92ee9876b` |
| `src/util/max_cliques.h` | `08ffe398e0d558ac867e0fea775cf6ff6b60c5752016a8e1ed552e598b70fbba` |
| `src/sat/CMakeLists.txt` | `9e3420309620143592d3e7692ba9d072195bbd25758bdedaa599d36fd4422f1a` |
| `src/sat/sat_cleaner.h` | `da27c8f85264d0ebbac26604cf2e9f42ea83a788e53ff905314b2c1fb92b3044` |
| `src/sat/sat_cleaner.cpp` | `c054a91bd51798136c59ee1dd29f228960c912ca44c23b1143dbf6b1b3af899e` |
| `src/sat/sat_simplifier.h` | `d2a2d46302077c837ed22480be93a570ec0d74339c8b071ed6808ed2332e254a` |
| `src/sat/sat_simplifier.cpp` | `3ca576a5a9af07d608d5f19352b0661abaff5a743b09522658031cc4df656b09` |
| `src/sat/sat_scc.h` | `dcd5f98bd7a41f9f4b31212bccac77596d5d19b4dc09cc5084c53a893ad6716f` |
| `src/sat/sat_scc.cpp` | `2d02927006a609549c4c2fcde160dd4492beb50c62db5197b7d33d2b3cda560e` |
| `src/sat/sat_asymm_branch.h` | `54c07e4db1dd6b83ac1f2be987dfba76067baa4e8b27be259350a9c74e10bb9b` |
| `src/sat/sat_asymm_branch.cpp` | `a811281bb3da28e5b0087d36bd4f43d2b79f1ed276a6ae6b4e0dab26e89d7f66` |
| `src/sat/sat_probing.h` | `0837280ffa8cb5fe8eea62deb146bb43ab1ad5133db53cb2216e757157721e96` |
| `src/sat/sat_probing.cpp` | `3dcf3eecc14646c45c2e9ed55210c6e7b87e1a459f77c548344700580694910e` |
| `src/sat/sat_mus.h` | `70b609ae89bee2afd1e0dc1cae6af78f731c778c6317c8f261cf05a5dc458042` |
| `src/sat/sat_mus.cpp` | `3d830423a53ed0e1bcad8eb8b80a81e75ede654b55230642637640aecd3b4a9e` |
| `src/sat/sat_model_converter.h` | `760b847933f74903f81b19dcc5cd98a560f1d01407dabc8042638aaab9d01c9a` |
| `src/sat/sat_model_converter.cpp` | `57f99643ce2dc3eaf8866c01a9c4e621a64c84b496154e071cc8d819b79ab6e9` |
| `src/sat/sat_elim_eqs.h` | `e70d68b7bdac18ae3db23871c3d28b73e90d3dbccb8cf663728ee4401751f166` |
| `src/sat/sat_elim_eqs.cpp` | `532c3170feeb3b5057b77103dab74d92aa39bade4e1ec62d6b31f3562b0b4566` |
| `src/sat/sat_clause_use_list.h` | `b2b968c1172f8c93495b2b4ce773b669dcae05314f5416c39842350efd641bc6` |
| `src/sat/sat_clause_use_list.cpp` | `97f55406648121cc4e0ed7ebfc53f085171e5e261776c3fd424233328e6fcf65` |
| `src/sat/sat_integrity_checker.h` | `c7bf5d610fa2f30d361a3f1d74105bbb1380c6b58b8dd91ec6e6fc946c3d5f08` |
| `src/sat/sat_integrity_checker.cpp` | `65ac0aa33d67ef60b2ecb3511ef5b3ce7201500068e9cfd4b3decd78e584fde8` |
| `src/sat/sat_iff3_finder.h` | `5504f07974a849512ad07d6b136de5a9f3098da6c76aadd8e359988ea23b1c7e` |
| `src/sat/sat_iff3_finder.cpp` | `61137a1c6e5afde5c9d9346dec028d5fba2a6c4413a548fc4eb5f17acc390195` |
| `src/sat/sat_par.h` | `a49acb20e847429edbcbdd601808406aa7d5c8bac963185d162810e0db464617` |
| `src/sat/sat_par.cpp` | `149ff96bbb21c6ae9edbf619b9057e40c58f7aac6d4bb125cc8b0e8ec8947dca` |

## Solver.cpp member definition index (all solver bodies currently unported)

- `src/sat/sat_solver.cpp:34`: `solver::solver(params_ref const & p, reslimit& l, extension * ext):`
- `src/sat/sat_solver.cpp:68`: `void solver::del_clauses(clause * const * begin, clause * const * end) {`
- `src/sat/sat_solver.cpp:75`: `void solver::copy(solver const & src) {`
- `src/sat/sat_solver.cpp:135`: `bool_var solver::mk_var(bool ext, bool dvar) {`
- `src/sat/sat_solver.cpp:161`: `void solver::mk_clause(unsigned num_lits, literal * lits) {`
- `src/sat/sat_solver.cpp:179`: `void solver::mk_clause(literal l1, literal l2) {`
- `src/sat/sat_solver.cpp:184`: `void solver::mk_clause(literal l1, literal l2, literal l3) {`
- `src/sat/sat_solver.cpp:189`: `void solver::del_clause(clause& c) {`
- `src/sat/sat_solver.cpp:195`: `clause * solver::mk_clause_core(unsigned num_lits, literal * lits, bool learned) {`
- `src/sat/sat_solver.cpp:223`: `void solver::mk_bin_clause(literal l1, literal l2, bool learned) {`
- `src/sat/sat_solver.cpp:235`: `bool solver::propagate_bin_clause(literal l1, literal l2) {`
- `src/sat/sat_solver.cpp:249`: `void solver::push_reinit_stack(clause & c) {`
- `src/sat/sat_solver.cpp:256`: `clause * solver::mk_ter_clause(literal * lits, bool learned) {`
- `src/sat/sat_solver.cpp:269`: `bool solver::attach_ter_clause(clause & c) {`
- `src/sat/sat_solver.cpp:294`: `clause * solver::mk_nary_clause(unsigned num_lits, literal * lits, bool learned) {`
- `src/sat/sat_solver.cpp:307`: `bool solver::attach_nary_clause(clause & c) {`
- `src/sat/sat_solver.cpp:340`: `void solver::attach_clause(clause & c, bool & reinit) {`
- `src/sat/sat_solver.cpp:372`: `unsigned solver::select_watch_lit(clause const & cls, unsigned starting_at) const {`
- `src/sat/sat_solver.cpp:413`: `unsigned solver::select_learned_watch_lit(clause const & cls) const {`
- `src/sat/sat_solver.cpp:428`: `bool solver::simplify_clause_core(unsigned & num_lits, literal * lits) const {`
- `src/sat/sat_solver.cpp:459`: `bool solver::simplify_clause(unsigned & num_lits, literal * lits) const {`
- `src/sat/sat_solver.cpp:466`: `void solver::detach_bin_clause(literal l1, literal l2, bool learned) {`
- `src/sat/sat_solver.cpp:471`: `void solver::detach_clause(clause & c) {`
- `src/sat/sat_solver.cpp:478`: `void solver::detach_nary_clause(clause & c) {`
- `src/sat/sat_solver.cpp:484`: `void solver::detach_ter_clause(clause & c) {`
- `src/sat/sat_solver.cpp:496`: `void solver::set_conflict(justification c, literal not_l) {`
- `src/sat/sat_solver.cpp:504`: `void solver::assign_core(literal l, justification j) {`
- `src/sat/sat_solver.cpp:530`: `lbool solver::status(clause const & c) const {`
- `src/sat/sat_solver.cpp:553`: `bool solver::propagate_core(bool update) {`
- `src/sat/sat_solver.cpp:711`: `bool solver::propagate(bool update) {`
- `src/sat/sat_solver.cpp:723`: `lbool solver::check(unsigned num_lits, literal const* lits) {`
- `src/sat/sat_solver.cpp:805`: `lbool solver::check_par(unsigned num_lits, literal const* lits) {`
- `src/sat/sat_solver.cpp:898`: `void solver::exchange_par() {`
- `src/sat/sat_solver.cpp:926`: `void solver::set_par(par* p) {`
- `src/sat/sat_solver.cpp:933`: `bool_var solver::next_var() {`
- `src/sat/sat_solver.cpp:954`: `bool solver::decide() {`
- `src/sat/sat_solver.cpp:993`: `lbool solver::bounded_search() {`
- `src/sat/sat_solver.cpp:1013`: `lbool solver::propagate_and_backjump_step(bool& done) {`
- `src/sat/sat_solver.cpp:1036`: `lbool solver::final_check() {`
- `src/sat/sat_solver.cpp:1056`: `bool solver::check_inconsistent() {`
- `src/sat/sat_solver.cpp:1067`: `void solver::init_assumptions(unsigned num_lits, literal const* lits) {`
- `src/sat/sat_solver.cpp:1102`: `void solver::update_min_core() {`
- `src/sat/sat_solver.cpp:1110`: `void solver::reset_assumptions() {`
- `src/sat/sat_solver.cpp:1115`: `void solver::add_assumption(literal lit) {`
- `src/sat/sat_solver.cpp:1120`: `void solver::pop_assumption() {`
- `src/sat/sat_solver.cpp:1125`: `void solver::reassert_min_core() {`
- `src/sat/sat_solver.cpp:1141`: `void solver::reinit_assumptions() {`
- `src/sat/sat_solver.cpp:1161`: `bool solver::tracking_assumptions() const {`
- `src/sat/sat_solver.cpp:1165`: `bool solver::is_assumption(literal l) const {`
- `src/sat/sat_solver.cpp:1169`: `void solver::init_search() {`
- `src/sat/sat_solver.cpp:1191`: `void solver::simplify_problem() {`
- `src/sat/sat_solver.cpp:1251`: `void solver::sort_watch_lits() {`
- `src/sat/sat_solver.cpp:1257`: `void solver::set_model(model const& mdl) {`
- `src/sat/sat_solver.cpp:1263`: `void solver::mk_model() {`
- `src/sat/sat_solver.cpp:1289`: `bool solver::check_model(model const & m) const {`
- `src/sat/sat_solver.cpp:1347`: `void solver::restart() {`
- `src/sat/sat_solver.cpp:1378`: `void solver::gc() {`
- `src/sat/sat_solver.cpp:1455`: `void solver::gc_glue() {`
- `src/sat/sat_solver.cpp:1460`: `void solver::gc_psm() {`
- `src/sat/sat_solver.cpp:1466`: `void solver::gc_glue_psm() {`
- `src/sat/sat_solver.cpp:1472`: `void solver::gc_psm_glue() {`
- `src/sat/sat_solver.cpp:1481`: `void solver::save_psm() {`
- `src/sat/sat_solver.cpp:1493`: `void solver::gc_half(char const * st_name) {`
- `src/sat/sat_solver.cpp:1518`: `void solver::gc_dyn_psm() {`
- `src/sat/sat_solver.cpp:1611`: `bool solver::activate_frozen_clause(clause & c) {`
- `src/sat/sat_solver.cpp:1652`: `unsigned solver::psm(clause const & c) const {`
- `src/sat/sat_solver.cpp:1675`: `bool solver::resolve_conflict() {`
- `src/sat/sat_solver.cpp:1687`: `bool solver::resolve_conflict_core() {`
- `src/sat/sat_solver.cpp:1822`: `void solver::process_antecedent_for_unsat_core(literal antecedent) {`
- `src/sat/sat_solver.cpp:1835`: `void solver::process_consequent_for_unsat_core(literal consequent, justification const& js) {`
- `src/sat/sat_solver.cpp:1885`: `bool solver::resolve_conflict_for_init() {`
- `src/sat/sat_solver.cpp:1945`: `bool solver::process_consequent_for_init(literal consequent, justification const& js) {`
- `src/sat/sat_solver.cpp:1989`: `void solver::process_antecedent_for_init(literal antecedent) {`
- `src/sat/sat_solver.cpp:1999`: `void solver::resolve_conflict_for_unsat_core() {`
- `src/sat/sat_solver.cpp:2088`: `unsigned solver::get_max_lvl(literal consequent, justification js) {`
- `src/sat/sat_solver.cpp:2148`: `unsigned solver::skip_literals_above_conflict_level() {`
- `src/sat/sat_solver.cpp:2162`: `void solver::process_antecedent(literal antecedent, unsigned & num_marks) {`
- `src/sat/sat_solver.cpp:2180`: `void solver::fill_ext_antecedents(literal consequent, justification js) {`
- `src/sat/sat_solver.cpp:2187`: `void solver::forget_phase_of_vars(unsigned from_lvl) {`
- `src/sat/sat_solver.cpp:2198`: `void solver::updt_phase_counters() {`
- `src/sat/sat_solver.cpp:2218`: `unsigned solver::num_diff_levels(unsigned num, literal const * lits) {`
- `src/sat/sat_solver.cpp:2238`: `bool solver::process_antecedent_for_minimization(literal antecedent) {`
- `src/sat/sat_solver.cpp:2261`: `bool solver::implied_by_marked(literal lit) {`
- `src/sat/sat_solver.cpp:2339`: `void solver::reset_unmark(unsigned old_size) {`
- `src/sat/sat_solver.cpp:2350`: `void solver::updt_lemma_lvl_set() {`
- `src/sat/sat_solver.cpp:2363`: `void solver::minimize_lemma() {`
- `src/sat/sat_solver.cpp:2393`: `void solver::reset_lemma_var_marks() {`
- `src/sat/sat_solver.cpp:2408`: `void solver::dyn_sub_res() {`
- `src/sat/sat_solver.cpp:2531`: `void solver::push() {`
- `src/sat/sat_solver.cpp:2545`: `void solver::pop_reinit(unsigned num_scopes) {`
- `src/sat/sat_solver.cpp:2551`: `void solver::pop(unsigned num_scopes) {`
- `src/sat/sat_solver.cpp:2566`: `void solver::unassign_vars(unsigned old_sz) {`
- `src/sat/sat_solver.cpp:2583`: `void solver::reinit_clauses(unsigned old_sz) {`
- `src/sat/sat_solver.cpp:2620`: `void solver::user_push() {`
- `src/sat/sat_solver.cpp:2628`: `void solver::gc_lit(clause_vector &clauses, literal lit) {`
- `src/sat/sat_solver.cpp:2644`: `void solver::gc_bin(literal lit) {`
- `src/sat/sat_solver.cpp:2663`: `bool_var solver::max_var(bool learned, bool_var v) {`
- `src/sat/sat_solver.cpp:2675`: `bool_var solver::max_var(clause_vector& clauses, bool_var v) {`
- `src/sat/sat_solver.cpp:2689`: `void solver::gc_var(bool_var v) {`
- `src/sat/sat_solver.cpp:2725`: `void solver::user_pop(unsigned num_scopes) {`
- `src/sat/sat_solver.cpp:2752`: `void solver::pop_to_base_level() {`
- `src/sat/sat_solver.cpp:2763`: `void solver::updt_params(params_ref const & p) {`
- `src/sat/sat_solver.cpp:2773`: `void solver::collect_param_descrs(param_descrs & d) {`
- `src/sat/sat_solver.cpp:2781`: `void solver::collect_statistics(statistics & st) const {`
- `src/sat/sat_solver.cpp:2790`: `void solver::reset_statistics() {`
- `src/sat/sat_solver.cpp:2804`: `void solver::rescale_activity() {`
- `src/sat/sat_solver.cpp:2818`: `void solver::collect_bin_clauses(svector<bin_clause> & r, bool learned) const {`
- `src/sat/sat_solver.cpp:2845`: `bool solver::check_invariant() const {`
- `src/sat/sat_solver.cpp:2852`: `bool solver::check_marks() const {`
- `src/sat/sat_solver.cpp:2859`: `void solver::display_binary(std::ostream & out) const {`
- `src/sat/sat_solver.cpp:2878`: `void solver::display_units(std::ostream & out) const {`
- `src/sat/sat_solver.cpp:2887`: `void solver::display(std::ostream & out) const {`
- `src/sat/sat_solver.cpp:2895`: `void solver::display_justification(std::ostream & out, justification const& js) const {`
- `src/sat/sat_solver.cpp:2902`: `unsigned solver::num_clauses() const {`
- `src/sat/sat_solver.cpp:2925`: `void solver::display_dimacs(std::ostream & out) const {`
- `src/sat/sat_solver.cpp:2957`: `void solver::display_wcnf(std::ostream & out, unsigned sz, literal const* lits, unsigned const* weights) const {`
- `src/sat/sat_solver.cpp:3003`: `void solver::display_watches(std::ostream & out) const {`
- `src/sat/sat_solver.cpp:3015`: `void solver::display_assignment(std::ostream & out) const {`
- `src/sat/sat_solver.cpp:3022`: `bool solver::is_unit(clause const & c) const {`
- `src/sat/sat_solver.cpp:3044`: `bool solver::is_empty(clause const & c) const {`
- `src/sat/sat_solver.cpp:3053`: `bool solver::check_missed_propagation(clause_vector const & cs) const {`
- `src/sat/sat_solver.cpp:3071`: `bool solver::check_missed_propagation() const {`
- `src/sat/sat_solver.cpp:3082`: `void solver::cleanup() {`
- `src/sat/sat_solver.cpp:3089`: `void solver::simplify(bool learned) {`
- `src/sat/sat_solver.cpp:3098`: `unsigned solver::scc_bin() {`
- `src/sat/sat_solver.cpp:3119`: `lbool solver::find_mutexes(literal_vector const& lits, vector<literal_vector> & mutexes) {`
- `src/sat/sat_solver.cpp:3252`: `lbool solver::get_consequences(literal_vector const& asms, bool_var_vector const& vars, vector<literal_vector>& conseq) {`
- `src/sat/sat_solver.cpp:3285`: `void solver::fixup_consequence_core() {`
- `src/sat/sat_solver.cpp:3301`: `lbool solver::get_bounded_consequences(literal_vector const& asms, bool_var_vector const& vars, vector<literal_vector>& conseq) {`
- `src/sat/sat_solver.cpp:3365`: `lbool solver::get_consequences(literal_vector const& asms, literal_vector const& lits, vector<literal_vector>& conseq) {`
- `src/sat/sat_solver.cpp:3473`: `void solver::delete_unfixed(literal_set& unfixed_lits, bool_var_set& unfixed_vars) {`
- `src/sat/sat_solver.cpp:3488`: `void solver::update_unfixed_literals(literal_set& unfixed_lits, bool_var_set& unfixed_vars) {`
- `src/sat/sat_solver.cpp:3503`: `void solver::extract_fixed_consequences(unsigned& start, literal_set const& assumptions, bool_var_set& unfixed, vector<literal_vector>& conseq) {`
- `src/sat/sat_solver.cpp:3512`: `void solver::extract_fixed_consequences(literal_set const& unfixed_lits, literal_set const& assumptions, bool_var_set& unfixed_vars, vector<literal_vector>& conseq) {`
- `src/sat/sat_solver.cpp:3524`: `bool solver::check_domain(literal lit, literal lit2) {`
- `src/sat/sat_solver.cpp:3536`: `bool solver::extract_assumptions(literal lit, index_set& s) {`
- `src/sat/sat_solver.cpp:3589`: `std::ostream& solver::display_index_set(std::ostream& out, index_set const& s) const {`
- `src/sat/sat_solver.cpp:3599`: `bool solver::extract_fixed_consequences1(literal lit, literal_set const& assumptions, bool_var_set& unfixed, vector<literal_vector>& conseq) {`
- `src/sat/sat_solver.cpp:3628`: `void solver::extract_fixed_consequences(literal lit, literal_set const& assumptions, bool_var_set& unfixed, vector<literal_vector>& conseq) {`
- `src/sat/sat_solver.cpp:3638`: `void solver::asymmetric_branching() {`
- `src/sat/sat_solver.cpp:3652`: `void solver::display_status(std::ostream & out) const {`
- `src/sat/sat_solver.cpp:3716`: `void stats::collect_statistics(statistics & st) const {`
- `src/sat/sat_solver.cpp:3734`: `void stats::reset() {`
- `src/sat/sat_solver.cpp:3753`: `void mk_stat::display(std::ostream & out) const {`

## First source RED and independently frozen prerequisites

The real source fixture `sat_solver_user_scope_test.go` was written FIRST, then tested in isolated HEAD0da9cbe + only that test: `go test -run '^TestSATSolverSourceUserScopeFiniteCycle$' -count=1 .` exited1 with undefined real `newSATRuntime`, `satRuntimeConfig`, `satSolverConfig`, and `NewSATSolver`. This test intentionally remains compile-RED until genuine runtime/solver prerequisites are implemented; it does not pass by disabling default inprocessing or substituting the oracle. This RED fixture is excluded from independent prerequisite GREEN snapshots, explicitly rather than hidden behind build tags.

Parent approved a concrete common nested-store architecture and real standalone model conversion as separately testable prerequisites. The four files `sat_solver_nested_store.go/test.go` and `sat_solver_model_converter.go/test.go` were separately FIRST compile-RED, then GREEN. An out-of-range Go enum guard had an actual behavioral RED (`<nil>, size1` instead of typed error/size0), then implementation; this is a Go boundary for C++-invalid enum conversions, not a claim of defined source enum2 behavior. Additional source-body coverage passed immediately against existing translations, reported honestly. Final selected coverage is13 groups, including four common literal-store groups and nine model-converter groups. No standalone registered model-converter test exists; its CPP has17 textual SASSERT sites and zero ENSURE. Source scope42 checks are a separate still-RED campaign.

Model-converter mapping:

| Source | Implemented concrete mapping |
| --- | --- |
| model_converter.cpp:25–34 | newSATModelConverter, Close, Reset: all actual cold headers, scalar directories, nested pages, ledger and retained-failure roots charged; Reset frees source outer capacity. |
| model_converter.h:43–55 | Source packed var31/kind1, explicit deep entry copy constructor: scalar varKind + nested list token; source unsigned variable truncation preserved and Mk debug assertions occur AFTER entry publication. Implicit class-level owning copy/assignment was subsequently integrated; see SAT_MODEL_COPY_PORT_STATUS.md. |
| cpp:36–91 | Apply exact reverse traversal, blocked flip, other undefined assignment, separator handling, eliminated-var/debug postcondition checks; no invented full assignment. |
| 93–123 | CheckModel exact separator-based clause validation and source unterminated prefix behavior. |
| 125–131 | Mk and checked EntryAt/KindAndVar adapter; source vector capacities and placement/publication. |
| 133–164 | InsertClause, InsertBinary, InsertWrapper: real clause/wrapper bytes, contains-variable debug condition, literal-by-literal completed prefix, then null separator; InsertLiterals is explicit additional raw-array adapter. |
| 166–189 | CheckInvariant exact outer/ELIM later-entry/literal debug conditions, with real unconstructed/lifetime bounds guards. |
| 191–219 | DisplayInto real NumericTextStream, exact source punctuation/labels, inherited numeric state and failure prefix; one caller-owned result only at caller String boundary. |
| 221–226 | Copy appends ascending deep copies, preserving nested capacities; this is NOT assignment. Copy captures source begin/end once: finite self-copy without relocation is allowed and deep-copies only the captured range. Self-copy relocation preserves completed append and growth prefixes, then returns ErrSATSelfAppend at the actual dangling source-entry access; this differs from vector::append growing-size nontermination. |
| 228–235 | CollectVars invokes real charged satSolverUintSet.Insert in source entry order, including reserve/flag/push failure prefixes. AppendVars remains a separately named convenience adapter and is not the source method implementation. |
| 237–251 | MaxVar exact saved literals only, null separator skipped; entry var alone does not contribute. |
| util/vector.h:65–111 | Nontrivial entry growth publishes new outer pointer/old size BEFORE ascending deep entry copies and old-entry destruction. Retained-old directory/copy cursor are actual charged Go continuation metadata; reads of not-yet-constructed entries fail at access. Retry resumes; Reset/Close clean all real roots exactly once. This is not native C++ exception recovery. |

Literal store is a storage adaptation, not a complete new vector-template claim. Each live list owns a stable scalar descriptor with generation ledger retained across Free/Reset and exhaustion retirement. Logical literal capacities exactly2,3,5,8,...; source trivial reallocate copies full prior capacity; nested deep copy allocates exact source capacity and copies logical size only. NewList/CloneList/CopyList/Push/Reserve/Resize/Shrink/Read/Write/Free/Reset/Close never expose arena slices. The cold descriptor directory, both actual owner headers, Words payload/block metadata/directories/descriptor graph and every old+new peak are charged. Physical arena class/free-page retention is an explicit pooling adaptation: physical free storage remains charged until Close, independent of source logical vector capacity. Caller model/append buffers and test-only original formulas/oracles are caller-owned boundaries.

The13 groups test reverse patching/blocked flips/other-variable assignments, exact display, source capacities and full-capacity retained tails, cross-owner nested copies and append ordering, real allocator+wrapper overloads, stale/foreign/copied/closed handles, default literal construction, generation ledger, source var-bitfield/debug publication, independent32 chain-model cases, constructor/invalid-config unwind, exact physical graph Used accounting immediately after fault, and source first-literal publication before later insert failure. Fault sweeps occupy real unused pool blocks to ensure the second different-class copy actually requires a reservation after the first copy/destructor; warming is not mistaken for injection coverage. Scalar cursor retains completed construction/destruction order, and Close is tested at epoch exhaustion without namespace reuse. Warm Push/Shrink under FailAfter0 performs zero host allocations and no budget delta.

Frozen prerequisite snapshot: `.cache/release/sat-solver-model-final` = HEAD92c5d0a + ONLY these four Go files. Hashes are `.cache/sat-solver-model-frozen-hashes.txt`,13 actual named tests in `.cache/sat-solver-model-names.json`. Full native/checked-debug, named race/actual JS, vet/WASI and source/no-generics gates are running; terminal evidence will be appended without changing frozen code.

Current model/solver surface: explicit owning Clone/Assign now implement the implicit model class-copy actions, and the concrete common WATCH store is committed (see SAT_MODEL_COPY_PORT_STATUS.md and SAT_SOLVER_WATCH_STORE_PORT_STATUS.md); all actual solver methods/default inprocessing/search/conflict/scopes remain unimplemented. Parent owns real global memory/runtime accounting and generated module fallback integration. Existing satUintSet/satLiteralSet use uncharged host make/append and cannot simply become retained fully-budgeted solver storage. New satSolverUintSet supplies the actual charged SAT namespace uint-set dependency; a literal-set facade over this concrete owner remains follow-on source scope. No fake memory getter, renderer, extension, resource collector or solver is introduced.

## Required source CollectVars correction and six-file freeze

Parent review correctly required actual CollectVars allocation behavior instead of a convenience append-to-slice adapter. Namespace resolution is explicit: `sat::bool_var_set` aliases LOCAL `sat::uint_set` at sat_types.h:227. Its membership is `svector<char>`, followed by `svector<unsigned>` insertion order; global util/uint_set.h is a different bitmap algorithm and is not used here. `sat_solver_uint_set.go/test.go` were FIRST observed compile-RED for real missing owner/CollectVars APIs, then implementation. Its by-value iterator factory also had a FIRST compile-contract RED before implementation; escaping iterator values perform zero host allocations under FailAfter0.

The complete explicit sat::uint_set source body mapping is newSATSolverUintSet (implicit empty constructor), Insert, Remove, Assign/Clone, Contains, Empty, Erase, Size, captured-end by-value Iter/Next replacing begin/end const unsigned pointers, Reset, Finalize, Intersect and Union. Direct At and Stats are additional read-only adapters. Real membership byte and insertion-word arrays retain source 2,3,5,... capacities and FULL-capacity trivial reallocation. Reserve/Resize fills only newly logical membership slots with false AFTER all required growth expansions, preserving completed intermediate capacity prefixes. Reset resets both logical sizes and retains both physical arrays; Finalize frees arrays, Close additionally frees actual cold owner header. Both exact-capacity member copies destroy their own prior array before allocation in source field order; first copy failure retains old insertion array, second failure retains completed membership copy. Clone has deep independent ownership and unpublishes/unwinds its real cold owner on failure.

Insert reserves membership FIRST, reads duplicate flag, publishes membership true BEFORE insertion-list push. Thus a failed list expansion leaves a genuine membership ghost and duplicate retry does not append/repair it. Remove clears membership BEFORE search/debug assertion/back/write; a release ghost with a spare initialized tail follows the defined source tail-write/pop path, while actual empty/OOB accesses receive typed guards at access. Intersect/Union traverse insertion order and preserve completed prefixes. UINTMAX v+1 wraps to0 in Insert; source debug index assertion and actual release bounds danger are diagnosed at access, not by a blanket earlier uint domain restriction.

Set iterators hold one cold owner pointer plus scalar epoch/index/captured end as returned caller values, never per-entry pointers. Only insertion backing relocation/free invalidates them; Reset/pop retains scalar backing lifetime and a previously captured endpoint remains readable. Cold owner copies/closed services are guarded; epochs never wrap/reuse. Budget charges exact reflected header + actual char/uint capacities and all old+new peaks, never inferred membership count. Shared external Budgets still require owner confinement.

Final selected scope is20 named groups: prior13 plus actual CollectVars, source set order/operators, two-stage ghost/refusal prefixes, deep copy/member assignment failure and warm duplicate, bounded1000-event independent set model and UINTMAX access, value iterator retained lifetime/escaping zeroalloc, defined spare-tail ghost release versus debug behavior. Actual CollectVars tests begin with a prepopulated set, preserve source traversal/dedup, exercise membership publication before failing first list allocation, and exercise later variable-ID membership growth refusal after an earlier duplicate returns. This supplies real source algorithm and allocation coverage; convenience AppendVars is not used to stand in for it.

The earlier four-file13-group freeze/gates are historical evidence and superseded by the six-file20-group packet, with fullsolver compile-RED still excluded. New final hashes and terminal gates follow. Source implicit model-converter Clone/Assign remain explicitly OPEN, as approved by parent; no source header-wide owning copy completion is claimed.

- `src/sat/sat_types.h` SHA256: `d6577e8da7b2a07cb13959816adf5bbed512cf7049ebf5351ff600b7620e1ba2` (local uint-set and bool-var alias source).

## Supplemental nonmember/conditional solver inventory

The member-index generator does not cover the destructor/operator/helper spellings; these are explicitly audited too: solver::~solver at cpp:60, GC comparator call operators glue_lt1412/psm_lt1422/glue_psm_lt1432/psm_glue_lt1445, neg_literal::negate at3113, static prune_unfixed3157/back_remove3169/brute_force_consequences3181/core_chunking3193, and operator<<(mk_stat)3765. The static consequences helpers' only selected caller remains disabled by `if(false && asms.empty())`; no active-branch support is inferred from their definitions. Full implementations remain required according to selected source branch inventory.

Full stopwatch.h audit: selected Linux source uses CLOCK_PROCESS_CPUTIME_ID, with CLOCK_MONOTONIC only BSD fallback, and get_seconds on a running watch mutates sampling state through stop→start. A wall-time time.Now/time.Since implementation would change source clock domain. Windows performance-counter and Apple SYSTEM_CLOCK branches are distinct; Apple constructor visibly calls host_get_clock_service twice. Root runtime prerequisite must provide real selected clock/profile behavior; JS/WASI CPU-clock domains are explicit OPEN adapters rather than zero clocks or fabricated timing.

## Final six-file21-group prerequisite evidence

Freeze on HEAD92c5d0a includes the six Go files in `.cache/sat-solver-model-set-frozen-hashes.txt`; all live and isolated snapshot bytes were rechecked equal after gates. Final source char-vector arithmetic has an independent scalar regression that FIRST failed on widened growth: old1551591827 wraps next to179904093 in source uint32, whereas widened helper returned2327387741. Concrete membership helper now computes actual uint32 (3*old+1)>>1 and source header-byte overflow comparisons before actual host allocation bounds. No shared utility file changed. This adds the21st named group; prior13/20 counts are historical steps, not final coverage.

Terminal gates on `.cache/release/sat-solver-model-set-final`, HEAD92c5d0a plus ONLY six frozen prerequisite Go files, fullsolverRED excluded:

- Full native ./... session25984 exit0: g3 2.420s, mpnum23.170s, every package PASS.
- Full checked-debug ./... session85814 exit0: g3 2.744s, mpnum23.842s, every package PASS.
- Named21 checked-debug race session13719 exit0: exactly all21 RUN and PASS verified, g3 1.046s.
- Actual named21 checked-debug JS/wasm with minimal Node wrapper session79593 exit0: exactly all21 RUN and PASS verified, g3 0.176s. Runtime execution, not cross-compilation alone.
- Full checked-debug vet session9172 exit0.
- Full checked-debug WASI production ./... build session27324 exit0, outputs/cache/temp remain below golang.
- Source/no-generics audit session81353 exit0:2102units/677APIs,33ported/78partial/1991unported using committed snapshot metadata (no new central rows claimed).

Exact selected names and logs: `.cache/sat-solver-model-set-names.json`, `.cache/sat-solver-model-set-{native,checked,race,js,vet,wasi,audit}.log`. No CPP compiler/source file dependency is needed by ordinary tests, and no production math/big, unsafe, cgo, interfaces in hot records or Go generics were introduced. Root review/integration is next; code/tests held frozen. Entire solver is still OPEN and its actual registered finite-cycle source test remains RED, with concrete runtime/default-inprocessing/search/scopes dependency roadmap above.

| Frozen file | SHA256 |
| --- | --- |
| `sat_solver_nested_store.go` | `1e7e91ff218347dec9118d1eeb2bbe61033c4320f8c2af7a5cf3158ee7110ea1` |
| `sat_solver_nested_store_test.go` | `4bb872d8b582dba2f56d5dd5c7d75bfe6ee6e160c4a8d8bf14fd8601a1e0059a` |
| `sat_solver_model_converter.go` | `0ffb4d4bb9029b56830bf454461b8809d6ddb4a6dac6eec9e0b07a3501040ed0` |
| `sat_solver_model_converter_test.go` | `3bf1ef4c966fc7225ec7e906874307fbec6e479e47e455a02c4e78a44019e473` |
| `sat_solver_uint_set.go` | `2b08e9082fa714fcfcbfa5ae6c0a50585ac18c2476b6e0d8cffef8d40e2d778a` |
| `sat_solver_uint_set_test.go` | `f54af827d47ece89f4192a8cda8933d0df2e1edf046f9c04f1e4e2a7484279a6` |

### Final captured-end self-copy correction (23 groups)

This supersedes the historical 21-group freeze above. Two behavioral tests first failed: defined size1/cap2 self-copy and size2/cap3 completed-prefix relocation. Copy now captures the original source end, permits finite deep self-copy, and diagnoses dangling source-entry access only after completed outer relocation; it preserves any earlier append prefix. No source growing-range nontermination is attributed to model_converter::copy.

Frozen snapshot: `.cache/release/sat-model-self-copy-final`, HEAD `2a6859809b16cbbcdc96aae37891cd2fd20c39cd` plus the same six files. Exact inventory `.cache/self-copy-names.json` (23 groups), regex `.cache/self-copy-regex`, hashes `.cache/self-copy-hashes.txt`. Full native 18933, full checked/debug 86706, 23 named checked/debug race 78127, actual 23 named checked/debug JS 80380, full checked/debug vet 81889, WASI build 76751 and source/no-generics audit 81603 all completed exit0. Both actual runtime logs were checked for every exact RUN/PASS name. Terminal logs: `.cache/self-copy-{native,debug,race,js,vet,wasi,audit}.log`.

Changed hashes:
- `sat_solver_model_converter.go`: `7c064f993925a17ba1413600ee29b17ef71a1519b0a730db6fdcf9f3279c0470`
- `sat_solver_model_converter_test.go`: `833bfb01530a706d0f456377678c900140ed9b72d49d2c1dc3366e2ee2f56622`

Other four hashes remain unchanged from the six-file freeze. Implicit model copy/assignment still open; real solver registered-test API remains RED and excluded from this independently green prerequisite.

### Final immediate debug-assertion correction (24 groups)

This final freeze supersedes both historical 21- and 23-group freezes. A new behavioral RED proved Apply's DEBUG_CODE must assert immediately at each clause separator: after flipping blocked variable0, the first violated clause asserts before a later invalid variable5 can be read. Apply now uses a distinct source-order assertion traversal; CheckModel still accumulates false clauses and reaches the later bounds guard. The regression checks both routes, completed assignment prefix and immediate physical ownership.

Frozen snapshot `.cache/release/sat-model-debug-final`: HEAD `0ec768f76ae60179db4acbf4a2dff7becd20324b` plus the six prerequisite files. Exact24 inventory `.cache/model-debug-names.json`, regex `.cache/model-debug-regex`, hashes `.cache/model-debug-hashes.txt`. Full native36423, full checked/debug43525, exact24 checked/debug race67747 (1.051s), actual exact24 checked/debug JS53703 (0.179s), full checked/debug vet64257, WASI82529 and source/no-generics76508 all terminal exit0. Every exact RUN/PASS name was verified in both runtime logs. Logs `.cache/model-debug-{native,debug,race,js,vet,wasi,audit}.log`.

Final changed hashes:
- `sat_solver_model_converter.go`: `464def69c215b8f3c4131adabb2ac5f623b38b19450223c67342b9de1c77b79d`
- `sat_solver_model_converter_test.go`: `1b3d8d2158746aa96880496ca576141209e75c9a41a0e46c548fb6958352fdc7`

The other four hashes remain unchanged. No source assertion inventory was inflated: these24 groups are port-specific source-order/independent tests, not24 source ENSURE sites. This complete explicit-body prerequisite remains partial for implicit model clone/assignment and does not implement full solver search/propagation/runtime.

## Coordinator integration evidence

Coordinator independently read the selected CPP/header/local-set/vector bodies
and reviewed all six production/test files. Cross-agent source review caught
finite captured-end self-copy and immediate Apply DEBUG_CODE assertions;
both required behavioral RED regressions and source-order corrections before
integration. Final live bytes equal all six frozen hashes in the final24 table.
Coordinator independently matched exactly24 RUN/PASS events in both race and
actual JS logs. The full native/debug gates run on HEAD0ec768f plus exactly
these six files include the committed memory counter and expression-array fixes.

The final ledger was separately checked in
`.cache/review/sat-model-final24-0ec768f`: source fingerprint, no-generics AST
scan and implementation/test declarations passed at2102units/677APIs,
33ported/82partial/1987unported. Only model_converter.h/.cpp advance from
unported to partial; full sat_solver.cpp/.h remain unported. Whole-unit method
and ownership audit flags remain false. This ledger evidence is an inventory,
not proof of the original whole solver or interpolation objective.

Independent final source review confirmed both corrected routes and found no
further selected ordinary-method mismatch. Literal storage validates physical
capacity/tokens but does not track never-written spare-slot construction; its
verified retained paths use initialized source slots. It does not claim all
C++ initialized-lifetime domains are checked.


## Subsequent implicit model ownership integration

The actual implicit class copy constructor and assignment now have concrete
Clone/Assign operations. Outer copy uses exact source capacity and deep nested
copies; assignment destroys destination entries ascending before allocating,
then publishes the full header before ascending constructors. Interrupted
source growth guards undefined capacity reads at that point; partial exact
copies preserve constructed prefixes and explicit Go cleanup/restart. There is
no invented implicit move for the source class with its user-declared destructor.

Eight tests-first ownership groups join the original twenty-four. The immutable
32-group packet passed full native/debug, named race and actual JS, vet, WASI
compilation and source/no-generics checks; coordinator integration also passed
fresh full suites against the committed sparse/shared packet. Exact hashes,
source paths, assertions and failure-accounting limits are recorded in
SAT_MODEL_COPY_PORT_STATUS.md. Historical open-copy notes and 24-group evidence
above describe their earlier snapshots and are superseded by this packet.
Full actual solver algorithms and runtime integration remain open.
