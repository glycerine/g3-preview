# Selected integer/rational number-theory port

Ownership: `mpnum/mpz_number_theory.go`, `mpnum/mpq_number_theory.go`,
`mpnum/mp_number_theory_test.go`, and this status. Concrete APIs only; no Go
generics, production `math/big`, unsafe storage, or global numeric manager.

## Source-method inventory

| Source | Go counterpart | Coverage |
| --- | --- | --- |
| `mpz.cpp:845` active array `gcd` | `IntegerManager.GCDMany` | Active sequential algorithm, empty/singleton, early one, destination/future-element aliases |
| `mpz.cpp:915` extended `gcd` | `IntegerManager.ExtendedGCD` | Selected release algorithm, exact quotient/coefficient action order, input-sign rereads, final result copy, six-local destruction |
| `mpz.cpp:985` `lcm` | `IntegerManager.LCM` | All signed shortcuts and division/product branch; destination/input aliases |
| `mpz.cpp:2054` `divides` | `IntegerManager.Divides` | Zero branch (`0` divides `0`), scoped remainder branch |
| `mpq.h:601` rational pair `gcd` | `RationalManager.GCD` | Integer inputs, numerator operation then denominator reset |
| `mpq.cpp:102` rational array `gcd` | `RationalManager.GCDMany` | Sequential branches; fractional singleton is allowed; future operands skipped after one |
| `mpq.h:609` rational extended `gcd` | `RationalManager.ExtendedGCD` | Output denominators reset before integer algorithm; selected release algorithm |
| `mpq.h:619` rational `lcm` | `RationalManager.LCM` | Integer inputs, numerator operation then denominator reset |
| `mpq.h:627` rational `divides` | `RationalManager.Divides` | Integer-input validation and integer delegation |

The integer-valued `mpz` forwarding overloads in `mpq.h` are represented by the
concrete IntegerManager API, not duplicate methods with ambiguous operand types.
Previously ported pair integer GCD remains the implementation used by array GCD
and LCM; this packet does not replace its algorithm.

## Source actions and adaptation

Array lengths are bounded by source uint32 widths. Only visited array elements
are validated: array output may overwrite a future element, and an early result
of one avoids inspecting remaining operands. Rational singleton GCD copies and
takes the absolute value without imposing pair-GCD's integer precondition.

Extended GCD publishes each source coefficient action directly. Output/output
and input/output aliases retain source release behavior, including surprising
coefficient results when a modified input sign is reread. Failure preserves
already published output prefixes; it does not impose a transaction around the
whole algorithm. Rational extended GCD resets all three output denominators
before allocating integer locals. The six integer locals are always deleted,
including on budget failure. LCM and Divides similarly clean their scoped local.
Numeric cells and Words payloads use the existing owner budget. Both rational
scratch policies are tested; no extra cached manager fields were introduced.

Source integer precondition assertions become `ErrRationalNotInteger`; stale
handles use existing manager guards. Raw records require genuine caller-supplied
owners; matching foreign slots/generations are not intrinsically detected. Signed LCM
shortcuts are intentionally preserved: for example LCM(-6,-6) is -6 and
LCM(-6,-3) is 6. Source debug tracing is not implemented.

## Tests and remaining conditional coverage

Tests were written first; the initial RED was missing all nine selected APIs.
Nine test groups cover the active `rational.cpp` tst1 gcd(12,8)=4 assertion and all
nine tst6 signed calls (which have no source assertions), strengthened by fixed
coefficients and independent `math/big` checks. The tst7 loop over 1..999 supplies
the same odd modular-inverse property to the manager implementation (500
assertion executions across999 cases). Source tst7
uses the separate `ext_gcd.h` wrapper algorithm: that wrapper remains unported,
so this property test does not claim its algorithm or wrapper API is ported.
The commented-out registration of mpz's long `tst_gcd` run and the disabled
`#if 0` sorted-array GCD optimization are not claimed as active coverage.

Independent finite oracles check 1,925 signed pairs, signed LCM fixtures, zero
cases, array ordering, fractional singleton/skipped-tail behavior, fixed release
alias coefficients, large scaled coefficients, and Fibonacci coefficients.
Injected allocation prefixes are exercised through first success for rational
extended GCD, LCM, Divides, array GCD, and pair GCD. Immediately after each
failure, before formatting, deletion, or Close, an independent live-root audit
checks numeric cells and Words references, allowing only published caller roots
and actual manager-owned scratch. Budget refunds after Close are checked too.

**Conditional coverage update:** `mpz.cpp` Extended GCD's `#ifdef Z3DEBUG`
mul/mul/add/equality sequence is now selected by the separate `g3debug` build
tag. `g3checked` still selects Go handle validation independently. The diagnostic
reuses the actual coefficient locals and rereads modified input aliases before
the final gcd assignment; a failed source assertion becomes a classified error
with source output prefixes intact. See EXTENDED_GCD_DEBUG_PORT_STATUS.md.
No blanket alias restriction suppresses finite release cases. The separate
ext_gcd.h wrapper has also since been translated; see EXT_GCD_WRAPPER_PORT_STATUS.md.

## Verification

Verification uses an isolated `HEAD:golang` archive plus only this packet under
`golang/.cache/release/number-theory-final`; all cache/temp/output paths stay
under golang. Full native and full checked module tests, checked mpnum tests, checked race
tests for all nine NumberTheory groups, vet, and the no-generics syntax audit
passed. Fresh actual checked JS execution (`-count=1 -v`) ran and passed all nine
named NumberTheory groups; the full checked JS module suite also passed, including
mpnum (30.016s). Checked WASI test compilation passed; WASI runtime execution is
not claimed.
