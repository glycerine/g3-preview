# LP port implementation and test coverage

The full `src/util/lp` port is **in progress**. The inventory includes all 110
files, including ignored-by-default `.hpp` implementation bodies. See
[PORT_STATUS.json](PORT_STATUS.json) for every source file's current targets and
tests; `port_manifest.json` remains the repository-wide source ledger.

There are currently forty-seven partial source units and 63 unported units. No complete
LP solver is available yet. A green Go coverage report does not measure the C++
code that has not been ported.

| Component implemented so far | Verification | Statement coverage |
| --- | --- | --- |
| Exact dense-block construction and pivot stages | 120 independent rational elimination worlds checked at every pivot; pooled/direct actual allocation refusals through success, partial L publication, borrowed parent and older comparison ownership; precise tiny coefficients and reached bounds/assertion prefixes | 82.4% native (197/239), 87.0% debug (208/239); gaps propagate child/cleanup errors, guard invariant-protected offsets or belong to the other profile; parent publication and tail applications remain open |
| `int_set.h` | Original `test_int_set`; dense swap order, resizing, reached invalid-domain clear prefix, 10,000-step independent map oracle, output/error cases | 100% |
| Four live `stacked_value.h` scalar types | Original `test_stacked_unsigned`; signed, status and strategy restoration, nested/excess/zero pops, unsigned wrap | 100% |
| Three concrete `stacked_vector.h` element types | Original `test_stacked_vector` with strengthened assertions; 12,000 operations against independent snapshots, repeated replacements, popped tails, baseline journals and checked scope/index guards | 100% |
| Exact `numeric_pair<mpq>` arithmetic | 400 arbitrary-precision worlds against `math/big.Rat`, 625 lexical comparisons, aliases, missing-cell error prefixes and strengthened ten-term source accumulation | 100% |
| Floating/unsigned indexed vectors | 12,000 operations against independent sparse maps; source equality/self-assignment/debug-prefix quirks, IEEE/tolerance edges and public-index guards | 99.0% per profile; only the opposite-profile assertion path is uncovered |
| Exact rational/pair indexed vectors | 5,000 operations against independent rational maps; exact values below floating tolerance; copy/assignment/self-assignment and bounds; real set/assign/clone/add/drop allocation refusals with named-root and direct-object census | 90.0%; remaining statements are child error propagation and the other profile assertion path |
| Exact `stacked_vector<numeric_pair<mpq>>` | 1,600 operations against independent snapshots; big-value ownership; real append/replace/restore allocation refusals with live-cell/raw-block census; older/replacement pending-root identity | 92.8%; uncovered return paths propagate cleanup/capture errors |
| Shared permutation index algorithms and double/double applications | Original `--test_perm`; 2,304 distinct compositions and 288 transpositions against independent dense products; all 120 size-five permutations across six dense/four sparse directions; literal alias and shrunk-domain prefixes | Index algorithms 91.2% native / 93.0% debug; floating applications 87.7% / 92.6% |
| Exact rational/rational and rational/pair permutation applications | All 24 size-four permutations across six dense/four retained rational sparse directions against independent exact matrix products; ordered indices including zeros; 1000-digit allocation refusals through first success; named-cell/direct-object census; stopped destination ownership and every pending mutation guard | 82.8% native / 91.5% debug; native excludes debug dense copies and exact comparison |
| Native int/unsigned/double priority queues and int/unsigned pair queues | Original `-tbq` and strengthened pair smoke scenario; 12,000 independent map operations per type (60,000 total); exact tie/LIFO order, source mutation prefixes, extent wrap, NaN/infinity and diagnostic gaps | 85.6% scalar algorithms; 92.2% pair wrappers; index helpers 87.7/89.5% heap, 90.6/88.7% pair |
| Exact rational/pair priority queues and strict pair relations | 5,000 operations per type against independent exact ordering; retained-priority/copy ownership, ties, exact fractions; actual new/increase/decrease/dequeue reservation campaigns; by-value argument stops, named-cell/direct-object census and older comparison identity | 86.2% queue code, 84.6% call receipts in each profile; pair arithmetic/relations file remains 100% |
| Floating sparse coefficient lists, column/row eta matrices and actual tail interface | 2,000 independent dyadic matrix worlds across four dense/sparse directions; all 24 size-four permutation conjugations; mixed tail chains; strict configurable/fixed tolerance edges, exact index order, duplicates, signed zeros, IEEE values and reached bounds/conjugation prefixes | Eta and row eta 98.9% native / 100% debug; sparse coefficients 91.7% / 100%; interface binding 100% |
| Exact rational sparse coefficient lists and division ownership | 120 arbitrary-precision list worlds against `math/big.Rat`; first-match lookup, duplicate/zero entries, tiny divisors and live divisor aliasing; real append/lookup/positive and negative divide/alias-divide refusal campaigns through first success; named-cell/direct-object census and older raw invocation identity | Coefficient storage 87.1% native / 88.7% debug; exact matrix call receipts 86.2%; remaining statements propagate child/cleanup errors or the opposite-profile assertion |
| Exact rational/rational and rational/pair column eta applications and tail bindings | 240 independent rational matrix worlds; all 24 size-four conjugations and element views; mixed permutation/eta dense/sparse chains; source duplicate/self-term/index and bounds prefixes; real coefficient, diagonal, small/large-pivot and zero-destination refusal campaigns; stopped-destination and older raw ownership | Eta 89.4% native / 89.8% debug; concrete rational expression helpers 90%; exact tail bindings 100%. Gaps include child/cleanup errors and the unreachable precise branch that drops a nonzero pivot after division by a valid finite diagonal |
| Exact rational/rational and rational/pair row eta applications | 240 independent rational matrix worlds across dense/sparse directions and pair left; all 24 size-four conjugations/element views; mixed permutation/column/row tail chains; source snapshot/live-reference, duplicate/self-term and descending conjugation prefixes; actual reservation campaigns through numerical success, including index-before-result-copy stops; stopped-destination/older-call ownership | 91.4% native / 91.8% debug; remaining statements propagate child/cleanup errors or the opposite-profile assertion |
| Floating static matrix storage, queries, row pivots and dimension scopes | 12,000 independent map mutations; 2,000 independent row pivots; reciprocal links, fill/cancellation, IEEE underflow, duplicates, literal offset/scoping/debug quirks and valid mutation prefixes | 90.9% native / 93.5% debug; precise iterator integration and formatting remain open |
| Exact rational static coefficient storage and row elimination | 1,800 independent mutations; 240 exact matrix worlds with shuffled term order and multiple cancellations; tiny exact coefficients, live aliases, literal mutation prefixes; eight actual reservation campaigns through first success with named-root/direct-object census and older invocation ownership | 90.8% native / 92.2% debug; exact term-with-basis fill is implemented; complete solver integration remains open |
| Exact rational/pair static products, column extraction and absolute extrema | 160 independent arbitrary-precision matrix worlds across scalar/pair rows and scalar columns; output/live-alpha aliases, literal sparse order, retained dense slots, duplicates and bounds prefixes; ten physical reservation campaigns with named-root/direct-object census and older comparison identity | 91.6% native / 91.6% debug; remaining statements are child/cleanup errors and profile-only guards |
| Exact static dimension scopes, row transposition and diagnostics | 1,400 operations against independent dimension/coordinate maps; all three owning transpose copy stages under real numeric refusal; self/empty transposes, scope prefixes, source diagnostic gaps and writer errors | 88.7% native / 94.0% debug; release omits the final source diagnostic block, remaining gaps propagate child errors or are tautological checks |
| Retained rational row/column/indexed/pivot-row iterators | 4,800 independent mixed cursor operations; literal publication/end/size quirks, four physical copy-failure campaigns, borrowed live values, index-only reservation exclusion and older comparison ownership | 94.8% in each profile; remaining statements propagate child/cleanup errors |
| Exact LAR term ownership, arithmetic and substitution | 1,600 independent map/order operations; explicit zero/cancellation/reinsertion and live pivot prefixes; eight physical refusal campaigns, partial construction/export and borrowed-owner census | 90.0% in each profile; remaining statements propagate child/cleanup errors |
| Exact term-with-basis iterator | Basis-first and coefficient copy publication; live values, cursor failure, stopped-owner guards and actual copy receipt census | 96.3% in each profile |
| Exact static last-row fill with basis elimination | 240 independent rational elimination worlds; literal zero/duplicate/stale workspace behavior; three physical refusal campaigns through success, owner-specific publication prefixes and older comparison identity | 85.0% in each profile; remaining guards include unreachable child Reset/bounds failures for the retained concrete term iterator |
| Floating sparse basis construction and permutation queries | Standalone source test_init_U fixture; 600 independent sparse worlds, duplicate/zero/IEEE values, reciprocal offsets, 16 permutation rounds per world and malformed-input prefix checks | 100% in each focused profile; factorization, elimination and solves remain open |
| Exact sparse basis coefficient storage and permutation queries | 120 independent arbitrary-precision sparse worlds; 12 permutation rounds each, duplicate/zero/first-match storage, independent source/row/column owning-cell census; two real allocation-refusal campaigns through success, column-before-row stops and older comparison identity | 94.2% in both focused profiles; remaining statements propagate cleanup errors or guard fresh constructor state |
| Floating sparse mutation, row elimination, maximum ordering and division | 6,000 independent map mutations and 800 dense pivot worlds; reciprocal links, shuffled permutations, exact cancellation/fill, strict tolerances, live divisor, IEEE values, duplicate publication/removal and reached bounds prefixes | 94.4% native / 95.6% debug/checked focused; remaining gaps propagate bounds errors on malformed permutation metadata or belong to the opposite assertion profile |
| Exact sparse mutation, row elimination and live-reference division | 1,200 independent mutations; 160 independent rational pivot worlds; row/column divisor aliases, tiny exact values, reciprocal ownership census, eleven pooled/direct physical failure campaigns through success and older comparison identity | 90.0% native / 90.9% debug/checked focused; uncovered returns propagate cleanup/valid-scalar errors, guard malformed metadata, or belong to the opposite assertion profile |
| Exact maximum ordering and floating/exact pivot preparation/selection | 120 independent rational/float queue worlds; stable maxima, minimum admissible scores, inverse permutations, strict thresholds, stale/inactive candidates, exhaustion recovery, unsigned wrap, six physical failure campaigns and older comparison identity | Exact 89.9% native / 90.4% debug; float 88.2% / 89.7%; remaining statements propagate child errors, protect malformed metadata or are opposite-profile guards |
| Floating/exact active shortening and eta-driven sparse elimination | 160 independent incidence/queue worlds with permutations and eta refresh; 100 exact dense elimination worlds; early zero-column/empty-target exits, stale candidates, inactive eta reinsertions, signed score profiles, bounds publication prefixes and pooled/direct physical failure campaigns with borrowed eta census | Exact 94.2% native / 96.1% debug; float 93.6% / 95.7%; remaining statements propagate queue errors under malformed queue state or belong to the opposite assertion profile |
| Sparse-column eta construction and repeated-pivot integration | 180 independently ordered/permuted column worlds; literal duplicate/unit/zero/tiny/IEEE and publication prefixes; four physical failure campaigns with matrix/eta ownership census; 120 repeated pivot worlds checked against exact dense residuals and permutation-expansion determinants | Exact 94.1% and floating 96.7% in each profile; remaining statements propagate unreachable Append assertion or child/capture/cleanup errors |
| Floating/exact sparse column replacement | 1,600 independent coordinate-map replacements with permutations, row maxima and reciprocal links; literal duplicate/unindexed/tolerance/IEEE inputs; ten physical failure campaigns, borrowed-owner guards and source publication prefixes | Exact 88.3%, floating 95.2% in each profile; remaining statements propagate child/cleanup errors or assert the nonempty row guaranteed by successful insertion |
| Dense unit-upper triangular scalar/pair solves and floating residual correction | 220 independent exact RHS/solution/residual worlds with permutations, tiny/zero terms and trailing slots; literal diagonal/duplicate/IEEE and publication prefixes; six physical refusal campaigns with required pair X-before-Y stop and caller ownership census | Exact 92.8%, floating 97.2% in each profile; remaining statements propagate scalar/cleanup errors or are invariant-protected correction bounds |
| Indexed graph traversal and floating/exact scalar triangular solves | 240 independent nilpotent inverse-polynomial worlds with sparse reachability/index membership; literal postorder/output prefixes, flag lifetimes, zero/tiny/unindexed inputs; four physical failure campaigns and borrowed-owner guards | Graphs 94.4% native / 97.2% debug each; exact solve 89.3%, floating 93.4% in both profiles; remaining statements propagate child/cleanup errors, protect index rebuild bounds or belong to the opposite assertion profile |
| Floating indexed residual correction/refinement | 240 independent exact inverse-polynomial and dense-residual worlds with shuffled permutations; strict density boundary, sparse/dense tolerance divergence, hidden/trailing data, fixed cleanup/order/IEEE and reached error prefixes | 91.4% native and debug/checked (32/35); remaining returns guard invariant-protected correction errors |
| Sparse identity extension, stored-entry count and LU permutation composition | 120 worlds with eight repeated dense-oracle permutation/growth rounds; retained workspace/queue/counters and reciprocal links; pooled/direct inline-one growth under disabled numerical reservations; native/debug shrink, real stopped-owner and publication/error prefixes | Floating 96.6% native / 100% debug; exact 91.3% / 93.5%; remaining returns propagate inline-one/capture/cleanup errors or belong to the opposite assertion profile |
| LU one-coordinate diagonal tails for floating/rational/pair paths | 240 independent matrix/conjugation worlds; exact tiny values, strict tolerance/IEEE/index quirks, constructor ownership, fourteen pooled/direct reservation campaigns, stopped caller receipts and actual pair X-before-Y failure; cached debug view | Floating 97.3% native / 100% debug; exact 86.1% / 88.9%; debug view 100%; gaps propagate child/cleanup/capture errors, test an unreachable exact-zero quotient or belong to the other profile |
| Floating LU dense-block factorization, parent update and tail applications | 180 independent exact elementary-elimination worlds checking E and E*A*P, column swaps, sparse parent values/backlinks and all four tail routes after conjugation; literal tolerance/IEEE/duplicate/trailing/growth cases and reached publication errors | Core 90.5% native / 93.2% debug; applications 81.9% / 83.2%; debug L check 91.7%, view 93.8%; remaining gaps are metadata/child propagation and assertion-profile paths |
| Settings defaults, stats, float tolerance families, random state and resource limits | All numerical/strategy defaults, source RNG golden vectors, mutable thresholds, strict CPU-time boundary, borrowed limit/error propagation and stats reset | 96.3%; uncovered constructor return propagates a real host-clock start error |
| Settings enum values and three formatting/parsing functions | All column/status cases, source missing cases, parser/output asymmetry, native/debug assertion profiles | 100% |
| Existing LP parameter collector | All five exact keys/kinds/defaults/descriptions/module names; each reached conflicting-kind error prefix; native backing under disabled Runtime reservations | 100% |

The last complete native LP run reported **90.7%** statement coverage
(6210/6843 existing Go statements), including sparse indexed graph traversal
and solves, before floating indexed refinement, sparse identity extension, diagonal tails and floating dense blocks were added. The last
complete combined debug/checked run reported **91.7%** before sparse basis storage
was added; sparse basis, elimination, pivot, active shortening, eta construction, column
replacement and dense/indexed solve components now also pass focused debug/checked tests. Exact scoped
vector allocation and rollback paths are tested; remaining uncovered statements
include numerical child and cleanup/capture error returns, plus profile-specific
assertion paths. The explicit
duplicate-index assertion branch is unreachable in release; its following
continuation is unreachable in debug. The two profiles exercise both behaviors.

Coverage is measured with native and combined `g3debug g3checked` builds. It
applies to the current Go code only. Debug assertions use the existing explicit
Go error profile; process-global debugger notification/stream binding is not
claimed. Go slices and maps replace STL/container internals. Printing uses Go
`io.Writer`, with file ownership and flushing left to its caller.

Floating static row balancing reaches the source double logarithm’s unimplemented
fatal path for nonzero terms. Go returns `ErrLPBalanceNotImplemented` at that
point; no invented logarithm or balancing algorithm is credited.

The next layers retain the full requested scope:

1. Concrete scoped vectors, actual numeral/numeric-pair types, priority-queue integration and
   indexed vectors, with independent state/ordering/arithmetic tests.
2. Sparse/static/dense matrices, remaining permutation integration, eta transformations and LU,
   checked against independent matrix operations and exact residuals.
3. Scaling, settings, random updates and primal/dual simplex, tested with feasible,
   infeasible, degenerate and unbounded programs and independent certificates.
4. LAR constraints, scopes, terms, bound propagation and explanations, including
   strict bounds and evidence validation.
5. MPS input and the remaining upstream test scenarios, followed by coverage and
   retained-consumer audits across the complete implementation.

`src/test/lp.cpp` has command-selected test branches and an unconditional return
before several trailing test calls. Those trailing calls are not credited as
executed upstream tests. Reachability of project templates must be checked before
adding concrete Go instantiations; definition-only templates and STL code are
not port targets.

Focused checks (from the repository root):

```sh
go -C pkg test ./util/lp -coverprofile=/tmp/g3-lp-native.cover
go -C pkg test -tags 'g3debug g3checked' ./util/lp -coverprofile=/tmp/g3-lp-debug.cover
go tool cover -func=/tmp/g3-lp-native.cover
go -C pkg vet ./util/lp
go -C pkg run ./cmd/portcheck
```
