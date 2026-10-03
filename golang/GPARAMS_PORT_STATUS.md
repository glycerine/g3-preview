# Actual global parameter registry packet

Registry is under active tests-first implementation, not complete. Source gparams.cpp/h full method inventory and ownership/API proposal is SAT_CONFIG_PACKET.md; config production awaits actual runtime module service. Initial real registry tests reached missing-API RED in isolated .cache/release/gparams-red: undefined GlobalParamsConfig/Registrar/NewGlobalParams/ErrGlobalParamsOwner. No caller-injected snapshot or fake default registry implements GetModule. Full generated catalogs, historical diagnostics and display dependencies remain required and must be classified individually before any whole-unit completion claim.

## Borrowed-symbol Params string dependency: frozen helper

Source gparams.cpp301–322 CPK_STRING interns symbol(value), then calls set_str on its bare pointer. This helper translates that retained pointer to a canonical pointer-free Symbol held in the existing paramEntry.symbol field with paramsText.borrowed=true. No second Words payload is allocated; borrowed symbols must outlive all copied/shared ParamsRef values. This extends the existing copied-string adaptation without silently using it for actual gparams source pointers.

Changed only util_params_ref.go and NEW util_params_string_symbol_test.go. Added SetStringSymbol(v,key,spelling), restricted to actual same-owner plain nonnull symbols; init/COW uses the existing real source Ref operations. Copy retains borrowed text metadata and canonical identity. Owned string setter/storeText remain the existing owned spelling path. delEntry frees only owned text. GetString and appendValue/display dispatch borrowed text directly through actual SymbolManager backing bytes into charged Params format workspace; only final returned string is caller-owned. Lock order remains Params→Symbol, like existing parameter lookup/render operations. No Symbol.String/BareString intermediate and no per-record Go pointer/closure/interface. The cold borrowing service/Budget remains externally confined; symbol teardown concurrent with an active borrower is forbidden.

The actual added bool is included in reflect.TypeOf(paramEntry{}) accounting, including page old/new peaks. It currently fits prior padding but no synthetic size assumption is made. One production positional paramsText literal was converted to named fields; no existing tests were changed. No metadata/Git/shared service edits beyond coordinator-approved util_params_ref.go.

Tests FIRST observed isolated missing-API RED for SetStringSymbol and borrowed tag before production. Four new actual test groups:
- TestParamsBorrowedStringSymbolCopyAndOwnership: real scalar token/kind/tag, getters/display spelling and NUL, Copy retaining canonical identity, no owned Words growth, source Del vs surviving independent borrowed token.
- TestParamsBorrowedStringSymbolReplacementAndWarmSet: frees replaced owned Words root, distinguishes present-empty spelling from null, warm scalar replacement 0 Go allocations with fault controller armed, existing owned setter preserved.
- TestParamsBorrowedStringSymbolProvenanceAndLifetimeGuards: null/numeric/marked/foreign guard before mutation, closed borrowed owner rejected at getter, cleanup never dereferences/frees borrowed spelling.
- TestParamsBorrowedStringSymbolSharedCopyFaultPrefixes: 12 COW fault prefixes, source immutable/shared token preserved, no owned spelling allocation, immediate actual full Params graph accounting before getter/teardown, successful destination token distinct.

Frozen helper snapshot .cache/release/params-symbol-final, HEAD1c2bb3358dda8be001b68326278fc588fd9e2129 + ONLY these two Go files. Separate registry tests/drafts excluded explicitly. All seven gates terminal exit0:
- full native ./... handle42712, .cache/params-symbol-native.log
- full checked-debug ./... handle52751, .cache/params-symbol-debug.log
- named22 existing+new Params checked-debug race handle47077, .cache/params-symbol-race.log, PASS1.160s
- actual JS checked-debug named22 handle81027, .cache/params-symbol-js.log, PASS0.220s
- checked-debug full vet handle49141, .cache/params-symbol-vet.log
- checked-debug WASI full build handle50062, .cache/params-symbol-wasi.log
- source/no-generics audit handle87750, .cache/params-symbol-audit.log,33/88/1981
All22 test names explicitly RUN/PASS verified in both race/JS logs. Inventory .cache/params-symbol-names.json/params-symbol-regex. Parent final integration onto newer MPZZP HEAD remains separate evidence.

Frozen SHA256:
- util_params_ref.go:e349fd21782b977ef0bfa4b7827472f16c6efbef5d2c6f3c7c90a314b23978e5
- util_params_string_symbol_test.go:010c7331479227590d835c0d8d9deb407e154afac82a00aae018af4edcfc4cfb

This completes only the real borrowed-symbol text prerequisite, not gparams registry/config/full solver. No invented source ENSURE or registered source-test counts; these are source action/lifetime and independent fault fixtures.

Root integration: immutable .cache/review/params-symbol-integration-1ccea0b includes
only the two helper Go files on committed HEAD1ccea0b. Full native12834 and
checked-debug24918 passed; source/no-generics58761 passed33/89/1980. Exact
agent Go hashes and all22 named race/actual-JS RUN/PASS results reverified.
Uncommitted registry/substitution/full-solver drafts were excluded.
