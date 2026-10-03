# Concrete rational extended-GCD wrapper

Complete concrete rational specialization implemented and verified.

Source `src/util/ext_gcd.h` is a separate Euclidean algorithm from the integer
manager's extended GCD. Its only source consumer is `src/test/rational.cpp::tst7`,
which visits999 positive inputs and evaluates its inverse condition on500 odd
inputs. Manager-level inverse tests do not port this wrapper.

Tests were written first and observed missing-API RED in isolated committed-base
snapshot `.cache/release/ext-gcd-wrapper-red`: all three original test groups
failed to compile because `RationalService.ExtendedGCD` did not exist. A fourth
group then added source output/input and output/output alias fixtures, and a
fifth checks provenance at every input/output position before output mutation.

`mpnum/ext_gcd_service.go` now translates the algorithm on explicitly owned
service values. Inputs copy before outputs initialize; fresh IDiv/Mod results
are assigned by deep copy. Coefficient expressions use the rational facade's
copy-before-compound and returned-lvalue-copy routes. Local cleanup reverses
source construction order. Outputs retain source publication prefixes on error;
no blanket output-alias prohibition or sign normalization is added.

Tests include all999 source inverse cases, an independent signed finite Bezout
and absolute-GCD oracle, source alias fixtures, and reservation-prefix failures
with immediate reachable-cell/payload/scratch accounting before formatting,
caller deletion or service closure. The actual source fixture initializes p
through Power, Copy and temporary deletion, then executes facade Mod/IsOne and
the short-circuited Mul/Mod/IsOne path:999 ENSURE conditions,500 inverse RHS.
The failure sweep observes the exact initialized-output prefix g=17,x=0,y=1
before a loop allocation fails, as well as immediate known-root cleanup.

All five groups passed native, checked race and actual checked JS. Final frozen
combined full-module native (30734), checked race (63802), and actual checked JS
(67573) passed; all41 new packet names have explicit JS RUN/PASS evidence.
Checked vet, WASI compilation and source/no-generics audit passed. No WASI
runtime execution is claimed. The full solver goal remains open.
