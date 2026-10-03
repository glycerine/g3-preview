# Complete preserved global parameter catalog — implemented collection/dispatcher

This is the full preserved `src/test/gparams_register_modules.cpp` catalog, not a selected SAT-only callback. The complete selected static collection and dispatcher is implemented and tests-first API RED was observed. Registry fourteen files and central metadata remain coordinator-owned except the explicitly approved narrow `util_gparams.go` catalog hook after tests-first RED.

## Generator and exact source surface

`scripts/mk_genfile_common.py:598–653` sorts headers by component path (42), scans exact REG_PARAMS/REG_MODULE_PARAMS/REG_MODULE_DESCRIPTION directives, and emits all globals first, modules second, descriptions last. It retains input/source order and duplicates. `scripts/mk_util.py:2853–2871` generates per-component dispatchers from each component's real dependency header set. The preserved complete test dispatcher is the selected executable catalog; other build-component selections remain distinct runtime integration, not fabricated catalogs. The generator docstring says init invokes registration, but actual `gparams.cpp:89` invokes it lazily from check_registered after publishing TRUE.

The dispatcher has 2 global collector calls: context_params (12 declarations then its 3 solver declarations, `context_params.cpp:158–177`) and env_params (5 declarations, `env_params.cpp:34–40`). There are 32 module calls, 438 insertion attempts, 430 distinct module descriptors over19 modules, and16 descriptions over15 modules. With the20 globals, complete dispatch attempts458 descriptor insertions and yields450 distinct global/module descriptors. These are source insertion counts, not source assertions. No standalone registered assertion suite for the generated file exists.

Every generated collector is a sequence of real `ParamDescrs` insertions. The NNF wrapper at `nnf.cpp:920` delegates through `imp::get_param_descrs:316` to nnf_params and is followed by a second direct nnf_params collection. Collecting these declarations requires no AST/solver execution. The generated value accessors, their owner-local lifetime wrappers, env_params::updt_params warning/memory effects, context creation, all actual solver algorithms, and generated build initializer wiring remain separate source units.

Important duplicates: NNF's5 keys are repeated; rewriter.hi_fp_unspecified is repeated; rewriter.flat is repeated twice. ParamDescrs keeps the first metadata. SAT totals45 distinct keys: asymmetric3 + existing actual SAT24 + SCC1 + simplifier17. Ackermannization merges1+2 keys. Its final description overwrites the earlier description. Description text aliases real borrowed bytes and is not copied by the registry.

## Concrete owner and APIs

Implemented NEW util_gparams_catalog*.go and tests only:

* `GlobalParamsCatalogConfig{Budget *arena.Budget, Descriptions arena.BytesRegionConfig, Text arena.VectorConfig}`. All actual owned allocations use one unambiguous Budget; nested nonnil mismatches reject before header allocation. Allocator page sizes are configurable for tiny/failure tests; source initial descriptor table capacity remains8.
* `NewGlobalParamsCatalog(*SymbolManager, GlobalParamsCatalogConfig) (*GlobalParamsCatalog,error)`.
* `(*GlobalParamsCatalog).Register(GlobalParamsRegistrar) error`, `CleanupPending() error`, `Close() error`.
* Approved registry hook: `GlobalParamsConfig.Catalog *GlobalParamsCatalog`, private `GlobalParams.catalog`; ensureRegistered publishes TRUE then calls the concrete method directly. Catalog+Register callback together is ErrConfig before header allocation. NewGlobalParams checks actual common SymbolManager and open catalog. Nil/nil keeps existing missing-dispatch semantics. No escaping stored closure/method-value environment is needed.

The charged cold catalog header contains mutex/self, borrowed SymbolManager/Budget, real owned BytesRegion, inline `[16]BytesRegionRef`, concrete text-config state, and one `pending *ParamDescrs` owner root. It has no host map, per-entry interface/pointer, pointer-bearing hot handle, closure environment, or uncharged root directory. The inline scalar refs are charged as part of the actual header. Description bytes are source exact literals, allocated sequentially with NUL into the actual owned region during cold construction; source read-only data is adapted to explicit charged byte ownership. Literal static metadata itself remains borrowed read-only source data. The catalog must outlive every registry borrowing that region; closing a registry never closes it.

Collector methods emit exact literal InsertName calls, not iteration over an invented global metadata map. The committed collectSATConfigParamDescrs is reused unchanged for the24-call body. All remaining generated collectors and the context/env pure collection bodies are concrete typed functions. NNF's exact delegation/repetition is preserved without claiming NNF algorithms are implemented.

## Failure order and ownership

Global descriptors are local owning stack values in source: real temporary owner creation→collector→RegisterGlobal copy→lexical cleanup, also on failure. Module descriptors are raw heap pointers: create→collect→RegisterModule. Source collection/insertion failure has no native RAII for that pointer. Approved Go adaptation retains the exact completed allocated prefix in one actual charged `pending` root. Successful transfer/merge clears that root, with merge incoming destruction still after successful destination copy. Failed merge keeps destination's completed prefix plus incoming pending owner. Catalog.Close cleans pending exactly once; explicit CleanupPending permits owner reuse without discarding it silently. Register rejects pending reuse before starting another dispatch. Normal ensureRegistered TRUE prevents callback retry regardless of catalog failure.

Description registration occurs only after all global/module collection calls. Each stored scalar ref points at the catalog-owned real region; dictionary/binding growth and overwrite ordering are existing actual registry algorithms. Closed description owner guards remain at actual later read actions; construction/registration do not validate borrowed spellings early to narrow the source pointer-alias contract. Native singleton/exception recovery is not falsely claimed. Successful catalog reuse across registries requires independent actual descriptor owners plus shared borrowed-description identity; closing one registry must not affect the other.

## Tests-first plan

Before production, compile-contract RED31990 established missing Catalog config/type/ctor/Register APIs. Exact complete metadata/hook tests then passed18322; subsequent independent ownership/fault checks passed immediately against those implementations. Permanent hermetic source metadata fixtures contain all exact tuples/order/default/kind/description/module bytes; Go tests never read reference files or invoke the generator/compiler. Independent live counts, first-wins duplicates, module physical bucket order and exact final overwritten description expectations validate the real registry. Fixed source hashes below bind this audit.

Finite fault tests cover constructor owner/region unwind, every selected collector body and full dispatcher prefix milestones, successful copy/merge versus pending retention, immediate budget graphs before teardown, later description insertion failures, closed/foreign owner domains, rejected callback ambiguity before allocation, unchanged TRUE registration after failure, CleanupPending, and successful reuse across two real registries. Large complete metadata tests use actual finite450 descriptors, not synthetic placeholder defaults. The constructor sweep covers180 fault prefixes. Actual complete-dispatch reserve threshold1529 is discovered through bounded doubling/binary search and exercised at representative early, module-pending, and late-description prefixes. Immediate budget graphs are checked before cleanup. Success tests check every one of458 source tuples,450 distinct descriptors, all19 publication positions,16 description assignments, and two independent registries sharing borrowed descriptions. Closed-region errors preserve the actual emitted description prefix. Final frozen gates are recorded below.

## Full module call order

| Order | Module | Source collector | Insertions | Actual source |
| --- | --- | --- | --- | --- |
| 1 | ackermannization | ackermannization_params::collect_param_descrs | 1 | `src/ackermannization/ackermannization_params.hpp` |
| 2 | rewriter | ackermannize_bv_tactic_params::collect_param_descrs | 1 | `src/ackermannization/ackermannize_bv_tactic_params.hpp` |
| 3 | rewriter | fpa2bv_rewriter_params::collect_param_descrs | 1 | `src/ast/fpa/fpa2bv_rewriter_params.hpp` |
| 4 | nnf | nnf::get_param_descrs | 5 | `src/ast/normal_forms/nnf_params.hpp` |
| 5 | nnf | nnf_params::collect_param_descrs | 5 | `src/ast/normal_forms/nnf_params.hpp` |
| 6 | pi | pattern_inference_params_helper::collect_param_descrs | 8 | `src/ast/pattern/pattern_inference_params_helper.hpp` |
| 7 | pp | pp_params::collect_param_descrs | 17 | `src/ast/pp_params.hpp` |
| 8 | rewriter | arith_rewriter_params::collect_param_descrs | 13 | `src/ast/rewriter/arith_rewriter_params.hpp` |
| 9 | rewriter | array_rewriter_params::collect_param_descrs | 3 | `src/ast/rewriter/array_rewriter_params.hpp` |
| 10 | rewriter | bool_rewriter_params::collect_param_descrs | 7 | `src/ast/rewriter/bool_rewriter_params.hpp` |
| 11 | rewriter | bv_rewriter_params::collect_param_descrs | 15 | `src/ast/rewriter/bv_rewriter_params.hpp` |
| 12 | rewriter | fpa_rewriter_params::collect_param_descrs | 1 | `src/ast/rewriter/fpa_rewriter_params.hpp` |
| 13 | rewriter | poly_rewriter_params::collect_param_descrs | 5 | `src/ast/rewriter/poly_rewriter_params.hpp` |
| 14 | rewriter | rewriter_params::collect_param_descrs | 10 | `src/ast/rewriter/rewriter_params.hpp` |
| 15 | interp | interp_params::collect_param_descrs | 2 | `src/interp/interp_params.hpp` |
| 16 | algebraic | algebraic_params::collect_param_descrs | 6 | `src/math/polynomial/algebraic_params.hpp` |
| 17 | rcf | rcf_params::collect_param_descrs | 6 | `src/math/realclosure/rcf_params.hpp` |
| 18 | model_evaluator | model_evaluator_params::collect_param_descrs | 5 | `src/model/model_evaluator_params.hpp` |
| 19 | model | model_params::collect_param_descrs | 5 | `src/model/model_params.hpp` |
| 20 | fixedpoint | fixedpoint_params::collect_param_descrs | 123 | `src/muz/base/fixedpoint_params.hpp` |
| 21 | nlsat | nlsat_params::collect_param_descrs | 11 | `src/nlsat/nlsat_params.hpp` |
| 22 | opt | opt_params::collect_param_descrs | 20 | `src/opt/opt_params.hpp` |
| 23 | parser | parser_params::collect_param_descrs | 3 | `src/parsers/util/parser_params.hpp` |
| 24 | sat | sat_asymm_branch_params::collect_param_descrs | 3 | `src/sat/sat_asymm_branch_params.hpp` |
| 25 | sat | sat_params::collect_param_descrs | 24 | `src/sat/sat_params.hpp` |
| 26 | sat | sat_scc_params::collect_param_descrs | 1 | `src/sat/sat_scc_params.hpp` |
| 27 | sat | sat_simplifier_params::collect_param_descrs | 17 | `src/sat/sat_simplifier_params.hpp` |
| 28 | smt | smt_params_helper::collect_param_descrs | 88 | `src/smt/params/smt_params_helper.hpp` |
| 29 | combined_solver | combined_solver_params::collect_param_descrs | 3 | `src/solver/combined_solver_params.hpp` |
| 30 | sls | sls_params::collect_param_descrs | 22 | `src/tactic/sls/sls_params.hpp` |
| 31 | ackermannization | qfufbv_tactic_params::collect_param_descrs | 2 | `src/tactic/smtlogics/qfufbv_tactic_params.hpp` |
| 32 | lp | lp_params::collect_param_descrs | 5 | `src/util/lp/lp_params.hpp` |

## Exact description assignment order

1. `ackermannization`: solving UF via ackermannization
2. `nnf`: negation normal form
3. `pi`: pattern inference (heuristics) for universal formulas (without annotation)
4. `pp`: pretty printer
5. `rewriter`: new formula simplification module used in the tactic framework, and new solvers
6. `interp`: interpolation parameters
7. `algebraic`: real algebraic number package
8. `rcf`: real closed fields
9. `fixedpoint`: fixedpoint parameters
10. `nlsat`: nonlinear solver
11. `opt`: optimization parameters
12. `sat`: propositional SAT solver
13. `smt`: smt solver based on lazy smt
14. `combined_solver`: combines two solvers: non-incremental (solver1) and incremental (solver2)
15. `sls`: Experimental Stochastic Local Search Solver (for QFBV only).
16. `ackermannization`: tactics based on solving UF-theories via ackermannization (see also ackr module)

## Reference SHA256

* `scripts/mk_genfile_common.py` — `86cdb958696d411d48d04b74fc56e379a6ad41eba03d1f52cf76c53813f15482`
* `scripts/mk_util.py` — `4b1921c9cdf40923f8d12b81c6319cab2f9cfbb8dd5a3abcc48deeb94b3179fd`
* `src/test/gparams_register_modules.cpp` — `682377f035394159980e4362ca7d303c6488d0ad17c592c8d4fdc5d40b8ff723`
* `src/cmd_context/context_params.cpp` — `d7a23c364cd4141afad2d31339e19f65e4f475a7f018021fdc4ec458b872fe19`
* `src/util/env_params.cpp` — `50f9f1037080177fc94238dc662a6db726dc4eecdcaa3e85bcb12aba322fad65`
* `src/ast/normal_forms/nnf.cpp` — `560de5b04fa04bb4304e696d12198c42ac91c3d8bbd47a03b74f007bb4645acc`
* `src/ackermannization/ackermannization_params.hpp` — `81c11ede810dc135a9e0c42b87ce721f2b2d972b32b32ca11b01b30918805d90`
* `src/ackermannization/ackermannize_bv_tactic_params.hpp` — `4869771a7e400341bea0672d30d82ad2973b310ea6d883d94463820810a8ee88`
* `src/ast/fpa/fpa2bv_rewriter_params.hpp` — `7d8895e0e210d35436b5aaaa93eb824b4ca64c57a31dd2ad7414114e7c008193`
* `src/ast/normal_forms/nnf_params.hpp` — `95bf0c852f3a37898c024cd94b8b7f39f8ae1999a242535725498b54b2bfa774`
* `src/ast/pattern/pattern_inference_params_helper.hpp` — `2087b33deb1ea8271a654b4d757af1f39bb88ba7d884328e69d6e251962fcdd9`
* `src/ast/pp_params.hpp` — `db35bc2546189e941fa931092b6c3030dfdde789d92b70344969c1aba151ce93`
* `src/ast/rewriter/arith_rewriter_params.hpp` — `3ea48a9aaf4e963842210187a2261ea4f020fc3e08a3eb68c4d3a42ca1c20871`
* `src/ast/rewriter/array_rewriter_params.hpp` — `7dc7e99f3ddb9bbde7a3edf1ffd1ecd910d15e86a2c09026f1df115a34d21de7`
* `src/ast/rewriter/bool_rewriter_params.hpp` — `b31fb8aad246efac68934880efa991671a885cdfa1355371f47d34c130adb311`
* `src/ast/rewriter/bv_rewriter_params.hpp` — `f8ab87c91980d2f1ed259f7e6c50ff955604d8fc511f96c9c47665c5ee36d3a2`
* `src/ast/rewriter/fpa_rewriter_params.hpp` — `826bdd3f5399261577ae024d62aee1adc8a2fa27de3b74a15cbadec8a2232742`
* `src/ast/rewriter/poly_rewriter_params.hpp` — `e58e0860c8fa2dd41ce6ed85a73c6490083427f13a1dcaf933de3251a2054033`
* `src/ast/rewriter/rewriter_params.hpp` — `0dce2c6cadee2f1b4a409b6f9549615567f0918d2568829470a12afaf36bf992`
* `src/interp/interp_params.hpp` — `daee2b3402763d0b71beb5ad0ae2c2d7da64f1a18bb57bd0ade8e107377dbc85`
* `src/math/polynomial/algebraic_params.hpp` — `964aff3521baa5624fb57240b12e897f94b1c91f6bea2f27e8e1eaf931ef3c5f`
* `src/math/realclosure/rcf_params.hpp` — `36e500d8969537cac4c628d9730cbaad01002a61e1621c50a4a58ef9a61b8689`
* `src/model/model_evaluator_params.hpp` — `0cf305165a4dbe83a019f6f600d37293862e72be5ac0cadcca9a15924e61badd`
* `src/model/model_params.hpp` — `38950508d7155ef0bc057faeb8598082950fbbe521fce1032bf8dd08da226870`
* `src/muz/base/fixedpoint_params.hpp` — `9b464292f7f8ad2409be764a04b9f45ab59c452cd2084fd06dd5ee8b672ad554`
* `src/nlsat/nlsat_params.hpp` — `5a303fe15ce6ce2334f637e957ac0e0ee45fee9670e3de9a535cfaa6e99e9e18`
* `src/opt/opt_params.hpp` — `e08ed20495a1f4bd11e39aa939a65b79e23e72ec2c1ba640c5c67b4c0c583881`
* `src/parsers/util/parser_params.hpp` — `18b17fb81d735175e45b0fb3f7fb00f0ad5569e0c7d82fb59f5b6efde492e5fd`
* `src/sat/sat_asymm_branch_params.hpp` — `58ffd6f8486fa11980b3934777b27c4e425e2afa442fd802eec92eecc05e45d6`
* `src/sat/sat_params.hpp` — `e253cef347626558a87c85938349996f7a203500d39280f390dc89abe3045284`
* `src/sat/sat_scc_params.hpp` — `3f230001e4b89584956eb6ba59ea4a7bb48977fb975096d914686315fa1144c5`
* `src/sat/sat_simplifier_params.hpp` — `6c5097defa9b8789e0e21bb92ac0ddc5406a8073a4bdb2c14d51678141790ab9`
* `src/smt/params/smt_params_helper.hpp` — `09b28acdc4c25f4f86ca7a4c1f149abe9c245fbe0f07ecb13fbc1c039600d25e`
* `src/solver/combined_solver_params.hpp` — `2378a45d9eae15beded43d64c4f1e61a16e12e5c35b6eedc0e325bfefbe815b4`
* `src/tactic/sls/sls_params.hpp` — `d75b2e4c33753b231ac4599d33c376e0f56e87da015796f9baddf1650709fa9c`
* `src/tactic/smtlogics/qfufbv_tactic_params.hpp` — `0c63264a9ddeba2668c08c36856e10e763a1f56f5b6dbe9b41267b4f56ea08e6`
* `src/util/lp/lp_params.hpp` — `05c3a08d15fec382fc2fa6854806fc2e7f090aaaac29895ed810b47d1076694a`

## Frozen executable packet

Six new catalog Go files and the narrow approved util_gparams.go hook are frozen on HEAD `2c89d3f53dc790bf3af247d20702dd683ad6e3c6`, snapshot `.cache/release/gparams-catalog-final`. Existing registry production/tests and SAT24 collector are otherwise unchanged. There are5 new catalog groups plus22 existing registry groups, all27 explicitly verified RUN/PASS in race and actual JS logs. Source methods are pure collections only: generated constructors/accessors and the broader containing AST/context/env modules remain partial. The full preserved dispatcher collection surface is executable; native singleton/runtime wiring and per-component alternative generated dispatchers remain separate integration.

Narrow registry hook adds only Catalog/config owner pointer fields, pre-header ambiguity/open/common-symbol checks, and direct Register invocation after TRUE publication. No stored closure is introduced. Collector function parameters reference plain static functions with no captured environment, never retained in a hot record or owner. Catalog header/backing/region/pending roots are charged; caller-owned static literals are borrowed program data. Source raw-pointer collector failure retains pending storage, with explicit CleanupPending/Close as documented Go cleanup continuation.

| Gate | Terminal result / handle | Log |
| --- | --- | --- |
| Full native ./... | PASS exit0 /6420 | `.cache/gparams-catalog-final-native.log` |
| Full checked-debug ./... | PASS exit0 /75527 | `.cache/gparams-catalog-final-debug.log` |
| Selected checked-debug race | 27 exact RUN/PASS, exit0,1.409s /4863 | `.cache/gparams-catalog-final-race.log` |
| Actual checked-debug JS/WASM | 27 exact RUN/PASS, exit0,0.306s /61149 | `.cache/gparams-catalog-final-js.log` |
| Full checked-debug vet | PASS exit0 /46060 | `.cache/gparams-catalog-final-vet.log` |
| Full checked-debug WASI build | PASS exit0 /22824 | `.cache/gparams-catalog-final-wasi.log` |

## Frozen SHA256

* `util_gparams.go` — `6e251ab345ff9b3cc9ebde2d86812959220085ee5a059119cf73062b94a81642`
* `util_gparams_catalog.go` — `5e79b765d0966775dc31a3c55639a10b6271e7385cba6a7eee6e8936feda9a49`
* `util_gparams_catalog_collectors.go` — `feb01833ab4ffd9f463e294ee40a97b7bb5877945211c6e855d15dfc02d69f1b`
* `util_gparams_catalog_dispatch.go` — `f00002468a41a1805ccf7609052a9f77fbd7deec61b2a29b96e129808ef32b61`
* `util_gparams_catalog_fault_test.go` — `7b5e831dda56c74c7feb55b9080ed2a4dd6b2d955831631cf9845597b961f998`
* `util_gparams_catalog_fixture_test.go` — `d85f1eb644c87c5ce8324eb9a0d4846e1a7e729344fdec806c0b37d750c8e0b5`
* `util_gparams_catalog_test.go` — `fa96b5730397f710d3d09e1912fd349a8a1931a71c5b62b126d8de9304016e95`

## New test groups

* `TestGlobalParamsCatalogCompleteMetadataAndExactDispatcher`
* `TestGlobalParamsCatalogConcreteHookGuardsAndIndependentReuse`
* `TestGlobalParamsCatalogConstructorFailureAndDomains`
* `TestGlobalParamsCatalogDescriptionBorrowReadAndClosedPublicationPrefix`
* `TestGlobalParamsCatalogDispatcherFaultRootsAndLateDescription`

Source/no-generics audit15787 terminal PASS exit0: `2102 units, 677 APIs: 33 ported, 102 partial, 1967 unported` against unchanged base ledger. This is gate evidence, not an already integrated new catalog manifest.

## Coordinator integration

Coordinator verified all seven frozen/live Go hashes and all twenty-seven
exact race/actual JS RUN/PASS names. Independently parsed every actual C++
collector body and compared all458 ordered tuples against the frozen fixture.
Integration snapshot `.cache/review/gparams-catalog-integration-2c89d3f` has
identical complete Go source to the agent-tested snapshot; full native/debug/
vet/WASI gates are reused. Updated source/no-generics audit26546 passed,
33ported/140partial/1929unported. Forty source rows now record selected static
collection/dispatch coverage, with38 newly partial. Actual sat_params.hpp
presence/hash was verified and its earlier inaccurate absence note corrected.
Full runtime/value accessors and solver/proof/interpolation remain open.
