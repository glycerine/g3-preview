# MPZ source publication reconciliation

This packet corrects selected IntegerManager allocation/publication actions and cleanup continuations. It does **not** complete mpz source topology, global memory accounting, or every higher-level numeral wrapper's setter order. Existing staged wrappers in mpq_ops, integer/rational root operations, and other unreviewed callers remain separate work. Numeric cells still use charged concrete descriptor/Words storage; these physical allocations must not be reported as source small-object allocator malloc events.

## Source mapping and completed actions

- `src/util/mpz.h:156–163` `ensure_tmp_capacity`: free insufficient old scratch first; compute `(3*size+1)>>1` with unsigned32 arithmetic afterward; allocate/publish replacement; perform the source final capacity assertion in debug. Allocation helper capacity-six assertion precedes allocation (`mpz.h:123–127`). Missing scratch is a scalar zero and rejected at its next actual scratch access.
- `src/util/mpz.cpp:274–279` result setter: publish sign, swap output and scratch references, publish size, then request missing scratch. A successful output stays readable and releasable after replacement refusal.
- `src/util/mpz.cpp:1183–1207` big_set: publish sign and free insufficient destination before replacement allocation. `mpz.cpp:315–334` unsigned digit setter publishes positive sign before its old-free/allocation prefix.
- `src/util/mpz.cpp:1556–1594` ensure_capacity retains old storage until a replacement is prepared. Its distinct lifetime is independently tested.
- `src/util/mpz.cpp:503,505` paired division: quotient setter precedes remainder setter; no preallocation of both replacement scratches.
- `src/util/mpz.h:468–497` Div/Mod wrappers operate on the caller destination and reread the possibly aliased divisor at the source adjustment action. Source successful alias quirks remain tested.
- All result-setter callers were audited: add/sub (`mpz.cpp:374,394,407`), multiply (`459`), paired division (`503,505`). Actual required scratch span is checked at the former resize/access preparation in add/sub, multiply and division, after the source allocation prefix.

## Explicit Go continuations and ownership

One permanently dead, retired descriptor per IntegerManager represents a failed replacement after an old destination was freed. It is fully charged auxiliary storage, never a live source cell or malloc event. Int's scalar layout is unchanged. Constructors prepare it before publication and unwind every charge on refusal. Read checks reject it. Del/Reset accept only this owner's exact failed token or an ordinarily live value, and preserve rejection of ordinary stale references. The owner closes the descriptor ledger once.

The wrapped-capacity helper can perform a real finite allocation smaller than the requested source span. At the actual span access Go returns ErrBounds without undoing the completed replacement or inventing invalid memory writes. Debug assertions retain their earlier source positions. A missing scratch does not poison valid published outputs and is not automatically repaired for retries.

RationalManager, BinaryRationalManager, BinaryIntervalManager, and RationalService cleanup preflights accept an owned failed leaf while retaining normal read guards. InfIntDel, InfDel, and InfEpsInfDel similarly validate provenance and cleanup each component. Nested Inf provenance remains checked. Live companion components are released exactly once; service tokens become invalid on Del. This recovery surface is a Go continuation after source-undefined allocation failure, not a claim that C++ recovers from those states.

## Tests-first evidence

Actual behavioral RED covered scratch free-before-allocation, result publication before scratch replacement, big_set/digit old-free, paired publication, Div/Mod destination action, wrapped-capacity assertion order, and partial rational/binary/service/Inf cleanup. The required span helper first produced a missing-method RED, then its independent scalar-span regression verified completed replacement roots stay intact. The initial ensure_capacity retention fixture passed immediately, honestly testing already correct behavior.

The new publication file has 16 named groups:

- `TestIntegerSourceTmpFreeBeforeReplacementFailure`
- `TestIntegerSourceResultPublishedBeforeMissingTmpFailure`
- `TestIntegerSourceBigSetFreesOldBeforeAllocationFailure`
- `TestIntegerSourceDigitSetterFreesOldBeforeAllocationFailure`
- `TestIntegerSourceEnsureCapacityRetainsOldUntilReplacement`
- `TestIntegerSourceFailedDescriptorConstructorAndUnwind`
- `TestIntegerSourceSmallDestinationFailureKeepsSignAndInvalidToken`
- `TestIntegerSourcePairedDivisionPublishesQuotientBeforeMissingScratch`
- `TestIntegerSourceTmpUnsignedCapacityGuardAfterOldFree`
- `TestIntegerSourceDivModWrapperDestinationBeforeFailure`
- `TestIntegerSourceTmpFiniteWrappedCapacities`
- `TestIntegerSourcePartialRationalCleanup`
- `TestIntegerSourcePartialBinaryCleanup`
- `TestIntegerSourcePartialOwnedRationalCleanup`
- `TestIntegerSourceScratchSpanGuardAfterFiniteWrappedAllocation`
- `TestIntegerSourcePartialInfCleanup`

Obsolete failure atomicity/retry sections only were corrected in existing integer tests; mathematical/source success fixtures remain. Immediate named-root audits include actual source scratch/constants/public outputs and NaturalManager payload scratch, exclude only the exact owned failed token, reject ordinary invalid roots, and compare the complete live cells and Words graph before formatting, fault disable, or teardown. Three scoped slot-fill loops now count Live+Retired, following observed nontermination when the new retired token was omitted.

Historical evidence: initial RED native `.cache/mpz-source-publication-red-native.log`; paired RED62418; wrapper/wrapped-capacity RED11515; composed cleanup RED43362; all three Inf cleanup subcases RED9609 then GREEN37375. New selected source groups native GREEN75784. Existing five division/GCD/fused groups GREEN6840 and four bitwise/facade/power groups GREEN36734. These targeted gates are not final whole-module evidence.

All writes, build cache, temporary files and snapshots for this packet are below golang. No Go generics, unsafe, production math/big, metadata edits or Git mutations were introduced by this agent. Final frozen hash and gate appendix follows after combined validation.

## Frozen selected packet and terminal targeted gates

Owned Go inventory: eleven production and eight test files (19 total), frozen SHA256 inventory at `.cache/mpz-source-publication-owned-hashes.json`. The isolated snapshot is `.cache/release/mpz-source-publication-final`, committed baseline `4b9337e` plus only those 19 files. Source tests were checked against exact names, not merely a package PASS.

The first tagged runs correctly caught a fixture bug: BinaryRationalManager requires a real diagnostic RationalService under g3debug. Only the new test fixture was corrected to construct/charge/close that service using the shared Budget; production stayed unchanged. The final publication test hash is `8af8b09c66f034d14e0eb6631d855b0013f6f5fd89673234e4acd1f7a0852205`.

Final targeted checked-debug77396, checked-debug race52556, and actual checked-debug JS93853 all terminated exit0. All sixteen new top-level names explicitly RUN/PASS in each log: `.cache/mpz-source-publication-target-checked.log`, `-race.log`, and `-js.log`. These selected gates do not substitute for the parent-owned final full-module, vet/WASI and dependency integration checks.

Full workspace mpnum diagnostic80279 terminated exit1 in22.856s with no hang. Three remaining sibling-owned old assumptions were reported to the parent: NumberTheoryRationalResetPrefixAndOutputAliasing, BinaryIntervalFailurePrefixes, and BinaryVectorFailurePrefixesAndScopedRoutes. Parent coordinates their source-prefix review; this agent has not narrowed or patched those tests.

Supplemental read-only caller audit found no hidden scratch repair in the selected core. Successful guards occur at actual scratch use. Existing bitwise operations still stage output in a local work Int and Move at return (`mpz_bitwise.go:119,184`), unlike source reset/set caller output followed by iterative add (`mpz.cpp:1034,1046,1161,1176`). Their mathematical tests and immediate root audits do not prove source intermediate publication. This requires a separate first-failing source action fixture and production reconciliation, explicitly OPEN. GCD's zero Copy+Abs fastpath is likewise staged in existing `mpz_gcd.go`; other staged wrapper debts listed above remain OPEN.

Frozen owned Go hashes:

- `mpnum/mpz.go`: `ac4be429c3c03fc9c7911d94e2be3fd8f9125e55da3aa15f735dd4a75a478e83`
- `mpnum/mpz_slab.go`: `3b8baa2daf97b5ada426f22e63434262d6895c156d75989f42e15be253a16190`
- `mpnum/mpz_div.go`: `bee5c5d952c8074f880d0fe5042822bf374c63bcae00c462929e22dcb0861af1`
- `mpnum/mpz_mul.go`: `e1ad78c8e215439457ec6e8e1b7b466900fb563d726f033818450d7617ea5f41`
- `mpnum/mpq.go`: `475f5e0918410f79de9e6079483c299b0f8bdc9f3c12ff90f0f42bf317e6dc61`
- `mpnum/mpbq.go`: `4f1f7794faa3c47a497463a519fc64ff08948a768e0a5546dd884e078e0b0837`
- `mpnum/mpbqi.go`: `ff4a28f4149eee8389e85b8ff8aa36c1b1449127e44b3784f01ca4c647caf994`
- `mpnum/rational_service.go`: `2d6a195c3e862d3856a4d209e4d61775cd1c68df7700d1b22b8b83df0287499e`
- `mpnum/inf_int_rational.go`: `aec17a34a1260c97d695637bccacc5baa8cda45f4841d49837c2d218e9451c9a`
- `mpnum/inf_rational.go`: `3e6b54e7a104a5efd597b1ba073e572a568af23079de3b922f7e1f8d1d6aaa02`
- `mpnum/inf_eps_rational.go`: `9cfe7a412436b696638e522f2cfefc5292c021071387a5851e2ea65bdef61edb`
- `mpnum/mpz_source_publication_test.go`: `8af8b09c66f034d14e0eb6631d855b0013f6f5fd89673234e4acd1f7a0852205`
- `mpnum/mpz_test.go`: `8c16b8b2a7484879df114345ca38603f94885dd833011a161766727647546509`
- `mpnum/mpz_mul_div_test.go`: `d3ff2a9956a50e6c18e2eb77426721e04eb0a05dc03c2d64ec293c2fbb99bd08`
- `mpnum/scoped_numeral_test.go`: `5358304856ec9e2c0ccfa3403467a44968b11f332dfa0abbffdda9cf5e70ce0c`
- `mpnum/mpz_bitwise_test.go`: `48518495e5cbf33eda965b0d5989ca767b05d598e4238825a0627f5f44f39bf9`
- `mpnum/mpz_facade_extra_test.go`: `5389a6b01d37822963b3dd9f0515255a9d893a407627e002d3a4a30cccb2d68e`
- `mpnum/mpz_gcd_test.go`: `30ae094b494e6a833f46771ed5d082fc30177596ab97fff961ab033f4ba3ef25`
- `mpnum/mpz_power_shift_test.go`: `79ab65cab8051ca584cd5eb3652b6998bd48226ad68745e3694734536814e4e3`

## Root frozen integration

Base `4b9337e` plus 37 selected Go files integrates the source packet and its
actual numeric, polynomial and S-expression callers. Full native passes
(root2.745s/mpnum22.417s); full g3checked,g3debug passes
(root3.073s/mpnum23.267s). All234 named groups have exact RUN/PASS entries
in checked/debug race (root1.488s/mpnum122.591s) and actual checked/debug
JS/WASM (root0.309s/mpnum30.281s). The initial120s race observation reached
its real test timeout; a600s gate passed unchanged numerical bodies.

Vet, all-package WASI build and separate root/mpnum WASI test compilation pass;
WASI is compilation evidence, not runtime execution. The source/no-generics
audit passes2102units/677APIs,33ported/145partial/1924unported. Compiler escape
output was inspected; actual warm allocation tests remain executable. Frozen
code hashes and named-gate coverage are recorded in
`.cache/mpz-final-inventory.json`; gate logs are `.cache/mpz-final-{native,debug,race,js,vet,wasi,audit,escape}.log`.

The root review independently observed and fixed polynomial diagnostic
miscounting of the dead token, a source slot-fill fixture that could never
saturate retired metadata, the failed second coefficient Copy read guard,
and a checked/debug S-expression comparison test that assumed all numeric
scratch survived refusal. Its final audit enumerates service constants/cache,
manager scratch, validated caller values and actual NaturalManager payload
roots before fault disable or formatting. No live-allocation scan justifies
an otherwise unreferenced temporary.

The tested tree excludes unrelated ongoing missing-API solver/Sturm tests.
No claim that the live development worktree or full solver port is complete
is supported by these selected integration gates.

Cache-only warmed Mul+MachineDivRem benchmark uses independent initial arithmetic
checks and distinct reusable outputs: three native200ms samples have median
3002ns/op,0B/op,0allocs/op; one checked/debug sample3133ns/op,0B/op,0allocs/op.
Whole physical Budget plateaus at3888B native/3920B checked (header-size
difference), including cells, Words and four real division scratch roots.
An armed zero-reservation warm loop succeeds and Close refunds all charges.
These local Go measurements are not a C++ performance comparison. Evidence:
`.cache/perf/mpz-publication-{native,debug}.log` and summary JSON.
