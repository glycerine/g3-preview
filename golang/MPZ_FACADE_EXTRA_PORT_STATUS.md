# MPZ public manager contract closure

Source: `src/util/mpz.h` public methods under the selected internal 32-bit
backend, and active `src/test/mpz.cpp::tst1` /
`src/test/mpq.cpp::tst1`. New files are `mpnum/mpz_facade_extra.go/test`.
No existing arithmetic or storage algorithm, owner header, or manager namespace
was changed.

Twenty-one concrete manager methods complete the previously unnamed public
routes: `Sign`, `IsPos`, `IsNeg`, `IsZero`, `IsNonPos`, `IsNonNeg`, `IsOne`,
`IsMinusOne`; `Inc`, `Dec`; `Equal`, `NotEqual`, `Less`, `Greater`, `LessEqual`,
`GreaterEqual`, `LessInt`; `SetInt32`, `SetUint32`; `Dup`; `Inv`.

Sign retains the entire inline int32 value, not a normalized mathematical sign.
Big values retain their existing -1/+1 sign representation. One/minus-one checks
require inline representation. Comparisons use the already translated Compare
algorithm; transposed/negated wrappers retain source operand order. LessInt
retains its inline fast path and otherwise creates a scalar inline argument.
Inc and Dec both call Add with source inline +1/-1, rather than substituting a
Sub route. SetInt32 explicitly Del's the old payload before inline publication.
SetUint32 includes INT_MAX in its inline range, unlike SetUint64's strict
threshold, and otherwise uses the signed int64 source constructor.

Dup copies into an initialized zero Int through the existing source Copy route,
preserving requested cell capacity and representation. A returned big Int owns
its independent cell; the caller must Del it and may not shallow-copy it.
Predictable failure destroys any unpublished output. No new persistent or
unbudgeted scratch storage is introduced, and small Dup allocates no storage.

Integer Inv has no supported numerical domain: source contains only
`SASSERT(false)`. In a release C++ build its body otherwise does nothing. The
Go method always returns `ErrIntegerInvUnsupported` after existing input/lifetime
guards. This explicit checked-domain adapter neither claims release no-op
observational equivalence nor supplies a field inverse. Input and budget remain
unchanged. Low-level manager-local raw Int provenance remains the established
caller ownership contract; these methods do not invent owner identity in Int.

The two active source tst1 bodies each contribute nine exact comparison ENSURE
executions, eighteen total, through the new manager methods. The MPQ test uses
existing `RationalManager.Integers()` composition, which already supplies every
inherited MPZ algorithm and now the new public methods. It is not a newly
implemented inheritance surface or a safe RationalService manager escape.
Each fixture retains actual string input, Mul/Add/Neg setup, source assertion
order, explicit numeral Del, and manager lexical scope. Source printed numeric
values have separately labeled added permanent expectations.

Six initial suites observed RED for missing concrete methods, then GREEN.
A seventh representation-equality/warm-allocation regression passed immediately.
Independent bounded math/big tests cover signed small values and large exact
increments/decrements, scalar comparisons, duplication independence, source
requested capacity, and uint32-vs-uint64 representation equality. Three
reservation-prefix sweeps (Inc, Dec, Dup) force actual growth with cold 40-limb
operands; immediate live-cell ownership is checked before rendering/Del/Close,
failed destinations remain unchanged, and final budget is zero. Nil/closed
inputs and unsupported Inv are diagnosed. Warm predicates/comparison/inline
Inc/Dec/Dup allocate zero Go objects and consume no owner budget even with the
next reservation forced to fail.

This closes selected public numerical method names, not every source conditional
or diagnostic. GMP/MSBIGNUM external representations are explicit platform
replacements, not separately translated algorithms. The selected non-GMP source
chooses Lehmer GCD; disabled `#if 0` sorted-array GCD and alternate BINARY_GCD,
EUCLID_GCD, LS_BINARY_GCD implementations remain inventoried alternatives.
Existing owner confinement replaces OpenMP shared-manager critical sections;
full TRACE/STRACE text routing and source invariant diagnostics beyond previously
ported ExtendedGCD remain open. Go tuple-return output adapters and caller-owned
render/decomposition buffers retain their previously documented contracts.
Historical partial inventories listing parser, power, roots, bitwise, logarithms,
rounding or GCD as absent are superseded by actual current implementations.

AST polynomial prerequisites remain open beyond MPZ/MPQ: actual polynomial.h
includes mpbqi.h, algebraic_numbers.cpp retains mpbq managers and interval cells,
and polynomial.cpp uses mpzzp_manager. Those binary-rational/interval/modular
manager algorithms are genuinely separate missing units, not replaced by
RationalService or an interface scaffold.

Verification passed on frozen prospective Go bytes in
`.cache/release/mpz-facade-extra-final`, HEAD plus only the two new Go files:

- Full native module, session4647 (mpnum10.060s).
- Full actual g3checked+g3debug module, session52445 (mpnum10.115s).
- All seven targeted checked-debug race tests, session40481.
- All seven actual checked-debug JS named RUN/PASS registrations, session7339.
- Full checked-debug vet94150, checked-debug WASI compile27663.
- No-generics/source syntax audit23819 (29 ported/51 partial/2022 unported;
  central numerical metadata still belongs to the coordinator).

WASI compilation is not runtime evidence. No git or central metadata edits.
All explicit writes/build/cache/temp artifacts remained under golang.

Coordinator integration independently matched all four string-buffer/facade Go
SHA256 values against `.cache/release/string-buffer-mpz-facade-integrated`.
Fresh full native and checked-debug modules, vet and WASI build passed there;
all17 targeted race and actual JS RUN/PASS registrations were checked in
`.cache/string-buffer-mpz-facade-integrated-{race,js}.log`. A subsequent combined
ID/chain integration also passed full native and checked-debug modules against
these same frozen facade bytes.
