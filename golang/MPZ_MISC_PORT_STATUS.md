# Integer scalar inspection

Selected source is `src/util/mpz.h` and `mpz.cpp`, internal 32-bit digits.
`mpz_misc.go` implements precise/field, is_uint/get_uint, is_int/get_int,
is_even/is_odd, size_info, get_double, hash, power_of_two_multiple and decompose.
This does not complete either MPZ unit or the rational/solver wrappers.

Five new tests first failed with missing methods in an isolated committed module.
They then passed after translation. Conversion tests preserve strict exclusions
of UINT_MAX and both signed 32-bit boundaries. Finite independent math/big tests
decode all emitted little-endian digits and check sign, parity and trailing zeros.
Inline zero emits one digit. INT_MIN magnitude uses explicit wider arithmetic,
replacing the source's signed-negation overflow with its intended unsigned digit.
SizeInfo remains the source live-limb sorting weight, not bytes or capacity.

Floating conversion retains UINT_MAX as the radix, rather than repairing it to
2^32. Explicit product rounding retains the source multiply/add sequence;
overflow followed by a zero interior digit propagates NaN. Tests cover inline
values, signed big values, low-limb zeros and this overflow behavior.

Hash retains unsigned inline casting, one-limb magnitude hashing and sign-free
multi-limb string_hash with initial value17. Indexed WordReader reuses the audited
source signed-char/little-endian hash without allocating or exposing limb storage.
The active rational.cpp tst_hash assignment-history assertion is adapted to the
underlying integer representation for zero and10000 deterministic signed values;
the rational wrapper itself is not claimed ported. Byte projections cover all
tail/block sizes through17 limbs and both signs. Source unsigned count/length
wrapping remains explicit in SizeInfo, Hash and trailing-zero accumulation.

All scalar reads borrow the manager and allocate no owner storage. Decompose
appends to caller-owned output and does not retain it; reuse of sufficient capacity
allocates nothing. Invalid/closed input uses existing typed guards before touching
caller output. Tests verify zero Go allocations and unchanged budget while every
reservation is forced to fail. Generic types, unsafe and per-record interfaces
are absent.

Full native and checked race module tests, actual checked JS execution of all
five new test names, checked vet and WASI cross-build are the verification gates.
WASI compilation is not runtime evidence. Source/no-generics inventory remains
open for all other MPZ and complete solver work.
