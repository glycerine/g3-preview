# MPQ mixed arithmetic, fused operations, and integer-valued wrappers

Source targets: `src/util/mpq.h` mixed arithmetic / fused overloads and
integer-valued rational wrappers, including the private mixed `rat_add` and
`rat_mul` routines. This packet completes these assigned overloads; it does not
claim the entire mpq.h / rational.h family is complete. Existing mpq.go rational
Add/Sub/Mul/Div and manager headers were not edited.

## Exact method map

| Source overload | Concrete Go counterpart |
| --- | --- |
| add(mpq,mpz,mpq), rat_add(mpq,mpz,mpq) | AddInt; integer fast path or one cross-product temporary |
| mul(mpz,mpq,mpq), rat_mul(mpz,mpq,mpq) | MulInt; integer fast path or copied denominator plus normalize |
| mul(mpz,mpz,mpq) | MulInts; integer product with denominator1 |
| div(mpq,mpz,mpq) | DivInt; numerator copy, denominator product, divisor-sign action, normalize |
| acc_div(mpq,mpz) | AccDiv; denominator product, divisor-sign action, normalize |
| rat_div(mpz,mpz,mpq) | RatDiv; copy numerator/denominator, normalize, no denominator sign repair |
| addmul/submul(mpq,mpq,mpq,mpq) | AddMul / SubMul |
| addmul/submul(mpq,mpz,mpq,mpq) | AddMulInt / SubMulInt |
| idiv(mpq,mpq,mpq), machine_idiv(mpq,mpq,mpq) | IDiv / MachineIDiv |
| idiv(mpq,mpq,mpz), machine_idiv(mpq,mpq,mpz) | IDivToInt / MachineIDivToInt |
| rem/mod(mpq,mpq,mpq) | Rem / Mod |
| rem/mod(mpq,mpq,mpz) | RemToInt / ModToInt |

The inherited integer-only add/sub/mul/div/machine_div/rem/mod/addmul/submul
operations already belong to IntegerManager; this packet delegates their
integer-valued rational wrappers rather than reproducing integer algorithms.
Increment/decrement forwards, other MPQ predicates/getters, power/root/rounding,
parsers, display and scoped numeral APIs belong to separate packets.

## Source branch/action and alias semantics

Fused +/-1 coefficient branches delegate Add/Sub directly, exchanging the
operation for coefficient-1. Other coefficients compute the product in the
source rational temporary, then perform the source Add/Sub. The non-GMP source
small +/-1 predicates are used, including denominator1 for rational
coefficients. Zero and non-unit coefficients take the full product path.

Synchronized mode uses local integer/rational temporaries and deletes them on
all returns. Non-synchronized mode reuses the existing addTmp1 and addMulTmp
manager fields; normalize uses the existing nTmp cache. Existing manager header
accounting already includes every persistent scratch scalar, so no additional
owning object or header modification was needed. Locals are concrete scalar
numerals backed by the same budgeted IntegerManager cells/Words service. No Go
generics, unsafe, math/big production arithmetic, boxed interfaces, per-numeral
Go heap owner, or independent resource service is introduced.

Public results are staged until the operation succeeds, following the existing
mpq.go predictable allocation failure contract. Source arithmetic and observable
alias actions are preserved rather than treating staging as permission to
correct source behavior. DivInt reads an output-numerator-aliased divisor after
the source numerator copy and an output-denominator-aliased divisor sign after
the product. AccDiv similarly reads an aliased denominator after publication.
RatDiv rereads its divisor after copying the numerator. Frozen examples include:

- RatDiv(6, output.num initially-9, output) yields1, because the source copies6
  into the numerator before rereading that divisor.
- DivInt(2/3, output.num initially-5, output) yields1/3 for the same reason.
- AccDiv(2/-3, the same denominator) yields2/9; the sign sees the positive
  product9 rather than the original-3.
- RatDiv(-5,-5) preserves source representation-1/-1; ordinary RatDiv(6,-9)
  stores2/-3. RatDiv deliberately differs from sign-repairing SetParts.

Integer-valued wrappers delegate directly to IntegerManager with the actual
output numerator identity. They preserve its known source divisor/output alias
quirks: IDiv(-26,7,divisor output) yields-2, IDiv(-26,-7,divisor output) yields2,
and Mod(-26,+/-7,divisor output) yields0. Denominator is reset to1 after successful
integer output. The ToInt overloads preserve the same component identity.

Raw uninitialized rationals, stale manager handles, noninteger operands to
integer-valued wrappers, and zero divisors receive typed errors before unsafe
source actions. ErrRationalNotInteger replaces the source debug assertion;
ErrDivideByZero is the established boundary guard. This also rejects a literal
zero divisor before an output alias could overwrite it. Caller confinement and
no shallow copying of owned numerals remain existing manager preconditions.

## Tests-first evidence and independent checks

Tests first failed RED against HEAD:golang with missing IDiv/Mod/AddInt/MulInt/
DivInt/AccDiv/fused APIs. Shared mpnum had unrelated concurrent display/scoped
RED tests, so the packet's RED and final gates use a HEAD archive under
`.cache/release/mpq-ops-red`, overlaid only the assigned ops production/tests.
No unrelated source files were changed or omitted from that archived baseline.

All24 active ENSURE assertions in src/test/rational.cpp tst9 are preserved:
eight source signed small/big Euclidean division/modulus fixtures each check
reconstruction, nonnegative modulus and the strict absolute-divisor bound.
No missing-overload-specific active assertion appears in src/test/mpq.cpp;
tst0/inverse aliases and tst1 inherited integer assertions are already covered
by existing mpq/mpz/parse tests. Other active rational.cpp scenarios, including
root/modular-inverse and performance cases, are separate source API domains;
this packet does not count them as ported.

Eight named test groups additionally cover bounded independent math/big.Rat /
math/big.Int expectations; both synchronization policies; mixed operations with
in-place output; every fused rational output alias and +/-1/zero/non-unit
branches; rational/int outputs for integer wrappers; big-limb mixed/fused
oracles; persistent versus local scratch behavior; frozen source component
aliases and negative-denominator representation. The initial component-alias
test was independently RED (RatDiv2/-3 versus source1) before implementing the
logical source publication identity.

For12 arithmetic categories, FailAfter advances through every failing prefix
until success. Immediately after a failed operation, **before any String,
Del or Close**, the test independently enumerates expected caller/base-manager
cell roots and only explicitly named source cached roots in nonsynch mode.
It compares exact live numeric-cell counts, then derives expected live Words
refs solely from these cell payloads and NaturalManager's division/format
scratch roots. Synchronized failures also require exactly the pre-call live
cell/Words baseline. Unreachable temporary handles cannot justify their own
presence, and owner Close cannot mask them. Input/output values are then
checked, and whole-owner cleanup refunds all budget charges. Large warmed fused
operations measure zero Go allocations for both source policies.

Native full archived module, native g3checked mpnum, vet, actual fresh checked
JS/WASM named ops tests and full mpnum suite, race ops tests, and WASI executable
build passed. Checked race/actual checked JS were repeated after strengthening
immediate ownership and component-alias audits. WASI semantic execution remains
blocked by the recorded custom Go runtime pre-main seed/environment issue;
this status claims executable build only. All edits, caches, archives, compiler
temporary files and outputs stay under golang. Commit/manifest are coordinator
owned.
