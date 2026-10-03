# Expression-array retained storage correction

Source parray.h get/set/set-from logical bounds are SASSERT-only. Three new
regressions first failed in both release and checked/debug, then passed after
preserving release retained root reads/writes and arbitrary-index nonroot SET.
Debug guards execute before allocation/copy and report ErrASTDomain.

A popped root slot retains its raw handle. Get returns that token in release;
later AST dereference rejects a freed generation. Root Set follows INC(new),
DEC(old), write, including a retained value outside logical size. Root deletion
decrements only logical elements. Nonroot SET stores its index/value without
touching root backing; matching Get returns it before any root-array read.
The test independently checks reference counts and unchanged logical sizes.

Go backing adds an actually allocated/charged4-byte initialized high-water
field to the8-byte capacity prefix (12-byte total). Writes advance that marker;
growth copies all old capacity bytes and the old initialized marker without
inventing initialized pointers. Source copy_values copies only logical elements
and therefore initializes only those new slots. A never-written physical read
returns arena.ErrBounds at the actual access. Values/storage statistics and all
deallocation sizes use the real12-byte header. No generic code or Go pointers
are encoded into hot records.

The frozen snapshot .cache/review/expr-array-retained-green contains committed
HEADa18b6ce plus only these3 files, excluding parallel marks/dependency/solver
drafts. Full native (root2.848s/mpnum21.947s), full checked/debug
(root3.380s/mpnum22.733s), six expression-array groups under checked/debug race
(1.033s), six actual checked/debug JavaScript RUN/PASS events, vet and WASI
compilation passed. The three added groups also passed actual release JavaScript with explicit
RUN/PASS events. Source fingerprint audit passed at2102 units/677 APIs with
33 ported/77 partial/1992 unported; final metadata/no-generics checks passed.
WASI execution is not claimed.

This is a correction to the existing real expression-array specialization.
The source AST/parray units remain partial; dependency arrays, full manager,
diagnostics, solver and interpolation remain separate obligations.

Frozen Go SHA-256 values:

- `ast_expr_array.go`: `ad77dc102ba92a9e77cc8f979cc82606efbfe6588c148f0e1ddce8fa4666f169`
- `ast_expr_array_values.go`: `7c5879abf60ca71e8c715d34c20324f67f7c2f2395fab46ec972065773c2036f`
- `ast_expr_array_retained_test.go`: `9e17689fa0b1e981e95ce6a47d09f44ed67f84efaeac9475971b26a9434b9355`

## Empty nonroot POP and invalid old SET follow-up

Three further regressions observed behavioral RED before changes. Release
empty nonroot POP allocates a POP patch with index0, making its unsigned logical
size UINT32_MAX without reading root backing. Both direct and From operations
preserve that finite branch. Debug's source empty assertion returns
ErrASTDomain before copying the From destination. Null/root empty POP remains
an explicit guard at the source undefined physical-access boundary.

Root rset performs INC(new) before reading or decrementing the old slot.
An uninitialized old slot or a retained freed AST is a typed Go guard, with the
completed incoming INC still visible. The backing slot remains unchanged; no
invented DEC rollback occurs. Caller-owned incoming references can explicitly
balance that prefix. Debug logical bounds reject before any INC.

The final isolated .cache/review/expr-array-prefix-green tree contains
HEAD2a68598 plus only the three expression-array Go files and final ledger.
Full native (root2.486s/mpnum22.679s) and checked/debug
(root3.026s/mpnum23.343s) passed. All nine expression-array groups passed
checked/debug race (1.053s) and actual JavaScript (0.191s); the three new
release-only failure-prefix regressions also ran and passed actual JavaScript
(0.200s). Vet, WASI test compilation and final source/no-generics evidence
checks passed. Coverage remains33 ported/80 partial/1989 unported.
WASI runtime and full solver completion are not claimed.

Final follow-up frozen Go SHA-256 values:

- `ast_expr_array.go`: `16558e5010c23e9dddaf8456b13a0d769c1fac49c253fb5c513a39c2bad10a40`
- `ast_expr_array_values.go`: `6322ec71b4acd18520e04cdce41da80f3afb6af40ad14448d512181a3961e269`
- `ast_expr_array_retained_test.go`: `f4407926698b0139aa928363e6b07570e1a7bbb94451133a8fcc45ac8008170f`
