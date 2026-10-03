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
