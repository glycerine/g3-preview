# IEEE64 numeric stream prerequisite

The approved concrete C-locale nearest/even IEEE64 numeric profile is implemented
in util_numeric_stream.go/test. It is an explicit platform replacement for
external libstdc++ num_put, not a translated repository/solver numeric algorithm.
Only these two new Go files, this status and bounded .cache scratch are written.
No existing production or central metadata/Git changes. No Go generics/unsafe/
cgo/boxed hot records. Statistics remains audit-only and the real solver remains
open. Other locales/platforms/process-rounding/custom stream runtime stay open.

## Real repository numeric profiles inspected

- util/statistics.cpp:139,169: fixed precision2 only on actual double branch;
  never restores flags/precision. Internal output193-201 inherits caller state.
  Integer counters and lowercase-code insertion inherit integer flags too.
- util/timeit.cpp:41-43, small_object_allocator.cpp:175,240, sat_types.h:291,
  cmd_context/simplify_cmd.cpp:138-140: fixed precision2 persistent on real stream.
- util/hwf.cpp:383: scientific inherited precision (default6 on a fresh stream).
- util/double_manager.h:89-93: fresh ostringstream, setprecision12 defaultfloat.
- util/lp/indexed_vector.hpp:49-52: value inserted BEFORE setprecision3 then
  space. First value inherits prior precision; later values use3. Preserve this
  action order, not a blanket precision3 render.
- util/mpfx.cpp:754, mpff.cpp:873,1168: integer hex, zero fill and width2*wordbytes
  on each limb. These flags persist and may affect later statistics internal
  integer codes. They are separate exact-numeral algorithms, not float64 callers.
- util/mpf.cpp:1623-1632: source sets uppercase/showpoint/showpos/hex, combines
  prior flags via setf, precision13; Windows branch uses actual hexfloat while
  non-Windows reads source representation bits as unsigned and prints hex.
  Do not claim this mpf-specific representation route by formatting a Go float.
- cmd_context/extra_cmds/dbg_cmds.cpp:258 uses left+setw6. Numerous SMT pp routines
  set widths (theory_arith_pp.h, smt_context_pp.cpp, theory_bv.cpp etc.). Source
  stream insertion of preceding strings/chars can consume a width before the
  next numeric value; the stream cannot be modeled by float formatting alone.

No source standalone float-stream test or fixed statistics output fixture exists.
Ordinary source numeric/solver tests retain their original domains; no unrelated
ENSURE counts may be claimed for this external stream dependency.

## External source baseline

Local g++13.3/libstdc++13 C-locale implementation inspected at
/usr/include/c++/13/bits/locale_facets.tcc::_M_insert_float (near993-1120).
It chooses precision6 if stored precision is negative, constructs a printf
format from float/showpoint/showpos/uppercase flags, converts under its C locale,
then applies the stream locale decimal point/grouping, padding, resets width0,
and writes the fully formatted span to the output iterator. Normal profile
precision is used for fixed/scientific/defaultfloat; combined fixed|scientific
is hexfloat and ignores precision. Conversion initially tries45 chars for IEEE64
(digits10*3), then alloca(requiredCount+1) if insufficient. Locales/custom num_put,
streambuf failures, sentry/tie/unitbuf/exceptions are external runtime behavior,
not repository algorithms silently transferable to ordinary Go strings.

Go proposal explicitly selects narrow-byte C-locale and IEEE round-to-nearest,
ties-to-even. It does not pretend to detect/replicate arbitrary caller locale,
process fenv rounding, custom streambuf/facets or Windows CRT tokens. All concrete
IEEE64 values, signs, default/fixed/scientific modes and requested precisions in
that profile must work; no default-only, integer-only or finite-only fake closure.
Full hexfloat/wide/localized/custom ostream surface stays inventoried open.

## Actual local Go primary implementation

Current GOROOT=/mnt/oldrog/usr/local/dev-go/go. Public
src/strconv/number.go:189-196 forwards FormatFloat/AppendFloat to internal/strconv;
ftoa.go is not under the public strconv directory in this compiler.
Read src/internal/strconv/ftoa.go, ftoafixed.go, ftoadbox.go and decimal.go.
AppendFloat calls genericFtoa, inspecting IEEE bits; fixed significant-digit
fast paths use local24-byte arrays; larger precision uses stack-resident decimal
with [800]byte digits and nearest/even decimal rounding. Shortest mode uses
Dragonbox in this dev compiler (Go1.25.3 reference uses Ryu); do NOT select
precision-1 for source defaultfloat. Source default means significant precision6,
not shortest round-trip. Negative stream precision also becomes effective6.

Escape output from go build -gcflags='internal/strconv=-m' confirms
bigFtoa:190 and roundShortest:292,311 new(decimal) do not escape. Append output
can allocate if dst capacity is insufficient: caller MUST preflight/charge full
worst-case destination BEFORE calling AppendFloat, never rely on its automatic
append growth. Internal decimal stack arrays are compiler frames, not uncharged
retained owner storage. No host formatted intermediate strings or fmt.Sprintf.
Toolchain changes require repeating meaningful slow-path allocation checks;
compiler source inspection and bounded probes are evidence, not a universal
promise about every future Go standard-library release.

## Semantic adapters needed

| Domain | Source C-locale baseline | Raw Go AppendFloat difference/required adapter |
| --- | --- | --- |
| +0/-0 | 0/-0; fixed2 0.00/-0.00 | signbit preserved; no zero normalization |
| positive Inf | inf; +inf only with showpos | Go emits +Inf: intercept/classify IEEE bits; explicit sign/case policy |
| negative Inf | -inf | explicit lower/uppercase token |
| signed NaN | nan/-nan, +nan with showpos if positive | Go emits NaN and loses sign; classify original IEEE sign bit, preserve signed token; payload not printed |
| uppercase | E marker, NAN/INF | E/G for ordinary output; explicit token case; fixed finite digits unchanged |
| showpos | positive finite/zero/Inf/NaN prefix+ | insert once from original sign bit; unsigned integer counters ignore showpos |
| showpoint | decimal point even at precision0; defaultfloat retains significant zeros | Go g trims zeros: rebuild point/significant trailing zeros before exponent in charged scratch; f/e precision0 need point insertion |
| defaultfloat | significant max(P,1), fixed if exponent>=-4 and exponent<P after rounding | g/G with effectiveP, NOT shortest; handle showpoint padding independently |
| fixed/scientific | P digits after point; scientific signed exponent at least2 digits | f/e/E; enforce source negative precision=>6, not Go negative shortest mode |
| exponent | e+06/e-05/e-324 | Go ordinary e/E matches minimum2 digit profile; no fabricated three-digit Windows exponent |
| ties | nearest/even exact binary input | Go default integer conversion matches selected nearest profile; process directed fenv mode remains outside profile |
| locale | C '.', no grouping in selected profile | reject/document other profiles; do not silently ignore selected non-C locale |

Do not use math.IsNaN and lose sign before token rendering; inspect Math.Float64bits
for classification/sign. Floating value2.675 rounds fixed2 to2.67 because actual
binary64 is below decimal halfway; no decimal-string double rounding. Sign of
rounded-to-zero negative finite input remains negative. Scalar arithmetic/state
continues separately; renderer is not performing rational mathematical rounding.

## Approved concrete API/storage

NewNumericTextStream(NumericTextStreamConfig{Budget *arena.Budget}) owns cold
header, nested actual StringBuffer64 owner and reusable byte numeric scratch.
Private enum mode Default/Fixed/Scientific; state fields Precision int64 (source
streamsize), Width int64, Fill byte, Alignment Right/Left/Internal, Uppercase,
ShowPoint, ShowPos, integer base Dec/Oct/Hex and ShowBase. Fresh source default:
defaultfloat, precision6, width0, fillspace, decimal, right adjustment.
Concrete manipulators change state without emitting bytes; scalar Stats snapshots
include exact persistent state and owned byte charges. Width<=0 emits no padding
but still resets after the next valid formatted insertion. Preserve setters'
source order; no boxed per-argument record or function callbacks.

Methods proposed: SetFloatMode, SetPrecision, SetWidth, SetFill, SetAlignment,
SetUppercase, SetShowPoint, SetShowPos, SetIntegerBase, SetShowBase;
AppendFloat64, AppendUint32, AppendInt32, AppendUint64, AppendInt64,
AppendCString, AppendByte; String, ResetText, Close. Destination is private owned
buffer, not an externally lockable SymbolManager. String returns one final caller
copy. ResetText only clears text and retains state/scratch/capacity, matching a
stream's str(empty) behavior; it must not reset flags implicitly. No generic
ostream object or fake CLI stream is invented. A future owned-byte adapter can
accept this concrete stream in Statistics display methods.

Formatted strings/chars consume Width, including the space inserted immediately
before stats double manipulators. State Fixed/Precision2 changes persist before
float conversion. For each numeric insertion, stage the entire formatted field
successfully, apply width/align logically, reset Width0 BEFORE writing its bytes
as num_put does. If byte-output growth later fails, already emitted prefix and
Width0 remain. Scratch preparation failure is a predictable Go allocation adapter:
retain prior width/output, while preceding explicit flag changes remain. Source
fatal alloca/OOM has no recoverable same-prefix contract; document the Go one.
Integer formatting must honor widths/base/prefix and actual signed type widths;
negative int32 hex is its32-bit unsigned pattern, uint32 showpos ignored. This
supports statistics codes/counters without assuming every stream is decimal.

The numeric owner/header, nested buffer/header/inline, and actual retained scratch
must all be charged once, with borrowed Budget never duplicated. Scratch growth
charges full new physical capacity while old remains, then releases old after
copy/staging. Output uses committed buffer growth; Close destroys nested storage
and scratch/header exactly once. Returned final text is caller-owned, no other
intermediate host string. State inspection is scalar, no escaping backing slice.

## Worst-case preflight before AppendFloat

Let effectiveP=6 for negative stored precision; defaultfloat P=max(effectiveP,1).
Check signed-to-host-int conversion, uint64 additions and physical allocation
size before reservation. Derived IEEE64 finite bounds including sign and enabled
showpoint (not C terminator; AppendFloat does not append one):

- Fixed: at most309 integer digits. For P>0: P+311 bytes (sign+point+fraction).
  At P0:310 bytes, or311 with showpoint. This covers negative MaxFloat64.
- Scientific: P+8 bytes (sign+leadingdigit+point+fraction+e+exponentsign+3digits).
- Defaultfloat: P+7 bytes; exponent -324..308, scientific form uses P significant
  digits; non-scientific small values need at most four leading zero positions.
  Showpoint can pad significant digits to requestedP; include this in the bound.
- Specials:4 bytes with sign; no showpoint decimal added.

Reserve/ensure actual scratch capacity>=bound BEFORE AppendFloat(rawdst[:0],f,
mode,effectiveP,64). Sign/showpoint normalization must fit that same bound or a
separately checked larger staging reservation. Never call AppendFloat first and
charge a returned enlarged slice afterwards. Width may exceed numeric field
length; padding length is checked max(width,fieldLength), then emitted as counts
rather than requiring another uncharged padded string. Output budget failure can
stop a huge width loop after a defined prefix; no arbitrary width cap. The signed
C precision ABI boundary below is separately classified, not a convenience cap.
Precisely document reusable scratch geometric/exact capacity policy as Go storage
adaptation; warm conversions and slow high-precision paths need zero allocation
checks and unchanged backing identity after conversion, not just budget counters.

## Bounded C++/Go reference probes and permanent expectations

All scratch under .cache/float-stream: probe.cpp/probe/probe.out and
 go_probe.go/go_probe.out/escape.out. g++13.3 -std=c++11 -O0, compiler TMPDIR below
Go. 17 scalar values tested in default6/fixed2/scientific2; extra flags/width/
locale/fenv cases. Probe results are local platform observations, not evidence
of an absent repository assertion. Normal future Go tests must use permanent
literal expectations and exact independent oracles, never compile/read C++ source.

Permanent finite fixtures to promote:

- zero: 0,0.00,0.00e+00; negative zero: -0,-0.00,-0.00e+00.
- default6 0.0001=>0.0001;0.00001=>1e-05;999999=>999999;
  1000000=>1e+06; minimum subnormal=>4.94066e-324.
- fixed2 binary2.675=>2.67; scientific2=>2.67e+00.
- nearest/even fixed0 2.5=>2,3.5=>4; showpoint fixed0 2.5=>2.
- showpoint default6 1.25=>1.25000; default0 1=>1.
- sign-bit NaN=>-nan; uppercase/showpos sign-bit NaN=>-NAN; positive Inf=>inf,
  uppercase=>INF; showpos positive=>+inf. Raw Go instead NaN/+Inf.
- internal fill'_' width9/showpos/fixed2 1.25=>+____1.25; Width0 afterward,
  Precision2 and Fixed remain. setw8 inserted space then1.25=>_______ 1.25.
- custom comma/grouping locale12345.5=>12_345,50 (outside selected C profile).
- minimum subnormal fixed2=>0.00/scientific2=>4.94e-324; negative counterpart
  preserves sign. MaxFloat64 fixed2 includes its full309-digit integer plus.00;
  permanently store that exact observed literal and independent bit-based oracle.
- directed fenv fixed0 2.5/3.5: downward2/3, upward3/4, towardzero2/3 versus
  nearest2/4. Source runtime boundary explicitly open; do not claim equivalence.
- combined float flags hexfloat1.25=>0x1.4p+0 ignores precision2, open profile.

Go preallocated12000 bytes probes modes f/e/g with precisions0,2,6,12,18,19,100,
1000,10000 on negativezero/minsubnormal/MaxFloat64/2.675/signedNaN/+Inf. All
reported zero allocations across testing.AllocsPerRun; signedNaN/Inf spelling
mismatch observed. These probes do not prove unrestricted precision; the checked
bound and compiler escape evidence are the implementation preflight rationale.
No production math/big needed; future independent test-only exact oracle allowed.

## TDD and remaining runtime gate

Tests FIRST after profile/API approval: permanent fixtures above, all finite IEEE
bit partitions/random bounded models, nearest tie/carry and exponent switches,
signed NaN/Inf/zero, showpoint all modes, negative/zero/large precision semantics,
all alignments/fills/base/sign/width and prefix order. Repeated same-stream calls
must show state persistence and one-shot width consumption, including preceding
CString/byte and stats shadowed double NO fixed-state mutation. Charge header/
inline/scratch/output peaks, failure stage-vs-emission prefixes and cleanup,
fullprecision slow-path zero allocations/no backing relocation under preflight,
closed/shallow owner guards and final text lifetime. Native/checked-debug/race/
actual named JS/vet/WASI/source-noGen gates; no runtime C++ compiler/source needs.

This selected C-locale nearest IEEE64 profile is an explicit external standard
library/platform adaptation. Full printf float conversions, Windows CRT,
hexfloat/wide/customlocale/process-rounding/customstream state remain separate
units, not falsely marked complete. Real source statistics collectors/borrowed
keys and the actual solver remain future dependencies.

## Primary source hashes

```
dda43e8c91d95f9dea0c46b2fec6637e5520cae099e40c5965dca0e41bbaab9e  GOROOT/src/strconv/number.go
2c12d1c3a17b9a978266a7c761862a4be8dd1f2fae14ad5bbdf6ff2dd8f3493d  GOROOT/src/internal/strconv/ftoa.go
6b8a2f52d5e53206930ba481165268f58d08b5477a4810b91eca55345fecc093  GOROOT/src/internal/strconv/ftoafixed.go
131ede3bec4e39c5737c83fc074401bf22fa4e03b2038c972a039310f1f7dfb3  GOROOT/src/internal/strconv/ftoadbox.go
23bd45989d0b5a7b441c45873bf43f210fdf8381132fc74424d381216ac3a8b4  GOROOT/src/internal/strconv/decimal.go
9ba29fa5b08f571ff4bfbee8acd35e3f71cf4647b2f6fd02712e5d11119f30fb  /usr/include/c++/13/bits/locale_facets.tcc
```


## Additional observed precision ABI boundary

Local bits/postypes.h:68 defines streamsize=ptrdiff_t (64bits on AMD64).
locale_facets.tcc passes that streamsize __prec into variadic __convert_from_v;
local x86_64-linux-gnu/c++/13/bits/c++locale.h:74-108 forwards va_list to
vsnprintf, whose dynamic precision consumes C int. This external implementation
thus exposes an ABI narrowing/type-mismatch boundary for wide stream precision.
A narrowly bounded additional probe stored precision2^32+2 but printed fixed1.25
as1.25, not billions of fractional digits: observed low32 behavior while stored
precision remains4294967298. Do not silently claim Go AppendFloat's wide integer
precision equals this source runtime. All actual repository precisions found are
small2/3/6/12/13 and fit the C precision domain.

The coordinator explicitly approved the observed dynamic-int platform adapter:
SetPrecision retains signed64 state. If stored state is negative, effective6;
otherwise narrow to signed32. A negative narrowed precision models omitted C
printf precision: default6 for f/e/g. Narrowed zero means0 fractional digits for
f/e and1 significant digit for g. This preserves finite2^32+2=>2 and negative
narrowing without a positive INTMAX rejection. It is a selected external ABI
profile, not portable C varargs equivalence across runtimes. Source backend signed output-count
overflow and negative conversion results are also open runtime domains; do not
fabricate a formatter fallback as if num_put guaranteed one. Future tests must
classify these boundaries separately from ordinary Budget or host-int errors.


## Implementation and observed RED/GREEN

Eight groups were written FIRST (initial five missing-API RED350427), with exact
independent fixed/scientific bit-rational oracles added before final freeze.
Meaningful regression RED5ecfef exposed the total-host-length cap on virtual
WidthMaxInt64 with existing position1: corrected to preserve uint64 padding count
and reset Width0 before byte emission, with bounds only on ACTUAL buffer growth.
Finite owner-budget requests emit through position64 then ErrBudget on every
architecture, including actual JS32. No arbitrary virtual width cap remains.
Meaningful REDb316d1 exposed String using CStr and truncating embedded NUL:
corrected to full-position stdstring byte copy, with no C terminator mutation or
capacity growth. Source char NUL survives output; source CString input stops NUL.

Float staging checks actual host slice capacity before AppendFloat. The full
scratch bound is reserved while old scratch remains charged; ensureScratch exact
new physical capacity is a Go storage policy. Scratch success remains retained
if later byte emission fails. Width reset happens after field staging and before
output, and emitted fill/sign/value prefixes remain on allocation failure. The
private output buffer takes no exposed buffer lock; stream mutex protects all
operations. Stream state is a pointer-free scalar snapshot. Owner self identity
rejects shallow copies; nil/closed guards are typed. State and output are distinct:
ResetText retains all flags/scratch/output capacity. Close releases nested buffer,
scratch and owner exactly once. One final full-byte caller string is independent
of owner Close. External Budget still requires confinement.

Named tests (no fabricated source ENSURE registrations):

- TestNumericStreamFloatProfilesAndSpecialBits
- TestNumericStreamShowpointSignUpperAndPrecisionABI
- TestNumericStreamPersistentWidthAlignmentAndIntegerDomains
- TestNumericStreamFullByteOutputStringAndWidthConsumption
- TestNumericStreamBudgetFailureStagesPrefixesAndPeak
- TestNumericStreamSlowPrecisionWarmAllocationsAndGuards
- TestNumericStreamIndependentExactBinaryRounding
- TestNumericStreamExactOwnerScratchOverlapAndHugeWidthPrefix

Independent test-only big.Rat exact binary decomposition/division/remainder
nearest-even oracles cover fixed AND scientific rendering across180 deterministic
finite IEEE64 bit patterns and endpoint/tie-neighbor inputs. Production imports no
math/big. Slow precision1000 MaxFloat64 warmed render reports0 Go allocations.
Exact owner charge is reflect(stream)+reflect(nested64 buffer), without duplicate
inline bytes. Scratch10->16 verifies physical overlap peak26 (25 fails unchanged,
26 succeeds). Fault tests inspect immediate actual charges and Width/output before
Close. Huge padding finite-budget prefixes are tested with nonzero prior content;
JS execution confirms no simulated32-only shortcut. All final Go files frozen.

Frozen tree .cache/release/numeric-stream-frozen is explicit git archive
HEAD7ad8dbd plus ONLY these two Go files. Terminal final gate appendix follows.

## Final frozen gate evidence

All terminal exit0, tree HEAD7ad8dbd+only2streamGo:

- Full native `go test ./... -count=1`: 255e89.
- Full checked/debug `go test -tags='g3checked g3debug' ./... -count=1`: a04443.
- Targeted checked/debug race: 62aadf, all8 named RUN/PASS.
- Checked/debug JS compilation194030 followed by actual minimal Node wrapper
  `-test.run '^TestNumericStream' -test.v -test.count=1`: 6aae3c,
  all8 named RUN/PASS, including virtualWidthMaxInt64 prefix on actual32 target.
- Full vet: 47a827.
- Checked/debug WASI compilation: 11917e.
- Source/no-generics audit with explicit reference root: bdd2cd,
  2102units/677APIs/33ported/60partial/2009unported before stream metadata integration.

Frozen Go hashes:

```
27eacf53f859a071be257999ad5029394352845eb5fb04218f82c37d1de2192e  util_numeric_stream.go
c7fcf550756dd4887f99aeb18d764d38081a0830d065ee632ec192ecc2dd895a  util_numeric_stream_test.go
```

Statistics audit was separately corrected at coordinator request: its internal
helper lacks key++ and repeats the FIRST byte/code forever for nonempty key.
No finite DisplayInternal claim or silent fix is implemented. Statistics methods
remain unported pending their next real borrowed-key/renderer/collector packet;
this stream is their external runtime prerequisite only.

## Independent parent integration

The parent `.cache/release/numeric-stream-integrated` is committed7ad8dbd plus
only the two frozen stream Go files. Both hashes above match the authoritative
worktree. Full native4499 PASS (main1.973s/mpnum10.593s); full checked/debug79846
PASS (main1.933s/mpnum10.911s). Checked/debug actual JS21642 PASS0.149s and
race77968 PASS1.034s each contain exactly all8 expected named RUN/PASS groups
and the PASS footer. Full checked/debug vet PASS; WASI compilation30020 PASS;
source/no-generics12525 PASS (2102units/677APIs/33ported/60partial/2009unported).
No source methods or API entries were relabeled for an external runtime
prerequisite. Actual Statistics rendering and collectors remain unimplemented.
