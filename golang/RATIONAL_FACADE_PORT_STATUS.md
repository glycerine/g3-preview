# Owner-local rational arithmetic/operator facade

Source: `src/util/rational.h` (Copyright (c) 2006 Microsoft Corporation),
selected wrapper methods over the synchronized manager. This is a concrete Go
service adaptation; C++ operators are named methods and every owning value is
an `OwnedRational` from one `RationalService`. No global numeric manager, foreign
raw handles, generic Go code, unsafe access, or production `math/big` is added.

## Completed source routes in this packet

- RR `+`, `-`, `*`, `/`: `Add`, `Sub`, `Mul`, `Div`, with the source initial
  lhs copy, in-place compound operation, and copy of the returned **lvalue** into
  the value result. The final copy is not elided or replaced by a manager output
  shortcut. Both int32 operand directions construct an actual rational(int).
- RR `+=`, `-=`, `*=`, `/=`, `%=`: corresponding `*Assign` methods; int32 `+=`
  constructs the rational temporary. `AssignInt` likewise constructs then copies.
- Fresh-result integer `div`, `machine_div`, `%`, `mod`, pair `gcd`; `IDivInto`
  and `ModInto` retain direct raw output identity, including source alias defects.
- Prefix/postfix ++/--: local source `mk_q(1)` rational with Add/Sub, not the
  service cached constant or manager mixed-int Inc/Dec;
  postfix copies the old value before changing the receiver.
- Unary minus, abs, member neg/reset; numerator/denominator value extraction;
  floor/ceil; power/expt (signed exponent casts to source uint32, not reciprocal).
- Structural equality remains core `Equal`. `NotEqual`, `Less`, `Greater`,
  `LessEqual`, `GreaterEqual` retain source swapped-LT and negated-LT routes.
  The four source rational-left int comparisons construct the scalar rational.
- Predicates, parity, power-of-two query, hash, bitsize/storage_size, 64-bit
  conversions, inclusive wrapper unsigned/int32 projections, and double query
  forward to actual manager branches under provenance validation and the lock.
  `IsBig` is the complement of source small representation, a named Go query.
- `AddMulAssign`/`SubMulAssign`: coefficient +/-1 receiver Add/Sub branches;
  otherwise copy **k**, multiply that copy by c in place, then Add/Sub receiver.
  They do not forward to the manager fused-product helper with different order.
- `ModHat`: positive-b assertion; fresh Mod, copy r, multiply copy by rational2,
  strict b<2r test, optional r-=b. Half ties remain positive.
- `NumBits`: construct two, check source domain, copy operand, repeatedly fresh
  IDiv followed by assignment Copy; zero returns one. No bitsize shortcut.

Private concrete `facadeValueLocked` and `facadeAssignLocked` helpers require
an already held service lock; parent `ext_gcd.h` integration can share the
source value-expression routes without passing foreign raw records.

## Ownership and domain adaptations

All input provenance is validated before numeric actions, including scalar
wrappers and multi-operand fused operations. The service mutex covers complete
source expressions. Every intermediate uses existing budgeted manager storage
and lexical destruction. Named-local value returns transfer ownership through
Move; returned compound lvalues use Copy. Errors return an invalid zero token;
no intermediate owning token survives a failed expression. Numeric scratch may
retain known owner-managed buffers according to already documented storage
policies. There is no extra per-value Go allocation or owner pointer.

Source assertion domains become `ErrRationalFacadeDomain` (negative NumBits or
nonpositive ModHat divisor) and existing `ErrRationalNotInteger`. Existing
PowerDomainError retains nonterminating uint32 mask behavior; `Expt(-1)` casts
and fails with that error. Zero^zero remains the source rational result one.
Source raw signed/unreduced denominators remain visible: 2/4 structurally differs
from 1/2 while neither is less, numerator-only positivity reports true for
1/-3, and multiplication by rational1 retains 1/-3. Direct output-aliased IDiv
on -26 with divisor7/-7 gives -2/+2; Mod gives zero. These source-action
fixtures are separate from canonical independent mathematical checks.

## Tests and completion limits

Tests were written before production; an isolated committed module with only
new tests failed because RationalService.Add/Sub/Mul/Div/Rem did not exist.
`TestRationalFacadeSourceTST1` executes all **64 ENSURE** assertions from static
`tst1`; `TestRationalFacadeSourceTST2` executes **56 ENSURE + one VERIFY** from
static `tst2`, each through this actual service. The tests enforce 64 and 57
counts. Five of these source assertions already had constructor-only coverage;
the packet does not claim all 121 as newly uncovered source assertions. Source
TRACE/commented output statements are not counted as active assertions.

Added independent bounded `big.Rat` arithmetic and comparison tests, all binary
self aliases, both scalar directions, postfix/extraction, fused +/-1/all receiver
aliases, finite ModHat/NumBits, source-domain and noncanonical fixtures, provenance
rejection, complete reservation-prefix sweeps for 20 value/expression routes and
18 compound/output alias routes, and warm zero-allocation reuse. Each injected
failure audits cells **and Words** from explicit caller/manager/service roots
before String, Del, or Close; each owner teardown returns its budget to zero.
Source fast paths that do not reserve memory are not forced to fabricate failures.

Central source review corrected ModHat's rational(2) lexical lifetime: it is
explicitly destroyed immediately after the multiplication full expression, before
LT/optional Sub; deferred cleanup remains valid on errors. Source test adapters
now construct actual signed integer numerals for rational(int), reserve SetString
for source text constructors, route copy constructors through Copy, and exercise
source != through service NotEqual. Counts remain 64 and 57; these route fixes do
not add assertions. A second review aligned the prefix local mk_q(1) route,
assertion full-expression cleanup, the TST2 brace-local r0 destructor, the two
nested-expression named constructors, and abs assignment. Each source assertion
now evaluates inside an h.assert closure whose owning-root mark precedes the
condition; newly created expression values are destroyed immediately afterward.
Named source values persist to their lexical boundary. Counts remain unchanged.

This packet does **not** claim rational.h complete. Remaining separate facade
groups include square/root output methods, LCM and manager-extended-GCD facade
outputs, rational bitwise forwarders (raw rational wrappers still missing),
decimal/display service conveniences and safe typed integer interoperability.
Source raw `to_mpq` and reference returns require explicit checked borrowing
policy rather than an unsafe public raw escape. `to_rational` can map to explicit
Copy; compile-time self/zero projections are separate facade decisions. Source
double constructors/assignment are UNREACHABLE, not supported conversion APIs.
Global source initialize/finalize is an owner-local constructor/Close adaptation;
infinitesimal hooks are being integrated separately, not claimed by this packet.
`util/ext_gcd.h` and tst7's999 assertions with500 inverse right-hand sides are reserved for the
parent's separate packet, not replaced by the manager's different ExtendedGCD.

## Completed validation

Isolated committed `HEAD:golang` plus only the two owned Go files passed full
native `go test ./...` and actual `-tags=g3checked` full-module tests. All eleven
`TestRationalFacade*` tests passed native and checked race execution. Actual
checked JavaScript/WASM execution listed and passed all eleven named tests;
checked WASI compiled. Checked full-module vet and `go run ./cmd/portcheck`
passed. After the reviewed source lifetime/constructor/comparison-route corrections,
the frozen bytes reran full native and full actual checked module tests, all
eleven checked race and actual checked JS tests, checked vet, and checked WASI
compilation successfully. No active gate remains.
The isolated prospective module is below `.cache/release/facade-red` (the initial
RED tree was subsequently overlaid with the approved production implementation).
Only the two named new Go files and this status file are the reviewable packet;
no existing service/manager file, ledger, git metadata, or external path was edited.

Parent final combined frozen verification also passed: full uncached native
(session30734), full checked race (63802), and full actual checked JS/Wasm
(67573). All41 new tests across facade, wrapper GCD, both infinitesimal types
and service lifecycle were explicitly observed RUN/PASS in
`.cache/rational-facade-inf-ext-gcd-frozen-checked-js.log`. Checked vet, WASI
cross-build and no-generics/source audit passed. Frozen source/test checksums
remained unchanged. WASI runtime execution is not claimed.
