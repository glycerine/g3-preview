# Owner-local rational service core

Status: selected core implemented and tested. Subsequent infinitesimal lifecycle
integration completes active `util/rational.cpp`; `util/rational.h` remains
**partial**. See RATIONAL_INF_LIFECYCLE_PORT_STATUS.md. Core files are
`mpnum/rational_service.go`, `mpnum/rational_service_test.go`, and this status.
Existing manager files are unchanged; the parent integrates the global port
ledger and manifest for this packet.
No Go generics, unsafe storage, boxed hot records, or per-value heap owners.

## Source mapping and scope

| Source | Concrete translation |
| --- | --- |
| `src/util/rational.cpp:26-30`, `:64-69` | `NewRationalService` creates one owned synchronized RationalManager and inline zero/one/minus-one constants in source order. Manager and constants are never global. |
| `src/util/rational.h:41` | `NewValue` returns an initialized zero numerator/one denominator with private scalar owner provenance. |
| `src/util/rational.h:43`, `:131-134` | `Copy` delegates the existing manager's deep pair copy. Independent output values never alias payload ownership. |
| `src/util/rational.h:44`; `src/util/mpq.h:34`; `src/util/mpz.h:97` | `Move` transfers raw references while preserving the moved-from numerator/denominator `m_val` history. It does not reset the source wrapper to zero. |
| `src/util/rational.h:46,48,50,59,62,65`; `src/util/mpq.h:673-739` | `SetInt64`, `SetUnsigned` (unsigned32 inclusive INT_MAX), `SetUint64` (typed64 strict INT_MAX), `SetFractionSigned`, `SetFraction`, `SetString` expose the selected constructor setter operations through existing numerical algorithms. |
| `src/util/rational.h:67` | `Del` performs numerical destruction and invalidates the caller's scalar owning token; a second Del rejects the destroyed token. |
| `src/util/rational.h:89` | `Swap` delegates exact manager pair swap without copies/allocations. |
| `src/util/rational.h:91-93,234`; `:302-313` | `String`, `Equal`, `Zero`, `One`, `MinusOne`; constant getters return independent owned copies instead of exposing the source const reference. |
| `src/util/rational.cpp:32-54` | `PowerOfTwo`, private `fillPower`, `pushCache`: first entry is deep-copy one; deep-copy current last entry; local two; multiply then placement-copy each entry through inclusive `k+1`; deep-copy result. No decimal conversion implements arithmetic/copy. |
| `src/util/vector.h:59-100,351-357` | `growCache` starts capacity 2 then `(3*old+1)/2`; each element is moved then its moved-from wrapper destroyed. Payload refs remain stable; old refs clear while sign history remains. `pushCache` grows first, then deep-copy constructs before incrementing length. |
| `src/util/vector.h:47-52,120-127`; `src/util/rational.cpp:78-83` | `Close` invokes private production `destroyCache` to destroy live cache entries ascending, frees backing, destroys zero/one/minus-one in order, closes manager, releases service header. Idempotent. |
| Owner adapter; underlying explicit `RationalManager.CopyFrom` bridge | `CopyFrom` checks both service provenance/lifetimes, orders two cold mutexes by scalar owner ID, and invokes root-owned limb copy bridge. Destination ownership remains independent after source close. |
| Owner/budget diagnostics adapter | `OwnerID`, `Stats`, constructor unwind and cold self-identity checks; no new numerical algorithms. |

Source infinitesimal initialize/finalize hooks at `rational.cpp:56-62,70-71,76-77`
are now implemented by the subsequent owner-local lifecycle packet. Subsequent
RATIONAL_FACADE_PORT_STATUS.md covers selected wrapper arithmetic/comparisons.
Remaining wrapper roots/squares, bitwise, display variants and checked raw
interoperability are not claimed by this core packet. It does not complete
all active `src/test/rational.cpp` tests.

## Ownership, synchronization and checked boundaries

`OwnedRational` holds only private `Rational` and scalar uint64 owner ID. Both
this record and the raw Rational cache backing pass `arena.ValidateRecordType`
under release and checked layouts. The service owns its manager; no public raw
numeral or manager borrow escapes. Scalar provenance rejects a foreign value
even when numerical slots match or both values are inline. Service Close
invalidates every outstanding value, and a copied value in another service
remains valid. IDs are monotonic global atomic namespace metadata only; no
numerical cache/manager/lifetime is global, and overflow never wraps/reuses.
Only serialized tests restore the namespace counter to exercise exhaustion.

Owning Go token shallow copies remain forbidden by contract, as with primitive
Int/Rational; no per-value ledger is invented. The cold service self pointer
rejects reconstructed/shallow copied owner identities. Tests do not copy a
live mutex as a Go value. All service manager/cache actions take its mutex;
CopyFrom locks both owners in ID order. An externally shared Budget remains
owner-confined under the existing Budget contract: separate service locks do
not make externally concurrent reservations safe.

Signed int32 fraction setters preserve the source valid-input sequence: negate
numerator then denominator when denominator is negative, before either setter.
The operation uses widened signed magnitudes for INT_MIN negation (undefined in
source C++); boundary results are an explicit defined adapter, not an observed
C++ result. Signed-pair SetFractionSigned stages both widened fields with signed integer
setters in numerator/denominator order, normalizes, then Move publishes. This
preserves the signed32 INT_MAX denominator representation; typed64 SetFraction
retains its distinct unsigned64 denominator shape. Local pair destruction
provides predictable failure safety. Extremely large power indexes reject uint32 endpoint/length or host
backing overflow before any cache/charge change.

## Budget and failure-prefix policy

The actual service header is charged with reflect size, including constants,
mutex, self identity, cache slice counters, and scalar ID. Manager/nested
numeric storage uses existing own graph charges; it is not precharged again.
Borrowed Budget and caller-owned value/output/string storage are external.
Cache backing is actual `capacity * sizeof(Rational)`. Slice size/capacity
metadata is already in the service header, so source vector header counters
are not duplicated. Every expansion reserves the full new array while the
old array remains charged, then transfers ownership and releases old backing.
Constructor failures close any constructed child and refund the owner header.
Close releases cache/header and the complete owned manager graph.

Predictable cache failure preserves successful source prefixes, including
backing growth before a failed copy. A placement-copy failure does not append a
live default record. Failed arithmetic/result copy destroys locals immediately.
A failed request for k can have committed cache[k] before failing the extra
cache[k+1]; retrying k then returns its cached value without extension, exactly
as source successful-prefix control flow dictates. Numerical manager staged
failure adaptations are inherited and explicitly distinct from source fatal
allocation failure. Successful reservations consume Budget.FailAfter prefixes
normally; teardown refunds Used charges without resetting the fault counter.

## Tests and validation

Tests were written before implementation. First observed RED was missing
RationalService/OwnedRational API compilation. Additional lifecycle/integration
checks against existing manager behavior passed without manufactured RED.

Five exact active source ENSUREs are ported from `src/test/rational.cpp` lines
30,31,74,119,120: reduced fraction equality, unequal one/half, string 100/20
normalization, equal huge reduced fractions, and unequal huge integer/fraction.
No source power-cache ENSURE exists; its tests use an independent bounded
math/big oracle and the audited source storage model.

15 named test groups:

- `TestRationalServiceSourceConstructorAssertions`
- `TestRationalServiceConstantsSettersAndMoveHistory`
- `TestRationalServicePowerCacheSourceModel`
- `TestRationalServiceProvenanceCopiesAndLifetimes`
- `TestRationalServiceLayoutIdentityAndExhaustion`
- `TestRationalServiceConstructorBudgetUnwind`
- `TestRationalServiceCacheFailurePrefixes`
- `TestRationalServiceCopyFailureAndWarmReuse`
- `TestRationalServiceCacheBackingPeakAndBounds`
- `TestRationalServiceConcurrentCacheAndClose`
- `TestRationalServiceCrossOwnerFailureAndSameOwnerCopy`
- `TestRationalServiceForeignSmallValuesAndInvalidGuards`
- `TestRationalServiceCacheRelocationMoveAndDestruction`
- `TestRationalServiceSourceUnsignedConstructorBoundary`
- `TestRationalServiceSourceSignedFractionShape`

Coverage includes cache powers through255, inclusive extra entries, exact
geometric capacities, payload independence, negative denominator/INT_MIN defined
boundaries, signed/unsigned64 constructors, explicit moved-from raw history,
real ascending cleanup order via LIFO reused slots, unique provenance and
cross-owner close, deep same/cross owner copies, cold owner copy guards,
70 constructor/cross-owner fault prefixes and160 cache fault prefixes, exact
old+new backing peaks, invalid guard no-charge behavior, and retained warmed
PowerOfTwo/Del at zero Go allocations under FailAfter(0). Failure oracles check
live cells and Words immediately before teardown; retained natural-manager
format/division scratch is explicitly counted, since diagnostic String may
legitimately grow persistent manager scratch.

Final gates are recorded below when complete.

## Source SHA256

- `src/util/rational.cpp`: `4479ae33541ebd0481884deab48fabea327183d53e34ab3c6639246812adbd69`
- `src/util/rational.h`: `a23db256b5025d27e9bc295e0aa3c0d2d3892baf7d5c6845f35cff3579ea4b9c`
- `src/util/vector.h`: `aafafc0fe6ca0afd7bafe0c781342da259f9c34c0b94b16732c415cf2733066e`
- `src/test/rational.cpp`: `a6ef1049711361be794fcca1404de39d063f2f46f709aa8d3312461e15e1f8b6`
- `src/util/mpq.h`: `89498b6aced1bee1e7ec72e782bee3ae1549c47c0cc228a46d0d288a6353db58`

### Final gate results

All build caches/temp/output remained under `golang/.cache`.

- Full native `go test ./... -count=1`: PASS, session42240, all packages.
- Full checked native `go test -tags=g3checked ./... -count=1`: PASS, session99640.
- Targeted checked race `go test -tags=g3checked -race ./mpnum -run '^TestRationalService' -count=1 -v`: PASS, session38978, all13 named RUN/PASS records.
- Checked `go vet -tags=g3checked ./mpnum`: PASS.
- Checked JS build: PASS, session76601, `.cache/rational-service-js.test.wasm`.
- Actual minimal-environment Node checked JS runtime with `-test.count=1 -test.v`: PASS, all13 named RUN/PASS records observed (not merely a cached summary).
- Checked WASI cross-build: PASS, session68308, `.cache/rational-service-wasi.test.wasm`.
- `go run ./cmd/portcheck`: PASS, session8347, no-generics/source syntax audit; inventory still14ported39partial2049unported before parent metadata integration.

The initial vet gate correctly rejected a test assignment that copied a mutex;
the test now reconstructs the invalid cold self identity without copying any
mutex. The ascending-cache-cleanup reuse observer now invokes production
`destroyCache`, the same helper used by Close, under the owner lock. It observes
actual source traversal via manager LIFO slot order before manager destruction.


### Final boundary refinement

Tests first observed RED for missing `SetUnsigned` and the production
`destroyCache` helper. Both are implemented. The unsigned32 constructor setter
is separate from typed uint64: unsigned32 includes INT_MAX inline, uint64 stores
INT_MAX as Big. Four additional independent source-boundary checks cover
INT_MAX-1, INT_MAX, INT_MAX+1, UINT_MAX, actual raw numerator/denominator shape,
mathematical value, and immediate release of an old big numerator/denominator.
These checks do not increase the count of five active source ENSUREs. Cache
cleanup tests exercise production traversal, not a mirrored test loop.
Production/tests frozen at14 named groups; final affected reruns follow.

Final affected gates after the boundary refinement all PASS:

- Targeted native all14 named groups, session92865.
- Targeted checked race all14 named groups, session66515.
- Checked vet, exit0.
- Fresh checked JS compile72333 and actual minimal Node execution, exit0 with all14 explicit RUN/PASS records including `TestRationalServiceSourceUnsignedConstructorBoundary`.
- Checked WASI compile63458.
- No-generics/source syntax audit21602, exit0 (parent in-progress inventory14ported40partial2048unported).

The earlier full native/checked module gates preceded this tiny refinement;
parent owns the final frozen combined full-module gates. No runtime PASS is
inferred from compilation or cached package summaries.


### Signed fraction representation correction

A source-first rawshape fixture observed actual behavioral RED for `(1,INT_MAX)`:
the previous delegation to the typed64 fraction path created a Big denominator.
The corrected staged signed pair uses z.SetInt64 for BOTH widened numerator and
denominator, normalizes, then Move publishes. Source sign flips occur before
both setters. Typed64 SetFraction is unchanged. Five independent signed32
pair fixtures check rawshape, mathematical values and immediate old-big cleanup;
24 fault prefixes check unchanged destination and immediate live cell/Words
baselines before owner teardown. Active source ENSURE count remains five.
Production/tests frozen at15 named groups after this correction.

Final affected gates after signed-pair correction all PASS:

- Targeted native all15 named tests, exit0.
- Targeted checked race session92049, all15 explicit RUN/PASS records.
- Checked vet, exit0.
- Fresh checked JS compile and actual minimal Node execution, exit0,
  all15 explicit RUN/PASS records including signed fraction rawshape/fault test.
- Checked WASI compile, exit0.
- No-generics/source syntax audit13916, exit0.

Earlier combined full-module binary evidence remains historically distinct;
parent records the final combined module gates against this frozen correction.

### Final combined verification

After the signed-fraction correction, full uncached native tests and full
checked race tests passed across every module package (sessions5421 and18297).
Full uncached actual checked JS/Wasm execution passed (session45897), with all
40 new tests from the copy, number-theory, inspection and service packets
observed explicitly RUN/PASS in
`.cache/copy-inspect-number-theory-service-final-checked-js.log`.
Full checked vet and checked WASI cross-build passed; WASI execution is not
claimed. Source inventory and no-generics audit passed (session24670):
2102 units, 677 APIs; 14 ported, 41 partial, 2047 unported.
Source copyright notices are retained in both translated service files.
