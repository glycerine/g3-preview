# MPFF executable internal32 packet

Source read in full: src/util/mpff.h (482 lines), mpff.cpp (1407), src/test/mpff.cpp (666), id_gen.h, scoped_numeral.h/scoped_numeral_vector.h, vector.h and the applicable bit_util/MPN/MPZ conversion algorithms. Source Copyright (c) 2012 Microsoft Corporation is preserved; helper specializations preserve their 2011 notice. Only the 16 new Go files below and this document belong to this packet. Existing owners, metadata and Git are coordinator-owned.

## Implemented source methods

FastFloat is a pointer-free fixed-precision significand handle/sign plus int32 exponent. This is source MPFF, not IEEE arithmetic. NewFastFloat/default scalar is zero; owning shallow copies are prohibited. FastFloatManager implements the complete selected executable manager families:

- Construction/destruction, precision/capacity block initialization, source reserve-zero VERIFY, owned one, ensure_capacity/expand/allocate, recycle/del/reset, scalar swap, explicit Copy, exponent getter/setter and Precise/Field.
- Both rounding modes and setter/query; sign/zero/positive/negative/nonpositive/nonnegative; integer/int64/uint64 predicates and getters; abs-one/one/minus-one/two; epsilon predicates; all six comparisons; neg/abs.
- Signed/unsigned32 and64 setters with distinct source routes, both fraction setter overloads, source copy setter; MPZ/MPQ setters, significand extraction and integer/rational conversions with either actual manager temporary policy. No hardware float/double setter is invented: the source declares none.
- Internal indexed buffer/shift/bit-test/normalization helpers, inc/dec significand, min/max significand and exponent saturation; epsilon/min/max/next/prev; source add_sub/Add/Sub/Inc/Dec, Algorithm M Mul, single Algorithm D Div, both Inv overloads, complete Power, both Floor/Ceil output forms, both power-of-two queries and previous-power query.
- Raw, ordinary, pretty, decimal and SMT2 displays; to_string/to_rational_string. String variants return one caller-owned final Go string; FormatInto variants publish into caller buffers only after successful staging, exclude NUL, and leave buffers unchanged on error/short-buffer.
- Concrete ScopedFastFloat: lifetime/copy/assignment/swap/neg/abs, every executable arithmetic compound and value operator, six comparisons, five scoped predicates and printing. Value arithmetic preserves source copy-before-compound followed by returned-lvalue-copy and statement cleanup. ScopedFastFloatVector: owning elements, 2→(3n+1)/2 growth, resize/push/pop/shrink/reset/back/at, ascending element destruction and idempotent Close. At/Get are borrowed views invalidated by the documented owner/vector lifetime; no claim that old Go pointers detect every invalidated vector borrow.

The selected internal32 numeric methods above are implemented. The whole source unit remains **partial**: persistent ostream flags/fail-state/custom ostream transport, all TRACE/STRACE (including trace-only Power copies), integration with the external z3_exception transport, and the real subpaving consumer/config/context/tactic remain open. Concrete scopes cover executable MPFF instantiations, not unrelated generic source-manager specializations.

## Physical ownership and source adaptations

Config is explicit FastFloatConfig{PrecisionWords,InitialCapacity,Storage}; DefaultFastFloatConfig uses source2/1024/minus-infinity. Accept precision>=2 including finite odd3/5; explicit little-endian limb composition replaces alignment-dependent uint64 reinterpret casts. Reject width0/1 and signed-bit/byte/offset/host-index overflow as checked-domain adapters; no parity cap. Borrowed Budget must exist. Owners are confined, not concurrent.

The manager charges its actual header, one owned arena.Words header/descriptor/pages, one private NaturalManager header borrowing those Words, contiguous significands, four 2*precision buffers, ID recycle storage, validity/generation sidecar, lazy set buffer and every persistent decimal/format workspace. The cold actual Words graph has11 live vector roots: significands, metadata, recycle, four MPFF buffers and four Natural scratch vectors. The manager's scalar one owns one block. Actual whole-graph byte tests sum nested physical stats without counting referenced Words twice. Caller conversion managers are borrowed and never closed by MPFF.

Source logical block capacity doubles; physical significand/vector requested backing grows2→(3n+1)/2 separately from arena geometry. Publication of logical capacity occurs after backing preparation; typed allocation failure retains completed physical growth and consumed source ID gaps. Allocation preflights actual free-ID backing BEFORE publishing each live block. This is a storage/failure adaptation ensuring lexical Del cannot allocate under an exhausted Budget. Source LIFO recycling and stale scalar bits after Del are preserved; Reset is separate. All builds own a real charged validity/generation sidecar; checked ffRef64 includes generation upper32, release ffRef32 retains packed sign/index31. Generation-exhausted slots retire. Raw numerical handles have implicit manager ownership; scalar handles do not promise detection of every cross-manager/recycled-release misuse. Cold scopes validate explicit provenance/self-copy before actions.

Division stages the divisor in the existing fourth MPFF buffer and invokes ONE existing Natural Algorithm D pass. Indexed offset A/S/M and MPZ SetLimbs/decompose specializations implement the same source algorithms without borrowed slices, interfaces or hidden per-call limb copies. No production math/big, unsafe or Go generics.

Decimal's source local sbuffers become lazy persistent owner-indexed workspaces: requested inline1024 for buffer1/buffer2/power,16 for buffer3, source doubling above that. Actual headers/backing/full old-new overlap are charged. Natural's owned byte workspace replaces the source local char buffer; only the final returned host string escapes accounting. Formatting does not invent lifetime reservation for escaped strings. The exp>=0 decimal branch is unreachable for initialized valid numbers because IsInteger selects ordinary display first; that source branch is not advertised as independently reachable.

Scope/vector cold headers and actual pointer-free element backing are charged. Growth reserves the entire new backing while old backing remains charged. Separate scalar bounds enforce source unsigned32 allocation bytes/header and actual Go host/stride limits (checked records are wider). Source arithmetic/publication prefixes survive errors; no blanket numerical transaction or arbitrary iteration cap is imposed. Internal lexical temporaries are reclaimed immediately with the injector still armed.

## Frozen source quirks and checked domains

- SetInt32(INT_MIN) computes the intended magnitude with explicit unsigned32 arithmetic. The source setter's literal signed negation is undefined for that input; this is a defined intended-magnitude adapter, not a C++-observed result claim. Source signed64 construction already uses its safe magnitude route.
- Ordinary positive exponents remain unnormalized: 2^512 prints `9223372036854775808*2^449` at precision2. Raw text is exactly eight hex digits per source limb. SMT2's small exponent factor misses a separator: exact3/2 produces `(/ 3.02.0)` or `(/ 32)`. Decimal truncation appends `?`; source32/128 defaults and threshold selection remain exact.
- MPZ→MPFF directed conversion increments once for EACH discarded nonzero limb. A separate inferred-source fixture freezes two increments. Its zero-leading-shift SASSERT(k!=0) is a g3debug invariant; release uses defined identity shift. Negative and unreduced raw rational shapes retain reachable finite conversions rather than a blanket canonicality rejection; denominator conversion temporarily flips rounding and restores it even on failure.
- Power preserves all fast paths, min-significand uint32 intermediate multiplication, output aliases and the final extra square. A generic p>=2^31 may overflow BEFORE mask wrap. Execute the actual source loop; only actual mask0 at the next loop boundary becomes ErrFastFloatPowerDomain, with completed result prefixes and lexical cleanup. Fixtures distinguish outward overflow before wrap from inward underflow-to-zero followed by actual wrap. Zero^zero release resets output; g3debug reports the exact source assertion before reset.
- Uint64/Int64 use source debug-only fit predicates. Release finite ±3/2 yields unsigned1/signed±1; actual negative or >=64 shifts and int32 unary-negation overflow receive typed Domain rather than Go zero-fill. Positive2^63 signed conversion explicitly selects a two's-complement host adapter. A raw negative half with defined shift63 yields0; oddprecision3 top64MIN with a nonzero discarded low word takes the source negative INT_MIN branch. Behavioral regression observed release RED on the prior universal fit guard, then GREEN in both profiles.
- ToInteger evaluates its integrality and negative-exponent bounds as source g3debug assertions only. Release finite ±3/2 truncates to ±1; a wide negative exponent zero-fills and yields0. A review regression observed RED on an incorrectly universal domain guard, then GREEN in release/debug with scratch/output prefix checks. The literal INT_MIN int32 unary negation has undefined source arithmetic: the concrete release adapter reports Domain after the completed buffer copy; debug stops at its earlier integrality assertion.
- Next(max) first wraps the significand to minimum then throws with the old exponent; this partial result is tested. Division diagnostic checks occur after allocation/sign publication and before Algorithm D allocation; a wide invalid-significand regression observed RED when scratch failure incorrectly preceded the source invariant, then GREEN after correction.
- Source generic/overflow/div0 exception messages are exact distinct sentinel error categories. Checked ownership/domain/storage errors are separate. External source exception-interface transport remains open; these manager errors do not claim that service bridge is implemented.
- g3debug evaluates actual source diagnostic expressions; release short-circuits their checks. Source VERIFY mutations execute in release as well. The original negative-overflow test typo initializes overflow=true; source coverage preserves that weak assertion and adds a separate meaningful exception expectation.

## TDD and actual source execution inventory

Initial isolated HEAD+test-only RED failed on missing actual FastFloatManager/scoped/set/display APIs before production scaffolding. The full executable packet then became GREEN. Additional vector simulated32host boundary regression observed RED on the missing checked allocation helper, then GREEN. The division diagnostic-order defect had its own observed RED→GREEN.

Source registration main.cpp:220 TST(mpff), src/test/CMakeLists.txt:70 and src/util/CMakeLists.txt:34 is active. Tests execute ALL original finite loop counts: capacity50000 for precisions2/4/8/16; set64 random1000 for2/4/6; all16 arithmetic random campaigns1000 with two exponent variants; exact fixed power/sign/limits/bugs/binary/previous-power/decimal routes. Count assertions remain explicit: **908316 ENSURE +2 VERIFY** (908297+2 in SourceActiveMatrix,5 in SourceBasic,14 in SourceDecimal). Commented tst_add_corner remains inactive.

The hermetic random adapter uses a deterministic nonnegative31bit RAND_MAX stream, preserving actual rand%max, draw/sign/order/campaign shapes. It is NOT an ambient libc sequence reproduction. Source functions are partitioned among tests and the fixed-family invocation order differs from the original registration; within-family statement routes and RNG campaign order/draw counts are retained. This partition is an explicit harness adaptation, not a full stdout transcript claim.

Print-only tst1..7 and tst_div(2) execute their real manager/scoped operations. Tst1 performs actual100 repeated squarings with the source catch shape; default inward positive overflow saturates, so no exception is reached in this finite run. Independent bounded-significand/exponent models check every print result without constructing billion-bit integers. There are **146 normal stdout print events checked as exact isolated line/text fixtures**:117 in SourcePrintedPaths,1 full2^512 line in IntegerConversionSource and28 ordinary+decimal lines in SourceDecimal; its separate14 stringstream decimal assertions remain included above. Original source runtime diagnostic transcript/TRACE and persistent ostream state remain open.

Independent test-only big.Int/big.Rat extraction from actual significand/exponent checks directed enclosure at precisions2/3/5/8 for signed fractions, all four operations and every output alias. Distinct source-defect fixtures retain non-mathematical outputs. Fault sweeps cover constructor, arithmetic/fraction/power/copy, conversions, all formatting modes, scoped returned copies and vector publication/growth to observed eventual success. Before any rendering/Del/Close they audit live block IDs/sidecars, all MPFF/Natural Words roots, owned text roots and actual external integer/rational cell/payload graphs. Inactive blocks are zero. No teardown-only leak claim. Tests also cover borrowed owner/self-copy rejection, LIFO IDs, generation retirement, source vector capacity/destructor order, full graph charges, old-new peak refusal, short buffers, round-mode restoration and warm prepared arithmetic/all FormatInto zero host allocations/reservations.

## File ownership and verification

The16 new Go files are mpff.go, mpff_arithmetic.go, mpff_convert.go, mpff_convert_test.go, mpff_format.go, mpff_format_test.go, mpff_limbs.go, mpff_ownership_test.go, mpff_predicates.go, mpff_ref.go, mpff_ref_checked.go, mpff_source_matrix_test.go, mpff_test.go, scoped_fast_float.go, scoped_fast_float_test.go and scoped_fast_float_vector.go, all under mpnum. This status is the only additional document. All writes/cache/temp/executables are under golang; no existing owner edits, Git writes or central metadata writes.

Final prospective tree is golang/.cache/release/numeric-fast-float-final, archived HEAD fa840a5 plus ONLY these files. Final gate results are appended after terminal completion. Source files remain partial for the external stream/trace/exception/consumer obligations above; AST, subpaving/polynomials, alternate float engines and global resource policy are not provided by this packet.

Final corrected frozen Go bytes (including both release/debug getter and ToInteger review fixes) terminal gates:

| Gate | Result / handle |
| --- | --- |
| Isolated full native ./... | PASS76896, mpnum21.526s |
| Isolated full g3checked+g3debug ./... | PASS33167, mpnum22.175s |
| Isolated full g3checked+g3debug race ./... | PASS15048, mpnum247.287s |
| Isolated full actual checked/debug JS ./... | PASS88174, mpnum66.538s |
| Actual checked/debug JS named22 MPFF tests | PASS74111,22 actual RUN/PASS,30.429s; source matrix27.20s |
| Actual release JS getter/truncation branches | PASS93165,2 actual RUN/PASS,0.149s |
| Named checked/debug race final getter/truncation branches | PASS24023,2 actual RUN/PASS,1.027s; earlier15 independent/fault/provenance names PASS99261,12.292s |
| Full checked/debug vet and source/no-generics verification | PASS86848, prospective pre-coordinator-metadata2102units677APIs33ported65partial2004unported |
| All-package checked/debug WASI cross-build | PASS64428, unique package artifact names under golang/.cache/wasi-fast-float; corrected mpnum rebuilt PASS4998 |

The coordinator's subsequent central ledger changes are independent of these frozen Go bytes and are separately audited by the coordinator. Earlier superseded full native/check/race/JS runs were repeated after concrete review corrections; the table identifies terminal corrected-byte evidence. No live owned gate sessions remain.

Named actual JS registrations are TestFastFloatIntegerConversionSource, RationalRawShapeAndConversionFailures, IntegerSourceDiscardedLimbDefects, ToIntegerSourceReleaseTruncation, GetterSourceReleaseActions, SourcePrintedPaths, IndependentDirectedAliases, ConstructorReservationPrefixes, ArithmeticFailureRoots, PowerWrapPrefixes, FormattingFailureAndWarmStorage, IDRecyclingAndRetirement, ScopedFailureLifetimes, DivisionSourceDiagnosticPrefix, ActualWholeGraphCharges, SourceActiveMatrix, SourceBasic, SourceDecimal, VectorCheckedAllocationBounds, OwnerProvenanceAndVectorLifetime, NeighborAndThinRoutes and VectorOldNewPeakBudget (every abbreviated item also has the TestFastFloat prefix).

Frozen Go SHA256 inventory:

```text
7cf5af3c555c9de336a55f3ee88039a2feacffcdf1a888c51eeb40e524601520 mpnum/mpff.go
83ac40e78b59843afb6112aedafb53e5af19ab7e8da58696e5c20d7a8f2c032a mpnum/mpff_arithmetic.go
4df215e19b89efe2a5f281e054217810325d5a6e18a5a954cd9607b06cfd3cf7 mpnum/mpff_convert.go
617149715a1df615db292fb21d2d22614db30aa179c89858060367ba1648586e mpnum/mpff_convert_test.go
373968bc2bf0ca5a79e487fd42d344b26749eb0239efa7aed0c16c9da3026fa2 mpnum/mpff_format.go
6c839152a82e8fe4963f3dda6d1470e25160857ffebab1b4502756764ba3eba9 mpnum/mpff_format_test.go
bc45e3c1087fabd20f88fb9894adc4811ebe6f0a581d29fe701175497bb8f0b2 mpnum/mpff_limbs.go
a951319957548ea9013df82e8b079fbbae29d64c2e0a639332818260f0682c84 mpnum/mpff_ownership_test.go
8a7e2ad5b033116ee6d61296c2b07622f2955747290cda7a28bd4c1b3f631490 mpnum/mpff_predicates.go
cd063e4b648ecdb823bf884c62a6e4c153f0ad72e034be1110202ae33d0be596 mpnum/mpff_ref.go
32706f5b924ed7af90734ac498d7539fe47c98fba74b41ff03d3724b654dcd7e mpnum/mpff_ref_checked.go
e427933a53b610962f63726ef83e95b40034f5c7d987d5751bdd45075c702f25 mpnum/mpff_source_matrix_test.go
87d6239ea10deacc425b1f4a5a4aaf52e45fea29ea2a59b86926ec3b7a8e2b5e mpnum/mpff_test.go
98bf8f7dbeb398b8bc3136ce4a349f53600c3b1616543615ee44a003a824a977 mpnum/scoped_fast_float.go
8e182567e732d0cf579bc8673c60ad729f9bfee9880d193acddff68626629c17 mpnum/scoped_fast_float_test.go
e93c250bb5cd5c2de2da2ae0daf6fe81920a8da7b22aaf497668541bad231b6e mpnum/scoped_fast_float_vector.go
```

All16 Go files and this document are frozen for coordinator review/commit. No central metadata, existing owner or Git edits were made by this packet.
