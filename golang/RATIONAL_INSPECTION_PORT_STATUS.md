# Rational inspection and extraction

## Selected source coverage

Complete selected branches were read in `src/util/mpq.h`,
`src/util/mpq.cpp::get_double`, and the explicit 32-bit wrapper projections in
`src/util/rational.h`. This packet changes only `mpnum/mpq_inspect.go`,
`mpnum/mpq_inspect_test.go`, and this note; it introduces no owner fields,
new scratch, global numeric manager, or ergonomic rational wrapper.

| Source methods | Concrete Go owner APIs |
| --- | --- |
| precise, field | Precise, Field |
| sign, is_pos/is_neg/is_zero/is_nonpos/is_nonneg | Sign, IsPos, IsNeg, IsZero, IsNonPos, IsNonNeg |
| is_one, is_minus_one, is_int | IsOne, IsMinusOne, IsInt |
| hash(mpq) | Hash |
| is_uint64/is_int64, get_uint64/get_int64 | IsUint64, IsInt64, Uint64, Int64 |
| rational.h is_unsigned/get_unsigned | IsUnsigned, Unsigned |
| rational.h is_int32/get_int32 | IsInt32, Int32 |
| get_numerator/get_denominator with mpz outputs | Numerator, Denominator |
| get_numerator/get_denominator with mpq outputs | NumeratorRational, DenominatorRational |
| bitsize, storage_size, get_double | Bitsize, StorageSize, Float64 |
| is_even, is_power_of_two(mpq,shift) | IsEven, IsPowerOfTwo |
| set(mpq,uint64), set(mpq,mpz), set(mpq,limbs) | SetUint64, SetInteger, SetLimbs |
| swap_numerator, inc, dec | SwapNumerator, Inc, Dec |

Source IsSmall/default construction already exists. Inherited integer overloads
remain in the explicit `Integers()` service. The selected wrapper projections
do not port rational.h/.cpp as units: their global initialize/finalize,
constants/power cache, automatic constructors/destructor, operator facade,
infinitesimal integration and AST parameter ownership still remain separate.
Number-theory methods and rational bitwise wrappers are outside this packet.

## Exact numerical and alias contracts

`Sign` returns source numerator m_val: the entire inline int32 value for small
numerators, and -1/+1 for big numerators. Source predicates inspect numerator
sign, even when a raw signed denominator would change the mathematical sign.
`IsInt` requires a small denominator exactly one; unreduced 2/2 is not an integer
under this representation test. Source +/-1 predicates require the corresponding
small numeral representations. Hash ignores the denominator and retains the
integer source's sign-free big-magnitude hash and unsigned inline casting;
collisions such as 1/2 and 1/3 are intentional and structural equality resolves
them. StorageSize is the sum of source live-limb sorting weights, not owner
bytes or reserved capacities. Bitsize keeps the integer-only branch and otherwise
sums numerator/denominator sizes with explicit uint32 addition.

The rational accessors enforce their actual source integer-fit assertions;
negative rational Uint64 gets ErrConversion even though the raw MPZ accessor
casts an inline negative or returns a big magnitude. IsUnsigned/Unsigned include
UINT_MAX (<2^32), whereas IntegerManager IsUint32/Uint32 strictly exclude it.
Rational IsInt32/Int32 include INT_MIN and INT_MAX, whereas the MPZ variants
exclude both. Naming and comments make these different source contracts explicit.

Float64 performs the source numerator conversion, denominator conversion and
n/d in order; it preserves the inherited UINT_MAX floating radix, Inf*zero NaN,
valid numerator Inf/NaN and Inf/Inf. ErrRationalFloatDomain replaces the source
SASSERT(d>0) for a negative or NaN converted denominator. This is distinct from
invalid initialization, conversion-fit errors and budget refusal; it does not
reinterpret or normalize raw negative denominators.

Extraction explicitly copies the selected integer payload, without normalizing
or borrowing the source lifetime. Rational-output extraction follows Copy then
reset_denominator. Thin setters likewise publish their integer result and then
reset denominator; the existing failure-atomic integer preparation is sufficient,
because a checked denominator reset cannot allocate. Source component and self
aliases retain this publication order. SwapNumerator exchanges actual ownership
with no allocation or normalization. Inc/Dec delegate the exact source AddInt
calls using inline +/-1. Raw 2/-3 increments to -1/-3 and decrements back to
2/-3. An internal Numerator(a,&a.den) alias for 0/7 completes the source Copy and
creates denominator zero; it is not silently repaired or retroactively rejected.
Subsequent numeral operations diagnose invalid initialization. Such component
references are private to the numeric package, rather than borrowed public fields.

## Storage and test evidence

All payloads and arithmetic scratch use existing concrete cells/Words and their
shared Budget. No production math/big, Go generics, unsafe, per-record interfaces,
Go heap numeral wrapper, unbudgeted intermediate numeric buffer or owner-header
change is introduced. Caller limb input is borrowed only during SetLimbs.
Getter copies acquire independent ownership; swaps transfer ownership. The
existing implicit same-owner caller contract remains in force; this packet does
not claim foreign-handle detection or the concurrently implemented CopyFrom API.

Eleven test functions first observed RED with missing actual manager APIs, then
passed. Source rational.cpp::tst2's active conversion block is preserved through
the manager: seven signed-fit checks, eight unsigned-fit checks, one VERIFY of
UINT64_MAX extraction and two signed boundary extraction assertions. The active
rational_tester::tst_hash history check runs zero plus 10001 deterministic signed
inline cases, preserving its finite case count while replacing C rand state.
Neither adaptation claims that the global rational wrapper has been ported.

Added finite expectations cover both source scratch policies, structural versus
mathematical integer classification, inclusive wrapper boundaries, raw signed
and unreduced values, all getter self/component aliases, copied lifetimes,
limb-input borrowing, ownership swaps, independent exact increments/decrements,
small statistics, big statistics/floating-radix quirks, NaN/Inf domain branches,
invalid/stale/closed values and zero false results on guarded boolean errors.

Nine mutator sweeps advance through every failing reservation prefix until
success; each actually allocating case demonstrates both failure and completion.
A fixture fills spare capacity-eight pool slots to force real UINT64 storage
growth. Additional component-copy alias sweeps distinguish actual growth from
storage-reusing operations. Immediately after each failure, before formatting,
Del or Close, the existing independently enumerated caller/base/source-cache
root audit checks numeric cells and all their Words plus the four division and
format scratch roots. Caller scalar identity and independently decoded live limb
contents then establish untouched input/output values. No enumeration of all
live slots can justify a lost temporary. Close refunds the complete shared
Budget. Read APIs and warmed copies/setters/Inc/Dec allocate zero Go objects;
warm operations also succeed with every reservation forced to fail.

The isolated prospective tree is HEAD:golang plus only these three files under
`.cache/release/rational-inspection-final`. Full native and actual g3checked
module tests passed. Checked targeted race, checked vet and the no-generics
portcheck audit passed; all eleven tests were registered and passed through the
actual checked JS wrapper. Checked WASI executable build passed, with no claim
of WASI runtime execution. Strengthened immediate limb-value failure assertions
also passed affected actual checked JS and checked race reruns. No global ledger
or git mutation was made by this agent.
