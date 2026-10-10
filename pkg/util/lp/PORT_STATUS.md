# LP port implementation and test coverage

The full `src/util/lp` port is **in progress**. The inventory includes all 110
files, including ignored-by-default `.hpp` implementation bodies. See
[PORT_STATUS.json](PORT_STATUS.json) for every source file's current targets and
tests; `port_manifest.json` remains the repository-wide source ledger.

There are currently twenty partial source units and 90 unported units. No complete
LP solver is available yet. A green Go coverage report does not measure the C++
code that has not been ported.

| Component implemented so far | Verification | Statement coverage |
| --- | --- | --- |
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
| Settings enum values and three formatting/parsing functions | All column/status cases, source missing cases, parser/output asymmetry, native/debug assertion profiles | 100% |
| Existing LP parameter collector | All five exact keys/kinds/defaults/descriptions/module names; each reached conflicting-kind error prefix; native backing under disabled Runtime reservations | 100% |

Current package statement coverage is **89.5% native and 91.9% combined debug/checked**. Exact scoped
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

The next layers retain the full requested scope:

1. Concrete scoped vectors, actual numeral/numeric-pair types, remaining exact priorities and
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
