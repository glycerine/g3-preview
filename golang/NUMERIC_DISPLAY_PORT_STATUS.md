# Numeric display packet

## Source coverage

The complete selected `_MP_INTERNAL` printing branches were read and translated:
`mpz_manager::display_smt2(mpz,bool)` maps to IntegerManager
`FormatSMT2Into` / `SMT2String`; `mpq_manager::display_smt2(mpq,bool)`
maps to the rational counterparts; `mpq_manager::display_decimal` maps to
`FormatDecimalInto` / `DecimalString`. The thin `to_rational_string(mpq)`
and `display_pp(mpq)` wrappers map to `ToRationalString` and
`FormatPrettyInto`. Existing Integer/Rational `String` / `FormatInto` already
cover ordinary `to_string` / `display`; the inherited integer overloads are
available through the rational owner's existing `Integers()` service. There
is no public global-manager rational wrapper, ostream interface, alternate
backend, locale/format flag support, or TRACE diagnostic port in this packet.

Integer SMT2 retains the source negative-only Copy/Neg scoped temporary and
`(- magnitude)` form, including `.0` inside the expression when requested.
Fractional SMT2 retains `(/ numerator denominator)`, with each operand printed
using the integer branch. Integral rationals retain the integer representation.
Decimal printing retains the exact source Copy/Copy, sign/Neg,
Div/display/Rem, and repeated Mul-by-ten/Div/display/Rem order, with cleanup
in the source ten/numerator/denominator/digit order. The source development
assertion tests only that each emitted quotient is below ten; it does not
require a nonnegative digit. Initialized source RatDiv/Root outputs can have
negative denominators or remain unreduced. These are accepted and printed
without normalization: raw 1/-3 produces `0.-3-3-3?` at precision three,
raw -1/-3 produces `-0.-3-3-3?`, and -1/-1 produces `--1`. SMT2 exposes
the signed denominator. Unreduced 2/4 remains `(/ 2 4)` in SMT2 and prints
`0.5` as a decimal. These source-action quirks are explicitly frozen separately
from the independent mathematical decimal tests for positive denominators. No rounding, zero padding, or blanket
precision limit is added. A noninteger with precision zero prints the decimal
point and optionally `?`; exact values terminate immediately, including
terminating values at UINT_MAX precision. Negative subunit values print `-0`.

## Ownership and publication

Only the three files in this packet are changed; no existing formatter or
owner-header edits are required. Static punctuation is copied by scalar index
into the existing budgeted byte workspace. Scoped numeric temporaries use the
integer owner's concrete cells and Words, with explicit cleanup on every
failure. All formatting limbs and bytes use the shared existing owner Budget.
Workspace growth preserves previously prepared prefixes, including a negative
fraction numerator while the denominator is formatted. Workspace reuse and
retention follow the prior formatting storage adaptation; this does not claim
the C++ stream allocation profile. Complete output is staged before publication,
so short buffers and predictable allocation errors leave caller bytes untouched.
This intentionally adapts source partial-stream-output failure behavior to the
Go caller-buffer contract. Required output length excludes a C NUL; no extra
byte is written. A String wrapper allocates one final caller-owned Go string;
its escaped host lifetime is outside the numeric owner Budget, and no hidden
intermediate string is used. Borrowed numeral inputs remain unchanged.

## Test evidence

Tests were written first and observed RED because the printing APIs did not
exist. Nine test functions are now GREEN. Source mpz/mpq/rational test files
were read: the relevant active decimal output in `src/test/mpq.cpp::tst2`
prints 1/3 and 1/4 at precision ten. Their exact expectations
`0.3333333333?` and `0.25` are independently derived, not C++ observations or
source assertions. No active SMT2 printing assertions occur in those tests.
Added finite tests cover both manager modes, small and big signed SMT2 operands,
all numerators -19..19 and denominators 1..19 at precision 0/1/2/3/10 with both
truncate choices, a test-only big.Rat oracle, integer/terminating/nonterminating
branches, raw-zero and closed-owner diagnostics, exact caller length/no NUL,
short buffers, and escaped output lifetime.

Fault sweeps reach both an actual failure and eventual successful completion
for every operation, both manager modes, and cold/warm formatter states. Each
prefix checks input identity and unmodified caller output, numeric-cell Live
immediately before any subsequent formatting or teardown, and Words Live
accounting for legitimate newly retained formatting refs; Close returns the
shared Budget to zero. Additional tests cover old-plus-new byte reservation
refusal, bounded-budget refusal of a UINT_MAX-precision repeating decimal,
workspace replacement/prefix preservation, zero-allocation warm caller-buffer
printing, and exactly one allocation for final host strings.

Validation: isolated HEAD:golang plus only these three files passed full native
and actual g3checked module tests. All nine new tests were registered and passed under actual g3checked JS/wasm
execution (the initial eight together, then the added noncanonical fixture).
Checked targeted race/vet and checked WASI cross-build passed. The final
UINT_MAX precision-bound fixture and added noncanonical source-action fixture
also passed their affected checked race and actual checked JS reruns. The final
isolated full-module reruns and their session results are recorded in the handoff.
