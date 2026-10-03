# Rational service infinitesimal lifecycle integration

Implemented and verified with both complete active infinitesimal representations.

Source `src/util/rational.cpp:64-83` initializes the synchronized manager,
rational zero/one/minus-one, rational infinitesimal constants, then integer
infinitesimal constants. Finalization destroys rational infinitesimal constants,
integer infinitesimal constants, powers cache, rational constants, then manager.
The six constants now belong to the explicit service rather than process globals;
their actual record storage is included in the reflected service-header charge.

The parent owns `mpnum/rational_service.go` integration and
`mpnum/rational_service_inf_lifecycle_test.go`. Representation files and complete
tests are separately owned by the two implementation agents. Constructor failure
closes partially initialized constants before manager/header release.

Initial behavioral RED: `TestRationalServiceInfinitesimalHeaderOwnership` failed
because the service lacked `infZero`. After integration, all three lifecycle tests
passed in an isolated snapshot under `.cache/release/rational-inf-lifecycle`:
native, checked race (session50345), and actual checked JS/Wasm (session89197),
with each named test observed RUN/PASS. This snapshot excludes concurrent agent
test files with APIs still under implementation and is not full-module evidence.
The constructor fault sweep covers80 reservation prefixes including successes;
the initial20-prefix sweep was insufficient to reach success and was extended.

The finalizer test invokes the production combined hook, observes immediate
live-cell/payload refunds before formatting or Close, verifies that the powers
cache remains intact, and observes actual reverse free-slot reuse. Constants
are traversed zero/one/minus-one; each rational infinitesimal destroys its second
component before first, matching reverse C++ member destruction. Each rational
component itself deletes numerator before denominator through the source manager.

Final combined frozen native suite (30734), full checked race suite (63802),
and full actual checked JS/Wasm execution (67573) passed. All41 new packet
tests, including these three lifecycle tests, have explicit JS RUN/PASS records
in `.cache/rational-facade-inf-ext-gcd-frozen-checked-js.log`.
Checked full-module vet, WASI build and source/no-generics audit passed. Frozen
source/test checksum verification passed. WASI runtime execution is not claimed.
The active rational.cpp algorithms/hooks are complete; its header facade and
the full solver/AST/proof/interpolation goal remain incomplete.

Final source inventory/no-generics audit87047 passed:2102 units,677 APIs;
21 ported,40 partial,2041 unported. Inventory does not prove solver completion.
