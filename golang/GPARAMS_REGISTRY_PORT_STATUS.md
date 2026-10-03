# Owner-local global parameter registry

The executable registry algorithms from `src/util/gparams.h/.cpp` are translated into concrete `GlobalParams`. This packet uses real `SymbolManager`, `ParamsContext`/`ParamsRef`, `ParamDescrs`, borrowed `BytesRegion` spellings, and charged byte destinations. It does not substitute injected configuration snapshots or fabricated parameter values. The entire source units remain **partial**: the complete generated `gparams_register_modules` catalog, process-global initializer/finalizer wiring, native process stdout, OpenMP integration, and physical global memory-counter hooks remain separate dependencies. The actual SAT24 collector is committed and exercised; it is not the entire generated catalog or a solver.

The previous committed `GPARAMS_PORT_STATUS.md` remains a historical prerequisite record. This file describes the subsequent registry implementation without changing that coordinator-owned record.

## Source mapping

All line references below are in `src/util/gparams.cpp`, unless stated otherwise.

| Source executable body | Concrete implementation |
| --- | --- |
| old-name lookup 28; rename lookup 69 | `oldName`, `renamedName`; exact static literal order, including first `pp_decimal` match |
| `check_registered` 89; descriptor getters 96–98 | `ensureRegistered`, private dictionary/descriptor access; TRUE published before the one cold dispatcher, failure does not clear it |
| constructor 101; destructor 106; reset 112 | `NewGlobalParams`, `Close`, `Reset`; real global Ref reset and physical bucket-order module-cell destruction, descriptor registration retained across Reset |
| register_global 130 | `RegisterGlobal` and registrar equivalent delegate to real `ParamDescrs.Copy` |
| register_module 136 | `RegisterModule`: source destination Copy before incoming destruction; ownership transfers only on successful new insertion |
| register_module_descr 150 | `RegisterModuleDescription`: nullable borrowed scalar origin/ref/offset; no spelling read or owned spelling copy |
| normalize 162 | `normalizeName`, `normalizeSymbol`, `normalizeWorkspace`: one leading colon, first C NUL, ASCII lowercase/hyphen conversion, first dot, suffix interned before module prefix |
| get_params 185 | `getParams`: null module aliases real global Ref; actual cold heap cell and dictionary publication precede kind failure |
| throw_unknown_parameter 200 | `unknownParameter`: exact legacy/rename/generic labels and real sorted descriptor output |
| validate_type 234 | `validateType`: digits (including empty unsigned), permissive double filter, exact bool values |
| internal set 271 | `setValue`: kind then module-cell acquisition; real setters, canonical Symbol string borrowing; unsupported kind keeps the completed empty-cell prefix |
| public set 327 and wrappers 569–578 | `SetName`, `SetSymbol`: normalize, actual registration/descriptor lookup, validation, setter; typed immutable exception returned after lock release |
| stored/default get 362,368; public get 378,580–588 | `getValue`, `getDefault`, `GetValueName`, `GetValueSymbol`: real stored-kind lookup before registration/default lookup, nullable default becomes `default` |
| get_module 424,598–605 | `GetModule`, `GetModuleName`: retained actual current shared Ref snapshot on each call; **no lazy registration** |
| get_ref 436,608 | `GetRef`: borrowed const-like alias to the real global Ref, not a snapshot |
| display 446,613 | `DisplayInto`: global heading before registration, descriptors, non-SMT2 instructions, physical module bucket order; uint32 indent+4 wraps as source |
| display_modules 469,618 | `DisplayModulesInto`: actual registration before first row; empty registry never touches an unused output sink |
| display_module 483,623 | `DisplayModuleInto`: raw module-name interning (no normalization), lookup/error before output, heading then actual descriptor display indent4/include-description true |
| display_parameter 513,628 | `DisplayParameterInto(out,debugOut,name)`: original C name/module/parameter diagnostic into explicit debug sink **before** descriptor lookup/error; exact detail fields into out |
| source g_imp/init/finalize 566,633,638 | explicit runtime-owned `NewGlobalParams`/`Close` lifetime adaptation; no mutable process-global cache or substitute singleton |

`dictionary<T>` in `dictionary.h:23` resolves to the actual map/core-hashtable algorithms. Three independent concrete hot record/backing specializations retain scalar keys, cached full hashes, state, and scalar root indices. Growth happens at the old strictly-greater-than-3/4 load, before overwrite; rehash walks old physical buckets; probing remembers the last deleted bucket; reset counts previously FREE slots and frees before a qualifying shrink allocation (`hashtable.h:286–309`). No scoped SymbolInt32Table, generic template, hot interface, or hidden spelling dictionary replaces these maps. The copied source algorithms are selected registry operations, not a claim that every map overload is ported.

## Ownership, budgets, synchronization and failure

One charged cold registry header owns its three real record arrays, global descriptor owner, successfully transferred module descriptor owners, real cold ParamsRef cells, root pointer directories, borrowed region-binding directory, and byte workspace. Every real retained backing/header is charged using the translated physical stride, host byte bounds, and old+new growth peaks. Nested descriptor graphs keep their own budget contracts; borrowed Symbol/Params/BytesRegion owners, caller-provided registration closure environment, and caller-returned immutable output/error strings are external boundaries. The source map's inline-small backing and local C++ strings are implemented with fully charged Go backings; these reservations are **storage adaptations**, not alleged C malloc events. Global physical allocation hooks remain open.

Cold directory preparation can precede source dictionary publication. A successful binding/directory growth remains owned if later map insertion fails; unpublished cells are released, and no failure-injection attempt counter is rewound. Module reset releases each real cell and reuses private directory slots on future creation, keeping retained metadata bounded. These slots are not user handles. Failed source shrink leaves native recovery undefined; the Go continuation exposes an absent guarded backing with released storage and safe Close, rather than reading freed source memory.

Merge failure retains both the source destination's completed Copy prefix and the caller's incoming descriptor owner; incoming destruction occurs only after full success. Unknown-kind set preserves actual get_params publication. Validation failure occurs before module-cell creation. Constructor failure releases all successfully created owner roots. Tests compare complete retained budget graphs immediately after failures, before Close can hide leaks.

The registry mutex is a source critical-section adaptation. Registrar methods are by-value cold capabilities valid only while the dispatcher executes under that lock; callbacks must not reenter public registry methods. Output helper lock order remains output→descriptor→Symbol. Registry operations acquire Symbol locks only while public output calls are not holding those locks, and private metadata writes use the existing descriptor helper ordering. Provided owners must outlive borrowers and be torn down only under caller coordination. Independent service locks do not make a shared external `Budget` globally thread-safe. Plain Go owner copies and shallow owning copies are forbidden; self-identity guards diagnose the supported cold misuse domain.

`GetRef` is const-like by contract: callers must not mutate or Del the borrowed pointer. GetModule snapshots are genuine owning shared refs and must be Del'ed with the borrowed ParamsContext; they can outlive the registry while that context remains live. Closing the registry never closes borrowed services or description regions. NULL description is distinguished from present-empty and sets the byte-stream adapter's typed failure at its actual output action; stale/closed region errors occur at the actual byte read, preserving emitted prefixes.

## Explicit source/platform boundaries

* Default selected C signed-long width is 64; `CLongBits:32` tests the alternate strtol saturation then uint32 narrowing. No host-width reinterpretation silently changes the selected profile.
* The public double validator admits only digits, minus, dot and slash. Its actual strtod consumed-prefix behavior is adapted to C-locale nearest IEEE64. The exact nearest overflow threshold is checked before `strconv.ParseFloat`, avoiding escaping host error allocations. Other locales, fenv modes, general C strtod spellings and ABI behavior are not claimed. No production math/big is used; tests derive an independent exact threshold.
* Numerical Symbol `bare_str` returns empty in source release but asserts in debug (`symbol.h:96`). Registry symbol overloads preserve this action, instead of universally rejecting the defined release route. NULL raw bare-string dereference has a typed guard. C strings stop at first NUL.
* The source's stdout diagnostic is preserved in an explicit `debugOut` byte stream. Native process stdout binding remains **OPEN**, not an invented CLI. Arbitrary ostream flags/failbit/locale behavior is an existing byte-stream boundary: nullable output returns typed failure immediately with the completed source prefix.
* A nil stream is checked only at its first actual output action. DisplayInto fails its heading before registration, module display can register/look up first, and unknown-module errors precede unused output errors.
* Static old-name/rename tables are borrowed immutable literal metadata, not a mutable global cache. Caller-provided catalogs are real descriptors but cannot establish complete generated catalog coverage.

## Tests-first and finite evidence

No standalone registered source gparams assertion suite was found. `src/test/main.cpp:128` is an actual command-line consumer; `src/test/gparams_register_modules.cpp:37` is the actual generated registration dispatcher. No source ENSURE count is fabricated. There are **22 Go test groups**, with exact inventory below and in `.cache/gparams-final-hashes.json`.

Initial registry, parser, set/get and display method tests observed missing-API RED before production. New integration tests of already available production methods passed immediately where appropriate. The reset plateau test observed a real behavioral RED (used bytes grew 2248→2256 after the second repeated cycle) and the implementation now reuses released private root slots. Nil-sink tests observed a real panic RED before the guards were added at source action positions.

Independent evidence includes physical bucket placement/rehash order, three pointer-free layout audits, strict load and overwrite failure, last-tombstone probe, release-before-shrink failure, retained module aliases/COW/reset, exact error labels and diagnostic fields, C-string/null/numerical domains, borrowed-byte mutation/closed-read prefixes, 30 constructor prefixes, 45 setter fault prefixes, 45 descriptor merge prefixes, binding-success/table-failure ownership, exact SAT24 real collector integration, and 100 repeated reset cycles. Warm GetModule/Del with an armed registry budget performs zero Go allocations. ParseDouble finite/overflow paths also have zero-allocation checks.

A narrowly bounded C probe lives only under `.cache/gparams-probes/{prefix.c,prefix,results.txt}`. Selected Linux long64 results include empty→0, 4294967296→0, saturated 9223372036854775808→UINTMAX, `2.3.4`→bits4002666666666666, `1/2`→3ff0000000000000, `-.`→+0, and `-.0`→−0. Permanent hermetic Go fixtures reproduce these cases; ordinary Go tests never invoke an external compiler.

## Frozen final gates

Snapshot `.cache/release/gparams-final`, base `2f9931b49f54cece187019c7eb667a892722eed3`, contains only the 14 registry Go files over archived HEAD, excluding live config/AST drafts. Production/test hashes remain unchanged.

| Gate | Terminal result / handle | Log |
| --- | --- | --- |
| Full native ./... | PASS exit0 / 43653 | `.cache/gparams-final-native.log` |
| Full g3checked+g3debug ./... | PASS exit0 / 2936 | `.cache/gparams-final-debug.log` |
| Selected checked-debug race | 22 exact RUN/PASS, exit0, 1.075s / 38889 | `.cache/gparams-final-race.log` |
| Actual checked-debug JS/WASM | 22 exact RUN/PASS, exit0, 0.191s / 79092 | `.cache/gparams-final-js.log` |
| Full checked-debug vet | PASS exit0 / 44917 | `.cache/gparams-final-vet.log` |
| Full checked-debug WASI build | PASS exit0 / 85427 | `.cache/gparams-final-wasi.log` |
| Source/no-generics audit | PASS exit0 / 84584; 2102 units,677 APIs,33 ported/97 partial/1972 unported on unchanged base ledger | `.cache/gparams-final-audit.log` |

The ledger counts are base evidence, not a claim that new registry source rows have already been integrated. Parent owns metadata/Git and will review selected source accounting separately.

## Frozen Go hashes

* `util_gparams.go` — `b9eff654f346a7a8df276069e8b4212bd22e2cd2c285bc79b5a49ad676638c00`
* `util_gparams_debug.go` — `bbe72a1340f335a6f964beab957134a91919e9bb9f2bf0cc53d24bfb23710d5e`
* `util_gparams_dictionary.go` — `9d6136d3cbee4a8e26ae27fb3262e1d0d1b8eb48fbf21cc3bb91d11911c9ace7`
* `util_gparams_dictionary_test.go` — `46d9fd7c28084a649265484c0ad6157314a50cd938e07d6e30e214039fe9db2b`
* `util_gparams_display.go` — `8137f1f77f28bb918343810aa4fb8129f98cf74827f5f14d496f7f5eddfbd533`
* `util_gparams_display_test.go` — `9576b1eed81899eb9a8ca2587ad75b8228f46b12d42771202c3e8d2574c3e21e`
* `util_gparams_fault_test.go` — `8e4aa4973c7216dcad6d000966f0fdf6655b5a04922a2c1cbe2f52613789998d`
* `util_gparams_names.go` — `8f64e13046f4161b56be9f0d359cb2f6143d8d8942c243ae5858a8cacecdcb40`
* `util_gparams_nodebug.go` — `8e7e1300a843b44210a98556f8406f29a1c6c31a2ded85a5a395be034b4ce98f`
* `util_gparams_parse.go` — `425d05e147380da616e5bd8aef2ca5b6d2a2212fb3020a173ee4235f3f0e8035`
* `util_gparams_parse_test.go` — `06dd3748d40b6901eb200f0e53de6fb0171f8a9d59fbe6d8823674e51277e11a`
* `util_gparams_test.go` — `6859fd867c32b56927245d59864a92ee2d6a72f6cbfae7e2cc4418909d3b491a`
* `util_gparams_values.go` — `eca05ff77f711c92e8b42b17ffadc0612fdd9f2cbc4a56d251a805cdfaa2df6c`
* `util_gparams_values_test.go` — `d6ae99138ebc4447e5f4f2107680b3bfa304eed3101b61c92d4b4b3804196f7c`

## Exact test inventory

* `TestGlobalParamsActualNormalizeAndScalarSetGetRoutes`
* `TestGlobalParamsActualSATCollectorCurrentSharedRegistry`
* `TestGlobalParamsBorrowedStringAndSharedSnapshotPublication`
* `TestGlobalParamsCDoubleFilteredPrefixAndIEEEBoundary`
* `TestGlobalParamsCDoubleNoHiddenOverflowErrorAllocation`
* `TestGlobalParamsCLongFilteredSaturationAndNarrow`
* `TestGlobalParamsConstructorUnwindAndColdOwnerGuards`
* `TestGlobalParamsDefaultLookupAndSourceHistoricalMessages`
* `TestGlobalParamsDescriptionBindingThenFailedTablePublication`
* `TestGlobalParamsDescriptionNullAndClosedReadPrefixes`
* `TestGlobalParamsDescriptorMergeFaultRetainsCallerOwner`
* `TestGlobalParamsDictionariesSourceThresholdAndTombstoneProbe`
* `TestGlobalParamsDictionaryResetReleaseBeforeFailedShrink`
* `TestGlobalParamsDisplayParameterDiagnosticBeforeLookup`
* `TestGlobalParamsModuleDescriptionBorrowAndDisplayBranches`
* `TestGlobalParamsNilStreamActualActionPrefixes`
* `TestGlobalParamsRealModuleSnapshotsAndReset`
* `TestGlobalParamsRegistrationTrueBeforeFailureAndActualMerge`
* `TestGlobalParamsResetOwnerDirectoryPlateauAndWarmSnapshots`
* `TestGlobalParamsSetterFaultPrefixesImmediateOwnedGraph`
* `TestGlobalParamsThreeDictionaryLayoutsAndPhysicalBucketOrder`
* `TestGlobalParamsValidationErrorDomainsAndPublishedEmptyModule`

## Reference SHA256

* `src/util/gparams.h` — `bf102cab1d1ad1bba0923daeec78d13ff9cbdb4bd305b7f311c33a3dc92546bc`
* `src/util/gparams.cpp` — `e00c0c7574b5a4e49926b9fe13a0289854c5e0f7b0824b40a3ac8ed039a6e235`
* `src/util/dictionary.h` — `defa2be995dc90b70c12342c23374a55159ce9fabde29ac83c76ddf2661b6289`
* `src/util/map.h` — `62db3b3819aa71f67bababdb3d5583d14fecebbde6f2d22866c28a86215aa850`
* `src/util/hashtable.h` — `bdc562a7db2c0d914dd896577c43e0648f501538873460d2e1a6c2d0932f64f2`
* `src/util/symbol.h` — `60ce765fb3b761ec178ceec7db59f7ddbce00dd0fad5148f6a8ec234f62b4fbf`
* `src/test/main.cpp` — `e18f789f7c4794ed08c2988aa637a778d58b72bb7c1bc616b8710a4c54c9e993`
* `src/test/gparams_register_modules.cpp` — `682377f035394159980e4362ca7d303c6488d0ad17c592c8d4fdc5d40b8ff723`

## Coordinator integration

Coordinator verified all fourteen live/frozen hashes and exact twenty-two named
race/actual JS RUN/PASS records. The integration snapshot
`.cache/review/gparams-integration-2f9931b` has identical complete Go source to
the agent-tested snapshot; full native/debug/vet/WASI gates are reused without
source changes. Updated source/no-generics audit handle91152 passed with
33ported/99partial/1970unported. Only gparams.cpp/h moved to partial; full
platform/catalog wiring and the full solver objective remain open.
