# Parameter descriptor byte-stream dependency

Actual source params.cpp imp::display is translated additively by ParamDescrs.DisplayInto(dst *StringBuffer64,indent uint32,smt2,includeDescr bool) error. Existing ParamDescrs.Display remains unchanged; output parity applies to its valid input domain. This real helper unblocks gparams descriptor/diagnostic composition without an uncharged intermediate Go string. It does not implement the global parameter registry, generated catalog or full solver.

Owned NEW four Go files only: util_param_descrs_stream.go, util_param_descrs_stream_debug.go/nodebug.go, util_param_descrs_stream_test.go. No existing owner/service/metadata/Git files modified. GPARAMS_PORT_STATUS.md remains parent-owned.

Source mapping: actual dictionary USED entries are collected in source bucket traversal order; temporary names vector has geometric 2,3,5,8 growth and exact uint32 source capacity/count/byte arithmetic. Source Symbol pointer width 4/8 (borrowed manager profile) is separate from actual Go Symbol stride; source vector.h old/new byte-wrap guard is checked by pdStreamNameCapacity, followed by real host bounds before actual typed allocation. Actual typed scratch backing bytes are charged to descriptor Budget; the temporary slice header is local stack metadata, and the source eight-byte header participates only in source extent arithmetic. Both old/new backing roots coexist during growth, old charge is released only after successful make/copy, and all temporary backing is refunded on any return. Existing concrete sortSymbols comparator/order/backend is reused; no claim exact std::sort external allocation/comparison sequence.

Each output row preserves source order: uint32 indent loop; optional colon; borrowed symbol spelling with '_'→'-' for SMT2, '-'→'_' otherwise and ASCII uppercase→lowercase; actual dictionary lookup; debug-only description assertion; kind; optional description; optional default; LF. Empty default differs from null. All spelling/metadata bytes read directly from real SymbolManager region/descriptor Words. No Symbol.String, builder/intermediate final text, generic sort/interface boxing or uncharged key slice.

Lock order is destination→descriptor→symbols. Only private destination appendByte/appendCString run under its lock, so no exported reentrant path acquires inverse symbol→buffer order. The cold destination is borrowed, remains owned/charged by its caller, and no backing slices escape. Shared external Budget remains owner-confined; individual locks do not synchronize independent owners that share it.

Source undefined/null boundary adapters occur at actual actions: a null/numeric spelling is guarded when source bare-str/strlen would read it (after emitted indentation); source debug SASSERT(d.m_descr) is diagnosed AFTER name emission. Release includeDescr=false never dereferences a null description and emits valid complete row. Release includeDescr=true emits kind+space then returns ErrParamsStreamFailure at null CString output. Source ostream persistent failbit/custom IO are not modeled by StringBuffer; typed return preserves first failure prefix and stops this helper, rather than claiming persistent external ostream state equivalence. All bounded failures retain actual emitted bytes and refund only temporary names. UINTMAX indentation is not arbitrarily capped; finite real destination growth refusal stops it after a real prefix.

Tests-first: initial isolated missing DisplayInto API/profile RED before production. Scalar scratch-width correction also first missing helper RED, then meaningful behavioral RED using prior pdGrowth: old357913941,width8 returned536870912,nil despite source byte-wrap overflow. Exact uint32 source arithmetic corrected it. The four ordinary groups plus independent scalar boundary group pass:
- TestParamDescrsDisplayIntoSourceBranchesAndSort: source labels/sorted original spelling, normalization, append prefix, SMT2/include-description combinations, null/empty defaults and embedded-NUL metadata, parity to existing Display in valid domain.
- TestParamDescrsDisplayIntoNullDescriptionActualAction: debug assertion after name, release finite omitted-description case, release requested-null output-space/failure prefix.
- TestParamDescrsDisplayIntoBudgetsScratchAndHugeIndentPrefix: immediate sort scratch refusal/refund and unchanged output prefix; UINTMAX indent reaches actual finite destination growth fault at64 bytes, immediate physical owner/overflow accounting.
- TestParamDescrsDisplayIntoGuardTimingAndIndependentOwner: singleton null key guarded after real indentation, nil/closed/bad-self identity. Invalid owner fixture constructs mismatched self explicitly instead of copying a Go mutex; earlier test-copylock vet failure was fixed before freeze.
- TestParamDescrsDisplayIntoSourceScratchCapacityWidths: independent source-width4/8 scalar counts around source byte-wrap transitions and UINTMAX; no huge backing allocation, and no claim these scalar checks execute a multi-GB payload.

No fabricated registered source tests/ENSURE counts. The source method contains actual debug SASSERT(d.m_descr), whose finite branch/prefix is covered; other tests are independent source-action/lifetime/storage fixtures. Full existing parameter descriptor tests remain unchanged.

Final frozen snapshot .cache/release/pd-stream-width-final, HEAD fb08673fd1bf9a4310c731ed543d692d263553f3 + ONLY four helper Go files. Current test hash below supersedes old copylock fixture. Older .cache/release/pd-stream-final17 groups and earlier width-final oldtest18 gates are historical and not final evidence. Final test names .cache/pd-stream-width-names.json/regex; hashes .cache/pd-stream-width-hashes.txt. All18 combined existing+new names explicitly RUN/PASS in final race/JS logs.

Terminal final gates:
- full native ./... handle32695, .cache/pd-stream-guard-native.log, exit0
- full checked-debug ./... handle57607, .cache/pd-stream-guard-debug.log, exit0
- named18 checked-debug race handle40830, .cache/pd-stream-guard-race.log, PASS2.991s exit0
- actual checked-debug JS named18 handle41177, .cache/pd-stream-guard-js.log, PASS0.562s exit0
- full checked-debug vet, final exec terminal exit0, .cache/pd-stream-guard-vet.log
- full checked-debug WASI build, final exec terminal exit0, .cache/pd-stream-guard-wasi.log
- final source/no-generics audit handle93638, .cache/pd-stream-guard-audit.log terminal exit0,33/91/1978

Frozen Go SHA256:
- util_param_descrs_stream.go:ddfba5248c8a2f46ad1d7b5ee669182c06bf76111772cfaf467adb4a79ac716f
- util_param_descrs_stream_debug.go:3e7d3002e0d259ac093b774ea9d8d51bc0dde855a74c6229591e41d1ef44e80b
- util_param_descrs_stream_nodebug.go:a7c4cd38ca69f4efabf312d9f7ee5d9880c504a351e34f450ddc656efb0ff034
- util_param_descrs_stream_test.go:a99cea4ff641e6fae8d84f9f85e811d95313b11133b94e6f1b41146503b6c625

Source SHA256:
- src/util/params.cpp:adfec5df1efec4cd4d08844d06463b58096d1c91e0374e7e432a5f1d9bb70dd9
- src/util/vector.h:aafafc0fe6ca0afd7bafe0c781342da259f9c34c0b94b16732c415cf2733066e

Root review and integration: .cache/review/pd-stream-guard-integration-fb08673
contains the identical final four Go files on the identical committed base,
with extended existing params.cpp/h method metadata. Full agent gates therefore
apply to this same Go tree. Root source/no-generics2595 terminal exit0, counts
33ported/91partial/1978unported. Exact Go hashes and every18 named race/JS
RUN/PASS reverified before commit. Prior root17/oldtest18 audit snapshots are
superseded. Root corrected accounting prose to distinguish stack slice metadata
from the charged physical backing and source-only eight-byte extent header.
