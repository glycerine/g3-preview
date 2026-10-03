# Actual SAT simplifier source proposal

Read-only proposal at HEAD4b9337e. Only this document is written. No production,
permanent tests, existing owners, central metadata, or Git are changed.
The complete `sat_simplifier.h/.cpp`, generated-parameter input,
`solver` initialization/creation/assignment/propagation paths, cleaner,
and registered test consumers were read. This is a proposal for the real
consumer and its dependency closure, not an alternative SAT solver.

## Scope and source registrations

The literal-indexed outer occurrence list is committed13be17b. Its full explicit
selected LAZY methods and child/iterator actions are available; implicit
copy/move instantiations, nonlazy profile and STRACE remain separately open.
All remaining source simplifier methods below are currently unimplemented.
No directly registered SAT simplifier unit test exists: source-port ENSURE/VERIFY
count for this unit is zero. `TST(simplifier)` at test/main.cpp169 refers to the
Windows-only public Z3 AST simplifier tests, not sat::simplifier, and must not
be counted as SAT assertions. Actual SAT consumers include `sat_user_scope`
(TST239), the argument-driven `cnf_backbones` (TST_ARGV247), and solver use by
actual tactics/SMT integration. `check_assumptions` (TST197) is an SMT-context
consumer, not a direct SAT simplifier fixture.

The registered user-scope campaign is unbounded. A finite source outer cycle
has42 executions of one ENSURE site,168 created clauses, seven initial variables
(0 through6), and original push-three/pop-two actions over seven frames followed
by seven final pops. The existing uncommitted `sat_solver_user_scope_test.go`
retains that planned route but its actual runtime/solver APIs do not exist.
It must not become green using a truth-table production implementation, rebuilt
formula state in the incremental solver, unknown-only Check, or disabled defaults.
A test-only bounded CNF oracle remains independent from actual search/model
reconstruction. The cnf_backbones input-driven unit requires a separate finite
fixture/source-registration inventory and is not included in the42 claim.

## Complete default execution path

Constructor binds the actual solver, initializes num_calls0, performs UpdateParams,
then resets five statistics. Other source scalars are not silently initialized
before their actual assignment sites. The parent solver constructs its children
in header order and later calls its own UpdateParams again; the resulting child
parameter update routes and real current module snapshot calls must remain visible.

Run(learned) returns first for actual solver inconsistency, then only if all three
subsumption/blocked-elimination/resolution flags are false. Defaults are subsumption
true, resolution true, elim_vars true, blocked boolean false, blocked_at2.
Initialize clears need_cleanup, calls real cleaner(true), captures current trail
size AFTER that cleaner, reinitializes occurrence lists, resets clause/bin/var
worklists, then resets/resizes visited char storage to2*num_vars. Diagnostic
check_invariant is conditional source debug code, not a fake true callback.

Run then resets two todos AGAIN, calls cleaner(true) AGAIN, recaptures trail size,
clears need_cleanup, reinitializes occurrence lists AGAIN, registers learned
clauses first only when learned=true, then problem clauses. Registration stable
sorts each actual clause vector by DESCENDING size, preserving equal-size order;
it skips frozen clauses and inserts only strengthened clauses into sub_todo.
BCE executes on ordinary calls when enabled OR num_calls==blocked_at. Thus default
blocked boolean false still reaches BCE on the THIRD ordinary call; learned calls
do not increment num_calls. Counters and old_num_elim_vars are set after BCE.
Only then is source scoped_finalize installed. A failure before that guard is
not an excuse to run the later guarded cleanup as if already installed.

Each loop runs subsume if enabled, returns on conflict, runs elim_vars only for
ordinary/resolution calls, returns on conflict, stops when subsumption disabled
or signed sub_counter<0, otherwise repeats while sub_todo is nonempty. Source
scoped cleanup runs on exits after installation: if need_cleanup, compact watches,
clean learned clauses then problem clauses; otherwise only remove affected
learned clauses if vars were eliminated. It checks invariants and then Finalize.

Explicit Finalize order is use_list, sub_todo, sub_bin_todo, elim_todo, visited,
bs_cs, bs_ls. It does NOT explicitly finalize pos_cls,neg_cls,new_cls there.
Their ordinary member destructors run in reverse declaration order when the
whole simplifier dies. Normal completed elim_vars finalizes those three; its
checkpoint/counter early return skips that explicit finalization. These distinct
retained storage histories must be reflected in accounting and Close.

## All algorithm families and finite action obligations

- Clause and literal cleanup overloads differ: clause cleanup scans all literals,
  copies even true literals, erases false-literal occurrences when live/unfrozen,
  shrinks and returns satisfaction. Literal-vector cleanup returns immediately
  at its first true literal WITHOUT final shrink. Preserve both prefixes.
- Subsumes1 marks all c2 literals, allows at most one complementary missing c1
  literal, publishes null/single deletion output, then unmarks all c2 literals.
  Subsumes0 requires direct inclusion. Min-occurrence choice is first strictly
  smaller count, with source unsigned sums. Approximate subset is only a filter.
- Collectors use actual compressing occurrence iterators, size/approx filters,
  signed cost deductions, then paired output pushes. Clause output push can
  succeed before a literal push fails. BackSubsumption1 handles deletion or
  literal elimination, converts learned subsumer into problem clause when needed,
  increments source stats and stops on actual conflict. BackSubsumption0 remains
  reachable when new-resolvent sub_counter<=0; it is not redundant dead code.
- PropagateUnit captures old trail size, actual Assign, then ONLY
  PropagateCore(false), since clause vectors/watches may temporarily be inconsistent.
  For each new trail literal it inserts false-literal occurrences into sub_todo,
  advances iterator before mutations, erases satisfied occurrences except current
  literal, then resets that positive occurrence list. A no-op propagation callback
  cannot implement this route.
- ElimLit publishes need_cleanup/stat/elim_todo before source clause elimination,
  physically erases its occurrence, deducts the remaining count/2, cleans clause,
  then handles satisfied/empty/unit/binary/long cases in source order. Binary
  creation precedes bin_todo publication and lazy removal.
- Binary subsumption starts at one real source RNG draw and rereads CURRENT watch
  size in its inner indexed loop because backward subsumption can add binaries.
  It ignores the false return of SubsumeWithBinaries at the caller's source site.
  Learned-to-problem marking affects reciprocal watch entries.
- ElimDupBins stable-sorts by the actual bin_lt: binaries first, literal index
  ascending, nonlearned before learned for identical binary literal, nonbinaries
  stable. Existing SortByKind is NOT this comparator. Compact both directions,
  then add eliminated halves/2 to the uint32 statistic.
- Subsume pops binary todo from the back, updates dummy, then processes clause-set
  Erase() in the real selected hash physical order. It unmarks strengthened state,
  deducts counters, conditionally cleans against new trail, handles empty/unit/bin,
  and performs backward subsumption. Do not substitute a Go map/worklist order.
- BCE's local heap comparator reads live opposite-literal occurrence count and
  CURRENT watched-vector length: weight=2*occ(~l)+watches[l].size(). Initial heap
  bounds128, insert reserve idx+1, decreasing priorities call real heap Decreased
  or Insert. Its process traverses long occurrences first, collects removals,
  publishes MC BLOCK_LIT entry on first blocked clause, stores original clauses,
  updates affected heap priorities, unmarks, then lazily removes collected clauses.
  Binary phase deletes reciprocal half before current compaction and writes MC.
  AllTautology filters nonlearned reciprocal binaries and checks complementary
  literals against actual simplifier marks. External/eliminated vars are excluded.
- Variable ordering collects eligible vars from real uint_set order, computes
  source uint32 wrapped 2*pos*neg+pos*bin_neg+neg*bin_pos, resets elim_todo before
  stable sorting increasing cost, and pushes a separate variable output vector.
- Resolve marks/pushes c1 except pivot, scans c2 except opposite pivot, detects
  complementary marks, suppresses only literals marked from c1, and unmarks c1.
  It DOES NOT mark each appended c2 literal, so duplicate raw c2 literals need
  exact finite tests rather than an invented global dedup algorithm. Signed
  elimination counter is charged for both input sizes before each scan.
- TryEliminate preserves every cutoff, before-literal totals and clause-count
  range boundary. It first computes all resolvents and refuses if non-tautological
  count exceeds original clauses. Accepted path creates MC ELIM_VAR and copies
  originals, marks eliminated, removes binaries and lazy occurrences, resets both
  occurrence lists, deducts the SAME cost a second time, then recomputes all
  resolvents. This is source's TWO genuine Resolve campaigns, not an optimization
  to one replay-less campaign. New resolvents can yield conflict, real unit BCP,
  binary creation+back-subsumption, or owned new clause→problem vector→occurrences
  before backward subsumption. No whole-elimination atomic staging.

## Concrete current storage and missing dependencies

Available actual owners: clause allocator/tmp_clause/flags/literals/approx sets;
clause and binary/wrapper vectors; clause_set, uint_set/literal_set; outer use lists;
watch store with initialized lifetime metadata; model converter with actual
ELIM_VAR/BLOCK_LIT entries and InsertClause/InsertBinary/InsertWrapper;
activity domain/var_queue; ParamsContext/GlobalParams/ParamDescrs; reslimit;
Statistics; raw utilMemoryRuntime and source RNG. These are storage/service
primitives, not an already implemented solver.

Required real solver boundary methods are direct calls on ONE concrete satSolver,
not an interface of fake callbacks: NumVars, ValueVar/ValueLiteral, IsExternal,
WasEliminated, GetWatch, Assign/AssignCore, SetConflict, MkBinClause,
AttachTerClause/AttachNaryClause, DelClause, PropagateCore/Propagate,
Checkpoint, CheckInvariant, and access to actual trail, watches, clauses,
learned, eliminated flags, stats, RNG and model converter. Removed long watches
must still be handled by source PropagateCore's live clause tests, since it is
called mid-simplifier. New watch insertion can grow another list while a captured
iterator traverses the current physical backing; no copied-slice view substitutes.

The source MkVar sequence has independent growth prefixes: two new watch objects,
two assignment entries, justification,decision,eliminated,external,activity,level,
mark,two literal marks,phase,previous phase,assigned_since_gc, actual decision
queue callback, then simplifier elim_todo Insert. No AoS growth transaction is
source-equivalent. These columns and root solver owning structure do not exist.
The source outer watch-vector object/literal index directory still needs actual
scalar specialization binding the committed watch store; do not allocate one Go
heap watch owner per literal. Clause wrapper/bin storage is already concrete.

Cancellation Checkpoint must use actual reslimit.Inc, reset MC and mark model
noncurrent BEFORE source cancellation exception. Every tenth enabled checkpoint
queries actual memory::get_allocation_size and checks real main max_memory.
Raw utilMemoryRuntime is now real selected malloc/realloc/counters storage, but
its own comment explicitly leaves runtime-wide typed-owner allocation hooks and
generated global initialization/finalization OPEN. Existing arena/SAT owners do
not automatically call it. Budget.Used cannot stand in for global source allocation
size, and directly allocating dummy raw blocks alongside real arrays would double
storage rather than integrate actual hooks. Coordinator must approve the real
cross-owner allocation binding before full runtime checkpoint claims.

Cleaner is a required actual dependency, not optional default simplification.
It captures trail size BEFORE propagation, propagates, checks inconsistency,
compares previous unit count to the OLD captured size, honors force/cost gate,
cleans watches/problem/learned clauses and propagates again. Its empty/unit/binary
frozen-clause branches use real conflict/assign/MkBinClause/DelClause and source
stats. BCP calls Cleaner.Dec, AsymmBranch.Dec and Probing.Dec in exact order.
Asymm/probing constructor scalar counters/decrement methods must be real source
methods, even before their separate full Run algorithms are available.

Child params are not all main config. Full solver descriptor collector order is
main24→simplifier17→asymm3→probing0(TODO)→SCC1=45. Existing committed collector
has main24 only. Simplifier default updates17 fields in source assignment order,
which differs from generated descriptor order (elim_vars last versus descriptor
position15). It obtains actual current SAT module snapshot via generated getter
helper, uses local→module→literal fallback, then destroys helper refs lexically.
Asymm clamps branch limit to INT_MAX after its three assignments. SCC reads one
bool. Probing has five LOCAL-only getters and the actual typo
`probing_chache_limit`, converting1024MB using selected32/64-bit source size_t.
These real policy/collector methods are a coherent missing constructor dependency,
not a replacement for the simplifier algorithms.

## Proposed typed APIs and ownership contract

Names for coordinated implementation (not existing declarations):

- `newSATSimplifier(s *satSolver,p *ParamsRef)(*satSimplifier,error)` binds actual
  parent owner; `Run(learned bool) error`, `UpdateParams(*ParamsRef) error`,
  `InsertElimTodo(satBoolVar) error`, `ResetTodos() error`, `Finalize() error`,
  `ResetStatistics() error`, `CollectStatistics(*Statistics) error`, `Close() error`.
- `collectSATSimplifierParamDescrs(*ParamDescrs) error` and actual full
  `collectSATSolverParamDescrs(*ParamDescrs) error` compose the45 source entries.
  Generated getter helpers preserve real current snapshots and source fallbacks.
- Private methods retain every source overload explicitly named for Clause versus
  LiteralVector, Var versus Literal and BackSubsumption binary versus long clause.
  No exported manager/value raw borrowing, fakeChildSolver, or test-snapshot
  injection API. Source bool-return helpers return `(bool,error)` and output refs
  publish in exact statement order. ScopedFinalize is an explicit lexical cleanup
  call installed only at the actual source point, not a method-entry defer.

One actual charged cold simplifier header borrows parent solver and budget;
separate actual source member owners account their headers/backings once. Hot
visited storage is uint8 scalar, todos/wrappers/bin refs are concrete fixed-width
records, and literal-indexed metadata remains pointer-free in actual arrays.
Retained scalar initialized bits distinguish source constructor assignments from
Go zero defaults. Member Finalize versus owner Close histories remain distinct.
Budget includes all real headers, backings, directories, pooled payloads,
old/new growth overlap, local variable/cost arrays, BCE heap and to_remove,
source var-order output, and stable-sort scratch. No source phantom stack-charge
or hidden escaping comparator closure.

BCE requires an actual concrete heap specialization with parent-bound comparator
reading current use-list/watch owners (or a separately approved actual cold
comparator binding to existing IntHeap). No constant/integer-default comparator
or replacement sorted worklist. Heap constructor128 and all source Reserve/
Insert/Decreased/EraseMin actions are retained. Source uint32 costs and signed32
counter conversions/subtractions need explicit selected-host arithmetic policy.
Most counter -= expressions have an unsigned size/cost operand, so C++ performs
uint32 subtraction and converts back to signed32; they are not universally
signed-overflow UB. Preserve selected signed-conversion behavior and uint32
wrap. Pure signed operations require a separate actual-domain audit before any
genuine UB guard, not arbitrary precision arithmetic or a blanket smaller cutoff.

Stable sorts need a fully budgeted concrete library adapter for clause-size,
bin_lt and var-cost comparators; existing watch kind partitions cannot replace
these orders. Proposed indexed stable merge with charged scalar workspace makes
external C++ library scratch behavior an explicit adapter, not a source Z3
algorithm change or a claim of identical comparison/allocation profiles.
Source equal-key order must remain exact; all workspace bytes and host/source
extents are checked. Comparator reads use real source clause callbacks.

Refusal/consumed-error cleanup must preserve completed AST-independent roots and
statement prefixes. New clause insertion failure cannot silently delete a
source-owned already published clause; a real pending root/publication result
may be necessary, with coordinator approval after behavioral RED. Refused
lexical destructors retain real cleanup continuations rather than discarding
borrowed tokens. Real source callbacks must not be replayed to fabricate rollback.

## Tests-first dependency and packet sequence

1. Implement actual constructor policies and full45 descriptor closure in NEW
   sat_simplifier_params*,sat_asymm_params*,sat_scc_params*,sat_probing_params*
   namespaces after naming approval. Observe API RED with actual registry/owners,
   then source defaults/local-module-hidden policies, snapshot freshness,
   source assignment prefixes, duplicate descriptor order, asymm clamp, typo,
   MB wrap/width, lexical ref cleanup and complete cold budget tests. Do not label
   this configuration packet as implemented solver/simplifier algorithms.
2. Coordinate actual sat_runtime*/sat_solver_state*/sat_solver_clause_core*/
   sat_solver_propagate*/sat_solver_cleaner* closure and child constructor/member
   ownership before exposing NewSATSolver as usable for Check. Source constructor
   field order, actual MkVar prefixes, clause normalization/creation/attachment,
   assignment/conflict/trail/BCP and full cleaner are tests-first independently.
   No missing algorithm may be supplied as a no-op just to link constructor calls.
3. Implement complete actual simplifier remaining methods in NEW
   sat_solver_simplifier.go,_subsumption.go,_blocked.go,_resolution.go,
   _storage.go,_params.go plus source/independent/fault tests. Existing owners or
   solver integration edits require explicit coordinated ownership. Test Run
   uses the actual concrete core and all defaults: ordinary/learned initialization,
   third-call BCE, learned-to-problem conversion, strengthened todos, unit cascades,
   every subsumption/resolution/BCE branch, model-conversion history and restore.
   Direct private helper fixtures remain actual method tests, not a fake Run.
4. Bind default SCC/elim_eqs/probing/asymm Run algorithms, then full CDCL/search,
   restart/GC/assumptions/user scopes and model restoration. Only this closure can
   satisfy registered sat_user_scope Check and the full intended solver goal.

Simplifier-added finite oracles: exhaustively evaluate small input/output CNFs;
for variable elimination compare existentially projected original assignments
against resolvents and actual MC reconstruction; for BCE validate MC extension;
for subsumption/literal elimination compare exact models. Test-only oracle never
controls production. Include pure/empty/single/binary/long clauses, frozen/removed/
strengthened, external/eliminated/assigned vars, learned paths, duplicate finite
raw literals, equal-cost/order ties, reciprocal bins and both-sided blockers.
Source assertions are tested at their real sites in actual debug mode; physical
bounds/lifetime/overflow adapters are separately labeled.

For every allocation refusal check immediate allocator-live clauses, use-list
Words.Live, worklist/payload roots and budget before rendering/Close; include
completed-prefix cases and all lexical exit points. Verify no unowned new clause
or MC entry, no destructor callback replay, and retained copied-tail reads only
where source lifetimes permit. Warm allocation invariants AND cache-only measured
realistic Run/initialization timings with exact owned graph are required before
storage changes are justified. Final immutable bytes receive full native/debug,
meaningful race/actual JS named registrations, vet/WASI/no-generics/source audit.

No implementation is authorized by this document. All files except this proposal
remain untouched. Default simplifier, runtime hooks, SCC/probing/asymm/CDCL,
proof/tactic/SMT integrations and fullsolver Check remain OPEN.

## Exact source definition and assertion inventory

All qualified definitions from the source file (line references below); constructors/destructor and header-only/nested helpers are listed separately.

- cpp:29: `void use_list::init(unsigned num_vars)`
- cpp:35: `void use_list::insert(clause & c)`
- cpp:42: `void use_list::erase(clause & c)`
- cpp:49: `void use_list::erase(clause & c, literal l)`
- cpp:69: `inline watch_list & simplifier::get_wlist(literal l)`
- cpp:71: `inline watch_list const & simplifier::get_wlist(literal l) const`
- cpp:73: `inline bool simplifier::is_external(bool_var v) const`
- cpp:75: `inline bool simplifier::was_eliminated(bool_var v) const`
- cpp:77: `lbool simplifier::value(bool_var v) const`
- cpp:79: `lbool simplifier::value(literal l) const`
- cpp:81: `inline void simplifier::checkpoint()`
- cpp:83: `void simplifier::register_clauses(clause_vector & cs)`
- cpp:97: `inline void simplifier::remove_clause_core(clause & c)`
- cpp:107: `inline void simplifier::remove_clause(clause & c)`
- cpp:112: `inline void simplifier::remove_clause(clause & c, literal l)`
- cpp:117: `inline void simplifier::remove_bin_clause_half(literal l1, literal l2, bool learned)`
- cpp:123: `void simplifier::init_visited()`
- cpp:128: `void simplifier::finalize()`
- cpp:138: `void simplifier::initialize()`
- cpp:151: `void simplifier::operator()(bool learned)`
- cpp:206: `void simplifier::scoped_finalize_fn()`
- cpp:230: `void simplifier::cleanup_watches()`
- cpp:254: `void simplifier::cleanup_clauses(clause_vector & cs, bool learned, bool vars_eliminated, bool in_use_lists)`
- cpp:319: `void simplifier::mark_all_but(clause const & c, literal l)`
- cpp:328: `void simplifier::unmark_all(clause const & c)`
- cpp:337: `bool_var simplifier::get_min_occ_var(clause const & c) const`
- cpp:357: `bool simplifier::subsumes1(clause const & c1, clause const & c2, literal & l)`
- cpp:387: `void simplifier::collect_subsumed1_core(clause const & c1, clause_vector & out, literal_vector & out_lits,`
- cpp:412: `void simplifier::collect_subsumed1(clause const & c1, clause_vector & out, literal_vector & out_lits)`
- cpp:421: `void simplifier::back_subsumption1(clause & c1)`
- cpp:451: `void simplifier::back_subsumption1(literal l1, literal l2, bool learned)`
- cpp:460: `literal simplifier::get_min_occ_var0(clause const & c) const`
- cpp:479: `bool simplifier::subsumes0(clause const & c1, clause const & c2)`
- cpp:502: `void simplifier::collect_subsumed0_core(clause const & c1, clause_vector & out, literal target)`
- cpp:523: `void simplifier::collect_subsumed0(clause const & c1, clause_vector & out)`
- cpp:532: `void simplifier::back_subsumption0(clause & c1)`
- cpp:553: `bool simplifier::cleanup_clause(clause & c, bool in_use_list)`
- cpp:588: `bool simplifier::cleanup_clause(literal_vector & c)`
- cpp:608: `inline void simplifier::propagate_unit(literal l)`
- cpp:643: `void simplifier::elim_lit(clause & c, literal l)`
- cpp:682: `bool simplifier::subsume_with_binaries()`
- cpp:716: `void simplifier::mark_as_not_learned_core(watch_list & wlist, literal l2)`
- cpp:727: `void simplifier::mark_as_not_learned(literal l1, literal l2)`
- cpp:747: `void simplifier::elim_dup_bins()`
- cpp:809: `void simplifier::subsume()`
- cpp:1078: `void simplifier::elim_blocked_clauses()`
- cpp:1085: `unsigned simplifier::get_num_non_learned_bin(literal l) const`
- cpp:1097: `unsigned simplifier::get_to_elim_cost(bool_var v) const`
- cpp:1116: `void simplifier::order_vars_for_elim(bool_var_vector & r)`
- cpp:1150: `void simplifier::collect_clauses(literal l, clause_wrapper_vector & r)`
- cpp:1177: `bool simplifier::resolve(clause_wrapper const & c1, clause_wrapper const & c2, literal l, literal_vector & r)`
- cpp:1217: `void simplifier::save_clauses(model_converter::entry & mc_entry, clause_wrapper_vector const & cs)`
- cpp:1225: `void simplifier::add_non_learned_binary_clause(literal l1, literal l2)`
- cpp:1256: `void simplifier::remove_bin_clauses(literal l)`
- cpp:1287: `void simplifier::remove_clauses(clause_use_list const & cs, literal l)`
- cpp:1301: `bool simplifier::try_eliminate(bool_var v)`
- cpp:1459: `void simplifier::elim_vars()`
- cpp:1482: `void simplifier::updt_params(params_ref const & _p)`
- cpp:1503: `void simplifier::collect_param_descrs(param_descrs & r)`
- cpp:1507: `void simplifier::collect_statistics(statistics & st) const`
- cpp:1515: `void simplifier::reset_statistics()`

Additional complete header/nested methods: simplifier ctor59/dtor65;
use_list mutable/const get+finalize; mark_visited/unmark_visited/is_marked;
size_lt; insert_elim_todo; reset_todos; scoped_finalize ctor/destructor;
bin_lt; bool_var_and_cost_lt; blocked_clause_elim ctor,insert,process_var,
operator(),process,all_tautology; its literal_lt ctor,weight,operator();
queue ctor,insert,decreased,next,empty; the three report ctor/destructors
(blocked/subsumption/elim_var) with actual source verbosity/stopwatch paths.
Verbose stream formatting/stopwatch profiles remain explicit external obligations;
verbosity0 is a valid selected profile, not disabling algorithm defaults.

Actual cpp contains15 SASSERT sites,4 CASSERT sites,zero ENSURE/VERIFY sites.
Their exact source predicates/sites are:

- cpp:118: `SASSERT(s.get_wlist(~l1).contains(watched(l2, learned)));`
- cpp:148: `CASSERT("sat_solver", s.check_invariant());`
- cpp:161: `CASSERT("sat_solver", s.check_invariant());`
- cpp:169: `CASSERT("sat_solver", s.check_invariant());`
- cpp:222: `CASSERT("sat_solver", s.check_invariant());`
- cpp:303: `SASSERT(!c.frozen());`
- cpp:394: `SASSERT(!c2.was_removed());`
- cpp:425: `SASSERT(m_bs_cs.size() == m_bs_ls.size());`
- cpp:507: `SASSERT(!c2.was_removed());`
- cpp:699: `SASSERT(wlist[j] == w);`
- cpp:898: `SASSERT(!m_queue.contains(idx));`
- cpp:908: `literal next() { SASSERT(!empty()); return to_literal(m_queue.erase_min()); }`
- cpp:964: `SASSERT(c.contains(l));`
- cpp:1155: `SASSERT(r.back().size() == it.curr().size());`
- cpp:1165: `SASSERT(r.back().size() == 2);`
- cpp:1179: `SASSERT(c1.contains(l));`
- cpp:1180: `SASSERT(c2.contains(~l));`
- cpp:1245: `SASSERT(it2 != end2);`
- cpp:1292: `SASSERT(c.contains(l));`

Header contains no additional SASSERT/CASSERT sites. Source diagnostic calls must execute only at the selected real profile sites; a true-valued fake invariant is not a port.
