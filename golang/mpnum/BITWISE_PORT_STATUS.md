# Source integer bitwise operations

Source: all four bitwise methods in util/mpz.cpp (the selected _MP_INTERNAL
32-bit numerical engine) and their util/mpz.h declarations. The active
source test is src/test/mpz.cpp::bug4; its alias equality assertion and exact
printed result are translated. Other MPZ methods and GMP compatibility are not
claimed by this chunk. No generics, production math/big, unbudgeted numeral
arrays, unsafe or interfaces were introduced.

## Algorithm and ownership map

BitwiseOr/BitwiseAnd/BitwiseXor preserve the source small inline arithmetic
paths and the large-number repeated64-bit chunk algorithm. Each general
iteration uses Mod by the manager's owned two64 constant, Uint64 conversion,
the specified machine bit operation, SetUint64, Mul by the multiplier, Add,
then multiplier Mul and both operand Div calls in source order. The loop
ends when either copied operand is zero; Or/Xor append either nonzero tail
through source Mul/Add, while And discards it. Even final unused multiplier
and division actions are retained. There is no limbwise replacement algorithm.

BitwiseNot preserves the small width<=63 mask branch and general width chunks,
including copying a large input at width0 and performing final unused multiplier
and division calls. Explicit unsigned mask arithmetic defines the intended
source width63 signed-shift result. Width is uint32; no new arbitrary size cap
is imposed. Source nonnegative assertions become ErrBitwiseDomain before any
mutation, and existing manager closed/handle errors propagate.

The source's local numerals are concrete stack Int headers with cell/payload
storage from IntegerManager's existing budgeted owners. Every prepared numeral
is explicitly Del'ed in source cleanup order. A separate staged result survives
until final Move; allocation errors leave the destination and aliased operands
unchanged. This adapts source clear-output-first partial mutation and local
cleanup on allocation exceptions, without changing any successful arithmetic,
branch order or alias result. The old destination stays charged while new work
exists, so the documented failure-atomicity peak is real. Earlier manager scratch
and pool-page growth may remain retained and charged after failure. Close refunds
the complete graph; no output/string conversion happens during production bitwise
operations. The owning manager remains confined/serialized as in the existing
numerical contract; the source lock variant is not a new implementation here.

## TDD and validation

The initial four tests failed compilation on missing methods (RED), before
translation. They then passed (GREEN). A fifth independent aliased-failure test
was immediately GREEN against the implemented staged contract. Every permanent
test requires no source checkout or C++ executable.

- Exact active bug4 inputs, equality assertion and independently expected
  diagnostic4294967295/4294967295 are preserved.
- Independent math/big oracle checks zeros, inline boundaries, UINT32/UINT64
  boundaries, long unequal tails and30 seeded192-bit operands. Binary operations
  cover independent outputs, either input alias and all-input identity. Not
  covers widths0/1/7/30/31/32/33/62/63/64/65/127/128/129/192/257, both output modes.
- Each operation reaches both injected failure and successful sweep endpoints,
  including both operand aliases; failed publication preserves all input/output
  values and Close refunds every owner. Negative-domain and closed-owner errors
  preserve output. Warmed all-four-operation runs allocate zero Go objects and
  succeed under FailAfter0.

Prospective tree .cache/release/bitwise-final was archived from committed
8a8ae0a plus only the two new Go files. Full native ./..., full g3checked race
./..., checked vet and checked WASI compilation passed. Full actual checked
JS/WASM ./... passed; all six new test names were explicitly checked for both
RUN and PASS in .cache/bitwise-checked-js-final.log. Source/no-generics audit passes.
WASI is compilation evidence only; the existing runtime-startup gap remains.

Independent review added immediately-GREEN assertions for live temporary-cell
counts BEFORE Close on every failure, so manager teardown cannot mask lost
numerals. A sixth test verifies each small-input fast path releases a preexisting
big destination under FailAfter0. Final full native and all6 targeted checked race
and actual checked JS tests pass. No production change was needed by review;
earlier full-module race/JS gates remain applicable.
