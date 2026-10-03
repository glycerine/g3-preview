# Integer/C-string printf dependency packet

Owned files: util_printf.go/test and this status only. All cache, probe source,
compiler temp/output, and Wasm binaries remain under golang/.cache. No existing
exception/manager/manifest/progress files or git state were edited. Production
uses concrete argument types and byte storage; no fmt.Sprintf, generics, unsafe,
cgo, production big integers, or retained boxed argument records.

## Selected implementation and source mapping

| Source | Concrete implementation |
| --- | --- |
| warning.cpp:106-113 local svector<char>/initial resize | PrintfFormatter.Format initial logical1024 for PrintfNonWindowsAMD64;128 for PrintfNonWindows32; per-call vector destroyed on every exit |
| vector.h:60-105 expand_vector and422-435 resize | resize uses capacities2,(3old+1)/2, actual byte arrays and old/new budget peaks; logicalsize separate from physicalcapacity |
| warning.cpp:114-121 BEGIN/VPRF/AMD64 END | pfRender explicit-argument C integer/string conversion domain; Go diagnostic Begin/Call/End instrumentation only |
| warning.cpp:123-127 negative result fallback | malformed backend/trailing conversion and signed count overflow give negativecount; AMD64 returns raw first-NUL format after End |
| warning.cpp:128-133 one-call truncation/ostream | bounded sink counts full output, stores at most logicalsize-1, emits first-NUL byte prefix; no UTF8 repair |
| warning.cpp:135-142 other-platform retry | NonWindows32 retries at logicalsize2n+1, same source vector expansion; stable explicit argument replay is the Go va_list adapter |
| z3_exception.cpp:61-68 formatted constructor | additive NewDefaultExceptionFormatted delegates formatter and constructs immutable DefaultException; selected integer/string domain only |

The formatter/default formatted constructor remain **partial**, not generic
C-printf complete. The supported actual integer/string routes are available;
warning globals/streams/prefix/flush/warning_displayer and command runtime remain
separate units. No fake CLI or warning I/O service is introduced.

## Platform/domain classification

Profiles are explicit and independent of the Go build host: NonWindowsAMD64 is
Linux/glibc LP64 and C locale, NonWindows32 is ILP32 with the source128 grow loop
and selected glibc C-locale conversion policy. Windows secure CRT invalid
parameter installation, previous-handler restoration, and persistent
_CrtSetReportMode state are **open**; config does not accept a Windows profile.
Linux BEGIN_ERR_HANDLER/END_ERR_HANDLER are empty macros. Observe events are
Go-only diagnostics, not claims of actual source callbacks/global CRT effects.
Observe environment is borrowed, must not reenter owner methods or mutate args,
and is not charged beyond the actual func field in the owner header.

Supported conversions: d,i,u,o,x,X,s,c and plain %%. Integer flags -, +, space,
#,0; literal/star widths and literal/star precisions, negative star width meaning
left alignment, negative precision ignored. String precision counts bytes,
C strings stop first NUL, char narrows to its byte and may insert NUL while full
output count continues. hh/h narrow32-promoted values; default32; l/z/t use the
profile32/64 width; ll/j use64. Explicit arg tags validate count/type domains;
extra args are ignored. Signed/unsigned corresponding32 values are accepted
only in representable default-width domains; narrowed hh/h use low bits.
Unsigned64 values and signed64 values retain their separate tags. Numeric digit
rendering uses a stack64-byte buffer, not host formatted intermediate strings.

GNU I flag is a no-op alternative-digit selector in the fixed C locale.
**%I64u means I flag + width64 + unsigned32 in the Linux profile**, not a
Windows64-bit length. The actual datalog callsites pass uint64_t and have a C
varargs type mismatch. PrintfUInt64 therefore receives ErrPrintfArgument; an
explicit caller-chosen PrintfUInt32 conversion demonstrates the observed low32
LinuxABI diagnostic without defining the original C undefined behavior.
Those datalog consumers are not yet ported and no complete support is claimed
for their original mismatched argument. The true Windows I64 length remains open.
Nullable CString uses the selected glibc behavior: (null), but explicit precision
0..5 emits empty and6+ emits the full six-byte label. This is a defined Go/glibc
profile adapter for source undefined null-%s behavior, not a portable C guarantee.

Legitimate unsupported C domains (f/F/e/E/g/G/a/A, long double, wide lc/ls,
positional arguments, locale grouping, GNU m, pointer p, count-write n) return
ErrPrintfUnsupported, never fabricated negative-backend/raw-format fallback.
Raw Go pointer exposure/count writes are not supported. A future safe p token
and typed n target adapter needs its own semantics/lifetime/action tests.
NonWindows32 persistent malformed/count errors receive typed finite failure;
the source loop would otherwise retry forever. Backend malformed/count errors
and unsupported/argument adapter failures have distinct diagnostic categories.

## Budget/lifetime and action prefixes

PrintfFormatter reserves its actual reflect-size cold header at construction,
self-guards shallow Go copies, and serializes calls with a cold mutex. Borrowed
Budget is caller-serialized if shared outside this owner. Close releases header
once; closed/foreign owner calls reject before allocations. Args/string pointers
are borrowed stable for one synchronous call, never retained by the owner.

Each call allocates the exact source byte-vector capacity sequence, reserves new
array while old is charged, copies, then releases old. The byte workspace is also
the bounded formatter output buffer and is fully charged. Final immutable ostream
text transfers to the caller-owned output boundary, as with other display APIs;
it is not a retained owner graph. All temporary buffers are destroyed on failure,
normal return, or observer panic. The owner retains no warmed scratch cache.
Source cap1065 for logical1024 has peak1775 payload bytes (710+1065); tests show
owner+1774 fails and owner+1775 succeeds. Logical128 retains cap140, peak233.

Initial growths complete before Begin. Call occurs once for AMD64 even on output
truncation; End precedes normal/fallback output. NonWindows32 logical sizes are
128,257,515,1031,2063,... and capacities140,315,710,1065,2397,... as required.
A growth failure after a completed Call retains observer prefix with no invented
End event. Typed unsupported/argument errors are diagnostic Go adapter exits,
not pretend backend failures. Formatting count is bounded signed32 independently
of storage. Large padding/precision spans count in O(1) while only bounded bytes
are written. Overflow includes escaped-percent additions and returns the source
negative-count fallback on AMD64; no giant host strings or arrays are built.

## Full fork callsite inventory and scratch probes

Read-only lexical inventory: golang/.cache/printf_inventory.py and JSON retain
path/line/format for117 candidates, including4 declarations manually excluded:
21 formatted exception calls,53 warning calls (including the dynamic
warning_displayer wrapper),38 other literal printf-family calls and one dynamic vfprintf
route. Across literal formats the only conversions are %d,%u,%s,%ld,%I64u.
The formatter has no need for float/p/n literal forms in current fork consumers,
but dynamic source formats keep the broader surface explicitly open.

Formatted exception sites:

- src/test/ex.cpp:57: Format %d %s.
- src/muz/rel/dl_instruction.cpp:387,450,499,545,601,841,913: seven relation-kind string diagnostics.
- src/muz/fp/datalog_parser.cpp:670,963,1079,1089,1099,1109,1300,1339,1343,1365,1387,1456,1460: thirteen string/line/name diagnostics;1300 contains mismatched %I64u.
- warning counterpart src/muz/fp/datalog_parser.cpp:1287 also contains mismatched %I64u.
- src/util/string_buffer.h:107 has %ld (with secure-CRT branch at105); direct numeric printf/string-buffer callers are inventory, not newly ported algorithms.
- src/util/warning.h:49 dynamic warning_displayer format and warning.cpp:156 dynamic vfprintf are not assumed restricted by the literal inventory.

Scratch cc experiments are diagnostic source evidence only:

- .cache/printf_probe.c/.cache/printf_probe: libc C-locale I64 uint64 mismatch
  warning;4294967297 prints width64 low32 value1; embedded %c NUL count6 but
  C-output x; trailing % returns-1; null pointer displays(nil) (p remains open).
- .cache/printf_null_probe.c/.cache/printf_null_probe: nullable%s precision0..5
  empty,6/7 (null). Compile TMPDIR was golang/.cache/tmp.

Permanent Go expected fixtures independently preserve Linux I64 width64,
uint32max, explicit low32 diagnostic, typed uint64 mismatch and nullable precision
rules, so runtime tests do not depend on scratch probes/source/C compilers.

## Tests and RED evidence

Initial eight groups were written first and observed missing-API RED (2d2a71).
Additional meaningful RED (015633) showed %2147483647d%% incorrectly returning
positive count/truncated padding after escaped-percent overflow. The sink now
classifies this as negative count overflow and AMD64 raw-format fallback.

src/test/ex.cpp:57 is active through main.cpp:193 TST(ex): its formatted printed
consumer now appears as exact Format 12 twelve fixture and constructor test. This
is **one ported source printed case, zero source ENSURE assertions**. The local
custom ex/ex2 printed case remains separate/unported. Additional fixtures and
bounded oracles are new Go coverage, not invented source assertion counts.

Eleven registered test groups:

- TestPrintfSourceExceptionConsumerAndCallsiteForms
- TestPrintfIntegerFlagsWidthsPrecisionsAndDomains
- TestPrintfCBytesNullPrecisionAndCharacterCount
- TestPrintfSourceBufferCapacitiesTruncationAndGrowLoop
- TestPrintfLinuxI64WidthAndExplicitMismatchBoundary
- TestPrintfMalformedUnsupportedAndSignedCountOverflow
- TestPrintfBudgetFailureEventsAndOwnerGuards
- TestPrintfIndependentFiniteIntegerAndPaddingOracle (1000 signed/hex/octal values and96 byte/width/precision scenarios)
- TestPrintfSignedCountOverflowOnEscapedPercent
- TestPrintfBudgetPeakExactAndObserverPanicCleanup
- TestPrintfByteBoundaryAndBackendEventClassification

Native all11 RUN/PASS35706e; final checked race and actual fresh checked JS
all eleven explicit RUN/PASS registrations passed, along with vet and WASI
build. Parent frozen integration also passed full native, full g3checked+g3debug,
targeted race, vet, WASI build and actual checked-debug JS for all eleven
registrations. Frozen formatter bytes matched after the gates. Tests require
no source checkout/C compiler.

## Final frozen gates

- Full native module `go test ./... -count=1`: exit0 (8004/1fb050).
- Full checked module `go test -tags=g3checked ./... -count=1`: exit0 (63510/a66be3).
- Target checked race `-race -run '^TestPrintf' -count=1 -v`: all11 explicit RUN/PASS, exit0 (25814/835720).
- Fresh checked JS compile then minimal Node actual execution: all11 explicit RUN/PASS, exit0 (686599).
- Root vet `go vet .`: exit0 (44a496).
- Checked WASI cross-compile: exit0 (9891/b5e561).
- Source/no-generics syntax audit: exit0 (87018/60b7bf),2102 units/677 APIs,25 ported/46 partial/2031 unported.

Production/tests frozen hashes:

```
42ede9c1113d315641fcc275cc37c4a5b5b8b2961ba67aa5d354a83d63628e30  golang/util_printf.go
1cae63afb0e3097ce43d1ef235f26f7902926685bbf5e14c5d0205ceccabead9  golang/util_printf_test.go
```

## Audited source SHA256

```
fe2dd5906178c5c4f5469e0dfff5cdf15ac3b993a65ceb4ff3ddd5eb1534c0e5  src/util/warning.h
6ef3f7c9d46d2d496f53f0d20ad5a66bc09b9df755997da2e0d9d53e2edc6db3  src/util/warning.cpp
aafafc0fe6ca0afd7bafe0c781342da259f9c34c0b94b16732c415cf2733066e  src/util/vector.h
3fef3958a95b8020caeb2feddbc5bb7b3074dcb38c205b6af74b67c562b9a74d  src/util/z3_exception.cpp
d610cf65557aae3d4fbbc4963f3916694bb9ddf9661c6cb21083a68518e9d0b3  src/test/ex.cpp
122fcabf395690b281e18d218a9a8fbb739bcbe226926d0342163f0ac1030c14  src/muz/fp/datalog_parser.cpp
943a03b91df62f385b005fd25cc01970fbf42905d16d33a50bf435732c3aa65d  src/muz/rel/dl_instruction.cpp
```
