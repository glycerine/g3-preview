# Arithmetic declaration plugin: proposed concrete runtime packet

Read-only proposal, held for coordinator review. No Go implementation or test has been written for this packet. The target remains the complete `src/ast/arith_decl_plugin.h` and `.cpp`, including their actual rational, algebraic, recognizer, utility, and owning `app_ref` specializations. Installing only the rational declarations must not be reported as completing this source unit.

## Existing prerequisites and integration boundary

The real AST owner, all five node kinds, parameter/info copying, scalar handles, AST interner, reference owners, symbol service, rational service, sexpr owner, charged IDs, MPBQ/MPBQI, and resource-limit utility now exist. They are usable prerequisites rather than substitute AST or numeric implementations.

Current AST storage has `plugins [5]astPluginBinding`; dispatch in `ast_core_plugin_dispatch.go` and value operations in `ast_builtin_plugins_values.go` reject `fid >= 5`. The constructor interns the `arith` family but deliberately does not install its plugin. It has no arithmetic plugin cold-state field. Source `ast_manager::register_plugin` (`ast.cpp:1635`) publishes registry membership before calling `set_manager`. These existing owner/dispatch fields require coordinated changes; a new-file-only implementation cannot honestly provide integration through a global side table.

The source lazy wrapper (`arith_decl_plugin.cpp:25–113`) requires an independent unsynchronized MPQ manager, a real algebraic-number manager borrowing the manager's resource limit, a charged ID generator, and a concrete owning algebraic numeral vector. There is no current algebraic-number, upolynomial, or sexpr-to-upolynomial Go implementation. The AST owner also does not yet embed/expose its source resource limit. Those are genuine prerequisites for `am`, `aw`, irrational numerals, root parsing, algebraic comparison, deletion, and translation. No stub algebraic token or unsupported callback may stand in for them.

## Proposed public API and private layout

All methods remain on the real `ASTManager` and acquire its lock; private helpers run under that lock. Proposed names are explicit and concrete:

- `InstallArithmeticPlugin() error`, `ArithmeticPluginInstalled() (bool,error)`, `ArithmeticFamilyID() (int32,error)`.
- `ArithmeticIntSort()/ArithmeticRealSort() (ASTHandle,error)`; `ArithmeticNumeral(mpnum.OwnedRational,bool) (ASTHandle,error)`; `ArithmeticPi()/ArithmeticE() (ASTHandle,error)`.
- Existing `MkFamilySort`, both family function-declaration routes, `MkFamilyApp`, value/unique/equal/distinct/some-value routes dispatch to the actual installed arithmetic state. Explicit selector overloads may be exposed as `ArithmeticFuncDecl` and `ArithmeticFuncDeclFromArgs` for source branch tests, using the existing concrete parameter owner and borrowed handle arrays.
- `ArithmeticNumeralValue(h, *mpnum.OwnedRational, *bool) (bool,error)` copies the source rational output before publishing the integer flag. A failed recognition leaves both outputs unchanged; failed copying preserves its actual source prefix.
- Concrete expression/sort/decl kind predicates and unary/binary matching methods cover every actually declared recognizer overload. Matching uses caller outputs, preserving false-output behavior. `ArithmeticIsZero`, `ArithmeticIsMinusOne`, and `ArithmeticTimesMinusOne` use real temporary rational actions.
- `ArithmeticApp(kind,args)`, explicit unary/binary/three-argument convenience wrappers, `ArithmeticEq`, `ArithmeticAddSimplify`, and `ArithmeticMulSimplify` cover the real utility routes. Owning simplify results use real `ASTRef`; owning arithmetic operator counterparts use a concrete checked app-reference specialization, with constructor/return/destructor actions rather than raw borrowed-handle substitution.
- `ArithmeticGetSortNames(output,logic)` and `ArithmeticGetOpNames(output,logic)` append into a concrete charged builtin-name owner. Each element contains actual `Symbol`, signed family ID, and declaration kind; no strings, interfaces, or record pointers. The output vector's source growth/copy/header arithmetic is audited independently of actual Go stride.
- Once the real backend exists: `ArithmeticAlgebraicNumeral(OwnedAlgebraicNumber,bool)`, `ArithmeticRootNumeral(SexprHandle,uint32)`, algebraic copy/accessor methods, and concrete external-parameter translation between two actual owners. The algebraic type is a prerequisite contract to design in its own source packet, not a proposed opaque placeholder here.

Private `astArithmeticPlugin` is a single charged cold object bound to its real AST owner/family ID. It contains three actual Symbol fields (`Int`, `Real`, `RootObject`), named scalar node fields for every source cached sort/declaration/app, two concrete scalar small-numeral vectors, the sticky conversion flag, lifecycle/continuation metadata, and an optional pointer to the one real cold algebraic wrapper. Hot cache entries contain only actual scalar `astNodeRef`; no per-entry pointer or boxed value. A coordinator-owned field holds the cold plugin. Header reservation occurs before escaping owner allocation. Manager stats account that header, every full backing capacity, old/new growth overlap, and the real lazy numeric graph separately.

Use all named cached declaration fields from the source, not an enum array whose convenient traversal silently changes constructor/finalizer order. Cache vectors use the actual `ptr_vector<app>` source allocation schedule and bounds; source `inc_ref` precedes `setx` growth. Failed growth retains the unlisted increment in the manager's real graph. Finalize releases only actual cached entries, then manager forced teardown reclaims retained nodes. No invented failed-push rollback.

The optional algebraic wrapper owns its independent rational manager, real algebraic manager, embedded charged IDs, and typed algebraic records/vector. Its members destroy in reverse declaration order. Source `mk_id` publishes the ID action before numeral-vector reserve/copy; `recycle_id` recycles before deleting the numeral. External parameters retain their source integer IDs, distinct from allocator slots/generations; actual algebraic values live in the wrapper. Closed/foreign scalar handles are checked at API boundaries.

## Complete source-method inventory

| Source surface | Concrete implementation/actions required |
| --- | --- |
| Constructor, `set_manager`, `finalize`, destructor, `mk_fresh` | Real symbols, exact eager cached nodes/pins/flags, source field order; fresh plugin starts empty and does not clone caches; lazy wrapper destruction plus reverse member cleanup. |
| `aw`, `am`, wrapper ctor/dtor, `mk_id`, `recycle_id`, `idx2anum`, `to_anum` | Real algebraic backend and source ID/vector actions; scalar private access checked at borrowed lifetime boundaries. |
| `del`, `translate` | Actual external-parameter recycle and cross-owner deep algebraic copy; source callback availability follows registry lifetime. |
| `mk_sort` | Real/Int cached sorts; parameters ignored; unknown kind returns null. |
| Private declaration selector, `fix_kind`, `check_arity`, `mk_num_decl` | Full switch, unary SUB becomes UMINUS; retain unused `check_arity` body without inventing calls; exact NUM parameter-kind/arity checks. |
| Both `mk_func_decl` overloads | NUM first, then zero-arity error except PI/E; coercion selector scans any Real only for its exact operation set, otherwise first domain/argument; ignore range/non-NUM parameters as source does. |
| Three numeral overloads, `mk_pi`, `mk_e` | Rational cache/general construction, real algebraic branch/conversion, actual sexpr polynomial root parsing, cached constants. |
| `get_sort_names`, `get_op_names` | Append order, symbol construction before output growth, sticky NRA conversion state, no fabricated underspecified names. |
| `is_value`, `is_unique_value`, `are_equal`, `are_distinct`, `get_some_value`, `is_considered_uninterpreted` | All base short circuits and algebraic branches, four ordered ADD/nonzero checks, exact sorts, three underspecified kinds. |
| Recognizer constructor/family accessor and all header predicates/matches | Explicit concrete expr/sort/decl routes, actual owned numeric copies, unary/binary arity tests and false-output behavior. |
| Utility ctor, manager/plugin access, `init_plugin`, arithmetic manager/accessors | Borrow the installed real plugin lazily; utility construction does not install one. Missing source plugin is a checked Go domain adapter. |
| All utility constructors and simplify overloads | Actual generic AST dispatch, exact singleton-app shortcut versus singleton-variable route, owning simplify returns, equality orientation by numeral and AST ID. |
| All inline `app_ref` numeral/arithmetic/comparison operators | Concrete owning app specialization; real numeric temporaries and full-expression references, copies, and cleanup. No generic dispatcher. |

## Source quirks and exact failure actions

1. `set_manager` eagerly builds Real then Int and every predicate/arithmetic/conversion/power/abs/trigonometric/constant/underspecified cached declaration. Domains/ranges/chainability/associativity/commutativity are real AST info, not recognizer metadata. Constructor and finalizer order differ: finalize releases MOD before REM, Int power before Real power, and cached small Int entries before Real entries.
2. Public declaration selection does not validate the selected declaration's expected arity or require every domain to be numeric. Source application checking/coercion performs later checks. Generic irrational kind is not a constant operation: zero arguments produce the ordinary no-arguments error; nonzero unknown selector returns null.
3. Direct NUM declarations accept any rational parameter when the second parameter is any nonzero signed integer. Unlike rational numeral construction, this route does not reject a nonintegral value tagged as integer.
4. Small cache includes unsigned values below 16. With sticky NRA conversion, an integer 15 becomes cached Real with flag 0; integer 16 becomes Real with the original flag 1. Recognition can therefore report integer despite a Real sort. `get_some_value(Int)` also follows the sticky conversion route. Do not canonicalize these quirks.
5. `get_sort_names` for NRA/QF_NRA/QF_UFNRA sets the conversion flag before append and never clears it for later logics. Normal sort append is Int then Real. Op names have their exact baseline and ALL/null extension; no underspecified names are exported.
6. Source plugin registration publishes membership before initialization. Failures retain completed source initialization prefixes with an owned continuation, so cleanup releases only completed pins once. Cache INC-before-growth prefixes remain until actual explicit balancing or manager forced teardown.
7. Real app coercion is necessary. `ast.cpp:2038–2098` allocates candidate app storage before determining/coercing arguments, uses actual TO_REAL/TO_INT apps and source inline16 owning buffer callbacks, and then checks/publishes. A plugin-local sort rewrite is not equivalent.
8. Source numeric parameters own real rational copies; temporary parameter arrays and info copies retain their lexical destruction and fault prefixes. Expression results never borrow a numeric handle past its owning statement.
9. Source debug checks occur at actual sites (external kinds, algebraic IDs/nonzero values, some-value sort, recognizer accessor kind, lazy utility plugin). Trace/SMT2 printer actions remain open until actual printers exist; g3checked alone does not claim these source actions.

## Required coordinated owner edits

The coordinator must approve/own existing AST field and plugin-binding integration, generic dispatch routes, manager graph stats, finalize/deallocate ordering, external-parameter callbacks, actual resource-limit field/accessor, and app coercion. This proposal does not authorize this agent to mutate those files. Constructor defaults remain the actual five eager plugins; installation is explicit. Format-manager/plugin-copy behavior follows real source registration, not automatic arithmetic installation.

A private concrete registered-plugin record may extend the current binding mechanism; it must preserve source registry growth/publication and permit later plugin families. Merely increasing a hard-coded bound to six and calling that the complete generic plugin registry would be a false scope claim. Exact selected arithmetic binding and full general registry coverage must be distinguished.

## Source assertions and tests-first plan

There is no dedicated registered arithmetic declaration-plugin test found. `src/test/main.cpp:176,196` registers arithmetic simplifier and rewriter consumers. `arith_rewriter.cpp` invokes actual parser/rewriter/normalization paths: its parser VERIFY and nonempty-assertion ENSURE execute per parsed example and cannot be claimed ported by standalone plugin fixtures. `var_subst.cpp` and algebraic/nlarith/QE consumers similarly require their real algorithms. Existing AST registered assertion totals remain unchanged.

After API/integration approval, observe missing-API RED before production for the following additional finite fixtures:

- Real installation and every cached declaration/sort/app flag, owner pin and exact finalize sequence; fresh-state and installation failure prefixes.
- Rational numerals 0/15/16/negative/large/fractional; cache reuse, warm allocation-free hits, INC-before-cache-growth faults; direct fractional NUM with nonzero flag; sticky NRA/normal logic sequence and recognizer output flags.
- Both declaration overloads, unknown/zero-arity/NUM branch order, ignored range/parameters, unary SUB, any-Real versus first-domain selection, and actual mixed-argument casts with candidate-before-coercion failure accounting.
- Every recognizer/matcher false-output route and copied rational failure; all utilities including singleton VAR versus APP, equality ID orientation, simplify owning results and explicit app-reference operators.
- Names append order and symbol-before-vector failure prefixes, three uninterpreted kinds, base unique-value shortcuts, all four ADD-distinct cases.
- Actual algebraic rational/irrational/root creation, ID recycle and reserve/copy failures, equality/distinctness, cross-manager external translation, deleting parameters while plugin live versus registry gone. These tests await the real backend, not invented stand-ins.
- Immediate whole graph/root/Words accounting before formatting/Close; synchronized temporary cleanup and retained source prefixes separately; budget/refusal/allocation tests, all source debug actions, warm plateau and no hidden owner allocations.

Native/full checkeddebug, named race and actual checkeddebug JS RUN/PASS, vet, WASI builds, source audit and no-generics checks follow immutable selected-file snapshots. All artifacts remain below golang. Until reviewed dependencies and integration are implemented, the complete arithmetic plugin and its registered consumer tests remain OPEN.

## Backend prerequisite audit and proposed next actual algorithm packet

The declaration plugin depends on `algebraic_numbers::manager`, which is a real 3073-line implementation plus a492-line public header. Its implementation owns rational and algebraic cells (polynomial coefficient array, MPBQI isolating interval, minimal/sign/not-rational/index flags), cached intervals, temporary numerals and vectors, actual polynomial/upolynomial managers, source allocator and borrowed resource limit. It requires more than the currently implemented raw MPZ/MPQ and MPBQ/MPBQI arithmetic. Public methods include lifecycle/conversion, arithmetic/roots/powers, order/equality and sign/evaluation, isolation/refinement, integer selection, parameter updates and rendering. None can be replaced by an `OwnedRational` pretending to represent an irrational.

The real dependency chain is:

`IntegerManager` → `mpzzp_manager` → dense upolynomial core arithmetic/division/GCD → Sturm chains/root isolation and refinement → algebraic cells/operations → plugin external parameters/numerals. Factorization, polynomial arithmetic/resultants and conversion dependencies join this path; `sexpr2upolynomial` then enables the plugin's actual polynomial-root sexpr entry point. Current MPBQ/MPBQI cover dyadic interval numerals, not these polynomial/root algorithms.

`upolynomial.h` is922lines and `.cpp`3127lines; the header also imports the actual multivariate polynomial header and source factor parameter definitions. Dense core owns a real `mpzzp_manager` plus basic/division/exact-division/GCD/CRA/six modular-GCD/square-free/power temporary vectors. Its ctor does not eagerly compute polynomials. Source reset/trim/set_size/set/neg/add/sub/mul have concrete owning coefficient actions and alias handling (`m_basic_tmp` then swap); this should be a later complete selected algorithm packet, not a copy-on-write Go slice substitute. GCD uses the real prime table, modular images and fallback; factorization lives in the additional upolynomial factorization source and uses its real RNG. Conversion from multivariate source `polynomial_ref` remains open until its actual owner is implemented.

`sexpr2upolynomial.cpp` accepts exactly +, -, *, ^, integer coefficients and symbol x; it recursively checks depth, builds real scoped coefficient vectors, and executes exact add/sub/mul/power actions. Its errors carry source line/position. The existing sexpr owner is a prerequisite, but a standalone parser cannot advance root creation until these polynomial algorithms exist.

The smallest coherent missing actual algorithm is **all defined operations of `src/util/mpzzp.h`**, the dynamically selected Z / GF(p) / composite Zp numeric manager used by upolynomial. It needs no fabricated polynomial object and can use current `IntegerManager` operations, including real ExtendedGCD, signed Rem, fused arithmetic, bit/log/root operations and owned formatting. Proposed independent ownership is new `mpnum/mpzzp.go`, `mpnum/mpzzp_test.go` and a separate status document, subject to coordinator approval.

Concrete `IntegerModularManager` borrows one actual unsynchronized `IntegerManager` and owns the source seven real Int fields: modulus, lower bound, upper bound, three inverse temporaries and division temporary. Its actual cold header must reserve before object publication. Close deletes these fields in explicit source order and refunds only its own header; the borrowed numeric manager outlives it. All arithmetic uses pointers to actual concrete Int records; no arbitrary precision substitute, interface dispatch, boxed coefficient or generics. Source mode/configuration, alias/publication and cached scratch are retained.

Proposed API maps every header-defined body: constructor(s), manager accessor, Precise/Field/Finite/Modular, modulus copy/read adapter, SetZ/SetZPInt/SetZPUint64/SetPSquare/SetZPSwap, IsPNormalized/PNormalize, Reset/Del/Neg/Abs, all sign/zero/one comparisons, Add/Sub/Inc/Dec/Mul/AddMul/SubMul, Inv/Div/Divides/Swap, both GCD forms and ExtendedGCD, every typed Set, mutable versus const machine-conversion routes, Hash/Power/PerfectSquare/Mul2k/PowerOfTwoMultiple/Log2/MLog2/MachineDiv2k/Root, formatting and destruction. Signatures remain explicit for mutable conversion operations, because those normalize their input while the const source overloads do not. A concrete manager's private helpers avoid duplicated numeric algorithms; normalization remains the actual Rem then upper comparison/Sub or lower comparison/Add, not canonical [0,p) modular arithmetic.

Important source defects/domains discovered in the complete `mpzzp.h` read:

- Bounds are centered: odd p has [-floor(p/2),floor(p/2)], even p has [1-p/2,p/2]. Normalization/Neg/Abs and ordering preserve this representation.
- Both modulus-taking constructors accept `prime` but never initialize `m_p_prime`. After such construction, `field()` and division's debug prime assertion read an indeterminate bool. The Z constructor plus explicit `set_zp` is the defined source route (set_zp sets true), used by registered upolynomial tests. Do not silently initialize the buggy constructor from its unused argument and claim faithful behavior. A Go constructor may represent an explicit unknown-prime state and diagnose only the actual undefined bool read, allowing otherwise defined normalization/arithmetic; status must mark that adapter precisely.
- `set_p_sq` sets prime=false before squaring modulus and recomputing bounds; `set_zp_swap` preserves the prior prime flag. `divides` short-circuits field/nonzero before raw IntegerManager.Divides.
- Inverse normalizes the caller first, performs real ExtendedGCD into all three cached outputs, normalizes its first coefficient, swaps into the caller, and only then debug-asserts gcd==1. Composite/nonunit release results are not to be strengthened into a preemptive mathematical inverse check. Undefined inputs and enabled source assertion adapters are separate.
- `power` uses a uint32 mask loop, initializes/copies its local power then writes output1, and squares the local on every iteration including the final one. For exponent>=2^31 the mask wraps0 and the source loop never terminates. Explicit typed nontermination handling must be documented rather than a replacement exponentiation algorithm; finite exponents preserve all source actions and aliases.
- Modular perfect-square calls source NOT_IMPLEMENTED_YET; modular roots have source debug !modular preconditions. These are explicit remaining unsupported/undefined conditional routes, not invented modular algorithms.

No direct registered mpzzp test exists. `src/test/main.cpp:205–207` registers polynomial, upolynomial and algebraic. Upolynomial's active entry calls GCD, lower bound, factorization, remainder, exact division, root isolation5, root isolation (twice), Sturm2, rational-to-dyadic conversion, translation, refinement/refinability, Sturm, one-half removal, basic arithmetic, Zp cases, and two extended-GCD cases. GCD body has an unconditional early return; its later examples are inactive. The Zp and extended-GCD examples print polynomial outcomes rather than standalone mpzzp ENSUREs. Algebraic's active entry calls root, isolation, ex1, sign evaluation, select-small, Dejan, Wilkinson, basic and dyadic refinement. Their registered assertion loops cannot be counted without full call/conditional evaluation, and are not claimed ported by the modular-manager unit.

Tests-FIRST for the modular packet should observe missing API RED, then independently freeze centered representatives for every residue in small odd/even moduli; source integer-mode delegation; prime/composite mode transitions; all inverse/division/fused aliases; exact ExtendedGCD cached coefficients and source output order; mutable versus const conversion; final unused power-square action failures; constructor unknown-prime adapter; immediate cells/Words/header accounting at every predictable allocation failure before formatting/Close; all real debug short circuits and warm cached zero-allocation paths. Test-only bounded integer arithmetic/math-big may provide independent numeric expectations. Full upolynomial/algebraic registered consumers remain open until their algorithms run.

After that leaf packet, propose the actual dense coefficient owner plus full core basic arithmetic/reset/trim/set/neg/add/sub/mul/derivative/equality/evaluation needed by subsequent division and Sturm isolation. Its exact concrete storage and selected source test assertions must be reviewed separately; this proposal does not authorize that implementation or pretend it completes root isolation.

## Subsequent implemented backend prerequisite

The approved mpzzp leaf proposal above has now been implemented and verified
in MPZZP_PORT_STATUS.md: actual IntegerModularManager owns the seven source
numerals and preserves every selected defined method and publication order,
with constructor unknown-prime and raw-Power error-cleanup adaptations explicit.
Thirteen tests-first groups and full native/debug/race/actual JS/vet/WASI/source
gates passed. This does not install an arithmetic plugin or execute polynomial
root isolation. Dense coefficient ownership and source arithmetic are the next
proposal; GCD/Sturm/algebraic/external-parameter and real AST dispatch/coercion
remain original complete-port requirements. No rational-only plugin replaces
those missing algorithms.
