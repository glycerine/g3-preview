# Dense polynomial content/division/GCD chain — frozen selected packet

The selected real source chain is implemented and held for coordinator integration. Source: `src/math/polynomial/upolynomial.h/.cpp`, active `polynomial_primes.h`, concrete `svector<mpz>` and scoped numeral/vector ownership, and registered `src/test/upolynomial.cpp`. This extends the committed storage/arithmetic core; whole upolynomial, Sturm/isolation/factorization, algebraic numbers and AST arithmetic plugin remain OPEN. No generic/template-family completion claim.

## Source method map

| Source body/wrapper | Concrete counterpart and actions |
|---|---|
| `get_primitive_and_content`, vector wrapper; `get_primitive` | `PrimitiveAndContent/PrimitiveAndContentViews`, `Primitive/PrimitiveView`, private `primitiveState/primitive`; array GCD precedes output reserve, zero coefficients use actual Set(0), set_size trims, local content destroys after body |
| `normalize` raw/vector | `Normalize/NormalizeView`, private `normalize/normalizeCoefficient`; zero-size no read, singleton positive→1 and otherwise→−1; actual GCD/divisions; g3debug copies old row, divides, multiplies and Equal-checks before reverse destruction |
| scalar coefficient `div` raw/vector | `DivScalar/DivScalarView`, private `divScalar`; debug nonzero/divisibility assertions at source sites; one shortcut, ascending division without trim |
| `div_rem_core` | private `divRemCore`; four source alias checks, d=0 before divisor-count assertion, literal constant branch, q reset/r copy/reserve, actual terminal checkpoints and distinct field/pseudo actions |
| `div_rem` counted/uncounted/raw/vector delegations | `DivRem/DivRemViews`; explicit Go counter pointer exposes the source reference (callers can use a local when irrelevant); scratch success publishes r swap before q swap, including same q/r owner |
| quotient polynomial `div` | `Div/DivViews`, private `divState`; same div_rem_core, reset r scratch before q swap |
| `rem` counted/uncounted/raw | `Rem/RemViews`, private `remState`; explicit counter adapter as above; direct output mutations, no fake transaction |
| `srem` | `SignedRem/SignedRemViews`, private `signedRemState`; repeats source alias assertions, real rem, negates only the actual source parity/sign condition |
| `divides` raw/vector | `Divides/DividesViews`, private `dividesViews`; no checkpoint added; exact leading divisibility/quotient/cancellation/zeroing/trim order in div1 scratch |
| `exact_div` raw/vector | `ExactDiv/ExactDivViews`, private `exactDivState`; initial false keeps old q, later false resets public q; exactDiv scratch success swaps only after set_size |
| `flip_sign_if_lm_neg` | private `flipSign`; empty no read, actual IsNeg then ascending negation |
| `mk_monic` both-output / leading-only / no-output / vector | `MakeMonicWithLeadingInverse*`, `MakeMonicWithLeading*`, `MakeMonic*`; LC and inverse initialized to 1, highest coefficient ownership swap before inverse, descending multiplication. No-output locals explicitly Del(lc) then Del(inv), leading-only owns/deletes only inverse |
| `CRA_combine_images` | private `combineImages`; real extended integer GCD, raw integer modulus, 12 real lexical numerals including unused lower bound, centered images and three ascending span loops, push(default) before copy, bound publication before C2 swap |
| `mod_gcd` | private `modularGCD/modularImage`; all six mgcd scratch vectors, actual content and leading gcds, exact 231-prime sequence, bad degree skips, source i==0 distinction/even-bound initialization, unlucky-degree discard, CRT, candidate short-circuit divisibility and genuine Euclid fallback |
| `euclid_gcd` | `EuclidGCD/EuclidGCDViews`, private `euclidGCD`; captures Field once, actual gcd1/gcd2/rem output swaps and normalization; empty input early copy/sign flip |
| `subresultant_gcd` raw/vector | `SubresultantGCD/SubresultantGCDViews`, private `subresultantGCD`; literal pseudo-remainder correction power, g/h normalization, swap/update order and source divisibility assertion; Field read at terminal site |
| `gcd` raw/vector | `GCD/GCDViews`, private `gcd`; empty early branches, exact Modular dispatch to modular or Euclid algorithm |
| `ext_gcd` raw/vector | `ExtendedGCD/ExtendedGCDViews`, private `extendedGCD`; six actual lexical vector states, all input/output debug aliases, real field assertion, source initialization/division/mul/sub/swap/last monic actions; actual debug BV/expected_D operations and equality before final publication |
| `square_free` | `SquareFree/SquareFreeView`, private `squareFree`; source sqf1 derivative/sqf2 GCD, constant shortcut or actual pseudo quotient then normalization |
| `is_square_free` raw/vector | `IsSquareFree/IsSquareFreeView`; size≤1 no read, actual derivative/GCD, source degree predicate |
| active `g_big_primes` constant table | exact private read-only-in-use `[231]uint32` in `upolynomial_gcd_primes.go`, 39103…41549. The alternative #if 0 table remains unported/inactive |

Existing `upolynomial_ops.go` changed only the approved factoring of `addSubViews` into `addSubState` and `DerivativeView` into `derivativeState`. Public old behavior/regressions stay green. No other existing numeric/service/owner/Git/metadata files changed.

## Ownership, storage and source adapters

The manager's already charged 21 actual vector states suffice: div1/div2/exactDiv/gcd1/gcd2/cra/mgcd[6]/sqf1/sqf2 are used by their actual algorithms. No added fake manager fields, external descriptors for internal polynomials, COW, Go slice polynomial values, generic dispatch or boxed per-coefficient objects. Actual coefficients remain typed pointer-free `Int` arrays in the charged cold backing directory with existing exact capacity/publication/old+new allocation behavior.

Extended GCD declares six stack `upolyVectorState` values; debug declares BV then expected_D. Each owns real directory-backed arrays, deleted/reset in lexical reverse order. `pushDefault` performs actual default construction and size publication before source caller Set. Actual local numeric roots use the borrowed IntegerManager; metadata/header/backing charges remain in the existing physical owners. Compiler escape audit has no moved-to-heap or heap-escaping new raw numerals/vector records/closures. Warm CRA/GCD/division and canceled-image paths prove zero host allocations. Actual extended-GCD temporary arrays allocate as source does, with charged old/new peaks and physical cleanup; they are not cached to invent a warm zero-allocation guarantee.

Array content GCD uses scalar `read(v,i)` at each actual source dereference: reset for zero size, singleton copy/Abs, first-pair raw IntegerManager GCD, early-one exit before later reads, then the final modular normalized assertion. There is no last-element preflight, captured coefficient slice or per-pair modular GCD substitution. `UPolynomialView.WithCount(uint32)` copies only its scalar count/binding without dereferencing its owner, exposing the source (pointer,unsigned size) boundary. Existing bounded View construction could not produce the oversized fixture; the explicit raw-size adapter now preserves that source contract. With first coefficients [2,3], Normalize skips an oversized tail after obtaining one. PrimitiveAndContent publishes content one, reserves output size3 and copies the first two coefficients before the actual tail read fails; [2,3,0] and logical size3 remain without final trim. Other views preserve physical swap lifetime and check actual input dereferences after source setup. Private origin/slot/generation guards map undefined pointers, invalid read spans and closed/foreign owners to typed errors; defined zero-count routes do not dereference backing.

The private scoped_set_zp call site copies the old modulus through normalized mpzzp Copy, changes mode, restores it, then destroys the saved value. Selected mod_gcd always enters from Z. The broader source scoped_set_z/scoped_set_zp API is not advertised as ported: when entered from modular mode its scoped_numeral assignment normalizes the saved modulus to zero and restoration attempts set_zp(0), a source defect, not silently corrected. Uint64 scoped constructors/general public scopes remain OPEN.

On Go allocation/assertion/cancellation returns, scoped locals and raw source helper locals are explicitly cleaned as the safe Go exception adapter. Source raw mpz has no destructor, so failed mk_monic overload locals and other raw exception paths do not have an equivalent C++ cleanup guarantee. Caller/scratch/counter prefixes are retained; cleanup does not fake operation rollback. If mode restoration itself fails, its typed error takes priority; otherwise the original body error is returned directly. There is no uncharged errors.Join wrapper. No borrowed numeric/resource/cooperation owner is silently closed. Actual source `_NO_OMP_` services are required by all runtime root fixtures; nil/default-success callbacks are not substituted.

Preserved finite quirks include singleton zero Normalize→−1; release primitive all-zero handling; non-field constant pseudo-division copies A into q without dividing; constant Rem resets even without reading its value; SignedRem skips negation for odd d/negative leading coefficient; initial/later ExactDiv rejection distinction; empty-field GCD sign-flips without monic conversion; MakeMonic publishes highest 1 before inverse failure. Source uint32 d wraps normally. Source ExtendedGCD makes initial A monic without rescaling U, so nonmonic release cases can violate the stated Bezout identity. Its actual debug check stops before final normalization, retaining partial coefficients. Shared U/V output owners are accepted, and their final inverse scaling happens twice. None of these are replaced by improved algebraic algorithms or invented alias restrictions.

## Tests-first and exact registered coverage

The first observed isolated RED on committed2c89d3f plus the new root division fixture reported missing ExactDiv/Divides/DivRem (`.cache/upoly-division-red.log`). All ten actual registered exact-div cases execute their three ENSURE routes: **30 executions**. Expanded integer factor coefficients are explicitly the multivariate-conversion input adapter; actual dense ExactDiv, Divides and source Equal routes execute. The actual multivariate package/conversion/printer test path remains OPEN.

Later observed missing APIs (`.cache/upoly-gcd-red.log`, `.cache/upoly-gcd-registered-red.log`) preceded the primitive/normalize/rem/GCD/extended/square-free algorithms. The approved state helper factoring followed this relevant missing-algorithm RED. Registered active first GCD derivative example, both Z/Zp and Z/Zp2 datasets, and both extended GCD examples execute the actual dense source calls with independent checks. These are print-only source cases, so add **zero** registered ENSURE/VERIFY count. Later tst_gcd examples after unconditional return, disabled huge gcd2, and unrelated factorization/Sturm/isolation registrations stay explicitly OPEN.

The remainder helper's nine ENSURE executions depend on two real multivariate univariate predicates and final multivariate equality. They are **not** counted as ported from tautological dense predicates; selected remainder value/partial-action fixtures are additional adapted/model coverage. Existing source nine pretranslation ENSURE fixtures remain unchanged.

There are **21 new permanent root test groups**, plus 18 existing root and 3 scalar mpnum groups, for **42** exact selected RUN/PASS in checkeddebug race and actual checkeddebug JS. Independent field convolution supplies nine divisions; exact integer/field input oracles and root multiplicity expansion check real outputs. 576 injected allocation ordinals across 12 algorithm families inspect actual numeric live-cell and Words-reference totals immediately before any String/Del/Close, permitting only actual caller content/leading/inverse output roots and real manager cached roots. Source completed publication prefixes, raw initialized-slot reserve effects, direct release aliases, same q/r, source U/V alias scaling, and nonmonic defect are frozen separately from mathematical expectations. Prime tests prove ordinary two-image CRT (10 checkpoints), unlucky degree discard (7), initial bad prime and i>0/even-bound initialization (10), and all231-primes-bad fallback (2), using permanent hermetic finite data.

Allocation audit found canceled modular images allocating two uncharged error-wrapper objects: `.cache/upoly-gcd-cancel-allocation-red.log` records RED=2; direct typed error return gives GREEN=0 in `.cache/upoly-gcd-cancel-allocation-green.log`. Successful warm GCD/CRA/division also has zero Go allocations and stable whole physical backing/budget usage. Root fixture cleanup verifies full borrowed-owner graph refund.


The late-CRA allocation sweep uses actual G=x^8−100000, A=G*(x+2), B=G*(x+3), with independent nine-coefficient expected G and ten real checkpoint calls. It executes 55 reservation ordinals through the successful last ordinal (54 successful reservations): seven refusals follow the first-image prefix, including four after an actual CRA coefficient append (`cra.size>0`). Together with the earlier 576 attempts this is 631 immediate live-cell/Words-root audits before formatting, Del or Close, with mode restored to Z. This bounds the tested workloads; it does not claim exhaustive allocation coverage for all inputs. The initial degree-one sweep exposed insufficient coverage (22 reservations/three late/zero after append), recorded in `.cache/upoly-gcd-cra-sweep-debug.log`; the real higher-degree algorithm fixture passes in `.cache/upoly-gcd-cra-sweep2-debug.log` without synthetic callbacks or allocator profiles.

Raw-size/read-timing tests first observed missing WithCount API in `.cache/upoly-gcd-raw-count-api-red.log`, then the premature last-read failure in `.cache/upoly-gcd-raw-count-behavior-red.log`. Literal indexed traversal passes in `.cache/upoly-gcd-raw-count-green.log` and preserves the content/output failure prefix above.

## Frozen evidence

Snapshot `.cache/release/upoly-gcd-final-13be17b`: committed HEAD `13be17b` plus only the ten Go files below. All final42 gates are terminal exit0. Earlier final2/40-group evidence is superseded by this snapshot.

| Gate | Session | Evidence |
|---|---:|---|
| full module native | 95439 | `.cache/upoly-gcd-final42-native.log`; root2.695s, mpnum22.232s, all modules pass |
| full module g3checked,g3debug | 46255 | `.cache/upoly-gcd-final42-debug.log`; root3.218s, mpnum22.815s, all modules pass |
| checkeddebug race, all selected | 35234 | `.cache/upoly-gcd-final42-race.log`; exact42 RUN/42 PASS |
| actual checkeddebug JS, all selected | 63342 | `.cache/upoly-gcd-final42-js.log`; exact42 RUN/42 PASS, real node executor preserves subtree TMPDIR |
| checkeddebug vet | 93729 | `.cache/upoly-gcd-final42-vet.log` |
| checkeddebug WASI build | 9442 | `.cache/upoly-gcd-final42-wasi.log` |
| reference-source and no-generics audit | 80758 | `.cache/upoly-gcd-final42-audit.log`;2102units677APIs33ported144partial1925unported before coordinator polynomial metadata |
| compiler escape audit | 26458 | `.cache/upoly-gcd-final42-escape.log`; no new raw local records/closures escape |

A separate cache-only performance tree `.cache/review/upoly-gcd-perf-13be17b` retains the identical frozen Go and adds only a benchmark fixture. One 200ms linux/amd64 run on AMD Ryzen Threadripper3960X uses actual IntegerManager, utilReslimit and utilCooperation bindings, warms real two-image modular GCD plus DivRem, and checks manager bytes/live backing and whole-budget plateau. Result: **16254 ns/op, 0 B/op, 0 allocs/op**, whole-budget plateau **6160 bytes**. Evidence `.cache/upoly-gcd-benchmark.log`, session99793 exit0. This is a single local measurement, not a source/C++ comparison or general performance guarantee. No benchmark file is part of the production/test packet.

All writes/build caches/temp files remained under golang; no external reference compiler invocation or external writes occurred in this packet.

Exact new test names:

- `TestUPolynomialExactDivisionSourceThirtyENSURE`
- `TestUPolynomialFieldDivisionIndependentFinite`
- `TestUPolynomialDivisionPublishedPrefixes`
- `TestUPolynomialContentNormalizationSourceBranches`
- `TestUPolynomialRemainderSignedSourceBranches`
- `TestUPolynomialExtendedGCDSourceZ7`
- `TestUPolynomialGCDAndSquareFreeIndependent`
- `TestUPolynomialGCDRegisteredZAndZPInputs`
- `TestUPolynomialExtendedGCDRegisteredZ13Input`
- `TestUPolynomialGCDRegisteredFirstDerivativeExample`
- `TestUPolynomialGCDChainImmediateFailureRoots`
- `TestUPolynomialGCDRawZeroCountsAndSourceAliases`
- `TestUPolynomialModularGCDCRAAndAllPrimeFallback`
- `TestUPolynomialGCDAndDivisionWarmPhysicalPlateau`
- `TestUPolynomialExtendedGCDNonmonicAndOutputAliasQuirks`
- `TestUPolynomialPrimitiveRawAliasReserveAndMonicPrefixes`
- `TestUPolynomialGCDChainRawOverloadAndEmptyFieldRoutes`
- `TestUPolynomialCancelledModularGCDZeroHostAllocation`
- `TestUPolynomialModularGCDUnluckyDegreeAndInitialBadPrime`
- `TestUPolynomialContentRawEarlyOneAndLateReadPrefix`
- `TestUPolynomialModularGCDCRALateFailureRootSweep`

Frozen Go SHA-256 inventory (`.cache/upoly-gcd-frozen-go.sha256`, all ten current files verified unchanged):

```text
0e5b8f6368096686d8c1f392e0cc30320b9226e958533dcac8c36459a546da4c  mpnum/upolynomial_ops.go
91751866907ca6260f36e5e018b4936e24a87970c54304dc6416b78a186445ee  mpnum/upolynomial_content.go
bcfe6b60bc45e5be67313f3a39f5b8250b5b4fba3455cd0b2efa61d82777717d  mpnum/upolynomial_division.go
ffb07363c7895a901f1bed9689a584e5d0971248774c7f4674f1bd02d39b95d8  mpnum/upolynomial_division_remainder.go
7b1ed3d4e5a38199a6897b698f7d83d426bb27454377fadb4a3dd1dec95f4e27  mpnum/upolynomial_gcd.go
8f150b0e0b60f187fb067a9f7a22bad9c6fceb0af8ac7cb281f5735346ca91bb  mpnum/upolynomial_gcd_extended.go
6576bdef0319ad2d828a971d74f90abe3f302b4f045e7dcee958b9af6a6eae1f  mpnum/upolynomial_gcd_modular.go
fa1145e05c167a3100b0f0b9e3d7d43e218376abe22418d26f5ecb0732c88e9e  mpnum/upolynomial_gcd_primes.go
f7a3b4c5119e3ccaf403c111493ca9ee3c025aeda3fcd51a184f1e5cd1fee346  upolynomial_division_source_test.go
75da8d974564582f57c603fef56b5da2afc0f79b630b96fe515a16dc2706ec39  upolynomial_gcd_source_test.go
```

_TRACE formatting/events, general polynomial display/SMT2, full scoped-mode API, factors, sequence objects, transforms/translation/reciprocal, complete resource-manager variants, GCD consumers through real multivariate conversions, Sturm/isolation/refinement and algebraic-number/AST integration remain OPEN. This packet supplies actual algorithm prerequisites; it is not final success for the full arithmetic backend.


## Coordinator final integration

Frozen integration `.cache/review/upoly-gcd-integration-13be17b` contains
committed13be17b plus only the ten reviewed polynomial Go files. Root full
native session86220 and checked/debug session69093 terminated exit0, with
root-package2.778s/3.198s and mpnum22.516s/22.978s. Final source/no-generics
audit session14656 terminated exit0:2102units677APIs33ported/145partial/1924unported.
Metadata evidence links executable root-package fixtures to their corresponding
methods; the full unit target lists retain the concrete mpnum implementation
files. Initial metadata-only missing-package fixture links were corrected;
verification code and all Go implementation/test bytes stayed unchanged.

Coordinator checked all ten SHA256 values and both exact42 named RUN/PASS
inventories in the terminal race/actualJS logs against
`.cache/upoly-gcd-integration-final-hashes.json`. Reviewed selected content,
division/remainder, all GCD variants, prime reconstruction and square-free
actions directly against the preserved C++ and obtained independent review.
All231 ordered active prime literals match source; an independent Python
integer product matches the permanent1064-digit all-prime fallback fixture.
Observed first missing-API REDs and raw early-one behavioral RED precede the
implementations/correction. The late reconstruction fixture reaches55 actual
reservation ordinals, including seven failures after the first image and four
after an actual CRA append, while every immediate numeric/Words root remains
accounted. This finite fixture is not a claim of exhaustive failures for all
possible polynomials.

The untracked full-SAT and next integer publication-failure packet are outside
this committed-base freeze. Their pending work does not establish a green full
live workspace or complete solver. The complete port/interpolation goal remains
active with the unported arithmetic and solver consumers listed above.
