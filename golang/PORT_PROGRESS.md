# Port progress

The full fork remains the objective. This foundation work supplies no solver API,
SAT result, theory, AST manager, proof, or interpolation implementation.

## Current baseline

`port_manifest.json` inventories 2,102 source/build/tool/example units and 677
API declarations. All production entries start unported. The 1,687 source files
with extensions `.cpp`, `.h`, `.hpp`, or `.pyg` have content fingerprint
`f8480997c14e2d6c414f40c60ce5095ef40a89ce6d1bcf1ed196c5d682d42d8f`.
This explicitly records the current checkout rather than claiming the plan's
1,619-file fingerprint matches. There are 68 untracked generated source files in
that set; no reference files were altered during this work. Restricting the set
to Git-tracked files gives
`caabc82ba8943415d98e1d87be9c219aaed988ce8afdf8b16da3be146b4f55cc`.
The plan's four individual AST/SMT/interpolation checksums match. Generated files
and changed-baseline provenance still need detailed generator audits.

Conditional directives, executable assertion locations, and registration lines
are lexical inventory, not proof that branch activation or ownership was audited.
Per-method, binding, registration, test-runner, and ownership audits remain open
until recorded by component. Header API declarations are inventoried; their
implementation dispatch and generated binding counterparts still need audits.
Source-existence coverage is distinct from semantic completion.

## Tests first

The checker tests first failed with undefined inventory/validation symbols, then
passed after implementation. Additional RED tests exposed omitted Python binding
files and missing executable-test validation before those fixes. Synthetic trees
make normal checker tests hermetic. Maintainer reference reads are opt-in:

```
go run ./cmd/portcheck -source ..
```

The checker checks hashes, newly added source/API coverage, statuses and executable
Go test declarations. Declarations are evidence that tests exist; their passing
execution and source-method coverage remain separate review requirements. It
refuses to regenerate over an existing ledger, preventing loss of progress.

Main-package handle tests likewise failed before implementation and now pass in
release and `g3checked` modes. Storage tokens always retain generation; internal
release references retain slots only. Public owner/kind validation awaits real
domain managers. The arena, numeric, hashing, and bit-vector packets document
source assertions, independent tests, adaptations and open methods in their
package `PORT_STATUS.md` files.

## Work queue and gates

Next: word/typed-vector payload pools and scoped regions; persistent numeral
cells and complete mpz/mpq methods; remaining containers/limits/parameters;
generated tables and complete registration/ownership audits. Foundation gates
must precede dependent AST/solver work. The source manifest remains open for all
other components, including all fixedpoint engines, QE, optimization, APIs and
frontends. wasm compilation is not wasm runtime acceptance.

All authored files, scratch, build outputs and configured Go cache/temp paths
stay below `golang/`. Git commits are explicitly requested by the user.
